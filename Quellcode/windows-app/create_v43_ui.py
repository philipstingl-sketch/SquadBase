from pathlib import Path
p=Path('web/index.html');s=p.read_text()
s=s.replace('SQUAD BASE · 4.2.3 · Spotify Library','SQUAD BASE · 4.3 · Music, Gaming & Cozy')
s=s.replace('v4.2.3</span>', 'v4.3</span>')
s=s.replace('<link rel="stylesheet" href="v423.css"></head>', '<link rel="stylesheet" href="v423.css"><link rel="stylesheet" href="v43.css"></head>')
s=s.replace('<script src="v42.js"></script>', '<script src="v42.js"></script><script src="v43.js"></script>')
s=s.replace('Intelligente Spielebibliothek</h1>','Spielebibliothek Pro</h1>')
s=s.replace('Eigene EXE hinzufügen, Favoriten setzen und direkt starten.', 'Eigene EXE hinzufügen, Favoriten setzen und direkt starten. Erfasste Spielzeit und letzte Starts bleiben nur lokal.')
s=s.replace('<select id="library-filter" aria-label="Quelle filtern">','<select id="v43-library-sort" aria-label="Spiele sortieren"><option value="name">Name A–Z</option><option value="recent">Zuletzt gestartet</option><option value="time">Erfasste Spielzeit</option><option value="favorites">Favoriten zuerst</option></select><select id="library-filter" aria-label="Quelle filtern">')
s=s.replace('Keine Cloud-Synchronisierung und keine übermittelten Spielpfade.', 'Keine Cloud-Synchronisierung und keine übermittelten Spielpfade. Spielzeit für eigene EXEs und Epic wird nur bei erkanntem Prozess während der App-Laufzeit gezählt; bei Steam ohne bekannte EXE wird nichts erfunden.')
spotify_panel='''
<article class="panel v43-music-center" id="v43-music-center"><div class="section-head"><div><h2>♫ Music Center 2.0</h2><p>Shuffle, Wiederholung und Spotify-Warteschlange – echte API-Daten, keine Simulation.</p></div><span class="tag">SPOTIFY CONNECT</span></div>
<div class="v43-music-actions"><button type="button" class="outline" id="v43-shuffle" aria-pressed="false">⇄ Shuffle aus</button><button type="button" class="outline" id="v43-repeat">↻ Repeat aus</button><button type="button" class="outline" id="v43-queue-refresh">↻ Warteschlange laden</button></div>
<div class="v43-queue-title"><strong>Als Nächstes</strong><span id="v43-queue-count">–</span></div>
<div class="v43-queue" id="v43-queue" aria-live="polite"><p class="hint">Verbinde Spotify und lade die Warteschlange.</p></div>
<p class="hint" id="v43-music-status" role="status">Manche Spotify-Apps dürfen Shuffle, Repeat oder die Warteschlange nicht über die Web API steuern. In diesem Fall erscheint hier die echte Fehlermeldung.</p></article>
'''
s=s.replace('<article class="panel spotify-playlists-panel">',spotify_panel+'<article class="panel spotify-playlists-panel">',1)
ai_panel='''<article class="panel v43-assistant-tools"><div class="section-head"><div><h2>✧ KI-Assistent 3.0 · PC-Check</h2><p>Systemzustand nur nach Klick lesen, ohne private Pfade, Passwörter oder Chats.</p></div></div><div class="v43-assistant-buttons"><button type="button" class="primary" id="v43-ai-check">◎ PC-Zustand prüfen</button><button type="button" class="outline" id="v43-ai-explain">✧ Diagnose als Frage übernehmen</button></div><div id="v43-ai-summary" class="v43-ai-summary" role="status">Noch keine Diagnose angefordert.</div><div class="v43-assistant-shortcuts"><button type="button" class="outline" data-v43-open="network">Internet testen</button><button type="button" class="outline" data-v43-open="library">Spielebibliothek prüfen</button><button type="button" class="outline" data-v43-open="maintenance">Fehlerdiagnose öffnen</button></div><p class="hint">Der Assistent führt keine Systembefehle automatisch aus. Wechsel zu einem Tool erfolgt nur durch deine Auswahl. Ollama bleibt optional und lokal.</p></article>'''
s=s.replace('<div class="assistant-layout">',ai_panel+'<div class="assistant-layout">',1)
theme_panel='''<article class="panel v43-theme-pro"><div class="section-head"><div><h2>☾ Cozy Theme Studio Pro</h2><p>Automatischer Tag-/Nachtstil und ruhige Animationen. Deine bisherigen Farben und Designs bleiben erhalten.</p></div><span class="tag">4.3</span></div><div class="v43-theme-grid"><label class="v42-setting-option"><input id="v43-auto-theme" type="checkbox"><span><strong>Tag-/Nachtwechsel</strong><small>Wechselt abends automatisch zum ausgewählten Nachtglasdesign</small></span></label><label class="v42-setting-option"><input id="v43-reduce-motion" type="checkbox"><span><strong>Bewegung reduzieren</strong><small>Beruhigt animierte Effekte, unabhängig vom Gaming-Modus</small></span></label><label>Abend beginnt um<select id="v43-night-from"></select></label><label>Morgen beginnt um<select id="v43-day-from"></select></label><label>Nachtglasdesign<select id="v43-night-style"><option value="midnight">Midnight</option><option value="warm">Warm Oak</option><option value="crystal">Crystal</option><option value="frosted">Frosted</option></select></label></div><div class="v43-actions"><button type="button" class="primary" id="v43-theme-save">Design-Automatik speichern</button><span class="hint" id="v43-theme-status" role="status">Lade Einstellungen …</span></div></article>'''
s=s.replace('<article class="panel v41-workspace-settings">',theme_panel+'<article class="panel v41-workspace-settings">',1)
s=s.replace('<p class="hint">Tastenkürzel: Strg + Umschalt + O.', '<p class="hint">Overlay 3.0: Im geöffneten Overlay die Kopfzeile bei ausgeschaltetem Klick-durchlassen ziehen, um es neu zu positionieren. Tastenkürzel: Strg + Umschalt + O.')
s=s.replace('<button type="button" class="outline" data-overlay-preset="social">☻ Social & Musik</button>','<button type="button" class="outline" data-overlay-preset="social">☻ Social & Musik</button><button type="button" class="outline" data-overlay-preset="music">♫ Musik-Fokus</button>')
p.write_text(s)
# Spotify playlist track enqueue button, link unchanged
p=Path('web/spotify-library.js');s=p.read_text()
s=s.replace('''const actions=document.createElement('div');actions.className='spotify-track-actions';actions.appendChild(play);''','''const actions=document.createElement('div');actions.className='spotify-track-actions';actions.appendChild(play);
   if(t.playable && /^[A-Za-z0-9]{22}$/.test(t.id||'')){
     const enqueue=document.createElement('button');enqueue.type='button';enqueue.className='spotify-track-queue';enqueue.textContent='＋ Queue';enqueue.title='Zur Spotify-Warteschlange hinzufügen';
     enqueue.addEventListener('click',async()=>{enqueue.disabled=true;try{await spotifyAPI('playback-options',{action:'enqueue',uri:'spotify:track:'+t.id});notify('Song zur Spotify-Warteschlange hinzugefügt.');window.dispatchEvent(new Event('squad:queue-changed'));}catch(e){notify(e.message,true)}finally{enqueue.disabled=false}});
     actions.appendChild(enqueue);
   }''')
