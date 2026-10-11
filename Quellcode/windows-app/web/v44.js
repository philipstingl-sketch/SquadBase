'use strict';
(() => {
 const $=id=>document.getElementById(id);
 const root=$('v44-board');if(!root)return;
 let clips=[],currentAudio=null;
 const status=(id,msg,bad=false)=>{const n=$(id);if(n){n.textContent=msg;n.classList.toggle('v44-error',bad)}};
 async function api(path,body,opts={}){
  const init={method:body===undefined?'GET':'POST',headers:{'X-Squad-Base':'1'},...opts};
  if(body!==undefined&&!(body instanceof Uint8Array)){init.headers['Content-Type']='application/json';init.body=JSON.stringify(body)}
  else if(body instanceof Uint8Array){init.headers['Content-Type']='application/octet-stream';init.body=body}
  const res=await fetch('api/'+path,init);const obj=await res.json();if(!res.ok)throw Error(obj.error||'HTTP '+res.status);return obj;
 }
 function renderBoard(){const list=$('v44-sound-list');list.replaceChildren();if(!clips.length){const p=document.createElement('p');p.className='hint';p.textContent='Noch keine Sounds importiert. Nutze „MP3, WAV oder OGG auswählen“.';list.appendChild(p);return}
  clips.forEach((clip,i)=>{
   const row=document.createElement('div');row.className='v44-sound';const number=document.createElement('span');number.className='v44-number';number.textContent=String(i+1);
   const name=document.createElement('strong');name.textContent=clip.name;name.title=clip.name;
   const play=document.createElement('button');play.className='primary';play.textContent='▶ Abspielen';play.addEventListener('click',()=>playClip(clip.id));
   const del=document.createElement('button');del.className='outline';del.textContent='✕';del.title='Sound löschen';del.setAttribute('aria-label','Sound '+clip.name+' löschen');del.addEventListener('click',async()=>{if(!confirm('Sound „'+clip.name+'“ wirklich löschen?'))return;try{await api('soundboard/delete',{id:clip.id});await loadBoard();status('v44-sound-status','Sound entfernt.')}catch(e){status('v44-sound-status',e.message,true)}});
   row.append(number,name,play,del);list.appendChild(row);
  });
 }
 function playClip(id){if(!clips.some(s=>s.id===id))return;try{if(currentAudio){currentAudio.pause();currentAudio.currentTime=0};currentAudio=new Audio('api/soundboard/file/'+encodeURIComponent(id));currentAudio.volume=Math.min(1,Math.max(0,Number($('v44-volume').value)/100));currentAudio.play().then(()=>status('v44-sound-status','Sound wird am PC abgespielt.')).catch(e=>status('v44-sound-status','Audio konnte nicht gestartet werden: '+e.message,true))}catch(e){status('v44-sound-status',e.message,true)}}
 async function loadBoard(){try{const r=await api('soundboard');clips=r.clips||[];renderBoard()}catch(e){status('v44-sound-status',e.message,true)}}
 $('v44-file').addEventListener('change',async e=>{const file=e.target.files?.[0];if(!file)return;if(file.size>8*1024*1024){status('v44-sound-status','Maximal 8 MB pro Datei.',true);e.target.value='';return}
  const name=file.name.replace(/\.[^.]+$/,'');status('v44-sound-status','Importiere '+name+' …');try{const buf=new Uint8Array(await file.arrayBuffer());const path='soundboard/upload?filename='+encodeURIComponent(file.name)+'&name='+encodeURIComponent(name);await api(path,buf);await loadBoard();status('v44-sound-status','✓ '+name+' importiert.')}catch(err){status('v44-sound-status',err.message,true)}finally{e.target.value=''}
 });
 $('v44-volume').addEventListener('input',()=>{$('v44-volume-output').textContent=$('v44-volume').value+'%';if(currentAudio)currentAudio.volume=Number($('v44-volume').value)/100;try{localStorage.setItem('squad-v44-sound-level',$('v44-volume').value)}catch{}});
 try{const stored=localStorage.getItem('squad-v44-sound-level');if(stored!==null&&Number(stored)>=0&&Number(stored)<=100){$('v44-volume').value=stored;$('v44-volume-output').textContent=stored+'%'}}catch{}
 document.addEventListener('keydown',e=>{if(!e.ctrlKey||!e.altKey||e.shiftKey||e.repeat)return;const n=Number(e.key);if(!Number.isInteger(n)||n<1||n>9)return;if(['INPUT','TEXTAREA','SELECT'].includes(document.activeElement?.tagName))return;if(clips[n-1]){e.preventDefault();playClip(clips[n-1].id)}});
 let remoteActive=false;
 async function refreshRemote(){try{const r=await api('remote');remoteActive=r.enabled;$('v44-remote-start').hidden=r.enabled;$('v44-remote-stop').hidden=!r.enabled;$('v44-remote-renew').hidden=!r.enabled;$('v44-remote-info').hidden=!r.enabled;if(r.enabled){$('v44-remote-url').textContent=r.url;$('v44-remote-pin').textContent=r.pin;const remaining=Math.max(0,Math.ceil((Date.parse(r.expires)-Date.now())/60000));status('v44-remote-status','Nur im eigenen WLAN · PIN noch etwa '+remaining+' Min. gültig.');}else{status('v44-remote-status','Ausgeschaltet. Dein PC ist nicht übers WLAN steuerbar.')}}catch(e){status('v44-remote-status',e.message,true)}}
 for(const [id,act] of [['v44-remote-start','start'],['v44-remote-stop','stop'],['v44-remote-renew','renew']])$(id).addEventListener('click',async()=>{try{await api('remote',{action:act});await refreshRemote()}catch(e){status('v44-remote-status',e.message,true)}});
 $('v44-remote-copy').addEventListener('click',async()=>{try{await navigator.clipboard.writeText($('v44-remote-url').textContent);status('v44-remote-status','Adresse kopiert. Öffne sie auf dem Handy im gleichen WLAN.')}catch{status('v44-remote-status','Adresse kannst du auch direkt abtippen.')}});
 let lastRemote=0;
 async function processRemote(){try{const res=await api('remote/events');for(const ev of res.events||[]){if(ev.id<=lastRemote)continue;lastRemote=ev.id;
   if(['dashboard','spotify','soundboard'].includes(ev.action)&&typeof tab==='function')tab(ev.action);
   if(ev.action==='sound')playClip(ev.value);
  }}catch{/* desktop remains usable when remote off */}}
 setInterval(processRemote,1300);
 // New design profiles only change view-specific CSS; old Theme Studio and
 // stored v4.2 presets remain intact.
 const key='squad-v44-glass';const profiles=['aurora','forest','frost','ember'];
 function applyProfile(p){if(!profiles.includes(p))p='forest';document.body.dataset.v44Glass=p;document.querySelectorAll('[data-v44-preset]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.v44Preset===p)));}
 try{applyProfile(localStorage.getItem(key)||'forest')}catch{applyProfile('forest')}
 document.querySelectorAll('[data-v44-preset]').forEach(b=>b.addEventListener('click',()=>{const p=b.dataset.v44Preset;applyProfile(p);try{localStorage.setItem(key,p)}catch{}status('v44-glass-status','✓ Design ausgewählt. Deine bisherigen Farb- und Glaswerte bleiben erhalten.')}));
 // Hide loading indicators and stop false success messages on network errors.
 loadBoard();refreshRemote();
})();
