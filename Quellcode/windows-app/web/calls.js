'use strict';
// SQUAD BASE 4.5.5 – Anruf-Engine mit Gruppenanrufen.
//
// Aufbau: Mesh. Jeder Teilnehmer hat eine eigene WebRTC-Verbindung zu jedem anderen (bis MAX_PARTICIPANTS).
// Der Cloud-Server wird NICHT verändert: Signalisierung läuft über die vorhandenen /api/signals-Endpunkte,
// die Zusatzinformationen stecken im frei formbaren "data"-Feld:
//
//   Einladung   kind "offer"  data.squadCall = {id, members, join:false}  → klingelt beim Empfänger
//   Mesh-Join   kind "offer"  data.squadCall = {id, members, join:true}   → wird nur automatisch angenommen,
//                                                                          wenn id zum eigenen Anruf passt
//   Antwort     kind "answer" data.squadCall = {id, members}
//   Roster      kind "ice"    data = {squadCtl:'roster', id, members}      → nur an Teilnehmer, die squadCall kennen
//
// Wer wen anruft: Die alphabetisch größere Person baut die Verbindung zur kleineren auf. Dadurch gibt es nie
// zwei gleichzeitige Angebote für dasselbe Paar. Bekannte Teilnehmer verbreiten sich per Roster-Nachricht.
// Ältere Clients (≤ 4.5.4) ignorieren die Zusatzfelder und funktionieren weiter als reiner 1:1-Anruf.
(() => {
 const core = window.SquadFriendsCore;
 if (!core) return;
 const { f, e, signal, alertText, node } = core;

 const MAX_PARTICIPANTS = 5;       // inklusive dir selbst; Mesh wird darüber hinaus zu schwer
 const INVITE_TIMEOUT_MS = 60000;  // so lange klingelt eine Einladung
 const RING_TIMEOUT_MS = 60000;    // so lange klingelt ein eingehender Anruf
 const GONE_COOLDOWN_MS = 60000;   // so lange ignorieren wir veraltete Roster-Einträge verlassener Personen
 const RETRY_MS = 15000;           // Mindestabstand für erneute Verbindungsversuche zur selben Person
 const USER_RE = /^[a-z0-9_]{3,20}$/;
 const ID_RE = /^[a-f0-9]{32}$/;

 let speakerId = '';

 // ---------- kleine Helfer ----------
 const current = () => f.call;
 const peerList = c => [...c.peers.values()];
 const accepted = c => peerList(c).filter(p => !p.pending);
 const headcount = c => c.known.size + peerList(c).filter(p => p.pending).length;
 const stopTracks = s => { try { s?.getTracks().forEach(t => t.stop()); } catch {} };
 const setStatus = text => { e('call-status').textContent = text; };
 const tileState = (p, text) => { if (p.tile) p.tile.state.textContent = text; };

 function randomId() {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return [...bytes].map(b => b.toString(16).padStart(2, '0')).join('');
 }
 // Alles, was von anderen Clients kommt, wird geprüft: nur gültige Benutzernamen, keine Duplikate, höchstens 8.
 function cleanUsers(list, exclude = []) {
  const out = [];
  if (!Array.isArray(list)) return out;
  for (const u of list) {
   if (typeof u === 'string' && USER_RE.test(u) && !out.includes(u) && !exclude.includes(u)) out.push(u);
   if (out.length >= 8) break;
  }
  return out;
 }
 function parseMeta(data) {
  const m = data && typeof data === 'object' ? data.squadCall : null;
  if (!m || typeof m !== 'object' || !ID_RE.test(String(m.id || ''))) return null;
  return { id: m.id, members: cleanUsers(m.members), join: m.join === true };
 }
 // „aufgelegt“/„besetzt“ aus einem früheren Anruf (kommt verspätet an) darf einen neuen Anruf nicht beenden.
 // Signale älterer Clients tragen keine ID und werden wie bisher angenommen.
 const sameCall = (c, data) => { const m = parseMeta(data); return !m || m.id === c.id; };
 const callRef = id => ({ squadCall: { id } });
 function newCall(id) {
  return {
   id, media: null, peers: new Map(), known: new Set([f.user]),
   gone: new Map(), attempted: new Map(), unreachable: new Set(),
   ringing: null, videoRequested: false, screen: null
  };
 }

 // ---------- Oberfläche ----------
 function cameraState(on) {
  e('v454-toggle-camera').textContent = on ? '📷 Kamera ausschalten' : '📷 Kamera einschalten';
  e('v454-camera-info').textContent = on ? 'Deine Kamera wird im Anruf übertragen.' : 'Deine Kamera ist aus.';
 }
 function titleFor(c) {
  if (c.ringing) {
   const rest = c.ringing.members.filter(u => u !== c.ringing.from);
   return rest.length ? `Gruppenanruf von ${c.ringing.from} (mit ${rest.join(', ')})` : 'Eingehender Anruf von ' + c.ringing.from;
  }
  const others = [...c.peers.keys()];
  return others.length > 1 ? 'Gruppenanruf mit ' + others.join(', ') : 'Anruf mit ' + (others[0] || '');
 }
 function showOverlay(c, status, { ringing = false, video = false } = {}) {
  e('call-overlay').hidden = false;
  e('call-title').textContent = titleFor(c);
  setStatus(status);
  e('call-accept').hidden = !ringing;
  e('v454-accept-video').hidden = !ringing || !video;
  e('call-reject').hidden = !ringing;
  e('call-share').hidden = ringing;
  e('call-hangup').hidden = false;
  e('v453-mute').textContent = '🎙 Mikrofon an';
  e('v453-mute').disabled = ringing;
 }
 function addTile(p) {
  const root = node('div', 'v455-tile');
  root.dataset.user = p.user;
  const video = document.createElement('video');
  video.autoplay = true;
  video.playsInline = true;
  video.setAttribute('aria-label', 'Video von ' + p.user);
  const bar = node('div', 'v455-tile-bar');
  const state = node('span', 'v455-tile-state', '');
  const vol = document.createElement('input');
  vol.type = 'range';
  vol.min = '0';
  vol.max = '100';
  vol.value = String(core.peerVolume(p.user));
  vol.className = 'v455-tile-vol';
  vol.title = 'Lautstärke von ' + p.user;
  vol.setAttribute('aria-label', 'Lautstärke von ' + p.user);
  vol.hidden = true;
  video.volume = Number(vol.value) / 100;
  vol.oninput = () => { const level = Number(vol.value); video.volume = level / 100; core.savePeerVolume(p.user, level); };
  if (speakerId && typeof video.setSinkId === 'function') video.setSinkId(speakerId).catch(() => {});
  bar.append(node('strong', 'v455-tile-name', p.user), state, vol);
  root.append(video, bar);
  e('v455-tiles').append(root);
  p.tile = { root, video, state, vol };
 }
 function renderInvite(c) {
  const row = e('v455-invite-row');
  const active = !c.ringing && c.peers.size > 0;
  row.hidden = !active;
  if (!active) return;
  const taken = new Set([f.user, ...c.peers.keys(), ...c.known]);
  const blocked = f.blocked || [];
  const candidates = (f.friends || []).filter(u => !taken.has(u) && !blocked.includes(u))
   .sort((a, b) => (f.online?.[b] ? 1 : 0) - (f.online?.[a] ? 1 : 0) || a.localeCompare(b));
  const full = headcount(c) >= MAX_PARTICIPANTS;
  const select = e('v455-invite-select'), button = e('v455-invite-btn');
  const fingerprint = candidates.map(u => u + (f.online?.[u] ? '+' : '-')).join(',') + '|' + full;
  if (select.dataset.fingerprint !== fingerprint) { // Auswahlliste nur neu bauen, wenn sich etwas geändert hat
   const old = select.value;
   select.replaceChildren(new Option('Freund einladen …', ''));
   candidates.forEach(u => select.add(new Option(u + (f.online?.[u] ? ' · online' : ''), u)));
   if (candidates.includes(old)) select.value = old;
   select.dataset.fingerprint = fingerprint;
  }
  select.disabled = full || !candidates.length;
  button.disabled = select.disabled;
  button.title = full ? `Maximal ${MAX_PARTICIPANTS} Personen pro Anruf.` : '';
 }
 function renderLayout() {
  const c = current();
  if (!c) return;
  const n = c.peers.size, group = n > 1;
  e('call-title').textContent = titleFor(c);
  e('v455-tiles').classList.toggle('multi', group);
  e('v455-tiles').dataset.count = String(n);
  for (const p of peerList(c)) if (p.tile) p.tile.vol.hidden = !group;
  // Bei genau einer Person bleibt der bekannte Lautstärkeregler, bei mehreren hat jede Kachel einen eigenen.
  e('v455-volume-row').hidden = group || n === 0;
  if (n === 1) {
   const p = peerList(c)[0], level = core.peerVolume(p.user);
   e('v453-call-volume').value = level;
   e('v453-volume-label').textContent = level + '%';
   e('v454-person-volume-name').textContent = 'Lautstärke von ' + p.user;
  }
  renderInvite(c);
 }
 function updateStatus(c) {
  const list = accepted(c);
  if (!list.length) return;
  const up = list.filter(p => p.pc.connectionState === 'connected').length;
  if (up === list.length) setStatus(list.length > 1 ? `● Verbunden – ${list.length + 1} Personen im Gespräch` : '● Verbunden – Gespräch läuft');
  else if (up > 0) setStatus(`● ${up + 1} von ${list.length + 1} Personen verbunden …`);
 }

 // ---------- Geräte ----------
 function audioConstraints() {
  const id = e('v453-microphones').value;
  return { echoCancellation: true, noiseSuppression: true, ...(id ? { deviceId: { ideal: id } } : {}) };
 }
 async function getMedia(video) {
  if (!navigator.mediaDevices?.getUserMedia) throw new Error('Mikrofon nicht verfügbar. Bitte Windows-Berechtigungen prüfen.');
  return navigator.mediaDevices.getUserMedia({ audio: audioConstraints(), video: !!video });
 }
 async function refreshMicList() {
  if (!navigator.mediaDevices?.enumerateDevices) return;
  try {
   const all = await navigator.mediaDevices.enumerateDevices();
   const inputs = all.filter(d => d.kind === 'audioinput'), outputs = all.filter(d => d.kind === 'audiooutput');
   const mic = e('v453-microphones'), oldMic = mic.value;
   mic.replaceChildren(new Option('Systemstandard', ''));
   inputs.forEach((d, i) => mic.add(new Option(d.label || 'Mikrofon ' + (i + 1), d.deviceId)));
   if (inputs.some(d => d.deviceId === oldMic)) mic.value = oldMic;
   const speaker = e('v453-speakers'), oldSpeaker = speaker.value;
   speaker.replaceChildren(new Option('Systemstandard', ''));
   outputs.forEach((d, i) => speaker.add(new Option(d.label || 'Lautsprecher ' + (i + 1), d.deviceId)));
   if (outputs.some(d => d.deviceId === oldSpeaker)) speaker.value = oldSpeaker;
   const supported = typeof document.createElement('video').setSinkId === 'function';
   speaker.disabled = !supported;
   speaker.title = supported ? '' : 'Ausgabegerät wechseln wird von diesem Windows-Browser nicht unterstützt.';
  } catch {}
 }
 async function changeMicrophone() {
  const c = current();
  if (!c?.media) return;
  let stream;
  try {
   stream = await navigator.mediaDevices.getUserMedia({ audio: audioConstraints(), video: false });
   if (current() !== c) { stopTracks(stream); return; }
   const fresh = stream.getAudioTracks()[0], old = c.media.getAudioTracks()[0];
   fresh.enabled = old?.enabled !== false;
   for (const p of peerList(c)) {
    const sender = p.pc.getSenders().find(s => s.track?.kind === 'audio');
    if (sender) await sender.replaceTrack(fresh);
   }
   if (old) { c.media.removeTrack(old); old.stop(); }
   c.media.addTrack(fresh);
   e('v453-mute').textContent = fresh.enabled ? '🎙 Mikrofon an' : '🔇 Stumm';
  } catch (err) {
   stopTracks(stream);
   alertText('Mikrofonwechsel fehlgeschlagen: ' + err.message, true);
  }
 }
 function toggleMic() {
  const track = current()?.media?.getAudioTracks()[0];
  if (!track) return;
  track.enabled = !track.enabled;
  e('v453-mute').textContent = track.enabled ? '🎙 Mikrofon an' : '🔇 Stumm';
 }
 async function setSpeaker(id) {
  speakerId = id;
  for (const video of document.querySelectorAll('#v455-tiles video')) {
   if (typeof video.setSinkId !== 'function') return;
   try { await video.setSinkId(id); } catch (err) { alertText('Lautsprecherwechsel nicht möglich: ' + err.message, true); }
  }
 }

 // ---------- ICE-Server / TURN ----------
 // Kurzlebige TURN-Zugangsdaten vor jedem Anruf frisch holen. Schlägt das fehl, gelten die vom Login.
 async function refreshIce() {
  let timer;
  const timeout = new Promise(resolve => { timer = setTimeout(() => resolve(null), 3000); });
  const reply = await Promise.race([core.request('/api/ice').catch(() => null), timeout]);
  clearTimeout(timer);
  if (Array.isArray(reply?.iceServers)) { // immer die aktuelle Antwort übernehmen, auch wenn TURN inzwischen entfernt wurde
   f.iceServers = reply.iceServers;
   f.hasTURN = Boolean(reply.hasTURN);
   f.turnMode = reply.turnMode || '';
  }
 }
 const failHint = () => f.hasTURN
  ? 'Auch der Relay-Server war nicht erreichbar. Bitte „Verbindung testen“ unter „Benachrichtigungen & Mikrofontest“ nutzen.'
  : 'Auf dem Server ist kein TURN-Dienst eingerichtet (siehe README, Abschnitt TURN).';
 // Sammelt ICE-Kandidaten und meldet, welche Arten entstanden sind (host, srflx, relay).
 function probeIce(servers, policy, ms = 8000) {
  return new Promise(resolve => {
   const types = new Set();
   let pc, finished = false, timer;
   const done = () => { if (finished) return; finished = true; clearTimeout(timer); try { pc?.close(); } catch {} resolve([...types]); };
   timer = setTimeout(done, ms);
   try { pc = new RTCPeerConnection({ iceServers: servers, iceTransportPolicy: policy }); } catch { done(); return; }
   pc.createDataChannel('probe');
   pc.onicecandidate = ev => {
    if (!ev.candidate) return done();
    const m = / typ (host|srflx|prflx|relay)/.exec(ev.candidate.candidate || '');
    if (m) { types.add(m[1]); if (policy === 'relay' && m[1] === 'relay') done(); }
   };
   pc.createOffer().then(offer => pc.setLocalDescription(offer)).catch(done);
  });
 }
 async function testConnection() {
  const status = e('v454-test-status'), button = e('v455-net-test');
  if (current()) { status.textContent = 'Der Verbindungstest ist während eines Anrufs nicht möglich.'; return; }
  button.disabled = true;
  status.textContent = 'Verbindungstest läuft … (bis zu 15 Sekunden)';
  try {
   await refreshIce();
   const relay = f.hasTURN ? await probeIce(f.iceServers, 'relay') : [];
   if (relay.includes('relay')) { status.textContent = '✓ Relay-Server (TURN) erreichbar. Anrufe funktionieren auch hinter strengen Firewalls.'; return; }
   const direct = (await probeIce(f.iceServers, 'all', 5000)).includes('srflx');
   if (f.hasTURN) status.textContent = '✗ TURN ist eingerichtet, aber nicht erreichbar (Zugangsdaten oder Firewall). ' + (direct ? 'Direkte Anrufe sind trotzdem möglich.' : 'Auch direkte Verbindungen wurden nicht gefunden.');
   else if (f.turnMode === 'error') status.textContent = '✗ Der TURN-Dienst des Servers antwortet gerade nicht. Direkte Anrufe: ' + (direct ? 'möglich.' : 'nicht gefunden.');
   else status.textContent = direct ? 'Direkte Verbindung möglich. Es ist kein TURN-Server eingerichtet: Hinter strengen Routern können Anrufe scheitern.' : '✗ Keine Verbindung ins Internet gefunden und kein TURN-Server eingerichtet.';
  } finally { button.disabled = false; }
 }

 // ---------- Verbindungen ----------
 // Alle Empfangsspuren (Ton + Bild) einer Person in einem Stream sammeln. Das 'track'-Ereignis feuert je nach
 // Rolle (Anrufer/Angerufener) nicht für alle Spuren, die Empfänger der Verbindung gibt es dagegen immer.
 function syncRemote(p) {
  const remote = (p.remoteMedia ??= new MediaStream());
  for (const receiver of p.pc.getReceivers?.() || []) {
   if (receiver.track && !remote.getTracks().includes(receiver.track)) remote.addTrack(receiver.track);
  }
  if (p.tile && remote.getTracks().length && p.tile.video.srcObject !== remote) {
   p.tile.video.srcObject = remote;
   p.tile.video.play().catch(() => setStatus('● Verbunden · Falls kein Ton hörbar ist, in das Anruffenster klicken.'));
  }
 }
 // Wird immer direkt nach setRemoteDescription aufgerufen.
 // Ein hängendes addIceCandidate darf die Signalverarbeitung nicht für immer blockieren.
 const within = (promise, ms) => { let timer; return Promise.race([promise, new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('Zeitüberschreitung')), ms); })]).finally(() => clearTimeout(timer)); };
 async function flushIce(p) {
  syncRemote(p);
  const queued = p.queuedIce;
  p.queuedIce = [];
  for (const ice of queued) {
   try { await within(p.pc.addIceCandidate(ice), 4000); } catch (err) { console.warn('ICE:', err.message); }
  }
 }
 const videoTransceiver = p => p.pc.getTransceivers().find(t => t.sender.track?.kind === 'video' || t.receiver.track?.kind === 'video');

 // Bildschirmfreigabe auf eine Verbindung anwenden (auch für Personen, die erst später dazukommen).
 async function applyScreenTo(c, p, renegotiate = false) {
  const track = c.screen.getVideoTracks()[0];
  const sender = p.pc.getSenders().find(s => s.track?.kind === 'video') || p.pc.getTransceivers().map(t => t.sender).find(s => !s.track);
  if (sender) { p.previousVideo = sender.track || null; await sender.replaceTrack(track); }
  else { p.pc.addTrack(track, c.screen); if (renegotiate) await sendOffer(p, {}); }
 }

 function createPeer(user, { initiator, pending = false, group = true }) {
  const c = current();
  const pc = new RTCPeerConnection({ iceServers: f.iceServers });
  const p = {
   user, pc, initiator, pending, group, pendingSince: Date.now(), connected: false,
   queuedIce: f.pendingIce[user] || [], remoteMedia: null, iceRetried: false, previousVideo: null, tile: null
  };
  delete f.pendingIce[user];
  c.peers.set(user, p);
  addTile(p);
  pc.onicecandidate = ev => { if (ev.candidate) signal(user, 'ice', ev.candidate.toJSON()).catch(console.warn); };
  pc.ontrack = ev => {
   // Ton und Bild einer Person gehören in EINEN Stream. Chromium liefert sie je nach Verhandlung in getrennten
   // Streams (oder ganz ohne), und ein Video-Element zeigt nur einen davon – dann fehlte Ton oder Bild.
   const remote = (p.remoteMedia ??= new MediaStream());
   if (!remote.getTracks().includes(ev.track)) remote.addTrack(ev.track);
   syncRemote(p);
  };
  pc.onconnectionstatechange = () => onConnectionState(c, p);
  for (const track of c.media.getTracks()) pc.addTrack(track, c.media);
  // Nur der Anrufer legt den (zunächst leeren) Video-Kanal an. Der Angerufene bekommt ihn aus dem Angebot und
  // schaltet ihn in allowSending() auf Senden. Ein eigener Kanal vor dem Angebot würde von Chromium nicht
  // zugeordnet: Die Kamera hinge dann an einem Kanal, der nie ausgehandelt wurde, und das Bild käme nie an.
  if (initiator && !c.media.getVideoTracks().length) pc.addTransceiver('video', { direction: 'sendrecv' });
  if (c.screen) applyScreenTo(c, p).catch(console.warn);
  e('my-video').srcObject = c.media;
  e('call-share').hidden = false;
  e('v453-mute').disabled = false;
  refreshMicList();
  renderLayout();
  return p;
 }
 function onConnectionState(c, p) {
  if (current() !== c || c.peers.get(p.user) !== p) return;
  const state = p.pc.connectionState;
  if (state === 'connected') { p.connected = true; tileState(p, ''); syncRemote(p); }
  if (state === 'disconnected') {
   tileState(p, 'Verbindung instabil …');
   if (c.peers.size === 1) setStatus('Verbindung instabil. Versuche erneut zu verbinden …');
  }
  if (state === 'failed') {
   tileState(p, 'Verbindung fehlgeschlagen');
   setStatus((c.peers.size === 1 ? 'Verbindung fehlgeschlagen. ' : `Verbindung zu ${p.user} fehlgeschlagen. `) + failHint());
   if (p.initiator && !p.iceRetried) { // nur die Seite, die das Angebot gemacht hat, startet einen ICE-Neustart
    p.iceRetried = true;
    p.pc.restartIce?.();
    sendOffer(p, { iceRestart: true }).catch(console.warn);
   }
  }
  updateStatus(c);
 }
 function closePeer(p) {
  try { p.pc.onicecandidate = null; p.pc.ontrack = null; p.pc.onconnectionstatechange = null; p.pc.close(); } catch {}
  if (p.tile) { p.tile.video.srcObject = null; p.tile.root.remove(); p.tile = null; }
 }
 function dropPeer(c, p) {
  closePeer(p);
  if (c.peers.get(p.user) === p) c.peers.delete(p.user);
  if (current() === c) renderLayout();
 }
 // Eine Person ist weg: Verbindung entfernen, Anruf beenden, wenn niemand mehr übrig ist.
 function peerLeft(c, p, text) {
  dropPeer(c, p);
  c.known.delete(p.user);
  c.gone.set(p.user, Date.now());
  if (!c.peers.size) { alertText(text.single); stop(false); }
  else alertText(text.group);
 }

 // ---------- Signale senden ----------
 async function sendOffer(p, { video, join = false, iceRestart = false } = {}) {
  const c = current();
  const offer = await p.pc.createOffer(iceRestart ? { iceRestart: true } : undefined);
  await p.pc.setLocalDescription(offer);
  const data = { type: offer.type, sdp: offer.sdp, squadCall: { id: c.id, members: [...c.known].filter(u => u !== p.user), join } };
  if (video !== undefined) data.squadVideo = video;
  await signal(p.user, 'offer', data);
 }
 // Angerufene Seite: den aus dem Angebot entstandenen Video-Kanal auf „senden und empfangen“ stellen,
 // damit die Kamera später ohne neue Verhandlung per replaceTrack eingeschaltet werden kann.
 function allowSending(p) {
  for (const tx of p.pc.getTransceivers?.() || []) {
   if (tx.receiver?.track?.kind === 'video' && !tx.sender.track && tx.direction === 'recvonly') tx.direction = 'sendrecv';
  }
 }
 async function sendAnswer(p) {
  const c = current();
  allowSending(p);
  const answer = await p.pc.createAnswer();
  await p.pc.setLocalDescription(answer);
  await signal(p.user, 'answer', { type: answer.type, sdp: answer.sdp, squadCall: { id: c.id, members: [...c.known] } });
 }

 // ---------- Teilnehmerliste (Roster) ----------
 function learn(c, list) {
  let changed = false;
  const now = Date.now();
  for (const u of cleanUsers(list, [f.user])) {
   if (c.known.has(u)) continue;
   const left = c.gone.get(u);
   if (left && now - left < GONE_COOLDOWN_MS) continue;
   if (c.known.size >= MAX_PARTICIPANTS) break;
   c.known.add(u);
   changed = true;
  }
  return changed;
 }
 // fresh = Personen, die gerade direkt mit uns den Beitritt abgeschlossen haben. Nur so werden Wiedereintritte
 // innerhalb der Sperrfrist akzeptiert; veraltete Roster-Einträge verlassener Personen bleiben wirkungslos.
 function gossip(c, except, fresh = []) {
  const message = { squadCtl: 'roster', id: c.id, members: [...c.known], fresh };
  for (const p of accepted(c)) {
   if (p.group && p.user !== except) signal(p.user, 'ice', message).catch(() => {});
  }
 }
 function onRoster(c, p, data) {
  if (data.squadCtl !== 'roster' || data.id !== c.id || p.pending || !p.group) return;
  for (const u of cleanUsers(data.fresh, [f.user])) c.gone.delete(u);
  if (learn(c, data.members)) { gossip(c, p.user); reconcile(c); renderLayout(); }
 }
 // Zu jeder bekannten Person ohne Verbindung baut die alphabetisch größere Seite die Verbindung auf.
 function reconcile(c) {
  if (current() !== c || c.ringing) return;
  const now = Date.now();
  for (const u of c.known) {
   if (u === f.user || c.peers.has(u) || c.unreachable.has(u)) continue;
   const left = c.gone.get(u);
   if (left && now - left < GONE_COOLDOWN_MS) continue;
   const tried = c.attempted.get(u);
   if (tried && now - tried < RETRY_MS) continue;
   if (f.user > u) connectTo(c, u);
  }
 }
 async function connectTo(c, user) {
  c.attempted.set(user, Date.now());
  const p = createPeer(user, { initiator: true });
  tileState(p, 'verbindet …');
  try { await sendOffer(p, { join: true }); }
  catch (err) {
   if (current() !== c) return;
   dropPeer(c, p);
   if (/bestätigte Freunde/.test(err.message)) { // Server erlaubt Signale nur zwischen Freunden
    c.unreachable.add(user);
    alertText(`Keine Verbindung zu ${user}: Ihr seid nicht befreundet.`, true);
   } else console.warn('Verbindung zu', user, err.message);
  }
 }

 // ---------- Anruf starten, annehmen, beenden ----------
 async function start(video = false) {
  if (current() || !f.peer) return;
  const peer = f.peer;
  let media;
  try { media = await getMedia(video); }
  catch (err) { alertText('Mikrofon/Kamera nicht freigegeben: ' + err.message, true); return; }
  await refreshIce();
  if (current()) { stopTracks(media); return; }
  const c = f.call = newCall(randomId());
  c.media = media;
  c.videoRequested = video;
  showOverlay(c, 'Rufe an …');
  cameraState(Boolean(media.getVideoTracks().length));
  const p = createPeer(peer, { initiator: true, pending: true });
  tileState(p, 'klingelt …');
  window.SquadFriendAlerts?.tone('calls');
  try { await sendOffer(p, { video, join: false }); }
  catch (err) { alertText('Anruf fehlgeschlagen: ' + err.message, true); stop(false); }
 }
 async function invite(user) {
  const c = current();
  if (!c || c.ringing || !c.media) return;
  if (!USER_RE.test(user) || !(f.friends || []).includes(user) || c.peers.has(user) || c.known.has(user)) return;
  if (headcount(c) >= MAX_PARTICIPANTS) { alertText(`Maximal ${MAX_PARTICIPANTS} Personen pro Anruf.`, true); return; }
  const legacy = accepted(c).find(p => !p.group);
  if (legacy) { alertText(`Gruppenanruf nicht möglich: ${legacy.user} nutzt eine ältere SQUAD-BASE-Version.`, true); return; }
  c.gone.delete(user);
  c.unreachable.delete(user);
  const p = createPeer(user, { initiator: true, pending: true });
  tileState(p, 'klingelt …');
  try { await sendOffer(p, { video: c.media.getVideoTracks().length > 0, join: false }); }
  catch (err) { if (current() === c) dropPeer(c, p); alertText('Einladung fehlgeschlagen: ' + err.message, true); }
 }
 function ring(sig, meta) {
  const c = f.call = newCall(meta?.id || randomId());
  c.ringing = {
   from: sig.from, offer: sig.data, members: meta ? meta.members : [], group: Boolean(meta),
   video: sig.data?.squadVideo === true, at: Date.now()
  };
  showOverlay(c, c.ringing.video ? 'Eingehender Videoanruf · Kamera bleibt aus, bis du sie einschaltest.' : 'Möchtest du den Sprachanruf annehmen?', { ringing: true, video: c.ringing.video });
  cameraState(false);
  renderLayout();
  window.SquadFriendAlerts?.startRing(sig.from);
 }
 async function accept(video = false) {
  const c = current();
  if (!c || !c.ringing) return;
  const invite = c.ringing;
  let media;
  try { media = await getMedia(video); }
  catch (err) { alertText('Mikrofon oder Kamera nicht freigegeben: ' + err.message, true); return; }
  await refreshIce();
  if (current() !== c || c.ringing !== invite) { stopTracks(media); return; }
  window.SquadFriendAlerts?.stopRing();
  c.media = media;
  c.ringing = null;
  c.id = ID_RE.test(c.id) ? c.id : randomId();
  cameraState(video);
  e('call-accept').hidden = true;
  e('v454-accept-video').hidden = true;
  e('call-reject').hidden = true;
  const p = createPeer(invite.from, { initiator: false, group: invite.group });
  c.known.add(invite.from);
  if (invite.group) learn(c, invite.members);
  try {
   await p.pc.setRemoteDescription(new RTCSessionDescription(invite.offer));
   await flushIce(p);
   await sendAnswer(p);
   setStatus('Verbinde …');
   tileState(p, 'verbindet …');
   reconcile(c);
   renderLayout();
  } catch (err) { alertText(err.message, true); stop(true); }
 }
 async function reject() {
  const c = current();
  if (!c) return;
  if (c.ringing) await signal(c.ringing.from, 'busy', c.ringing.group ? callRef(c.id) : null).catch(() => {});
  stop(false);
 }
 function stop(tell = true) {
  window.SquadFriendAlerts?.stopRing();
  const c = current();
  if (!c) { e('call-overlay').hidden = true; return; }
  f.call = null;
  if (tell) {
   if (c.ringing) signal(c.ringing.from, 'busy', c.ringing.group ? callRef(c.id) : null).catch(() => {});
   for (const p of peerList(c)) signal(p.user, 'hangup', callRef(c.id)).catch(() => {});
  }
  for (const p of peerList(c)) { delete f.pendingIce[p.user]; closePeer(p); }
  if (c.ringing) delete f.pendingIce[c.ringing.from];
  c.peers.clear();
  stopTracks(c.screen);
  stopTracks(c.media);
  e('v455-tiles').replaceChildren();
  e('my-video').srcObject = null;
  e('call-overlay').hidden = true;
  e('v453-mute').disabled = true;
  cameraState(false);
 }

 // ---------- Kamera und Bildschirm ----------
 async function toggleCamera() {
  const c = current();
  if (!c?.media || !c.peers.size) { alertText('Zuerst einen Anruf annehmen oder starten.', true); return; }
  if (c.screen) { alertText('Bitte zuerst die Bildschirmfreigabe beenden.', true); return; }
  const old = c.media.getVideoTracks()[0];
  if (old) {
   try {
    for (const p of peerList(c)) await videoTransceiver(p)?.sender.replaceTrack(null);
    c.media.removeTrack(old);
    old.stop();
    e('my-video').srcObject = c.media;
    cameraState(false);
   } catch (err) { alertText('Kamera konnte nicht beendet werden: ' + err.message, true); }
   return;
  }
  let stream;
  try {
   stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
   if (current() !== c) { stopTracks(stream); return; }
   const track = stream.getVideoTracks()[0];
   for (const p of peerList(c)) {
    const tx = videoTransceiver(p);
    if (tx) await tx.sender.replaceTrack(track);
    else { p.pc.addTrack(track, c.media); await sendOffer(p, { video: true }); }
   }
   c.media.addTrack(track);
   e('my-video').srcObject = c.media;
   cameraState(true);
  } catch (err) {
   stopTracks(stream);
   alertText('Kamera konnte nicht gestartet werden: ' + err.message, true);
  }
 }
 async function screenShare() {
  const c = current();
  if (!c || !c.peers.size) return;
  if (!navigator.mediaDevices?.getDisplayMedia) { alertText('Bildschirmfreigabe wird von diesem Browser nicht unterstützt.', true); return; }
  let screen;
  try { screen = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: false }); }
  catch (err) { if (err.name !== 'NotAllowedError') alertText(err.message, true); return; }
  if (current() !== c) { stopTracks(screen); return; }
  const track = screen.getVideoTracks()[0];
  try {
   c.screen = screen;
   for (const p of peerList(c)) await applyScreenTo(c, p, true);
   e('my-video').srcObject = screen;
   setStatus('▣ Bildschirm wird geteilt.');
   track.onended = async () => {
    if (current() !== c || c.screen !== screen) return;
    for (const p of peerList(c)) {
     const sender = p.pc.getSenders().find(s => s.track === track);
     if (sender) await sender.replaceTrack(p.previousVideo || null);
     p.previousVideo = null;
    }
    c.screen = null;
    e('my-video').srcObject = c.media;
    setStatus('Bildschirmfreigabe beendet.');
   };
  } catch (err) {
   c.screen = null;
   stopTracks(screen);
   alertText('Teilen fehlgeschlagen: ' + err.message, true);
  }
 }

 // ---------- Eingehende Signale ----------
 async function onOffer(c, p, sig) {
  const meta = parseMeta(sig.data);
  if (p) { // bestehende Verbindung wird neu verhandelt (z. B. Kamera eingeschaltet)
   await p.pc.setRemoteDescription(new RTCSessionDescription(sig.data));
   await flushIce(p);
   await sendAnswer(p);
   if (meta && meta.id === c.id && learn(c, meta.members)) { gossip(c, p.user); reconcile(c); }
   return;
  }
  const inCall = c && !c.ringing;
  if (inCall && meta && meta.join && meta.id === c.id) { // Mitglied unseres Anrufs will sich verbinden
   if (c.peers.size + 1 >= MAX_PARTICIPANTS) { await signal(sig.from, 'busy', callRef(meta.id)).catch(() => {}); return; }
   await acceptJoin(c, sig, meta);
   return;
  }
  if (meta && meta.join) { await signal(sig.from, 'busy', callRef(meta.id)).catch(() => {}); return; } // veraltet oder fremder Anruf
  if (c) { await signal(sig.from, 'busy', meta ? callRef(meta.id) : null).catch(() => {}); return; } // wir sind schon im Gespräch
  ring(sig, meta);
 }
 async function acceptJoin(c, sig, meta) {
  const p = createPeer(sig.from, { initiator: false, group: true });
  c.gone.delete(sig.from);
  c.known.add(sig.from);
  const changed = learn(c, meta.members);
  try {
   await p.pc.setRemoteDescription(new RTCSessionDescription(sig.data));
   await flushIce(p);
   await sendAnswer(p);
   tileState(p, 'verbindet …');
   if (changed) gossip(c, sig.from, [sig.from]);
   reconcile(c);
   renderLayout();
  } catch (err) {
   console.warn('Beitritt:', err);
   if (current() === c) dropPeer(c, p);
  }
 }
 async function onAnswer(c, p, sig) {
  await p.pc.setRemoteDescription(new RTCSessionDescription(sig.data));
  await flushIce(p);
  const meta = parseMeta(sig.data);
  if (!p.pending) { // Antwort auf eine Neuverhandlung oder einen Mesh-Beitritt
   if (meta && meta.id === c.id && learn(c, meta.members)) { gossip(c, p.user); reconcile(c); }
   return;
  }
  p.pending = false;
  if (!meta || meta.id !== c.id) { // Gegenstelle kennt keine Gruppenanrufe (ältere Version)
   p.group = false;
   if (c.known.size > 1) { // andere sind schon im Anruf → diese Person kann nicht dazu
    signal(p.user, 'hangup', { squadCall: { id: c.id } }).catch(() => {});
    dropPeer(c, p);
    alertText(`${p.user} nutzt eine ältere SQUAD-BASE-Version und kann nicht am Gruppenanruf teilnehmen.`, true);
    return;
   }
  } else p.group = true;
  c.gone.delete(p.user);
  c.known.add(p.user);
  tileState(p, 'verbindet …');
  setStatus('Verbinde …');
  if (p.group) {
   learn(c, meta.members);
   if (accepted(c).some(x => x !== p && x.group)) gossip(c, undefined, [p.user]); // die anderen müssen von der neuen Person erfahren
   reconcile(c);
  }
  renderLayout();
 }
 async function handleSignal(sig) {
  const c = current();
  const data = sig.data;
  const p = c?.peers.get(sig.from);
  if (sig.kind === 'ice' && data && typeof data === 'object' && data.squadCtl) { // Steuernachricht, kein ICE-Kandidat
   if (c && p) onRoster(c, p, data);
   return;
  }
  if (sig.kind === 'offer') return onOffer(c, p, sig);
  if (sig.kind === 'ice') {
   if (!p) { const queue = (f.pendingIce[sig.from] ??= []); if (queue.length < 64) queue.push(data); return; }
   if (p.pc.remoteDescription) { try { await within(p.pc.addIceCandidate(data), 4000); } catch (err) { console.warn('ICE:', err.message); } }
   else p.queuedIce.push(data);
   return;
  }
  if (sig.kind === 'answer') { if (c && p) await onAnswer(c, p, sig); return; }
  if ((sig.kind === 'hangup' || sig.kind === 'busy') && c && !sameCall(c, data)) return; // Signal aus einem früheren Anruf
  if (sig.kind === 'hangup') {
   if (c && p) peerLeft(c, p, { single: 'Dein Freund hat aufgelegt.', group: `${p.user} hat den Anruf verlassen.` });
   else if (c?.ringing && c.ringing.from === sig.from) { alertText('Dein Freund hat aufgelegt.'); stop(false); }
   return;
  }
  if (sig.kind === 'busy' && c && p) {
   peerLeft(c, p, p.pending
    ? { single: 'Anruf abgelehnt oder besetzt.', group: `${p.user} hat den Anruf nicht angenommen.` }
    : { single: 'Anruf abgelehnt oder besetzt.', group: `${p.user} konnte nicht verbunden werden.` });
  }
 }

 // ---------- Zeitüberwachung ----------
 setInterval(() => {
  const c = current();
  if (!c) return;
  const now = Date.now();
  if (c.ringing) {
   if (now - c.ringing.at > RING_TIMEOUT_MS) { alertText('Anruf von ' + c.ringing.from + ' verpasst.'); reject(); }
   return;
  }
  for (const p of peerList(c)) {
   if (p.pending && now - p.pendingSince > INVITE_TIMEOUT_MS) {
    signal(p.user, 'hangup', { squadCall: { id: c.id } }).catch(() => {});
    peerLeft(c, p, { single: p.user + ' antwortet nicht.', group: p.user + ' antwortet nicht.' });
    if (current() !== c) return;
   }
  }
  reconcile(c);
  renderInvite(c);
 }, 1000);

 // ---------- Bedienelemente ----------
 e('call-voice').onclick = () => start(false);
 e('call-video').onclick = () => start(true);
 e('call-accept').onclick = () => accept(false);
 e('v454-accept-video').onclick = () => accept(true);
 e('v454-toggle-camera').onclick = () => toggleCamera();
 e('call-reject').onclick = () => reject();
 e('call-hangup').onclick = () => stop(true);
 e('call-share').onclick = () => screenShare();
 e('v453-mute').onclick = () => toggleMic();
 e('v453-microphones').onchange = () => changeMicrophone();
 e('v453-speakers').onchange = ev => setSpeaker(ev.target.value);
 e('v453-call-volume').oninput = ev => {
  const level = Number(ev.target.value), c = current(), p = c && peerList(c)[0];
  e('v453-volume-label').textContent = level + '%';
  if (!p) return;
  if (p.tile) p.tile.video.volume = level / 100;
  core.savePeerVolume(p.user, level);
 };
 e('v455-net-test').onclick = () => testConnection();
 e('v455-invite-btn').onclick = () => { const user = e('v455-invite-select').value; if (user) invite(user); };

 window.SquadCalls = {
  start, accept, reject, stop, invite, handleSignal, toggleMic, toggleCamera, screenShare, testConnection,
  // Nur für Tests und Fehlersuche: aktueller Zustand ohne Verbindungsobjekte.
  state() {
   const c = current();
   if (!c) return null;
   return {
    id: c.id, known: [...c.known].sort(), ringing: c.ringing ? c.ringing.from : null,
    peers: peerList(c).map(p => ({ user: p.user, pending: p.pending, group: p.group, connection: p.pc.connectionState, tile: Boolean(p.tile) }))
   };
  }
 };
})();
