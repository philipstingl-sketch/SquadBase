"""Gruppenanruf-Test mit ECHTEM WebRTC: mehrere Chromium-Fenster (je ein Benutzer) sprechen über einen
nachgebauten Signal-Server miteinander. Geprüft werden Aufbau, Wachstum, gleichzeitige Einladungen,
Verlassen, Nicht-Freunde und eine ältere Client-Version.

Start:  python3 Tests/mesh_v455.py      (Playwright + Chromium nötig)
"""
import asyncio, json, re, sys, time
from pathlib import Path
from playwright.async_api import async_playwright

ROOT = Path(__file__).resolve().parents[1] / 'Quellcode' / 'windows-app' / 'web'
if not ROOT.exists():
    ROOT = Path(__file__).resolve().parents[1] / 'windows-app' / 'web'
ORIGIN = 'http://localhost:8455'
BASE = 'https://squadbase-friends-cloud.test'
USERS = ['alice', 'bob', 'carol', 'dave', 'erin']

# ---------- die App als eine HTML-Datei (Skripte und Styles eingebettet) ----------
html = (ROOT / 'index.html').read_text(encoding='utf-8')
html = re.sub(r'<link rel="stylesheet" href="([^"]+)"\s*>', lambda m: '<style>' + (ROOT / m.group(1)).read_text(encoding='utf-8') + '</style>', html)
html = re.sub(r'<script src="([^"]+)"\s*></script>', lambda m: '<script>' + (ROOT / m.group(1)).read_text(encoding='utf-8').replace('</script>', '<\\/script>') + '</script>', html)

# ---------- nachgebauter Cloud-Server (gleiche Regeln wie worker.mjs) ----------
class Cloud:
    def __init__(self):
        self.signals, self.next_id = [], 1
        self.friends = {u: set(USERS) - {u} for u in USERS}
        self.friends['lena'] = {'alice'}; self.friends['alice'].add('lena')
        self.prefs = {}
    def unfriend(self, a, b):
        self.friends[a].discard(b); self.friends[b].discard(a)
    def push(self, frm, to, kind, data):
        self.signals.append({'id': self.next_id, 'from': frm, 'to': to, 'kind': kind, 'data': data, 'at': int(time.time() * 1000)})
        self.next_id += 1
    def handle(self, user, req):
        path, method, body = req['path'], req['method'], json.loads(req['body']) if req.get('body') else None
        ok = lambda d: (200, d)
        if path.startswith('/api/signals'):
            if method == 'POST':
                if body['kind'] not in ('offer', 'answer', 'ice', 'hangup', 'busy'): return 400, {'error': 'Ungültiges Signal'}
                if body['to'] not in self.friends.get(user, ()): return 403, {'error': 'Nur bestätigte Freunde'}
                self.push(user, body['to'], body['kind'], body['data']); return ok({'ok': True})
            after = int(re.search(r'after=(\d+)', path).group(1)) if 'after=' in path else 0
            rows = [s for s in self.signals if s['id'] > after and s['to'] == user and s['from'] in self.friends[user]][:100]
            return ok({'signals': rows, 'latest': self.next_id - 1})
        if path == '/api/friend-prefs':
            if method == 'POST': self.prefs[user] = body
            return ok(self.prefs.get(user, {'messages': True, 'calls': True, 'sent': False, 'invites': True, 'volume': 55, 'volumes': {}}))
        table = {
            '/api/autoclicker': {'intervalMs': 200, 'mouseButton': 'left', 'hotkey': 'F8', 'active': False, 'clicks': 0, 'supported': True},
            '/api/friends-cloud-config': {'url': BASE},
            '/api/health': {'status': 'ok', 'name': 'SQUAD BASE Friends Cloud', 'capabilities': {'files': False, 'invites': True, 'persistent': True, 'unread': True, 'search': True, 'accountExport': True, 'accountDelete': True}},
            '/api/me': {'username': user},
            '/api/ice': {'iceServers': []},  # Test läuft lokal: nur Host-Kandidaten, kein STUN nötig
            '/api/friends': {'friends': sorted(self.friends[user]), 'incoming': [], 'outgoing': [], 'online': {u: True for u in self.friends[user]}, 'blocked': []},
            '/api/unread': {'unread': {}},
            '/api/read': {'ok': True},
        }
        if path in table: return ok(table[path])
        if path.startswith('/api/messages?') or path.startswith('/api/search?'): return ok({'messages': []})
        return 404, {'error': 'mock missing ' + path}

