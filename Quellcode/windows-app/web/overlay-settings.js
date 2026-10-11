'use strict';
// Unified control for the native overlay: preview, presets, options and persistence.
(() => {
 const $=id=>document.getElementById(id);
 const options=['cpu','gpu','ram','gpuTemp','fps','ping','music','friends','clock','clickThrough','compact','minimalHeader'];
 let working=false,loaded=false,visible=false;
 const message=s=>{const e=$('overlay-message');if(e)e.textContent=s};
 const request=async(route,payload)=>{
  const r=await fetch('api/'+route,{method:payload===undefined?'GET':'POST',headers:payload===undefined?{}:{'Content-Type':'application/json','X-Squad-Base':'1'},body:payload===undefined?undefined:JSON.stringify(payload),cache:'no-store'});
  let d={};try{d=await r.json()}catch{}if(!r.ok)throw Error(d.error||'Verbindung fehlgeschlagen');return d;
 };
 function values(){return {enabled:true,opacity:Number($('overlay-opacity').value),scale:Number($('overlay-scale').value),x:Number($('overlay-x').value),y:Number($('overlay-y').value),...Object.fromEntries(options.map(k=>[k,!!document.querySelector('[data-ov="'+k+'"]').checked]))};}
 function draw(){const c=values(),pv=$('overlay-preview');if(!pv)return;
  pv.classList.toggle('v4-preview-compact',c.compact);pv.classList.toggle('v4-preview-minimal',c.minimalHeader);pv.style.opacity=String(c.opacity/100);pv.style.setProperty('--preview-size',String(c.scale/100));
  for(const k of ['cpu','gpu','ram','fps','ping']){const el=$('preview-'+k);if(el)el.hidden=!c[k]}
  const extras=[];if(c.music)extras.push('♫ Musik');if(c.friends)extras.push('☻ Freunde');if(c.clock)extras.push('◷ Uhr');if(c.gpuTemp)extras.push('♨ GPU °C');
  const e=$('overlay-preview-extras');e.textContent=extras.join(' · ');e.hidden=!extras.length;
  ['opacity','scale'].forEach(k=>$('overlay-'+k+'-val').textContent=c[k]+'%');
  document.querySelectorAll('[data-overlay-position]').forEach(b=>{const pos=b.dataset.overlayPosition; const right=c.x>200, bottom=c.y>200;b.setAttribute('aria-pressed',String(pos===(bottom?'b':'t')+(right?'r':'l')))});
  const selected=Object.entries({mini:['cpu','gpu','clock'],performance:['cpu','gpu','ram','gpuTemp','fps','ping','clock'],social:['music','friends','clock','cpu'],music:['music','clock'],music:['music','clock']}).find(([,items])=>options.filter(k=>!['clickThrough','compact','minimalHeader'].includes(k)).every(k=>c[k]===items.includes(k)));
  document.querySelectorAll('[data-overlay-preset]').forEach(b=>b.setAttribute('aria-pressed',String(selected?.[0]===b.dataset.overlayPreset)));
 }
 function applyConfig(c){['opacity','scale','x','y'].forEach(k=>{if($( 'overlay-'+k))$('overlay-'+k).value=c[k]});options.forEach(k=>{const e=document.querySelector('[data-ov="'+k+'"]');if(e)e.checked=!!c[k]});draw()}
 async function load(){try{const d=await request('overlay');applyConfig(d.config);visible=!!d.visible;$('overlay-toggle').textContent=visible?'▧ Overlay ausblenden':'▧ Overlay starten';message(d.supported?'Bereit · Strg + Umschalt + O zum Ein-/Ausblenden':'Overlay benötigt Microsoft Edge im App-Modus.');loaded=true}catch(e){message('⚠ '+e.message)}}
 $('overlay-settings')?.querySelectorAll('input').forEach(e=>e.addEventListener('input',draw));
 document.querySelectorAll('[data-overlay-preset]').forEach(b=>b.addEventListener('click',()=>{
  const keys={mini:['cpu','gpu','clock'],performance:['cpu','gpu','ram','gpuTemp','fps','ping','clock'],social:['music','friends','clock','cpu'],music:['music','clock'],music:['music','clock']}[b.dataset.overlayPreset];if(!keys)return;
  options.filter(k=>!['clickThrough','compact','minimalHeader'].includes(k)).forEach(k=>document.querySelector('[data-ov="'+k+'"]').checked=keys.includes(k));draw();message('Layout gewählt. Mit „Einstellungen speichern“ übernehmen.');
 }));
 document.querySelectorAll('[data-overlay-position]').forEach(b=>b.addEventListener('click',()=>{
  const pos=b.dataset.overlayPosition,sc=Number($('overlay-scale').value)/100;
  const safeWidth=Math.max(640,window.screen?.availWidth||1280),safeHeight=Math.max(480,window.screen?.availHeight||720);
  $('overlay-x').value=pos.endsWith('r')?Math.max(0,Math.floor(safeWidth-440*sc-20)):18;
  $('overlay-y').value=pos.startsWith('b')?Math.max(0,Math.floor(safeHeight-300*sc-90)):18;
  draw();message('Position gesetzt. Mit „Einstellungen speichern“ übernehmen.');
 }));
 $('overlay-save')?.addEventListener('click',async()=>{if(working)return;working=true;try{await request('overlay',values());message('✓ Overlay-Einstellungen gespeichert.')}catch(e){message('⚠ '+e.message)}finally{working=false}});
 $('overlay-toggle')?.addEventListener('click',async()=>{if(working)return;working=true;try{
  await request('overlay',values());const d=await request('overlay/toggle',{});visible=!!d.visible;
  $('overlay-toggle').textContent=visible?'▧ Overlay ausblenden':'▧ Overlay starten';message(visible?'✓ Overlay läuft · Strg + Umschalt + O zum Ausblenden':'Overlay ausgeblendet');
 }catch(e){message('⚠ '+e.message)}finally{working=false}});
 load();
 window.SquadOverlaySettings={values,draw,loaded:()=>loaded};
})();
