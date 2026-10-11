/* SQUAD BASE 2.6 – lokale Design-Einstellungen und ressourcenschonende Szenen */
(()=>{'use strict';
 const KEY='squad.base.appearance.v1';
 const PALETTES={
  oak:{name:'Cozy Oak',hex:'#92dbaf',rgb:'146,219,175',blue:'#61bcfa',amber:'#eeba6a',purple:'#c997f4',contrast:'#102018'},
  violet:{name:'Violett',hex:'#bfa2ff',rgb:'191,162,255',blue:'#84baff',amber:'#ffcb81',purple:'#dcacff',contrast:'#171124'},
  ocean:{name:'Ocean',hex:'#66d9eb',rgb:'102,217,235',blue:'#70bcff',amber:'#f7cb82',purple:'#b8a1ef',contrast:'#09222a'},
  sunset:{name:'Sunset',hex:'#ffb676',rgb:'255,182,118',blue:'#6cbff1',amber:'#ffce72',purple:'#d9a0ed',contrast:'#23170d'},
  rose:{name:'Rose',hex:'#ff8faa',rgb:'255,143,170',blue:'#8ab7fd',amber:'#ffc17e',purple:'#d2a7ed',contrast:'#260d17'},
  ice:{name:'Ice Blue',hex:'#a4c5ff',rgb:'164,197,255',blue:'#92c7ff',amber:'#f5ca81',purple:'#c7aef6',contrast:'#0e1a32'}
 };
 const SCENES={oak:'Cozy Oak',aurora:'Aurora',stars:'Night Sky',rain:'Cozy Rain',embers:'Fireflies',snow:'Snowfall',clouds:'Cloud Drift',dusk:'Golden Hour'};
 const DEFAULT={color:'oak',scene:'oak',motion:true,speed:'slow',cozy:'cozy',brightness:100,dim:0,vignette:20,density:'normal'};
 function fromSaved(){let data={};try{const raw=localStorage.getItem(KEY);data=raw?JSON.parse(raw):{}}catch{}return normalize(data)}
 function clampInt(value,min,max,fallback){const n=Number(value);return Number.isFinite(n)&&n>=min&&n<=max?Math.round(n):fallback}
 function normalize(x){return{color:PALETTES[x?.color]?x.color:'oak',scene:SCENES[x?.scene]?x.scene:'oak',motion:x?.motion!==false,speed:['slow','normal','fast'].includes(x?.speed)?x.speed:'slow',cozy:['classic','cozy','extra'].includes(x?.cozy)?x.cozy:'cozy',brightness:clampInt(x?.brightness,55,140,100),dim:clampInt(x?.dim,0,65,0),vignette:clampInt(x?.vignette,0,65,20),density:['low','normal','high'].includes(x?.density)?x.density:'normal'}}
 let settings=fromSaved(),anim=0,last=0,particles=[],w=0,h=0,dpr=1;
 let changedByUser=false,persistTimer=0,saving=false,pending=false;
 const canvas=document.getElementById('squad-background-fx'),ctx=canvas?.getContext('2d',{alpha:true});
 const body=document.body,root=document.documentElement;
 const reduce=window.matchMedia?.('(prefers-reduced-motion: reduce)');
 const hasReducedMotion=()=>!!reduce?.matches;
 const animate=()=>settings.motion&&!hasReducedMotion()&&!document.body.classList.contains('v42-smart-active')&&settings.scene!=='oak'&&!document.hidden;
 function apply(){const p=PALETTES[settings.color];
   root.style.setProperty('--green',p.hex);root.style.setProperty('--theme-primary',p.hex);
   root.style.setProperty('--theme-rgb',p.rgb);root.style.setProperty('--theme-contrast',p.contrast);
   root.style.setProperty('--theme-aurora-duration',settings.speed==='fast'?'12s':settings.speed==='normal'?'22s':'34s');
   root.style.setProperty('--blue',p.blue);root.style.setProperty('--amber',p.amber);root.style.setProperty('--purple',p.purple);
   body.dataset.scene=settings.scene;body.dataset.motion=settings.motion?'on':'off';body.dataset.cozy=settings.cozy;
   root.style.setProperty('--scene-brightness',String(settings.brightness/100));
   root.style.setProperty('--scene-dim',String(settings.dim/100));
   root.style.setProperty('--scene-vignette',String(settings.vignette/100));
   for(const e of document.querySelectorAll('button[data-cozy]'))e.setAttribute('aria-pressed',String(e.dataset.cozy===settings.cozy));
   for(const e of document.querySelectorAll('[data-color]'))e.setAttribute('aria-pressed',String(e.dataset.color===settings.color));
   for(const e of document.querySelectorAll('button[data-scene]'))e.setAttribute('aria-pressed',String(e.dataset.scene===settings.scene));
   const a=document.getElementById('theme-animate'),s=document.getElementById('theme-speed'),t=document.getElementById('theme-status');
   if(a)a.checked=settings.motion;if(s)s.value=settings.speed;
   for(const [key,suffix] of [['brightness','%'],['dim','%'],['vignette','%']]){
     const input=document.getElementById('theme-'+key),output=document.getElementById('theme-'+key+'-value');
     if(input)input.value=String(settings[key]);if(output)output.textContent=settings[key]+suffix;
   }
   const density=document.getElementById('theme-density');if(density)density.value=settings.density;
   if(t)t.textContent=PALETTES[settings.color].name+' · '+SCENES[settings.scene]+' · '+({classic:'Classic',cozy:'Cozy',extra:'Extra Cozy'}[settings.cozy])+(settings.motion?'':' · ohne Animation');
   if(canvas)canvas.style.display=animate()?'block':'none';
   // Neu aufsetzen verhindert sichtbare Sprünge nach einem Szenenwechsel.
   resetParticles();if(anim)cancelAnimationFrame(anim);anim=0;last=0;
   if(animate()&&ctx)anim=requestAnimationFrame(frame);
   window.dispatchEvent(new Event('squad-appearance-changed'));
 }
 async function flushSave(){
   if(saving)return;
   saving=true;
   while(pending){
     pending=false;
     const value={...settings};
     try{
       const response=await fetch('api/appearance',{method:'POST',headers:{'Content-Type':'application/json','X-Squad-Base':'1'},body:JSON.stringify(value)});
       if(!response.ok)throw Error('HTTP '+response.status);
     }catch{
       const status=document.getElementById('theme-status');
       if(status)status.textContent+=' · Speichern derzeit nicht möglich';
     }
   }
   saving=false;
 }
 function save(){changedByUser=true;try{localStorage.setItem(KEY,JSON.stringify(settings))}catch{}apply();pending=true;clearTimeout(persistTimer);persistTimer=setTimeout(flushSave,280)}
 async function loadFromPC(){
   try{
     const r=await fetch('api/appearance');if(!r.ok)return;
     const remote=await r.json();if(changedByUser)return;
     settings=normalize(remote);
     try{localStorage.setItem(KEY,JSON.stringify(settings))}catch{}
     apply();
   }catch{/* Lokales Farbschema bleibt als Fallback erhalten. */}
 }
 function resize(){if(!canvas||!ctx)return;w=Math.max(1,innerWidth);h=Math.max(1,innerHeight);dpr=Math.min(1.5,window.devicePixelRatio||1);canvas.width=Math.round(w*dpr);canvas.height=Math.round(h*dpr);ctx.setTransform(dpr,0,0,dpr,0,0);resetParticles()}
 function random(min,max){return min+Math.random()*(max-min)}
 function resetParticles(){if(!ctx)return;const mode=settings.scene;const base=mode==='rain'?50:mode==='snow'?52:mode==='clouds'?17:mode==='stars'?65:mode==='embers'?36:0;const n=Math.round(base*({low:.45,normal:1,high:1.6}[settings.density]||1));
  particles=Array.from({length:n},()=>({x:Math.random()*w,y:Math.random()*h,size:random(.55,mode==='embers'?3.5:2),drift:random(.15,1),alpha:random(.2,.75),phase:random(0,6.28)}));}
 function frame(now){anim=0;if(!animate()||!ctx)return;
  const minMs=settings.speed==='fast'?24:settings.speed==='normal'?33:45;
  if(now-last<minMs){anim=requestAnimationFrame(frame);return}
  const dt=Math.min(2,(now-last)/33||1);last=now;
  ctx.clearRect(0,0,w,h);
  const mode=settings.scene;
  if(mode==='rain'||mode==='snow'){
   ctx.lineWidth=mode==='snow'?1.5:.8;ctx.strokeStyle='rgba(149,200,225,.26)';ctx.beginPath();
   for(const p of particles){p.y+=p.drift*(mode==='snow'?1.25:4)*dt;p.x+=(mode==='snow'?Math.sin(p.phase)*.24:-p.drift*.6)*dt;if(p.y>h+24){p.y=-30;p.x=Math.random()*w}if(p.x<0)p.x=w;
     if(mode==='snow'){ctx.moveTo(p.x+p.size,p.y);ctx.arc(p.x,p.y,p.size*1.2,0,Math.PI*2)}else{ctx.moveTo(p.x,p.y);ctx.lineTo(p.x-4,p.y+15+p.size*3)}}
   if(mode==='snow'){ctx.fillStyle='rgba(212,235,255,.58)';ctx.fill()}else ctx.stroke();
  }else if(mode==='clouds'){
   for(const p of particles){p.x+=p.drift*.35*dt;if(p.x>w+100)p.x=-110;ctx.fillStyle=`rgba(140,184,207,${p.alpha*.047})`;ctx.beginPath();ctx.ellipse(p.x,p.y,Math.max(30,p.size*30),Math.max(14,p.size*15),0,0,6.283);ctx.fill()}
  }else if(mode==='stars'||mode==='embers'){
   for(const p of particles){p.phase+=.012*dt;p.y+=(mode==='embers'?-1:1)*p.drift*.2*dt;p.x+=Math.sin(p.phase)*.15*dt;
    if(p.y<-12)p.y=h+12;if(p.y>h+12)p.y=-12;
    const a=Math.max(.07,p.alpha*(.75+.25*Math.sin(p.phase*1.5)));ctx.beginPath();ctx.arc(p.x,p.y,p.size,0,6.283);
    ctx.fillStyle=mode==='embers'?`rgba(255,179,87,${a})`:`rgba(194,220,255,${a})`;ctx.fill();
    if(mode==='embers'){ctx.beginPath();ctx.arc(p.x,p.y,p.size*3,0,6.283);ctx.fillStyle=`rgba(255,169,88,${a*.065})`;ctx.fill()}
   }
  }
  anim=requestAnimationFrame(frame);
 }
 document.addEventListener('click',e=>{
  const color=e.target.closest('button[data-color]'),scene=e.target.closest('button[data-scene]'),cozy=e.target.closest('button[data-cozy]');
  if(cozy&&['classic','cozy','extra'].includes(cozy.dataset.cozy)){settings.cozy=cozy.dataset.cozy;save()}
  else if(color&&PALETTES[color.dataset.color]){settings.color=color.dataset.color;save()}
  else if(scene&&SCENES[scene.dataset.scene]){settings.scene=scene.dataset.scene;save()}
  else if(e.target.closest('#theme-reset')){settings={...DEFAULT};save()}
 });
 document.getElementById('theme-animate')?.addEventListener('change',e=>{settings.motion=!!e.target.checked;save()});
 document.getElementById('theme-speed')?.addEventListener('change',e=>{settings.speed=e.target.value;save()});
 for(const key of ['brightness','dim','vignette'])document.getElementById('theme-'+key)?.addEventListener('input',e=>{settings[key]=Number(e.target.value);save()});
 document.getElementById('theme-density')?.addEventListener('change',e=>{settings.density=e.target.value;save()});
 document.addEventListener('visibilitychange',()=>{if(!document.hidden){last=0;apply()}else {if(anim)cancelAnimationFrame(anim);anim=0;ctx?.clearRect(0,0,w,h)}});
 window.addEventListener('squad-smart-mode',apply);
 window.addEventListener('resize',resize,{passive:true});
 reduce?.addEventListener?.('change',apply);
 resize();apply();loadFromPC();
 window.SquadAppearance={get:()=>({...settings}),set:(o)=>{settings=normalize({...settings,...o});save()},palettes:Object.keys(PALETTES),scenes:Object.keys(SCENES)};
})();
