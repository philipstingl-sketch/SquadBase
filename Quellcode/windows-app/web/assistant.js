'use strict';
// Local curated help is always available. Optional Ollama is a REAL locally
// running language model; do not misrepresent the curated fallback as AI.
(() => {
  const el=id=>document.getElementById(id);
  const form=el('assistant-form'), input=el('assistant-input'), log=el('assistant-log');
  const toggle=el('assistant-local-ai');
  toggle.checked=localStorage.getItem('squad-assistant-ollama-v1')==='1';
  const norm=s=>s.toLocaleLowerCase('de-DE');
  function offlineAnswer(q){
    const s=norm(q);
    if(/norisk|minecraft.*start|launcher|\.exe|exepfad|profil/.test(s))return 'Öffne Einstellungen → Minecraft → EXE auswählen. Wähle die echte noriskclient-launcher-v3.exe im Installationsordner. Alternativ den vollständigen Pfad ohne Anführungszeichen einfügen und speichern. Im NoRisk Launcher wählst du anschließend dein Profil mit Mods und Shadern aus. Ein direkter Profilstart setzt eine von NoRisk angebotene Schnittstelle voraus.';
    if(/wiederbele|kristall|revive|rettung/.test(s))return 'In Cube Rush sammelst du Kristalle, indem du beim Springen die goldenen Steine berührst. Bei einem Crash kannst du ab 5 Kristallen „Wiederbeleben“ wählen. Die erste Rettung kostet 5, die zweite 8, die dritte 11 usw. Du bekommst danach 2,8 Sekunden Schutz. Kristalle gelten nur im aktuellen Durchlauf.';
    if(/schwier|level|geschwind|cube|jump|springen/.test(s))return 'Cube Rush hat die Formen Würfel, Raumschiff, Wave und Gravity Ball. Du kannst die Startform, mehrere Schwierigkeitsgrade und drei Modi wählen. Drücke bei Würfel/Ball kurz, bei Raumschiff/Wave halte Leertaste oder Maus. Escape pausiert. Kristalle und Rekorde werden lokal gespeichert.';
    if(/spotify|playlist|musik|soundtrack|lautstärke/.test(s))return 'Die eigenen SQUAD Sounds findest du unten links: Play, Track wechseln und Lautstärke einstellen. Für Spotify: Reiter Spotify öffnen, optional Spotify Developer Client-ID eintragen und dich über die offizielle Spotify-Anmeldung verbinden. Der SQUAD Soundtrack funktioniert unabhängig von Spotify.';
    if(/ping|internet|server|latenz|speedtest|download/.test(s))return 'Im Reiter Internet & Ping kannst du Cloudflare, Google, Quad9, Minecraft oder einen eigenen Host auswählen. Der separate Ping-Test misst echte TCP-Verbindungszeiten, nicht deinen Ingame-Ping. Der Internet-Speedtest misst Upload und Download separat.';
    if(/freund|chat|call|bildschirm|datei/.test(s))return 'Im Reiter Freunde & Chat kannst du ein Konto auf deinem Squad-Relay-Server erstellen. Nur bestätigte Freunde dürfen mit dir schreiben, Dateien senden und anrufen. Für verschiedene PCs brauchst du einen erreichbaren Freunde-Server mit HTTPS. Bildschirmteilen fragt zuvor nach deiner Freigabe.';
    if(/hardware|grafik|cpu|gpu|ram|auslastung|temperatur/.test(s))return 'Im Dashboard werden echte Windows-Werte für CPU, RAM und – falls der Grafiktreiber es unterstützt – GPU angezeigt. Den ausführlichen Auslastungsverlauf findest du rechts im Hardware Monitor Pro. Unter den Ringen stehen aktive Spiele und Schnellaktionen. Ohne verfügbare GPU-Messung bleibt deren Linie leer statt Zufallswerte anzuzeigen.';
    if(/update|patch|neuigkeit/.test(s))return 'Unter Spiele-Updates findest du abrufbare offizielle Minecraft- und Steam-News und Links zu den Patchnotes weiterer Spiele. Diese Meldungen zeigen Veröffentlichungen – sie prüfen nicht automatisch, ob deine Spiele aktuell installiert sind.';
    if(/reaktion|millisek|fehlstart|challenge/.test(s))return 'Beim Reaktionszeit-Test wartest du, bis die Fläche grün wird. Klicke dann möglichst schnell. Ein Klick davor ist ein Fehlstart. Die 5-Runden-Challenge berechnet den Durchschnitt und speichert deinen besten Versuch lokal.';
    if(/bild|generier|zeichnung|foto|wallpaper/.test(s))return 'Im KI-Assistenten findest du unten das Bilderstudio. Für wirklich generierte Bilder starte lokal Stable Diffusion WebUI mit der Option --api und einem Bildmodell. Die Bilder werden auf deinem PC gespeichert. Ohne Modell erfindet SQUAD BASE keine Ergebnisse.';
    if(/ki|ollama|modell|offline/.test(s))return 'Die eingebaute Hilfe funktioniert offline mit geprüften Antworten zu SQUAD BASE. Für frei formulierte KI-Antworten installierst du optional Ollama auf deinem PC und ein Modell, zum Beispiel llama3.2:3b. Aktiviere dann rechts „Lokales KI-Modell versuchen“. Ohne Modell beantworte ich die häufigsten Themen aus der eingebauten Hilfe.';
    return 'Ich kann dir zu SQUAD BASE, NoRisk/Minecraft, dem Freunde-System, Spotify, Speedtest, Hardware sowie Cube Rush und dem Reaktionstest helfen. Wähle rechts ein Thema oder formuliere deine Frage genauer. Für offene KI-Unterhaltungen kannst du optional ein lokales Ollama-Modell aktivieren.';
  }
  function add(role,content){
    const bubble=document.createElement('div');bubble.className='assistant-bubble '+role;
    const label=document.createElement('small');label.textContent=role==='bot'?'SQUAD ASSISTENT':'DU';
    const p=document.createElement('p');p.textContent=content;
    bubble.append(label,p);log.append(bubble);log.scrollTop=log.scrollHeight;return p;
  }
  let loading=false;
  let history=[];
  window.SquadAssistantHistory={load:(messages)=>{
    log.replaceChildren();history=[];
    for(const m of messages.slice(-40)){
      if(!['user','assistant'].includes(m.role)||typeof m.content!=='string')continue;
      add(m.role==='user'?'user':'bot',m.content);
      history.push({role:m.role,content:m.content});
    }
    history=history.slice(-8);
  },get:()=>history.slice()};
  const modelSelect=el('assistant-model');
  const savedModel=localStorage.getItem('squad-assistant-model-v3')||'auto';
  async function loadModels(){
    const old=modelSelect.value!=='auto'?modelSelect.value:savedModel;
    try{
      const data=await api('assistant/models');modelSelect.replaceChildren(new Option('Automatisch','auto'));
      for(const name of data.models||[])modelSelect.append(new Option(name,name));
      modelSelect.value=[...modelSelect.options].some(o=>o.value===old)?old:'auto';
      el('assistant-status').textContent='✓ '+(data.models||[]).length+' lokale Modelle verfügbar';
    }catch(e){el('assistant-status').textContent='⚠ '+e.message+' – eingebaute Hilfe bleibt verfügbar'}
  }
  el('assistant-check-models').addEventListener('click',loadModels);
  modelSelect.addEventListener('change',()=>localStorage.setItem('squad-assistant-model-v3',modelSelect.value));
  if(toggle.checked)loadModels();
  async function sendQuestion(q){
    q=q.trim().slice(0,600);if(!q||loading)return;
    add('user',q);input.value='';loading=true;el('assistant-send').disabled=true;
    const response=add('bot',toggle.checked?'Lokales KI-Modell wird gefragt …':'Eingebaute Hilfe wird durchsucht …');
    if(toggle.checked){
      try{
        const r=await api('assistant',{message:q,history:history.slice(-8),model:modelSelect.value});
        response.textContent=r.answer||offlineAnswer(q);
        history.push({role:'user',content:q},{role:'assistant',content:response.textContent});history=history.slice(-8);
        el('assistant-status').textContent='✓ Antwort von lokalem Modell: '+(r.model||'Ollama');
      }catch(e){
        response.textContent=offlineAnswer(q)+'\n\n(Lokale KI nicht erreichbar: '+e.message+'. Offline-Hilfe bleibt verfügbar.)';
        el('assistant-status').textContent='⚠ Ollama nicht erreichbar – Offline-Hilfe aktiv';
      }
    }else response.textContent=offlineAnswer(q);
    window.dispatchEvent(new CustomEvent('squad:assistant-answer',{detail:{text:response.textContent}}));
    loading=false;el('assistant-send').disabled=false;input.focus();
  }
  form.addEventListener('submit',e=>{e.preventDefault();sendQuestion(input.value)});
  document.querySelectorAll('[data-help]').forEach(b=>b.addEventListener('click',()=>sendQuestion(b.dataset.help)));
  toggle.addEventListener('change',()=>{
    localStorage.setItem('squad-assistant-ollama-v1',toggle.checked?'1':'0');
    el('assistant-status').textContent=toggle.checked?'Lokaler KI-Modus aktiviert (nur bei laufendem Ollama)':'Modus: eingebaute Hilfe';
  });
  el('assistant-clear').addEventListener('click',()=>{
    log.replaceChildren();history=[];add('bot','Neuer Chat! Was möchtest du über SQUAD BASE wissen?');
  });
})();
