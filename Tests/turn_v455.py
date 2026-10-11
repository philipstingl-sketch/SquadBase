"""TURN im Client (v4.5.5): frische Zugangsdaten bei jedem Anruf, Rückfall bei Ausfall, Verbindungstest.

Start:  python3 Tests/turn_v455.py      (Playwright + Chromium nötig)
Der Test nutzt echte WebRTC-Verbindungsobjekte, aber keinen echten TURN-Server: Die positive Antwort
„Relay erreichbar“ wird mit einem untergeschobenen Verbindungsobjekt geprüft, der Rest mit echtem Chromium.
"""
import asyncio, json, re, time
from pathlib import Path
from playwright.async_api import async_playwright

ROOT = Path(__file__).resolve().parents[1] / 'Quellcode' / 'windows-app' / 'web'
if not ROOT.exists():
    ROOT = Path(__file__).resolve().parents[1] / 'windows-app' / 'web'
ORIGIN, BASE = 'http://localhost:8456', 'https://squadbase-friends-cloud.test'

html = (ROOT / 'index.html').read_text(encoding='utf-8')
html = re.sub(r'<link rel="stylesheet" href="([^"]+)"\s*>', lambda m: '<style>' + (ROOT / m.group(1)).read_text(encoding='utf-8') + '</style>', html)
html = re.sub(r'<script src="([^"]+)"\s*></script>', lambda m: '<script>' + (ROOT / m.group(1)).read_text(encoding='utf-8').replace('</script>', '<\\/script>') + '</script>', html)

NO_TURN = {'iceServers': [{'urls': ['stun:127.0.0.1:9']}], 'hasTURN': False, 'turnMode': 'none'}  # STUN-Adresse antwortet nicht
def turn(cred):
    return {'iceServers': [{'urls': ['turn:127.0.0.1:9?transport=udp'], 'username': 'u-' + cred, 'credential': cred}], 'hasTURN': True, 'turnMode': 'cloudflare', 'ttl': 86400}

state = {'ice': NO_TURN, 'ice_calls': 0, 'ice_status': 200, 'signals': []}

def api(arg):
    req = json.loads(arg); path = req['path']
    if path.startswith('/api/ice'):
        state['ice_calls'] += 1
        return state['ice_status'], state['ice']
    if path.startswith('/api/signals'):
        if req['method'] == 'POST':
            state['signals'].append(json.loads(req['body'])); return 200, {'ok': True}
        return 200, {'signals': [], 'latest': 0}
    table = {
        '/api/autoclicker': {'intervalMs': 200, 'mouseButton': 'left', 'hotkey': 'F8', 'active': False, 'clicks': 0, 'supported': True},
        '/api/friends-cloud-config': {'url': BASE},
        '/api/health': {'status': 'ok', 'capabilities': {'persistent': True, 'invites': True, 'unread': True, 'search': True, 'turn': True}},
        '/api/me': {'username': 'alice'},
        '/api/friends': {'friends': ['bob'], 'incoming': [], 'outgoing': [], 'online': {'bob': True}, 'blocked': []},
        '/api/unread': {'unread': {}}, '/api/read': {'ok': True},
        '/api/friend-prefs': {'messages': True, 'calls': True, 'sent': False, 'invites': True, 'volume': 55, 'volumes': {}},
    }
    if path in table: return 200, table[path]
    if path.startswith('/api/messages?') or path.startswith('/api/search?'): return 200, {'messages': []}
    return 404, {'error': 'mock missing ' + path}

async def handler(arg):
    status, data = api(arg)
    return json.dumps({'status': status, 'data': data})

INIT = r'''
(() => {
 localStorage.setItem('sb-relay-url', '%BASE%');
 sessionStorage.setItem('sb-relay-token-base', '%BASE%');
 sessionStorage.setItem('sb-relay-token', 'a'.repeat(64));
 sessionStorage.setItem('sb-relay-user', 'alice');
 window.fetch = async (url, opts = {}) => {
  const u = new URL(String(url), location.href);
  const reply = JSON.parse(await window.__api(JSON.stringify({path: u.pathname + u.search, method: opts.method || 'GET', body: opts.body || null})));
  return new Response(JSON.stringify(reply.data), {status: reply.status, headers: {'Content-Type': 'application/json'}});
 };
 // Konfigurationen mitschreiben, mit denen die App Verbindungen anlegt
 window.__configs = [];
 const Real = window.RTCPeerConnection;
 window.RTCPeerConnection = function (cfg) { window.__configs.push(JSON.parse(JSON.stringify(cfg || {}))); return new Real(cfg); };
 window.RTCPeerConnection.prototype = Real.prototype;
 window.__Real = Real;
})();
'''.replace('%BASE%', BASE)

async def wait_for(page, js, what, timeout=40):
    end = time.time() + timeout
    while time.time() < end:
        if await page.evaluate(js): return
        await asyncio.sleep(0.3)
    raise AssertionError('Zeitüberschreitung: ' + what + ' | Status: ' + await page.inner_text('#v454-test-status'))