cloud = Cloud()

INIT = r'''
(() => {
 const user = %USER%;
 localStorage.setItem('sb-relay-url', '%BASE%');
 sessionStorage.setItem('sb-relay-token-base', '%BASE%');
 sessionStorage.setItem('sb-relay-token', 'a'.repeat(64));
 sessionStorage.setItem('sb-relay-user', user);
 window.__toasts = [];
 document.addEventListener('DOMContentLoaded', () => {
  const t = document.getElementById('toast');
  if (t) new MutationObserver(() => { if (t.textContent) window.__toasts.push(t.textContent); }).observe(t, {childList: true, characterData: true, subtree: true});
 });
 window.fetch = async (url, opts = {}) => {
  const u = new URL(String(url), location.href);
  const reply = JSON.parse(await window.__api(JSON.stringify({path: u.pathname + u.search, method: opts.method || 'GET', body: opts.body || null})));
  return new Response(JSON.stringify(reply.data), {status: reply.status, headers: {'Content-Type': 'application/json'}});
 };
})();
'''

async def wait_until(check, what, timeout=40, every=0.4):
    end = time.time() + timeout
    last = None
    while time.time() < end:
        try:
            last = await check()
            if last: return last
        except Exception as err:  # Seite lädt noch
            last = err
        await asyncio.sleep(every)
    raise AssertionError(f'Zeitüberschreitung: {what} (letzter Wert: {last})')

class User:
    def __init__(self, name, page): self.name, self.page, self.errors, self.logs = name, page, [], []
    async def state(self): return await self.page.evaluate('window.SquadCalls.state()')
    async def toast(self): return ' | '.join(await self.page.evaluate("window.__toasts"))
    async def open_chat(self, friend):
        await self.page.locator('#friend-list .friend-item', has_text=friend).first.click()

async def connected(users, expected_peers, what, timeout=45):
    """Alle genannten Benutzer sind im selben Anruf und mit allen anderen echt (ICE+DTLS) verbunden."""
    names = sorted(u.name for u in users)
    async def check():
        for u in users:
            st = await u.state()
            if not st or st['ringing']: return False
            others = sorted(set(names) - {u.name})
            if sorted(p['user'] for p in st['peers']) != others: return False
            if any(p['connection'] != 'connected' or p['pending'] for p in st['peers']): return False
            if st['known'] != names: return False
        return True
    try:
        await wait_until(check, what, timeout)
    except AssertionError:
        for u in users:
            print('  Zustand', u.name, json.dumps(await u.state(), ensure_ascii=False), '| Hinweis:', (await u.toast())[:90])
        raise

async def hung_up(user):
    await wait_until(lambda: user.page.evaluate("document.getElementById('call-overlay').hidden"), user.name + ' hat keinen aktiven Anruf mehr', 20)

async def accept(user, title_part=None):
    await user.page.wait_for_selector('#call-accept:visible', timeout=25000)
    if title_part:
        title = await user.page.inner_text('#call-title')
        assert title_part in title, (user.name, title)
    await user.page.click('#call-accept')

async def invite(inviter, friend):
    await wait_until(lambda: inviter.page.evaluate(f"[...document.querySelectorAll('#v455-invite-select option')].some(o=>o.value==='{friend}')"), f'{friend} steht bei {inviter.name} in der Einladen-Liste', 15)
    await inviter.page.select_option('#v455-invite-select', friend)
    await inviter.page.click('#v455-invite-btn')

