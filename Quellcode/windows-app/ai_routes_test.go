package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewAppAssetsAndRoutes(t *testing.T) {
	a := &App{folder: t.TempDir(), prefix: "/s/check/", lastSeen: time.Now()}
	for _, file := range []string{"ai-images.js", "assistant.js", "minigames.js", "index.html", "style.css"} {
		r := httptest.NewRequest("GET", "http://127.0.0.1:4711/s/check/"+file, nil)
		r.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		a.handle(w, r)
		if w.Code != 200 || w.Body.Len() < 20 {
			t.Errorf("embedded resource %s returned %d", file, w.Code)
		}
	}
	r := httptest.NewRequest("GET", "http://127.0.0.1:4711/s/check/api/ai-image/gallery", nil)
	r.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	a.handle(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "images") {
		t.Fatalf("gallery route: %d %s", w.Code, w.Body.String())
	}
}
