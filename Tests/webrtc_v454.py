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
 window.__requests=[]; window.__signals=[];
 class FakeTrack {constructor(kind){this.kind=kind;this.enabled=true;this.readyState='live'} stop(){this.readyState='ended'}}
 class FakeMedia {constructor(tracks=[]){this.tracks=tracks}getTracks(){return this.tracks}getAudioTracks(){return this.tracks.filter(x=>x.kind==='audio')}getVideoTracks(){return this.tracks.filter(x=>x.kind==='video')}addTrack(t){this.tracks.push(t)}removeTrack(t){this.tracks=this.tracks.filter(x=>x!==t)}}
 class FakePC {constructor(){this.senders=[];this.transceivers=[];this.connectionState='new'}
  addTrack(track,stream){const sender={track,replaceTrack:async(t)=>{sender.track=t}};this.senders.push(sender);return sender}
  addTransceiver(kind){const sender={track:null,replaceTrack:async(t)=>{sender.track=t}};const tx={sender,receiver:{track:{kind}}};this.transceivers.push(tx);this.senders.push(sender);return tx}
  getTransceivers(){return this.transceivers}getSenders(){return this.senders}
  async createOffer(){return {type:'offer',sdp:'fake'}}async createAnswer(){return {type:'answer',sdp:'fake'}}
  async setLocalDescription(x){this.localDescription=x}async setRemoteDescription(x){this.remoteDescription=x}
  async addIceCandidate(){}close(){} restartIce(){}
 }
 window.MediaStream=FakeMedia;window.RTCPeerConnection=FakePC;
 window.RTCSessionDescription=class{constructor(x){Object.assign(this,x)}};
 Object.defineProperty(HTMLMediaElement.prototype,'srcObject',{configurable:true,get(){return this._fakeStream||null},set(v){this._fakeStream=v}});
 HTMLMediaElement.prototype.play=async function(){return};
 Object.defineProperty(navigator,'mediaDevices',{configurable:true,value:{
  getUserMedia:async(cfg)=>new FakeMedia([...cfg.audio?[new FakeTrack('audio')]:[],...cfg.video?[new FakeTrack('video')]:[]]),
  enumerateDevices:async()=>[{kind:'audioinput',deviceId:'mic1',label:'Testmikrofon'},{kind:'audiooutput',deviceId:'spk1',label:'Testspeaker'}]
 }});
 window.__clicker={intervalMs:200,mouseButton:'left',hotkey:'F8',active:false,clicks:0,supported:true};
 window.__prefs={messages:true,calls:true,sent:false,invites:true,volume:55,volumes:{}};
 window.fetch=async (url,opts={})=>{
  const value=String(url),path=value.startsWith('https://')?new URL(value).pathname+new URL(value).search:value.replace(/^api\//,'/api/');window.__requests.push({path,method:opts.method||'GET'});
  let data={},status=200;
  if(path==='/api/autoclicker'){
    if(opts.method==='POST'){const action=JSON.parse(opts.body||'{}');if(action.action==='configure')Object.assign(window.__clicker,{intervalMs:action.intervalMs,mouseButton:action.mouseButton,hotkey:action.hotkey,active:false});if(action.action==='start')window.__clicker.active=true;if(action.action==='stop')window.__clicker.active=false;}data=window.__clicker;
  }else if(path==='/api/friend-prefs'){
    if(opts.method==='POST')window.__prefs=JSON.parse(opts.body);data=window.__prefs;
  }else if(path==='/api/friends-cloud-config')data={url:u};
  else if(path==='/api/health')data={status:'ok',name:'SQUAD BASE Friends Cloud',capabilities:{files:false,invites:true,persistent:true,unread:true,search:true,accountExport:true,accountDelete:true}};
  else if(path==='/api/me')data={username:'alice'};
  else if(path==='/api/ice')data={iceServers:[{urls:'stun:stun.l.google.com:19302'}]};
  else if(path==='/api/friends')data={friends:['bob'],incoming:[],outgoing:[],online:{bob:true},blocked:[]};
  else if(path==='/api/unread')data={unread:{bob:2}};
  else if(path.startsWith('/api/messages?'))data={messages:[{id:3,from:'bob',to:'alice',text:'Hallo Testnachricht!',at:1760110000000}]};
  else if(path==='/api/read')data={ok:true,lastReadId:3};
  else if(path.startsWith('/api/search?'))data={messages:[{id:3,from:'bob',to:'alice',text:'Hallo Testnachricht!',at:1760110000000}]};
  else if(path.startsWith('/api/signals?')){const after=Number(new URLSearchParams(path.split('?')[1]).get('after')||0);data={signals:window.__signals.filter(x=>x.id>after),latest:window.__signals.length?window.__signals.at(-1).id:0};}
  else if(path==='/api/signals')data={ok:true};
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
 page=b.new_page(viewport={'width':1200,'height':900})
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.set_content(html,wait_until='load')
 page.locator('[data-tab="friends"]').first.click()
 page.wait_for_selector('#friend-list .friend-item',timeout=5000)
 page.locator('#friend-list .friend-item').first.click()
 page.locator('#call-video').click()
 page.wait_for_function("window.__requests.some(x=>x.path==='/api/signals'&&x.method==='POST')")
 assert page.locator('#call-overlay').is_visible()
 assert 'Kamera ausschalten' in page.locator('#v454-toggle-camera').inner_text()
 page.locator('#v454-toggle-camera').click()
 page.wait_for_function("document.getElementById('v454-toggle-camera').textContent.includes('einschalten')")
 page.locator('#v454-toggle-camera').click()
 page.wait_for_function("document.getElementById('v454-toggle-camera').textContent.includes('ausschalten')")
 page.locator('#v453-call-volume').fill('35')
 page.wait_for_function("window.__prefs.volumes.bob===35")
 page.locator('#call-hangup').click()
 page.wait_for_function("document.getElementById('call-overlay').hidden")
 # New incoming video call signaled by a trusted, confirmed friend.
 page.evaluate("window.__signals=[{id:1,from:'bob',kind:'offer',data:{type:'offer',sdp:'fake',squadVideo:true}}]")
 page.wait_for_selector('#v454-accept-video:visible',timeout=9000)
 assert 'Videoanruf' in page.locator('#call-status').inner_text()
 page.locator('#v454-accept-video').click()
 page.wait_for_function("document.getElementById('v454-accept-video').hidden",timeout=3000)
 assert 'Kamera ausschalten' in page.locator('#v454-toggle-camera').inner_text()
 page.locator('#call-hangup').click()
 assert not errors, errors
 print('PASS WebRTC 2-way camera controls, video offer, incoming video accept, per-friend volume; no JS exceptions',flush=True)
 page.close();b.close()
