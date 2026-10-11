'use strict';
// Notification audio is synthesized locally: no uploaded sound files or extra cloud permissions.
(() => {
 const $=id=>document.getElementById(id);
 const defaults={messages:true,calls:true,sent:false,invites:true,volume:55,volumes:{}};
 let prefs={...defaults};try{prefs={...defaults,...JSON.parse(localStorage.getItem('squad-v454-alerts')||'{}')}}catch{}
 let ctx=null,ringInterval=null,testStream=null,testAudio=null,testSource=null,testAnalyser=null,testFrame=0;
 const webAudio=window.AudioContext||window.webkitAudioContext;
 function audioContext(){if(!webAudio)return null;if(!ctx)ctx=new webAudio();if(ctx.state==='suspended')ctx.resume().catch(()=>{});return ctx;}
 function save(){try{localStorage.setItem('squad-v454-alerts',JSON.stringify(prefs))}catch{}
  fetch('api/friend-prefs',{method:'POST',headers:{'X-Squad-Base':'1','Content-Type':'application/json'},body:JSON.stringify(prefs)}).catch(()=>{});}
 async function loadSaved(){try{const res=await fetch('api/friend-prefs',{cache:'no-store'});if(!res.ok)return;const loaded=await res.json();prefs={...defaults,...loaded,volumes:loaded.volumes||{}};
  for(const k of ['messages','calls','sent','invites'])if($('v454-notify-'+k))$('v454-notify-'+k).checked=!!prefs[k];
  if($('v454-notify-volume')){$('v454-notify-volume').value=prefs.volume;$('v454-notify-volume-text').textContent=prefs.volume+'%'}
 }catch{}}
 function tone(kind='messages'){
  if(kind!=='test'&&prefs[kind]===false)return;
  const audio=audioContext();if(!audio||prefs.volume===0)return;
  const now=audio.currentTime;const notes=kind==='calls'?[523,659,784]:kind==='sent'?[520]:kind==='invites'?[622,740]:[720,880];
  notes.forEach((hz,i)=>{const osc=audio.createOscillator(),amp=audio.createGain(),at=now+i*.15;
   osc.type='sine';osc.frequency.setValueAtTime(hz,at);amp.gain.setValueAtTime(0.0001,at);
   amp.gain.exponentialRampToValueAtTime(Math.max(.0002,prefs.volume/100*.075),at+.025);
   amp.gain.exponentialRampToValueAtTime(.0001,at+.20);osc.connect(amp);amp.connect(audio.destination);osc.start(at);osc.stop(at+.22);});
 }
 function toast(text){if(typeof window.notify==='function')window.notify(text);else {const t=$('toast');if(t){t.textContent=text;t.className='show';setTimeout(()=>t.className='',4000)}}}
 function desktop(title,body){if(document.hidden&&'Notification' in window&&Notification.permission==='granted'){try{const n=new Notification(title,{body,tag:'squad-friends'});setTimeout(()=>n.close(),6500)}catch{}}}
 function alert(kind,title,body){tone(kind);toast(title+(body?' · '+body:''));desktop(title,body)}
 function startRing(peer){stopRing();tone('calls');if(prefs.calls)ringInterval=setInterval(()=>tone('calls'),2600);desktop('SQUAD BASE · Anruf',peer+' ruft dich an')}
 function stopRing(){if(ringInterval!==null){clearInterval(ringInterval);ringInterval=null}}
 async function stopMic(){if(testFrame)cancelAnimationFrame(testFrame);testFrame=0;
  if(testStream){testStream.getTracks().forEach(t=>t.stop());testStream=null}try{testSource?.disconnect()}catch{};testSource=null;
  $('v454-mic-meter').style.width='0%';$('v454-mic-test').hidden=false;$('v454-mic-stop').hidden=true;
  if($('v454-test-status'))$('v454-test-status').textContent='Mikrofon ist wieder ausgeschaltet.';
 }
 async function startMic(){if(testStream)return;
  try{if(!navigator.mediaDevices?.getUserMedia)throw Error('Mikrofon nicht verfügbar. Windows-Berechtigung prüfen.');
   testStream=await navigator.mediaDevices.getUserMedia({audio:{echoCancellation:true,noiseSuppression:true},video:false});
   const a=audioContext();if(!a)throw Error('Audiotest wird vom Browser nicht unterstützt.');
   testSource=a.createMediaStreamSource(testStream);testAnalyser=a.createAnalyser();testAnalyser.fftSize=512;testSource.connect(testAnalyser);
   $('v454-mic-test').hidden=true;$('v454-mic-stop').hidden=false;$('v454-test-status').textContent='Mikrofon aktiv: Sprich etwas. Dein Audio wird nicht gespeichert oder übertragen.';
   const samples=new Uint8Array(testAnalyser.fftSize);
   function measure(){if(!testStream)return;testAnalyser.getByteTimeDomainData(samples);let sum=0;for(const v of samples){const d=(v-128)/128;sum+=d*d};
    const rms=Math.sqrt(sum/samples.length);$('v454-mic-meter').style.width=Math.min(100,Math.round(rms*420))+'%';testFrame=requestAnimationFrame(measure)}measure();
  }catch(err){await stopMic();$('v454-test-status').textContent='Mikrofontest fehlgeschlagen: '+err.message}
 }
 function setup(){
  for(const k of ['messages','calls','sent','invites']){const el=$('v454-notify-'+k);if(!el)continue;el.checked=Boolean(prefs[k]);el.onchange=()=>{prefs[k]=el.checked;save();if(k==='calls'&&!el.checked)stopRing()}}
  const vol=$('v454-notify-volume');if(vol){vol.value=prefs.volume;$('v454-notify-volume-text').textContent=prefs.volume+'%';vol.oninput=()=>{prefs.volume=Math.max(0,Math.min(100,Number(vol.value)||0));$('v454-notify-volume-text').textContent=prefs.volume+'%';save()}}
  if($('v454-test-tone'))$('v454-test-tone').onclick=()=>tone('test');
  if($('v454-enable-desktop'))$('v454-enable-desktop').onclick=async()=>{if(!('Notification' in window)){$('v454-test-status').textContent='Windows-Desktopbenachrichtigungen sind hier nicht verfügbar.';return}
   try{const result=await Notification.requestPermission();$('v454-test-status').textContent=result==='granted'?'Desktop-Benachrichtigungen erlaubt.':'Keine Desktop-Erlaubnis erteilt. Hinweise in der App bleiben aktiv.'}catch(err){$('v454-test-status').textContent='Desktop-Hinweise nicht unterstützt: '+err.message}}
  if($('v454-mic-test'))$('v454-mic-test').onclick=startMic;if($('v454-mic-stop'))$('v454-mic-stop').onclick=stopMic;
  window.addEventListener('beforeunload',stopMic);loadSaved();
  // Browsers can require an interaction before WebAudio is allowed to play.
  document.addEventListener('pointerdown',()=>audioContext(),{once:true});
 }
 function peerVolume(peer){const value=prefs.volumes?.[peer];return Number.isInteger(value)&&value>=0&&value<=100?value:100}
 function savePeerVolume(peer,level){if(!/^[a-z0-9_]{3,20}$/.test(peer))return;prefs.volumes??={};prefs.volumes[peer]=Math.max(0,Math.min(100,level));save()}
 window.SquadFriendAlerts={alert,tone,startRing,stopRing,setup,stopMic,peerVolume,savePeerVolume};
 if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',setup);else setup();
})();
