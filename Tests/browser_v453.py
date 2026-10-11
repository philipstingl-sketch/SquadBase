"""Run the embedded HTML in policy-restricted Chromium with deterministic fake APIs."""
from pathlib import Path
from playwright.sync_api import sync_playwright
import re
root=Path(__file__).resolve().parents[1]/'windows-app'/'web'
html=(root/'index.html').read_text()
# Style and script assets are embedded into test DOM due to Chromium policy blocking ALL navigation.
html=re.sub(r'<link rel="stylesheet" href="([^"]+)"\s*>',lambda m:'<style>'+ (root/m.group(1)).read_text()+'</style>',html)
mock=r'''<script>
(()=>{
 const m=new Map(), sess=new Map();const u='https://squadbase-friends-cloud.squadbase-gaming.workers.dev';
 Object.defineProperty(window,'localStorage',{configurable:true,value:{getItem:k=>m.get(k)||null,setItem:(k,v)=>m.set(k,''+v),removeItem:k=>m.delete(k),clear:()=>m.clear()}});
 Object.defineProperty(window,'sessionStorage',{configurable:true,value:{getItem:k=>sess.get(k)||null,setItem:(k,v)=>sess.set(k,''+v),removeItem:k=>sess.delete(k),clear:()=>sess.clear()}});
 localStorage.setItem('sb-relay-url',u);sessionStorage.setItem('sb-relay-token-base',u);sessionStorage.setItem('sb-relay-token','a'.repeat(64));sessionStorage.setItem('sb-relay-user','alice');
 window.__requests=[];
 window.fetch=async (url,opts={})=>{
  const value=String(url),path=value.startsWith('https://')?new URL(value).pathname+new URL(value).search:value.replace(/^api\//,'/api/');window.__requests.push({path,method:opts.method||'GET'});
  let data={},status=200;
  if(path==='/api/friends-cloud-config')data={url:u};
  else if(path==='/api/health')data={status:'ok',name:'SQUAD BASE Friends Cloud',capabilities:{files:false,invites:true,persistent:true,unread:true,search:true,accountExport:true,accountDelete:true}};
  else if(path==='/api/me')data={username:'alice'};
  else if(path==='/api/ice')data={iceServers:[{urls:'stun:stun.l.google.com:19302'}]};
  else if(path==='/api/friends')data={friends:['bob'],incoming:[],outgoing:[],online:{bob:true},blocked:[]};
  else if(path==='/api/unread')data={unread:{bob:2}};
  else if(path.startsWith('/api/messages?'))data={messages:[{id:3,from:'bob',to:'alice',text:'Hallo Testnachricht!',at:1760110000000}]};
  else if(path==='/api/read')data={ok:true,lastReadId:3};
  else if(path.startsWith('/api/search?'))data={messages:[{id:3,from:'bob',to:'alice',text:'Hallo Testnachricht!',at:1760110000000}]};
  else if(path.startsWith('/api/signals?'))data={signals:[],latest:0};
  else if(path==='/api/account/export')data={messages:[],profile:{username:'alice'},friends:[]};
  else {data={error:'mock missing'};status=404;}
  return new Response(JSON.stringify(data),{status,headers:{'Content-Type':'application/json'}});
 };
})();
</script>'''
html=html.replace('</head>',mock+'</head>')
html=re.sub(r'<script src="([^"]+)"\s*></script>',lambda m:'<script>'+ (root/m.group(1)).read_text()+'</script>',html)
with sync_playwright() as p:
 b=p.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
 for width in (640,900,1440):
  page=b.new_page(viewport={'width':width,'height':850})
  errs=[];page.on('pageerror',lambda e:errs.append(str(e)))
  page.set_content(html,wait_until='load');page.wait_for_timeout(400)
  page.locator('[data-tab="friends"]').first.click()
  print('DIAG errors=', errs[:5], 'state=', page.locator('#v453-connection-text').inner_text(), 'setup=',page.locator('#friends-setup').get_attribute('hidden'), 'requests=',page.evaluate('window.__requests?.slice(0,8)'))
  page.wait_for_selector('#friends-app:not([hidden])',timeout=5000)
  assert 'angemeldet' in page.locator('#v453-connection-text').inner_text()
  page.wait_for_selector('#friend-list .friend-item',timeout=5000)
  page.wait_for_timeout(100)
  # Unread is fetched via the new endpoint and then shown in list; can be cleared after chat is read.
  unread=page.locator('#v453-unread-total').inner_text();assert unread in ('0','2'),unread
  page.locator('#friend-list .friend-item').first.click()
  page.wait_for_selector('#chat-messages .chat-bubble',timeout=5000)
  assert 'Hallo Testnachricht' in page.locator('#chat-messages').inner_text()
  page.locator('#v453-chat-search').fill('Hallo')
  page.locator('#v453-chat-find').click()
  page.wait_for_selector('#v453-chat-back:visible')
  assert '1 Treffer' in page.locator('#v453-search-state').inner_text()
  page.locator('#v453-chat-back').click();page.wait_for_timeout(80)
  page.locator('.v453-privacy > summary').click()
  assert page.locator('#v453-export').is_visible()
  assert page.locator('#v453-mute').count()==1
  assert not errs,(width,errs)
  horizontal=page.evaluate('document.documentElement.scrollWidth-document.documentElement.clientWidth')
  assert horizontal<=5,f'horizontal overflow at {width}: {horizontal}px'
  if width==900:page.screenshot(path='/mnt/data/SquadBase_v453_Freunde_Vorschau.png')
  print('PASS embedded browser',width,'px, friend list, chat, search, privacy, no JS error, overflow',horizontal,flush=True)
  page.close()
 b.close()
