/* Liquid Glass 4.0 — opt-in settings saved by the existing per-user backend. */
'use strict';
(() => {
 const byId=id=>document.getElementById(id);
 const area=byId('v45-glass-studio');if(!area)return;
 const profileButtons=[...area.querySelectorAll('[data-v45-profile]')];
 const defaults={preset:'original',blur:18,opacity:68,radius:20,glow:true,ambient:true};
 let persisted={...defaults};let draft={...defaults};let busy=false;
 const status=(s,error=false)=>{const e=byId('v45-status');e.textContent=s;e.classList.toggle('v45-error',error)};
 function preview(config){
   if(config.preset==='original')delete document.body.dataset.v45Profile;
   else document.body.dataset.v45Profile=config.preset;
   document.body.dataset.v45Glow=String(config.glow);
   document.body.dataset.v45Ambient=String(config.ambient);
   document.body.style.setProperty('--v45-blur',config.blur+'px');
   document.body.style.setProperty('--v45-radius',config.radius+'px');
   document.body.style.setProperty('--v45-opacity-decimal',String(config.opacity/100));
   for(const b of profileButtons)b.setAttribute('aria-pressed',String(b.dataset.v45Profile===config.preset));
   for(const field of ['blur','opacity','radius'])byId('v45-'+field).value=String(config[field]);
   byId('v45-blur-label').textContent=config.blur+' px';
   byId('v45-opacity-label').textContent=config.opacity+' %';
   byId('v45-radius-label').textContent=config.radius+' px';
   byId('v45-glow').checked=config.glow;
   byId('v45-ambient').checked=config.ambient;
 }
 function modified(){return JSON.stringify(draft)!==JSON.stringify(persisted)}
 for(const b of profileButtons)b.addEventListener('click',()=>{draft.preset=b.dataset.v45Profile;preview(draft);status('Vorschau aktiv · zum Übernehmen speichern.')});
 for(const field of ['blur','opacity','radius'])byId('v45-'+field).addEventListener('input',e=>{draft[field]=Number(e.target.value);preview(draft);status('Vorschau aktiv · zum Übernehmen speichern.')});
 for(const field of ['glow','ambient'])byId('v45-'+field).addEventListener('change',e=>{draft[field]=e.target.checked;preview(draft);status('Vorschau aktiv · zum Übernehmen speichern.')});
 byId('v45-undo').addEventListener('click',()=>{draft={...persisted};preview(draft);status('Nicht gespeicherte Änderungen verworfen.')});
 byId('v45-save').addEventListener('click',async()=>{
   if(busy)return;busy=true;byId('v45-save').disabled=true;
   try{const r=await fetch('api/glass45',{method:'POST',headers:{'Content-Type':'application/json','X-Squad-Base':'1'},body:JSON.stringify(draft)});const j=await r.json();if(!r.ok||!j.saved)throw Error(j.error||'Speichern fehlgeschlagen');persisted={...draft};status('✓ Gespeichert. Beim nächsten Start wieder verfügbar.');}
   catch(e){status('Konnte Design nicht speichern: '+e.message,true)}finally{busy=false;byId('v45-save').disabled=false}
 });
 // Fetch only after listeners are wired; local appearance survives random UI ports.
 fetch('api/glass45',{headers:{'X-Squad-Base':'1'}}).then(async r=>{const j=await r.json();if(!r.ok)throw Error(j.error||'Keine Einstellungen');return j}).then(j=>{
  if(!['original','moonstone','boreal','rosewood','starlight'].includes(j.preset))throw Error('Ungültige Einstellungen');
  persisted={...defaults,...j};draft={...persisted};preview(draft);status('Dein Design wurde aus dem Windows-Profil geladen.');
 }).catch(e=>{preview(draft);status('Design nur als Vorschau verfügbar: '+e.message,true)});
 // Block losing draft without explicitly restoring on a reload in the same browser.
 window.addEventListener('pagehide',()=>{if(modified())preview(persisted)});
})();
