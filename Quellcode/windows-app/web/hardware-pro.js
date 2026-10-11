'use strict';
(() => {
 const $=id=>document.getElementById(id),canvas=$('hw-pro-chart');if(!canvas)return;
 const ctx=canvas.getContext('2d');const data=[],enabled=new Set(['cpu','gpu','ram','gpuTemp']);
 let range=180,lastSample=0;
 const color=()=>{const s=getComputedStyle(document.documentElement);return {cpu:s.getPropertyValue('--green').trim()||'#92dbaa',gpu:'#efb976',ram:'#73bdff',gpuTemp:'#df88ec'}};
 function draw(){
  let r=canvas.getBoundingClientRect(),dpr=Math.min(devicePixelRatio||1,2);
  if(r.width<2)return;
  const w=Math.max(220,Math.round(r.width*dpr)),h=Math.round(220*dpr);
  if(canvas.width!==w||canvas.height!==h){canvas.width=w;canvas.height=h}
  ctx.clearRect(0,0,w,h);const l=36,t=13,rgt=w-10,b=h-28;
  ctx.font=`${11*dpr}px Segoe UI`;for(let q=0;q<=100;q+=25){const y=b-(b-t)*q/100;ctx.beginPath();ctx.setLineDash([3*dpr,6*dpr]);ctx.moveTo(l,y);ctx.lineTo(rgt,y);ctx.strokeStyle='#a8efd228';ctx.stroke();ctx.setLineDash([]);ctx.fillStyle='#a5c3b4';ctx.textAlign='right';ctx.fillText(q+'',l-7,y+4*dpr)}
  if(!data.length){ctx.fillStyle='#b2cbbb';ctx.textAlign='center';ctx.fillText('Warte auf Windows-Messwerte …',w/2,h/2);return}
  const now=data.at(-1).t,min=now-range*1000,colors=color();for(const key of enabled){ctx.beginPath();let active=false;
   for(const point of data){if(point.t<min)continue;const val=point[key];if(typeof val!=='number'||!Number.isFinite(val)){active=false;continue}
    const x=l+(point.t-min)/(range*1000)*(rgt-l),y=b-Math.min(100,Math.max(0,val))*(b-t)/100;
    if(active)ctx.lineTo(x,y);else ctx.moveTo(x,y);active=true;
   }ctx.strokeStyle=colors[key];ctx.lineWidth=2*dpr;ctx.stroke();
  }
  ctx.font=`${10*dpr}px Segoe UI`;ctx.fillStyle='#a5c3b4';ctx.textAlign='left';ctx.fillText('vor '+Math.round(range/60)+' Min.',l,h-8*dpr);ctx.textAlign='right';ctx.fillText('jetzt',rgt,h-8*dpr)
 }
 window.SquadHardwarePro=h=>{
  if(!h)return;
  $('pro-vram').textContent=typeof h.gpuVramUsed==='number'&&typeof h.gpuVramTotal==='number'?h.gpuVramUsed+' / '+h.gpuVramTotal+' MiB':'Nicht verfügbar';
  $('pro-fan').textContent=typeof h.gpuFan==='number'?h.gpuFan+' %':'Nicht verfügbar';
  $('pro-disk').textContent=typeof h.diskFreeGB==='number'&&typeof h.diskTotalGB==='number'?h.diskFreeGB.toFixed(1)+' / '+h.diskTotalGB.toFixed(1)+' GB frei/gesamt':'Nicht verfügbar';
  $('pro-fps').textContent=typeof h.fps==='number'?h.fps.toFixed(0)+' FPS':'Keine Messquelle';
  $('pro-cpu-temp').textContent=typeof h.cpuTemp==='number'?h.cpuTemp+' °C':'Nicht verfügbar';
  $('pro-gpu-temp').textContent=typeof h.gpuTemp==='number'?h.gpuTemp+' °C':'Nicht verfügbar';
  const now=Date.now();if(now-lastSample<2500)return;lastSample=now;
  data.push({t:now,cpu:h.cpu,gpu:h.gpu,ram:h.ram,gpuTemp:h.gpuTemp});while(data.length>205||data[0].t<now-600000)data.shift();draw()
 };
 $('pro-range').addEventListener('change',e=>{range=Number(e.target.value)||180;draw()});
 document.querySelectorAll('[data-hw-field]').forEach(e=>e.addEventListener('change',()=>{if(e.checked)enabled.add(e.dataset.hwField);else enabled.delete(e.dataset.hwField);draw()}));
 if(typeof ResizeObserver!=='undefined')new ResizeObserver(draw).observe(canvas);window.addEventListener('squad-appearance-changed',draw);
  if(typeof state!=='undefined' && state?.hardware)window.SquadHardwarePro(state.hardware);
})();
