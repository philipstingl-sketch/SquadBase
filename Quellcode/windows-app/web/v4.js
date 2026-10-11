'use strict';
// SQUAD BASE 4.0: native-local features; no analytics or data transfers to SQUAD BASE.
(() => {
 const get=id=>document.getElementById(id);
 const request=async (path,payload,method='POST')=>{
  const opts=payload===undefined?{}:{method,headers:{'Content-Type':'application/json','X-Squad-Base':'1'},body:JSON.stringify(payload)};
  const res=await fetch('api/'+path,opts);let data={};try{data=await res.json()}catch{}
  if(!res.ok)throw Error(data.error||('HTTP '+res.status));return data;
 };
 const status=(id,msg,error=false)=>{const e=get(id);if(!e)return;e.textContent=msg;e.classList.toggle('v4-error',error)};
 const busy=async(btn,fn)=>{if(btn.disabled)return;btn.disabled=true;try{await fn()}catch(e){window.notify?.(e.message,true)}finally{btn.disabled=false}};

 // GAME LIBRARY. Names and paths are always rendered as text, never innerHTML.
 let lib=[],libraryLoaded=false,libraryActivity={};
 const sourceMark={Steam:'S',Epic:'E','Eigene EXE':'+'};
 async function loadLibrary(refresh=false){
  status('library-status','Suche nach installierten Spielen …');
  const d=refresh?await request('library',{action:'refresh'}):await request('library');
  lib=Array.isArray(d.games)?d.games:[];libraryLoaded=true;
  try{const stats=await request('library/pro');libraryActivity=stats.activity||{}}catch{libraryActivity={}}
  paintLibrary();
  status('library-status',lib.length+' lokale Spiele erkannt · kein Daten-Upload');
 }
 const add=(parent,tag,cls,text)=>{const e=document.createElement(tag);if(cls)e.className=cls;if(text!==undefined)e.textContent=text;parent.append(e);return e};
 function paintLibrary(){const grid=get('library-list');grid.replaceChildren();get('library-count').textContent=lib.length;
  const text=get('library-search').value.trim().toLocaleLowerCase('de-DE'),filter=get('library-filter').value;
  const filtered=lib.filter(g=>(filter==='all'||(filter==='favorites'?g.favorite:g.source===filter))&&g.name.toLocaleLowerCase('de-DE').includes(text));
  const sort=get('v43-library-sort')?.value||'name';
  if(sort==='time')filtered.sort((a,b)=>(libraryActivity[b.id]?.seconds||0)-(libraryActivity[a.id]?.seconds||0)||a.name.localeCompare(b.name));
  else if(sort==='recent')filtered.sort((a,b)=>String(libraryActivity[b.id]?.lastLaunched||'').localeCompare(String(libraryActivity[a.id]?.lastLaunched||''))||a.name.localeCompare(b.name));
  else if(sort==='favorites')filtered.sort((a,b)=>Number(b.favorite)-Number(a.favorite)||a.name.localeCompare(b.name));
  else filtered.sort((a,b)=>a.name.localeCompare(b.name));
  if(!filtered.length){add(grid,'p','hint',lib.length?'Keine Spiele mit diesem Filter gefunden.':'Noch keine Steam-/Epic-Spiele gefunden. Du kannst deine EXE manuell hinzufügen.');return}
  for(const g of filtered){const card=add(grid,'article','v4-library-card');
   const h=add(card,'div','v4-game-icon',sourceMark[g.source]||'▣');h.setAttribute('aria-hidden','true');
   const detail=add(card,'div','v4-game-details');add(detail,'strong','',g.name);add(detail,'small','',g.source+(g.installed?' · Installiert':' · Nicht gefunden'));
   const act=libraryActivity[g.id]||{};
   const duration=g.source==='Steam'?'Spielzeit nicht ermittelbar':('Erfasst: '+(Math.floor((act.seconds||0)/60))+' Min.');
   let recent='';if(act.lastLaunched){const d=new Date(act.lastLaunched);if(!Number.isNaN(d.getTime()))recent=' · Start: '+d.toLocaleDateString('de-DE')}
   add(detail,'small','v43-library-stats',duration+recent);
   if(g.source==='Steam'&&/^\d{1,15}$/.test(g.appId||'')){const image=document.createElement('img');image.className='v43-game-cover';image.loading='lazy';image.alt='';image.src='https://cdn.akamai.steamstatic.com/steam/apps/'+g.appId+'/header.jpg';image.addEventListener('error',()=>image.remove());detail.prepend(image)}
   const buttons=add(card,'div','v4-game-buttons');
   const fav=add(buttons,'button','v4-fav',g.favorite?'★':'☆');fav.type='button';fav.title=g.favorite?'Favorit entfernen':'Als Favorit markieren';fav.setAttribute('aria-label',fav.title);fav.dataset.libFavorite=g.id;
   const launch=add(buttons,'button','primary','▶ Start');launch.type='button';launch.disabled=!g.installed;launch.dataset.libLaunch=g.id;
   if(g.source==='Eigene EXE'){const del=add(buttons,'button','outline','×');del.type='button';del.title='Aus Bibliothek entfernen';del.setAttribute('aria-label',del.title);del.dataset.libRemove=g.id}
  }
 }
 get('library-search')?.addEventListener('input',paintLibrary);
 get('library-filter')?.addEventListener('change',paintLibrary);
 get('v43-library-sort')?.addEventListener('change',paintLibrary);
 get('library-refresh')?.addEventListener('click',e=>busy(e.currentTarget,()=>loadLibrary(true)));
 get('library-add')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{await request('library',{action:'add'});await loadLibrary(true)}));
 get('library-list')?.addEventListener('click',async e=>{const btn=e.target.closest('button');if(!btn)return;
  const key=btn.dataset.libFavorite?'favorite':btn.dataset.libLaunch?'launch':btn.dataset.libRemove?'remove':'';
  const id=btn.dataset.libFavorite||btn.dataset.libLaunch||btn.dataset.libRemove;if(!key)return;
  await busy(btn,async()=>{if(key==='remove'&&!confirm('Dieses selbst hinzugefügte Spiel wirklich aus der Bibliothek entfernen?'))return;
   const payload={action:key,id};if(key==='favorite')payload.favorite=!(lib.find(g=>g.id===id)?.favorite);
   await request('library',payload);if(key==='launch')window.notify?.('Spielstart angefordert.');else await loadLibrary(true);
  });
 });

 // THEME STUDIO. All changes are local. Image via native Windows picker.
 const studioKeys=['blur','radius','tint','outline'];let studio={blur:18,radius:18,tint:55,outline:18},hasImage=false;
 const studioDefault={blur:18,radius:18,tint:55,outline:18};
 function applyStudio(){const r=document.documentElement;const st=studio;
  r.style.setProperty('--v4-glass-blur',st.blur+'px');r.style.setProperty('--v4-card-radius',st.radius+'px');
  r.style.setProperty('--v4-glass-tint',(st.tint/100).toFixed(2));r.style.setProperty('--v4-glass-stroke',(st.outline/100).toFixed(2));
  studioKeys.forEach(k=>{const i=get('v4-'+k),o=get('v4-'+k+'-val');if(i)i.value=String(st[k]);if(o)o.textContent=st[k]+(k==='blur'||k==='radius'?' px':'%')});
 }
 function wallpaperCSS(){document.body.classList.toggle('v4-wallpaper-on',hasImage);if(hasImage)document.documentElement.style.setProperty('--v4-wallpaper-url',`url("api/studio/wallpaper?v=${Date.now()}")`)}
 async function loadStudio(){try{const data=await request('studio');for(const k of studioKeys)if(Number.isFinite(data[k]))studio[k]=data[k];hasImage=!!data.customImage;applyStudio();wallpaperCSS()}catch(e){status('v4-studio-status','⚠ '+e.message,true)}}
 for(const k of studioKeys)get('v4-'+k)?.addEventListener('input',e=>{studio[k]=Number(e.target.value);applyStudio()});
 get('v4-studio-save')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{await request('studio',{action:'save',...studio});status('v4-studio-status','✓ Design dauerhaft gespeichert.')}));
 get('v4-studio-reset')?.addEventListener('click',()=>{studio={...studioDefault};applyStudio();status('v4-studio-status','Standardwerte als Vorschau. Zum Übernehmen speichern.')});
 get('v4-wallpaper-choose')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{const d=await request('studio',{action:'choose-wallpaper'});if(d.cancelled)return;hasImage=true;wallpaperCSS();status('v4-studio-status','✓ Bild auf diesem PC gespeichert.')}));
 get('v4-wallpaper-clear')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{await request('studio',{action:'clear-wallpaper'});hasImage=false;wallpaperCSS();status('v4-studio-status','Eigenes Hintergrundbild entfernt.')}));

 // RELEASE CHECK AND USER-CONFIRMED APP UPDATE.
 function releaseNotes(d){const wrap=get('v4-release-notes-wrap'),body=get('v4-release-notes');if(!wrap||!body)return;const notes=typeof d.latestNotes==='string'?d.latestNotes:(typeof d.notes==='string'?d.notes:'');if(!notes.trim()){wrap.hidden=true;body.textContent='';return}const published=d.latestPublished||d.published;let stamp='';if(published){const when=new Date(published);if(Number.isFinite(when.getTime()))stamp='Veröffentlicht: '+when.toLocaleDateString('de-DE')+'\n\n'}body.textContent=stamp+notes;wrap.hidden=false}
 let latestAvailableVersion='',lastNotifiedUpdate='';
 function renderReleaseInfo(d,notifyUpdate=false){
  const install=get('v4-update-install'),open=get('v4-update-open');
  get('v4-version').textContent='v'+(d.currentVersion||get('v4-version').textContent.replace(/^v/, '')||'–');
  releaseNotes(d);
  if(d.available){latestAvailableVersion=d.version||'';status('v4-update-status',d.canInstall?'Neue Version '+d.version+' verfügbar. Die EXE besitzt eine passende SHA-256-Prüfsumme und kann nach deiner Bestätigung installiert werden.':'Neue Version '+d.version+' verfügbar, aber es fehlt eine passende EXE mit verifizierbarer SHA-256-Prüfsumme. Bitte Release manuell prüfen.');if(install)install.hidden=!d.canInstall;if(open)open.hidden=!d.url;if(notifyUpdate&&lastNotifiedUpdate!==d.version){window.notify?.('SquadBase '+d.version+' ist verfügbar. Öffne „Updates & Diagnose“, um das Update zu installieren.');lastNotifiedUpdate=d.version}}
  else{latestAvailableVersion='';lastNotifiedUpdate='';status('v4-update-status',d.message||(d.version?'Du nutzt bereits die neueste veröffentlichte Version ('+d.version+').':'Kein Update-Ergebnis verfügbar.'));if(install)install.hidden=true;if(open)open.hidden=!d.url}
 }
 async function checkAppRelease(notifyUpdate=false){status('v4-update-status','Prüfe die GitHub-Veröffentlichung …');const d=await request('maintenance',{action:'check'});renderReleaseInfo(d,notifyUpdate);return d}
 async function loadUpdate(checkIfDue=false){try{const d=await request('maintenance');get('v4-repo').value=d.repository||'';get('v4-auto-check').checked=!!d.autoCheck;get('v4-version').textContent='v'+d.version;releaseNotes(d);
  if(d.available)renderReleaseInfo({available:true,canInstall:!!d.canInstall,currentVersion:d.version,version:d.latest,url:'https://github.com/'+d.repository+'/releases/latest',latestNotes:d.latestNotes,latestPublished:d.latestPublished,message:'Update bereits erkannt.'},checkIfDue&&d.autoCheck);
  else status('v4-update-status',d.repository?'Repository gespeichert. Letzte Prüfung: '+(d.lastCheck?new Date(d.lastCheck).toLocaleString('de-DE'):'noch keine')+'.':'Kein GitHub-Repository eingerichtet.');
  if(checkIfDue&&d.autoCheck&&d.repository){const last=d.lastCheck?Date.parse(d.lastCheck):0;if(!last||Date.now()-last>=12*60*60*1000)await checkAppRelease(true)}
 }catch(e){status('v4-update-status',e.message,true)}}
 get('v453-preupdate-backup')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{const d=await request('diagnostics',{action:'backup'});status('v4-update-status','✓ Lokales Einstellungs-Backup erstellt.');window.notify?.(d.message||'Einstellungen gesichert.')}));
 get('v4-update-save')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{await request('maintenance',{action:'configure',repository:get('v4-repo').value.trim(),autoCheck:get('v4-auto-check').checked});const shouldCheck=get('v4-auto-check').checked&&get('v4-repo').value.trim()!=='';status('v4-update-status','✓ Repository gespeichert.');get('v4-update-open').hidden=true;get('v4-update-install').hidden=true;await loadUpdate(false);if(shouldCheck)await checkAppRelease(false)}));
 get('v4-update-check')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{await checkAppRelease(false)}));
 get('v4-update-open')?.addEventListener('click',e=>busy(e.currentTarget,()=>request('maintenance',{action:'open'})));
 get('v4-update-install')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{if(!window.confirm('SquadBase '+(latestAvailableVersion||'aktualisieren')+' installieren? Deine Einstellungen werden vorher lokal gesichert. Die App schließt sich kurz und startet anschließend neu.'))return;status('v4-update-status','Sichere Einstellungen und lade die geprüfte EXE herunter …');await request('diagnostics',{action:'backup'});const d=await request('maintenance',{action:'install'});status('v4-update-status',d.message||'Update wird vorbereitet.');window.notify?.('Update wird installiert. SquadBase startet anschließend neu.')}));
 setTimeout(()=>loadUpdate(true),6500);setInterval(()=>loadUpdate(true),15*60*1000);
 // DIAGNOSTICS & LOCAL BACKUPS.
 async function showDiagnostics(){const d=await request('diagnostics');const parent=get('v4-diagnostics');parent.replaceChildren();
  for(const [label,val] of [['Version',d.version],['Windows-Build',d.system],['GPU-Messung',d.gpuDataAvailable?'Verfügbar':'Nicht verfügbar'],['CPU-Temperatur',d.cpuTemperatureAvailable?'Verfügbar':'Nicht verfügbar'],['GPU-Temperatur',d.gpuTemperatureAvailable?'Verfügbar':'Nicht verfügbar'],['Spielpfade eingerichtet',d.configuredLauncherCount],['Eigene Spiele',d.userAddedGameCount],['Overlay',d.overlayEnabled?'Konfiguriert':'Deaktiviert'],['Signierung','Nicht vorhanden']]){
   const line=add(parent,'div','v4-diag-line');add(line,'span','',label);add(line,'strong','',String(val??'–'));
  }
 }
 get('v4-diag-refresh')?.addEventListener('click',e=>busy(e.currentTarget,showDiagnostics));
 for(const [id,action] of [['v4-diag-save','report'],['v4-config-backup','backup'],['v4-config-folder','folder']]){
  get(id)?.addEventListener('click',e=>busy(e.currentTarget,async()=>{const d=await request('diagnostics',{action});window.notify?.(d.message||'Ordner geöffnet')}));
 }

 // Assistant 2.0 – user-controlled local saved chat. No friend data involved.
 const assistantLog=get('assistant-log'),savedCheck=get('v4-assistant-history');let writingHistory=false,historyTimer=0;
 function recentMessages(){return Array.from(assistantLog.querySelectorAll('.assistant-bubble.user,.assistant-bubble.bot')).map(e=>({role:e.classList.contains('user')?'user':'assistant',content:(e.querySelector('p')?.textContent||'').slice(0,1900)})).filter(m=>m.content.trim()).slice(-40)}
 async function saveConversation(){if(!savedCheck.checked)return;try{await request('assistant/history',{messages:recentMessages()})}catch(e){window.notify?.('Chat konnte nicht gespeichert werden: '+e.message,true)}}
 async function loadConversation(){try{const data=await request('assistant/history');if(!data.messages?.length)return;
  writingHistory=true;window.SquadAssistantHistory?.load(data.messages);writingHistory=false;
  status('assistant-status','✓ Lokaler Chatverlauf wiederhergestellt.');
 }catch(e){writingHistory=false;window.notify?.(e.message,true)}}
 get('v4-assistant-save')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{await request('assistant/history',{messages:recentMessages()});window.notify?.('Chat lokal gespeichert.')}));
 savedCheck?.addEventListener('change',()=>{if(savedCheck.checked){loadConversation();saveConversation()}else{status('assistant-status','Automatische Chat-Speicherung ausgeschaltet.')}});
 const obs=new MutationObserver(()=>{if(writingHistory||!savedCheck.checked)return;clearTimeout(historyTimer);historyTimer=setTimeout(saveConversation,1100)});
 if(assistantLog)obs.observe(assistantLog,{childList:true,subtree:true,characterData:true});
 get('assistant-clear')?.addEventListener('click',()=>{clearTimeout(historyTimer);request('assistant/history',{},'DELETE').catch(()=>{})});
 get('v4-assistant-pc')?.addEventListener('click',e=>busy(e.currentTarget,async()=>{const s=await request('state');const h=s.hardware;const m=x=>typeof x==='number'?x.toFixed(0)+'%':'nicht verfügbar';get('v4-assistant-pc-info').textContent='CPU '+m(h.cpu)+' · GPU '+m(h.gpu)+' · RAM '+m(h.ram)+' · Kein Zugriff auf private Nachrichten.'}));

 document.addEventListener('click',e=>{const tab=e.target.closest('[data-tab],[data-goto]');if(!tab)return;const target=tab.dataset.tab||tab.dataset.goto;
  if(target==='library'&&!libraryLoaded)loadLibrary().catch(err=>status('library-status',err.message,true));
  if(target==='maintenance'){loadUpdate();showDiagnostics().catch(err=>window.notify?.(err.message,true))}
 });
 loadStudio();
})();
