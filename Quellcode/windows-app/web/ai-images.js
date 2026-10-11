'use strict';
// Connects exclusively to the real local image engine via the Squad Base backend.
// No generated placeholders, API keys, or private prompts sent to external services.
(() => {
 const el=id=>document.getElementById(id);
 const gallery=el('ai-image-gallery'),preview=el('ai-image-preview'),status=el('ai-image-status');
 const url=id=>'api/ai-image/file?id='+encodeURIComponent(id);
 function showImage(name){
   preview.replaceChildren();
   const img=document.createElement('img');img.src=url(name);img.alt='Von lokaler KI erzeugtes Bild';
   const link=document.createElement('a');link.href=url(name);link.download=name;link.textContent='↓ PNG speichern';
   preview.append(img,link);
 }
 async function refresh(){
   try{
    const data=await api('ai-image/gallery');gallery.replaceChildren();
    for(const item of data.images||[]){
     const link=document.createElement('a');link.href=url(item.name);link.download=item.name;link.title=new Date(item.created).toLocaleString('de-DE');
     const img=document.createElement('img');img.src=url(item.name);img.loading='lazy';img.alt='KI-Bild';link.append(img);
     link.addEventListener('click',e=>{if(!e.ctrlKey){e.preventDefault();showImage(item.name)}});gallery.append(link);
    }
    if(!gallery.children.length){const p=document.createElement('p');p.className='hint';p.textContent='Noch keine Bilder erstellt.';gallery.append(p)}
   }catch(e){status.textContent='Galerie nicht erreichbar: '+e.message}
 }
 el('ai-image-form').addEventListener('submit',async ev=>{
   ev.preventDefault();const btn=el('ai-image-create');if(btn.disabled)return;
   const prompt=el('ai-image-prompt').value.trim();const style=el('ai-image-style').value;const landscape=el('ai-image-format').value==='768';
   if(prompt.length<3)return;
   btn.disabled=true;btn.textContent='✦ Erstelle Bild …';status.textContent='Das lokale Modell rendert dein Bild. Das kann einige Minuten dauern …';
   try{
    const res=await api('ai-image/generate',{prompt,style,width:landscape?768:512,height:512});
    showImage(res.name);status.textContent='✓ Echtes KI-Bild erstellt und lokal gespeichert.';await refresh();
   }catch(e){status.textContent='⚠ '+e.message;window.notify?.(e.message,true)}
   finally{btn.disabled=false;btn.textContent='✦ KI-Bild erstellen'}
 });
 el('ai-image-refresh').addEventListener('click',refresh);
 refresh();
})();
