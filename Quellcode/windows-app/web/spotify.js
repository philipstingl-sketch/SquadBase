'use strict';
// Real Spotify account information comes from the official Web API through the
// local Go backend. No fake playback data and no passwords in this UI.
let spotifyReady=false, spotifyConnected=false, spotifyNow=null, spotifyLoadingPlaylists=false,spotifyLastPlaylists=0,spotifyDesktopControls=false;
const spotifyAPI=(route,payload)=>api('spotify/'+route,payload);
const musicTime=ms=>{const x=Math.floor(Math.max(0,ms||0)/1000);return Math.floor(x/60)+':'+String(x%60).padStart(2,'0')};
function spotifyImage(el,src){if(src && /^https:\/\/(i\.scdn\.co|mosaic\.scdn\.co)\//.test(src)){el.src=src;el.hidden=false}else{el.hidden=true;el.removeAttribute('src')}}
function spotifyPlayer(now){spotifyNow=now;window.dispatchEvent(new CustomEvent("squad:spotify-now",{detail:{now}}));const active=!!(now&&now.active);$('spotify-song').textContent=active?now.title||'Unbekannter Titel':'Keine aktive Wiedergabe';$('spotify-artist').textContent=active?now.artists||'Spotify':'Öffne Spotify und starte einen Song, um ihn hier anzuzeigen.';$('spotify-device').textContent=active?(now.device||'Aktives Spotify-Gerät'):'SPOTIFY CONNECT';$('spotify-time-now').textContent=musicTime(now?.progress);$('spotify-time-end').textContent=musicTime(now?.duration);$('spotify-progress').style.width=active&&now.duration>0?Math.min(100,now.progress/now.duration*100)+'%':'0%';$('spotify-song-link').href=now?.url?.startsWith('https://open.spotify.com/')?now.url:'https://open.spotify.com/';spotifyImage($('spotify-cover'),now?.cover);$('spotify-art-placeholder').hidden=!!(active&&now?.cover);
  for(const id of ['spotify-play','spotify-prev','spotify-next'])$(id).disabled=!spotifyConnected&&!spotifyDesktopControls;
  $('spotify-play').textContent=!spotifyConnected?'⏯':now?.playing?'Ⅱ':'▶';$('spotify-play').title=now?.playing?'Pause':'Wiedergabe fortsetzen';$('spotify-mini-title').textContent=active?now.title||'Spotify':'Keine aktive Wiedergabe';$('spotify-mini-artist').textContent=active?now.artists||'':'Windows-Medientasten bereit · Songtitel nur mit Spotify API';$('spotify-mini-play').textContent=!spotifyConnected?'⏯':now?.playing?'Ⅱ':'▶';spotifyImage($('spotify-mini-art'),now?.cover);
 const seek=$('v41-spotify-seek'),volume=$('v41-spotify-volume');
 seek.disabled=!active||!spotifyConnected||!now.duration;seek.value=active&&now.duration?Math.round(now.progress/now.duration*1000):0;
 $('v41-mini-current').textContent=musicTime(now?.progress);$('v41-mini-total').textContent=musicTime(now?.duration);
 volume.disabled=!spotifyConnected||!active;
 if(Number.isFinite(now?.volume)&&document.activeElement!==volume)volume.value=now.volume;
 $('spotify-mini').dataset.dock=document.getElementById('v41-spotify-dock-select')?.value||'bar';
}
function spotifyStateUI(st){spotifyConnected=!!st.connected;spotifyDesktopControls=!!st.desktopControls;spotifyReady=true;
 const setup=$('spotify-help-status');if(setup)setup.textContent=st.callbackError || 'Spotify ist vorbereitet. Klicke auf „Mit Spotify anmelden“ und erlaube den Zugriff auf dein eigenes Konto.';
 for(const id of ['spotify-desktop-prev','spotify-desktop-toggle','spotify-desktop-next']){const b=$(id);if(b)b.disabled=!spotifyDesktopControls;}
$('spotify-connected').hidden=!spotifyConnected;$('spotify-disconnected').hidden=spotifyConnected;$('spotify-connection-label').textContent=spotifyConnected?'Mit Spotify verbunden':'Noch nicht mit Spotify verbunden';$('spotify-mini').hidden=(!spotifyConnected&&!spotifyDesktopControls)||(document.getElementById('v41-spotify-dock-select')?.value==='hidden');if(!spotifyConnected){spotifyPlayer(null);$('spotify-playlists-status').textContent='Automatische Playlists brauchen eine Spotify-Anmeldung. Gespeicherte Spotify-Links kannst du trotzdem öffnen.'}spotifyRenderBookmarks(st.bookmarks||[])}
async function spotifyLoadState(){try{spotifyStateUI(await spotifyAPI('status'));if(spotifyConnected)await spotifyRefreshNow();return true}catch(e){notify('Spotify-Status: '+e.message,true);return false}}
async function spotifyRefreshNow(){if(!spotifyConnected)return;try{const x=await spotifyAPI('now');spotifyPlayer(x);$('spotify-connection-label').textContent='Spotify-API verbunden'}catch(e){spotifyPlayer(null);$('spotify-connection-label').textContent='Spotify-Zugriff nicht möglich – bitte Zugang überprüfen';$('spotify-artist').textContent=e.message;$('spotify-desktop-status').textContent='Die Spotify-API meldet: '+e.message+' · Du kannst Spotify weiter über die Windows-Medientasten bedienen.'}}
function spotifyPlaylistCard(p,bookmark=false){const el=document.createElement('div');el.className='spotify-playlist';const pic=document.createElement('div');pic.className='playlist-placeholder';pic.textContent='♫';el.appendChild(pic);if(p.cover){const image=document.createElement('img');image.alt='Playlist-Cover';spotifyImage(image,p.cover);if(!image.hidden){pic.replaceWith(image)}}const info=document.createElement('div');info.className='playlist-info';const title=document.createElement('strong');title.textContent=p.name;title.title=p.name;info.appendChild(title);const count=document.createElement('small');count.textContent=bookmark?'Gespeicherte Playlist':typeof p.total==='number'?`${p.total} Titel · Spotify`:'Spotify-Playlist';info.appendChild(count);const buttons=document.createElement('div');buttons.className='playlist-buttons';const open=document.createElement('button');open.textContent='↗ Öffnen';open.addEventListener('click',()=>spotifyOpenPlaylist(p.id));buttons.appendChild(open);if(spotifyConnected){const browse=document.createElement('button');browse.textContent='▦ Songs ansehen';browse.className='spotify-browse';browse.addEventListener('click',()=>window.spotifySelectPlaylist?.(p));buttons.appendChild(browse);const play=document.createElement('button');play.textContent='▶ Abspielen';play.title='Premium-Abo und aktives Spotify-Gerät nötig';play.addEventListener('click',()=>spotifyControl('playlist',p.id));buttons.appendChild(play)}if(bookmark){const rm=document.createElement('button');rm.textContent='✕ Entfernen';rm.addEventListener('click',async()=>{try{const s=await spotifyAPI('bookmarks',{action:'remove',id:p.id});spotifyStateUI(s)}catch(e){notify(e.message,true)}});buttons.appendChild(rm)}info.appendChild(buttons);el.appendChild(info);return el}
function spotifyRenderBookmarks(list){const root=$('spotify-bookmarks');root.replaceChildren();if(!list.length){const p=document.createElement('p');p.className='hint';p.textContent='Noch keine Playlist gespeichert. Oben einen Spotify-Link hinzufügen.';root.appendChild(p);return}for(const p of list){root.appendChild(spotifyPlaylistCard(p,true))}}
async function spotifyLoadPlaylists(force=false){if(!spotifyConnected||spotifyLoadingPlaylists)return;if(!force&&Date.now()-spotifyLastPlaylists<90_000)return;spotifyLoadingPlaylists=true;$('spotify-playlists-status').textContent='Lade deine echten Spotify-Playlists …';try{const x=await spotifyAPI('playlists');spotifyLastPlaylists=Date.now();const root=$('spotify-playlists');root.replaceChildren();const list=x.playlists||[];$('spotify-playlists-status').textContent=list.length?`${list.length} Playlists aus deinem Spotify-Konto geladen.`:'Spotify hat keine verfügbaren Playlists zurückgegeben.';if(!list.length){const p=document.createElement('p');p.className='hint';p.textContent='Du kannst unten Playlist-Links manuell speichern.';root.appendChild(p)}for(const p of list)root.appendChild(spotifyPlaylistCard(p))}catch(e){$('spotify-playlists-status').textContent=e.message;notify(e.message,true)}finally{spotifyLoadingPlaylists=false}}
async function spotifyDesktop(action){
 const target=$('spotify-desktop-status');try{const result=await spotifyAPI('desktop',{action});if(target)target.textContent=result.message||'Windows-Medienbefehl gesendet.';if(action!=='open')notify('Medienbefehl gesendet. Spotify muss auf deinem PC laufen.');}
 catch(e){if(target)target.textContent=e.message;notify(e.message,true)}
}
async function spotifyControl(action,id,value){
 if(!spotifyConnected && spotifyDesktopControls){
   if(action==='play'||action==='pause')return spotifyDesktop('toggle');
   if(action==='next'||action==='previous')return spotifyDesktop(action);
   return notify('Für diese Funktion musst du dein Spotify-Konto über die Developer-API verbinden.',true);
 }
 try{await spotifyAPI('control',{action,id:id||'',value:Number.isFinite(value)?value:0});await new Promise(r=>setTimeout(r,650));await spotifyRefreshNow()}catch(e){notify(e.message,true);const t=$('spotify-desktop-status');if(t)t.textContent='Spotify-API: '+e.message+' · Die Windows-Medientasten oben können trotzdem funktionieren.'}
}
async function spotifyOpenPlaylist(id){try{await spotifyAPI('open',{id})}catch(e){notify(e.message,true)}}
async function spotifyOpenTab(){if(!spotifyReady)await spotifyLoadState();if(spotifyConnected){await Promise.all([spotifyRefreshNow(),spotifyLoadPlaylists()])}}
for(const [id,action] of [['spotify-desktop-open','open'],['spotify-desktop-toggle','toggle'],['spotify-desktop-prev','previous'],['spotify-desktop-next','next']]){
 $(id).addEventListener('click',()=>spotifyDesktop(action));
}
$('spotify-login').addEventListener('click',async()=>{try{await spotifyAPI('start',{});notify('Spotify-Anmeldung wurde im Browser geöffnet.')}catch(e){notify(e.message,true)}});
$('spotify-logout').addEventListener('click',async()=>{try{await spotifyAPI('logout',{});spotifyLastPlaylists=0;$('spotify-playlists').replaceChildren();$('spotify-library-browser').hidden=true;await spotifyLoadState();notify('Spotify-Verbindung beendet.')}catch(e){notify(e.message,true)}});
$('spotify-open').addEventListener('click',()=>spotifyOpenPlaylist(''));
$('spotify-refresh').addEventListener('click',()=>spotifyRefreshNow());
$('spotify-playlists-refresh').addEventListener('click',()=>spotifyLoadPlaylists(true));
$('spotify-play').addEventListener('click',()=>spotifyControl(spotifyNow?.playing?'pause':'play'));
$('spotify-prev').addEventListener('click',()=>spotifyControl('previous'));
$('spotify-next').addEventListener('click',()=>spotifyControl('next'));
$('spotify-mini-play').addEventListener('click',()=>spotifyControl(spotifyNow?.playing?'pause':'play'));
$('spotify-mini-prev').addEventListener('click',()=>spotifyControl('previous'));
$('spotify-mini-next').addEventListener('click',()=>spotifyControl('next'));
$('spotify-mini-more').addEventListener('click',()=>tab('spotify'));
$('v41-spotify-seek')?.addEventListener('change',e=>{if(!spotifyNow?.active||!spotifyNow.duration)return;const ms=Math.round(Number(e.target.value)/1000*spotifyNow.duration);spotifyControl('seek','',ms)});
$('v41-spotify-volume')?.addEventListener('change',e=>spotifyControl('volume','',Number(e.target.value)));

$('spotify-mini-toggle').addEventListener('click',()=>{const p=$('spotify-mini');p.classList.toggle('collapsed');$('spotify-mini-toggle').textContent=p.classList.contains('collapsed')?'+':'−'});
$('spotify-bookmark-add').addEventListener('click',async()=>{try{const st=await spotifyAPI('bookmarks',{action:'add',url:$('spotify-bookmark-url').value.trim(),name:$('spotify-bookmark-name').value.trim()});$('spotify-bookmark-url').value='';$('spotify-bookmark-name').value='';spotifyStateUI(st);notify('Playlist-Link gespeichert.')}catch(e){notify(e.message,true)}});
// A lightweight account status check also notices when the browser OAuth flow finishes.
spotifyLoadState();
setInterval(async()=>{if(!spotifyReady)return; if(document.querySelector('#spotify.active')&&!spotifyConnected){await spotifyLoadState();if(spotifyConnected){await spotifyRefreshNow();await spotifyLoadPlaylists()}}else if(spotifyConnected){await spotifyRefreshNow()}},12000);
