'use strict';
// SQUAD BASE 4.2 – all per-user choices are stored by the local Go backend.
(() => {
 const el=id=>document.getElementById(id);
 const cards=['system','monitor','games','network','news','clockwidget','musicwidget','friendswidget','sessionwidget'];
 const names={system:'System & Hardware',monitor:'Hardware Monitor Pro',games:'Spiele',network:'Internet',news:'Spiele-News',clockwidget:'Cozy Uhr',musicwidget:'Spotify Widget',friendswidget:'Freunde & Chat',sessionwidget:'Gaming Session'};
 const allWidgets=['clockwidget','musicwidget','friendswidget','sessionwidget'];
 const presets={
  gaming:{order:['system','monitor','games','network','news',...allWidgets],hidden:[...allWidgets],wide:['news']},
  cozy:{order:['system','clockwidget','games','musicwidget','monitor','news','network','friendswidget','sessionwidget'],hidden:['network','friendswidget','sessionwidget'],wide:['news']},
  minimal:{order:['system','clockwidget','games','network','monitor','news','musicwidget','friendswidget','sessionwidget'],hidden:['network','monitor','news','musicwidget','friendswidget','sessionwidget'],wide:['system']},
  social:{order:['friendswidget','musicwidget','games','news','network','system','monitor','clockwidget','sessionwidget'],hidden:['monitor','clockwidget','sessionwidget'],wide:['news']}
 };
 let prefs={preset:'gaming',...presets.gaming,readAloud:false,voiceRate:1,voiceName:'',spotifyDock:'bar',version:2,cozyStart:true,smartGaming:true,glassStyle:'frosted'};
 let editing=false, board, selectedDrag=null, saveQueue=Promise.resolve(), beforeEdit=null;
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
   const details=document.createElement('details');details.className='v41-nav-group';details.dataset.v43Group=grp.ids[0]==='games'?'games':'tools';
   const summary=document.createElement('summary');summary.textContent=grp.label;summary.title=grp.label;details.append(summary);
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
  const widgetTemplates={
    clockwidget:`<div class="section-head"><div><h2>◷ Cozy Uhr</h2><p>Lokale Zeit und Datum</p></div><span class="tag">LIVE</span></div><div class="v42-clock-display" id="v42-clock-display">--:--</div><div class="v42-widget-detail" id="v42-clock-date">Datum wird geladen</div><button class="quiet" data-goto="settings">Uhr gestalten →</button>`,
    musicwidget:`<div class="section-head"><div><h2>♫ Spotify</h2><p>Deine Musik auf einen Blick</p></div><span class="tag">PLAYER</span></div><div class="v42-widget-main">♫ <strong id="v42-widget-track">Nicht verbunden</strong></div><div class="v42-widget-detail" id="v42-widget-artist">Melde dich in Spotify an, um deinen Song zu sehen.</div><button class="outline" type="button" data-goto="spotify">Spotify öffnen ↗</button>`,
    friendswidget:`<div class="section-head"><div><h2>☻ Deine Freunde</h2><p>Freunde & Nachrichten</p></div><span class="tag">PRIVAT</span></div><div class="v42-widget-main"><strong id="v42-widget-friends">Nicht verbunden</strong></div><div class="v42-widget-detail">Die Kontodaten bleiben im Freunde-System.</div><button class="outline" data-goto="friends" type="button">Zu Freunden & Chat →</button>`,
    sessionwidget:`<div class="section-head"><div><h2>⚡ Gaming Session</h2><p>Deine erkannten Spiele</p></div><span class="tag">LOKAL</span></div><div class="v42-widget-main"><strong id="v42-widget-session">Kein Spiel erkannt</strong></div><div class="v42-widget-detail" id="v42-widget-hours">Spielzeit wird geladen</div><button type="button" class="outline" data-goto="library">Spielebibliothek →</button>`
  };
  for(const id of allWidgets){const card=document.createElement('article');card.className='panel v42-widget';card.innerHTML=widgetTemplates[id];articles.push(card)}
  const cardIds=cards;
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
   card.addEventListener('drop',e=>{if(!editing||!selectedDrag||selectedDrag===id)return;e.preventDefault();let newOrder=prefs.order.filter(x=>x!==selectedDrag);let at=newOrder.indexOf(id);newOrder.splice(at,0,selectedDrag);prefs.order=newOrder;prefs.preset='custom';applyLayout()});
  });
  upper.remove();lower.remove();
  // The news card was moved into the board; insert board below the editor toolbar.
  el('v41-edit-help').after(board);
  // Existing global event delegation continues to handle widget shortcut buttons.
 }
 function editBtn(symbol,label){const b=document.createElement('button');b.type='button';b.className='v41-icon-btn';b.textContent=symbol;b.title=label;b.setAttribute('aria-label',label);return b}
 function applyLayout(){
  if(!board)return;
  const selected=prefs.order.filter(id=>cards.includes(id));for(const id of cards)if(!selected.includes(id))selected.push(id);
  for(const id of selected){const card=cardFor(id);if(!card)continue;board.append(card);card.classList.toggle('v41-is-wide',prefs.wide.includes(id));card.classList.toggle('v41-is-hidden',prefs.hidden.includes(id));card.draggable=editing}
  board.classList.toggle('v41-edit-mode',editing);
  el('v41-edit-toggle').hidden=editing;el('v41-edit-toggle').setAttribute('aria-pressed',String(editing));
  el('v42-save').hidden=!editing;el('v42-cancel').hidden=!editing;el('v42-edit-options').hidden=!editing;
  el('v41-edit-help').hidden=!editing;el('v41-layout-reset').hidden=!editing;
  el('v41-layout-preset').value=prefs.preset;
  const msg=prefs.preset==='custom'?'Individuelles Layout':({gaming:'Gaming',cozy:'Cozy',minimal:'Minimal',social:'Social'}[prefs.preset]||'Gaming');
  el('v41-layout-status').textContent=msg+(editing?' · Bearbeitungsmodus':' · gespeichert');
 }
 function markEdit(){prefs.preset='custom';applyLayout()}
 function move(id,dir){if(!editing)return;const xs=prefs.order;const at=xs.indexOf(id);const next=at+dir;if(next<0||next>=xs.length)return;[xs[at],xs[next]]=[xs[next],xs[at]];markEdit()}
 function toggleWidth(id){if(!editing)return;prefs.wide=prefs.wide.includes(id)?prefs.wide.filter(x=>x!==id):[...prefs.wide,id];markEdit()}
 function toggleHidden(id){if(!editing)return;if(!prefs.hidden.includes(id)){if(prefs.hidden.length===cards.length-1){notifyMsg('Mindestens eine Karte muss sichtbar bleiben.',true);return}prefs.hidden.push(id)}else prefs.hidden=prefs.hidden.filter(x=>x!==id);markEdit()}
 function beginEditing(){if(editing)return;beforeEdit=JSON.parse(JSON.stringify(prefs));editing=true;applyLayout()}
 function cancelEditing(){if(!editing)return;prefs=beforeEdit||prefs;editing=false;beforeEdit=null;applyLayout();notifyMsg('Änderungen verworfen.')}
 async function finishEditing(){if(!editing)return;const btn=el('v42-save');btn.disabled=true;const original=prefs.preset;
  // Ensure customized order/selection stays as a custom preset.
  try{const saved=await api('workspace',JSON.parse(JSON.stringify(prefs)));if(!saved.saved)throw Error('Speichern fehlgeschlagen');editing=false;beforeEdit=null;applyLayout();notifyMsg('Dashboard gespeichert.')}catch(e){prefs.preset=original;notifyMsg('Speichern fehlgeschlagen: '+e.message,true)}finally{btn.disabled=false}
 }
 function initDashboard(){
  initCards();
  el('v41-layout-preset').addEventListener('change',e=>{
   if(!editing)return;const preset=e.target.value;
   if(preset==='custom'){prefs.preset='custom'}else{prefs={...prefs,preset,...JSON.parse(JSON.stringify(presets[preset]))}}
   applyLayout();
  });
  el('v41-edit-toggle').addEventListener('click',beginEditing);
  el('v42-open-edit').addEventListener('click',()=>{tab('dashboard');beginEditing()});
  el('v42-save').addEventListener('click',finishEditing);
  el('v42-cancel').addEventListener('click',cancelEditing);
  el('v41-layout-reset').addEventListener('click',()=>{if(!editing)return;const mode=prefs.preset==='custom'?'gaming':prefs.preset;prefs={...prefs,preset:mode,...JSON.parse(JSON.stringify(presets[mode]))};applyLayout()});
  el('v42-widget-add').addEventListener('click',()=>{if(!editing)return;const id=el('v42-widget-select').value;if(!allWidgets.includes(id))return;prefs.hidden=prefs.hidden.filter(x=>x!==id);markEdit();cardFor(id)?.scrollIntoView({block:'nearest',behavior:'smooth'});el('v42-widget-select').value=''});
  document.addEventListener('keydown',event=>{if(editing&&event.key==='Escape'){event.preventDefault();cancelEditing()}});
 }
 function setDock(){const mini=el('spotify-mini');if(!mini)return;
  mini.dataset.dock=prefs.spotifyDock;
  el('v41-spotify-dock-select').value=prefs.spotifyDock;
  if(prefs.spotifyDock==='hidden')mini.hidden=true;
  else if(typeof spotifyConnected==='boolean')mini.hidden=!spotifyConnected && !(typeof spotifyDesktopControls==='boolean' && spotifyDesktopControls);
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
 function updateWelcomeAndWidgets(){
  const now=new Date(),hour=now.getHours(),shortTime=now.toLocaleTimeString('de-DE',{hour:'2-digit',minute:'2-digit'});
  const day=now.toLocaleDateString('de-DE',{weekday:'long',day:'numeric',month:'long',year:'numeric'});
  el('v42-greeting').textContent=hour<11?'Guten Morgen ✦':hour<18?'Schönen Nachmittag ✦':'Schönen Abend ✦';
  el('v42-hero-clock').textContent=shortTime;el('v42-clock-display').textContent=shortTime;
  el('v42-hero-date').textContent=day;el('v42-clock-date').textContent=day;
  const current=typeof state==='object'&&state?state:null;
  const active=games.filter(g=>current?.playing?.[g.id]);
  const gameName=active.length?active.map(g=>g.name).join(', '):'Kein Spiel erkannt';
  el('v42-hero-game').textContent=gameName;el('v42-widget-session').textContent=gameName;
  el('v42-hero-system').textContent=current?.hardware?.gpuName?current.hardware.gpuName:'System bereit';
  let total=0;for(const g of games)total+=Number(current?.usage?.[g.id]||0);
  el('v42-widget-hours').textContent='Erfasste Spielzeit: '+fmtTime(total);
  const song=el('spotify-song')?.textContent||'';
  el('v42-widget-track').textContent=(song&&song!=='Noch kein Song')?song:'Spotify nicht verbunden';
  el('v42-widget-artist').textContent=(song&&song!=='Noch kein Song')?(el('spotify-artist')?.textContent||''):'Deine Playlists findest du im Spotify-Reiter.';
  const friendApp=el('friends-app');const online=friendApp&&!friendApp.hidden;
  const friendNodes=online?[...document.querySelectorAll('#friend-list button.friend-item')]:[];
  const connected=friendNodes.filter(n=>n.textContent.includes('online')).length;
  el('v42-widget-friends').textContent=online?friendNodes.length+' Freunde · '+connected+' online':'Noch nicht verbunden';
  const smart=!!prefs.smartGaming&&active.length>0;
  if(document.body.classList.contains('v42-smart-active')!==smart){document.body.classList.toggle('v42-smart-active',smart);window.dispatchEvent(new Event('squad-smart-mode'))}
  el('v42-smart-status').textContent=!prefs.smartGaming?'Smart Gaming ist ausgeschaltet.':smart?'⚡ Smart Gaming aktiv: Animationen und Hintergrundeffekte sind reduziert, während das Spiel erkannt wird.':'Smart Gaming ist bereit und aktiviert sich automatisch bei einem erkannten Spiel.';
 }
 function applyGlass(){
  const name=['frosted','crystal','warm','midnight'].includes(prefs.glassStyle)?prefs.glassStyle:'frosted';
  document.body.dataset.glassStyle=name;
  document.querySelectorAll('[data-glass]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.glass===name)));
  el('v42-glass-status').textContent='Design: '+({frosted:'Frosted',crystal:'Crystal',warm:'Warm Oak',midnight:'Midnight'}[name])+' · lokal gespeichert';
 }
 function initGlass(){
  document.querySelectorAll('[data-glass]').forEach(btn=>btn.addEventListener('click',()=>{
   prefs.glassStyle=btn.dataset.glass;applyGlass();savePrefs();
  }));applyGlass();
 }
 function initCozy(){
  const start=el('v42-start-toggle'),smart=el('v42-smart-toggle');
  start.checked=!!prefs.cozyStart;smart.checked=!!prefs.smartGaming;
  el('v42-cozy-welcome').hidden=!prefs.cozyStart;
  start.addEventListener('change',()=>{prefs.cozyStart=start.checked;el('v42-cozy-welcome').hidden=!start.checked;savePrefs()});
  smart.addEventListener('change',()=>{prefs.smartGaming=smart.checked;updateWelcomeAndWidgets();savePrefs()});
  updateWelcomeAndWidgets();setInterval(updateWelcomeAndWidgets,2200);
 }
 async function initialize(){makeSidebarGroups();initDashboard();try{
  const saved=await api('workspace');if(saved?.order?.length===cards.length)prefs=saved;
 }catch(err){notifyMsg('Arbeitsbereich konnte nicht geladen werden: '+err.message,true)}
 applyLayout();initDock();initVoice();initRecovery();initCozy();initGlass();
 }
 initialize();
})();
