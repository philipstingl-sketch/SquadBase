'use strict';
// SQUAD BASE 4.3: all optional features and settings stay local by default.
(() => {
 const $=id=>document.getElementById(id);
 const request=async(route,body)=>{
  const opts={cache:'no-store'};
  if(body!==undefined){opts.method='POST';opts.headers={'Content-Type':'application/json','X-Squad-Base':'1'};opts.body=JSON.stringify(body)}
  const res=await fetch('api/'+route,opts);let data={};try{data=await res.json()}catch{}
  if(!res.ok)throw Error(data.error||`HTTP ${res.status}`);
  return data;
 };
 const report=(id,msg,bad=false)=>{const target=$(id);if(!target)return;target.textContent=msg;target.classList.toggle('v43-error',bad)};
 const connected=()=>typeof spotifyConnected!=='undefined'&&!!spotifyConnected;
 let queueBusy=false;
 async function loadQueue(){
  if(queueBusy)return;
  if(!connected()){report('v43-music-status','Bitte zuerst Spotify verbinden. Die Windows-Medientasten funktionieren unabhängig davon.',true);return}
  queueBusy=true;const button=$('v43-queue-refresh');button.disabled=true;
  report('v43-music-status','Lade Warteschlange …');
  try{
   const data=await request('spotify/queue'),items=Array.isArray(data.items)?data.items:[];
   const root=$('v43-queue');root.replaceChildren();$('v43-queue-count').textContent=items.length+' Titel';
   if(!items.length){const empty=document.createElement('p');empty.className='hint';empty.textContent='Spotify meldet eine leere Warteschlange.';root.append(empty)}
   items.forEach((item,index)=>{const row=document.createElement('div');row.className='v43-queue-row';
    const order=document.createElement('span');order.className='v43-queue-index';order.textContent=String(index+1).padStart(2,'0');
    const detail=document.createElement('div');const name=document.createElement('strong');name.textContent=item.title||'Ohne Titel';const artist=document.createElement('small');artist.textContent=item.artists||'Spotify';detail.append(name,artist);row.append(order,detail);root.append(row)});
   report('v43-music-status','Die Warteschlange stammt direkt von Spotify. Songs bleiben auf dem Spotify-Wiedergabegerät.');
  }catch(e){report('v43-music-status',e.message+' · Öffne Spotify, falls deine Developer-App diese Funktion nicht unterstützt.',true)}
  finally{queueBusy=false;button.disabled=false}
 }
 $('v43-queue-refresh')?.addEventListener('click',loadQueue);
 window.addEventListener('squad:queue-changed',()=>loadQueue());
 let lastShuffle=null,lastRepeat='off';const modes=['off','context','track'];
 function updateOptions(now){
  const yes=connected(),shuffle=$('v43-shuffle'),repeat=$('v43-repeat');
  shuffle.disabled=!yes;repeat.disabled=!yes;
  if(!yes){shuffle.textContent='⇄ Shuffle (Login nötig)';repeat.textContent='↻ Repeat (Login nötig)';return}
  if(now?.active){lastShuffle=!!now.shuffle;if(modes.includes(now.repeat))lastRepeat=now.repeat}
  shuffle.textContent=lastShuffle?'⇄ Shuffle an':'⇄ Shuffle aus';shuffle.setAttribute('aria-pressed',String(!!lastShuffle));
  repeat.textContent={'off':'↻ Repeat aus','context':'↻ Ganze Playlist','track':'↻ Ein Titel'}[lastRepeat];
 }
 window.addEventListener('squad:spotify-now',e=>updateOptions(e.detail?.now));
 $('v43-shuffle')?.addEventListener('click',async e=>{const b=e.currentTarget;const next=!lastShuffle;b.disabled=true;
  try{await request('spotify/playback-options',{action:'shuffle',enabled:next});lastShuffle=next;updateOptions(null);report('v43-music-status','Shuffle '+(next?'aktiviert.':'ausgeschaltet.'))}
  catch(err){report('v43-music-status',err.message,true)}finally{b.disabled=!connected()}
 });
 $('v43-repeat')?.addEventListener('click',async e=>{const b=e.currentTarget;const next=modes[(modes.indexOf(lastRepeat)+1)%modes.length];b.disabled=true;
  try{await request('spotify/playback-options',{action:'repeat',repeat:next});lastRepeat=next;updateOptions(null);report('v43-music-status','Wiederholung aktualisiert.')}
  catch(err){report('v43-music-status',err.message,true)}finally{b.disabled=!connected()}
 });
 updateOptions(null);
 // Render local health diagnostics only after opt-in: never collect account or file data.
 let diagnostic=null;
 async function loadDiagnostic(){
  report('v43-ai-summary','Prüfe PC-Zustand …');
  try{
   diagnostic=await request('assistant/context');
   const cpu=Number(diagnostic.cpuPercent).toFixed(0),ram=Number(diagnostic.ramPercent).toFixed(0);
   const items=[`CPU ${cpu}%`,`RAM ${ram}%`,diagnostic.gpuAvailable?'GPU-Messung vorhanden':'GPU-Messung nicht verfügbar',diagnostic.overlayEnabled?'Overlay sichtbar':'Overlay ausgeblendet',diagnostic.spotifyConnected?'Spotify verbunden':'Spotify nicht verbunden',`${diagnostic.libraryGames} Spiele zuletzt im Bibliothekscache`];
   report('v43-ai-summary',items.join('  ·  '));
  }catch(e){diagnostic=null;report('v43-ai-summary',e.message,true)}
 }
 $('v43-ai-check')?.addEventListener('click',loadDiagnostic);
 $('v43-ai-explain')?.addEventListener('click',async()=>{
  if(!diagnostic)await loadDiagnostic();if(!diagnostic)return;
  const input=$('assistant-input');if(!input)return;
  input.value=`Bitte erkläre mir meine lokalen SQUAD-BASE-Werte (CPU ${Math.round(diagnostic.cpuPercent)}%, RAM ${Math.round(diagnostic.ramPercent)}%, GPU verfügbar: ${diagnostic.gpuAvailable?'ja':'nein'}, Spotify verbunden: ${diagnostic.spotifyConnected?'ja':'nein'}). Welche Einstellungen könnte ich prüfen?`;
  input.focus();report('v43-ai-summary','Diagnose eingefügt. Du kannst den Text prüfen und selbst auf „Senden“ klicken.');
 });
 document.querySelectorAll('[data-v43-open]').forEach(button=>button.addEventListener('click',()=>{
  const destination=button.dataset.v43Open;
  if(['network','library','maintenance'].includes(destination)&&typeof tab==='function')tab(destination)
 }));
 // Automatic theme does not overwrite the saved glass preset from v4.2.
 let theme={autoDayNight:false,nightFrom:20,dayFrom:7,nightGlass:'midnight',reducedMotion:false};let daytimeStyle='frosted';
 function nightNow(){const hour=new Date().getHours(),n=theme.nightFrom,d=theme.dayFrom;return n>d?(hour>=n||hour<d):(hour>=n&&hour<d)}
 function applyTheme(){
  const selected=document.querySelector('[data-glass][aria-pressed="true"]');
  if(selected&&selected.dataset.glass&&document.body.dataset.glassStyle!==theme.nightGlass)daytimeStyle=selected.dataset.glass;
  const night=theme.autoDayNight&&nightNow();
  document.body.classList.toggle('v43-night',night);
  document.body.classList.toggle('v43-reduced-motion',!!theme.reducedMotion);
  document.body.dataset.glassStyle=night?theme.nightGlass:(daytimeStyle||'frosted');
 }
 async function loadTheme(){
  for(let hour=0;hour<24;hour++){const label=String(hour).padStart(2,'0')+':00';$('v43-night-from').append(new Option(label,String(hour)));$('v43-day-from').append(new Option(label,String(hour)))}
  try{theme=await request('theme-pro');
   $('v43-auto-theme').checked=theme.autoDayNight;$('v43-reduce-motion').checked=theme.reducedMotion;
   $('v43-night-from').value=String(theme.nightFrom);$('v43-day-from').value=String(theme.dayFrom);$('v43-night-style').value=theme.nightGlass;
   const selected=document.querySelector('[data-glass][aria-pressed="true"]');if(selected)daytimeStyle=selected.dataset.glass;
   applyTheme();report('v43-theme-status','Cozy-Automatik bereit. Änderungen werden erst mit „Speichern“ übernommen.');
  }catch(err){report('v43-theme-status',err.message,true)}
 }
 $('v43-theme-save')?.addEventListener('click',async()=>{
  const next={autoDayNight:$('v43-auto-theme').checked,reducedMotion:$('v43-reduce-motion').checked,nightFrom:Number($('v43-night-from').value),dayFrom:Number($('v43-day-from').value),nightGlass:$('v43-night-style').value};
  try{await request('theme-pro',next);theme=next;applyTheme();report('v43-theme-status','✓ Tag-/Nachtstil gespeichert. Gilt beim nächsten Start ebenfalls.')}catch(e){report('v43-theme-status',e.message,true)}
 });
 document.querySelectorAll('[data-glass]').forEach(button=>button.addEventListener('click',()=>{
  daytimeStyle=button.dataset.glass;
  setTimeout(applyTheme,100);
 }));
 window.setInterval(applyTheme,60000);
 loadTheme();
 // UI maintenance: avoid rigid height clipping on narrow and scaled displays.
 window.addEventListener('resize',()=>{document.documentElement.style.setProperty('--v43-vh',window.innerHeight+'px')});
})();
