'use strict';
(() => {
 const $=id=>document.getElementById(id);
 let config=null, spotify=null;
 const fetchJSON=async (path, payload)=>{
  const res=await fetch('api/'+path,{method:payload?'POST':'GET',headers:payload?{'Content-Type':'application/json','X-Squad-Base':'1'}:{},body:payload?JSON.stringify(payload):undefined,cache:'no-store'});
  if(!res.ok)throw Error('Nicht verfügbar');return res.json();
 };
 const fmt=v=>typeof v==='number'&&Number.isFinite(v)?String(Math.round(v)):'–';
 async function loadCfg(){try{
  const d=await fetchJSON('overlay');config=d.config;
  for(const name of ['cpu','gpu','ram','gpuTemp','fps','ping'])$('metric-'+name).hidden=!config[name];
  $('music').hidden=!config.music;$('friends').hidden=!config.friends;$('clock').hidden=!config.clock;
  document.body.classList.toggle('compact',!!config.compact);document.body.classList.toggle('minimal-header',!!config.minimalHeader);
  $('metrics').hidden=!['cpu','gpu','ram','gpuTemp','fps','ping'].some(k=>config[k]);
  try {const a=await fetchJSON('appearance');const colors={oak:'#99efbe',violet:'#c3acff',ocean:'#73ddf2',sunset:'#ffb67a',rose:'#ff97b7',ice:'#a6cbff'};
       document.documentElement.style.setProperty('--m',colors[a.color]||colors.oak); document.body.dataset.cozy=a.cozy||'cozy'}catch{}
 }catch{/* Haupt-App noch nicht verfügbar */}}
 async function update(){
  if(!config)await loadCfg();if(!config)return;
  if(config.clock)$('clock').textContent=new Date().toLocaleTimeString('de-DE',{hour:'2-digit',minute:'2-digit'});
  try{
   const s=await fetchJSON('state'),h=s.hardware;
   $('o-cpu').textContent=fmt(h.cpu)+'%';$('o-gpu').textContent=h.gpu===null?'–':fmt(h.gpu)+'%';$('o-ram').textContent=fmt(h.ram)+'%';
   $('o-temp').textContent=h.gpuTemp===null?'–':fmt(h.gpuTemp)+'°';
   $('o-fps').textContent=h.fps===null?'n/v':fmt(h.fps);
   $('o-ping').textContent=s.ping?.average===null||!s.ping?.average?'–':fmt(s.ping.average);
  }catch{$('o-cpu').textContent='Offline'}
 }
 async function updateMusic(){
  if(!config?.music)return;
  try{
   const st=await fetchJSON('spotify/status');if(!st.connected){$('song-title').textContent='Spotify nicht verbunden';return}
   spotify=await fetchJSON('spotify/now');
   $('song-title').textContent=spotify.title||'Kein Song';$('song-sub').textContent=spotify.artists||'Spotify';$('song-play').textContent=spotify.playing?'Ⅱ':'▶';
  }catch{spotify=null;$('song-title').textContent='Spotify nicht verfügbar'}
 }
 async function musicAction(action){try{await fetchJSON('spotify/control',{action});setTimeout(updateMusic,800)}catch{$('song-sub').textContent='Premium / aktives Gerät benötigt'}}
 $('song-play').onclick=()=>musicAction(spotify?.playing?'pause':'play');$('song-next').onclick=()=>musicAction('next');$('song-back').onclick=()=>musicAction('previous');
 $('friends-open').onclick=()=>window.open('./','_blank','noopener');
 window.addEventListener('focus',loadCfg);
 loadCfg().then(()=>{update();updateMusic()});setInterval(update,2500);setInterval(updateMusic,12000);setInterval(loadCfg,15000);
})();
