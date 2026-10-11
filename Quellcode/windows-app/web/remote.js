'use strict';(()=>{
 const $=id=>document.getElementById(id);
 const readToken=()=>{try{return sessionStorage.getItem('squad-remote-token')||''}catch{return ''}};
 const saveToken=value=>{try{if(value)sessionStorage.setItem('squad-remote-token',value);else sessionStorage.removeItem('squad-remote-token')}catch{}};
 let token=readToken();
 async function request(path,data){const opt={method:data?'POST':'GET',headers:{'Content-Type':'application/json'}};if(token)opt.headers.Authorization='Bearer '+token;if(data)opt.body=JSON.stringify(data);const r=await fetch('/api/'+path,opt);const j=await r.json();if(!r.ok)throw Error(j.error||'Verbindung fehlgeschlagen');return j}
 function show(connected){$('pair').hidden=connected;$('controls').hidden=!connected}
 async function verify(){if(!token)return show(false);try{const state=await request('state');show(true);renderSounds(state.clips||[])}catch{token='';saveToken('');show(false)}}
 function renderSounds(clips){const list=$('sounds');list.replaceChildren();for(const clip of clips){const b=document.createElement('button');b.dataset.action='sound';b.dataset.sound=clip.id;b.textContent='▶ '+clip.name;list.appendChild(b)}if(!clips.length){const p=document.createElement('p');p.className='note';p.textContent='Noch keine Sounds auf dem PC importiert.';list.appendChild(p)}}
 $('connect').addEventListener('click',async()=>{const code=$('pin').value.trim();if(!/^\d{6}$/.test(code)){$('pair-status').textContent='Bitte genau sechs Ziffern eingeben.';return}try{const result=await request('pair',{code});token=result.token;saveToken(token);$('pin').value='';$('pair-status').textContent='Verbunden';await verify()}catch(e){$('pair-status').textContent=e.message}});
 $('disconnect').addEventListener('click',()=>{token='';saveToken('');show(false)});
 document.addEventListener('click',async e=>{const btn=e.target.closest('[data-action]');if(!btn)return;btn.disabled=true;try{await request('command',{action:btn.dataset.action,value:btn.dataset.sound||''});$('remote-status').textContent='Befehl gesendet: '+btn.textContent.trim()}catch(err){$('remote-status').textContent=err.message}finally{btn.disabled=false}});
 // The sound list is deliberately minimal and is updated by the desktop UI
 // through a separate authenticated endpoint in future versions.
 verify();
})();
