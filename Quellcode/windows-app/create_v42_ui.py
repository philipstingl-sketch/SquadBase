from pathlib import Path
p=Path(__file__).parent
s=(p/'web/v41.js').read_text()
s=s.replace('SQUAD BASE 4.1', 'SQUAD BASE 4.2')
s=s.replace("const cards=['system','monitor','games','network','news'];", "const cards=['system','monitor','games','network','news','clockwidget','musicwidget','friendswidget','sessionwidget'];")
s=s.replace("news:'Spiele-News'};", "news:'Spiele-News',clockwidget:'Cozy Uhr',musicwidget:'Spotify Widget',friendswidget:'Freunde & Chat',sessionwidget:'Gaming Session'};")
start=s.index(' const presets={')
end=s.index(' let prefs=',start)
s=s[:start]+''' const allWidgets=['clockwidget','musicwidget','friendswidget','sessionwidget'];
 const presets={
  gaming:{order:['system','monitor','games','network','news',...allWidgets],hidden:[...allWidgets],wide:['news']},
  cozy:{order:['system','clockwidget','games','musicwidget','monitor','news','network','friendswidget','sessionwidget'],hidden:['network','friendswidget','sessionwidget'],wide:['news']},
  minimal:{order:['system','clockwidget','games','network','monitor','news','musicwidget','friendswidget','sessionwidget'],hidden:['network','monitor','news','musicwidget','friendswidget','sessionwidget'],wide:['system']},
  social:{order:['friendswidget','musicwidget','games','news','network','system','monitor','clockwidget','sessionwidget'],hidden:['monitor','clockwidget','sessionwidget'],wide:['news']}
 };
''' + s[end:]
s=s.replace("spotifyDock:'bar'};", "spotifyDock:'bar',version:2,cozyStart:true,smartGaming:true};",1)
s=s.replace('let editing=false, board, selectedDrag=null, saveQueue=Promise.resolve();','let editing=false, board, selectedDrag=null, saveQueue=Promise.resolve(), beforeEdit=null;')
# avoid remote changes while editing preferences? keep old save for dock/voice only.
needle='  const cardIds=[\'system\',\'monitor\',\'games\',\'network\',\'news\'];'
new='''  const widgetTemplates={
    clockwidget:`<div class="section-head"><div><h2>◷ Cozy Uhr</h2><p>Lokale Zeit und Datum</p></div><span class="tag">LIVE</span></div><div class="v42-clock-display" id="v42-clock-display">--:--</div><div class="v42-widget-detail" id="v42-clock-date">Datum wird geladen</div><button class="quiet" data-goto="settings">Uhr gestalten →</button>`,
    musicwidget:`<div class="section-head"><div><h2>♫ Spotify</h2><p>Deine Musik auf einen Blick</p></div><span class="tag">PLAYER</span></div><div class="v42-widget-main">♫ <strong id="v42-widget-track">Nicht verbunden</strong></div><div class="v42-widget-detail" id="v42-widget-artist">Melde dich in Spotify an, um deinen Song zu sehen.</div><button class="outline" type="button" data-goto="spotify">Spotify öffnen ↗</button>`,
    friendswidget:`<div class="section-head"><div><h2>☻ Deine Freunde</h2><p>Freunde & Nachrichten</p></div><span class="tag">PRIVAT</span></div><div class="v42-widget-main"><strong id="v42-widget-friends">Nicht verbunden</strong></div><div class="v42-widget-detail">Die Kontodaten bleiben im Freunde-System.</div><button class="outline" data-goto="friends" type="button">Zu Freunden & Chat →</button>`,
    sessionwidget:`<div class="section-head"><div><h2>⚡ Gaming Session</h2><p>Deine erkannten Spiele</p></div><span class="tag">LOKAL</span></div><div class="v42-widget-main"><strong id="v42-widget-session">Kein Spiel erkannt</strong></div><div class="v42-widget-detail" id="v42-widget-hours">Spielzeit wird geladen</div><button type="button" class="outline" data-goto="library">Spielebibliothek →</button>`
  };
  for(const id of allWidgets){const card=document.createElement('article');card.className='panel v42-widget';card.innerHTML=widgetTemplates[id];articles.push(card)}
  const cardIds=cards;'''
assert needle in s;s=s.replace(needle,new)
s=s.replace('applyLayout();savePrefs()});\n  });','applyLayout()});\n  });',1)  # Drop handler save
s=s.replace('  const buttons=[...section.querySelectorAll(\'button[data-goto]\')]; // continue to use original event delegation\n  void buttons;', '  // Existing global event delegation continues to handle widget shortcut buttons.')
start=s.index(' function applyLayout(){')
end=s.index(' function setDock(){',start)
s=s[:start]+''' function applyLayout(){
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
''' + s[end:]
# avoid all widget cards shown in edit if hidden - existing behavior helpful
# new welcome and smart update logic
mark=' async function initialize(){'
new=''' function updateWelcomeAndWidgets(){
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
 function initCozy(){
  const start=el('v42-start-toggle'),smart=el('v42-smart-toggle');
  start.checked=!!prefs.cozyStart;smart.checked=!!prefs.smartGaming;
  el('v42-cozy-welcome').hidden=!prefs.cozyStart;
  start.addEventListener('change',()=>{prefs.cozyStart=start.checked;el('v42-cozy-welcome').hidden=!start.checked;savePrefs()});
  smart.addEventListener('change',()=>{prefs.smartGaming=smart.checked;updateWelcomeAndWidgets();savePrefs()});
  updateWelcomeAndWidgets();setInterval(updateWelcomeAndWidgets,2200);
 }
'''
s=s.replace(mark,new+mark)
s=s.replace(' applyLayout();initDock();initVoice();initRecovery();',' applyLayout();initDock();initVoice();initRecovery();initCozy();')
# ensure widget drag no saves, fix any remainder
s=s.replace('prefs.preset=\'custom\';applyLayout();savePrefs()});','prefs.preset=\'custom\';applyLayout()});')
(p/'web/v42.js').write_text(s)
# themes actual animation pause, not config overwritten
th=(p/'web/themes.js').read_text().replace("const animate=()=>settings.motion&&!hasReducedMotion()&&settings.scene!=='oak'&&!document.hidden;", "const animate=()=>settings.motion&&!hasReducedMotion()&&!document.body.classList.contains('v42-smart-active')&&settings.scene!=='oak'&&!document.hidden;")
th=th.replace(" window.addEventListener('resize',resize,{passive:true});", " window.addEventListener('squad-smart-mode',apply);\n window.addEventListener('resize',resize,{passive:true});")
(p/'web/themes.js').write_text(th)
