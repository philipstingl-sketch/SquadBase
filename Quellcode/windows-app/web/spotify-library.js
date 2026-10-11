'use strict';
// Native SQUAD BASE Spotify playlist browser. Spotify audio stays in Spotify;
// only authenticated metadata and playback instructions use its Web API.
(() => {
 const get = id => document.getElementById(id);
 const browser = get('spotify-library-browser');
 const tracksRoot = get('spotify-track-list');
 const status = get('spotify-library-progress');
 const errorEl = get('spotify-library-error');
 const filterStatus = get('spotify-library-filter-status');
 const nextButton = get('spotify-library-next');
 const allButton = get('spotify-library-all');
 const search = get('spotify-library-search');
 const selected = { id:'',name:'',total:0, tracks:[], nextOffset:0, busy:false, generation:0, loadingAll:false };
 function toText(v) {return String(v ?? '');}
 function setError(message='') {errorEl.textContent=message;}
 function renderStatus(){
   const n=selected.tracks.length;
   status.textContent = selected.busy ? `Lade Songs … ${n} von ${selected.total||'?'} geladen` : `${n} von ${selected.total} Songs geladen`;
   nextButton.hidden=selected.nextOffset===null;
   allButton.hidden=selected.nextOffset===null;
   nextButton.disabled=selected.busy;
   allButton.disabled=selected.busy && !selected.loadingAll;
   allButton.textContent=selected.loadingAll?'Ladevorgang stoppen':'Alle Songs laden';
   browser.setAttribute('aria-busy',String(selected.busy));
 }
 function rowFor(t){
   const r=document.createElement('div');r.className='spotify-track-row';
   if(!t.playable)r.classList.add('spotify-unavailable');
   const n=document.createElement('span');n.className='spotify-track-num';n.textContent=String(t.position+1);
   const main=document.createElement('div');main.className='spotify-track-main';
   if(t.cover && /^https:\/\/i\.scdn\.co\//.test(t.cover)){
     const img=document.createElement('img'); img.src=t.cover; img.alt='';img.loading='lazy';main.appendChild(img);
   }
   const labels=document.createElement('div');labels.style.minWidth='0';
   const name=document.createElement('strong');name.textContent=toText(t.name||'Nicht verfügbar');name.title=name.textContent;
   const artists=document.createElement('small');artists.textContent=toText(t.artists||'Unbekannter Künstler'); artists.title=artists.textContent;
   labels.append(name,artists);main.appendChild(labels);
   const album=document.createElement('span');album.className='spotify-track-album';album.textContent=toText(t.album||'—');album.title=album.textContent;
   const duration=document.createElement('span');duration.className='spotify-track-duration';duration.textContent=typeof musicTime==='function'?musicTime(t.duration):'—';
   const playlistForRow=selected.id;
   const play=document.createElement('button');play.className='spotify-track-play';play.type='button';
   play.textContent=t.playable?'▶ Spielen':'Nicht verfügbar';play.disabled=!t.playable;
   play.title=t.playable?'Diesen Song an dieser Stelle der Playlist über Spotify starten':'Spotify kann diesen Eintrag nicht wiedergeben';
   play.setAttribute('aria-label',t.playable?`Titel ${t.name} in Spotify abspielen`:'Titel nicht verfügbar');
   if(t.playable)play.addEventListener('click',async()=>{
     if(play.disabled || selected.id!==playlistForRow)return;
     play.disabled=true;play.classList.add('spotify-pending');play.textContent='…';
     try{
       await spotifyAPI('control',{action:'playlist-track',id:playlistForRow,position:t.position});
       notify(`${t.name} wird in Spotify gestartet.`);
       await new Promise(res=>setTimeout(res,500));
       if(typeof spotifyRefreshNow==='function')await spotifyRefreshNow();
     }catch(e){notify(e.message,true);setError('Wiedergabe nicht möglich: '+e.message+' Öffne Spotify, starte einen Song und versuche es erneut.');}
     finally{play.disabled=false;play.classList.remove('spotify-pending');play.textContent='▶ Spielen';}
   });
   const actions=document.createElement('div');actions.className='spotify-track-actions';actions.appendChild(play);
   if(t.playable && /^[A-Za-z0-9]{22}$/.test(t.id||'')){
     const enqueue=document.createElement('button');enqueue.type='button';enqueue.className='spotify-track-queue';enqueue.textContent='＋ Queue';enqueue.title='Zur Spotify-Warteschlange hinzufügen';
     enqueue.addEventListener('click',async()=>{enqueue.disabled=true;try{await spotifyAPI('playback-options',{action:'enqueue',uri:'spotify:track:'+t.id});notify('Song zur Spotify-Warteschlange hinzugefügt.');window.dispatchEvent(new Event('squad:queue-changed'));}catch(e){notify(e.message,true)}finally{enqueue.disabled=false}});
     actions.appendChild(enqueue);
   }
   if(/^https:\/\/open\.spotify\.com\/(track|episode)\/[a-zA-Z0-9]{22}$/.test(t.url||'')){
     const link=document.createElement('a');link.href=t.url;link.target='_blank';link.rel='noopener noreferrer';link.className='spotify-track-external';link.textContent='↗';link.title='Titel direkt in Spotify öffnen';link.setAttribute('aria-label',`${t.name} in Spotify öffnen`);actions.appendChild(link);
   }
   r.append(n,main,album,duration,actions);return r;
 }
 function paint(){
   const query=search.value.trim().toLocaleLowerCase('de-DE');
   const filtered=query?selected.tracks.filter(t=>(`${t.name||''} ${t.artists||''} ${t.album||''}`).toLocaleLowerCase('de-DE').includes(query)):selected.tracks;
   const previousScroll=tracksRoot.scrollTop;
   tracksRoot.replaceChildren();
   if(!filtered.length){const div=document.createElement('div');div.className='spotify-track-empty';
     div.textContent=selected.busy?'Songs werden geladen …':query?'Keine Treffer unter den bisher geladenen Songs.':'Noch keine Songs geladen oder die Playlist ist leer.';
     tracksRoot.appendChild(div);
   }else{
     // Render in chunks to avoid one giant DOM repaint on huge playlists.
     const frag=document.createDocumentFragment();for(const t of filtered)frag.appendChild(rowFor(t));tracksRoot.appendChild(frag);
   }
   tracksRoot.scrollTop=previousScroll;
   filterStatus.textContent=query?`${filtered.length} Treffer in ${selected.tracks.length} geladenen Songs${selected.nextOffset!==null?' · Für vollständige Suche „Alle Songs laden“ wählen':''}`:'Original-Reihenfolge der Playlist';
   renderStatus();
 }
 async function loadNext(){
   if(selected.busy||selected.nextOffset===null||!selected.id)return false;
   const generation=selected.generation, id=selected.id, offset=selected.nextOffset;
   selected.busy=true;setError('');renderStatus();
   try{
     const page=await spotifyAPI(`playlist-items?id=${encodeURIComponent(id)}&offset=${offset}`);
     if(generation!==selected.generation)return false;
     if(page.id!==id || page.offset!==offset || !Array.isArray(page.tracks))throw Error('Spotify hat einen unerwarteten Playlist-Abschnitt geschickt.');
     selected.tracks.push(...page.tracks);selected.total=Math.max(0,Number(page.total)||0);
     selected.nextOffset=typeof page.nextOffset==='number'?page.nextOffset:null;
     if(selected.nextOffset===offset)selected.nextOffset=null;
     paint();return true;
   }catch(e){
     if(generation!==selected.generation)return false;
     setError(e.message);notify('Spotify-Playlist: '+e.message,true);
     paint();return false;
   }finally{if(generation===selected.generation){selected.busy=false;paint();}}
 }
 async function loadAll(){
   if(selected.loadingAll){selected.loadingAll=false;renderStatus();return;}
   if(selected.busy)return;
   selected.loadingAll=true;const generation=selected.generation;renderStatus();
   let pages=0;
   while(selected.loadingAll && generation===selected.generation && selected.nextOffset!==null && pages<200){
     const ok=await loadNext();if(!ok)break;pages++;
     // Avoid an aggressive batch of Spotify requests under development quotas.
     if(selected.nextOffset!==null)await new Promise(r=>setTimeout(r,350));
   }
   if(generation===selected.generation){selected.loadingAll=false;paint();}
 }
 async function selectPlaylist(p){
   if(!p?.id || !/^[a-zA-Z0-9]{22}$/.test(p.id))return notify('Ungültige Playlist-ID',true);
   selected.generation++;selected.id=p.id;selected.name=p.name||'Spotify-Playlist';selected.tracks=[];selected.total=typeof p.total==='number'?p.total:0;selected.nextOffset=0;selected.busy=false;selected.loadingAll=false;
   get('spotify-library-name').textContent=selected.name;
   const cover=get('spotify-library-cover');
   if(p.cover && /^https:\/\/i\.scdn\.co\//.test(p.cover)){cover.src=p.cover;cover.hidden=false;}else{cover.hidden=true;cover.removeAttribute('src');}
   search.value='';setError('');browser.hidden=false;paint();tracksRoot.scrollTop=0;
   browser.scrollIntoView({behavior:window.matchMedia('(prefers-reduced-motion: reduce)').matches?'instant':'smooth',block:'start'});
   await loadNext();
 }
 window.spotifySelectPlaylist=selectPlaylist;
 get('spotify-library-close').addEventListener('click',()=>{selected.generation++;selected.loadingAll=false;selected.busy=false;browser.hidden=true;});
 get('spotify-library-spotify').addEventListener('click',async()=>{if(!selected.id)return;await spotifyOpenPlaylist(selected.id);});
 nextButton.addEventListener('click',loadNext);
 allButton.addEventListener('click',loadAll);
 search.addEventListener('input',paint);
})();