p.write_text(s)
# extra Spotify status update hook : status refresh triggers library/pro status from v43 js via dispatch
p=Path('web/spotify.js');s=p.read_text().replace('function spotifyPlayer(now){spotifyNow=now;', 'function spotifyPlayer(now){spotifyNow=now;window.dispatchEvent(new CustomEvent("squad:spotify-now",{detail:{now}}));')
p.write_text(s)
# library pro sort and session metadata, load activity async but no fabricated historical playtime
p=Path('web/v4.js');s=p.read_text()
s=s.replace('let lib=[],libraryLoaded=false;','let lib=[],libraryLoaded=false,libraryActivity={};')
s=s.replace('''lib=Array.isArray(d.games)?d.games:[];libraryLoaded=true;paintLibrary();''','''lib=Array.isArray(d.games)?d.games:[];libraryLoaded=true;
  try{const stats=await request('library/pro');libraryActivity=stats.activity||{}}catch{libraryActivity={}}
  paintLibrary();''')
s=s.replace('''  if(!filtered.length){add(grid,''','''  const sort=get('v43-library-sort')?.value||'name';
  if(sort==='time')filtered.sort((a,b)=>(libraryActivity[b.id]?.seconds||0)-(libraryActivity[a.id]?.seconds||0)||a.name.localeCompare(b.name));
  else if(sort==='recent')filtered.sort((a,b)=>String(libraryActivity[b.id]?.lastLaunched||'').localeCompare(String(libraryActivity[a.id]?.lastLaunched||''))||a.name.localeCompare(b.name));
  else if(sort==='favorites')filtered.sort((a,b)=>Number(b.favorite)-Number(a.favorite)||a.name.localeCompare(b.name));
  else filtered.sort((a,b)=>a.name.localeCompare(b.name));
  if(!filtered.length){add(grid,''')
s=s.replace('''add(detail,'small','',g.source+(g.installed?' · Installiert':' · Nicht gefunden'));''','''add(detail,'small','',g.source+(g.installed?' · Installiert':' · Nicht gefunden'));
   const act=libraryActivity[g.id]||{};
   const duration=g.source==='Steam'?'Spielzeit nicht ermittelbar':('Erfasst: '+(Math.floor((act.seconds||0)/60))+' Min.');
   let recent='';if(act.lastLaunched){const d=new Date(act.lastLaunched);if(!Number.isNaN(d.getTime()))recent=' · Start: '+d.toLocaleDateString('de-DE')}
   add(detail,'small','v43-library-stats',duration+recent);
   if(g.source==='Steam'&&/^\\d{1,15}$/.test(g.appId||'')){const image=document.createElement('img');image.className='v43-game-cover';image.loading='lazy';image.alt='';image.src='https://cdn.akamai.steamstatic.com/steam/apps/'+g.appId+'/header.jpg';image.addEventListener('error',()=>image.remove());detail.prepend(image)}''')
s=s.replace(''' get('library-filter')?.addEventListener('change',paintLibrary);''',''' get('library-filter')?.addEventListener('change',paintLibrary);
 get('v43-library-sort')?.addEventListener('change',paintLibrary);''')
p.write_text(s)
# overlay new preset
p=Path('web/overlay-settings.js');s=p.read_text().replace("social:['music','friends','clock','cpu']", "social:['music','friends','clock','cpu'],music:['music','clock']")
s=s.replace("social:['music','friends','clock','cpu']", "social:['music','friends','clock','cpu'],music:['music','clock']")
p.write_text(s)
