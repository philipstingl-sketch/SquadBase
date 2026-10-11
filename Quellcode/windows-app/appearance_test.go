package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppearancePersistsBetweenAppStarts(t *testing.T) {
	dir := t.TempDir()
	a := &App{prefix: "/s/abcd/", folder: dir, appearance: defaultAppearance()}
	val := `{"color":"violet","scene":"rain","motion":false,"speed":"normal"}`
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:5500/s/abcd/api/appearance", strings.NewReader(val))
	req.RemoteAddr = "127.0.0.1:3555"
	req.Header.Set("X-Squad-Base", "1")
	resp := httptest.NewRecorder()
	a.handle(resp, req)
	if resp.Code != 200 {
		t.Fatalf("POST appearance: %d: %s", resp.Code, resp.Body.String())
	}
	var stored AppearanceConfig
	loadJSON(filepath.Join(dir, "appearance.json"), &stored)
	if !validAppearance(stored) || stored.Color != "violet" || stored.Motion != false || stored.Scene != "rain" {
		t.Fatalf("saved state: %+v", stored)
	}
	b := &App{prefix: "/s/second/", folder: dir, appearance: stored}
	req2 := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:5540/s/second/api/appearance", nil)
	req2.RemoteAddr = "127.0.0.1:3556"
	resp2 := httptest.NewRecorder()
	b.handle(resp2, req2)
	var got AppearanceConfig
	if err := json.Unmarshal(resp2.Body.Bytes(), &got); err != nil || got != stored {
		t.Fatalf("GET appearance: %s, %+v", resp2.Body.String(), got)
	}
}
func TestAppearanceRejectsInvalidAndUntrustedPost(t *testing.T) {
	a := &App{prefix: "/s/test/", folder: t.TempDir(), appearance: defaultAppearance()}
	sendPost := func(body string, token bool) int {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:5500/s/test/api/appearance", strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:2345"
		if token {
			req.Header.Set("X-Squad-Base", "1")
		}
		w := httptest.NewRecorder()
		a.handle(w, req)
		return w.Code
	}
	if got := sendPost(`{"color":"oak","scene":"aurora","motion":true,"speed":"fast"}`, false); got != 403 {
		t.Fatal("missing header", got)
	}
	if got := sendPost(`{"color":"not_a_theme","scene":"oak","motion":true,"speed":"fast"}`, true); got != 422 {
		t.Fatal("invalid theme", got)
	}
	if got := sendPost(`{"color":"oak","scene":"../","motion":true,"speed":"fast"}`, true); got != 422 {
		t.Fatal("invalid scene", got)
	}
}
