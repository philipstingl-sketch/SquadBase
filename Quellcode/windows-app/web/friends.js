'use strict';
// Friends feature runs against a self-hosted Squad Base relay. Nothing is simulated.
(() => {
 const e = id => document.getElementById(id);
 const savedBase=localStorage.getItem('sb-relay-url')||'';
 const sameServer=sessionStorage.getItem('sb-relay-token-base')===savedBase;
 const f = {base:savedBase, token:sameServer?(sessionStorage.getItem('sb-relay-token')||''):'', user:sameServer?(sessionStorage.getItem('sb-relay-user')||''):'', peer:'', friends:[],incoming:[],outgoing:[],messageId:0,signalId:0,iceServers:[{urls:'stun:stun.l.google.com:19302'}], pendingIce:{}, chatPoll:false, signalPoll:false,call:null,friendPoll:false,filesAvailable:true,invitesRequired:false,cloudMode:false,lastFriendsAt:0,lastMessagesAt:0,lastSignalAt:0,lastUnreadAt:0,unread:{},alertsReady:false,invitesReady:false,lastIncoming:[],lastUnreadSnapshot:{},capabilities:{},connected:false,consecutiveErrors:0,nextRetryAt:0,searching:false,readSent:0};
 const alertText=(v,bad=false)=>typeof notify==='function'?notify(v,bad):window.alert(v);
 function connectionStatus(state,message){
  const badge=e('v453-connection');if(!badge)return;
  badge.dataset.state=state;e('v453-connection-text').textContent=message;
  f.connected=state==='online';
 }
 function networkGood(){f.consecutiveErrors=0;f.nextRetryAt=0;if(f.base)connectionStatus('online','Cloud-Server erreichbar · '+(f.user?'angemeldet als '+f.user:'Anmeldung möglich'));}
 function networkBad(){f.consecutiveErrors=Math.min(f.consecutiveErrors+1,6);
  const wait=Math.min(60000,3000*Math.pow(2,f.consecutiveErrors-1));f.nextRetryAt=Date.now()+wait;
  connectionStatus('offline','Verbindung unterbrochen · erneuter Versuch in '+Math.ceil(wait/1000)+' s');
 }
 function retryAllowed(){return Date.now()>=f.nextRetryAt;}
 
 function validBase(v){try{const u=new URL(v);if(u.username||u.password||u.search||u.hash)return false;return u.protocol==='https:'||(u.protocol==='http:'&&['127.0.0.1','localhost'].includes(u.hostname));}catch{return false}}
 async function request(path,{method='GET',body,upload=false}={}) {
  if(!f.base||(!f.token&& !['/api/register','/api/login','/api/health'].includes(path.split('?')[0])))throw new Error('Nicht angemeldet');
  const headers={};if(f.token&&!['/api/health','/api/login','/api/register'].includes(path.split('?')[0]))headers.Authorization='Bearer '+f.token;if(!upload&&body!==undefined)headers['Content-Type']='application/json';
  let response;try{response=await fetch(f.base+path,{method,headers,body:body===undefined?undefined:upload?body:JSON.stringify(body),cache:'no-store',signal:AbortSignal.timeout(upload?60000:12000)});}catch{networkBad();throw new Error('Freundeserver nicht erreichbar. Automatischer Wiederverbindungsversuch läuft.');}
  if(response.status>=500)networkBad();else networkGood();
  if(!response.ok){let msg='Serverfehler: '+response.status;try{const r=await response.json();msg=r.error||msg}catch{};if(response.status===401&&f.token&&!['/api/health','/api/login','/api/register'].includes(path.split('?')[0])){f.token='';sessionStorage.removeItem('sb-relay-token');showAuth()};throw new Error(msg)}
  if(path.startsWith('/api/file?'))return response.blob();return response.json();
 }
 const post=(p,body)=>request(p,{method:'POST',body});
 function showAuth(){e('friends-setup').hidden=false;e('auth-box').hidden=!f.base;e('friends-app').hidden=true;if(f.call)stopCall(false);if(f.base&&f.connected)connectionStatus('online','Cloud-Server erreichbar · Anmeldung möglich')}
 function showApp(){e('friends-setup').hidden=true;e('friends-app').hidden=false;e('my-friend-name').textContent=f.user;networkGood();request('/api/health').then(h=>applyCapabilities(h)).catch(()=>{});request('/api/ice').then(r=>{f.iceServers=r.iceServers||f.iceServers;f.hasTURN=!!r.hasTURN;f.turnMode=r.turnMode||''}).catch(()=>{});refreshFriends()}
 function applyCapabilities(h){
  f.capabilities=h?.capabilities||{};f.filesAvailable=h?.capabilities?.files!==false;f.invitesRequired=h?.capabilities?.invites===true;f.cloudMode=h?.capabilities?.persistent===true;
  e('cloud-server-capabilities').textContent=h?.capabilities?.persistent?'☁ Cloud-Verbindung: Konten und Chats dauerhaft gespeichert'+(f.filesAvailable?' · Dateianhänge aktiv':' · Dateianhänge benötigen optional R2'):'Eigener Freunde-Server';
  e('cloud-server-capabilities').hidden=false;e('cloud-invite-container').hidden=!f.invitesRequired;
  if(!f.filesAvailable){e('chat-file').disabled=true;e('chat-file').title='Dateianhänge sind auf diesem Cloud-Server nicht aktiviert (R2 benötigt).';}
  const privacy=e('v453-privacy');if(privacy)privacy.title=f.capabilities.accountDelete?'Konto und Daten im Cloud-Server verwalten':'Erfordert Cloud-Server v4.5.3';
 }
 async function persistBase(url){const response=await fetch('api/friends-cloud-config',{method:'POST',headers:{'Content-Type':'application/json','X-Squad-Base':'1'},body:JSON.stringify({url})});if(!response.ok)throw new Error('Cloud-Adresse konnte nicht auf dem PC gespeichert werden')}
 async function connect(){const url=e('relay-url').value.trim().replace(/\/+$/,'');if(!validBase(url)){alertText('Bitte eine HTTPS-Adresse eingeben (HTTP nur auf localhost).',true);return};if(url!==f.base){f.token='';f.user='';sessionStorage.removeItem('sb-relay-token');sessionStorage.removeItem('sb-relay-token-base');sessionStorage.removeItem('sb-relay-user')}f.base=url;localStorage.setItem('sb-relay-url',url);f.nextRetryAt=0;connectionStatus('connecting','Prüfe Cloud-Server …');try{const h=await request('/api/health');await persistBase(url);e('relay-info').textContent=h.name+' · erreichbar';applyCapabilities(h);e('auth-box').hidden=false;alertText('Server erreichbar. Jetzt anmelden oder registrieren.')}catch(err){e('auth-box').hidden=true;alertText(err.message,true)}}
 async function auth(register){const username=e('auth-name').value.toLowerCase().trim(),password=e('auth-password').value;if(!/^[a-z0-9_]{3,20}$/.test(username)||password.length<10){alertText('Name 3–20 Zeichen (a-z, 0–9, _), Passwort mindestens 10 Zeichen.',true);return};const payload={username,password};if(register&&f.invitesRequired)payload.inviteCode=e('auth-invite-code').value.trim();const r=await post(register?'/api/register':'/api/login',payload);f.token=r.token;f.user=r.username;sessionStorage.setItem('sb-relay-token',r.token);sessionStorage.setItem('sb-relay-token-base',f.base);sessionStorage.setItem('sb-relay-user',r.username);e('auth-password').value='';e('auth-invite-code').value='';f.peer='';f.messageId=0;f.signalId=0;f.unread={};f.alertsReady=false;f.invitesReady=false;f.lastUnreadSnapshot={};f.lastIncoming=[];f.lastUnreadAt=0;showApp();alertText(register?'Konto erstellt!':'Erfolgreich angemeldet.');}
 async function logout(){try{await post('/api/logout',{})}catch{};f.token='';f.user='';f.peer='';sessionStorage.removeItem('sb-relay-token');sessionStorage.removeItem('sb-relay-token-base');sessionStorage.removeItem('sb-relay-user');stopCall(false);f.unread={};f.alertsReady=false;f.invitesReady=false;showAuth()}
 function node(tag, cls, text){const n=document.createElement(tag);if(cls)n.className=cls;if(text!==undefined)n.textContent=text;return n}
 async function refreshFriends(){if(!f.token||f.friendPoll)return;f.friendPoll=true;try{const d=await request('/api/friends');f.friends=d.friends||[];f.incoming=d.incoming||[];f.outgoing=d.outgoing||[];f.online=d.online||{};
  if(f.invitesReady){const newInvites=f.incoming.filter(name=>!f.lastIncoming.includes(name));if(newInvites.length)window.SquadFriendAlerts?.alert('invites','Freundschaftsanfrage',newInvites.join(', '));}
  f.invitesReady=true;f.lastIncoming=[...f.incoming];f.blocked=d.blocked||[];renderFriends();if(f.peer&&!f.friends.includes(f.peer)){f.peer='';resetChat();if(f.call)stopCall(false)}}catch(err){if(!err.message.includes('Bitte anmelden'))console.warn('Friends:',err.message)}finally{f.friendPoll=false}}
 async function refreshUnread(){if(!f.token||!f.capabilities.unread||!retryAllowed())return;
  try{const data=await request('/api/unread');const next=data.unread||{};
   if(f.alertsReady){const newly=Object.keys(next).filter(name=>Number(next[name])>Number(f.lastUnreadSnapshot[name]||0));
    if(newly.length)window.SquadFriendAlerts?.alert('messages','Neue Nachricht',newly.join(', '));}
   f.alertsReady=true;f.lastUnreadSnapshot={...next};f.unread=next;renderFriends();}
  catch(err){if(!/Nicht angemeldet|nicht erreichbar/i.test(err.message))console.warn('Ungelesene Nachrichten:',err.message)}
 }
 function clearSearch(){f.searching=false;e('v453-chat-back').hidden=true;e('v453-search-state').hidden=true;e('v453-chat-search').value='';}
 async function runChatSearch(){if(!f.peer||!f.token)return;
  const query=e('v453-chat-search').value.trim();if(query.length<2){alertText('Bitte mindestens zwei Zeichen eingeben.',true);return}
  const peer=f.peer;e('v453-chat-find').disabled=true;e('v453-search-state').hidden=false;e('v453-search-state').textContent='Suche nach Nachrichten …';
  try{const r=await request('/api/search?peer='+encodeURIComponent(peer)+'&q='+encodeURIComponent(query));if(f.peer!==peer)return;
   f.searching=true;e('v453-chat-back').hidden=false;e('chat-messages').replaceChildren();
   for(const m of r.messages||[])addMessage(m);
   e('v453-search-state').textContent=(r.messages||[]).length+' Treffer'+(r.limited?' · maximal 100 Treffer':'');
   if(!r.messages?.length)e('chat-messages').append(node('p','hint','Keine Nachrichten gefunden.'));
  }catch(err){e('v453-search-state').textContent=err.message.includes('404')||err.message.includes('Unbekannter')?'Suche benötigt das v4.5.3-Cloud-Server-Update.':err.message;}
  finally{e('v453-chat-find').disabled=false}
 }
 async function exportAccount(){try{if(!f.capabilities.accountExport)throw Error('Bitte zuerst den Cloud-Server auf v4.5.3 aktualisieren.');
  const data=await request('/api/account/export');const blob=new Blob([JSON.stringify(data,null,2)],{type:'application/json'});
  const uri=URL.createObjectURL(blob),a=document.createElement('a');a.href=uri;a.download='squadbase-meine-daten-'+f.user+'.json';a.click();setTimeout(()=>URL.revokeObjectURL(uri),10000);
  alertText(data.limited?'Export enthält nur die letzten 2000 Nachrichten.':'Deine Daten wurden als JSON exportiert.');
 }catch(err){alertText(err.message,true)}}
 async function deleteOwnAccount(){if(!f.capabilities.accountDelete){alertText('Bitte zuerst den Cloud-Server auf v4.5.3 aktualisieren.',true);return}
  const confirmName=e('v453-delete-name').value.trim(),password=e('v453-delete-password').value;
  if(confirmName!==f.user||!password){alertText('Gib deinen genauen Benutzernamen und dein Passwort in die Datenschutzfelder ein.',true);return}
  if(!confirm('Dein Konto und deine Nachrichten werden unwiderruflich aus D1 gelöscht. Vorher Daten exportieren?'))return;
  try{await post('/api/account/delete',{confirm:confirmName,password});e('v453-delete-password').value='';e('v453-delete-name').value='';await logout();alertText('Konto gelöscht. Cloud-Dateien in R2 können separat zu löschen sein.');}
  catch(err){alertText(err.message,true)}
 }
 function renderFriends(){const invite=e('friend-invites');invite.replaceChildren();if(f.incoming.length){invite.append(node('h3','',`Anfragen (${f.incoming.length})`));f.incoming.forEach(name=>{const item=node('div','invite');item.append(node('span','',name+' möchte dein Freund sein'));const actions=node('div','invite-actions');const accept=node('button','outline','✓ Annehmen');accept.onclick=()=>friendAction('accept',name);const decline=node('button','outline','✕ Ablehnen');decline.onclick=()=>friendAction('decline',name);actions.append(accept,decline);item.append(actions);invite.append(item)})}if(f.outgoing.length){invite.append(node('p','hint','Ausstehend: '+f.outgoing.join(', ')))}
  if(f.blocked?.length){const area=node('div','invite');area.append(node('div','',`Blockiert: ${f.blocked.join(', ')}`));f.blocked.forEach(name=>{const b=node('button','outline','Blockierung für '+name+' aufheben');b.onclick=()=>friendAction('unblock',name);area.append(b)});invite.append(area)}const list=e('friend-list');list.replaceChildren();if(!f.friends.length)list.append(node('p','hint','Noch keine Freunde. Sende eine Anfrage über den Benutzernamen.'));
  const total=f.friends.reduce((n,name)=>n+(Number(f.unread[name])||0),0);e('v453-unread-total').textContent=String(total);e('v453-unread-total').hidden=!total;
  f.friends.forEach(name=>{const b=node('button','friend-item'+(f.peer===name?' selected':''));b.append(node('span','friend-icon',name.slice(0,1).toUpperCase()),node('span','',name+(f.online?.[name]?' · 🟢 online':' · offline')));if(f.unread[name]>0){const counter=node('span','v453-unread',String(f.unread[name]));counter.setAttribute('aria-label',f.unread[name]+' ungelesene Nachrichten');b.append(counter)}b.onclick=()=>selectPeer(name);list.append(b)})
 }
 async function friendAction(action,username){try{await post('/api/'+action,{username});alertText(action==='request'?'Freundschaftsanfrage verschickt.':'Änderung gespeichert.');await refreshFriends()}catch(err){alertText(err.message,true)}}
 function resetChat(){e('chat-title').textContent='Wähle einen Freund';e('chat-subtitle').textContent='Danach kannst du schreiben, Dateien senden und anrufen.';e('chat-messages').replaceChildren(node('div','chat-empty','Wähle links einen Freund aus.'));['chat-text','chat-send','chat-file','call-voice','call-video','chat-block','chat-unfriend','v453-chat-search','v453-chat-find'].forEach(id=>e(id).disabled=true);clearSearch()}
 async function selectPeer(peer){if(f.call){alertText('Lege zuerst deinen aktuellen Anruf auf.',true);return};f.peer=peer;f.messageId=0;f.readSent=0;clearSearch();e('chat-title').textContent='✦ '+peer;e('chat-subtitle').textContent='Privater Chat · bestätigter Freund';['chat-text','chat-send','call-voice','call-video','chat-block','chat-unfriend','v453-chat-search','v453-chat-find'].forEach(id=>e(id).disabled=false);e('chat-file').disabled=!f.filesAvailable;e('chat-messages').replaceChildren();renderFriends();await loadMessages();}
 function addMessage(m){const box=node('div','chat-bubble'+(m.from===f.user?' mine':''));const when=new Date(m.at).toLocaleTimeString('de-DE',{hour:'2-digit',minute:'2-digit'});box.append(node('div','chat-author',`${m.from} · ${when}`));if(m.text)box.append(node('div','',m.text));if(m.fileId){const b=node('button','chat-download',`📎 ${m.fileName||'Datei'} · ${(m.fileSize/1024).toFixed(1)} KB  ⬇`);b.onclick=()=>download(m.fileId,m.fileName);box.append(b)}e('chat-messages').append(box)}
 async function loadMessages(){if(!f.peer||f.chatPoll||!f.token||f.searching||!retryAllowed())return;f.chatPoll=true;const current=f.peer;try{const r=await request('/api/messages?peer='+encodeURIComponent(current)+'&after='+f.messageId);if(f.peer!==current)return;const scroller=e('chat-messages');const atBottom=scroller.scrollTop+scroller.clientHeight>=scroller.scrollHeight-55;for(const m of r.messages||[]){const isNew=f.messageId>0&&Number(m.id)>f.messageId&&m.from!==f.user;if(isNew)window.SquadFriendAlerts?.alert('messages','Neue Nachricht',current);f.messageId=Math.max(f.messageId,m.id);addMessage(m)}if(atBottom)scroller.scrollTop=scroller.scrollHeight;
  if(f.capabilities.unread&&f.messageId>f.readSent&&!document.hidden&&e('friends').classList.contains('active')){f.readSent=f.messageId;post('/api/read',{peer:current,id:f.messageId}).then(()=>{if(f.unread[current]){delete f.unread[current];renderFriends()}}).catch(()=>{})}}catch(err){console.warn('Chat:',err.message)}finally{f.chatPoll=false}}
 async function sendMessage(){const text=e('chat-text').value.trim();if(!text||!f.peer)return;e('chat-send').disabled=true;try{await post('/api/messages',{to:f.peer,text});e('chat-text').value='';await loadMessages();window.SquadFriendAlerts?.tone('sent')}catch(err){alertText(err.message,true)}finally{e('chat-send').disabled=false}}
 async function uploadFile(file){if(!file||!f.peer)return;if(!f.filesAvailable){alertText('Dateianhänge benötigen R2-Speicher beim Cloud-Server.',true);return}if(file.size>12*1024*1024){alertText('Maximale Dateigröße: 12 MB.',true);return};const form=new FormData();form.append('file',file,file.name);try{alertText('Datei wird hochgeladen …');await request('/api/upload?to='+encodeURIComponent(f.peer),{method:'POST',body:form,upload:true});await loadMessages();alertText('Datei gesendet.')}catch(err){alertText(err.message,true)}finally{e('chat-file').value=''}}
 async function download(fileId,name){try{const blob=await request('/api/file?id='+encodeURIComponent(fileId));const url=URL.createObjectURL(blob);const a=document.createElement('a');a.href=url;a.download=name||'datei';a.click();setTimeout(()=>URL.revokeObjectURL(url),10000)}catch(err){alertText(err.message,true)}}
 function peerVolume(peer){if(window.SquadFriendAlerts?.peerVolume)return window.SquadFriendAlerts.peerVolume(peer);try{const values=JSON.parse(localStorage.getItem('squad-v454-friend-volumes')||'{}');const level=values[peer];return Number.isInteger(level)&&level>=0&&level<=100?level:100}catch{return 100}}
 function savePeerVolume(peer,volume){if(window.SquadFriendAlerts?.savePeerVolume)return window.SquadFriendAlerts.savePeerVolume(peer,volume);if(!peer||!/^[a-z0-9_]{3,20}$/.test(peer))return;try{const values=JSON.parse(localStorage.getItem('squad-v454-friend-volumes')||'{}');values[peer]=Math.max(0,Math.min(100,volume));localStorage.setItem('squad-v454-friend-volumes',JSON.stringify(values))}catch{}}
 // Die Anruf-Engine (inkl. Gruppenanrufe) liegt in calls.js. friends.js liefert ihr nur Zustand und Signale.
 async function signal(to,kind,data){return post('/api/signals',{to,kind,data:data===undefined?null:data})}
 function stopCall(tell=true){window.SquadCalls?.stop(tell)}
 window.SquadFriendsCore={f,e,request,signal,alertText,node,peerVolume,savePeerVolume};
 function guarded(promise,ms){let timer;return Promise.race([promise,new Promise((_,reject)=>{timer=setTimeout(()=>reject(new Error('Zeitüberschreitung')),ms)})]).finally(()=>clearTimeout(timer))}
 async function processSignals(){if(!f.token||f.signalPoll)return;f.signalPoll=true;try{const d=await request('/api/signals?after='+f.signalId);
  for(const sig of d.signals||[]){f.signalId=Math.max(f.signalId,sig.id);try{await guarded(Promise.resolve(window.SquadCalls?.handleSignal(sig)),20000)}catch(err){console.warn('Call signaling:',err)}}
  // Bewusst KEIN Vorspringen auf d.latest: Ein Signal, das zwischen den beiden Server-Abfragen eintrifft, ginge sonst verloren.
 }catch(err){console.warn('Signal relay:',err.message)}finally{f.signalPoll=false}}
 e('v453-chat-find').onclick=()=>runChatSearch();e('v453-chat-back').onclick=()=>{clearSearch();e('chat-messages').replaceChildren();f.messageId=0;f.readSent=0;loadMessages()};
 e('v453-chat-search').addEventListener('keydown',ev=>{if(ev.key==='Enter'){ev.preventDefault();runChatSearch()}});
 e('v453-export').onclick=()=>exportAccount();e('v453-delete-account').onclick=()=>deleteOwnAccount();
 e('v453-reconnect').onclick=async()=>{f.nextRetryAt=0;try{const h=await request('/api/health');applyCapabilities(h);alertText('Cloud-Server erreichbar.')}catch(err){alertText(err.message,true)}};
 e('relay-connect').onclick=()=>connect();e('friends-login').onclick=()=>auth(false).catch(er=>alertText(er.message,true));e('friends-register').onclick=()=>auth(true).catch(er=>alertText(er.message,true));e('friends-logout').onclick=()=>logout();e('friend-add').onclick=()=>{const n=e('friend-request-name').value.toLowerCase().trim();if(n)friendAction('request',n)};
 e('chat-compose').addEventListener('submit',ev=>{ev.preventDefault();sendMessage()});e('chat-file').onchange=ev=>uploadFile(ev.target.files[0]);
 e('chat-unfriend').onclick=async()=>{if(f.peer&&confirm('Freund '+f.peer+' wirklich entfernen?')){await friendAction('unfriend',f.peer);f.peer='';resetChat()}};
 e('chat-block').onclick=async()=>{if(f.peer&&confirm('Nutzer '+f.peer+' blockieren? Nachrichten und Anrufe werden unterbunden.')){await friendAction('block',f.peer);f.peer='';resetChat()}};
 async function startFriends(){
  try{const response=await fetch('api/friends-cloud-config',{cache:'no-store'});if(response.ok){const stored=await response.json();if(stored.url&&stored.url!==f.base){f.base=stored.url;f.token='';f.user='';sessionStorage.removeItem('sb-relay-token');sessionStorage.removeItem('sb-relay-token-base');sessionStorage.removeItem('sb-relay-user')}else if(stored.url){f.base=stored.url}}}catch{}
  e('relay-url').value=f.base;
  resetChat();if(f.base&&f.token){request('/api/me').then(r=>{f.user=r.username;showApp()}).catch(()=>{f.token='';sessionStorage.removeItem('sb-relay-token');showAuth()})}else showAuth();
 }
 startFriends();
 setInterval(async()=>{if(!f.base)return;
  if(!retryAllowed())return;
  try{const h=await request('/api/health');applyCapabilities(h);if(f.token&&f.capabilities.unread)refreshUnread();}
  catch(err){console.warn('Cloud heartbeat:',err.message)}
 },20000);
 window.addEventListener('online',()=>{f.nextRetryAt=0;if(f.token){refreshFriends();loadMessages();processSignals()}});
 document.addEventListener('visibilitychange',()=>{if(!document.hidden&&f.token){f.nextRetryAt=0;refreshFriends();loadMessages();processSignals();refreshUnread()}});
 // Cloud mode uses quota-aware polling; the previous local relay keeps its fast intervals.
 // Incoming cloud calls may ring after a few seconds while idle; during a call signaling is fast.
 setInterval(()=>{if(!f.token||!retryAllowed())return;if(!f.cloudMode){refreshFriends();loadMessages();return}
  const now=Date.now();if(now-f.lastFriendsAt>(document.hidden?30000:17000)){f.lastFriendsAt=now;refreshFriends()}
  if(f.peer&&now-f.lastMessagesAt>(document.hidden?30000:7000)){f.lastMessagesAt=now;loadMessages()}
  if(now-f.lastUnreadAt>(document.hidden?30000:17000)){f.lastUnreadAt=now;refreshUnread()}
 },2600);
 setInterval(()=>{if(!f.token||!retryAllowed())return;if(!f.cloudMode){processSignals();return}
  const now=Date.now(),period=f.call?1100:document.hidden?12000:5500;
  if(now-f.lastSignalAt>=period){f.lastSignalAt=now;processSignals()}
 },1100);
})();
