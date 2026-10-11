from pathlib import Path
from playwright.sync_api import sync_playwright
import re
root=Path(__file__).parent
html=(root/'web/index.html').read_text()
for match in list(re.finditer(r'<link rel="stylesheet" href="([^"]+)">',html)):
    html=html.replace(match.group(0),'<style>'+(root/'web'/match.group(1)).read_text()+'</style>')
for match in list(re.finditer(r'<script src="([^"]+)"></script>',html)):
    html=html.replace(match.group(0),'<script>'+(root/'web'/match.group(1)).read_text().replace('</script>','<\\/script>')+'</script>')
mock='''(() => {
const store=new Map();Object.defineProperty(window,'localStorage',{configurable:true,value:{getItem:k=>store.get(k)||null,setItem:(k,v)=>store.set(k,String(v)),removeItem:k=>store.delete(k)}});Object.defineProperty(window,'sessionStorage',{configurable:true,value:{getItem:k=>store.get(k)||null,setItem:(k,v)=>store.set(k,String(v)),removeItem:k=>store.delete(k)}});
window.spotifyCommands=[];window.__auth=true;window.__forbidden=false;window.__playlistCalls=[];
window.fetch=async(url,opts={})=>{
 const u=String(url).replace(/^api\\//,'');let result={};let code=200;
 if(u==='state')result={hardware:{cpu:19,ram:32,gpu:8,cpuName:'Test CPU',gpuName:'Test GPU',totalRam:32,usedRam:10},config:{server:'play.hugosmp.de',gamePaths:{}},usage:{},playing:{},speed:{running:false},server:{online:false,error:'Offline'}};
 else if(u==='spotify/status')result={connected:window.__auth,hasClientId:window.__auth,desktopControls:true,bookmarks:[{id:'4s4odYwQo9Cykw1yuw7DxI',name:'Freunde-Playlist',url:'https://open.spotify.com/playlist/4s4odYwQo9Cykw1yuw7DxI'}]};
 else if(u==='spotify/desktop')result={message:'Medientaste gesendet'};
 else if(u==='spotify/now')result={active:true,playing:true,title:'Läuft gerade',artists:'Ein Künstler',duration:220000,progress:45000,device:'PC',volume:55};
 else if(u==='spotify/playlists')result={playlists:[{id:'3cEYpjA9oz9GiPac4AsH4n',name:'Meine Lieblingssongs',total:70,cover:''},{id:'4s4odYwQo9Cykw1yuw7DxI',name:'Meine zweite Liste',total:1}]};
 else if(u.startsWith('spotify/playlist-items?')){ const sp=new URLSearchParams(u.split('?')[1]);let id=sp.get('id'),offset=Number(sp.get('offset'));window.__playlistCalls.push({id,offset});
 if(window.__forbidden){code=403;result={error:'Spotify erlaubt die komplette Songliste nur für eigene Playlists oder solche, bei denen du mitarbeitest.'}}
 else if(offset===0){result={id,total:70,offset:0,nextOffset:50,tracks:Array.from({length:50},(_,i)=>({position:i,id:'5yEPktRqvIhko5QFF3aBH3',name:i===3?'Test Suchwort':'Song '+(i+1),artists:'Künstler '+(i+1),album:'Album A',duration:100000,playable:true}))}}
 else {result={id,total:70,offset:50,nextOffset:null,tracks:Array.from({length:20},(_,i)=>({position:i+50,id:'5yEPktRqvIhko5QFF3aBH3',name:'Titel '+(i+51),artists:'Zweiter Künstler',album:'Album B',duration:200000,playable:true}))}}
 }
 else if(u==='spotify/control'){window.spotifyCommands.push(JSON.parse(opts.body));result={message:'Spotify-Befehl gesendet'}}
 else if(u==='spotify/bookmarks'){result={connected:true,hasClientId:true,desktopControls:true,bookmarks:[]}}
 else if(u==='workspace')result={version:2,preset:'gaming',order:['system','monitor','games','network','news','clockwidget','musicwidget','friendswidget','sessionwidget'],hidden:['clockwidget','musicwidget','friendswidget','sessionwidget'],wide:['news'],cozyStart:true,smartGaming:false,glassStyle:'frosted',spotifyDock:'hidden'};
 else if(u==='studio')result={blur:18,radius:18,tint:55,outline:18};
 else if(u==='appearance')result={color:'oak',scene:'oak',cozy:'cozy',motion:true,speed:'slow',brightness:100,dim:0,vignette:20,density:'normal'};
 else if(u==='clock'||u==='overlay'||u==='library'||u==='ping')result={};
 else if(u==='maintenance')result={version:'4.2.3',repository:'',autoCheck:false};
 else if(u==='assistant/history')result={messages:[]};
 else if(u==='news')result={items:[]};
 return new Response(JSON.stringify(result),{status:code,headers:{'Content-Type':'application/json'}})
};})();'''
html=html.replace('</head>','<script>'+mock+'</script></head>')
with sync_playwright() as p:
 b=p.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox','--disable-web-security'])
 page=b.new_page(viewport={'width':1450,'height':980});errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.set_content(html,wait_until='load');page.wait_for_timeout(800)
 page.locator('[data-tab="spotify"]').click();page.wait_for_timeout(1100)
 assert page.locator('#spotify-playlists .spotify-playlist').count()==2,'playlists not rendered'
 page.locator('#spotify-playlists').get_by_text('▦ Songs ansehen').first.click()
 page.wait_for_timeout(500)
 assert page.locator('#spotify-library-browser').is_visible()
 assert page.locator('.spotify-track-row').count()==50,'first page missing'
 assert '50 von 70' in page.locator('#spotify-library-progress').inner_text()
 page.locator('#spotify-track-list .spotify-track-play').nth(3).click()
 page.wait_for_timeout(800)
 cmd=page.evaluate('window.spotifyCommands.at(-1)')
 assert cmd['action']=='playlist-track' and cmd['position']==3 and cmd['id']=='3cEYpjA9oz9GiPac4AsH4n',cmd
 page.locator('#spotify-library-search').fill('Suchwort');page.wait_for_timeout(160)
 assert page.locator('.spotify-track-row').count()==1,'search broken'
 page.locator('#spotify-library-search').fill('');page.locator('#spotify-library-next').click();page.wait_for_timeout(300)
 assert page.locator('.spotify-track-row').count()==70
 assert '70 von 70' in page.locator('#spotify-library-progress').inner_text()
 page.screenshot(path='/mnt/data/SquadBase_v423_Spotify_Playlist_Vorschau.png',full_page=False)
 page.locator('#spotify-library-close').click();assert page.locator('#spotify-library-browser').is_hidden()
 page.evaluate('window.__forbidden=true')
 page.locator('#spotify-bookmarks .spotify-browse').first.click();page.wait_for_timeout(350)
 assert 'eigene Playlists' in page.locator('#spotify-library-error').inner_text()
 page.locator('#spotify-library-close').click()
 page.locator('[data-tab="dashboard"]').click();page.wait_for_timeout(100)
 assert page.locator('#dashboard').is_visible()
 assert not errors,errors
 print('PASS: Playlist list, pagination 50+20, search, selected track position 3, 403 fallback, dashboard tab, no JS errors')
 b.close()
