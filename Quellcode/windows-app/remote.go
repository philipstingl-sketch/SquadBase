package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Remote access is never started automatically. It is reachable only on one
// private IPv4 interface and exposes a deliberately narrow command set.
type RemoteController struct {
	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
	address  string
	pin      string
	expires  time.Time
	sessions map[string]time.Time
	tries    map[string]attemptLimit
	events   []RemoteEvent
	nextID   int64
	sounds   *SoundBoard
}
type attemptLimit struct {
	Count int
	Since time.Time
}
type RemoteEvent struct {
	ID     int64  `json:"id"`
	Action string `json:"action"`
	Value  string `json:"value,omitempty"`
}

func secureHex(n int) (string, error) {
	b := make([]byte, n)
	_, e := rand.Read(b)
	return hex.EncodeToString(b), e
}
func privateIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch x := addr.(type) {
			case *net.IPNet:
				ip = x.IP
			case *net.IPAddr:
				ip = x.IP
			}
			v := ip.To4()
			if v != nil && v.IsPrivate() {
				return v.String()
			}
		}
	}
	return ""
}
func (s *RemoteController) details() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{"enabled": s.listener != nil, "url": s.address, "pin": s.pin, "expires": s.expires.Format(time.RFC3339), "paired": len(s.sessions) > 0}
}
func (s *RemoteController) newPINLocked() error {
	token, e := secureHex(4)
	if e != nil {
		return e
	}
	value, _ := strconv.ParseUint(token, 16, 32)
	s.pin = fmt.Sprintf("%06d", value%1000000)
	s.expires = time.Now().Add(5 * time.Minute)
	s.tries = map[string]attemptLimit{}
	return nil
}
func (s *RemoteController) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return nil
	}
	ip := privateIPv4()
	if ip == "" {
		return errors.New("Kein privates WLAN-/LAN-IPv4-Netz gefunden. Verbinde PC und Handy mit demselben WLAN.")
	}
	ln, e := net.Listen("tcp4", net.JoinHostPort(ip, "0"))
	if e != nil {
		return fmt.Errorf("WLAN-Zugang konnte nicht geöffnet werden: %w", e)
	}
	if e = s.newPINLocked(); e != nil {
		_ = ln.Close()
		return e
	}
	s.listener = ln
	s.address = "http://" + ln.Addr().String() + "/"
	s.sessions = map[string]time.Time{}
	s.events = nil
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handlePublic)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 12 * time.Second, WriteTimeout: 12 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	s.server = srv
	go func() { _ = srv.Serve(ln) }()
	return nil
}
func (s *RemoteController) stop() {
	s.mu.Lock()
	srv := s.server
	s.server = nil
	s.listener = nil
	s.address = ""
	s.pin = ""
	s.sessions = nil
	s.events = nil
	s.mu.Unlock()
	if srv != nil {
		_ = srv.Close()
	}
}
func (s *RemoteController) handleLocal(a *App, w http.ResponseWriter, r *http.Request, events bool) {
	if events {
		if r.Method != http.MethodGet {
			failure(w, 405, "Nur GET")
			return
		}
		s.mu.Lock()
		result := append([]RemoteEvent(nil), s.events...)
		s.events = nil
		s.mu.Unlock()
		send(w, 200, map[string]any{"events": result})
		return
	}
	switch r.Method {
	case http.MethodGet:
		send(w, 200, s.details())
	case http.MethodPost:
		var q struct {
			Action string `json:"action"`
		}
		if json.NewDecoder(r.Body).Decode(&q) != nil {
			failure(w, 400, "Ungültige Eingabe")
			return
		}
		var e error
		switch q.Action {
		case "start":
			e = s.start()
		case "stop":
			s.stop()
		case "renew":
			s.mu.Lock()
			if s.listener == nil {
				e = errors.New("Fernbedienung ist ausgeschaltet")
			} else {
				e = s.newPINLocked()
			}
			s.mu.Unlock()
		default:
			failure(w, 422, "Unbekannte Aktion")
			return
		}
		if e != nil {
			failure(w, 422, e.Error())
			return
		}
		send(w, 200, s.details())
	default:
		failure(w, 405, "Methode nicht erlaubt")
	}
}
func remotePeerAllowed(r *http.Request) bool {
	ipText, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		return false
	}
	ip := net.ParseIP(ipText)
	return ip != nil && ip.To4() != nil && ip.IsPrivate()
}
func (s *RemoteController) handlePublic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self'; frame-ancestors 'none'; form-action 'self'")
	if !remotePeerAllowed(r) {
		failure(w, 403, "Nur im privaten Netzwerk verfügbar")
		return
	}
	s.mu.Lock()
	addr := strings.TrimSuffix(strings.TrimPrefix(s.address, "http://"), "/")
	active := s.listener != nil
	s.mu.Unlock()
	if !active || r.Host != addr {
		failure(w, 403, "Unbekannter Host")
		return
	}
	switch r.URL.Path {
	case "/", "/remote.js", "/remote.css":
		if r.Method != http.MethodGet {
			failure(w, 405, "Nur GET")
			return
		}
		name := "remote.html"
		if r.URL.Path != "/" {
			name = strings.TrimPrefix(r.URL.Path, "/")
		}
		contents, e := assets.ReadFile("web/" + name)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		switch name {
		case "remote.js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case "remote.css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		_, _ = w.Write(contents)
		return
	case "/api/pair":
		if r.Method != http.MethodPost {
			failure(w, 405, "Nur POST")
			return
		}
		if !sameRemoteOrigin(r) {
			failure(w, 403, "Ungültiger Origin")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		var q struct {
			Code string `json:"code"`
		}
		if json.NewDecoder(r.Body).Decode(&q) != nil {
			failure(w, 400, "Ungültige PIN")
			return
		}
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		s.mu.Lock()
		lim := s.tries[ip]
		if time.Since(lim.Since) > time.Minute {
			lim = attemptLimit{Since: time.Now()}
		}
		if lim.Count >= 5 {
			s.mu.Unlock()
			failure(w, 429, "Zu viele Versuche. Bitte eine Minute warten.")
			return
		}
		lim.Count++
		s.tries[ip] = lim
		ok := time.Now().Before(s.expires) && len(q.Code) == 6 && subtle.ConstantTimeCompare([]byte(q.Code), []byte(s.pin)) == 1
		if !ok {
			s.mu.Unlock()
			failure(w, 403, "PIN falsch oder abgelaufen")
			return
		}
		key, e := secureHex(24)
		if e == nil {
			s.sessions[key] = time.Now().Add(12 * time.Hour)
		}
		s.mu.Unlock()
		if e != nil {
			failure(w, 500, "Kopplung nicht möglich")
			return
		}
		send(w, 200, map[string]string{"token": key})
		return
	case "/api/state", "/api/command":
		if r.URL.Path == "/api/command" && (r.Method != http.MethodPost || !sameRemoteOrigin(r)) {
			failure(w, 403, "Nicht erlaubt")
			return
		}
		if r.URL.Path == "/api/state" && r.Method != http.MethodGet {
			failure(w, 405, "Nur GET")
			return
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		s.mu.Lock()
		expires, found := s.sessions[token]
		valid := found && time.Now().Before(expires)
		s.mu.Unlock()
		if !valid || len(token) != 48 {
			failure(w, 401, "Bitte erneut koppeln")
			return
		}
		if r.URL.Path == "/api/state" {
			var clips []SoundClip
			if s.sounds != nil {
				clips = s.sounds.all()
			}
			send(w, 200, map[string]any{"status": "connected", "clips": clips})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		var q struct {
			Action string `json:"action"`
			Value  string `json:"value"`
		}
		if json.NewDecoder(r.Body).Decode(&q) != nil {
			failure(w, 400, "Ungültige Aktion")
			return
		}
		if len(q.Value) > 96 {
			failure(w, 422, "Wert zu lang")
			return
		}
		switch q.Action {
		case "playpause", "next", "previous":
			if err := desktopMediaAction(map[string]string{"playpause": "toggle", "next": "next", "previous": "previous"}[q.Action]); err != nil {
				failure(w, 503, "Musiksteuerung auf diesem PC nicht verfügbar")
				return
			}
		case "dashboard", "spotify", "soundboard", "sound":
			if q.Action == "sound" && !validSoundID(q.Value) {
				failure(w, 422, "Ungültiger Sound")
				return
			}
			s.mu.Lock()
			s.nextID++
			s.events = append(s.events, RemoteEvent{ID: s.nextID, Action: q.Action, Value: q.Value})
			if len(s.events) > 32 {
				s.events = s.events[len(s.events)-32:]
			}
			s.mu.Unlock()
		default:
			failure(w, 422, "Aktion nicht erlaubt")
			return
		}
		send(w, 200, map[string]string{"status": "ok"})
		return
	default:
		http.NotFound(w, r)
	}
}
func sameRemoteOrigin(r *http.Request) bool {
	// Browsers add Origin for cross-site POSTs; reject mismatches.
	if r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+r.Host {
		return false
	}
	// Reject browser cross-site form submits (which omit custom headers).
	return r.Header.Get("Content-Type") == "application/json"
}
