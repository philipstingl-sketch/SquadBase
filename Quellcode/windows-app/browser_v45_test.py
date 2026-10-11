from pathlib import Path
from playwright.sync_api import sync_playwright
import re
root=Path(__file__).parent
html=(root/'web/index.html').read_text()
for m in list(re.finditer(r'<link rel="stylesheet" href="([^"]+)">',html)):
 html=html.replace(m.group(0),'<style>'+(root/'web'/m.group(1)).read_text()+'</style>')
for m in list(re.finditer(r'<script src="([^"]+)"></script>',html)):
 html=html.replace(m.group(0),'<script>'+(root/'web'/m.group(1)).read_text().replace('</script>','<\\/script>')+'</script>')
old=(root/'browser_v44_test.py').read_text()
mock=old.split("mock=r'''",1)[1].split("'''",1)[0]
mock=mock.replace("window.__connected=true;","window.__connected=true;window.__glass45={preset:'original',blur:18,opacity:68,radius:20,glow:true,ambient:true};")
mock=mock.replace("else if(path==='theme-pro')", "else if(path==='glass45'){if(opts.method==='POST'){window.__glass45=JSON.parse(opts.body);data={saved:true}}else data=window.__glass45;}\n else if(path==='theme-pro')")
html=html.replace('</head>','<script>'+mock+'</script></head>')
with sync_playwright() as p:
 browser=p.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox','--disable-web-security'])
 for width in [1440,900,640]:
  pg=browser.new_page(viewport={'width':width,'height':930},device_scale_factor=1)
  pg.set_default_timeout(10000)
  errors=[];pg.on('pageerror',lambda err:errors.append(str(err)))
  pg.set_content(html,wait_until='domcontentloaded');pg.wait_for_timeout(600)
  assert not pg.locator('.v41-card-editbar').first.is_visible(),'Dashboard edit controls visible'
  pg.locator('[data-tab="settings"]').click();pg.wait_for_timeout(150)
  assert pg.locator('#v45-glass-studio').is_visible(),'glass studio invisible'
  assert pg.locator('[data-v44-preset="forest"]').count()==1,'old theme presets missing'
  pg.locator('[data-v45-profile="boreal"]').click()
  assert pg.locator('body').get_attribute('data-v45-profile')=='boreal'
  pg.locator('#v45-blur').fill('14')
  assert pg.locator('#v45-blur-label').inner_text()=='14 px'
  pg.locator('#v45-undo').click()
  assert pg.locator('body').get_attribute('data-v45-profile') is None,'undo did not revert'
  pg.locator('[data-v45-profile="moonstone"]').click()
  pg.locator('#v45-radius').fill('24')
  pg.locator('#v45-save').click();pg.wait_for_timeout(180)
  assert pg.evaluate('window.__glass45.preset')=='moonstone','backend not saved'
  assert pg.evaluate('window.__glass45.radius')==24,'radius not saved'
  assert 'Gespeichert' in pg.locator('#v45-status').inner_text()
  assert not errors,errors
  overflow=pg.evaluate('document.documentElement.scrollWidth - window.innerWidth')
  assert overflow<=12, f'horizontal overflow at {width}: {overflow}'
  if width==1440:
   pg.screenshot(path='/mnt/data/squadbase_v45_release/SquadBase_v45_ThemeStudio.png',full_page=False)
   pg.locator('[data-tab="dashboard"]').click();pg.wait_for_timeout(200)
   pg.screenshot(path='/mnt/data/squadbase_v45_release/SquadBase_v45_Dashboard.png',full_page=False)
  if width==900:pg.screenshot(path='/mnt/data/squadbase_v45_release/SquadBase_v45_ThemeStudio_Kompakt.png',full_page=False)
  print('PASS',width,'save, restore, preset, old settings, no JS error, overflow',overflow,flush=True)
  pg.close()
 browser.close()
