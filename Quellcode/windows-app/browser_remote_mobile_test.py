from pathlib import Path
from playwright.sync_api import sync_playwright
root=Path(__file__).parent
html=(root/'web/remote.html').read_text().replace('<link rel="stylesheet" href="/remote.css">','<style>'+(root/'web/remote.css').read_text()+'</style>')
mock='''<script>
window.__remoteCalls=[];
window.fetch=async (path,opts={})=>{
 const p=String(path).replace('/api/','');let status=200;let result={};window.__remoteCalls.push({p,opts});
 if(p==='pair'){const code=JSON.parse(opts.body).code;if(code==='123456')result={token:'0123456789abcdef0123456789abcdef0123456789abcdef'};else {status=403;result={error:'PIN falsch'}}}
 else if(p==='state')result={status:'connected',clips:[{id:'1234567890abcdef1234',name:'Test Glocke'}]};
 else if(p==='command')result={status:'ok'};
 return new Response(JSON.stringify(result),{status,headers:{'Content-Type':'application/json'}})
};
</script>'''
html=html.replace('</head>',mock+'</head>').replace('<script src="/remote.js" defer></script>','<script>'+(root/'web/remote.js').read_text()+'</script>')
with sync_playwright() as p:
 browser=p.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox','--disable-web-security'])
 page=browser.new_page(viewport={'width':390,'height':844},device_scale_factor=1,is_mobile=True,has_touch=True)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.set_content(html,wait_until='load');page.wait_for_timeout(250)
 assert page.locator('#pair').is_visible();assert not page.locator('#controls').is_visible()
 page.locator('#pin').fill('111111');page.locator('#connect').click();page.wait_for_timeout(100)
 assert 'PIN falsch' in page.locator('#pair-status').inner_text()
 page.locator('#pin').fill('123456');page.locator('#connect').click();page.wait_for_timeout(170)
 assert page.locator('#controls').is_visible()
 assert 'Test Glocke' in page.locator('#sounds').inner_text()
 page.locator('[data-action="sound"]').click();page.wait_for_timeout(90)
 assert page.evaluate('window.__remoteCalls.some(x=>x.p==="command" && JSON.parse(x.opts.body).action==="sound")')
 assert not errors,errors
 page.screenshot(path='/mnt/data/SquadBase_v44_Handy_Vorschau.png',full_page=True)
 print('PASS mobile 390px: PIN failure/success, sound list, command, no browser JS errors')
 browser.close()
