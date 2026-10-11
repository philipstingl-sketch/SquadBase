package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppearanceAssetsV26(t *testing.T) {
	a := &App{prefix: "/s/test/"}
	for _, tc := range []struct{ path, mime, needle string }{
		{"index.html", "text/html", "theme-colors"},
		{"style.css", "text/css", "theme-scene-grid"},
		{"themes.js", "text/javascript", "squad.base.appearance.v1"},
		{"hardware-chart.js", "text/javascript", "squad-appearance-changed"},
	} {
		r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/s/test/"+tc.path, nil)
		r.RemoteAddr = "127.0.0.1:3021"
		w := httptest.NewRecorder()
		a.handle(w, r)
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), tc.mime) || !strings.Contains(w.Body.String(), tc.needle) {
			t.Errorf("%s status=%d type=%s", tc.path, w.Code, w.Header().Get("Content-Type"))
		}
	}
}
