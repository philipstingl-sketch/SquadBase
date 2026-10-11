package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestV42PreservesOlderWorkspace(t *testing.T) {
	old := WorkspacePrefs{Preset: "custom", Order: []string{"games", "system", "news", "network", "monitor"}, Hidden: []string{"monitor"}, Wide: []string{"games"}, ReadAloud: true, VoiceRate: 1.2, VoiceName: "Microsoft Anna", SpotifyDock: "floating"}
	migrated := migrateWorkspace(old)
	if !validWorkspace(migrated) {
		t.Fatalf("invalid migration: %+v", migrated)
	}
	if migrated.Version != 2 || !migrated.CozyStart || !migrated.SmartGaming || migrated.GlassStyle != "frosted" {
		t.Fatalf("new defaults missing: %+v", migrated)
	}
	if !reflect.DeepEqual(migrated.Order[:5], old.Order) {
		t.Fatal("legacy card order changed")
	}
	if !reflect.DeepEqual(migrated.Hidden[:1], old.Hidden) || !reflect.DeepEqual(migrated.Wide, old.Wide) {
		t.Fatal("legacy display preferences changed")
	}
	if migrated.VoiceRate != old.VoiceRate || migrated.VoiceName != old.VoiceName || migrated.SpotifyDock != old.SpotifyDock {
		t.Fatal("voice or music preferences changed")
	}
	if len(migrated.Hidden) != 5 || len(migrated.Order) != 9 {
		t.Fatal("new widgets not safely hidden by default")
	}
	// Once migrated, a user's deliberate disabled welcome screen stays disabled.
	migrated.CozyStart = false
	migrated.SmartGaming = false
	migrated.GlassStyle = "midnight"
	again := migrateWorkspace(migrated)
	if again.CozyStart || again.SmartGaming || again.GlassStyle != "midnight" {
		t.Fatal("migrated choice not retained")
	}
}
func TestV42WorkspacePOSTPersistence(t *testing.T) {
	a := &App{folder: t.TempDir(), workspace: defaultWorkspace()}
	updated := defaultWorkspace()
	updated.Hidden = []string{"network", "news"}
	updated.Order = []string{"clockwidget", "system", "monitor", "games", "network", "news", "musicwidget", "friendswidget", "sessionwidget"}
	updated.Preset = "custom"
	updated.CozyStart = false
	updated.SmartGaming = false
	updated.GlassStyle = "crystal"
	b, _ := json.Marshal(updated)
	w := httptest.NewRecorder()
	a.workspaceAPI(w, httptest.NewRequest(http.MethodPost, "/api/workspace", bytes.NewReader(b)))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var file WorkspacePrefs
	loadJSON(filepath.Join(a.folder, "workspace-v41.json"), &file)
	if !reflect.DeepEqual(updated, file) {
		t.Fatalf("workspace lost in file: %#v", file)
	}
	for _, style := range []string{"garbage", "", "../../"} {
		invalid := updated
		invalid.GlassStyle = style
		body, _ := json.Marshal(invalid)
		w := httptest.NewRecorder()
		a.workspaceAPI(w, httptest.NewRequest(http.MethodPost, "/api/workspace", bytes.NewReader(body)))
		if w.Code != 422 {
			t.Errorf("unexpected glass mode accepted: %s (%d)", style, w.Code)
		}
	}
}
func TestV42FilesActuallyServed(t *testing.T) {
	a := &App{prefix: "/s/testing/", lastSeen: time.Now()}
	for _, asset := range []string{"v42.js", "v42.css", "index.html"} {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9090/s/testing/"+asset, nil)
		req.RemoteAddr = "127.0.0.1:3333"
		w := httptest.NewRecorder()
		a.handle(w, req)
		if w.Code != 200 || w.Body.Len() < 100 {
			t.Fatalf("asset not embedded %s: %d", asset, w.Code)
		}
		if asset == "index.html" && !strings.Contains(w.Body.String(), "v42.js") {
			t.Fatal("app did not load v42")
		}
	}
}
