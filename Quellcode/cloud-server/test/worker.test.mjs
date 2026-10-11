import {test} from 'node:test';
import assert from 'node:assert/strict';
import {DatabaseSync} from 'node:sqlite';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {createHmac} from 'node:crypto';
import cloud from '../src/worker.mjs';

const schema=fs.readFileSync(new URL('../migrations/0001_init.sql',import.meta.url),'utf8')+ '\n'+fs.readFileSync(new URL('../migrations/0002_chat_privacy.sql',import.meta.url),'utf8');
class D1Mock {
  constructor(native){this.native=native;}
  prepare(sql){
    const p=this.native.prepare(sql);
    return {bind(...vals){return {all:async()=>({results:p.all(...vals)}),first:async()=>p.get(...vals)||null,
      run:async()=>{const r=p.run(...vals);return {success:true,meta:{changes:r.changes,last_row_id:Number(r.lastInsertRowid)}}}};}};
  }
  async batch(statements){
    this.native.exec('BEGIN TRANSACTION');
    try{const result=[];for(const stmt of statements)result.push(await stmt.run());this.native.exec('COMMIT');return result;}
    catch(e){this.native.exec('ROLLBACK');throw e;}
  }
}
function setup(){const native=new DatabaseSync(':memory:');native.exec(schema);return {env:{DB:new D1Mock(native),INVITE_CODE:'Private_Invitation_Code_2026'},native};}
async function hit(env,method,route,token='',payload,origin='http://127.0.0.1:4545'){
 const headers={Origin:origin,'CF-Connecting-IP':'10.0.0.12'};
 if(token)headers.Authorization='Bearer '+token;
 if(payload!==undefined)headers['Content-Type']='application/json';
 const request=new Request('https://fake.workers.dev'+route,{method,headers,body:payload!==undefined?JSON.stringify(payload):undefined});
 const response=await cloud.fetch(request,env);
 return {status:response.status,json:await response.json().catch(()=>null),headers:response.headers};
}
async function signup(env,username){const r=await hit(env,'POST','/api/register','',{username,password:'safe_password_for_test_2026',inviteCode:env.INVITE_CODE});assert.equal(r.status,200,JSON.stringify(r));return r.json.token;}
async function pair(env,alice,bob){
 assert.equal((await hit(env,'POST','/api/request',alice,{username:'bob'})).status,200);
 assert.equal((await hit(env,'POST','/api/accept',bob,{username:'alice'})).status,200);
}
test('healthy and CORS only local desktop origins',async()=>{
 const {env}=setup();const h=await hit(env,'GET','/api/health');assert.equal(h.status,200);assert.equal(h.json.capabilities.persistent,true);assert.equal(h.json.capabilities.files,false);
 assert.equal(h.headers.get('access-control-allow-origin'),'http://127.0.0.1:4545');
 assert.equal((await cloud.fetch(new Request('https://fake.workers.dev/api/messages',{method:'OPTIONS',headers:{Origin:'http://127.0.0.1:2345'}}),env)).status,204);
 assert.equal((await hit(env,'GET','/api/health','',undefined,'https://evil.site')).status,403);
 assert.equal((await hit(env,'GET','/api/health','',undefined,'http://localhost.evil.tld')).status,403);
 assert.equal((await hit(env,'GET','/api/health','',undefined,'http://127.0.0.1.evil.tld')).status,403);
});
test('invitation, auth, persisted sessions, password compatibility and logout',async()=>{
 const {env,native}=setup();
 assert.equal((await hit(env,'POST','/api/register','',{username:'alice',password:'long_pass_12345',inviteCode:'wrong-code'})).status,403);
 const a=await signup(env,'alice');assert.equal(a.length,64);
 const hash=await env.DB.prepare('SELECT salt,pass_hash FROM users WHERE username=?').bind('alice').first();
 const {pbkdf2Sync}=await import('node:crypto');
 assert.equal(hash.pass_hash,pbkdf2Sync('safe_password_for_test_2026',Buffer.from(hash.salt,'hex'),100000,32,'sha256').toString('base64'));
 assert.equal((await hit(env,'POST','/api/register','',{username:'alice',password:'valid_password',inviteCode:env.INVITE_CODE})).status,409);
 assert.equal((await hit(env,'POST','/api/login','',{username:'alice',password:'wrong_password_567'})).status,401);
 assert.equal((await hit(env,'GET','/api/me',a)).json.username,'alice');
 const restarted={...env,DB:new D1Mock(native)};assert.equal((await hit(restarted,'GET','/api/me',a)).status,200);
 assert.equal((await hit(restarted,'POST','/api/logout',a,{})).status,200);
 assert.equal((await hit(env,'GET','/api/me',a)).status,401);
});
test('friends, privacy, chat persistence, blocks and WebRTC signaling',async()=>{
 const {env,native}=setup();const alice=await signup(env,'alice'),bob=await signup(env,'bob'),eve=await signup(env,'eve');
 assert.equal((await hit(env,'POST','/api/messages',alice,{to:'bob',text:'nope'})).status,403);
 assert.equal((await hit(env,'POST','/api/signals',alice,{to:'bob',kind:'offer',data:{type:'offer'}})).status,403);
 await pair(env,alice,bob);
 const invite=await hit(env,'GET','/api/friends',bob);assert.deepEqual(invite.json.friends,['alice']);
 let sent=await hit(env,'POST','/api/messages',alice,{to:'bob',text:'Hallo <script>alert(1)</script>'});
 assert.equal(sent.status,200);assert.equal(sent.json.id,1);
 assert.equal((await hit(env,'POST','/api/messages',eve,{to:'bob',text:'spam'})).status,403);
 const read=await hit(env,'GET','/api/messages?peer=alice&after=0',bob);
 assert.equal(read.status,200);assert.equal(read.json.messages.length,1);assert.equal(read.json.messages[0].text,'Hallo <script>alert(1)</script>');
 assert.equal((await hit({...env,DB:new D1Mock(native)},'GET','/api/messages?peer=alice',bob)).json.messages.length,1);
 assert.equal((await hit(env,'POST','/api/signals',alice,{to:'bob',kind:'offer',data:{type:'offer',sdp:'ok'}})).status,200);
 const signal=await hit(env,'GET','/api/signals?after=0',bob);assert.equal(signal.json.signals.length,1);assert.equal(signal.json.signals[0].data.sdp,'ok');
 assert.equal((await hit(env,'POST','/api/block',bob,{username:'alice'})).status,200);
 assert.equal((await hit(env,'GET','/api/messages?peer=bob',alice)).status,403);
 assert.equal((await hit(env,'GET','/api/signals?after=0',bob)).json.signals.length,0);
 assert.equal((await hit(env,'POST','/api/request',alice,{username:'bob'})).status,403);
 assert.equal((await hit(env,'POST','/api/unblock',bob,{username:'alice'})).status,200);
 assert.equal((await hit(env,'GET','/api/messages?peer=alice',bob)).status,403); // must re-friend
});
test('optional attachments explicitly fail without R2; invalid requests rejected',async()=>{
 const {env}=setup(),a=await signup(env,'alice'),b=await signup(env,'bob');await pair(env,a,b);
 assert.equal((await hit(env,'POST','/api/upload?to=bob',a)).status,503);
 assert.equal((await hit(env,'GET','/api/file?id=abcd',a)).status,503);
 assert.equal((await hit(env,'POST','/api/messages',a,{to:'bob',text:''})).status,400);
 assert.equal((await hit(env,'POST','/api/signals',a,{to:'bob',kind:'hack',data:{}})).status,400);
});
test('R2 optional attachments upload, stream and authorization', async()=>{
 const {env}=setup();const storage=new Map();
 env.FILES={
   async put(k,stream){storage.set(k,Buffer.from(await new Response(stream).arrayBuffer()));},
   async get(k){const b=storage.get(k);return b?{body:new Blob([b]).stream()}:null;},
   async delete(k){storage.delete(k);}
 };
 const alice=await signup(env,'alice'),bob=await signup(env,'bob'),eve=await signup(env,'eve');
 await pair(env,alice,bob);
 const form=new FormData();form.append('file',new Blob([new TextEncoder().encode('hello file')]),'nice.txt');
 const auth={Origin:'http://127.0.0.1:4444',Authorization:'Bearer '+alice};
 const response=await cloud.fetch(new Request('https://fake.workers.dev/api/upload?to=bob',{method:'POST',headers:auth,body:form}),env);
 assert.equal(response.status,200,await response.clone().text());
 const obj=await response.json(); assert.equal(obj.fileName,'nice.txt');assert.equal(obj.fileSize,10);
 const read=await cloud.fetch(new Request('https://fake.workers.dev/api/file?id='+obj.fileId,{headers:{Origin:'http://127.0.0.1:4444',Authorization:'Bearer '+bob}}),env);
 assert.equal(read.status,200);assert.equal(await read.text(),'hello file');
 assert.equal((await hit(env,'GET','/api/file?id='+obj.fileId,eve)).status,403);
 await hit(env,'POST','/api/unfriend',bob,{username:'alice'});
 assert.equal((await hit(env,'GET','/api/file?id='+obj.fileId,bob)).status,403);
});
test('daily cleanup only expires temporary data, never messages', async()=>{
 const {env,native}=setup();const a=await signup(env,'alice'),b=await signup(env,'bob');await pair(env,a,b);
 await hit(env,'POST','/api/messages',a,{to:'bob',text:'Must stay after cleanup'});
 native.prepare('INSERT INTO signals(sender,recipient,kind,created_at) VALUES(?,?,?,?)').run('alice','bob','offer',Date.now()-3600000);
 native.prepare('INSERT INTO sessions(token_hash,username,expires_at) VALUES(?,?,?)').run('0'.repeat(64),'alice',1);
 await cloud.scheduled({},env);
 assert.equal(native.prepare('SELECT COUNT(*) AS c FROM messages').get().c,1);
 assert.equal(native.prepare('SELECT COUNT(*) AS c FROM signals').get().c,0);
 assert.equal(native.prepare('SELECT COUNT(*) AS c FROM sessions').get().c,2);
});

