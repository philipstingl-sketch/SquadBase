package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestV34DashboardLayout(t *testing.T) {
	a := &App{prefix: "/s/test/", lastSeen: time.Now()}
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:10000/s/test/", nil)
	req.RemoteAddr = "127.0.0.1:56789"
	out := httptest.NewRecorder()
	a.handle(out, req)
	if out.Code != http.StatusOK {
		t.Fatalf("UI returned %d: %s", out.Code, out.Body.String())
	}
	html := out.Body.String()
	for _, must := range []string{"class=\"system-spotlight\"", "id=\"spotlight-game\"", "id=\"spotlight-hours\"", "id=\"hw-pro-chart\"", "Cube Rush 4.1", "v4.1"} {
		if !strings.Contains(html, must) {
			t.Errorf("missing UI element %q", must)
		}
	}
	for _, duplicate := range []string{"id=\"hardware-chart\"", "game-news-prompt"} {
		if strings.Contains(html, duplicate) {
			t.Errorf("duplicate UI element remains %q", duplicate)
		}
	}
}

func TestV34EmbeddedAssets(t *testing.T) {
	a := &App{prefix: "/s/test/", lastSeen: time.Now()}
	tests := map[string]string{
		"style.css":             "text/css",
		"app.js":                "text/javascript",
		"minigames.js":          "text/javascript",
		"hardware-pro.js":       "text/javascript",
		"ai-images.js":          "text/javascript",
		"squad_cozy_orbit.ogg":  "audio/ogg",
		"squad_neon_sprint.ogg": "audio/ogg",
	}
	for path, want := range tests {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:10000/s/test/"+path, nil)
		req.RemoteAddr = "127.0.0.1:55432"
		out := httptest.NewRecorder()
		a.handle(out, req)
		if out.Code != 200 || out.Body.Len() == 0 || !strings.Contains(out.Header().Get("Content-Type"), want) {
			t.Errorf("asset %s returned %d (type %s, bytes %d)", path, out.Code, out.Header().Get("Content-Type"), out.Body.Len())
		}
	}
}
