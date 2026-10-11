from pathlib import Path
from playwright.sync_api import sync_playwright
import re
root=Path(__file__).parent
html=(root/'web/index.html').read_text()
for m in list(re.finditer(r'<link rel="stylesheet" href="([^"]+)">',html)):
 html=html.replace(m.group(0),'<style>'+(root/'web'/m.group(1)).read_text()+'</style>')
for m in list(re.finditer(r'<script src="([^"]+)"></script>',html)):
 html=html.replace(m.group(0),'<script>'+(root/'web'/m.group(1)).read_text().replace('</script>','<\\/script>')+'</script>')
mock=r'''(() => {
const old={version:2,preset:'gaming',order:['system','monitor','games','network','news','clockwidget','musicwidget','friendswidget','sessionwidget'],hidden:['clockwidget','musicwidget','friendswidget','sessionwidget'],wide:['news'],cozyStart:true,smartGaming:true,glassStyle:'frosted',readAloud:false,voiceRate:1,voiceName:'',spotifyDock:'hidden'};
const store=new Map();const storage={getItem:k=>store.get(k)||null,setItem:(k,v)=>store.set(k,String(v)),removeItem:k=>store.delete(k),clear:()=>store.clear()};
Object.defineProperty(window,'localStorage',{configurable:true,value:storage});Object.defineProperty(window,'sessionStorage',{configurable:true,value:storage});
window.__calls=[];window.__theme={autoDayNight:false,nightFrom:20,dayFrom:7,nightGlass:'midnight',reducedMotion:false};window.__workspace=structuredClone(old);window.__connected=true;
window.fetch=async (url,opts={})=>{
 const path=String(url).replace(/^api\//,'').split('?')[0];window.__calls.push({path,method:opts.method||'GET',body:opts.body?JSON.parse(opts.body):null});
 let data={},code=200;
 if(path==='state')data={hardware:{cpu:17.1,ram:35.2,gpu:9,cpuName:'Test CPU',gpuName:'GeForce',totalRam:32,usedRam:10},config:{server:'play.hugosmp.de',gamePaths:{}},usage:{},playing:{},speed:{running:false},server:{online:false,error:'Offline'}};
 else if(path==='spotify/status')data={connected:window.__connected,hasClientId:window.__connected,desktopControls:true,bookmarks:[]};
 else if(path==='spotify/now')data={active:true,playing:true,title:'Demo-Musik',artists:'Testartist',duration:220000,progress:40000,device:'Desktop',volume:40,shuffle:false,repeat:'off'};
 else if(path==='spotify/queue')data={items:[{title:'Song Nummer Eins',artists:'Band',uri:'spotify:track:5yEPktRqvIhko5QFF3aBH3'},{title:'Song Nummer Zwei',artists:'Band 2',uri:'spotify:track:5yEPktRqvIhko5QFF3aBH3'}]};
 else if(path==='spotify/playback-options')data={message:'An Spotify übergeben'};
 else if(path==='spotify/playlists')data={playlists:[{id:'3cEYpjA9oz9GiPac4AsH4n',name:'Meine Playlist',total:2}]};
 else if(path==='spotify/playlist-items')data={id:'3cEYpjA9oz9GiPac4AsH4n',total:2,offset:0,nextOffset:null,tracks:[{position:0,id:'5yEPktRqvIhko5QFF3aBH3',name:'Song A',artists:'Band A',album:'Album',duration:120000,playable:true},{position:1,id:'5yEPktRqvIhko5QFF3aBH3',name:'Song B',artists:'Band B',album:'Album',duration:120000,playable:true}]};
 else if(path==='workspace'){if(opts.method==='POST'){window.__workspace=JSON.parse(opts.body);data={saved:true}}else data=window.__workspace}
 else if(path==='library')data={games:[{id:'custom-one',name:'Test Game',source:'Eigene EXE',installed:true,path:'C:\\Games\\Test.exe'},{id:'steam-42',name:'Portal 2',source:'Steam',installed:true,appId:'620'}]};
 else if(path==='library/pro')data={activity:{'custom-one':{seconds:1530,lastLaunched:'2026-10-09T15:00:00Z'},'steam-42':{}}};
 else if(path==='studio')data={blur:18,radius:18,tint:55,outline:18};
 else if(path==='theme-pro'){if(opts.method==='POST'){window.__theme=JSON.parse(opts.body);data={saved:true}}else data=window.__theme}
 else if(path==='assistant/context')data={platform:'windows',version:'4.3.0',cpuPercent:17.1,ramPercent:35.2,gpuAvailable:true,gpuPercent:9,overlayEnabled:false,spotifyConnected:true,smartGamingEnabled:true,libraryGames:2,speedtestRunning:false};
 else if(path==='appearance')data={color:'oak',scene:'oak',cozy:'cozy',motion:true,speed:'slow',brightness:100,dim:0,vignette:20,density:'normal'};
 else if(path==='overlay')data={config:{cpu:true,gpu:true,ram:true,gpuTemp:true,ping:true,music:true,clock:true,opacity:88,scale:100,x:18,y:18},visible:false,supported:true};
 else if(path==='clock')data={};
 else if(path==='maintenance')data={version:'4.3.0',repository:'',autoCheck:false};
 else if(path==='diagnostics')data={version:'4.3.0',system:'Windows Test',gpuDataAvailable:true,configuredLauncherCount:0,userAddedGameCount:1};
 else if(path==='assistant/history')data={messages:[]};
 else if(path==='news')data={items:[]};
 else if(path==='spotify/bookmarks')data={connected:true,hasClientId:true,desktopControls:true,bookmarks:[]};
 return new Response(JSON.stringify(data),{status:code,headers:{'Content-Type':'application/json'}});
};
})();'''
html=html.replace('</head>','<script>'+mock+'</script></head>')
with sync_playwright() as playwright:
 browser=playwright.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox','--disable-web-security'])
 for width in [1440,900]:
  page=browser.new_page(viewport={'width':width,'height':960}); errors=[];page.on('pageerror',lambda error:errors.append(str(error)))
  print('page start',width,flush=True)
  page.set_content(html,wait_until='load');page.wait_for_timeout(1100)
  # legacy cozy and dashboard editor must still work
  assert page.locator('#v42-cozy-welcome').is_visible()
  # v4.7 clean navigation: KI/Soundboard are nested under Tools, Arcade remains a direct tab.
  tools=page.locator('#nav .v41-nav-group[data-v43-group="tools"]')
  assert tools.count()==1 and tools.locator('[data-tab="assistant"]').count()==1 and tools.locator('[data-tab="soundboard"]').count()==1
  assert page.locator('#nav > button[data-tab="minigames"]').count()==1
  assert page.locator('#nav > button[data-tab="overlay"]').count()==0 and page.locator('#overlay').count()==0
  page.locator('#nav > button[data-tab="minigames"]').click()
  assert page.locator('#minigames').evaluate('(e)=>e.classList.contains("active")')
  page.locator('#nav > button[data-tab="settings"]').click()
  overlay_group=page.locator('#overlay-settings-group')
  assert overlay_group.count()==1 and not overlay_group.evaluate('(e)=>e.open')
  overlay_group.locator('summary').click()
  assert overlay_group.evaluate('(e)=>e.open') and page.locator('#overlay-settings #overlay-opacity').count()==1
  assert not page.locator('.v41-card-editbar').first.is_visible(),'editbar shown without editing'
  print('loaded ui',width,flush=True)
  page.locator('[data-tab="spotify"]').click();page.wait_for_timeout(550)
  assert page.locator('#v43-music-center').is_visible()
  page.locator('#v43-queue-refresh').click();page.wait_for_timeout(230)
  assert page.locator('.v43-queue-row').count()==2
  page.locator('#v43-shuffle').click();page.wait_for_timeout(220)
  assert page.locator('#v43-shuffle').get_attribute('aria-pressed')=='true'
  page.locator('#v43-repeat').click();page.wait_for_timeout(180)
  assert 'Playlist' in page.locator('#v43-repeat').inner_text()
  page.locator('#spotify-playlists .spotify-browse').first.click();page.wait_for_timeout(250)
  assert page.locator('#spotify-track-list .spotify-track-queue').count()==2
  page.locator('#spotify-track-list .spotify-track-queue').first.click();page.wait_for_timeout(300)
  assert page.evaluate("window.__calls.filter(x=>x.path==='spotify/playback-options').at(-1).body.action")=='enqueue'
  page.locator('.v41-nav-group').first.locator('summary').click();page.locator('[data-tab="library"]').click();page.wait_for_timeout(430)
  print('LIB STATS:',page.locator('.v43-library-stats').all_inner_texts(),'STATUS:',page.locator('#library-status').inner_text(),flush=True)
  assert any('25 Min' in x for x in page.locator('.v43-library-stats').all_inner_texts())
  page.locator('#v43-library-sort').select_option('time')
  assert page.locator('.v4-game-details strong').first.inner_text()=='Test Game'
  page.locator('#nav .v41-nav-group[data-v43-group="tools"] > summary').click(); page.locator('[data-tab="assistant"]').click();page.locator('#v43-ai-check').click();page.wait_for_timeout(250)
  assert 'CPU 17%' in page.locator('#v43-ai-summary').inner_text()
  page.locator('#v43-ai-explain').click();assert 'CPU 17%' in page.locator('#assistant-input').input_value()
  page.locator('[data-tab="settings"]').click();page.wait_for_timeout(180)
  page.locator('#v43-auto-theme').check();page.locator('#v43-reduce-motion').check()
  page.locator('#v43-night-from').select_option('21');page.locator('#v43-day-from').select_option('8')
  page.locator('#v43-night-style').select_option('warm');page.locator('#v43-theme-save').click();page.wait_for_timeout(190)
  assert page.evaluate('window.__theme.autoDayNight && window.__theme.reducedMotion && window.__theme.nightGlass===\'warm\'')
  assert page.locator('body').evaluate('(el)=>el.classList.contains("v43-reduced-motion")')
  assert not errors,errors
  page.locator('[data-tab="spotify"]').click();page.screenshot(path=f'/mnt/data/SquadBase_v43_MusicCenter_{width}.png')
  print('PASS: width',width,'Spotify queue/shuffle/repeat/enqueue, library, AI, theme, legacy navigation; JS errors:',errors)
  page.close()
 browser.close()
