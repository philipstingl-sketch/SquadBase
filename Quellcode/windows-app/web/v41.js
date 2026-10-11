'use strict';
// SQUAD BASE 4.1 – all per-user choices are stored by the local Go backend.
(() => {
 const el=id=>document.getElementById(id);
 const cards=['system','monitor','games','network','news'];
 const names={system:'System & Hardware',monitor:'Hardware Monitor Pro',games:'Spiele',network:'Internet',news:'Spiele-News'};
 const presets={
  gaming:{order:['system','monitor','games','network','news'],hidden:[],wide:['news']},
  cozy:{order:['system','games','monitor','news','network'],hidden:['network'],wide:['news']},
  minimal:{order:['system','games','network','monitor','news'],hidden:['monitor','news'],wide:['system']},
  social:{order:['games','news','network','system','monitor'],hidden:['monitor'],wide:['news']}
 };
 let prefs={preset:'gaming',...presets.gaming,readAloud:false,voiceRate:1,voiceName:'',spotifyDock:'bar'};
 let editing=false, board, selectedDrag=null, saveQueue=Promise.resolve();
 const notifyMsg=(message,bad=false)=>{ if(typeof notify==='function')notify(message,bad) };
 async function savePrefs(){
  const snapshot=JSON.parse(JSON.stringify(prefs));
  saveQueue=saveQueue.catch(()=>{}).then(async()=>{
   const result=await api('workspace',snapshot);if(!result.saved)throw Error('Speichern fehlgeschlagen');
   el('v41-layout-status').textContent='✓ '+(snapshot.preset==='custom'?'Individuell':snapshot.preset)+' · gespeichert';
  }).catch(e=>{el('v41-layout-status').textContent='⚠ Speichern fehlgeschlagen';notifyMsg('Arbeitsbereich: '+e.message,true)});
  return saveQueue;
 }
 const cardFor=id=>board?.querySelector(`[data-dashboard-card="${id}"]`);
 function makeSidebarGroups(){
  const nav=el('nav');if(!nav)return;
  const groups=[{label:'🎮 Spiele & News',ids:['games','library','updates']},{label:'🧰 Tools',ids:['assistant','soundboard','stats','discord','maintenance','remote']}];
  for(const grp of groups){
   const details=document.createElement('details');details.className='v41-nav-group';
   const summary=document.createElement('summary');summary.textContent=grp.label;details.append(summary);
   const box=document.createElement('div');box.className='v41-nav-links';details.append(box);
   for(const name of grp.ids){const b=nav.querySelector(`button.tab[data-tab="${name}"]`);if(b)box.append(b)}
   const anchor=grp.ids[0]==='games'?nav.querySelector('[data-tab="dashboard"]'):nav.querySelector('[data-tab="minigames"]');
   anchor?.after(details);
   details.addEventListener('toggle',()=>{if(details.open)document.querySelectorAll('.v41-nav-group').forEach(d=>{if(d!==details)d.open=false})});
   box.addEventListener('click',e=>{if(window.innerWidth<=750&&e.target.closest('button[data-tab]'))details.open=false});
  }
  // Dashboard, Freunde, Minispiele and Spotify remain direct tabs. KI and Soundboard live under Tools.
  document.addEventListener('click',e=>{
   const b=e.target.closest('button[data-tab],button[data-goto]');if(!b)return;
   const target=b.dataset.tab||b.dataset.goto;
   for(const d of nav.querySelectorAll('.v41-nav-group'))if(d.querySelector(`[data-tab="${target}"]`))d.open=true;
  });
 }
 function initCards(){
  const section=el('dashboard');if(!section)return;
  board=document.createElement('div');board.id='v41-dashboard-board';board.className='v41-dashboard-board';
  // Existing live cards are MOVED, not copied: handlers, ids and real data remain untouched.
  const upper=section.querySelector('.grid.two');
  const lower=section.querySelector('.grid.two.lower');
  const news=section.querySelector('.dashboard-news-panel');
  const articles=[...upper.querySelectorAll(':scope > article'),...lower.querySelectorAll(':scope > article'),news];
  const cardIds=['system','monitor','games','network','news'];
  articles.forEach((card,index)=>{
   const id=cardIds[index];card.dataset.dashboardCard=id;card.dataset.dashboardLabel=names[id];
   card.classList.add('v41-dashboard-card');
   const controls=document.createElement('div');controls.className='v41-card-editbar';
   const handle=document.createElement('span');handle.className='v41-card-handle';handle.textContent='⠿ '+names[id];handle.title='Zum Verschieben ziehen';handle.setAttribute('aria-label',names[id]+' verschieben');
   const up=editBtn('↑','Nach oben');const down=editBtn('↓','Nach unten');const wide=editBtn('⇔','Breite wechseln');const hide=editBtn('◉','Ausblenden / Einblenden');
   up.addEventListener('click',()=>move(id,-1));down.addEventListener('click',()=>move(id,1));wide.addEventListener('click',()=>toggleWidth(id));hide.addEventListener('click',()=>toggleHidden(id));
   controls.append(handle,up,down,wide,hide);card.prepend(controls);card.draggable=false;board.append(card);
   card.addEventListener('dragstart',e=>{if(!editing){e.preventDefault();return}selectedDrag=id;e.dataTransfer.effectAllowed='move';e.dataTransfer.setData('text/plain',id);card.classList.add('v41-dragging')});
   card.addEventListener('dragend',()=>{selectedDrag=null;card.classList.remove('v41-dragging')});
   card.addEventListener('dragover',e=>{if(!editing||!selectedDrag)return;e.preventDefault();e.dataTransfer.dropEffect='move'});
   card.addEventListener('drop',e=>{if(!editing||!selectedDrag||selectedDrag===id)return;e.preventDefault();let newOrder=prefs.order.filter(x=>x!==selectedDrag);let at=newOrder.indexOf(id);newOrder.splice(at,0,selectedDrag);prefs.order=newOrder;prefs.preset='custom';applyLayout();savePrefs()});
  });
  upper.remove();lower.remove();
  // The news card was moved into the board; insert board below the editor toolbar.
  el('v41-edit-help').after(board);
  const buttons=[...section.querySelectorAll('button[data-goto]')]; // continue to use original event delegation
  void buttons;
 }
 function editBtn(symbol,label){const b=document.createElement('button');b.type='button';b.className='v41-icon-btn';b.textContent=symbol;b.title=label;b.setAttribute('aria-label',label);return b}
 function applyLayout(){
  if(!board)return;
  const selected=prefs.order.filter(id=>cards.includes(id));for(const id of cards)if(!selected.includes(id))selected.push(id);
  for(const id of selected){const card=cardFor(id);if(!card)continue;board.append(card);card.classList.toggle('v41-is-wide',prefs.wide.includes(id));card.classList.toggle('v41-is-hidden',prefs.hidden.includes(id));card.draggable=editing}
  board.classList.toggle('v41-edit-mode',editing);
  el('v41-edit-toggle').textContent=editing?'✓ Bearbeiten beenden':'✎ Dashboard bearbeiten';
  el('v41-edit-toggle').setAttribute('aria-pressed',String(editing));
  el('v41-edit-help').hidden=!editing;el('v41-layout-reset').hidden=!editing;
  el('v41-layout-preset').value=prefs.preset;
  el('v41-layout-status').textContent=(prefs.preset==='custom'?'Individuelles Layout':({gaming:'Gaming',cozy:'Cozy',minimal:'Minimal',social:'Social'}[prefs.preset]))+' · lokal gespeichert';
 }
 function move(id,dir){const xs=prefs.order;const at=xs.indexOf(id);const next=at+dir;if(next<0||next>=xs.length)return;[xs[at],xs[next]]=[xs[next],xs[at]];prefs.preset='custom';applyLayout();savePrefs()}
 function toggleWidth(id){prefs.wide=prefs.wide.includes(id)?prefs.wide.filter(x=>x!==id):[...prefs.wide,id];prefs.preset='custom';applyLayout();savePrefs()}
 function toggleHidden(id){if(!prefs.hidden.includes(id)){if(prefs.hidden.length===cards.length-1){notifyMsg('Mindestens eine Karte muss sichtbar bleiben.',true);return}prefs.hidden.push(id)}else prefs.hidden=prefs.hidden.filter(x=>x!==id);prefs.preset='custom';applyLayout();savePrefs()}
 function initDashboard(){
  initCards();
  el('v41-layout-preset').addEventListener('change',e=>{
   const preset=e.target.value;if(preset==='custom'){prefs.preset='custom'}else{prefs={...prefs,preset,...JSON.parse(JSON.stringify(presets[preset]))}}
   applyLayout();savePrefs();
  });
  el('v41-edit-toggle').addEventListener('click',()=>{editing=!editing;applyLayout()});
  el('v41-layout-reset').addEventListener('click',()=>{const mode=prefs.preset==='custom'?'gaming':prefs.preset;prefs={...prefs,preset:mode,...JSON.parse(JSON.stringify(presets[mode]))};applyLayout();savePrefs()});
 }
 function setDock(){const mini=el('spotify-mini');if(!mini)return;
  mini.dataset.dock=prefs.spotifyDock;
  el('v41-spotify-dock-select').value=prefs.spotifyDock;
  if(prefs.spotifyDock==='hidden')mini.hidden=true;
  else if(typeof spotifyConnected==='boolean')mini.hidden=!spotifyConnected;
 }
 function initRecovery(){const btn=el('v41-config-restore');btn?.addEventListener('click',async()=>{
 if(!confirm('Letzte SQUAD-BASE-Sicherung wiederherstellen? Deine aktuellen lokalen Einstellungen werden vorher gesichert. Änderungen gelten nach einem Neustart.'))return;
 btn.disabled=true;try{const x=await api('diagnostics',{action:'restore-latest'});notifyMsg(x.message||'Wiederhergestellt. App neu starten.')}catch(err){notifyMsg(err.message,true)}finally{btn.disabled=false}
 });}
 function initDock(){el('v41-spotify-dock-select').addEventListener('change',e=>{prefs.spotifyDock=e.target.value;setDock();savePrefs()});setDock()}
 let recognizer=null,recognizing=false;let utteranceCount=0;
 function voiceStatus(msg){el('v41-voice-indicator').textContent=msg}
 function stopRecognition(){try{recognizer?.stop()}catch{}recognizing=false;el('v41-voice-record').setAttribute('aria-pressed','false');el('v41-voice-record').textContent='🎙 Spracheingabe'}
 function initVoice(){
  const Mic=window.SpeechRecognition||window.webkitSpeechRecognition;
  const recButton=el('v41-voice-record'),input=el('assistant-input');
  recButton.addEventListener('click',()=>{
   if(recognizing){stopRecognition();voiceStatus('Spracheingabe beendet.');return}
   if(!Mic){input.focus();voiceStatus('Spracherkennung hier nicht unterstützt. Alternativ Windows-Taste + H.');return}
   try{
    recognizer=new Mic();recognizer.lang='de-DE';recognizer.interimResults=false;recognizer.continuous=false;recognizer.maxAlternatives=1;
    recognizer.onresult=e=>{const text=e.results?.[0]?.[0]?.transcript||'';if(text){input.value=(input.value?input.value+' ':'')+text;input.dispatchEvent(new Event('input'));voiceStatus('Text erkannt · jetzt Senden drücken')}};
    recognizer.onerror=e=>{voiceStatus('Spracheingabe: '+String(e.error||'Fehler')+' · prüfe Mikrofonfreigabe')};
    recognizer.onend=()=>{recognizing=false;recButton.setAttribute('aria-pressed','false');recButton.textContent='🎙 Spracheingabe'};
    recognizer.start();recognizing=true;recButton.setAttribute('aria-pressed','true');recButton.textContent='● Aufnahme stoppen';voiceStatus('Mikrofon aktiv – sprich jetzt. Keine Aufnahme in der App gespeichert.');
   }catch(e){voiceStatus('Mikrofon konnte nicht gestartet werden: '+e.message)}
  });
  const read=el('v41-read-aloud');read.checked=!!prefs.readAloud;
  read.addEventListener('change',()=>{prefs.readAloud=read.checked;if(!read.checked)window.speechSynthesis?.cancel();savePrefs()});
  const rate=el('v41-voice-rate');rate.value=prefs.voiceRate;el('v41-voice-rate-value').textContent=Number(prefs.voiceRate).toFixed(1).replace('.',',')+'×';
  rate.addEventListener('change',()=>{prefs.voiceRate=Number(rate.value);el('v41-voice-rate-value').textContent=prefs.voiceRate.toFixed(1).replace('.',',')+'×';savePrefs()});
  const select=el('v41-voice-name');
  function refreshVoices(){const engine=window.speechSynthesis;if(!engine)return;const list=engine.getVoices();const chosen=prefs.voiceName;select.replaceChildren(new Option('Windows-Standard',''));
   for(const v of list.filter(v=>v.lang.startsWith('de')).concat(list.filter(v=>!v.lang.startsWith('de')))){select.append(new Option(v.name+' · '+v.lang,v.name))}
   select.value=[...select.options].some(o=>o.value===chosen)?chosen:'';
  }
  if(window.speechSynthesis){refreshVoices();window.speechSynthesis.onvoiceschanged=refreshVoices}
  else voiceStatus('Sprachausgabe in diesem App-Fenster nicht verfügbar.');
  select.addEventListener('change',()=>{prefs.voiceName=select.value;savePrefs()});
  el('v41-voice-stop').addEventListener('click',()=>{window.speechSynthesis?.cancel();stopRecognition();voiceStatus('Sprachausgabe gestoppt.')});
  window.addEventListener('squad:assistant-answer',e=>{
   if(!prefs.readAloud||!window.speechSynthesis)return;
   const text=e.detail?.text;if(typeof text!=='string'||!text.trim())return;
   window.speechSynthesis.cancel();const speech=new SpeechSynthesisUtterance(text.slice(0,1500));speech.lang='de-DE';speech.rate=prefs.voiceRate;const chosen=window.speechSynthesis.getVoices().find(v=>v.name===prefs.voiceName);if(chosen)speech.voice=chosen;
   speech.onerror=()=>voiceStatus('Vorlesen konnte nicht gestartet werden.');
   window.speechSynthesis.speak(speech);utteranceCount++;
  });
  document.addEventListener('visibilitychange',()=>{if(document.hidden)stopRecognition()});
 }
 async function initialize(){makeSidebarGroups();initDashboard();try{
  const saved=await api('workspace');if(saved?.order?.length===cards.length)prefs=saved;
 }catch(err){notifyMsg('Arbeitsbereich konnte nicht geladen werden: '+err.message,true)}
 applyLayout();initDock();initVoice();initRecovery();
 }
 initialize();
})();