test('v4.5.3: unread counts, read acknowledgements and safe search',async()=>{
 const {env}=setup(),a=await signup(env,'alice'),b=await signup(env,'bob');await pair(env,a,b);
 const msg=await hit(env,'POST','/api/messages',a,{to:'bob',text:'Hallo im Testchat 100% sicher'});
 assert.equal(msg.status,200);
 assert.equal((await hit(env,'GET','/api/unread',b)).json.unread.alice,1);
 assert.equal((await hit(env,'GET','/api/search?peer=alice&q=Testchat',b)).json.messages[0].text,'Hallo im Testchat 100% sicher');
 assert.equal((await hit(env,'GET','/api/search?peer=alice&q=%25z',b)).json.messages.length,0); // percent as literal '%25'
 assert.equal((await hit(env,'POST','/api/read',b,{peer:'alice',id:msg.json.id})).status,200);
 assert.deepEqual((await hit(env,'GET','/api/unread',b)).json.unread,{});
 assert.equal((await hit(env,'POST','/api/read',b,{peer:'eve',id:msg.json.id})).status,403);
 assert.equal((await hit(env,'GET','/api/search?peer=eve&q=Hallo',b)).status,403);
});
test('v4.5.3: private export excludes hashes/tokens; account removal revokes sessions',async()=>{
 const {env,native}=setup(),a=await signup(env,'alice'),b=await signup(env,'bob');await pair(env,a,b);
 await hit(env,'POST','/api/messages',a,{to:'bob',text:'Hallo Datenexport'});
 const ex=await hit(env,'GET','/api/account/export',a);
 assert.equal(ex.status,200);assert.equal(ex.json.messages[0].body,'Hallo Datenexport');
 assert.equal(JSON.stringify(ex.json).includes('pass_hash'),false);
 assert.equal((await hit(env,'POST','/api/account/delete',a,{confirm:'alice',password:'wrong_password'})).status,403);
 assert.equal((await hit(env,'POST','/api/account/delete',a,{confirm:'alice',password:'safe_password_for_test_2026'})).status,200);
 assert.equal((await hit(env,'GET','/api/me',a)).status,401);
 assert.equal(native.prepare('SELECT COUNT(*) AS n FROM messages').get().n,0);
 assert.equal((await hit(env,'GET','/api/friends',b)).json.friends.length,0);
});

