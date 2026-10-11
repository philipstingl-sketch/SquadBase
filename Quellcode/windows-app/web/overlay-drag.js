'use strict';
// Dragging the overlay's header updates native window coordinates. No process
// injection and no polling while dragging. Click-through disables interaction.
(() => {
 const header=document.querySelector('header');if(!header)return;
 let active=null,clickThrough=false,x=0,y=0;
 async function load(){try{const r=await fetch('api/overlay',{cache:'no-store'});const d=await r.json();clickThrough=!!d.config?.clickThrough;x=d.config?.x||0;y=d.config?.y||0;header.title=clickThrough?'Klick-durchlassen aktiv – Verschieben in SQUAD BASE':'Zum Verschieben ziehen';}catch{}}
 header.addEventListener('pointerdown',e=>{
  if(clickThrough||e.button!==0||e.target.closest('button'))return;
  active={screenX:e.screenX,screenY:e.screenY,x,y};header.setPointerCapture(e.pointerId);
  document.body.classList.add('dragging');e.preventDefault();
 });
 header.addEventListener('pointermove',e=>{
  if(!active)return;
  const dx=e.screenX-active.screenX,dy=e.screenY-active.screenY;
  document.querySelector('.glass').style.transform=`translate(${dx}px,${dy}px)`;
 });
 async function finish(e){
  if(!active)return;
  const base=active;active=null;document.body.classList.remove('dragging');
  document.querySelector('.glass').style.transform='';
  x=Math.max(0,Math.min(8000,Math.round(base.x+e.screenX-base.screenX)));
  y=Math.max(0,Math.min(8000,Math.round(base.y+e.screenY-base.screenY)));
  try{const r=await fetch('api/overlay/position',{method:'POST',headers:{'Content-Type':'application/json','X-Squad-Base':'1'},body:JSON.stringify({x,y})});if(!r.ok)throw Error('Konnte Position nicht speichern')}catch{await load()}
 }
 header.addEventListener('pointerup',finish);
 header.addEventListener('pointercancel',()=>{active=null;document.querySelector('.glass').style.transform='';document.body.classList.remove('dragging')});
 load();window.setInterval(load,20000);
})();
