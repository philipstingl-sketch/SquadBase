package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewV25AssetsAreServed(t *testing.T) {
	a := &App{prefix: "/s/test/"}
	checks := []struct{ path, contentType string }{
		{"assistant.js", "text/javascript"},
		{"hardware-chart.js", "text/javascript"},
		{"soundtrack.js", "text/javascript"},
		{"squad_cozy_orbit.ogg", "audio/ogg"},
		{"squad_neon_sprint.ogg", "audio/ogg"},
	}
	for _, tc := range checks {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/s/test/"+tc.path, nil)
		req.RemoteAddr = "127.0.0.1:23456"
		rr := httptest.NewRecorder()
		a.handle(rr, req)
		if rr.Code != 200 || !strings.HasPrefix(rr.Header().Get("Content-Type"), tc.contentType) || rr.Body.Len() < 100 {
			t.Errorf("%s status=%d contenttype=%s size=%d", tc.path, rr.Code, rr.Header().Get("Content-Type"), rr.Body.Len())
		}
	}
}
func TestAssistantRequiresCSRFHeader(t *testing.T) {
	a := &App{prefix: "/s/test/"}
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/s/test/api/assistant", strings.NewReader(`{"message":"Hallo"}`))
	req.RemoteAddr = "127.0.0.1:2334"
	rr := httptest.NewRecorder()
	a.handle(rr, req)
	if rr.Code != 403 {
		t.Fatalf("got status %d", rr.Code)
	}
}
func TestAssistantValidatesInput(t *testing.T) {
	a := &App{prefix: "/s/test/"}
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/s/test/api/assistant", strings.NewReader(`{"message":""}`))
	req.Header.Set("X-Squad-Base", "1")
	req.RemoteAddr = "127.0.0.1:2334"
	rr := httptest.NewRecorder()
	a.handle(rr, req)
	var res map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &res)
	if rr.Code != 400 || res["error"] == "" {
		t.Fatalf("got status %d body %s", rr.Code, rr.Body.String())
	}
}