// ---------- v4.5.5: TURN ----------
const CF_SAMPLE={iceServers:[
 {urls:['stun:stun.cloudflare.com:3478']},
 {urls:['turn:turn.cloudflare.com:3478?transport=udp','turn:turn.cloudflare.com:53?transport=udp','turn:turn.cloudflare.com:80?transport=tcp','turns:turn.cloudflare.com:443?transport=tcp'],username:'user-abc',credential:'secret-xyz'}]};
async function withFetch(stub,fn){const real=globalThis.fetch;globalThis.fetch=stub;try{return await fn();}finally{globalThis.fetch=real;}}
const relayOf=r=>r.json.iceServers.filter(x=>x.username);

test('v4.5.5 TURN: without configuration only STUN is offered; endpoint needs a login',async()=>{
 const {env}=setup();const a=await signup(env,'alice');
 assert.equal((await hit(env,'GET','/api/ice')).status,401);
 const r=await hit(env,'GET','/api/ice',a);
 assert.equal(r.status,200);assert.equal(r.json.hasTURN,false);assert.equal(r.json.turnMode,'none');
 assert.ok(r.json.iceServers.length>=1&&r.json.iceServers.every(x=>!x.username));
 assert.equal((await hit(env,'GET','/api/health')).json.capabilities.turn,false);
});

