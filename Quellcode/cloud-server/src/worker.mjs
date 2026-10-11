// SQUAD BASE Friends Cloud – Worker + D1, API-compatible with the v4.5 desktop client.
// Never expose the D1 database or any credentials to the desktop/browser.
const enc = new TextEncoder();
const MAX_FILE = 12 * 1024 * 1024;
const PASSWORD_ITERATIONS = 100000; // Cloudflare WebCrypto limit; existing 4.5.1 cloud accounts use 100000.
const USER_RE = /^[a-z0-9_]{3,20}$/;
const hex = bytes => [...new Uint8Array(bytes)].map(x => x.toString(16).padStart(2, '0')).join('');
const unhex = s => Uint8Array.from(s.match(/.{2}/g) || [], n => parseInt(n, 16));
const b64 = bytes => btoa(String.fromCharCode(...new Uint8Array(bytes)));
const fresh = count => crypto.getRandomValues(new Uint8Array(count));
const time = () => Date.now();
const ok = value => json(value, 200);
const isUser = u => USER_RE.test(u);
const ordered = (a,b) => a < b ? [a,b] : [b,a];
const queryAll = async (db, sql, ...values) => (await db.prepare(sql).bind(...values).all()).results || [];
const queryOne = async (db, sql, ...values) => await db.prepare(sql).bind(...values).first();
const run = (db, sql, ...values) => db.prepare(sql).bind(...values).run();
const httpError = (status, message) => Object.assign(new Error(message), { status });
function json(value, code=200) {
  return new Response(JSON.stringify(value), { status:code, headers:{'Content-Type':'application/json; charset=utf-8','Cache-Control':'no-store'} });
}
function error(code, reason) { return json({error:reason}, code); }
function originAllowed(origin) {
  if (!origin) return true; // Non-browser clients still need credentials and registration invitation.
  try { const u = new URL(origin); return ['http:','https:'].includes(u.protocol) &&
    ['127.0.0.1','localhost','[::1]'].includes(u.hostname) && !u.username && !u.password; }
  catch {return false;}
}
function protectedResponse(response, origin) {
  const headers = new Headers(response.headers);
  headers.set('X-Content-Type-Options','nosniff');
  headers.set('Referrer-Policy','no-referrer');
  headers.set('X-Frame-Options','DENY');
  headers.set('Cross-Origin-Resource-Policy','cross-origin');
  headers.set('Cache-Control', 'no-store');
  if (origin) {
    headers.set('Access-Control-Allow-Origin',origin);
    headers.set('Access-Control-Allow-Headers','Authorization, Content-Type');
    headers.set('Access-Control-Allow-Methods','GET, POST, OPTIONS');
    headers.set('Access-Control-Max-Age','600');
    headers.set('Vary','Origin');
  }
  return new Response(response.body,{status:response.status, headers});
}
async function readJSON(req, maxBytes=8192) {
  if (+(req.headers.get('content-length')||0) > maxBytes) throw httpError(413,'Anfrage zu groß');
  const raw = await req.text();
  if (enc.encode(raw).length > maxBytes) throw httpError(413,'Anfrage zu groß');
  let data;
  try {data = JSON.parse(raw);} catch {throw httpError(400,'Ungültige JSON-Daten');}
  if (!data || typeof data!=='object' || Array.isArray(data)) throw httpError(400,'Ungültige Daten');
  return data;
}
async function sha256(data) {return hex(await crypto.subtle.digest('SHA-256', enc.encode(data)));}
async function passHash(pass, salt) {
  const key = await crypto.subtle.importKey('raw',enc.encode(pass),'PBKDF2',false,['deriveBits']);
  return b64(await crypto.subtle.deriveBits({name:'PBKDF2',salt:unhex(salt),hash:'SHA-256',iterations:PASSWORD_ITERATIONS},key,256));
}
async function secureEqual(a,b) {
  // Hash-then-compare to avoid early prefix leakage and compare full lengths.
  const x = unhex(await sha256(a)), y = unhex(await sha256(b));
  let diff=0; for(let i=0;i<32;i++)diff |= x[i]^y[i]; return diff===0;
}
async function friends(db,a,b) {
  if(!isUser(b) || a===b) return false;
  const [x,y]=ordered(a,b);
  const row=await queryOne(db,`SELECT 1 AS yes FROM friendships WHERE user_a=? AND user_b=?
    AND NOT EXISTS(SELECT 1 FROM blocks WHERE (blocker=? AND blocked=?) OR (blocker=? AND blocked=?))`,x,y,a,b,b,a);
  return !!row;
}
async function blocked(db,a,b) {
  return !!(await queryOne(db,'SELECT 1 AS yes FROM blocks WHERE (blocker=? AND blocked=?) OR (blocker=? AND blocked=?)',a,b,b,a));
}
function msgRow(m) {
  return {id:m.id,from:m.sender,to:m.recipient,at:m.created_at,...(m.body?{text:m.body}:{}),
    ...(m.file_id?{fileId:m.file_id,fileName:m.file_name,fileSize:m.file_size}:{})};
}
function cleanFileName(filename) {
  return (String(filename||'datei').split(/[\\/]/).pop() || 'datei').replace(/[\x00-\x1f\x7f]/g,'').slice(0,160)||'datei';
}
async function throttleAuth(env,request) {
  const addr = request.headers.get('cf-connecting-ip') || 'unknown';
  const key = await sha256((env.INVITE_CODE||'no-invite')+':'+addr);
  const now=time();
  const row=await queryOne(env.DB,'SELECT attempts,period_start,blocked_until FROM auth_limits WHERE address_hash=?',key);
  if(row && row.blocked_until>now) throw httpError(429,'Zu viele Anmeldeversuche. Bitte später erneut versuchen.');
  if(row && row.period_start>now-15*60_000 && row.attempts>=18) {
    await run(env.DB,'UPDATE auth_limits SET blocked_until=? WHERE address_hash=?',now+15*60_000,key);
    throw httpError(429,'Zu viele Anmeldeversuche. Bitte später erneut versuchen.');
  }
  return {key,row};
}
async function failedAuth(env, state) {
  const now=time();
  if(state.row && state.row.period_start>now-15*60_000)
    await run(env.DB,'UPDATE auth_limits SET attempts=attempts+1 WHERE address_hash=?',state.key);
  else await run(env.DB,`INSERT INTO auth_limits(address_hash,attempts,period_start,blocked_until) VALUES(?,1,?,0)
    ON CONFLICT(address_hash) DO UPDATE SET attempts=1,period_start=excluded.period_start,blocked_until=0`,state.key,now);
}
async function loginOrRegister(req,env,register) {
  if(!env.INVITE_CODE || env.INVITE_CODE.length<16) throw httpError(503,'Server-Administrator muss zuerst den Einladungscode konfigurieren.');
  const q=await readJSON(req,4096);
  const username=String(q.username||'').trim().toLowerCase();
  const password=q.password;
  if(!isUser(username)||typeof password!=='string'||password.length<10||password.length>128) throw httpError(400,'Name: 3–20 Zeichen (a-z,0–9,_), Passwort mindestens 10 Zeichen.');
  const limit=await throttleAuth(env,req);
  if(register) {
    if(typeof q.inviteCode!=='string'||!await secureEqual(q.inviteCode,env.INVITE_CODE)) {
      await failedAuth(env,limit); throw httpError(403,'Ungültiger Einladungscode');
    }
    if(await queryOne(env.DB,'SELECT username FROM users WHERE username=?',username)) throw httpError(409,'Benutzername vergeben');
    const salt=hex(fresh(16));const hashed=await passHash(password,salt);
    try {await run(env.DB,'INSERT INTO users(username,salt,pass_hash,created_at,last_seen) VALUES(?,?,?,?,?)',username,salt,hashed,time(),time());}
    catch {throw httpError(409,'Benutzername vergeben');}
  } else {
    const user=await queryOne(env.DB,'SELECT salt,pass_hash FROM users WHERE username=?',username);
    // Run same KDF even for unknown username to reduce account enumeration by timing.
    const salt=user?.salt || '00000000000000000000000000000000';
    const hashed=await passHash(password,salt);
    if(!user||!await secureEqual(hashed,user.pass_hash)) {
      await failedAuth(env,limit); throw httpError(401,'Anmeldung fehlgeschlagen');
    }
  }
  const token=hex(fresh(32));
  await run(env.DB,'INSERT INTO sessions(token_hash,username,expires_at) VALUES(?,?,?)',await sha256(token),username,time()+7*86400_000);
  return ok({token,username});
}
async function authenticate(req,env) {
  const match=/^Bearer ([a-f0-9]{64})$/.exec(req.headers.get('Authorization')||'');
  if(!match) throw httpError(401,'Bitte anmelden');
  const row=await queryOne(env.DB,`SELECT sessions.username AS username,sessions.expires_at AS expires_at,users.last_seen AS last_seen
    FROM sessions JOIN users ON users.username=sessions.username WHERE sessions.token_hash=?`,await sha256(match[1]));
  if(!row || row.expires_at<time()) throw httpError(401,'Bitte anmelden');
  if(row.last_seen < time()-9000) await run(env.DB,'UPDATE users SET last_seen=? WHERE username=?',time(),row.username);
  return row.username;
}
async function friendList(db,user) {
  const [f,incoming,outgoing,b,online]=await Promise.all([
    queryAll(db,'SELECT CASE WHEN user_a=? THEN user_b ELSE user_a END AS name FROM friendships WHERE user_a=? OR user_b=? ORDER BY name',user,user,user),
    queryAll(db,'SELECT sender AS name FROM friend_requests WHERE recipient=? ORDER BY created_at DESC',user),
    queryAll(db,'SELECT recipient AS name FROM friend_requests WHERE sender=? ORDER BY created_at DESC',user),
    queryAll(db,'SELECT blocked AS name FROM blocks WHERE blocker=? ORDER BY name',user),
    queryAll(db,`SELECT username,last_seen FROM users WHERE username IN (SELECT CASE WHEN user_a=? THEN user_b ELSE user_a END FROM friendships WHERE user_a=? OR user_b=?)`,user,user,user)
  ]);
  const state={}; for(const row of online)state[row.username]=row.last_seen>=time()-12000;
  return {friends:f.map(x=>x.name),incoming:incoming.map(x=>x.name),outgoing:outgoing.map(x=>x.name),blocked:b.map(x=>x.name),online:state};
}
async function friendAction(req,db,user,path) {
  const {username:target}=await readJSON(req,2048);
  if(!isUser(target)||target===user || !await queryOne(db,'SELECT 1 AS yes FROM users WHERE username=?',target)) throw httpError(404,'Benutzer nicht gefunden');
  const action=path.slice('/api/'.length);
  if(action!=='unblock' && await blocked(db,user,target)) throw httpError(403,'Kontakt blockiert');
  const [a,b]=ordered(user,target);
  switch(action) {
    case 'request':
      if(!await friends(db,user,target)) await run(db,'INSERT OR IGNORE INTO friend_requests(sender,recipient,created_at) VALUES(?,?,?)',user,target,time());
      break;
    case 'accept': {
      const reqRow=await queryOne(db,'SELECT 1 AS yes FROM friend_requests WHERE sender=? AND recipient=?',target,user);
      if(!reqRow) throw httpError(409,'Keine Anfrage vorhanden');
      await db.batch([
        db.prepare('INSERT OR IGNORE INTO friendships(user_a,user_b) SELECT ?,? WHERE EXISTS(SELECT 1 FROM friend_requests WHERE sender=? AND recipient=?)').bind(a,b,target,user),
        db.prepare('DELETE FROM friend_requests WHERE (sender=? AND recipient=?) OR (sender=? AND recipient=?)').bind(user,target,target,user)
      ]);
      break;
    }
    case 'decline':
      await run(db,'DELETE FROM friend_requests WHERE (sender=? AND recipient=?) OR (sender=? AND recipient=?)',user,target,target,user);
      break;
    case 'unblock':
      await run(db,'DELETE FROM blocks WHERE blocker=? AND blocked=?',user,target);
      break;
    case 'block':
      await run(db,'INSERT OR IGNORE INTO blocks(blocker,blocked) VALUES(?,?)',user,target);
      // intentional fallthrough
    case 'unfriend':
      await db.batch([
        db.prepare('DELETE FROM friendships WHERE user_a=? AND user_b=?').bind(a,b),
        db.prepare('DELETE FROM friend_requests WHERE (sender=? AND recipient=?) OR (sender=? AND recipient=?)').bind(user,target,target,user)
      ]);
      break;
    default: throw httpError(404,'Unbekannter Aufruf');
  }
  return ok({ok:true});
}
async function getMessages(env,user,url) {
  const peer=url.searchParams.get('peer')||'';
  if(!await friends(env.DB,user,peer)) throw httpError(403,'Nur bestätigte Freunde');
  const after=Math.max(0,Number(url.searchParams.get('after'))||0);
  const rows=await queryAll(env.DB,`SELECT * FROM messages WHERE id>? AND
    ((sender=? AND recipient=?) OR (sender=? AND recipient=?)) ORDER BY id ASC LIMIT 150`,after,user,peer,peer,user);
  return ok({messages:rows.map(msgRow)});
}
async function sendMessage(req,env,user) {
  const q=await readJSON(req,8192);
  const peer=String(q.to||'');
  if(typeof q.text!=='string'||!q.text.trim()||enc.encode(q.text.trim()).length>4000)throw httpError(400,'Nachricht muss 1–4000 Bytes lang sein');
  if(!await friends(env.DB,user,peer))throw httpError(403,'Nur bestätigte Freunde');
  const txt=q.text.trim(), now=time();
  const result=await env.DB.prepare(`INSERT INTO messages(sender,recipient,body,created_at)
    SELECT ?,?,?,? WHERE EXISTS (SELECT 1 FROM friendships WHERE user_a=? AND user_b=?)
    AND NOT EXISTS(SELECT 1 FROM blocks WHERE (blocker=? AND blocked=?) OR (blocker=? AND blocked=?))`)
    .bind(user,peer,txt,now,...ordered(user,peer),user,peer,peer,user).run();
  if(result.meta?.changes!==1)throw httpError(403,'Nur bestätigte Freunde');
  return ok({id:result.meta.last_row_id,from:user,to:peer,text:txt,at:now});
}
// v4.5.3: unread counters are derived from D1; no message text is exposed in the counter endpoint.
async function unreadSummary(env,user) {
  const rows=await queryAll(env.DB,`SELECT m.sender AS peer, COUNT(*) AS unread
    FROM messages m LEFT JOIN read_receipts r ON r.reader=? AND r.peer=m.sender
    WHERE m.recipient=? AND m.id>COALESCE(r.last_read_id,0)
    AND EXISTS (SELECT 1 FROM friendships f WHERE (f.user_a=? AND f.user_b=m.sender) OR (f.user_b=? AND f.user_a=m.sender))
    AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker=? AND b.blocked=m.sender) OR (b.blocker=m.sender AND b.blocked=?))
    GROUP BY m.sender ORDER BY unread DESC LIMIT 100`,user,user,user,user,user,user);
  return ok({unread:Object.fromEntries(rows.map(row=>[row.peer,row.unread]))});
}
async function markRead(req,env,user) {
  const q=await readJSON(req,1024), peer=String(q.peer||'');
  if(!await friends(env.DB,user,peer))throw httpError(403,'Nur bestätigte Freunde');
  const id=Number(q.id);
  if(!Number.isSafeInteger(id)||id<0)throw httpError(400,'Ungültige Nachrichten-ID');
  const last=await queryOne(env.DB,`SELECT COALESCE(MAX(id),0) AS id FROM messages
    WHERE id<=? AND ((sender=? AND recipient=?) OR (sender=? AND recipient=?))`,id,user,peer,peer,user);
  const safe=last?.id||0;
  await run(env.DB,`INSERT INTO read_receipts(reader,peer,last_read_id,updated_at) VALUES(?,?,?,?)
    ON CONFLICT(reader,peer) DO UPDATE SET last_read_id=MAX(read_receipts.last_read_id,excluded.last_read_id), updated_at=excluded.updated_at`,user,peer,safe,time());
  return ok({ok:true,lastReadId:safe});
}
async function searchMessages(env,user,url) {
  const peer=url.searchParams.get('peer')||'',q=(url.searchParams.get('q')||'').trim();
  if(!await friends(env.DB,user,peer))throw httpError(403,'Nur bestätigte Freunde');
  if(q.length<2||q.length>100)throw httpError(400,'Suche: 2 bis 100 Zeichen');
  // LIKE wildcards are escaped, user searches for literal text only.
  const needle='%'+q.replace(/[%_\\]/g,x=>String.fromCharCode(92)+x)+'%';
  const rows=await queryAll(env.DB,`SELECT * FROM messages WHERE body IS NOT NULL AND body LIKE ? ESCAPE '\\'
    AND ((sender=? AND recipient=?) OR (sender=? AND recipient=?)) ORDER BY id DESC LIMIT 100`,needle,user,peer,peer,user);
  return ok({messages:rows.reverse().map(msgRow),limited:rows.length===100});
}
async function accountExport(env,user) {
  const friends=await friendList(env.DB,user);
  const messages=await queryAll(env.DB,'SELECT id,sender,recipient,body,file_name,file_size,created_at FROM messages WHERE sender=? OR recipient=? ORDER BY id DESC LIMIT 2000',user,user);
  const self=await queryOne(env.DB,'SELECT username,created_at,last_seen FROM users WHERE username=?',user);
  const total=await queryOne(env.DB,'SELECT COUNT(*) AS n FROM messages WHERE sender=? OR recipient=?',user,user);
  // This export excludes passwords, salts, session tokens and other users' credentials.
  return ok({exportedAt:new Date().toISOString(),profile:self,friends,messages:messages.reverse(),limited:(total?.n||0)>2000});
}
async function deleteAccount(req,env,user) {
  const q=await readJSON(req,4096);
  if(q.confirm!==user||typeof q.password!=='string')throw httpError(400,'Bestätigung und Passwort erforderlich');
  const row=await queryOne(env.DB,'SELECT salt,pass_hash FROM users WHERE username=?',user);
  if(!row||!await secureEqual(await passHash(q.password,row.salt),row.pass_hash))throw httpError(403,'Passwort stimmt nicht');
  // D1's foreign keys cascade account-related rows: messages, friendships, signals, sessions.
  // R2 attachment objects are not guaranteed to be removed; see README.
  await run(env.DB,'DELETE FROM users WHERE username=?',user);
  return ok({ok:true,attachments:'R2-Dateiobjekte gegebenenfalls separat vom Administrator löschen'});
}
async function sendSignal(req,env,user) {
  const q=await readJSON(req,18000);
  if(!['offer','answer','ice','hangup','busy'].includes(q.kind)||!isUser(q.to))throw httpError(400,'Ungültiges Signal');
  const data=JSON.stringify(q.data??null);
  if(enc.encode(data).length>14000)throw httpError(413,'Signal zu groß');
  if(!await friends(env.DB,user,q.to))throw httpError(403,'Nur bestätigte Freunde');
  await run(env.DB,`INSERT INTO signals(sender,recipient,kind,data_json,created_at) VALUES(?,?,?,?,?)`,user,q.to,q.kind,data,time());
  return ok({ok:true});
}
async function getSignals(env,user,url) {
  const after=Math.max(0,Number(url.searchParams.get('after'))||0);
  // Signals expire after five minutes; stale offers must not ring after days away.
  const rows=await queryAll(env.DB,`SELECT id,sender,kind,data_json,created_at FROM signals WHERE id>? AND recipient=? AND created_at>?
    ORDER BY id ASC LIMIT 100`,after,user,time()-5*60_000);
  const latest=await queryOne(env.DB,'SELECT COALESCE(MAX(id),0) AS latest FROM signals');
  const visible=[];
  for(const m of rows) if(await friends(env.DB,user,m.sender))visible.push({id:m.id,from:m.sender,to:user,kind:m.kind,data:JSON.parse(m.data_json),at:m.created_at});
  return ok({signals:visible,latest:latest?.latest||0});
}
async function upload(req,env,user,url) {
  if(!env.FILES)throw httpError(503,'Dateianhänge sind noch nicht aktiviert (Cloudflare R2 erforderlich).');
  const peer=url.searchParams.get('to')||'';
  if(!await friends(env.DB,user,peer))throw httpError(403,'Nur bestätigte Freunde');
  if(+(req.headers.get('content-length')||0)>MAX_FILE+65536)throw httpError(413,'Maximal 12 MB');
  let form;try{form=await req.formData();}catch{throw httpError(400,'Ungültige Datei');}
  const file=form.get('file');
  if(!file || typeof file.size!=='number')throw httpError(400,'Datei fehlt');
  if(file.size<=0||file.size>MAX_FILE)throw httpError(413,'Maximal 12 MB');
  const fileId=hex(fresh(16)), name=cleanFileName(file.name), now=time();
  await env.FILES.put(fileId,file.stream(),{httpMetadata:{contentType:'application/octet-stream'}});
  try {
    // D1 transaction: file metadata + chat entry. R2 blob is cleaned if this fails.
    const results=await env.DB.batch([
      env.DB.prepare('INSERT INTO files(id,sender,recipient,file_name,size,created_at) VALUES(?,?,?,?,?,?)').bind(fileId,user,peer,name,file.size,now),
      env.DB.prepare(`INSERT INTO messages(sender,recipient,file_id,file_name,file_size,created_at)
        SELECT ?,?,?,?,?,? WHERE EXISTS(SELECT 1 FROM friendships WHERE user_a=? AND user_b=?)
        AND NOT EXISTS(SELECT 1 FROM blocks WHERE (blocker=? AND blocked=?) OR (blocker=? AND blocked=?))`).bind(user,peer,fileId,name,file.size,now,...ordered(user,peer),user,peer,peer,user)
    ]);
    if(results[1].meta?.changes!==1){await run(env.DB,'DELETE FROM files WHERE id=?',fileId);throw httpError(403,'Nur bestätigte Freunde');}
    return ok({id:results[1].meta.last_row_id,from:user,to:peer,fileId,fileName:name,fileSize:file.size,at:now});
  } catch(err) {await env.FILES.delete(fileId);throw err;}
}
async function getFile(env,user,url) {
  if(!env.FILES)throw httpError(503,'Dateianhänge sind noch nicht aktiviert');
  const id=url.searchParams.get('id')||'';
  if(!/^[a-f0-9]{32}$/.test(id))throw httpError(404,'Datei nicht gefunden');
  const meta=await queryOne(env.DB,'SELECT * FROM files WHERE id=?',id);
  if(!meta || ![meta.sender,meta.recipient].includes(user) || !await friends(env.DB,meta.sender,meta.recipient))throw httpError(403,'Nicht berechtigt');
  const blob=await env.FILES.get(id);
  if(!blob)throw httpError(404,'Datei nicht gefunden');
  const fileName=encodeURIComponent(meta.file_name).replace(/[!'()*]/g,c=>'%'+c.charCodeAt(0).toString(16).toUpperCase());
  return new Response(blob.body,{status:200,headers:{'Content-Type':'application/octet-stream',
    'Content-Disposition':`attachment; filename="file"; filename*=UTF-8''${fileName}`,
    'Cache-Control':'private, no-store'}});
}
// ---------- ICE / TURN ----------
// Drei Betriebsarten, in dieser Reihenfolge ausgewertet:
//  1. Cloudflare Realtime TURN: Secrets TURN_KEY_ID + TURN_KEY_API_TOKEN (kurzlebige Zugangsdaten per API).
//  2. Eigener coturn mit "use-auth-secret": Secrets TURN_URL (ein oder mehrere, durch Komma getrennt) + TURN_SECRET.
//     Pro Benutzer entsteht ein befristeter Zugang (Benutzername "ablaufzeit:name", Passwort = HMAC-SHA1).
//  3. Statisch (wie bisher): TURN_URL + TURN_USER + TURN_PASSWORD.
// Ohne TURN-Konfiguration liefert der Server nur STUN. Die Zugangsdaten werden nie fest in der App gespeichert.
const DEFAULT_STUN = [{urls:'stun:stun.l.google.com:19302'},{urls:'stun:stun1.l.google.com:19302'}];
const ICE_URL_RE = /^(turns?|stuns?):[^\s]+$/i;
const TURN_FAILURE_RETRY_MS = 30000;
const urlPort = u => { const m = /:(\d+)(?:\?|$)/.exec(u); return m ? Number(m[1]) : 0; };
const iceUrlList = value => String(value||'').split(/[\s,]+/).filter(u => ICE_URL_RE.test(u));
let cloudflareCache = null; // gemeinsame Zugangsdaten für alle Benutzer dieser Worker-Instanz
// Antworten externer Dienste nie ungeprüft an Clients weiterreichen. Port 53 blockieren Browser ohnehin.
function cleanIceServers(list) {
  if (!Array.isArray(list)) return [];
  const out = [];
  for (const entry of list) {
    const urls = (Array.isArray(entry?.urls) ? entry.urls : [entry?.urls])
      .filter(u => typeof u === 'string' && ICE_URL_RE.test(u) && urlPort(u) !== 53).slice(0, 8);
    if (!urls.length) continue;
    const item = {urls};
    if (urls.some(u => /^turns?:/i.test(u))) {
      if (typeof entry.username !== 'string' || typeof entry.credential !== 'string' || !entry.username || !entry.credential) continue;
      item.username = entry.username;
      item.credential = entry.credential;
    }
    out.push(item);
  }
  return out.slice(0, 4);
}
async function cloudflareTurn(env, ttl) {
  const key = env.TURN_KEY_ID + ':' + ttl;
  const now = time();
  if (cloudflareCache && cloudflareCache.key === key && now < cloudflareCache.until) {
    if (cloudflareCache.failed) throw new Error('TURN-Dienst zuletzt nicht erreichbar');
    return cloudflareCache.servers;
  }
  try {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 4000);
    let res;
    try {
      res = await fetch('https://rtc.live.cloudflare.com/v1/turn/keys/' + encodeURIComponent(env.TURN_KEY_ID) + '/credentials/generate-ice-servers', {
        method:'POST', signal:controller.signal,
        headers:{'Authorization':'Bearer '+env.TURN_KEY_API_TOKEN,'Content-Type':'application/json'},
        body:JSON.stringify({ttl})
      });
    } finally { clearTimeout(timer); }
    if (!res.ok) throw new Error('Cloudflare TURN antwortete mit Status ' + res.status);
    const servers = cleanIceServers((await res.json()).iceServers);
    if (!servers.some(x => x.username)) throw new Error('Cloudflare TURN lieferte keine Zugangsdaten');
    // Gemeinsam genutzte Zugangsdaten regelmäßig erneuern, damit Clients immer lange gültige bekommen.
    cloudflareCache = {key, servers, until: now + Math.min(ttl * 250, 3600000)};
    return servers;
  } catch (err) {
    cloudflareCache = {key, failed:true, until: now + TURN_FAILURE_RETRY_MS};
    throw err;
  }
}
async function secretTurn(env, user, ttl) {
  const urls = iceUrlList(env.TURN_URL).filter(u => /^turns?:/i.test(u));
  if (!urls.length) throw new Error('TURN_URL enthält keine gültige turn:/turns:-Adresse');
  const username = (Math.floor(time() / 1000) + ttl) + ':' + user;
  const key = await crypto.subtle.importKey('raw', enc.encode(env.TURN_SECRET), {name:'HMAC',hash:'SHA-1'}, false, ['sign']);
  const credential = b64(await crypto.subtle.sign('HMAC', key, enc.encode(username)));
  return [{urls, username, credential}];
}
async function buildIce(env, user) {
  const ttl = Math.min(172800, Math.max(3600, Number(env.TURN_TTL) || 86400));
  let turn = [], mode = 'none';
  try {
    if (env.TURN_KEY_ID && env.TURN_KEY_API_TOKEN) { mode = 'cloudflare'; turn = await cloudflareTurn(env, ttl); }
    else if (env.TURN_SECRET && env.TURN_URL) { mode = 'secret'; turn = await secretTurn(env, user, ttl); }
    else if (/^turns?:/i.test(String(env.TURN_URL||'').trim())) {
      mode = 'static';
      const urls = iceUrlList(env.TURN_URL).filter(u => /^turns?:/i.test(u));
      turn = [{urls, username:env.TURN_USER||'', credential:env.TURN_PASSWORD||''}];
    }
  } catch (err) { console.warn('TURN:', err.message); turn = []; mode = 'error'; }
  const stun = turn.filter(x => !x.username);
  const relay = turn.filter(x => x.username);
  const iceServers = [...(stun.length ? stun : DEFAULT_STUN), ...relay];
  const hasTURN = relay.length > 0;
  return {iceServers, hasTURN, turnMode:mode, ...(hasTURN ? {ttl} : {})};
}

async function handle(req,env) {
  if(!env.DB)throw httpError(503,'Cloud-Datenbank nicht konfiguriert');
  const url=new URL(req.url), path=url.pathname;
  if(path==='/api/health' && req.method==='GET')return ok({status:'ok',name:'SQUAD BASE Friends Cloud',version:'4.5.5',
    capabilities:{persistent:true,turn:!!((env.TURN_KEY_ID&&env.TURN_KEY_API_TOKEN)||env.TURN_URL),files:!!env.FILES,signaling:true,invites:true,unread:true,search:true,accountExport:true,accountDelete:true}});
  if((path==='/api/register'||path==='/api/login')&&req.method==='POST')return loginOrRegister(req,env,path==='/api/register');
  const user=await authenticate(req,env);
  if(path==='/api/me'&&req.method==='GET')return ok({username:user});
  if(path==='/api/ice'&&req.method==='GET')return ok(await buildIce(env,user));
  if(path==='/api/logout'&&req.method==='POST') {
    await run(env.DB,'DELETE FROM sessions WHERE token_hash=?',await sha256(req.headers.get('Authorization').slice(7)));
    return ok({ok:true});
  }
  if(path==='/api/friends'&&req.method==='GET')return ok(await friendList(env.DB,user));
  if(path==='/api/unread'&&req.method==='GET')return unreadSummary(env,user);
  if(path==='/api/read'&&req.method==='POST')return markRead(req,env,user);
  if(path==='/api/search'&&req.method==='GET')return searchMessages(env,user,url);
  if(path==='/api/account/export'&&req.method==='GET')return accountExport(env,user);
  if(path==='/api/account/delete'&&req.method==='POST')return deleteAccount(req,env,user);
  if(['/api/request','/api/accept','/api/decline','/api/block','/api/unblock','/api/unfriend'].includes(path)&&req.method==='POST')return friendAction(req,env.DB,user,path);
  if(path==='/api/messages') {
    if(req.method==='GET')return getMessages(env,user,url);
    if(req.method==='POST')return sendMessage(req,env,user);
  }
  if(path==='/api/signals') {
    if(req.method==='GET')return getSignals(env,user,url);
    if(req.method==='POST')return sendSignal(req,env,user);
  }
  if(path==='/api/upload'&&req.method==='POST')return upload(req,env,user,url);
  if(path==='/api/file'&&req.method==='GET')return getFile(env,user,url);
  throw httpError(404,'Unbekannter Aufruf');
}
export default {
  async scheduled(_controller,env) {
    // Remove expired sessions and short-lived signaling/rate-limit data.
    // Messages, user accounts and friendships are NEVER purged by this task.
    if(!env.DB)return;
    const now=time();
    await env.DB.batch([
      env.DB.prepare('DELETE FROM sessions WHERE expires_at<?').bind(now),
      env.DB.prepare('DELETE FROM signals WHERE created_at<?').bind(now-10*60_000),
      env.DB.prepare('DELETE FROM auth_limits WHERE period_start<?').bind(now-2*86400_000)
    ]);
  },
  async fetch(req,env) {
    const origin=req.headers.get('Origin');
    if(!originAllowed(origin))return protectedResponse(error(403,'Zugriff nur über die lokale SQUAD BASE App'),null);
    if(req.method==='OPTIONS')return protectedResponse(new Response(null,{status:204}),origin);
    let res;
    try{res=await handle(req,env);}catch(err){
      if(err.status) res=error(err.status,err.message);
      else {console.error('SQUAD BASE Cloud internal error',err);res=error(500,'Interner Serverfehler');}
    }
    return protectedResponse(res,origin);
  }
};
