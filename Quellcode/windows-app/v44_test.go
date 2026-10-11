package main

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestV44EmbeddedAndVersion(t *testing.T) {
	a := newV4TestApp(t)
	for _, f := range []string{"v44.js", "v44.css", "index.html"} {
		r := requestV40(a, "GET", f, nil)
		if r.Code != 200 || r.Body.Len() < 200 {
			t.Errorf("asset %s: %d", f, r.Code)
		}
	}
	for _, f := range []string{"remote.html", "remote.js", "remote.css"} {
		b, e := assets.ReadFile("web/" + f)
		if e != nil || len(b) < 100 {
			t.Errorf("remote asset %s", f)
		}
	}
	if squadVersion != "4.7.0" {
		t.Fatalf("version: %s", squadVersion)
	}
}
func TestV44SoundPersistenceAndAPI(t *testing.T) {
	dir := t.TempDir()
	s := newSoundBoard(dir)
	if len(s.all()) != 0 {
		t.Fatal("expected empty")
	}
	raw := append([]byte("RIFF\x00\x00\x00\x00WAVEfmt "), bytes.Repeat([]byte{0x11}, 16000)...)
	c, e := s.add(" Test WAV ", raw, ".wav")
	if e != nil {
		t.Fatal(e)
	}
	if c.Name != "Test WAV" || c.Type != "audio/wav" || !validSoundID(c.ID) {
		t.Fatalf("bad clip: %+v", c)
	}
	s2 := newSoundBoard(dir)
	if len(s2.all()) != 1 {
		t.Fatal("sound not persistent")
	}
	recorder := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://127.0.0.1/s/x/api/soundboard/file/"+c.ID, nil)
	s2.serve(recorder, r, c.ID)
	if recorder.Code != 200 || !bytes.Equal(recorder.Body.Bytes(), raw) {
		t.Fatalf("audio content corrupted %d", recorder.Code)
	}
	if _, e = s2.add("bad", []byte("not audio"), ".wav"); e == nil {
		t.Fatal("invalid sound should be rejected")
	}
	if _, e = s2.add("huge", bytes.Repeat([]byte{'A'}, int(maxSoundBytes)+1), ".wav"); e == nil {
		t.Fatal("oversized sound accepted")
	}
	if e = s2.remove(c.ID); e != nil {
		t.Fatal(e)
	}
	if len(newSoundBoard(dir).all()) != 0 {
		t.Fatal("delete not persisted")
	}
	// Local backend upload must not inherit the 8KB JSON cap.
	a := newV4TestApp(t)
	a.soundboard = newSoundBoard(filepath.Join(dir, "new-app"))
	b := append([]byte("ID3"), bytes.Repeat([]byte{0x33}, 20000)...)
	path := "api/soundboard/upload?filename=ding.mp3&name=" + url.QueryEscape("Test")
	req := httptest.NewRequest("POST", "http://127.0.0.1:12345/s/v40/"+path, bytes.NewReader(b))
	req.RemoteAddr = "127.0.0.1:9999"
	req.Header.Set("Origin", "http://127.0.0.1:12345")
	req.Header.Set("X-Squad-Base", "1")
	w := httptest.NewRecorder()
	a.handle(w, req)
	if w.Code != 201 {
		t.Fatalf("upload failure %d: %s", w.Code, w.Body.String())
	}
	if len(a.soundboard.all()) != 1 {
		t.Fatal("upload missing")
	}
}
func fakeRemote(t *testing.T) (*RemoteController, net.Listener) {
	t.Helper()
	l, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	s := &RemoteController{listener: l, address: "http://192.168.1.20:45678/", pin: "654321", expires: time.Now().Add(time.Minute), sessions: map[string]time.Time{}, tries: map[string]attemptLimit{}}
	return s, l
}
func phoneCall(s *RemoteController, path, method, body, token, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://192.168.1.20:45678"+path, strings.NewReader(body))
	r.RemoteAddr = "192.168.1.30:58881"
	if method == "POST" {
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	s.handlePublic(w, r)
	return w
}
func TestV44RemotePairingAndRestrictedCommands(t *testing.T) {
	s, l := fakeRemote(t)
	defer l.Close()
	if w := phoneCall(s, "/api/state", "GET", "", "", ""); w.Code != 401 {
		t.Fatalf("unpaired allowed: %d", w.Code)
	}
	if w := phoneCall(s, "/api/pair", "POST", `{"code":"654321"}`, "", "http://malicious.example"); w.Code != 403 {
		t.Fatal("cross-origin pairing allowed")
	}
	if w := phoneCall(s, "/api/pair", "POST", `{"code":"654320"}`, "", ""); w.Code != 403 {
		t.Fatal("bad PIN allowed")
	}
	w := phoneCall(s, "/api/pair", "POST", `{"code":"654321"}`, "", "")
	if w.Code != 200 {
		t.Fatalf("valid pairing failed %d %s", w.Code, w.Body.String())
	}
	var result struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if len(result.Token) != 48 {
		t.Fatal("session token weak")
	}
	if w = phoneCall(s, "/api/state", "GET", "", result.Token, ""); w.Code != 200 {
		t.Fatal("paired state rejected")
	}
	if w = phoneCall(s, "/api/command", "POST", `{"action":"launch","value":"powershell"}`, result.Token, ""); w.Code != 422 {
		t.Fatal("unsafe remote command allowed")
	}
	if w = phoneCall(s, "/api/command", "POST", `{"action":"sound","value":"../config.json"}`, result.Token, ""); w.Code != 422 {
		t.Fatal("invalid sound path allowed")
	}
	if w = phoneCall(s, "/api/command", "POST", `{"action":"dashboard"}`, result.Token, ""); w.Code != 200 {
		t.Fatalf("safe remote command failed %d", w.Code)
	}
	s.mu.Lock()
	if len(s.events) != 1 || s.events[0].Action != "dashboard" {
		t.Fatal("wrong remote events")
	}
	s.mu.Unlock()
	s.stop()
	if w = phoneCall(s, "/api/state", "GET", "", result.Token, ""); w.Code != 403 {
		t.Fatal("remote remains open after stop")
	}
}
func TestV44RemoteRateLimitAndPrivateRestriction(t *testing.T) {
	s, l := fakeRemote(t)
	defer l.Close()
	for i := 0; i < 5; i++ {
		w := phoneCall(s, "/api/pair", "POST", `{"code":"123456"}`, "", "")
		if w.Code != 403 {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
	if w := phoneCall(s, "/api/pair", "POST", `{"code":"654321"}`, "", ""); w.Code != 429 {
		t.Fatal("rate limit bypass")
	}
	r := httptest.NewRequest("GET", "http://192.168.1.20:45678/", nil)
	r.RemoteAddr = "8.8.8.8:9999"
	w := httptest.NewRecorder()
	s.handlePublic(w, r)
	if w.Code != 403 {
		t.Fatal("public peer accepted")
	}
}
