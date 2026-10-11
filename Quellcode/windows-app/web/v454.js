'use strict';
(() => {
 const $=id=>document.getElementById(id);
 if(!$('v454-show-sounds'))return;
 const message=(text,bad=false)=>{const el=$('v454-clicker-state');el.textContent=text;el.dataset.active='false';if(bad)el.style.color='#ffbaa6'};
 function subtab(type){const sound=type==='sound';$('v454-sound-section').hidden=!sound;$('v454-clicker-section').hidden=sound;
  $('v454-show-sounds').className=sound?'primary':'outline';$('v454-show-clicker').className=sound?'outline':'primary';
  $('v454-show-sounds').setAttribute('aria-pressed',String(sound));$('v454-show-clicker').setAttribute('aria-pressed',String(!sound));}
 $('v454-show-sounds').onclick=()=>subtab('sound');$('v454-show-clicker').onclick=()=>subtab('clicker');
 const hotkeySelect=$('v454-clicker-hotkey');const keys=['F6','F7','F8','F9','F10','F11','F12'];
 for(let i=65;i<=90;i++){let value='CTRL+ALT+'+String.fromCharCode(i);hotkeySelect.add(new Option('Strg + Alt + '+String.fromCharCode(i),value));}
 let binding=false,updating=false;
 $('v454-clicker-bind').onclick=()=>{binding=true;$('v454-clicker-bind').textContent='Taste drücken … (Esc = abbrechen)';$('v454-clicker-bind').focus()};
 document.addEventListener('keydown',event=>{if(!binding)return;event.preventDefault();event.stopImmediatePropagation();
  if(event.key==='Escape'){binding=false;$('v454-clicker-bind').textContent='⌨ Neue Tastenkombination drücken';return;}
  let k=event.key.toUpperCase();if(!keys.includes(k)&&!(event.ctrlKey&&event.altKey&&/^[A-Z]$/.test(k)))return;
  hotkeySelect.value=keys.includes(k)?k:'CTRL+ALT+'+k;binding=false;$('v454-clicker-bind').textContent='⌨ Neue Tastenkombination drücken';
  message('Hotkey gewählt: '+hotkeySelect.selectedOptions[0].text+'. Zum Aktivieren Einstellungen speichern.');},true);
 async function call(action,body={}){const response=await fetch('api/autoclicker',{method:'POST',headers:{'X-Squad-Base':'1','Content-Type':'application/json'},body:JSON.stringify({action,...body})});const data=await response.json();if(!response.ok)throw Error(data.error||'Fehler '+response.status);return data}
 function status(data){if(!data||updating)return;const el=$('v454-clicker-state');el.style.color='';el.dataset.active=String(data.active);
  el.textContent=(data.active?'● Aktiv: ':'○ Aus: ')+(data.clicks||0)+' Klicks · '+data.intervalMs+' ms · '+data.hotkey+(data.active?' · Zum Stoppen Hotkey erneut drücken':'');}
 async function load(){try{let res=await fetch('api/autoclicker',{cache:'no-store'}),data=await res.json();if(!res.ok)throw Error(data.error||'Server nicht bereit');$('v454-clicker-ms').value=data.intervalMs;$('v454-clicker-button').value=data.mouseButton;$('v454-clicker-hotkey').value=data.hotkey;status(data)}catch(e){message(e.message,true)}}
 async function act(action){updating=true;try{const s=await call(action,action==='configure'?{intervalMs:Number($('v454-clicker-ms').value),mouseButton:$('v454-clicker-button').value,hotkey:hotkeySelect.value}:{});updating=false;status(s)}catch(e){updating=false;message(e.message,true)}}
 $('v454-clicker-save').onclick=()=>act('configure');$('v454-clicker-start').onclick=()=>act('start');$('v454-clicker-stop').onclick=()=>act('stop');
 setInterval(async()=>{if(!document.getElementById('soundboard').classList.contains('active')||$('v454-clicker-section').hidden||updating)return;try{let r=await fetch('api/autoclicker',{cache:'no-store'});if(r.ok)status(await r.json())}catch{}},700);
 load();
})();