async def leave(user):
    # Wenn die letzte andere Person schon aufgelegt hat, endet der Anruf von selbst und der Knopf ist weg.
    if await user.page.is_visible('#call-hangup'):
        await user.page.click('#call-hangup', timeout=5000)

async def main():
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True, args=[
            '--no-sandbox', '--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream',
            '--allow-loopback-in-peer-connection', '--autoplay-policy=no-user-gesture-required', '--disable-features=WebRtcHideLocalIpsWithMdns'])
        users = {}
        for name in USERS + ['lena']:
            if name == 'lena': continue
            ctx = await browser.new_context(permissions=['microphone', 'camera'])
            def make(n): 
                async def api(arg):
                    status, data = cloud.handle(n, json.loads(arg))
                    return json.dumps({'status': status, 'data': data})
                return api
            await ctx.expose_function('__api', make(name))
            await ctx.add_init_script(INIT.replace('%USER%', json.dumps(name)).replace('%BASE%', BASE))
            await ctx.route(ORIGIN + '/**', lambda route: route.fulfill(status=200, content_type='text/html; charset=utf-8', body=html))
            page = await ctx.new_page()
            u = users[name] = User(name, page)
            page.on('pageerror', lambda e, u=u: u.errors.append(str(e)))
            page.on('console', lambda m, u=u: u.logs.append(f'{m.type}: {m.text}'[:220]) if m.type in ('warning', 'error') and 'mock missing' not in m.text and '404' not in m.text else None)
            await page.goto(ORIGIN + '/')
            await page.locator('[data-tab="friends"]').first.click()
            await page.wait_for_selector('#friend-list .friend-item', timeout=15000)
        A, B, C, D, E = (users[n] for n in USERS)
        try:
            await scenarios(A, B, C, D, E, users)
        except BaseException:
            # Bei jedem Fehler den Zustand aller Fenster ausgeben, damit sich seltene Aussetzer klären lassen
            print('  LETZTE SIGNALE:', [(s['id'], s['from'] + '>' + s['to'], s['kind'], (s['data'] or {}).get('squadCall', {}).get('id', '')[:6] if isinstance(s['data'], dict) else '') for s in cloud.signals[-14:]])
            for u in users.values():
                print('  KONSOLE', u.name, u.logs[-8:])
                try:
                    print('  ZUSTAND', u.name, json.dumps(await u.state(), ensure_ascii=False), '| Hinweise:', (await u.toast())[-160:], '| JS-Fehler:', u.errors)
                except Exception as err:
                    print('  ZUSTAND', u.name, 'nicht lesbar:', err)
            raise
        await browser.close()

