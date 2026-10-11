package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setup(t *testing.T) *Relay {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "files"), 0700); err != nil {
		t.Fatal(err)
	}
	return &Relay{dir: d, db: DB{Users: map[string]*User{}, Files: map[string]FileMeta{}}, sessions: map[string]Session{}, lastSeen: map[string]time.Time{}}
}
func hit(t *testing.T, s *Relay, method, path, token string, payload any) (int, map[string]any) {
	t.Helper()
	var data []byte
	if payload != nil {
		data, _ = json.Marshal(payload)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(data))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.handler(w, r)
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	return w.Code, got
}
func register(t *testing.T, s *Relay, n string) string {
	t.Helper()
	c, j := hit(t, s, "POST", "/api/register", "", map[string]string{"username": n, "password": "very_secure_password_2026"})
	if c != 200 {
		t.Fatalf("register %s: %d %#v", n, c, j)
	}
	return j["token"].(string)
}
func TestFriendshipMessagesAndPrivacy(t *testing.T) {
	s := setup(t)
	alice := register(t, s, "alice")
	bob := register(t, s, "bob")
	eve := register(t, s, "eve")
	for _, tt := range []struct {
		path  string
		token string
		data  map[string]string
		want  int
	}{
		{"/api/messages", alice, map[string]string{"to": "bob", "text": "hi"}, 403},
		{"/api/request", alice, map[string]string{"username": "bob"}, 200},
		{"/api/accept", bob, map[string]string{"username": "alice"}, 200},
		{"/api/messages", alice, map[string]string{"to": "bob", "text": "hello"}, 200},
		{"/api/messages", eve, map[string]string{"to": "bob", "text": "spam"}, 403},
	} {
		c, j := hit(t, s, "POST", tt.path, tt.token, tt.data)
		if c != tt.want {
			t.Fatalf("%s got %d expected %d: %#v", tt.path, c, tt.want, j)
		}
	}
	for _, tt := range []struct {
		user, peer string
		want       int
	}{{alice, "bob", 200}, {bob, "alice", 200}, {eve, "alice", 403}} {
		c, j := hit(t, s, "GET", "/api/messages?peer="+tt.peer, tt.user, nil)
		if c != tt.want {
			t.Fatalf("messages GET expected %d, got %d %#v", tt.want, c, j)
		}
		if c == 200 && len(j["messages"].([]any)) != 1 {
			t.Fatalf("expected a message: %#v", j)
		}
	}
	c, _ := hit(t, s, "POST", "/api/block", bob, map[string]string{"username": "alice"})
	if c != 200 {
		t.Fatal("block failed")
	}
	c, _ = hit(t, s, "GET", "/api/messages?peer=bob", alice, nil)
	if c != 403 {
		t.Fatal("blocked friend still reading messages")
	}
}
func TestFilesCannotBeFetchedByThirdParty(t *testing.T) {
	s := setup(t)
	alice := register(t, s, "alice")
	bob := register(t, s, "bob")
	eve := register(t, s, "eve")
	hit(t, s, "POST", "/api/request", alice, map[string]string{"username": "bob"})
	hit(t, s, "POST", "/api/accept", bob, map[string]string{"username": "alice"})
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	w, _ := mw.CreateFormFile("file", "hi.txt")
	fmt.Fprint(w, "test file")
	mw.Close()
	r := httptest.NewRequest("POST", "/api/upload?to=bob", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Authorization", "Bearer "+alice)
	resp := httptest.NewRecorder()
	s.handler(resp, r)
	if resp.Code != 200 {
		t.Fatalf("upload %d: %s", resp.Code, resp.Body.String())
	}
	var msg Message
	json.Unmarshal(resp.Body.Bytes(), &msg)
	if len(msg.FileID) == 0 {
		t.Fatal("missing file id")
	}
	for _, tc := range []struct {
		token  string
		expect int
	}{{alice, 200}, {bob, 200}, {eve, 403}} {
		c, _ := hit(t, s, "GET", "/api/file?id="+msg.FileID, tc.token, nil)
		if c != tc.expect {
			t.Errorf("file access: got %d want %d", c, tc.expect)
		}
	}
}
func TestBadLoginAndCors(t *testing.T) {
	s := setup(t)
	register(t, s, "alice")
	c, _ := hit(t, s, "POST", "/api/login", "", map[string]string{"username": "alice", "password": "wrong_password_123"})
	if c != 401 {
		t.Fatalf("bad login returned %d", c)
	}
	r := httptest.NewRequest("GET", "/api/health", nil)
	r.Header.Set("Origin", "https://random-website.example")
	w := httptest.NewRecorder()
	s.handler(w, r)
	if w.Code != 403 {
		t.Fatal("bad origin allowed")
	}
	r = httptest.NewRequest("OPTIONS", "/api/messages", strings.NewReader(""))
	r.Header.Set("Origin", "http://127.0.0.1:5050")
	w = httptest.NewRecorder()
	s.handler(w, r)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("desktop app CORS preflight failed")
	}
}

func TestSignalingOnlyBetweenFriends(t *testing.T) {
	s := setup(t)
	alice := register(t, s, "alice")
	bob := register(t, s, "bob")
	eve := register(t, s, "eve")
	signal := map[string]any{"to": "bob", "kind": "offer", "data": map[string]string{"type": "offer", "sdp": "test"}}
	c, _ := hit(t, s, "POST", "/api/signals", alice, signal)
	if c != 403 {
		t.Fatal("signal between strangers accepted")
	}
	hit(t, s, "POST", "/api/request", alice, map[string]string{"username": "bob"})
	hit(t, s, "POST", "/api/accept", bob, map[string]string{"username": "alice"})
	c, _ = hit(t, s, "POST", "/api/signals", alice, signal)
	if c != 200 {
		t.Fatalf("signal between friends denied: %d", c)
	}
	c, r := hit(t, s, "GET", "/api/signals?after=0", bob, nil)
	if c != 200 || len(r["signals"].([]any)) != 1 {
		t.Fatalf("offer missing for friend: %d %#v", c, r)
	}
	c, r = hit(t, s, "GET", "/api/signals?after=0", eve, nil)
	if c != 200 || len(r["signals"].([]any)) != 0 {
		t.Fatalf("signal leaked to stranger: %d %#v", c, r)
	}
}
