package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestV36AppearanceSettingsPersistAndValidate(t *testing.T) {
	dir := t.TempDir()
	a := &App{prefix: "/s/demo/", folder: dir, appearance: defaultAppearance(), lastSeen: time.Now()}
	good := `{"color":"ocean","scene":"snow","motion":true,"speed":"slow","cozy":"extra","brightness":115,"dim":20,"vignette":45,"density":"high"}`
	for _, tc := range []struct {
		payload string
		code    int
	}{
		{good, 200},
		{`{"color":"ocean","scene":"snow","motion":true,"speed":"slow","cozy":"extra","brightness":300,"dim":20,"vignette":45,"density":"high"}`, 422},
		{`{"color":"oak","scene":"unknown","motion":true,"speed":"slow","cozy":"cozy"}`, 422},
		{`{"color":"oak","scene":"clouds","motion":true,"speed":"slow","cozy":"cozy","density":"infinite"}`, 422},
	} {
		req := httptest.NewRequest("POST", "http://127.0.0.1/s/demo/api/appearance", strings.NewReader(tc.payload))
		req.RemoteAddr = "127.0.0.1:54932"
		req.Header.Set("X-Squad-Base", "1")
		w := httptest.NewRecorder()
		a.handle(w, req)
		if w.Code != tc.code {
			t.Fatalf("post %s got %d, expected %d: %s", tc.payload, w.Code, tc.code, w.Body.String())
		}
	}
	var saved AppearanceConfig
	loadJSON(filepath.Join(dir, "appearance.json"), &saved)
	if saved.Scene != "snow" || saved.Brightness != 115 || saved.Dim != 20 || saved.Vignette != 45 || saved.Density != "high" {
		t.Fatalf("unexpected saved values %+v", saved)
	}
}
func TestV36CompactCubeAndThemeAssets(t *testing.T) {
	a := &App{prefix: "/s/demo/", lastSeen: time.Now()}
	for _, tc := range []struct{ path, needle string }{
		{"index.html", "id=\"cube-mode-details\""},
		{"index.html", "id=\"cube-shop-details\""},
		{"index.html", "id=\"cube-audio-details\""},
		{"index.html", "id=\"theme-brightness\""},
		{"index.html", "data-scene=\"snow\""},
		{"cozy-v36.css", ".cube-setting-group"},
		{"themes.js", "brightness:"},
		{"minigames.js", "refreshCubeSummary"},
	} {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/s/demo/"+tc.path, nil)
		req.RemoteAddr = "127.0.0.1:5555"
		w := httptest.NewRecorder()
		a.handle(w, req)
		if w.Code != 200 || !strings.Contains(w.Body.String(), tc.needle) {
			t.Errorf("asset %s failed %d, missing %s", tc.path, w.Code, tc.needle)
		}
	}
}
