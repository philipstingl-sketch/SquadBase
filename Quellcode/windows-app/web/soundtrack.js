'use strict';
// Two ORIGINAL embedded soundtracks and small synthesized SFX. Music is OPT-IN.
// Spotify player and the soundtrack are independent; this player never reads Spotify streams.
(() => {
  const el=id=>document.getElementById(id);
  const tracks=[
    {name:'Cozy Orbit', url:'squad_cozy_orbit.ogg', desc:'Sanfter Lo-Fi-Hub · 90 BPM'},
    {name:'Neon Sprint', url:'squad_neon_sprint.ogg', desc:'Arcade-Elektro · 144 BPM'}
  ];
  let track=0, userStarted=false;
  const savedLevel=localStorage.getItem('squad-original-volume-v1');
  let level=savedLevel===null?24:Number(savedLevel);
  if(!Number.isFinite(level)||level<0||level>100)level=24;
  const player = new Audio(tracks[0].url);
  player.loop=true;player.preload='none';player.volume=level/100;
  const button=el('sound-play');
  const title=el('sound-title');
  const dock=el('squad-music-dock');
  const vol=el('sound-volume');
  vol.value=String(level); el('sound-volume-label').textContent=level+'%';
  const update=()=>{
    title.textContent=tracks[track].name;
    button.textContent=player.paused?'▶ Musik starten':'Ⅱ Musik pausieren';
    dock.classList.toggle('playing',!player.paused);
    dock.querySelector('.music-track small').textContent=tracks[track].desc+' · Original Soundtrack';
  };
  const setTrack=(i,onlyIfPlaying=false)=>{
    if(onlyIfPlaying&&!userStarted)return;
    const wasPlaying=!player.paused;
    track=(i+tracks.length)%tracks.length;
    player.pause();player.src=tracks[track].url;player.load();
    if(wasPlaying)player.play().catch(()=>{userStarted=false;update()});
    update();
  };
  const play=()=>{
    if(!player.paused){player.pause();update();return}
    userStarted=true;
    player.play().then(update).catch(()=>{userStarted=false;update();
      window.notify?.('Musikstart wurde vom Browser verhindert. Bitte erneut auf ▶ klicken.',true);
    });
  };
  button.addEventListener('click',play);
  el('sound-prev').addEventListener('click',()=>setTrack(track-1));
  el('sound-next').addEventListener('click',()=>setTrack(track+1));
  player.addEventListener('pause',update);
  player.addEventListener('play',update);
  player.addEventListener('error',()=>{window.notify?.('Musikdatei konnte nicht geladen werden.',true);update()});
  vol.addEventListener('input',()=>{
    const v=Math.min(100,Math.max(0,Number(vol.value)));
    player.volume=v/100;el('sound-volume-label').textContent=v+'%';
    localStorage.setItem('squad-original-volume-v1',String(v));
  });
  el('sound-collapse').addEventListener('click',()=>{
    dock.classList.toggle('collapsed');
    el('sound-collapse').textContent=dock.classList.contains('collapsed')?'+':'−';
  });
  const auto=el('sound-auto-cube');
  auto.checked=localStorage.getItem('squad-sound-auto-cube-v1')!=='0';
  auto.addEventListener('change',()=>localStorage.setItem('squad-sound-auto-cube-v1',auto.checked?'1':'0'));
  let effectEnabled=true, ctx=null;
  const effect=(type)=>{
    if(!effectEnabled||document.hidden)return;
    try{
      ctx ||= new (window.AudioContext||window.webkitAudioContext)();
      if(ctx.state==='suspended')ctx.resume().catch(()=>{});
      const now=ctx.currentTime;
      const presets={
        tick:[[380,.065,'sine',.035]],ready:[[910,.08,'triangle',.040]],
        success:[[610,.055,'sine',.05],[880,.12,'triangle',.037]],
        fail:[[180,.13,'triangle',.045]],
        jump:[[410,.095,'triangle',.048],[610,.045,'sine',.019]],
        crystal:[[720,.045,'sine',.036],[1080,.11,'triangle',.03]],
        crash:[[220,.16,'sawtooth',.045],[98,.24,'triangle',.025]],
        revive:[[495,.09,'sine',.033],[740,.12,'triangle',.034],[990,.19,'sine',.033]],
        pause:[[385,.11,'sine',.035]],
        transform:[[290,.065,'triangle',.045],[440,.085,'sine',.04],[770,.12,'sine',.032]],
        portal:[[260,.13,'triangle',.045],[560,.12,'sine',.03]]
      };
      for(let i=0;i<(presets[type]||[]).length;i++){
        const [f,d,wave,gain]=presets[type][i],start=now+i*.05;
        const o=ctx.createOscillator(),v=ctx.createGain();
        o.type=wave;o.frequency.setValueAtTime(f,start);
        if(type==='crash')o.frequency.exponentialRampToValueAtTime(Math.max(65,f*.47),start+d);
        v.gain.setValueAtTime(.0001,start);
        v.gain.exponentialRampToValueAtTime(gain,start+.008);
        v.gain.exponentialRampToValueAtTime(.0001,start+d);
        o.connect(v);v.connect(ctx.destination);
        o.start(start);o.stop(start+d+.01);
      }
    }catch{/* sound is optional */}
  };
  window.SquadSounds={
    effect,
    setEffects:enabled=>{effectEnabled=!!enabled},
    gameStarted:()=>{if(auto.checked&&userStarted)setTrack(1,true)},
    leaveArcade:()=>{if(auto.checked&&userStarted&&track===1)setTrack(0,true)},
    trackName:()=>tracks[track].name
  };
  update();
})();
