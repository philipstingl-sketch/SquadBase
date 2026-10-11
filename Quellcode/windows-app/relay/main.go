// Squad Base Friends Relay: a small, self-hosted authenticated friends/chat/file/signaling server.
// Intended to run behind HTTPS reverse proxy. Not a TURN server.
package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type User struct {
	Name     string   `json:"name"`
	Salt     string   `json:"salt"`
	Hash     string   `json:"hash"`
	Friends  []string `json:"friends"`
	Incoming []string `json:"incoming"`
	Outgoing []string `json:"outgoing"`
	Blocked  []string `json:"blocked"`
}
type Message struct {
	ID       int64  `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Text     string `json:"text,omitempty"`
	FileID   string `json:"fileId,omitempty"`
	FileName string `json:"fileName,omitempty"`
	FileSize int64  `json:"fileSize,omitempty"`
	At       int64  `json:"at"`
}
type Signal struct {
	ID   int64           `json:"id"`
	From string          `json:"from"`
	To   string          `json:"to"`
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data,omitempty"`
	At   int64           `json:"at"`
}
type FileMeta struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	From string `json:"from"`
	To   string `json:"to"`
}
type DB struct {
	Users       map[string]*User    `json:"users"`
	Messages    []Message           `json:"messages"`
	Files       map[string]FileMeta `json:"files"`
	NextMessage int64               `json:"nextMessage"`
}
type Session struct {
	User string
	Exp  time.Time
}
type Relay struct {
	mu         sync.Mutex
	db         DB
	sessions   map[string]Session
	signals    []Signal
	nextSignal int64
	lastSeen   map[string]time.Time
	dir        string
}

var userRE = regexp.MustCompile(`^[a-z0-9_]{3,20}$`)

func random(n int) string {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}

