'use strict';
// Honest charts: populated exclusively by the actual /api/state hardware samples.
(() => {
  const c=document.getElementById('hardware-chart');
  const ctx=c.getContext('2d');
  const samples=[];
  const graphColors=()=>{const css=getComputedStyle(document.documentElement);return{cpu:css.getPropertyValue('--green').trim()||'#91e9b1',ram:css.getPropertyValue('--blue').trim()||'#73c4ff',gpu:css.getPropertyValue('--amber').trim()||'#f3bd75'}};
  const clamp=(v,a,b)=>Math.min(b,Math.max(a,v));
  function resize(){
    const bounds=c.getBoundingClientRect();if(bounds.width<2)return;
    const dpr=Math.min(devicePixelRatio||1,2),w=Math.round(bounds.width*dpr),h=Math.round(182*dpr);
    if(c.width!==w||c.height!==h){c.width=w;c.height=h}
    draw();
  }
  function draw(){
    const w=c.width,h=c.height;if(w<2||h<2)return;
    ctx.clearRect(0,0,w,h);
    const left=36,top=12,right=w-10,bottom=h-28,chartH=bottom-top;
    ctx.font=`${Math.max(9,Math.round(h/18))}px Segoe UI, sans-serif`;
    for(let v=0;v<=100;v+=25){
      const y=bottom-chartH*v/100;
      ctx.beginPath();ctx.setLineDash([4,5]);ctx.moveTo(left,y);ctx.lineTo(right,y);
      ctx.strokeStyle='#96c7aa27';ctx.lineWidth=1;ctx.stroke();ctx.setLineDash([]);
      ctx.fillStyle='#9db5a7';ctx.textAlign='right';ctx.fillText(v+'%',left-6,y+3);
    }
    ctx.fillStyle='#aacbbb';ctx.textAlign='left';ctx.fillText('vor 3 Min.',left,bottom+19);ctx.textAlign='right';ctx.fillText('jetzt',right,bottom+19);
    if(!samples.length){ctx.textAlign='center';ctx.fillStyle='#a5b9a9';ctx.fillText('Warte auf echte Windows-Messwerte …',w/2,h/2);return}
    const colors=graphColors();
    // Rolling 180-second window. A missing GPU sample never produces fabricated zeros.
    const end=samples.at(-1).t,from=end-180000;
    for(const field of ['cpu','ram','gpu']){
      let active=false;
      ctx.beginPath();
      for(const s of samples){
        const v=s[field];if(v===null||!Number.isFinite(v)){active=false;continue}
        const x=left+clamp((s.t-from)/180000,0,1)*(right-left),y=bottom-clamp(v,0,100)/100*chartH;
        if(!active)ctx.moveTo(x,y);else ctx.lineTo(x,y);active=true;
      }
      ctx.strokeStyle=colors[field];ctx.lineWidth=Math.max(1.5,h/96);ctx.lineJoin='round';ctx.lineCap='round';ctx.stroke();
    }
  }
  let lastSample=0;
  window.SquadHardwareChart=(h)=>{
    if(!h)return;
    const now=Date.now();if(now-lastSample<2400)return;lastSample=now;
    const n=v=>typeof v==='number'&&Number.isFinite(v)?clamp(v,0,100):null;
    const entry={t:now,cpu:n(h.cpu),ram:n(h.ram),gpu:n(h.gpu)};
    samples.push(entry);
    while(samples.length>65||samples[0]?.t<now-180000)samples.shift();
    const time=document.getElementById('trend-time');time.textContent=`${samples.length} Messpunkte · ${new Date(now).toLocaleTimeString('de-DE')}`;
    draw();
  };
  if(typeof ResizeObserver!=='undefined')new ResizeObserver(resize).observe(c);
  window.addEventListener('resize',resize);
  window.addEventListener('squad-appearance-changed',draw);
  if (typeof state !== "undefined" && state?.hardware) window.SquadHardwareChart(state.hardware);
  setTimeout(resize,250);
})();
