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
window.__remote={enabled:false,url:'',pin:'',expires:new Date(Date.now()+300000).toISOString()};window.__calls=[];window.__theme={autoDayNight:false,nightFrom:20,dayFrom:7,nightGlass:'midnight',reducedMotion:false};window.__workspace=structuredClone(old);window.__connected=true;
window.fetch=async (url,opts={})=>{
 const path=String(url).replace(/^api\//,'').split('?')[0];window.__calls.push({path,method:opts.method||'GET',body:opts.body && typeof opts.body==='string'?JSON.parse(opts.body):null});
 let data={},code=200;
 if(path==='soundboard')data={clips:[{id:'1234567890abcdef1234',name:'Demoklang',type:'audio/mpeg',size:1234}]};
 else if(path==='remote/events')data={events:[]};
 else if(path==='remote'){if(opts.method==='POST'){const q=JSON.parse(opts.body||'{}');window.__remote.enabled=q.action!=='stop';window.__remote.url='http://192.168.1.22:54321/';window.__remote.pin='424242'}data=window.__remote}
 else if(path==='state')data={hardware:{cpu:17.1,ram:35.2,gpu:9,cpuName:'Test CPU',gpuName:'GeForce',totalRam:32,usedRam:10},config:{server:'play.hugosmp.de',gamePaths:{}},usage:{},playing:{},speed:{running:false},server:{online:false,error:'Offline'}};
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
 for width in [1440,900,640]:
  page=browser.new_page(viewport={'width':width,'height':960});errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
  print('loading',width,flush=True)
  page.set_content(html,wait_until='load');page.wait_for_timeout(900)
  assert not page.locator('.v41-card-editbar').first.is_visible(),'legacy edit buttons visible'
  page.locator('[data-tab="soundboard"]').click();page.wait_for_timeout(220)
  assert page.locator('#v44-board').is_visible()
  assert 'Demoklang' in page.locator('#v44-sound-list').inner_text()
  page.locator('[data-tab="settings"]').click();page.wait_for_timeout(160)
  page.locator('[data-v44-preset="ember"]').click()
  assert page.locator('body').get_attribute('data-v44-glass')=='ember'
  assert page.locator('[data-v44-preset="ember"]').get_attribute('aria-pressed')=='true'
  # Remote tab is inside the secondary tools group, not among the permanent core tabs.
  groups=page.locator('.v41-nav-group')
  tools=groups.last
  if not tools.evaluate('(el)=>el.open'):tools.locator('summary').click()
  page.locator('[data-tab="remote"]').click();page.wait_for_timeout(120)
  assert page.locator('#v44-remote-start').is_visible()
  page.locator('#v44-remote-start').click();page.wait_for_timeout(170)
  assert page.locator('#v44-remote-pin').inner_text()=='424242'
  assert page.locator('#v44-remote-stop').is_visible()
  page.locator('#v44-remote-stop').click();page.wait_for_timeout(140)
  assert page.locator('#v44-remote-start').is_visible()
  assert not errors,errors
  if width==1440:page.screenshot(path='/mnt/data/SquadBase_v44_Fernbedienung_Vorschau.png')
  if width==900:
   page.locator('[data-tab="soundboard"]').click();page.wait_for_timeout(120)
   page.screenshot(path='/mnt/data/SquadBase_v44_Soundboard_Vorschau.png')
  print('PASS width',width,'Soundboard, Remote, Theme, legacy Dashboard, JS errors:',errors,flush=True)
  page.close()
 browser.close()