async def scenarios(A, B, C, D, E, users):
    if True:

        # ---- S1: normaler 1:1-Anruf bleibt unverändert ----
        await A.open_chat('bob'); await A.page.click('#call-voice')
        await accept(B, None)
        await connected([A, B], 1, 'S1 alice↔bob verbunden')
        assert await A.page.is_visible('#v455-volume-row'), '1:1 zeigt den bekannten Lautstärkeregler'
        assert await A.page.locator('#v455-tiles .v455-tile').count() == 1
        assert await A.page.evaluate("document.querySelector('#v455-tiles video').srcObject?.getAudioTracks().length>0"), 'Audio kommt an'
        print('S1 ok: 1:1-Anruf verbunden, Ton vorhanden')

        # ---- S2: Gruppe wächst – alice lädt carol ein, bob lädt dave ein (dave ist alphabetisch der Größte) ----
        await invite(A, 'carol')
        await accept(C, 'Gruppenanruf von alice')
        assert 'bob' in await C.page.inner_text('#call-title')
        await connected([A, B, C], 2, 'S2a alice+bob+carol verbunden')
        await invite(B, 'dave')
        await accept(D, 'Gruppenanruf von bob')
        await connected([A, B, C, D], 3, 'S2b vier Personen im Vollmesh')
        assert not await A.page.is_visible('#v455-volume-row'), 'bei Gruppen kein globaler Regler'
        assert await A.page.locator('#v455-tiles .v455-tile').count() == 3
        assert await A.page.locator('#v455-tiles .v455-tile-vol:visible').count() == 3, 'jede Kachel hat einen eigenen Regler'
        for u in (A, B, C, D):
            tracks = await u.page.evaluate("[...document.querySelectorAll('#v455-tiles .v455-tile')].map(t=>{const s=t.querySelector('video').srcObject;return [t.dataset.user,s?s.getAudioTracks().length:0,s?s.getVideoTracks().length:0]})")
            assert all(a == 1 and v == 1 for _, a, v in tracks), (u.name, tracks)  # pro Person je ein Ton- und ein Videostrom in derselben Kachel
        print('S2 ok: 4 Personen vollständig verbunden (6 Verbindungen)')

        # ---- S2c: Kamera in der Gruppe – alice schaltet ein, alle sehen echtes Bild; danach wieder aus ----
        await A.page.click('#v454-toggle-camera')
        async def alice_picture_everywhere():
            for u in (B, C, D):
                if not await u.page.evaluate("(()=>{const v=document.querySelector('#v455-tiles .v455-tile[data-user=\"alice\"] video');return !!v&&v.videoWidth>0})()"): return False
            return True
        try:
            await wait_until(alice_picture_everywhere, 'alices Kamerabild kommt bei bob, carol und dave an', 25)
        except AssertionError:
            dump = "[...document.querySelectorAll('#v455-tiles .v455-tile')].map(t=>({u:t.dataset.user,tracks:(t.querySelector('video').srcObject?.getTracks()||[]).map(x=>x.kind+':'+x.readyState+':muted='+x.muted),w:t.querySelector('video').videoWidth,paused:t.querySelector('video').paused}))"
            for u in (A, B, C, D): print('  DEBUG', u.name, json.dumps(await u.page.evaluate(dump)))
            print('  DEBUG alice eigenes Video:', await A.page.evaluate("document.getElementById('my-video').srcObject?.getTracks().map(x=>x.kind+':'+x.readyState)"))
            raise
        async def camera_label(part): return part in await A.page.inner_text('#v454-toggle-camera')
        await wait_until(lambda: camera_label('ausschalten'), 'Kamera-Knopf zeigt „ausschalten“', 10)
        await A.page.click('#v454-toggle-camera')
        await wait_until(lambda: camera_label('einschalten'), 'Kamera-Knopf zeigt „einschalten“', 10)
        print('S2c ok: Kamera wird an alle drei Verbindungen übertragen')

        # ---- S3: Auflegen – Rest bleibt verbunden ----
        await leave(B)
        await hung_up(B)
        async def b_gone():
            for u in (A, C, D):
                st = await u.state()
                if not st or 'bob' in [p['user'] for p in st['peers']] or 'bob' in st['known']: return False
                if await u.page.locator('#v455-tiles .v455-tile[data-user="bob"]').count(): return False
            return True
        await wait_until(b_gone, 'bob ist bei allen entfernt', 20)
        await connected([A, C, D], 2, 'S3 alice+carol+dave bleiben verbunden')
        print('S3 ok: Verlassen entfernt nur bob, Rest bleibt verbunden')

        # ---- S4: gleichzeitige Einladungen auf 5 Personen (bob ist kleiner als alle außer alice) ----
        await asyncio.gather(invite(C, 'bob'), invite(D, 'erin'))
        await asyncio.gather(accept(B), accept(E))
        await connected([A, B, C, D, E], 4, 'S4 fünf Personen im Vollmesh', timeout=70)
        for u in (A, B, C, D, E):
            assert await u.page.is_disabled('#v455-invite-btn') or await u.page.locator('#v455-invite-select option').count() == 1, 'bei 5 Personen kein weiteres Einladen'
        print('S4 ok: 5 Personen vollständig verbunden (10 Verbindungen), Limit greift')

        # ---- S5: alle legen nacheinander auf, letzter Anruf endet automatisch ----
        for u in (E, D, C):
            await leave(u); await hung_up(u)
        await connected([A, B], 1, 'S5 alice+bob als Rest')
        await leave(A); await hung_up(A)
        await hung_up(B)  # bob sieht „aufgelegt“ und beendet selbst
        print('S5 ok: Rest-Anruf endet, wenn niemand mehr da ist')

        # ---- S6: Nicht-Freunde in einer Gruppe: keine Verbindung, klare Meldung, kein Dauerfeuer ----
        cloud.unfriend('bob', 'carol')
        mark = len(cloud.signals)  # nur Signale ab hier zählen
        await A.page.evaluate("window.dispatchEvent(new Event('online'))"); await B.page.evaluate("window.dispatchEvent(new Event('online'))"); await C.page.evaluate("window.dispatchEvent(new Event('online'))")
        await asyncio.sleep(6)  # Freundeslisten neu laden
        await A.open_chat('bob'); await A.page.click('#call-voice'); await accept(B)
        await connected([A, B], 1, 'S6a alice↔bob')
        await invite(A, 'carol'); await accept(C)
        async def not_friends_note():
            return 'nicht befreundet' in (await C.toast()) or 'nicht befreundet' in (await B.toast())
        await wait_until(not_friends_note, 'Hinweis auf fehlende Freundschaft', 40)
        st = await A.state()
        assert sorted(p['user'] for p in st['peers']) == ['bob', 'carol'], st
        cloud_offers = [s for s in cloud.signals[mark:] if s['kind'] == 'offer' and {s['from'], s['to']} == {'carol', 'bob'}]
        assert len(cloud_offers) == 0, 'keine verbotenen Signale im Server'
        for u in (C, B, A):
            await leave(u)
        await hung_up(A)
        print('S6 ok: Nicht-Freunde werden sauber gemeldet, alice bleibt mit beiden verbunden')

        # ---- S7: ältere Version (antwortet ohne squadCall) ----
        await asyncio.sleep(2)
        await A.open_chat('bob'); await A.page.click('#call-voice'); await accept(B)
        await connected([A, B], 1, 'S7a alice↔bob')
        # alice lädt „lena“ ein; lena ist eine ältere Version und antwortet ohne Gruppeninfo
        await wait_until(lambda: A.page.evaluate("[...document.querySelectorAll('#v455-invite-select option')].some(o=>o.value==='lena')"), 'lena in der Liste', 20)
        await A.page.select_option('#v455-invite-select', 'lena'); await A.page.click('#v455-invite-btn')
        async def offer_to_lena(): return next((s for s in cloud.signals if s['kind'] == 'offer' and s['to'] == 'lena'), None)
        async def hangup_to_lena(): return next((s for s in cloud.signals if s['kind'] == 'hangup' and s['to'] == 'lena'), None)
        offer = await wait_until(offer_to_lena, 'Angebot an lena', 15)
        legacy_sdp = await A.page.evaluate("""async (sdp) => {
            const pc = new RTCPeerConnection(); const m = await navigator.mediaDevices.getUserMedia({audio:true});
            m.getTracks().forEach(t => pc.addTrack(t, m));
            await pc.setRemoteDescription({type:'offer', sdp}); const a = await pc.createAnswer(); return a.sdp; }""", offer['data']['sdp'])
        cloud.push('lena', 'alice', 'answer', {'type': 'answer', 'sdp': legacy_sdp})
        await wait_until(hangup_to_lena, 'alice weist die ältere Version ab', 20)
        assert 'ältere' in await A.toast()
        st = await A.state()
        assert [p['user'] for p in st['peers']] == ['bob'], st
        print('S7 ok: ältere Version wird im Gruppenanruf sauber abgelehnt, 1:1 mit bob bleibt')
        await leave(A); await hung_up(A); await hung_up(B)

        # ---- keine JS-Fehler in irgendeinem Fenster ----
        for u in users.values():
            assert not u.errors, (u.name, u.errors)
        print('PASS Gruppenanrufe mit echtem WebRTC: 1:1, Wachstum bis 5, gleichzeitige Einladungen, Verlassen, Nicht-Freunde, ältere Version; keine JS-Fehler')

asyncio.run(main())
