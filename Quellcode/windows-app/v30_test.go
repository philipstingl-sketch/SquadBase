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

func TestOverlayPersistenceAndValidation(t *testing.T) {
	dir := t.TempDir()
	a := &App{folder: dir, overlay: defaultOverlay()}
	v := defaultOverlay()
	v.CPU = false
	v.Music = false
	v.ClickThrough = true
	v.Scale = 125
	v.X = 27
	b, _ := json.Marshal(v)
	r := httptest.NewRequest(http.MethodPost, "/overlay", bytes.NewReader(b))
	w := httptest.NewRecorder()
	a.overlayAPI(w, r)
	if w.Code != 200 {
		t.Fatalf("overlay save: %d %s", w.Code, w.Body)
	}
	var saved OverlayConfig
	loadJSON(filepath.Join(dir, "overlay.json"), &saved)
	if saved.CPU || saved.Music || !saved.ClickThrough || saved.Scale != 125 || saved.X != 27 {
		t.Fatalf("overlay persistence failed: %+v", saved)
	}
	v.Opacity = 2
	b, _ = json.Marshal(v)
	w = httptest.NewRecorder()
	a.overlayAPI(w, httptest.NewRequest(http.MethodPost, "/overlay", bytes.NewReader(b)))
	if w.Code != 422 {
		t.Fatalf("invalid opacity accepted: %d", w.Code)
	}
}
func TestArcadeStorageAndKeySecurity(t *testing.T) {
	dir := t.TempDir()
	a := &App{folder: dir, arcade: map[string]json.RawMessage{}}
	keys := []string{"squad-cube-wallet-v3", "squad-cube-best-v2-extreme", "squad-cube-unlocked-v3"}
	for _, key := range keys {
		w := httptest.NewRecorder()
		body := `{"key":"` + key + `","value":42}`
		a.arcadeAPI(w, httptest.NewRequest(http.MethodPost, "/arcade", strings.NewReader(body)))
		if w.Code != 200 {
			t.Fatalf("save %s: %d", key, w.Code)
		}
	}
	var disk map[string]json.RawMessage
	b, err := os.ReadFile(filepath.Join(dir, "arcade.json"))
	if err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal(b, &disk) != nil || len(disk) != 3 {
		t.Fatalf("persisted state invalid %s", b)
	}
	w := httptest.NewRecorder()
	a.arcadeAPI(w, httptest.NewRequest(http.MethodGet, "/arcade", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "squad-cube-wallet-v3") {
		t.Fatal("cannot read save")
	}
	for _, key := range []string{"../secret", "squad-bad", "squad-cube-hacked/../../../other"} {
		w = httptest.NewRecorder()
		a.arcadeAPI(w, httptest.NewRequest(http.MethodPost, "/arcade", strings.NewReader(`{"key":"`+key+`","value":0}`)))
		if w.Code != 422 {
			t.Fatalf("accepted invalid key %q: %d", key, w.Code)
		}
	}
}
func TestOverlayAssetEmbedding(t *testing.T) {
	for _, name := range []string{"overlay.html", "overlay.js", "overlay.css", "hardware-pro.js", "overlay-settings.js", "minigames.js"} {
		b, e := assets.ReadFile("web/" + name)
		if e != nil || len(b) < 50 {
			t.Fatalf("web asset %q missing", name)
		}
	}
}
