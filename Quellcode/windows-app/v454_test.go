package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV454AutoclickerSafetyPersistence(t *testing.T) {
	dir := t.TempDir()
	c := newAutoClicker(dir)
	defer c.shutdown()
	defer func() {
		if c.snapshot().Active {
			t.Fatal("must stop autoclicker")
		}
	}()
	if c.snapshot().Active {
		t.Fatal("must be off by default")
	}
	for _, hot := range []string{"F6", "F8", "F12", "CTRL+ALT+G"} {
		if err := validateClickerSettings(ClickerSettings{200, "left", hot}); err != nil {
			t.Error(hot, err)
		}
	}
	for _, q := range []ClickerSettings{{50, "left", "F8"}, {200, "middle", "F8"}, {200, "left", "CTRL+SHIFT+P"}, {6000, "right", "F8"}} {
		if validateClickerSettings(q) == nil {
			t.Errorf("unexpectedly permitted %+v", q)
		}
	}
	next := ClickerSettings{IntervalMs: 340, MouseButton: "right", Hotkey: "CTRL+ALT+Q"}
	if err := c.action("configure", next); err != nil {
		t.Fatal(err)
	}
	if c.snapshot().Active {
		t.Fatal("configure must stop")
	}
	file, err := os.ReadFile(filepath.Join(dir, "autoclicker.json"))
	if err != nil || !bytes.Contains(file, []byte(`"intervalMs": 340`)) {
		t.Fatalf("not persisted %s %v", file, err)
	}
	c.shutdown()
	again := newAutoClicker(dir)
	defer again.shutdown()
	if got := again.snapshot(); got.IntervalMs != 340 || got.MouseButton != "right" || got.Hotkey != "CTRL+ALT+Q" || got.Active {
		t.Fatalf("bad restore %+v", got)
	}
}
func TestV454AutoclickerLocalAPI(t *testing.T) {
	c := newAutoClicker(t.TempDir())
	defer c.shutdown()
	a := &App{clicker: c}
	rr := httptest.NewRecorder()
	a.clickerAPI(rr, httptest.NewRequest(http.MethodGet, "/api/autoclicker", nil))
	if rr.Code != 200 {
		t.Fatalf("GET %d", rr.Code)
	}
	for _, tc := range []struct {
		payload string
		status  int
	}{{`{"action":"configure","intervalMs":100,"mouseButton":"left","hotkey":"F9"}`, 200}, {`{"action":"configure","intervalMs":1,"mouseButton":"left","hotkey":"F9"}`, 400}, {`{"action":"unknown"}`, 400}} {
		r := httptest.NewRecorder()
		a.clickerAPI(r, httptest.NewRequest(http.MethodPost, "/api/autoclicker", strings.NewReader(tc.payload)))
		if r.Code != tc.status {
			t.Errorf("%s => %d want %d", tc.payload, r.Code, tc.status)
		}
	}
}
func TestV454FriendPrefsPersistence(t *testing.T) {
	dir := t.TempDir()
	p := newFriendPrefStore(dir)
	if !p.prefs.Messages || !p.prefs.Calls || !p.prefs.Invites || p.prefs.Volume != 55 {
		t.Fatal("invalid defaults")
	}
	updated := FriendPrefs{Messages: false, Calls: true, Sent: true, Invites: false, Volume: 75, Volumes: map[string]int{"bob": 22}}
	b, _ := json.Marshal(updated)
	rr := httptest.NewRecorder()
	p.api(rr, httptest.NewRequest(http.MethodPost, "/api/friend-prefs", bytes.NewReader(b)))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	again := newFriendPrefStore(dir)
	if again.prefs.Volume != 75 || again.prefs.Volumes["bob"] != 22 || again.prefs.Messages {
		t.Fatalf("bad prefs %+v", again.prefs)
	}
	bad := FriendPrefs{Volume: 101, Volumes: map[string]int{"bob": 20}}
	buf, _ := json.Marshal(bad)
	rr = httptest.NewRecorder()
	p.api(rr, httptest.NewRequest(http.MethodPost, "/api/friend-prefs", bytes.NewReader(buf)))
	if rr.Code != 400 {
		t.Fatalf("invalid prefs accepted: %d", rr.Code)
	}
}
func TestV454Assets(t *testing.T) {
	for _, f := range []string{"v454.js", "v454.css", "friend-alerts.js"} {
		b, err := assets.ReadFile("web/" + f)
		if err != nil || len(b) < 100 {
			t.Fatalf("asset missing %s: %v", f, err)
		}
	}
	b, err := assets.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"v454-clicker-hotkey", "v454-clicker-bind", "v454-toggle-camera", "v454-notify-calls", "v454-test-tone", "v454-accept-video"} {
		if !bytes.Contains(b, []byte(`id="`+id+`"`)) {
			t.Errorf("missing UI %s", id)
		}
	}
}

func TestV454RoutesServeNewAssetsAndAPIs(t *testing.T) {
	dir := t.TempDir()
	click := newAutoClicker(dir)
	defer click.shutdown()
	a := &App{prefix: "/s/test/", clicker: click, friendPrefs: newFriendPrefStore(dir)}
	for _, path := range []string{"v454.js", "v454.css", "friend-alerts.js", "api/autoclicker", "api/friend-prefs"} {
		request := httptest.NewRequest(http.MethodGet, "http://localhost:1234/s/test/"+path, nil)
		request.RemoteAddr = "127.0.0.1:45454"
		response := httptest.NewRecorder()
		a.handle(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("route %s => %d %s", path, response.Code, response.Body.String())
		}
	}
}