// PBKDF2-HMAC-SHA256, 210000 iterations, 32 bytes, implemented without external modules.
func passwordHash(password string, salt []byte) []byte {
	pass := []byte(password)
	var block [4]byte
	block[3] = 1
	mac := hmac.New(sha256.New, pass)
	mac.Write(salt)
	mac.Write(block[:])
	u := mac.Sum(nil)
	out := append([]byte(nil), u...)
	for i := 1; i < 210000; i++ {
		mac.Reset()
		_, _ = mac.Write(u)
		u = mac.Sum(u[:0])
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return out
}
func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}
func add(v []string, s string) []string {
	if contains(v, s) {
		return v
	}
	return append(v, s)
}
func remove(v []string, s string) []string {
	r := v[:0]
	for _, x := range v {
		if x != s {
			r = append(r, x)
		}
	}
	return r
}
func (s *Relay) areFriends(a, b string) bool {
	x, y := s.db.Users[a], s.db.Users[b]
	return x != nil && y != nil && contains(x.Friends, b) && contains(y.Friends, a) && !contains(x.Blocked, b) && !contains(y.Blocked, a)
}
func saveFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if e := os.WriteFile(tmp, data, 0600); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
func (s *Relay) save() error {
	b, e := json.MarshalIndent(s.db, "", "  ")
	if e != nil {
		return e
	}
	return saveFile(filepath.Join(s.dir, "users_and_messages.json"), b)
}
func jsonOut(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func errOut(w http.ResponseWriter, c int, msg string) { jsonOut(w, c, map[string]string{"error": msg}) }
func (s *Relay) auth(r *http.Request) string {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(token) != 64 {
		return ""
	}
	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[key]
	if !ok || time.Now().After(session.Exp) {
		delete(s.sessions, key)
		return ""
	}
	s.lastSeen[session.User] = time.Now()
	return session.User
}
func (s *Relay) issue(user string) string {
	token := random(32)
	h := sha256.Sum256([]byte(token))
	s.sessions[hex.EncodeToString(h[:])] = Session{user, time.Now().Add(7 * 24 * time.Hour)}
	return token
}
func decode(r *http.Request, v any, limit int64) error {
	r.Body = http.MaxBytesReader(nil, r.Body, limit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (s *Relay) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	// Browser requests are permitted only from the local Squad Base desktop interface.
	origin := r.Header.Get("Origin")
	if origin != "" {
		u := strings.TrimPrefix(strings.TrimPrefix(origin, "http://"), "https://")
		host, _, e := net.SplitHostPort(u)
		if e != nil {
			host = u
		}
		if host != "127.0.0.1" && host != "localhost" {
			errOut(w, 403, "Zugriff nur über die lokale Squad Base App")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	}
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}
	if r.URL.Path == "/api/health" {
		jsonOut(w, 200, map[string]string{"status": "ok", "name": "Squad Base Friends Relay"})
		return
	}
	if r.Method == "POST" && (r.URL.Path == "/api/register" || r.URL.Path == "/api/login") {
		var q struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if decode(r, &q, 4096) != nil {
			errOut(w, 400, "Ungültige Daten")
			return
		}
		q.Username = strings.ToLower(strings.TrimSpace(q.Username))
		if !userRE.MatchString(q.Username) || len(q.Password) < 10 || len(q.Password) > 128 {
			errOut(w, 400, "Name: 3–20 Zeichen (a-z, 0-9, _). Passwort: mindestens 10 Zeichen.")
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		u := s.db.Users[q.Username]
		if r.URL.Path == "/api/register" {
			if u != nil {
				errOut(w, 409, "Benutzername vergeben")
				return
			}
			salt := random(16)
			sb, _ := hex.DecodeString(salt)
			u = &User{Name: q.Username, Salt: salt, Hash: base64.StdEncoding.EncodeToString(passwordHash(q.Password, sb))}
			s.db.Users[q.Username] = u
			if e := s.save(); e != nil {
				delete(s.db.Users, q.Username)
				errOut(w, 500, "Speichern fehlgeschlagen")
				return
			}
		}
		if u == nil {
			errOut(w, 401, "Anmeldung fehlgeschlagen")
			return
		}
		sb, _ := hex.DecodeString(u.Salt)
		want, _ := base64.StdEncoding.DecodeString(u.Hash)
		if subtle.ConstantTimeCompare(passwordHash(q.Password, sb), want) != 1 {
			errOut(w, 401, "Anmeldung fehlgeschlagen")
			return
		}
		jsonOut(w, 200, map[string]string{"token": s.issue(q.Username), "username": q.Username})
		return
	}
	user := s.auth(r)
	if user == "" {
		errOut(w, 401, "Bitte anmelden")
		return
	}
	switch r.URL.Path {
	case "/api/ice":
		if r.Method == "GET" {
			ice := []map[string]string{{"urls": "stun:stun.l.google.com:19302"}, {"urls": "stun:stun1.l.google.com:19302"}}
			if url := strings.TrimSpace(os.Getenv("SQUAD_TURN_URL")); strings.HasPrefix(url, "turn:") || strings.HasPrefix(url, "turns:") {
				ice = append(ice, map[string]string{"urls": url, "username": os.Getenv("SQUAD_TURN_USER"), "credential": os.Getenv("SQUAD_TURN_PASSWORD")})
			}
			jsonOut(w, 200, map[string]any{"iceServers": ice, "hasTURN": len(ice) > 2})
			return
		}
	case "/api/me":
		if r.Method == "GET" {
			jsonOut(w, 200, map[string]string{"username": user})
			return
		}
	case "/api/logout":
		if r.Method == "POST" {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			h := sha256.Sum256([]byte(token))
			s.mu.Lock()
			delete(s.sessions, hex.EncodeToString(h[:]))
			s.mu.Unlock()
			jsonOut(w, 200, map[string]bool{"ok": true})
			return
		}
	case "/api/friends":
		if r.Method == "GET" {
			s.mu.Lock()
			u := *s.db.Users[user]
			online := map[string]bool{}
			for _, friend := range u.Friends {
				online[friend] = time.Since(s.lastSeen[friend]) < 12*time.Second
			}
			jsonOut(w, 200, map[string]any{"friends": u.Friends, "incoming": u.Incoming, "outgoing": u.Outgoing, "blocked": u.Blocked, "online": online})
			return
		}
	case "/api/request", "/api/accept", "/api/decline", "/api/block", "/api/unblock", "/api/unfriend":
		if r.Method != "POST" {
			break
		}
		var q struct {
			Username string `json:"username"`
		}
		if decode(r, &q, 2048) != nil {
			errOut(w, 400, "Ungültige Daten")
			return
		}
		q.Username = strings.ToLower(strings.TrimSpace(q.Username))
		s.mu.Lock()
		defer s.mu.Unlock()
		self, other := s.db.Users[user], s.db.Users[q.Username]
		if other == nil || q.Username == user {
			errOut(w, 404, "Benutzer nicht gefunden")
			return
		}
		if r.URL.Path != "/api/unblock" && (contains(self.Blocked, q.Username) || contains(other.Blocked, user)) {
			errOut(w, 403, "Kontakt blockiert")
			return
		}
		switch r.URL.Path {
		case "/api/unblock":
			self.Blocked = remove(self.Blocked, q.Username)
		case "/api/request":
			if !contains(self.Friends, q.Username) {
				self.Outgoing = add(self.Outgoing, q.Username)
				other.Incoming = add(other.Incoming, user)
			}
		case "/api/accept":
			if !contains(self.Incoming, q.Username) {
				errOut(w, 409, "Keine Anfrage vorhanden")
				return
			}
			self.Incoming = remove(self.Incoming, q.Username)
			other.Outgoing = remove(other.Outgoing, user)
			self.Friends = add(self.Friends, q.Username)
			other.Friends = add(other.Friends, user)
		case "/api/decline":
			self.Incoming = remove(self.Incoming, q.Username)
			other.Outgoing = remove(other.Outgoing, user)
			self.Outgoing = remove(self.Outgoing, q.Username)
			other.Incoming = remove(other.Incoming, user)
		case "/api/block":
			self.Blocked = add(self.Blocked, q.Username)
			fallthrough
		case "/api/unfriend":
			self.Friends = remove(self.Friends, q.Username)
			other.Friends = remove(other.Friends, user)
			self.Incoming = remove(self.Incoming, q.Username)
			other.Incoming = remove(other.Incoming, user)
			self.Outgoing = remove(self.Outgoing, q.Username)
			other.Outgoing = remove(other.Outgoing, user)
		}
		if e := s.save(); e != nil {
			errOut(w, 500, e.Error())
			return
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	case "/api/messages":
		if r.Method == "GET" {
			peer := r.URL.Query().Get("peer")
			after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
			s.mu.Lock()
			if !s.areFriends(user, peer) {
				s.mu.Unlock()
				errOut(w, 403, "Nur bestätigte Freunde")
				return
			}
			msgs := make([]Message, 0)
			for _, m := range s.db.Messages {
				if m.ID > after && ((m.From == user && m.To == peer) || (m.From == peer && m.To == user)) {
					msgs = append(msgs, m)
					if len(msgs) >= 150 {
						break
					}
				}
			}
			s.mu.Unlock()
			jsonOut(w, 200, map[string]any{"messages": msgs})
			return
		}
		if r.Method == "POST" {
			var q struct {
				To   string `json:"to"`
				Text string `json:"text"`
			}
			if decode(r, &q, 8192) != nil {
				errOut(w, 400, "Ungültige Nachricht")
				return
			}
			q.Text = strings.TrimSpace(q.Text)
			if q.Text == "" || len(q.Text) > 4000 {
				errOut(w, 400, "Nachricht muss 1–4000 Zeichen lang sein")
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if !s.areFriends(user, q.To) {
				errOut(w, 403, "Nur bestätigte Freunde")
				return
			}
			s.db.NextMessage++
			m := Message{ID: s.db.NextMessage, From: user, To: q.To, Text: q.Text, At: time.Now().UnixMilli()}
			s.db.Messages = append(s.db.Messages, m)
			if len(s.db.Messages) > 25000 {
				s.db.Messages = s.db.Messages[len(s.db.Messages)-25000:]
			}
			if e := s.save(); e != nil {
				errOut(w, 500, e.Error())
				return
			}
			jsonOut(w, 200, m)
			return
		}
	case "/api/upload":
		if r.Method == "POST" {
			peer := r.URL.Query().Get("to")
			s.mu.Lock()
			ok := s.areFriends(user, peer)
			s.mu.Unlock()
			if !ok {
				errOut(w, 403, "Nur bestätigte Freunde")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 13<<20)
			f, h, e := r.FormFile("file")
			if e != nil {
				errOut(w, 400, "Datei fehlt oder zu groß (max. 12 MB)")
				return
			}
			defer f.Close()
			if h.Size > 12<<20 {
				errOut(w, 413, "Maximal 12 MB")
				return
			}
			name := filepath.Base(strings.ReplaceAll(h.Filename, "\\", "/"))
			if len(name) > 160 {
				name = name[:160]
			}
			if name == "." || name == "" {
				name = "datei"
			}
			id := random(16)
			path := filepath.Join(s.dir, "files", id)
			out, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				errOut(w, 500, "Speichern fehlgeschlagen")
				return
			}
			n, e := io.Copy(out, io.LimitReader(f, (12<<20)+1))
			ce := out.Close()
			if e != nil || ce != nil || n > 12<<20 {
				os.Remove(path)
				errOut(w, 413, "Datei zu groß oder Übertragung fehlgeschlagen")
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if !s.areFriends(user, peer) {
				os.Remove(path)
				errOut(w, 403, "Nicht mehr befreundet")
				return
			}
			s.db.Files[id] = FileMeta{Name: name, Size: n, From: user, To: peer}
			s.db.NextMessage++
			msg := Message{ID: s.db.NextMessage, From: user, To: peer, FileID: id, FileName: name, FileSize: n, At: time.Now().UnixMilli()}
			s.db.Messages = append(s.db.Messages, msg)
			if e := s.save(); e != nil {
				delete(s.db.Files, id)
				os.Remove(path)
				errOut(w, 500, "Speichern fehlgeschlagen")
				return
			}
			jsonOut(w, 200, msg)
			return
		}
	case "/api/file":
		if r.Method == "GET" {
			id := r.URL.Query().Get("id")
			s.mu.Lock()
			m, ok := s.db.Files[id]
			allowed := ok && (user == m.From || user == m.To) && s.areFriends(m.From, m.To)
			s.mu.Unlock()
			if !allowed {
				errOut(w, 403, "Nicht berechtigt")
				return
			}
			f, e := os.Open(filepath.Join(s.dir, "files", id))
			if e != nil {
				errOut(w, 404, "Datei nicht gefunden")
				return
			}
			defer f.Close()
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": m.Name}))
			w.Header().Set("Cache-Control", "private, no-store")
			_, _ = io.Copy(w, f)
			return
		}
	case "/api/signals":
		if r.Method == "POST" {
			var q struct {
				To   string          `json:"to"`
				Kind string          `json:"kind"`
				Data json.RawMessage `json:"data"`
			}
			if decode(r, &q, 18000) != nil {
				errOut(w, 400, "Ungültiges Signal")
				return
			}
			if !contains([]string{"offer", "answer", "ice", "hangup", "busy"}, q.Kind) || len(q.Data) > 14000 {
				errOut(w, 400, "Ungültiges Signal")
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if !s.areFriends(user, q.To) {
				errOut(w, 403, "Nur bestätigte Freunde")
				return
			}
			s.nextSignal++
			sig := Signal{ID: s.nextSignal, From: user, To: q.To, Kind: q.Kind, Data: q.Data, At: time.Now().UnixMilli()}
			s.signals = append(s.signals, sig)
			if len(s.signals) > 5000 {
				s.signals = s.signals[len(s.signals)-5000:]
			}
			jsonOut(w, 200, map[string]bool{"ok": true})
			return
		}
		if r.Method == "GET" {
			after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
			s.mu.Lock()
			arr := make([]Signal, 0)
			for _, v := range s.signals {
				if v.ID > after && v.To == user && s.areFriends(user, v.From) {
					arr = append(arr, v)
					if len(arr) > 100 {
						break
					}
				}
			}
			latest := s.nextSignal
			s.mu.Unlock()
			jsonOut(w, 200, map[string]any{"signals": arr, "latest": latest})
			return
		}
	}
	errOut(w, 404, "Unbekannter Aufruf")
}
func main() {
	listen := flag.String("listen", "127.0.0.1:8787", "Listener; use reverse proxy for HTTPS")
	dir := flag.String("data", "./squadbase-relay-data", "Data directory")
	flag.Parse()
	if e := os.MkdirAll(filepath.Join(*dir, "files"), 0700); e != nil {
		log.Fatal(e)
	}
	s := &Relay{dir: *dir, sessions: make(map[string]Session), lastSeen: make(map[string]time.Time), db: DB{Users: make(map[string]*User), Files: make(map[string]FileMeta)}}
	b, e := os.ReadFile(filepath.Join(*dir, "users_and_messages.json"))
	if e == nil {
		if e = json.Unmarshal(b, &s.db); e != nil {
			log.Fatal("Database invalid: ", e)
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		log.Fatal(e)
	}
	if s.db.Users == nil {
		s.db.Users = map[string]*User{}
	}
	if s.db.Files == nil {
		s.db.Files = map[string]FileMeta{}
	}
	sort.Slice(s.db.Messages, func(i, j int) bool { return s.db.Messages[i].ID < s.db.Messages[j].ID })
	handler := http.HandlerFunc(s.handler)
	server := &http.Server{Addr: *listen, Handler: handler, ReadHeaderTimeout: 8 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 65 * time.Second}
	fmt.Printf("SQUAD BASE FRIENDS RELAY – listening on %s (HTTPS required for internet use)\n", *listen)
	log.Fatal(server.ListenAndServe())
}
