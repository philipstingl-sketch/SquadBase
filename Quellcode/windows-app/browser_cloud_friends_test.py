from playwright.sync_api import sync_playwright
from pathlib import Path
import re
root=Path(__file__).parent
html=(root/'web/index.html').read_text()
for match in list(re.finditer(r'<link rel="stylesheet" href="([^"]+)">',html)):
 html=html.replace(match.group(0),'<style>'+(root/'web'/match.group(1)).read_text()+'</style>')
for match in list(re.finditer(r'<script src="([^"]+)"></script>',html)):
 html=html.replace(match.group(0),'<script>'+(root/'web'/match.group(1)).read_text().replace('</script>','<\\/script>')+'</script>')
old=(root/'browser_v44_test.py').read_text()
mock=old.split("mock=r'''",1)[1].split("'''",1)[0]
cloudmock=r'''(() => {
const baseFetch=window.fetch;window.__friendRequests=[];
window.fetch=async (url,opts={})=>{
 if(!String(url).startsWith('https://squadbase-example.workers.dev'))return baseFetch(url,opts);
 const path=new URL(url).pathname, method=opts.method||'GET';
 if(method==='OPTIONS')return new Response(null,{status:204});
 let d={ok:true};
 if(path==='/api/health')d={status:'ok',name:'SQUAD BASE Friends Cloud',capabilities:{persistent:true,files:false,invites:true}};
 if(['/api/register','/api/login'].includes(path)){const obj=JSON.parse(opts.body||'{}');window.__friendRequests.push({path,...obj});d={token:'a'.repeat(64),username:obj.username};}
 if(path==='/api/me')d={username:'alice'};
 if(path==='/api/ice')d={iceServers:[{urls:'stun:stun.l.google.com:19302'}]};
 if(path==='/api/friends')d={friends:['bob'],incoming:[],outgoing:[],blocked:[],online:{bob:true}};
 if(path==='/api/messages')d={messages:[]};
 if(path==='/api/signals')d={signals:[],latest:0};
 return new Response(JSON.stringify(d),{status:200,headers:{'Content-Type':'application/json'}});
}
})();'''
html=html.replace('</head>','<script>'+mock+'</script><script>'+cloudmock+'</script></head>')
with sync_playwright() as p:
 browser=p.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox','--disable-web-security'])
 for width in (1440,900,640):
  pg=browser.new_page(viewport={'width':width,'height':900});errors=[];pg.on('pageerror',lambda e:errors.append(str(e)))
  pg.set_content(html,wait_until='domcontentloaded');pg.wait_for_timeout(500)
  pg.locator('[data-tab="friends"]').click()
  pg.locator('#relay-url').fill('https://squadbase-example.workers.dev')
  pg.locator('#relay-connect').click();pg.wait_for_timeout(120)
  assert pg.locator('#cloud-invite-container').is_visible(),'missing invitation input'
  assert pg.locator('#cloud-server-capabilities').is_visible(),'missing cloud status'
  pg.locator('#auth-name').fill('alice');pg.locator('#auth-password').fill('test_password_12345')
  pg.locator('#auth-invite-code').fill('SQUAD-PRIVATE-INVITE-CODE')
  pg.locator('#friends-register').click();pg.wait_for_timeout(150)
  assert pg.locator('#friends-app').is_visible(),'app not shown'
  assert pg.evaluate('window.__friendRequests[0].inviteCode')=='SQUAD-PRIVATE-INVITE-CODE'
  assert pg.evaluate("sessionStorage.getItem('sb-relay-token-base')")=='https://squadbase-example.workers.dev'
  pg.locator('.friend-item').first.click();pg.wait_for_timeout(150)
  assert pg.locator('#chat-file').is_disabled(),'file control should disable without R2'
  pg.locator('#friends-logout').click();pg.wait_for_timeout(150)
  assert not pg.evaluate("sessionStorage.getItem('sb-relay-token')"),'logout token not cleared'
  assert not errors,errors
  assert pg.evaluate('document.documentElement.scrollWidth-window.innerWidth')<15,'horizontal overflow'
  print('PASS',width,'cloud connect, invite, register, server token scope, optional files, logout, zero errors',flush=True)
  pg.close()
 browser.close()
