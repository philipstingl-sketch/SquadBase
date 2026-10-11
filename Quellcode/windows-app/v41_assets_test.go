package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestV41AssetsServedFromLocalApp(t *testing.T) {
	a := &App{prefix: "/s/test/", lastSeen: time.Now()}
	for _, p := range []string{"v41.css", "v41.js"} {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9090/s/test/"+p, nil)
		req.RemoteAddr = "127.0.0.1:4343"
		w := httptest.NewRecorder()
		a.handle(w, req)
		if w.Code != 200 || w.Body.Len() < 100 || !strings.Contains(w.Header().Get("Content-Type"), map[string]string{"v41.css": "text/css", "v41.js": "text/javascript"}[p]) {
			t.Fatalf("asset %s: %d (%s)", p, w.Code, w.Body.String())
		}
	}
}
