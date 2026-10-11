package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGlass45Valid(t *testing.T) {
	g := defaultGlass45()
	if !validGlass45(g) {
		t.Fatal("default invalid")
	}
	for _, preset := range []string{"original", "moonstone", "boreal", "rosewood", "starlight"} {
		g.Preset = preset
		if !validGlass45(g) {
			t.Fatalf("preset %q invalid", preset)
		}
	}
	g.Blur = 31
	if validGlass45(g) {
		t.Fatal("invalid blur accepted")
	}
	g = defaultGlass45()
	g.Preset = "unknown"
	if validGlass45(g) {
		t.Fatal("invalid preset accepted")
	}
}
func TestGlass45APIAndStorage(t *testing.T) {
	folder := t.TempDir()
	a := &App{folder: folder, glass45: defaultGlass45()}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	a.glass45API(w, req)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	g := Glass45{Preset: "boreal", Blur: 15, Opacity: 73, Radius: 25, Glow: true, Ambient: false}
	payload, _ := json.Marshal(g)
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	w = httptest.NewRecorder()
	a.glass45API(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	data, e := os.ReadFile(filepath.Join(folder, "glass45.json"))
	if e != nil {
		t.Fatal(e)
	}
	var got Glass45
	if json.Unmarshal(data, &got) != nil || got != g {
		t.Fatalf("not saved: %+v", got)
	}
	g.Blur = 99
	payload, _ = json.Marshal(g)
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	w = httptest.NewRecorder()
	a.glass45API(w, req)
	if w.Code != 422 {
		t.Fatalf("bad value accepted: %d", w.Code)
	}
}
func TestBackupCompatibilityGlass45(t *testing.T) {
	a := &App{glass45: defaultGlass45(), appearance: defaultAppearance(), clock: defaultClock(), overlay: defaultOverlay(), studio: defaultStudio(), workspace: defaultWorkspace()}
	a.glass45.Preset = "rosewood"
	b := a.settingsSnapshot()
	if b.Glass45 == nil || b.Glass45.Preset != "rosewood" {
		t.Fatal("glass config not in backup")
	}
	if !validSettingsBackup(b) {
		t.Fatal("new backup invalid")
	}
	b.Glass45 = nil
	if !validSettingsBackup(b) {
		t.Fatal("old backup without glass config rejected")
	}
}
