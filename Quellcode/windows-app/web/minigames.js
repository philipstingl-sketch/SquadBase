'use strict';
// SQUAD ARCADE 3.3: standalone, offline mini-games embedded into the Windows app.
// No third-party game scripts, generated scores, accounts or plug-ins.
(() => {
  const el = id => document.getElementById(id);
  const store = (key, fallback) => { try { return JSON.parse(localStorage.getItem(key)) ?? fallback; } catch { return fallback; } };
  const persist = (key, value) => {
    try { localStorage.setItem(key, JSON.stringify(value)); } catch { /* private mode */ }
    if (/^(squad-arcade-|squad-cube-)[a-z0-9-]+$/.test(key)) {
      fetch('api/arcade',{method:'POST',headers:{'Content-Type':'application/json','X-Squad-Base':'1'},body:JSON.stringify({key,value})}).catch(()=>{});
    }
  };
  const inArcade = () => el('minigames')?.classList.contains('active');
  const clamp = (x, a, b) => Math.min(b, Math.max(a, x));
  const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches || false;

  // Reaction test: five-round challenge, genuine monotonic timing, false starts.
  const scoresKey = 'squad-arcade-reaction-v1'; // keep old personal bests
  let reactionScores = store(scoresKey, []);
  if (!Array.isArray(reactionScores)) reactionScores = [];
  reactionScores = reactionScores.filter(x => Number.isFinite(x) && x > 0 && x < 15000).slice(-100);
  let reactionPhase = 'ready', waitTimer = 0, greenAt = 0, falseStarts = 0;
  let challengeActive = false, challengeScores = [];
  const challengeKey = 'squad-arcade-reaction-challenge-v2';
  let challengeBest = store(challengeKey, null);
  if (!Number.isFinite(challengeBest) || challengeBest < 1) challengeBest = null;
  function updateReactionStats() {
    const last = reactionScores.at(-1);
    const last5 = reactionScores.slice(-5);
    el('reaction-last').textContent = last ? last + ' ms' : '–';
    el('reaction-best').textContent = reactionScores.length ? Math.min(...reactionScores) + ' ms' : '–';
    el('reaction-average').textContent = last5.length ? Math.round(last5.reduce((a,b)=>a+b,0)/last5.length) + ' ms' : '–';
    el('reaction-challenge-best').textContent = challengeBest ? challengeBest + ' ms' : '–';
    el('reaction-false-starts').textContent = String(falseStarts);
    el('reaction-round').textContent = challengeActive ? `${challengeScores.length + 1} / 5` : 'Freies Training';
    el('reaction-progress').style.width = (challengeActive ? challengeScores.length * 20 : 0) + '%';
  }
  function reactionLook(phase, glyph, title, description) {
    reactionPhase = phase;
    const pad = el('reaction-pad');
    pad.className = 'reaction-pad ' + phase;
    el('reaction-emoji').textContent = glyph;
    el('reaction-label').textContent = title;
    el('reaction-hint').textContent = description;
    pad.setAttribute('aria-label', title + '. ' + description);
  }
  function cancelReaction() {
    clearTimeout(waitTimer); waitTimer = 0;
    reactionLook('ready', '⚡', 'Zum Starten klicken', 'Warten, bis die Fläche grün wird.');
  }
  function startWaiting() {
    clearTimeout(waitTimer);
    reactionLook('waiting', '◉', 'Warten …', 'Nicht klicken – warte auf GRÜN!');
    waitTimer = setTimeout(() => {
      if (reactionPhase !== 'waiting') return;
      if (!inArcade() || document.hidden) {cancelReaction();return;}
      greenAt = performance.now(); window.SquadSounds?.effect('ready');
      reactionLook('go', '✦', 'JETZT!', 'So schnell wie möglich klicken oder Leertaste drücken.');
    }, 1550 + Math.random() * 3200);
  }
  function reactionTap() {
    if (!inArcade()) return;
    if (['ready', 'result', 'early', 'challenge-done'].includes(reactionPhase)) { window.SquadSounds?.effect('tick'); return startWaiting(); }
    if (reactionPhase === 'waiting') {
      clearTimeout(waitTimer); falseStarts++; window.SquadSounds?.effect('fail');
      reactionLook('early', '✕', 'Fehlstart!', 'Du warst zu früh. Zum Wiederholen klicken.');
      updateReactionStats(); return;
    }
    if (reactionPhase !== 'go') return;
    const ms = Math.max(1, Math.round(performance.now() - greenAt)); window.SquadSounds?.effect('success');
    reactionScores.push(ms);
    reactionScores = reactionScores.slice(-100);
    persist(scoresKey, reactionScores);
    if (challengeActive) {
      challengeScores.push(ms);
      if (challengeScores.length >= 5) {
        const avg = Math.round(challengeScores.reduce((a,b)=>a+b,0) / 5);
        if (challengeBest === null || avg < challengeBest) {
          challengeBest = avg; persist(challengeKey, avg);
        }
        challengeActive = false;
        el('reaction-challenge').textContent = '⚡ 5-Runden-Challenge starten';
        reactionLook('challenge-done', '🏆', 'Ø ' + avg + ' ms', 'Alle 5 Runden geschafft! Klick für freien Test.');
      } else {
        reactionLook('result', '✓', ms + ' ms', `Runde ${challengeScores.length}/5 fertig. Weiterklicken für Runde ${challengeScores.length + 1}.`);
      }
    } else {
      reactionLook('result', '✓', ms + ' ms', 'Nächsten Versuch starten.');
    }
    updateReactionStats();
  }
  el('reaction-pad').addEventListener('click', reactionTap);
  el('reaction-challenge').addEventListener('click', () => {
    challengeActive = true; challengeScores = []; falseStarts = 0;
    el('reaction-challenge').textContent = '↻ Challenge neu starten';
    cancelReaction(); updateReactionStats();
    el('reaction-pad').focus();
  });
  el('reaction-reset').addEventListener('click', () => {
    reactionScores = []; challengeScores = []; challengeActive = false; falseStarts = 0; challengeBest = null;
    persist(scoresKey, []); persist(challengeKey, null);
    el('reaction-challenge').textContent = '⚡ 5-Runden-Challenge starten';
    cancelReaction(); updateReactionStats();
  });
  updateReactionStats();

  // Cube Rush 3.3: fixed-step physics, jump buffering, coyote time, stable framerate,
  // mathematically spaced obstacles, collectible sparks, difficulty modes and saved records.
  const canvas = el('cube-canvas');
  const ctx = canvas.getContext('2d', { alpha: false });
  const W = 1000, H = 380, FLOOR = 318, CEIL = 54, SIDE = 36;
  const gravity = 1900, jumpImpulse = -705, timestep = 1 / 120;
  const difficulties = {
    easy:   { label:'Entspannt', base:225, max:505, ramp:1.55, gap:1.53, double:.08, block:.18, bonus:1 },
    normal: { label:'Normal', base:268, max:625, ramp:2.05, gap:1.28, double:.19, block:.28, bonus:1.25 },
    hard:   { label:'Expert', base:322, max:720, ramp:2.65, gap:1.17, double:.30, block:.35, bonus:1.6 },
    extreme:{ label:'Extrem', base:365, max:790, ramp:3.20, gap:1.13, double:.32, block:.4, bonus:2.0 }
  };
  const settingKey = 'squad-cube-mode-v2';
  let difficulty = localStorage.getItem(settingKey) || 'normal';
  if (!difficulties[difficulty]) difficulty = 'normal';
  el('cube-difficulty').value = difficulty;
  const bestKey = d => 'squad-cube-best-v2-' + d;
  const originalBest = store('squad-arcade-cube-best-v1', 0);
  const bests = {};
  for (const d of Object.keys(difficulties)) bests[d] = Number(store(bestKey(d), d === 'normal' ? originalBest : 0)) || 0;
  let mode = 'ready', raf = 0, frameTime = 0, accumulator = 0;
  const playModes=['endless','campaign','practice'];
  let playMode=store('squad-cube-play-mode-v3','endless');if(!playModes.includes(playMode))playMode='endless';
  let unlocked=Math.max(1,Math.min(10,Number(store('squad-cube-unlocked-v3',1))||1));
  let stage=Math.max(1,Math.min(unlocked,Number(store('squad-cube-stage-v3',1))||1));
  let wallet=Math.max(0,Number(store('squad-cube-wallet-v3',0))||0);
  const skins={mint:{price:0,color:'#95f6ba',stroke:'#e8fff0'},ocean:{price:20,color:'#71cafe',stroke:'#d5f3ff'},violet:{price:50,color:'#bda6ff',stroke:'#f6ebff'},gold:{price:100,color:'#f9cc68',stroke:'#fff1c4'},ember:{price:140,color:'#ff9862',stroke:'#ffe4b9'},ice:{price:185,color:'#89e7ff',stroke:'#dffaff'},neon:{price:250,color:'#f08cff',stroke:'#ffe0ff'}};
  let unlockedSkins=store('squad-cube-skins-v3',['mint']);if(!Array.isArray(unlockedSkins))unlockedSkins=['mint'];
  let skin=store('squad-cube-skin-v3','mint');if(!unlockedSkins.includes(skin)||!skins[skin])skin='mint';
  let orbReady=0, speedBoost=0, gravitySign=1, powers=[], checkpoint=null, lastCheckpoint=0;
  const forms=['cube','ship','wave','ball'];
  let startForm=store('squad-cube-start-form-v4','cube');if(!forms.includes(startForm))startForm='cube';
  let playerForm=startForm, inputHeld=false, spawnCount=0, combo=0, maxCombo=0;
  let runMissions=[false,false,false], missionRewarded=false, collectedTotal=0;
  const formTitles={cube:'◆ WÜRFEL',ship:'➤ RAUMSCHIFF',wave:'〽 WAVE',ball:'◉ GRAVITY BALL'};
  el('cube-start-form').value=startForm;
  el('cube-play-mode').value=playMode;el('cube-stage').value=String(stage);el('cube-skin').value=skin;
  const campaignSeconds=()=>26+stage*12;
  const updateShop=()=>{
    el('cube-wallet').textContent=String(wallet);
    const choice=el('cube-skin').value,price=skins[choice].price;
    el('cube-buy-skin').textContent=unlockedSkins.includes(choice)?'✓ Skin ausgerüstet':`Freischalten · ${price} 💎`;
    el('cube-stage').value=String(stage);
    [...el('cube-stage').options].forEach(opt=>{opt.disabled=Number(opt.value)>unlocked});
  };
  updateShop();
  let distance = 0, speed = 0, spawnDistance = 0, points = 0, gems = 0, coins = [], obstacles = [], particles = [];
  let elapsed = 0, level = 1, revives = 0, invincible = 0;
  const reviveCost = () => 5 + revives * 3;
  let rider = { x: 155, y: FLOOR-SIDE, vy: 0, angle: 0, grounded: true, coyote: 0, buffered: 0 };
  let lastHUDAt = 0;
  let runRewardEnabled=true; // Practice is training, not a repeatable crystal farm.
  let checkpointPenalty=0;
  let boundaryTime=0;
  let sound = localStorage.getItem('squad-cube-sound-v2') !== '0';
  el('cube-sound').textContent = sound ? '♫ Spielsounds an' : '♪ Spielsounds aus';
  let audioCtx = null;
  let quality=localStorage.getItem('squad-cube-quality-v4')||'high';if(!['low','high'].includes(quality))quality='high';
  el('cube-quality').value=quality;
  function playSound(frequency, duration, type='sine', volume=0.045) {
    if (!sound) return;
    try {
      audioCtx ||= new (window.AudioContext || window.webkitAudioContext)();
      if (audioCtx.state === 'suspended') audioCtx.resume().catch(()=>{});
      const osc = audioCtx.createOscillator(), gain = audioCtx.createGain(), start = audioCtx.currentTime;
      osc.type = type; osc.frequency.setValueAtTime(frequency, start);
      gain.gain.setValueAtTime(volume, start); gain.gain.exponentialRampToValueAtTime(0.001, start+duration);
      osc.connect(gain); gain.connect(audioCtx.destination); osc.start(start); osc.stop(start+duration+.01);
    } catch { /* audio optional */ }
  }
  const background = document.createElement('canvas');
  background.width = W; background.height = H;
  const bg = background.getContext('2d');
  const g = bg.createLinearGradient(0,0,0,H);
  g.addColorStop(0,'#0a1728'); g.addColorStop(.6,'#172d34'); g.addColorStop(1,'#122f26');
  bg.fillStyle = g; bg.fillRect(0,0,W,H);
  for (let i=0;i<44;i++) {
    const x=((i*263+71)%1000), y=((i*71+49)%225), radius=1+(i%3)*.5;
    bg.fillStyle='rgba(206,255,222,'+(0.15+(i%4)*.1)+')';
    bg.beginPath();bg.arc(x,y,radius,0,Math.PI*2);bg.fill();
  }
  const moon = bg.createRadialGradient(780,80,6,780,80,100);
  moon.addColorStop(0,'#b1efc688'); moon.addColorStop(.35,'#a5fac32a'); moon.addColorStop(1,'#a5fac300');
  bg.fillStyle=moon;bg.fillRect(680,0,210,210);
  bg.beginPath();bg.arc(780,80,20,0,2*Math.PI);bg.fillStyle='#c7f6d9aa';bg.fill();
  function obstacleGap() {
    const cfg = difficulties[difficulty];
    // More room at high speeds: never spawn hazards closer than the airborne distance + reaction gap.
    const airTime = Math.abs(2 * jumpImpulse / gravity);
    return Math.max(speed * (airTime + .32), speed * (cfg.gap + (Math.random()-.5)*.23));
  }
  function hud(force=false) {
    const now = performance.now();
    if (!force && now-lastHUDAt<90) return;
    lastHUDAt=now;
    el('cube-score').textContent=String(points);
    el('cube-best').textContent=String(bests[difficulty]);
    el('cube-speed').textContent=(speed/difficulties[difficulty].base).toFixed(1)+'×';
    el('cube-status').textContent=({ready:'Bereit',running:'Läuft',paused:'Pause',revivable:playMode==='practice'?'Checkpoint':'Rettung?',dead:'Game Over',won:'Level geschafft'})[mode];
    el('cube-danger-level').textContent=`${playMode==='campaign'?'KAMPAGNE '+stage:'LEVEL '+level} · ${Math.floor(elapsed)}s${playMode==='campaign'?'/'+campaignSeconds()+'s':''}`;
    el('cube-revive-cost').textContent=String(reviveCost());
    el('cube-gems').textContent=String(gems);el('cube-wallet').textContent=String(wallet);
    el('cube-form-label').textContent=formTitles[playerForm];
    el('cube-combo-label').textContent='Combo ×'+combo+(maxCombo>combo?' · Best '+maxCombo:'');
    el('cube-mission-label').textContent='Missionen: '+runMissions.filter(Boolean).length+'/3';
    const fraction=playMode==='campaign'?Math.min(1,elapsed/campaignSeconds()):Math.min(1,(elapsed%18)/18);
    el('cube-progress-label').textContent=playMode==='campaign'?'Fortschritt: '+Math.round(fraction*100)+'%':'Level '+level;
    el('cube-mission-progress').style.width=(runMissions.filter(Boolean).length/3*100)+'%';
  }
  function overlay(title,text,visible) {
    el('cube-overlay-title').textContent=title;
    el('cube-overlay-desc').textContent=text;
    el('cube-overlay').hidden=!visible;
  }
  function reset() {
    cancelAnimationFrame(raf);
    runRewardEnabled=playMode!=='practice';checkpointPenalty=0;boundaryTime=0;
    mode='ready'; distance=0; points=0; gems=0; coins=[]; obstacles=[]; powers=[]; particles=[]; elapsed=0; level=1; revives=0; invincible=0;orbReady=0;speedBoost=0;gravitySign=1;checkpoint=null;lastCheckpoint=0;playerForm=startForm;inputHeld=false;spawnCount=0;combo=0;maxCombo=0;runMissions=[false,false,false];missionRewarded=false;collectedTotal=0;
    el('cube-revive').hidden=true;
    speed=difficulties[difficulty].base; spawnDistance=300; accumulator=0; frameTime=0;
    rider={ x:155,y:(startForm==='ship'||startForm==='wave'?(FLOOR+CEIL-SIDE)/2:FLOOR-SIDE),vy:0,angle:0,grounded:!(startForm==='ship'||startForm==='wave'),coyote:.08,buffered:0 };
    el('cube-pause').disabled=true;el('cube-pause').textContent='Ⅱ Pause';
    el('cube-start').textContent='▶ Spiel starten';
    overlay('◆ CUBE RUSH 3.6','Vier Spielformen, 10 Level, neue Patterns und Rhythmus! '+(playMode==='practice'?'Im Practice-Modus gibt es Checkpoints.':playMode==='campaign'?'Schaffe Level '+stage+'!':'Wie weit kommst du?'),true);
    hud(true);draw();
  }
  function start(autojump=false) {
    const held=inputHeld;reset();inputHeld=held;mode='running';overlay('','',false);window.SquadSounds?.gameStarted();
    el('cube-start').textContent='↻ Neustart';el('cube-pause').disabled=false;
    if (autojump) rider.buffered=.12;
    frameTime=performance.now();hud(true);
    raf=requestAnimationFrame(frame);
  }
  function pause(resume=true) {
    if (mode==='running') {
      mode='paused';cancelAnimationFrame(raf);el('cube-pause').textContent='▶ Fortsetzen';window.SquadSounds?.effect('pause');
      overlay('Ⅱ PAUSE','Fortsetzen, wenn du bereit bist.',true);hud(true);
    } else if (resume && mode==='paused') {
      mode='running';el('cube-pause').textContent='Ⅱ Pause';overlay('','',false);
      frameTime=performance.now();accumulator=0;hud(true);raf=requestAnimationFrame(frame);
    }
  }
  function spark(x,y,color,count=10) {
    if (reducedMotion||quality==='low') return;
    for(let i=0;i<count;i++){
      if(particles.length>100)break;
      const angle=Math.random()*Math.PI*2, velocity=50+Math.random()*140;
      particles.push({x,y,vx:Math.cos(angle)*velocity,vy:Math.sin(angle)*velocity,t:.3+Math.random()*.32,color,size:2+Math.random()*3});
    }
  }
  function jump() {
    if(!inArcade())return;
    if(mode==='ready'||mode==='dead'||mode==='won'){start(true);return}
    if(mode==='revivable')return;
    if(mode==='paused'){pause(true);return}
    if(playerForm==='ship'||playerForm==='wave'){inputHeld=true;return}
    if (!rider.grounded && orbReady > 0 && playerForm==='cube') {
      orbReady--;rider.vy=jumpImpulse*.93*gravitySign;rider.buffered=0;window.SquadSounds?.effect('jump');
      spark(rider.x+SIDE/2,rider.y+SIDE/2,'#8cd8ff',14);return;
    }
    rider.buffered=.13;
  }
  function finish() {
    if (mode !== 'running') return;
    cancelAnimationFrame(raf);
    window.SquadSounds?.effect('crash');
    inputHeld=false; // avoid thrust carrying over after death
    spark(rider.x+18,rider.y+18,'#f6a46e',22);
    if(playMode!=='practice'&&points>bests[difficulty]) {bests[difficulty]=points;persist(bestKey(difficulty),points)}
    el('cube-start').textContent='↻ Neuer Run';el('cube-pause').disabled=true;
    const canRevive=playMode==='practice'||(gems>=reviveCost()&&wallet>=reviveCost());
    combo=0;
    mode=canRevive?'revivable':'dead';
    el('cube-revive').hidden=!canRevive;el('cube-revive').textContent=playMode==='practice'?'↻ Zum Checkpoint':'💎 Wiederbeleben';
    overlay(playMode==='practice'?'◉ PRACTICE CHECKPOINT':canRevive?'💎 RETTUNG MÖGLICH':'GAME OVER',canRevive
      ? playMode==='practice'?'Kostenlos zum letzten Checkpoint zurückkehren (ohne Bank-Kristalle).':`${points} Punkte · Setze ${reviveCost()} Kristalle ein und erhalte 2,8 Sek. Schutz!`
      : `${points} Punkte · ${gems} Kristalle · Rekord: ${bests[difficulty]}`,true);
    hud(true);draw();
  }
  function revive() {
    if(mode!=='revivable'||(playMode!=='practice'&&(gems<reviveCost()||wallet<reviveCost())))return;
    if(playMode==='practice') {
      const c=checkpoint;if(c){distance=c.distance;elapsed=c.elapsed;level=c.level;points=c.points;gems=c.gems;collectedTotal=c.collectedTotal;combo=c.combo;maxCombo=c.maxCombo;spawnCount=c.spawnCount;}
      else {distance=0;elapsed=0;level=1;points=0;gems=0;collectedTotal=0;combo=0;maxCombo=0;spawnCount=0;}
      lastCheckpoint=elapsed;invincible=3.2;
    } else {
      gems-=reviveCost();wallet=Math.max(0,wallet-reviveCost());persist('squad-cube-wallet-v3',wallet);revives++;invincible=2.8;
    }
    mode='running';powers=[];orbReady=0;speedBoost=0;
    // Protect the respawn: remove obstacles and crystals just ahead and put the
    // player on solid ground. Keep score/time/difficulty, no points reset.
    obstacles=obstacles.filter(o=>o.x>rider.x+speed*2.8+SIDE+140);
    coins=coins.filter(c=>c.x>rider.x+speed*2.8+SIDE+140);
    spawnDistance=distance+Math.max(700,speed*2.4);
    obstacles=[];coins=[];
    gravitySign=1;boundaryTime=0;rider.y=FLOOR-SIDE;rider.vy=0;rider.grounded=true;rider.angle=0;rider.buffered=0;inputHeld=false;
    el('cube-revive').hidden=true;el('cube-pause').disabled=false;
    overlay('','',false);window.SquadSounds?.effect('revive');
    frameTime=performance.now();accumulator=0;hud(true);raf=requestAnimationFrame(frame);
  }
  el('cube-revive').addEventListener('click',revive);
  // Curated combinations instead of pure random hazards. Distances are calibrated
  // against the player's maximum jump, so higher difficulties remain readable.
  const patternSets={
    easy:['single','single','double','block','saw','lowgate'],
    normal:['single','double','block','saw','lowgate','triple','dual'],
    hard:['double','block','saw','lowgate','triple','dual','drone'],
    extreme:['double','block','saw','lowgate','triple','dual','drone','mixed']
  };
  function spawn() {
    const cfg=difficulties[difficulty];
    const pool=patternSets[difficulty];
    const pattern=pool[(spawnCount*5+Math.floor(spawnCount/3)+stage+Math.floor(elapsed/11))%pool.length];
    spawnCount++;
    let width=38,kind='spike',count=1,height=38,side='floor';
    if(pattern==='double'){count=2;width=76}
    if(pattern==='triple'){count=3;width=114}
    if(pattern==='block'){kind='block';height=44;width=36}
    if(pattern==='saw'){kind='saw';height=42;width=43}
    if(pattern==='lowgate'){kind='gate';height=66;width=45}
    if(pattern==='dual'){kind='spike';count=2;width=76;side='ceiling'}
    if(pattern==='drone'){kind='drone';height=36;width=40}
    if(pattern==='mixed'){kind='gate';height=80;width=55}
    const x=W+43;
    obstacles.push({x,width,height,kind,count,side,pattern});
    // Expose special forms only after the first few hazards and keep the switch
    // in front of the next obstacle for time to adapt.
    if(spawnCount>2&&spawnCount%4===0){
      const cycle=['ship','cube','wave','cube','ball','cube'];
      powers.push({x:x-260,y:185,type:'form',form:cycle[(Math.floor(spawnCount/4)+stage)%cycle.length],hit:false});
    }else if(elapsed>8&&spawnCount%3===0){
      const types=['orb','pad','portal','gravity'];
      const type=types[(spawnCount+level)%types.length];
      const py=type==='orb'?(gravitySign>0?FLOOR-112:CEIL+112):type==='pad'?(gravitySign>0?FLOOR-8:CEIL+8):185;
      powers.push({x:x-210,y:py,type,hit:false});
    }
    // Collectible groups on safe, predictable arcs.
    if(spawnCount%5!==0){
      const cy=playerForm==='ship'||playerForm==='wave'?183:(gravitySign>0?FLOOR-108:CEIL+108);
      coins.push({x:x+width*.5,y:cy,r:12,hit:false});
      if(level>=3&&spawnCount%3===0)coins.push({x:x+width*.5+48,y:cy-27,r:12,hit:false});
    }
    // For larger patterns reserve extra time to prepare the next jump.
    const hazardTime=(pattern==='triple'||pattern==='mixed')?.24:0;
    const minGap=Math.max(obstacleGap()+speed*hazardTime,width+speed*.83);
    // Quantize distances to 144 BPM (four eighth-notes etc.); fixed-step
    // gameplay stays stable even when the soundtrack is muted.
    const beatDistance=speed*(60/144);
    spawnDistance=distance+Math.ceil(minGap/beatDistance)*beatDistance;
  }
  function collide(o) {
    // Forgiving hit boxes; triangle collisions only near the inner half of spikes.
    const l=rider.x+7, r=rider.x+SIDE-7, t=rider.y+6, b=rider.y+SIDE-4;
    const top=o.side==='ceiling';
    if(o.kind==='gate')return l<o.x+o.width-7&&r>o.x+7&&(t<CEIL+o.height-2||b>FLOOR-o.height+2);
    if(o.kind==='drone'){const cy=185+Math.sin(elapsed*3+o.x*.02)*32;const dx=rider.x+SIDE/2-(o.x+o.width/2),dy=rider.y+SIDE/2-cy;return dx*dx+dy*dy<31*31}
    if(o.kind==='block')return l<o.x+o.width-6&&r>o.x+6&&(top?(t<CEIL+o.height-4&&b>CEIL+3):(b>FLOOR-o.height+4&&t<FLOOR-3));
    if(o.kind==='saw'){
      const y=(top?CEIL+21:FLOOR-21)+Math.sin(elapsed*3.3+o.x*.03)*7;
      const dx=rider.x+SIDE/2-(o.x+o.width/2),dy=rider.y+SIDE/2-y;
      return dx*dx+dy*dy<29*29;
    }
    for(let i=0;i<o.count;i++){
      const x=o.x+i*38;
      if(l<x+31&&r>x+7&&(top?(t<CEIL+o.height-11&&b>CEIL+3):(b>FLOOR-o.height+11&&t<FLOOR-3)))return true;
    }
    return false;
  }
  function update(dt) {
    const cfg=difficulties[difficulty];
    elapsed+=dt;const lastLevel=level;level=1+Math.floor(elapsed/18);invincible=Math.max(0,invincible-dt);
    if(level>lastLevel)window.SquadSounds?.effect('ready');
    speedBoost=Math.max(0,speedBoost-dt);
    speed=Math.min(cfg.max,cfg.base+elapsed*cfg.ramp)*(speedBoost>0?1.13:1);
    if(playMode==='campaign'&&elapsed>=campaignSeconds()){winStage();return}
    if(playMode==='practice'&&elapsed-lastCheckpoint>=12){
      checkpoint={distance,elapsed,level,points,gems,collectedTotal,combo,maxCombo,spawnCount};lastCheckpoint=elapsed;window.SquadSounds?.effect('success');
    }
    const dx=speed*dt;distance+=dx;
    points=Math.floor(distance/35*cfg.bonus)+collectedTotal*15+maxCombo*4;
    // Four independently controlled forms, all with fixed-step physics.
    rider.coyote=rider.grounded?.09:Math.max(0,rider.coyote-dt);
    rider.buffered=Math.max(0,rider.buffered-dt);
    if(playerForm==='cube'){
      if(rider.buffered>0&&(rider.grounded||rider.coyote>0)){
        rider.vy=jumpImpulse*gravitySign;rider.grounded=false;rider.coyote=0;rider.buffered=0;
        window.SquadSounds?.effect('jump');spark(rider.x+SIDE/2,rider.y+SIDE/2,'#94ffc0',8);
      }
      rider.vy+=gravity*gravitySign*dt;rider.y+=rider.vy*dt;
    }else if(playerForm==='ball'){
      if(rider.buffered>0){gravitySign*=-1;rider.vy=gravitySign*160;rider.buffered=0;
        rider.grounded=false;invincible=Math.max(.20,invincible);window.SquadSounds?.effect('portal')}
      rider.vy+=gravity*gravitySign*dt;rider.y+=rider.vy*dt;
    }else if(playerForm==='ship'){
      const thrust=inputHeld?-1280:850;
      rider.vy=clamp((rider.vy+thrust*dt)*.987,-390,390);
      rider.y+=rider.vy*dt;rider.angle=clamp(rider.vy/750,-.34,.34);
    }else if(playerForm==='wave'){
      rider.vy=(inputHeld?-320:320);
      rider.y+=rider.vy*dt;rider.angle=inputHeld?-.65:.65;
    }
    const boundaryContact=rider.y>=FLOOR-SIDE || rider.y<=CEIL;
    if((playerForm==='wave'||playerForm==='ship')&&boundaryContact){boundaryTime+=dt;if(boundaryTime>.11&&invincible<=0){finish();return;}}else boundaryTime=0;
    if(rider.y>=FLOOR-SIDE){rider.y=FLOOR-SIDE;rider.vy=0;rider.grounded=true;
      if(playerForm==='cube'||playerForm==='ball')rider.angle=Math.round(rider.angle/(Math.PI/2))*(Math.PI/2)
    }else if(rider.y<=CEIL){rider.y=CEIL;rider.vy=0;rider.grounded=true;
      if(playerForm==='cube'||playerForm==='ball')rider.angle=Math.round(rider.angle/(Math.PI/2))*(Math.PI/2)
    }else{rider.grounded=false;if(playerForm==='cube'||playerForm==='ball')rider.angle+=dt*5.8}
    if(distance>=spawnDistance)spawn();
    for(const p of powers)p.x-=dx;
    for(const p of powers){
      if(p.hit||Math.abs(rider.x+SIDE/2-p.x)>SIDE*.8)continue;
      if(p.type==='pad'&&rider.grounded&&playerForm==='cube'){rider.vy=jumpImpulse*gravitySign*1.15;rider.grounded=false;p.hit=true;window.SquadSounds?.effect('jump');spark(p.x,gravitySign===1?FLOOR:CEIL,'#90f7e8',15)}
      if(p.type==='orb'&&Math.abs(rider.y+SIDE/2-p.y)<65){orbReady=1;p.hit=true;window.SquadSounds?.effect('crystal');spark(p.x,p.y,'#91cfff',12)}
      if(p.type==='portal'){speedBoost=3.2;p.hit=true;window.SquadSounds?.effect('ready');spark(p.x,p.y,'#ecadff',15)}
      if(p.type==='form'){playerForm=p.form;gravitySign=1;boundaryTime=0;rider.vy=0;rider.y=clamp(rider.y,CEIL+20,FLOOR-SIDE-20);invincible=Math.max(invincible,.8);p.hit=true;window.SquadSounds?.effect('transform');spark(p.x,p.y,'#ddadff',22)}
      if(p.type==='gravity'){gravitySign*=-1;rider.vy=gravitySign*200;rider.grounded=false;invincible=Math.max(invincible,.9);p.hit=true;window.SquadSounds?.effect('revive');spark(p.x,p.y,'#a9b8ff',22)}
    }
    powers=powers.filter(p=>!p.hit&&p.x>-70);
    for(const o of obstacles)o.x-=dx;
    obstacles=obstacles.filter(o=>o.x+o.width>-60);
    for(const c of coins){c.x-=dx;
      if(!c.hit&&Math.abs((rider.x+SIDE/2)-c.x)<SIDE/2+c.r-3&&Math.abs((rider.y+SIDE/2)-c.y)<SIDE/2+c.r-3){
        c.hit=true;gems++;collectedTotal++;combo++;maxCombo=Math.max(maxCombo,combo);if(runRewardEnabled){wallet++;persist('squad-cube-wallet-v3',wallet);}window.SquadSounds?.effect('crystal');if(!window.SquadSounds)playSound(750,.095,'sine',.027);spark(c.x,c.y,'#fceca1',12);
      }
    }
    if(coins.some(c=>!c.hit&&c.x<rider.x-30))combo=0;
    coins=coins.filter(c=>!c.hit&&c.x>-50);
    for(const p of particles){p.x+=p.vx*dt;p.y+=p.vy*dt;p.vy+=260*dt;p.t-=dt}
    particles=particles.filter(p=>p.t>0);
    runMissions=[elapsed>=20,collectedTotal>=5,level>=3];
    if(runMissions.every(Boolean)&&!missionRewarded){
      missionRewarded=true;if(runRewardEnabled){wallet+=12;persist('squad-cube-wallet-v3',wallet);}
      window.SquadSounds?.effect('success');spark(rider.x+SIDE/2,rider.y+SIDE/2,'#ffe68a',30);
    }
    if(mode==='running'&&invincible<=0 && obstacles.some(collide))finish();
  }
  function winStage(){
    if(mode!=='running')return;
    mode='won';cancelAnimationFrame(raf);
    if(stage>=unlocked&&unlocked<10){unlocked++;persist('squad-cube-unlocked-v3',unlocked)}
    if(points>bests[difficulty]){bests[difficulty]=points;persist(bestKey(difficulty),points)}
    el('cube-pause').disabled=true;el('cube-start').textContent='▶ Nächstes Level';
    wallet+=stage*3;persist('squad-cube-wallet-v3',wallet);
    overlay('🏆 LEVEL GESCHAFFT',`Level ${stage} geschafft! ${points} Punkte · +${stage*3} Bonuskristalle · Nächstes Level wählen.`,true);
    window.SquadSounds?.effect('success');updateShop();hud(true);draw();
  }
  function frame(now) {
    if(mode!=='running')return;
    const dt=clamp((now-frameTime)/1000,0,.05);frameTime=now;
    accumulator+=dt;
    let loops=0;
    while(accumulator>=timestep&&loops<6&&mode==='running') {update(timestep);accumulator-=timestep;loops++}
    if(loops===6)accumulator=0; // never enter a catch-up spiral on slow machines
    draw();hud();
    if(mode==='running')raf=requestAnimationFrame(frame);
  }
  function triangle(x,y,width,height,fill,stroke){
    ctx.beginPath();ctx.moveTo(x,y);ctx.lineTo(x+width/2,y-height);ctx.lineTo(x+width,y);ctx.closePath();
    ctx.fillStyle=fill;ctx.fill();ctx.lineWidth=2;ctx.strokeStyle=stroke;ctx.stroke();
  }
  function draw() {
    if(!ctx)return;
    ctx.drawImage(background,0,0);
    // Stage-dependent ambience and beat pulse; all drawing stays on this canvas.
    ctx.fillStyle=['#082f1822','#20437522','#62224d20','#47371c22'][Math.floor((stage-1)/3)%4];ctx.fillRect(0,0,W,H);
    const beat=(Math.sin(elapsed*Math.PI*4.8)+1)*.5;ctx.globalAlpha=.06+beat*.07;ctx.fillStyle='#89fbc6';ctx.fillRect(0,CEIL,W,3);ctx.globalAlpha=1;
    // Two lightweight forest layers at different scroll speeds.
    for(let layer=0;layer<(quality==='low'?1:2);layer++){
      const step=95+layer*37, shift=(distance*(layer?.24:.09))%step;
      const y=FLOOR-(layer?17:40);
      ctx.fillStyle=layer?'#1b4339':'#153331';
      for(let i=-1;i<Math.ceil(W/step)+1;i++){
        const x=i*step-shift;
        ctx.beginPath();ctx.moveTo(x-29,y);ctx.lineTo(x+2,y-76-layer*15);ctx.lineTo(x+33,y);ctx.fill();
        ctx.fillRect(x-3,y-5,7,32);
      }
    }
    ctx.fillStyle='#152a23';ctx.fillRect(0,FLOOR,W,H-FLOOR);
    if(gravitySign<0){ctx.fillStyle='#152a23';ctx.fillRect(0,0,W,CEIL);ctx.fillStyle='#86caff';ctx.fillRect(0,CEIL+SIDE-1,W,3)}
    ctx.fillStyle='#8df2b5';ctx.fillRect(0,FLOOR,W,4);
    ctx.strokeStyle='#84efaa22';ctx.lineWidth=1;
    for(let x=-(distance%52);x<W;x+=52){ctx.beginPath();ctx.moveTo(x,FLOOR+5);ctx.lineTo(x-18,H);ctx.stroke()}
    for(const o of obstacles){
      const top=o.side==='ceiling';
      if(o.kind==='gate'){
        ctx.fillStyle='#746de0';ctx.shadowColor='#817dfd';ctx.shadowBlur=9;
        ctx.fillRect(o.x,CEIL,o.width,o.height);ctx.fillRect(o.x,FLOOR-o.height,o.width,o.height);
        ctx.shadowBlur=0;ctx.strokeStyle='#cdccff';ctx.strokeRect(o.x,CEIL,o.width,o.height);ctx.strokeRect(o.x,FLOOR-o.height,o.width,o.height);
      }else if(o.kind==='drone'){
        const y=185+Math.sin(elapsed*3+o.x*.02)*32;
        ctx.save();ctx.translate(o.x+o.width/2,y);ctx.rotate(elapsed*3);
        ctx.beginPath();ctx.arc(0,0,20,0,Math.PI*2);ctx.fillStyle='#f069c7';ctx.shadowColor='#f77eca';ctx.shadowBlur=18;ctx.fill();ctx.shadowBlur=0;ctx.fillStyle='#290d27';ctx.fillRect(-7,-7,14,14);ctx.restore();
      }else if(o.kind==='spike'){
        for(let i=0;i<o.count;i++)triangle(o.x+i*38,top?CEIL:FLOOR,38,top?-o.height:o.height,'#dd9162','#ffddb2');
      }else if(o.kind==='saw') {
        const cy=(top?CEIL+21:FLOOR-21)+Math.sin(elapsed*3.3+o.x*.03)*7;
        ctx.save();ctx.translate(o.x+o.width/2,cy);ctx.rotate(elapsed*5.3);
        ctx.beginPath();for(let j=0;j<24;j++){const ang=j/24*Math.PI*2,rad=j%2?18:22;const px=Math.cos(ang)*rad,py=Math.sin(ang)*rad;if(!j)ctx.moveTo(px,py);else ctx.lineTo(px,py)}ctx.closePath();ctx.fillStyle='#e6b1c7';ctx.fill();ctx.strokeStyle='#fff0f4';ctx.lineWidth=2;ctx.stroke();ctx.beginPath();ctx.arc(0,0,7,0,Math.PI*2);ctx.fillStyle='#553547';ctx.fill();ctx.restore();
      }else{
        const y=top?CEIL:FLOOR-o.height;
        ctx.fillStyle='#376d9a';ctx.fillRect(o.x,y,o.width,o.height);
        ctx.strokeStyle='#a8e8ff';ctx.lineWidth=3;ctx.strokeRect(o.x+1,y+1,o.width-2,o.height-2);
        ctx.fillStyle='#c7ecff55';ctx.fillRect(o.x+8,y+6,o.width-16,5);
      }
    }
    for(const p of powers){
      ctx.save();ctx.translate(p.x,p.y);const pulse=1+Math.sin(elapsed*11)*.08;ctx.scale(pulse,pulse);
      if(p.type==='form'){ctx.fillStyle='#221a46';ctx.fillRect(-12,-42,24,84);ctx.strokeStyle='#bca2ff';ctx.lineWidth=4;ctx.strokeRect(-12,-42,24,84);ctx.fillStyle='#e0d1ff';ctx.font='bold 15px sans-serif';ctx.fillText(({cube:'◆',ship:'➤',wave:'〽',ball:'◉'})[p.form],-8,4)}
      if(p.type==='orb'){ctx.beginPath();ctx.arc(0,0,15,0,Math.PI*2);ctx.fillStyle='#7bcafb';ctx.shadowColor='#60bffe';ctx.shadowBlur=18;ctx.fill();ctx.strokeStyle='#e4f5ff';ctx.lineWidth=3;ctx.stroke()}
      if(p.type==='pad'){ctx.fillStyle='#60e9a8';ctx.shadowColor='#3bec8d';ctx.shadowBlur=16;ctx.fillRect(-24,-2,48,9)}
      if(p.type==='portal'){ctx.beginPath();ctx.ellipse(0,0,14,42,0,0,Math.PI*2);ctx.strokeStyle='#cf9cfa';ctx.lineWidth=5;ctx.shadowColor='#a267ff';ctx.shadowBlur=17;ctx.stroke()}
      if(p.type==='gravity'){ctx.beginPath();ctx.ellipse(0,0,14,45,0,0,Math.PI*2);ctx.strokeStyle='#88bfff';ctx.lineWidth=5;ctx.shadowColor='#7fa9ff';ctx.shadowBlur=17;ctx.stroke();ctx.font='bold 20px system-ui';ctx.fillStyle='#d4e5ff';ctx.fillText('⇅',-10,7)}
      ctx.restore();
    }
    const now=distance/65;
    for(const c of coins){
      ctx.save();ctx.translate(c.x,c.y+Math.sin(now+c.x*.01)*4);ctx.rotate(Math.PI/4);
      ctx.shadowColor='#ffd875';ctx.shadowBlur=12;ctx.fillStyle='#f8d976';ctx.fillRect(-8,-8,16,16);
      ctx.shadowBlur=0;ctx.fillStyle='#fff5c2';ctx.fillRect(-3,-3,6,6);ctx.restore();
    }
    for(const p of particles){
      ctx.globalAlpha=clamp(p.t*2,0,1);ctx.fillStyle=p.color;ctx.fillRect(p.x,p.y,p.size,p.size);
    }
    ctx.globalAlpha=1;
    ctx.save();ctx.translate(rider.x+SIDE/2,rider.y+SIDE/2);ctx.rotate(rider.angle);
    ctx.globalAlpha=invincible>0 ? .64+.24*Math.sin(elapsed*20) : 1;
    ctx.fillStyle=skins[skin].color;
    if(playerForm==='ball'){ctx.beginPath();ctx.arc(0,0,SIDE/2,0,Math.PI*2);ctx.fill();ctx.lineWidth=3;ctx.strokeStyle=skins[skin].stroke;ctx.stroke();ctx.beginPath();ctx.arc(0,0,6,0,Math.PI*2);ctx.fillStyle='#254e55';ctx.fill()}
    else if(playerForm==='ship'){
      ctx.beginPath();ctx.moveTo(SIDE/2,0);ctx.lineTo(-SIDE/2,-SIDE/2);ctx.lineTo(-SIDE/3,0);ctx.lineTo(-SIDE/2,SIDE/2);ctx.closePath();ctx.fill();ctx.lineWidth=3;ctx.strokeStyle=skins[skin].stroke;ctx.stroke();
      if(inputHeld){ctx.fillStyle='#ffcb73';ctx.fillRect(-SIDE/2-13,-4,13,8)}
    }else if(playerForm==='wave'){
      ctx.beginPath();ctx.moveTo(0,-SIDE/2);ctx.lineTo(SIDE/2,SIDE/2);ctx.lineTo(-SIDE/2,SIDE/2);ctx.closePath();ctx.fill();ctx.lineWidth=3;ctx.strokeStyle=skins[skin].stroke;ctx.stroke()
    }else{
      ctx.fillRect(-SIDE/2,-SIDE/2,SIDE,SIDE);ctx.strokeStyle=skins[skin].stroke;ctx.lineWidth=3;ctx.strokeRect(-SIDE/2+2,-SIDE/2+2,SIDE-4,SIDE-4);
      ctx.fillStyle='#295c4c';ctx.fillRect(-8,-6,5,6);ctx.fillRect(4,-6,5,6);ctx.fillRect(-6,6,12,4)
    }
    ctx.restore();ctx.globalAlpha=1;
    if(orbReady>0){ctx.beginPath();ctx.arc(rider.x+SIDE/2,rider.y+SIDE/2,27,0,Math.PI*2);ctx.strokeStyle='#66c9ff';ctx.lineWidth=3;ctx.stroke()}
    if(invincible>0){ctx.beginPath();ctx.arc(rider.x+SIDE/2,rider.y+SIDE/2,30,0,Math.PI*2);ctx.strokeStyle='#9deeff';ctx.lineWidth=3;ctx.stroke()}
    ctx.globalAlpha=.28+.22*Math.sin(elapsed*(144/60)*2*Math.PI);ctx.fillStyle='#aaf1d8';ctx.fillRect(0,0,W,3);ctx.globalAlpha=1;
    ctx.fillStyle='#9fe6ba';ctx.fillRect(18,34,180,5);ctx.fillStyle='#e0c17d';ctx.fillRect(18,34,Math.min(180,(elapsed%18)/18*180),5);
    ctx.fillStyle='#baf7d7bb';ctx.font='bold 13px system-ui, sans-serif';
    ctx.fillText(formTitles[playerForm]+' · '+difficulties[difficulty].label.toUpperCase()+'  •  LEVEL '+level+'  •  '+Math.floor(elapsed)+'s',18,25);
  }
  function resize() {
    const r=canvas.getBoundingClientRect(); if(!r.width||!r.height)return;
    const dpi=Math.min(window.devicePixelRatio||1,2);
    const width=Math.round(r.width*dpi),height=Math.round(r.height*dpi);
    if(canvas.width!==width||canvas.height!==height){canvas.width=width;canvas.height=height}
    ctx.setTransform(canvas.width/W,0,0,canvas.height/H,0,0);draw();
  }
  if(typeof ResizeObserver!=='undefined')new ResizeObserver(resize).observe(canvas);
  window.addEventListener('resize',resize);
  el('cube-start').addEventListener('click',()=>{
    if(mode==='won'&&stage<10){stage++;persist('squad-cube-stage-v3',stage);el('cube-stage').value=String(stage);updateShop()}
    start(false)
  });
  el('cube-pause').addEventListener('click',()=>pause(true));
  const releaseAction=()=>{inputHeld=false};
  el('cube-jump').addEventListener('pointerdown',e=>{e.preventDefault();jump()});
  el('cube-jump').addEventListener('pointerup',releaseAction);
  el('cube-jump').addEventListener('pointercancel',releaseAction);
  canvas.addEventListener('pointerdown',e=>{jump();if(playerForm==='ship'||playerForm==='wave')e.preventDefault()});
  canvas.addEventListener('pointerup',releaseAction);
  canvas.addEventListener('pointercancel',releaseAction);
  window.addEventListener('pointerup',releaseAction);
  el('cube-retry').addEventListener('click',()=>start(false));
  el('cube-start-form').addEventListener('change',e=>{
    if(!forms.includes(e.target.value))return;
    startForm=e.target.value;persist('squad-cube-start-form-v4',startForm);reset();
  });
  el('cube-difficulty').addEventListener('change',e=>{
    if(!difficulties[e.target.value])return;
    difficulty=e.target.value;persist(settingKey,difficulty);reset();
  });
  el('cube-play-mode').addEventListener('change',e=>{
    if(!playModes.includes(e.target.value))return;playMode=e.target.value;persist('squad-cube-play-mode-v3',playMode);reset()
  });
  el('cube-stage').addEventListener('change',e=>{const wanted=Number(e.target.value);
    if(!Number.isInteger(wanted)||wanted<1||wanted>unlocked){e.target.value=String(stage);return}
    stage=wanted;persist('squad-cube-stage-v3',stage);reset()
  });
  el('cube-skin').addEventListener('change',e=>{const wanted=e.target.value;if(!skins[wanted])return;
    if(unlockedSkins.includes(wanted)){skin=wanted;persist('squad-cube-skin-v3',skin);draw()}
    updateShop()
  });
  el('cube-buy-skin').addEventListener('click',()=>{const choice=el('cube-skin').value;if(!skins[choice])return;
    if(unlockedSkins.includes(choice)){skin=choice;persist('squad-cube-skin-v3',skin);updateShop();draw();return}
    if(wallet<skins[choice].price){el('cube-buy-skin').textContent=`Noch ${skins[choice].price-wallet} 💎 nötig`;return}
    wallet-=skins[choice].price;unlockedSkins.push(choice);skin=choice;
    persist('squad-cube-wallet-v3',wallet);persist('squad-cube-skins-v3',unlockedSkins);persist('squad-cube-skin-v3',skin);
    window.SquadSounds?.effect('success');updateShop();draw()
  });
  el('cube-quality').addEventListener('change',e=>{quality=e.target.value;localStorage.setItem('squad-cube-quality-v4',quality);draw()});
  el('cube-sound').addEventListener('click',()=>{
    sound=!sound;localStorage.setItem('squad-cube-sound-v2',sound?'1':'0');
    el('cube-sound').textContent=sound?'♫ Spielsounds an':'♪ Spielsounds aus';window.SquadSounds?.setEffects(sound);if(sound)window.SquadSounds?.effect('success');
  });
  el('arcade-jump-to-cube').addEventListener('click',()=>el('cube-panel').scrollIntoView({behavior:reducedMotion?'instant':'smooth',block:'start'}));
  document.addEventListener('keydown',e=>{
    if(!inArcade())return;
    const target=e.target?.tagName?.toLowerCase();
    if(['input','textarea','select'].includes(target)||e.target?.isContentEditable)return;
    if(e.target?.closest('button:not(#reaction-pad), a, summary, .cube-settings-accordion'))return;
    if(e.code==='Escape'&&(mode==='running'||mode==='paused')){e.preventDefault();pause(mode==='paused');return}
    if(!['Space','ArrowUp'].includes(e.code))return;
    e.preventDefault();
    // Reaction test only when it is focused, or when a trial is in progress.
    if(document.activeElement===el('reaction-pad')){if(!e.repeat)reactionTap()}
    else {inputHeld=true;if(!e.repeat)jump()}
  });
  document.addEventListener('keyup',e=>{if(['Space','ArrowUp'].includes(e.code))inputHeld=false});
  function leaving(){inputHeld=false;if(mode==='running')pause(false);if(reactionPhase==='waiting'||reactionPhase==='go')cancelReaction()}
  window.minigamesTabChanged=page=>{if(page!=='minigames')leaving();else requestAnimationFrame(resize)};
  document.addEventListener('visibilitychange',()=>{if(document.hidden)leaving()});
  window.addEventListener('blur',()=>{if(mode==='running')pause(false)});
  fetch('api/arcade').then(r=>r.ok?r.json():{}).then(saved=>{
    for(const [k,v] of Object.entries(saved))if(/^(squad-arcade-|squad-cube-)[a-z0-9-]+$/.test(k))localStorage.setItem(k,JSON.stringify(v));
    if(mode==='running'||mode==='paused'||mode==='revivable')return; // never overwrite a live run from an asynchronous load
    const sc=store(scoresKey,[]);if(Array.isArray(sc))reactionScores=sc.filter(x=>Number.isFinite(x)&&x>0&&x<15000).slice(-100);
    challengeBest=store(challengeKey,null);if(!Number.isFinite(challengeBest))challengeBest=null;
    for(const d of Object.keys(difficulties))bests[d]=Number(store(bestKey(d),0))||0;
    wallet=Math.max(0,Number(store('squad-cube-wallet-v3',0))||0);
    unlocked=Math.max(1,Math.min(10,Number(store('squad-cube-unlocked-v3',1))||1));
    unlockedSkins=store('squad-cube-skins-v3',['mint']);if(!Array.isArray(unlockedSkins))unlockedSkins=['mint'];
    skin=store('squad-cube-skin-v3','mint');if(!skins[skin]||!unlockedSkins.includes(skin))skin='mint';
    difficulty=store(settingKey,'normal');if(!difficulties[difficulty])difficulty='normal';
    playMode=store('squad-cube-play-mode-v3','endless');if(!playModes.includes(playMode))playMode='endless';
    stage=Math.max(1,Math.min(unlocked,Number(store('squad-cube-stage-v3',1))||1));
    startForm=store('squad-cube-start-form-v4','cube');if(!forms.includes(startForm))startForm='cube';el('cube-start-form').value=startForm;
    el('cube-stage').value=String(stage);el('cube-difficulty').value=difficulty;el('cube-play-mode').value=playMode;el('cube-skin').value=skin;
    updateReactionStats();updateShop();refreshCubeSummary();if(mode==='ready'||mode==='dead')reset();
  }).catch(()=>{});
  window.SquadSounds?.setEffects(sound);reset();requestAnimationFrame(resize);


  // Compact Cube Rush menus: keep the game screen uncluttered and show one
  // related group at a time. They never replace or delete saved game settings.
  const cubeMenus=[...document.querySelectorAll('.cube-setting-group')];
  cubeMenus.forEach(menu=>menu.addEventListener('toggle',()=>{
    if(!menu.open)return;
    cubeMenus.forEach(other=>{if(other!==menu)other.open=false});
    el('cube-open-setup')?.setAttribute('aria-expanded',String(el('cube-mode-details').open));
  }));
  el('cube-open-setup')?.addEventListener('click',()=>{
    const menu=el('cube-mode-details');menu.open=!menu.open;
    el('cube-open-setup').setAttribute('aria-expanded',String(menu.open));
    if(menu.open)menu.scrollIntoView({block:'nearest',behavior:reducedMotion?'instant':'smooth'});
  });
  const refreshCubeSummary=()=>{
    const current=id=>el(id).selectedOptions[0]?.textContent?.trim()||'';
    el('cube-selected-summary').textContent=[current('cube-play-mode'),current('cube-difficulty'),current('cube-start-form')].join('  ·  ');
    el('cube-stage').disabled=playMode==='endless';
    el('cube-stage').closest('label')?.classList.toggle('setting-disabled',playMode==='endless');
  };
  for(const id of ['cube-play-mode','cube-difficulty','cube-start-form','cube-stage'])el(id).addEventListener('change',refreshCubeSummary);
  refreshCubeSummary();

  // Readable only in explicitly opted-in automated tests. Never enabled in the shipped app.
  if (window.__SQUAD_TESTING__ === true) {
    window.__squadArcadeTest = Object.freeze({
      state: () => ({mode,playMode,playerForm,difficulty,elapsed,level,points,gems,wallet,revives,invincible,boundaryTime,checkpoint,rider:{...rider}}),
      step: (count=1) => {for(let i=0;i<Math.min(3000,count)&&mode==='running';i++)update(timestep);hud(true);draw()},
      crash: () => finish(),
      grantRunGems: count => {gems=clamp(Math.trunc(count),0,1000);hud(true)},
      addCoin: () => coins.push({x:rider.x+SIDE/2+speed*timestep,y:rider.y+SIDE/2,r:12,hit:false}),
      forceCheckpoint: () => {checkpoint={distance,elapsed,level,points,gems,collectedTotal,combo,maxCombo,spawnCount};lastCheckpoint=elapsed},
      setInvincible: n => {invincible=n}
    });
  }
  // Selectable, genuine TCP connection latency (not an ICMP ping or game ping).
  const presets=['cloudflare','google','quad9','minecraft','custom'];
  const selected=localStorage.getItem('squad-ping-target-v1');
  el('ping-target').value=presets.includes(selected)?selected:'cloudflare';
  el('ping-custom').value=localStorage.getItem('squad-ping-custom-v1')||'';
  function targetChanged(){el('ping-custom-wrap').hidden=el('ping-target').value!=='custom';localStorage.setItem('squad-ping-target-v1',el('ping-target').value)}
  el('ping-target').addEventListener('change',targetChanged);
  el('ping-custom').addEventListener('change',()=>localStorage.setItem('squad-ping-custom-v1',el('ping-custom').value.trim()));
  targetChanged();
  el('ping-test').addEventListener('click',async()=>{
    const custom=el('ping-custom').value.trim();
    try{
      el('ping-test').disabled=true;localStorage.setItem('squad-ping-custom-v1',custom);
      await api('ping',{target:el('ping-target').value,custom});
      el('ping-status').textContent='◉ Messe 5 TCP-Verbindungen …';
    }catch(e){notify(e.message,true);el('ping-test').disabled=false;el('ping-status').textContent='⚠ '+e.message}
  });
  window.renderTargetPing=p=>{
    if(!p)return;
    const show=v=>typeof v==='number'&&Number.isFinite(v)?v.toFixed(1)+' ms':'-- ms';
    el('ping-average').textContent=show(p.average);
    el('ping-minimum').textContent=show(p.minimum);
    el('ping-maximum').textContent=show(p.maximum);
    el('ping-success').textContent=(p.successful||0)+' / '+(p.attempts||5);
    el('ping-test').disabled=!!p.running;
    if(p.running)el('ping-status').textContent='◉ Test läuft: '+p.label+' ('+p.endpoint+')';
    else if(p.testedAt)el('ping-status').textContent=(p.error?'⚠ ':'✓ ')+p.label+' · '+p.endpoint+' · '+(p.error||'Test abgeschlossen')+' · '+new Date(p.testedAt).toLocaleTimeString('de-DE');
  };
})();
