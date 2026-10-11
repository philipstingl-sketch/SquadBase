'use strict';
// SQUAD BASE v3.5. Local, user-configurable clock with genuine flip-style digits.
(() => {
 const $=id=>document.getElementById(id);
 const KEY='squad.base.clock.v1';
 const DEFAULT={style:'digital',seconds:true,twentyFour:true,date:true,weekday:true,session:true,localBadge:true};
 const fields={seconds:'clock-seconds',twentyFour:'clock-24h',date:'clock-date',weekday:'clock-weekday',session:'clock-session',localBadge:'clock-local'};
 const begin=Date.now();let cfg={...DEFAULT};let changed=false,timer=0;
 const norm=o=>({
  style:['digital','flip','minimal'].includes(o?.style)?o.style:'digital',
  ...Object.fromEntries(Object.keys(fields).map(k=>[k,typeof o?.[k]==='boolean'?o[k]:DEFAULT[k]]))
 });
 const status=s=>{const e=$('clock-settings-status');if(e)e.textContent=s};
 function draw(){
  const now=new Date(), hour=cfg.twentyFour?now.getHours():(now.getHours()%12||12);
  const h=String(hour).padStart(2,'0'),m=String(now.getMinutes()).padStart(2,'0'),ss=String(now.getSeconds()).padStart(2,'0');
  const suffix=cfg.twentyFour?'':(now.getHours()<12?' AM':' PM');
  const text=h+':'+m+(cfg.seconds?':'+ss:'')+suffix;
  if($('flip-ampm')){$('flip-ampm').hidden=cfg.twentyFour;$('flip-ampm').textContent=suffix.trim();}
  const shell=$('clock-shell');if(!shell)return;
  shell.className='clock clock-'+cfg.style+(cfg.seconds?'':' no-seconds');
  shell.setAttribute('aria-label','Lokale Uhrzeit '+text);
  $('clock').textContent=text;
  const values={h1:h[0],h2:h[1],m1:m[0],m2:m[1],s1:ss[0],s2:ss[1]};
  Object.keys(values).forEach(k=>{const el=document.querySelector('[data-flip="'+k+'"]');if(el&&el.textContent!==values[k]){el.textContent=values[k];if(!window.matchMedia('(prefers-reduced-motion: reduce)').matches){el.classList.remove('flip-turn');void el.offsetWidth;el.classList.add('flip-turn')}}});
  const dateBits=[];
  if(cfg.weekday)dateBits.push(new Intl.DateTimeFormat('de-DE',{weekday:'long'}).format(now));
  if(cfg.date)dateBits.push(new Intl.DateTimeFormat('de-DE',{day:'2-digit',month:'long',year:'numeric'}).format(now));
  $('date').textContent=dateBits.join(' · ');
  $('date').hidden=!dateBits.length;
  $('header-session').hidden=!cfg.session;
  if(cfg.session){const min=Math.max(0,Math.floor((Date.now()-begin)/60000));$('header-session').textContent='Session '+(min>=60?Math.floor(min/60)+' Std. '+min%60+' Min.':min+' Min.');}
  $('header-local').hidden=!cfg.localBadge;
  $('header-context').hidden=!(cfg.session||cfg.localBadge);
 }
 function paintInputs(){
  Object.entries(fields).forEach(([key,id])=>{if($(id))$(id).checked=cfg[key]});
  document.querySelectorAll('[data-clock-style]').forEach(e=>e.setAttribute('aria-pressed',String(e.dataset.clockStyle===cfg.style)));
  draw();
 }
 async function persist(){
  try{localStorage.setItem(KEY,JSON.stringify(cfg))}catch{}
  status('Speichere Uhreinstellungen …');
  try{const r=await fetch('api/clock',{method:'POST',headers:{'Content-Type':'application/json','X-Squad-Base':'1'},body:JSON.stringify(cfg)});if(!r.ok)throw Error('HTTP '+r.status);status('✓ Auf diesem PC gespeichert.')}catch{status('⚠ Speichern in Windows nicht möglich; Einstellung gilt vorerst in dieser Sitzung.');}
 }
 function choose(next){cfg=norm({...cfg,...next});changed=true;paintInputs();clearTimeout(timer);timer=setTimeout(persist,300)}
 try{const raw=localStorage.getItem(KEY);if(raw)cfg=norm(JSON.parse(raw))}catch{}
 document.querySelectorAll('[data-clock-style]').forEach(e=>e.addEventListener('click',()=>choose({style:e.dataset.clockStyle})));
 Object.entries(fields).forEach(([key,id])=>$(id)?.addEventListener('change',e=>choose({[key]:e.target.checked})));
 $('clock-reset')?.addEventListener('click',()=>choose({...DEFAULT}));
 paintInputs();setInterval(draw,1000);
 (async()=>{try{const r=await fetch('api/clock',{cache:'no-store'});if(!r.ok)return;const value=norm(await r.json());if(changed)return;cfg=value;try{localStorage.setItem(KEY,JSON.stringify(cfg))}catch{}paintInputs()}catch{}})();
 window.SquadClock={get:()=>({...cfg}),tick:draw};
})();
