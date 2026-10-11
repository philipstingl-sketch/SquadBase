package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestV35ClockAndAppearancePersists(t *testing.T) {
	dir := t.TempDir()
	a := &App{prefix: "/s/test/", folder: dir, lastSeen: time.Now(), appearance: defaultAppearance(), clock: defaultClock()}
	for _, tt := range []struct{ route, payload string }{
		{"api/clock", `{"style":"flip","seconds":false,"twentyFour":true,"date":true,"weekday":true,"session":false,"localBadge":true}`},
		{"api/appearance", `{"color":"violet","scene":"rain","motion":true,"speed":"slow","cozy":"extra"}`},
	} {
		req := httptest.NewRequest("POST", "http://127.0.0.1/s/test/"+tt.route, strings.NewReader(tt.payload))
		req.RemoteAddr = "127.0.0.1:49990"
		req.Header.Set("X-Squad-Base", "1")
		w := httptest.NewRecorder()
		a.handle(w, req)
		if w.Code != 200 {
			t.Fatalf("post %s failed: %d %s", tt.route, w.Code, w.Body.String())
		}
	}
	var cp ClockConfig
	loadJSON(filepath.Join(dir, "clock.json"), &cp)
	if cp.Style != "flip" || cp.Seconds {
		t.Fatalf("bad clock state %+v", cp)
	}
	var ap AppearanceConfig
	loadJSON(filepath.Join(dir, "appearance.json"), &ap)
	if ap.Cozy != "extra" || ap.Scene != "rain" {
		t.Fatalf("bad appearance state %+v", ap)
	}
	b := &App{prefix: "/s/next/", folder: dir, lastSeen: time.Now(), clock: cp, appearance: ap}
	for _, v := range []struct{ route, expected string }{{"api/clock", "\"style\":\"flip\""}, {"api/appearance", "\"cozy\":\"extra\""}} {
		req := httptest.NewRequest("GET", "http://127.0.0.1/s/next/"+v.route, nil)
		req.RemoteAddr = "127.0.0.1:44999"
		w := httptest.NewRecorder()
		b.handle(w, req)
		if w.Code != 200 || !strings.Contains(w.Body.String(), v.expected) {
			t.Errorf("GET %s %d %s", v.route, w.Code, w.Body.String())
		}
	}
}
func TestV35RejectInvalidClockAndUntrustedRequests(t *testing.T) {
	a := &App{prefix: "/s/test/", folder: t.TempDir(), clock: defaultClock()}
	for _, tt := range []struct {
		body    string
		trusted bool
		code    int
	}{{`{"style":"not-real"}`, true, 422}, {`{"style":"flip"}`, false, 403}} {
		req := httptest.NewRequest("POST", "http://127.0.0.1/s/test/api/clock", strings.NewReader(tt.body))
		req.RemoteAddr = "127.0.0.1:55555"
		if tt.trusted {
			req.Header.Set("X-Squad-Base", "1")
		}
		w := httptest.NewRecorder()
		a.handle(w, req)
		if w.Code != tt.code {
			t.Fatalf("unexpected status %d wanted %d", w.Code, tt.code)
		}
	}
}
func TestV35EmbeddedUI(t *testing.T) {
	a := &App{prefix: "/s/test/", lastSeen: time.Now()}
	for _, tt := range []struct{ asset, want string }{{"index.html", "data-clock-style=\"flip\""}, {"index.html", "data-cozy=\"extra\""}, {"index.html", "data-overlay-preset=\"performance\""}, {"clock.js", "squad.base.clock.v1"}, {"cozy-v35.css", ".flip-clock"}} {
		r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/s/test/"+tt.asset, nil)
		r.RemoteAddr = "127.0.0.1:30000"
		w := httptest.NewRecorder()
		a.handle(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), tt.want) {
			t.Errorf("%s: code %d missing %q", tt.asset, w.Code, tt.want)
		}
	}

}