test('v4.5.5 TURN: static credentials keep working exactly as before',async()=>{
 const {env}=setup();env.TURN_URL='turn:turn.example.com:3478';env.TURN_USER='u';env.TURN_PASSWORD='p';
 const a=await signup(env,'alice');const r=await hit(env,'GET','/api/ice',a);
 assert.equal(r.json.hasTURN,true);assert.equal(r.json.turnMode,'static');
 assert.deepEqual(relayOf(r),[{urls:['turn:turn.example.com:3478'],username:'u',credential:'p'}]);
 assert.ok(r.json.iceServers.some(x=>!x.username),'STUN bleibt dabei');
 assert.equal((await hit(env,'GET','/api/health')).json.capabilities.turn,true);
 env.TURN_URL='http://not-a-turn-url';assert.equal((await hit(env,'GET','/api/ice',a)).json.hasTURN,false,'ungültige Adresse wird nicht ausgeliefert');
});

test('v4.5.5 TURN: coturn shared secret yields per-user expiring HMAC credentials',async()=>{
 const {env}=setup();env.TURN_URL='turn:t1.example.com:3478?transport=udp, turns:t1.example.com:5349';env.TURN_SECRET='geheimnis';env.TURN_TTL='7200';
 const a=await signup(env,'alice'),b=await signup(env,'bob');
 const before=Math.floor(Date.now()/1000);
 const ra=await hit(env,'GET','/api/ice',a),rb=await hit(env,'GET','/api/ice',b);
 assert.equal(ra.json.turnMode,'secret');assert.equal(ra.json.ttl,7200);
 const [relay]=relayOf(ra);
 assert.deepEqual(relay.urls,['turn:t1.example.com:3478?transport=udp','turns:t1.example.com:5349']);
 const [expiry,name]=relay.username.split(':');
 assert.equal(name,'alice');assert.ok(Number(expiry)>=before+7200&&Number(expiry)<=before+7202);
 assert.equal(relay.credential,createHmac('sha1','geheimnis').update(relay.username).digest('base64'),'gleiche Berechnung wie coturn');
 assert.equal(relayOf(rb)[0].username.split(':')[1],'bob');assert.notEqual(relayOf(rb)[0].credential,relay.credential);
 env.TURN_TTL='5';assert.equal((await hit(env,'GET','/api/ice',a)).json.ttl,3600,'TTL wird nach unten begrenzt');
 env.TURN_TTL='999999';assert.equal((await hit(env,'GET','/api/ice',a)).json.ttl,172800,'TTL wird nach oben begrenzt');
});

