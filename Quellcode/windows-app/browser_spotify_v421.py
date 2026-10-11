from pathlib import Path
from playwright.sync_api import sync_playwright
import re
root=Path(__file__).parent
html=(root/'web/index.html').read_text()
for match in list(re.finditer(r'<link rel="stylesheet" href="([^"]+)">',html)):
    html=html.replace(match.group(0),'<style>'+(root/'web'/match.group(1)).read_text()+'</style>')
for match in list(re.finditer(r'<script src="([^"]+)"></script>',html)):
    html=html.replace(match.group(0),'<script>'+(root/'web'/match.group(1)).read_text().replace('</script>','<\\/script>')+'</script>')
mock=r'''(()=>{
const store=new Map();Object.defineProperty(window,'localStorage',{configurable:true,value:{getItem:k=>store.get(k)||null,setItem:(k,v)=>store.set(k,String(v)),removeItem:k=>store.delete(k)}});Object.defineProperty(window,'sessionStorage',{configurable:true,value:{getItem:k=>store.get(k)||null,setItem:(k,v)=>store.set(k,String(v)),removeItem:k=>store.delete(k)}});
window.spotifyCommands=[];window.__auth=false;
window.fetch=async(url,opts={})=>{
 let p=String(url).replace(/^api\//,'');let result={};
 if(p==='state')result={hardware:{cpu:18,ram:36,gpu:21,cpuName:'Test CPU',gpuName:'Test GPU',totalRam:32,usedRam:12},config:{server:'',gamePaths:{}},usage:{},playing:{},speed:{running:false},server:{online:false,error:'Offline'}}
 else if(p==='spotify/status')result={connected:window.__auth,hasClientId:window.__auth,desktopControls:true,bookmarks:[]}
 else if(p==='spotify/desktop'){window.spotifyCommands.push(JSON.parse(opts.body));result={message:'Windows-Medientaste gesendet'}}
 else if(p==='spotify/now')result={active:true,playing:true,title:'Beispieltitel',artists:'Beispielkünstler',duration:210000,progress:35000,device:'Spotify PC',volume:43}
 else if(p==='spotify/playlists')result={playlists:[{id:'37i9dQZF1DXcBWIGoYBM5M',name:'Hits',total:15}]}
 else if(p==='spotify/control'){window.spotifyCommands.push(JSON.parse(opts.body));result={message:'Spotify-Befehl gesendet'}}
 else if(p==='workspace')result={version:2,preset:'gaming',order:['system','monitor','games','network','news','clockwidget','musicwidget','friendswidget','sessionwidget'],hidden:['clockwidget','musicwidget','friendswidget','sessionwidget'],wide:['news'],cozyStart:true,smartGaming:false,glassStyle:'frosted',spotifyDock:'bar'}
 else if(p==='studio')result={blur:18,radius:18,tint:55,outline:18}
 else if(p==='appearance')result={color:'oak',scene:'oak',cozy:'cozy',motion:true,speed:'slow',brightness:100,dim:0,vignette:20,density:'normal'}
 else if(p==='clock'||p==='overlay'||p==='library'||p==='ping')result={}
 else if(p==='maintenance')result={version:'4.2.1',repository:'',autoCheck:false}
 else if(p==='assistant/history')result={messages:[]}
 else if(p==='news')result={items:[]};
 return new Response(JSON.stringify(result),{status:200,headers:{'Content-Type':'application/json'}});
};})();'''
html=html.replace('</head>','<script>'+mock+'</script></head>')
with sync_playwright() as p:
 b=p.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox','--disable-web-security'])
 page=b.new_page(viewport={'width':1400,'height':900});errors=[]
 page.on('pageerror',lambda e:errors.append(str(e)))
 page.set_content(html,wait_until='load');page.wait_for_timeout(900)
 page.locator('[data-tab="spotify"]').click();page.wait_for_timeout(350)
 assert page.locator('#spotify-desktop-card').count()==0  # not a duplicate widget
 assert page.locator('#spotify-desktop-open').is_visible()
 assert not page.locator('#spotify-desktop-toggle').is_disabled()
 print('DEBUG MINI',page.evaluate('({hidden:document.getElementById(\"spotify-mini\").hidden, dock:document.getElementById(\"spotify-mini\").dataset.dock, connected:spotifyConnected, desktop:spotifyDesktopControls, ready:spotifyReady, html:document.getElementById(\"spotify-mini\").outerHTML.slice(0,220)})'), errors)
 assert page.locator('#spotify-mini').is_visible()
 page.locator('#spotify-desktop-toggle').click();page.wait_for_timeout(100)
 assert page.evaluate('window.spotifyCommands.at(-1).action')=='toggle'
 page.locator('#spotify-mini-next').click();page.wait_for_timeout(100)
 assert page.evaluate('window.spotifyCommands.at(-1).action')=='next'
 assert 'Windows-Medientaste' in page.locator('#spotify-desktop-status').inner_text()
 page.screenshot(path='/mnt/data/SquadBase_v421_Spotify_Fix_Vorschau.png')
 # Simulate a fully authorised API session as distinct from the desktop fallback.
 page.evaluate('window.__auth=true');page.evaluate('spotifyLoadState()');page.wait_for_timeout(700)
 assert page.locator('#spotify-song').inner_text()=='Beispieltitel'
 assert page.locator('#spotify-playlists').get_by_text('Hits').count()==0 # loaded only when tab opened
 page.evaluate('spotifyLoadPlaylists(true)');page.wait_for_timeout(250)
 assert page.locator('#spotify-playlists').get_by_text('Hits').is_visible()
 page.locator('#spotify-play').click();page.wait_for_timeout(900)
 assert page.evaluate('window.spotifyCommands.at(-1).action')=='pause'
 assert not errors,errors
 print('PASS Spotify UI: disconnected Windows fallback, global media action, saved controls, authenticated metadata/playlists, API pause; JS errors:',errors)
 b.close()
