from playwright.sync_api import sync_playwright
from pathlib import Path
import re, json
web=Path(__file__).parent/'web'
html=(web/'index.html').read_text()
for mat in list(re.finditer(r'<link rel="stylesheet" href="([^"]+)">',html)):
 html=html.replace(mat.group(0),'<style>'+(web/mat.group(1)).read_text()+'</style>')
for mat in list(re.finditer(r'<script src="([^"]+)"></script>',html)):
 html=html.replace(mat.group(0),'<script>'+(web/mat.group(1)).read_text().replace('</script>','<\\/script>')+'</script>')
MOCK=r'''(() => {
 const old={version:2,preset:'gaming',order:['system','monitor','games','network','news','clockwidget','musicwidget','friendswidget','sessionwidget'],hidden:['clockwidget','musicwidget','friendswidget','sessionwidget'],wide:['news'],cozyStart:true,smartGaming:true,glassStyle:'frosted',readAloud:false,voiceRate:1,voiceName:'',spotifyDock:'bar'};
 window.__workspace=structuredClone(old);window.__writeCount=0;window.__playing=false;
 const data=new Map(),mock={getItem:k=>data.has(k)?data.get(k):null,setItem:(k,v)=>data.set(k,String(v)),removeItem:k=>data.delete(k),clear:()=>data.clear()};
 Object.defineProperty(window,'localStorage',{configurable:true,value:mock});Object.defineProperty(window,'sessionStorage',{configurable:true,value:mock});
 window.fetch=async (url,options={})=>{
 const x=String(url);if(!x.startsWith('api/'))return new Response('{}',{status:200});
 const path=x.slice(4).split('?')[0];let response={};
 if(path==='workspace'){if(options.method==='POST'){window.__writeCount++;window.__workspace=JSON.parse(options.body);response={saved:true}}else response=structuredClone(window.__workspace)}
 else if(path==='state')response={hardware:{cpu:22,ram:38,gpu:19,cpuName:'Intel Core i7',gpuName:'NVIDIA GeForce',usedRam:12,totalRam:32,host:'Test-PC'},config:{server:'play.hugosmp.de',gamePaths:{}},usage:{minecraft:1320},playing:{minecraft:window.__playing},speed:{running:false},server:{online:false,error:'Offline'}};
 else if(path==='spotify/status')response={connected:false,bookmarks:[]};
 else if(path==='spotify/now')response={active:false};
 else if(path==='spotify/playlists')response={playlists:[]};
 else if(path==='studio')response={blur:18,radius:18,tint:55,outline:18,customImage:false};
 else if(path==='appearance')response={color:'oak',scene:'oak',cozy:'cozy',motion:true,speed:'slow',brightness:100,dim:0,vignette:20,density:'normal'};
 else if(path==='overlay')response={}; else if(path==='clock')response={};
 else if(path==='assistant/history')response={messages:[]};
 else if(path==='library')response={games:[]};else if(path==='ping')response={};
 else if(path==='maintenance')response={version:'4.2.0',repository:'',autoCheck:false};
 else if(path==='diagnostics')response={version:'4.2.0',system:'Windows',gpuDataAvailable:true,configuredLauncherCount:0,userAddedGameCount:0};
 else if(path==='news')response={items:[]};
 return new Response(JSON.stringify(response),{status:200,headers:{'Content-Type':'application/json'}});
 };
})();'''
html=html.replace('</head>','<script>'+MOCK+'</script></head>')
with sync_playwright() as p:
 browser=p.chromium.launch(executable_path='/usr/bin/chromium',headless=True,args=['--no-sandbox','--disable-web-security'])
 page=browser.new_page(viewport={'width':1440,'height':920},device_scale_factor=1)
 errors=[]
 page.on('pageerror',lambda e:errors.append(str(e)))
 page.set_content(html,wait_until='load')
 page.wait_for_timeout(1300)
 assert page.locator('#v41-dashboard-board > article').count()==9, 'Nine dashboard cards required'
 assert page.locator('#v42-cozy-welcome').is_visible()
 assert not page.locator('.v41-card-editbar').first.is_visible(), 'No permanent edit tools'
 assert page.locator('#v42-edit-options').is_hidden()
 assert page.locator('#v42-hero-clock').inner_text()!='--:--'
 assert page.locator('#v42-widget-session').inner_text()=='Kein Spiel erkannt'
 page.evaluate("document.getElementById('toast').className=''")
 page.screenshot(path='/mnt/data/SquadBase_v42_Dashboard_Vorschau.png')
 # Editing should be draft-only and no persistence until Save
 page.locator('#v41-edit-toggle').click()
 assert page.locator('#v42-edit-options').is_visible()
 assert page.locator('.v41-card-editbar').first.is_visible()
 page.locator('#v42-widget-select').select_option('clockwidget')
 page.locator('#v42-widget-add').click()
 assert 'v41-is-hidden' not in page.locator('[data-dashboard-card="clockwidget"]').get_attribute('class')
 assert page.evaluate('window.__writeCount')==0
 page.locator('#v42-cancel').click()
 assert 'v41-is-hidden' in page.locator('[data-dashboard-card="clockwidget"]').get_attribute('class')
 assert page.locator('#v42-edit-options').is_hidden()
 assert page.evaluate('window.__writeCount')==0
 # Save a widget + preset
 page.locator('#v41-edit-toggle').click()
 page.locator('#v41-layout-preset').select_option('cozy')
 page.locator('#v42-save').click();page.wait_for_timeout(160)
 assert page.evaluate('window.__writeCount')==1
 assert page.evaluate('window.__workspace.preset')=='cozy'
 assert page.locator('#v42-edit-options').is_hidden()
 page.screenshot(path='/mnt/data/SquadBase_v42_Widgets_Vorschau.png')
 # Existing sections still reachable
 page.locator('button[data-tab="settings"]').click()
 page.locator('#v42-start-toggle').uncheck();page.wait_for_timeout(200)
 assert page.locator('#v42-cozy-welcome').is_hidden()
 assert page.evaluate('window.__workspace.cozyStart')==False
 page.locator('#v42-start-toggle').check()
 page.locator('#v42-smart-toggle').check()
 page.evaluate('window.__playing=true')
 page.wait_for_timeout(6500) # real poll interval and smart watcher
 print("SMART DEBUG",page.evaluate("({playing:window.__playing, state:typeof state!=='undefined'?state.playing:null, class:document.body.className, status:document.getElementById('v42-smart-status').textContent, checked:document.getElementById('v42-smart-toggle').checked, errors: []})"),"JS",errors)
 assert page.locator('body').evaluate("e=>e.classList.contains('v42-smart-active')"),'Smart gaming should activate with detected game'
 page.locator('#v42-smart-toggle').uncheck();page.wait_for_timeout(150)
 assert not page.locator('body').evaluate("e=>e.classList.contains('v42-smart-active')")
 page.locator('[data-glass="warm"]').click();page.wait_for_timeout(160)
 assert page.locator('body').get_attribute('data-glass-style')=='warm'
 assert page.evaluate('window.__workspace.glassStyle')=='warm'
 page.screenshot(path='/mnt/data/SquadBase_v42_Einstellungen_Vorschau.png')
 page.locator('button[data-tab="minigames"]').click()
 assert page.locator('#minigames').evaluate("e=>e.classList.contains('active')")
 # Confirms the game still has JS mounted
 assert page.locator('canvas').count()>0
 print('PASS v4.2: 9 cards, normal mode clean, widget add/save/cancel, existing navigation, cozy hero, smart gaming, minigame tab')
 print('Errors:', errors)
 assert not errors,errors
 browser.close()