async def main():
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True, args=['--no-sandbox', '--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream', '--allow-loopback-in-peer-connection'])
        ctx = await browser.new_context(permissions=['microphone', 'camera'])
        await ctx.expose_function('__api', handler)
        await ctx.add_init_script(INIT)
        await ctx.route(ORIGIN + '/**', lambda route: route.fulfill(status=200, content_type='text/html; charset=utf-8', body=html))
        page = await ctx.new_page()
        errors = []; page.on('pageerror', lambda e: errors.append(str(e)))
        await page.goto(ORIGIN + '/')
        await page.locator('[data-tab="friends"]').first.click()
        await page.wait_for_selector('#friend-list .friend-item', timeout=15000)
        assert state['ice_calls'] >= 1, 'beim Anmelden werden ICE-Server geladen'

        # 1) Anruf holt frische Zugangsdaten und nutzt genau diese
        state['ice'] = turn('frisch-1'); state['ice_calls'] = 0
        await page.locator('#friend-list .friend-item', has_text='bob').first.click()
        await page.click('#call-voice')
        await wait_for(page, "window.__configs.length>0", 'Verbindung wurde angelegt', 15)
        cfg = await page.evaluate('window.__configs.at(-1)')
        assert state['ice_calls'] == 1, state['ice_calls']
        assert cfg['iceServers'][0]['credential'] == 'frisch-1', cfg
        await page.click('#call-hangup')
        print('1 ok: jeder Anruf holt frische TURN-Zugangsdaten und nutzt sie')

        # 2) Server-Ausfall beim Abruf: Anruf startet trotzdem mit den zuletzt bekannten Zugangsdaten
        state['ice_status'] = 500
        await page.evaluate('window.__configs.length = 0')
        await page.click('#call-voice')
        await wait_for(page, "window.__configs.length>0", 'Verbindung trotz Ausfall angelegt', 15)
        cfg = await page.evaluate('window.__configs.at(-1)')
        assert cfg['iceServers'][0]['credential'] == 'frisch-1', cfg
        assert not await page.evaluate("document.getElementById('call-overlay').hidden"), 'Anruf läuft'
        await page.click('#call-hangup'); state['ice_status'] = 200
        print('2 ok: bei Ausfall des Abrufs gelten die letzten Zugangsdaten, der Anruf startet')

        # 3) Verbindungstest: TURN eingerichtet, aber unerreichbar (127.0.0.1:9 antwortet nicht)
        await page.click('summary:has-text("Mikrofontest")')  # Bereich aufklappen wie ein Nutzer
        await page.click('#v455-net-test')
        await wait_for(page, "document.getElementById('v454-test-status').textContent.startsWith('✗') || document.getElementById('v454-test-status').textContent.startsWith('✓')", 'Testergebnis (unerreichbar)', 40)
        text = await page.inner_text('#v454-test-status')
        assert text.startswith('✗ TURN ist eingerichtet, aber nicht erreichbar'), text
        assert not await page.is_disabled('#v455-net-test'), 'Knopf ist nach dem Test wieder frei'
        print('3 ok: unerreichbarer TURN wird als Fehler gemeldet:', text[:70])

        # 4) Verbindungstest: kein TURN eingerichtet
        state['ice'] = NO_TURN
        await page.click('#v455-net-test')
        await wait_for(page, "!document.getElementById('v454-test-status').textContent.includes('läuft')", 'Testergebnis (ohne TURN)', 40)
        text = await page.inner_text('#v454-test-status')
        assert 'kein TURN-Server eingerichtet' in text.lower().replace('es ist kein', 'kein') or 'kein TURN-Server' in text, text
        print('4 ok: fehlendes TURN wird erklärt:', text[:70])

        # 5) Verbindungstest: Relay-Kandidat vorhanden (untergeschobenes Verbindungsobjekt)
        state['ice'] = turn('frisch-2')
        await page.evaluate("""() => { window.RTCPeerConnection = class { constructor(cfg){ this.cfg = cfg; }
            createDataChannel(){} async createOffer(){ return {}; }
            async setLocalDescription(){ setTimeout(() => this.onicecandidate && this.onicecandidate({candidate: {candidate: 'candidate:1 1 udp 1 203.0.113.9 3478 typ relay'}}), 20); }
            close(){} }; }""")
        await page.click('#v455-net-test')
        await wait_for(page, "document.getElementById('v454-test-status').textContent.startsWith('✓')", 'Testergebnis (Relay ok)', 20)
        print('5 ok: erreichbarer Relay wird als Erfolg gemeldet')

        assert not errors, errors
        print('PASS TURN im Client: frische Zugangsdaten, Rückfall bei Ausfall, Verbindungstest; keine JS-Fehler')
        await browser.close()

asyncio.run(main())