test('v4.5.5 TURN: Cloudflare mode calls the API correctly, filters port 53, caches and never leaks the API token',async()=>{
 const {env}=setup();env.TURN_KEY_ID='key-cache-test';env.TURN_KEY_API_TOKEN='api-token-geheim';
 const a=await signup(env,'alice');const calls=[];
 await withFetch(async(url,init)=>{calls.push({url:String(url),init});return new Response(JSON.stringify(CF_SAMPLE),{status:201});},async()=>{
  const r=await hit(env,'GET','/api/ice',a);
  assert.equal(calls.length,1);
  assert.equal(calls[0].url,'https://rtc.live.cloudflare.com/v1/turn/keys/key-cache-test/credentials/generate-ice-servers');
  assert.equal(calls[0].init.method,'POST');assert.equal(calls[0].init.headers.Authorization,'Bearer api-token-geheim');
  assert.deepEqual(JSON.parse(calls[0].init.body),{ttl:86400});
  assert.equal(r.json.turnMode,'cloudflare');assert.equal(r.json.hasTURN,true);
  const [relay]=relayOf(r);
  assert.ok(!relay.urls.some(u=>u.includes(':53')),'Port 53 wird herausgefiltert');
  assert.deepEqual(relay.urls,['turn:turn.cloudflare.com:3478?transport=udp','turn:turn.cloudflare.com:80?transport=tcp','turns:turn.cloudflare.com:443?transport=tcp']);
  assert.deepEqual(r.json.iceServers[0],{urls:['stun:stun.cloudflare.com:3478']},'STUN des Anbieters statt Google');
  assert.ok(!JSON.stringify(r.json).includes('api-token-geheim'),'API-Token darf nie zum Client');
  await hit(env,'GET','/api/ice',a);assert.equal(calls.length,1,'zweite Anfrage kommt aus dem Zwischenspeicher');
 });
});

test('v4.5.5 TURN: Cloudflare failures and malformed answers fall back to STUN and are not retried every call',async()=>{
 const {env}=setup();env.TURN_KEY_API_TOKEN='x';const a=await signup(env,'alice');
 let n=0;
 env.TURN_KEY_ID='key-fail-500';
 await withFetch(async()=>{n++;return new Response('boom',{status:500});},async()=>{
  const r=await hit(env,'GET','/api/ice',a);
  assert.equal(r.status,200);assert.equal(r.json.hasTURN,false);assert.equal(r.json.turnMode,'error');assert.ok(r.json.iceServers.length>=1);
  await hit(env,'GET','/api/ice',a);assert.equal(n,1,'Fehler wird kurz gemerkt, kein Dauerfeuer auf die API');
 });
 env.TURN_KEY_ID='key-fail-shape';
 await withFetch(async()=>new Response(JSON.stringify({iceServers:[{urls:['turn:turn.cloudflare.com:3478'],username:'nur-name'}]}),{status:201}),async()=>{
  const r=await hit(env,'GET','/api/ice',a);assert.equal(r.json.hasTURN,false);assert.equal(r.json.turnMode,'error');
 });
 env.TURN_KEY_ID='key-fail-net';
 await withFetch(async()=>{throw new Error('Netzwerk weg');},async()=>{
  const r=await hit(env,'GET','/api/ice',a);assert.equal(r.status,200);assert.equal(r.json.turnMode,'error');
 });
});
