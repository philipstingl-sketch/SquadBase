package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSpotifyRedirectFixedPort(t *testing.T) {
	s := newSpotify(t.TempDir())
	s.clientID = strings.Repeat("b", 32)
	s.redirectURI = spotifyCallbackURI
	link, err := s.authURL()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Query().Get("redirect_uri"); got != spotifyCallbackURI {
		t.Fatalf("redirect mismatch: %q", got)
	}
	if parsed.Query().Get("code_challenge_method") != "S256" || parsed.Query().Get("state") == "" {
		t.Fatal("PKCE/state missing")
	}
	st := s.status()
	if st["redirectRegister"] != spotifyCallbackURI {
		t.Fatal("dashboard URI wrong")
	}
}

func TestSpotifyCallbackRejectsInvalidState(t *testing.T) {
	s := newSpotify(t.TempDir())
	req := httptest.NewRequest("GET", spotifyCallbackURI+"?code=notvalid&state=wrong", nil)
	w := httptest.NewRecorder()
	s.callback(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected state rejection, got %d", w.Code)
	}
	body, _ := io.ReadAll(w.Result().Body)
	if !strings.Contains(string(body), "Sicherheitsprüfung") {
		t.Fatalf("wrong response: %s", body)
	}
}

func TestSpotifyPortOccupiedShowsClearMessage(t *testing.T) {
	s := newSpotify(t.TempDir())
	s.clientID = strings.Repeat("b", 32)
	s.callbackError = "Spotify-Port 8888 belegt"
	_, err := s.authURL()
	if err == nil || !strings.Contains(err.Error(), "8888") {
		t.Fatalf("missing port warning: %v", err)
	}
}

func TestSpotifyCallbackListenerResponds(t *testing.T) {
	s := newSpotify(t.TempDir())
	server, err := startSpotifyCallbackServer(s)
	if err != nil {
		t.Skipf("Port 8888 already in use on test host: %v", err)
	}
	defer func() { _ = server.Close() }()
	if s.redirectURI != spotifyCallbackURI {
		t.Fatalf("wrong configured URI: %q", s.redirectURI)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(spotifyCallbackURI + "?code=bogus&state=bogus")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("invalid state must be rejected; got %d", res.StatusCode)
	}
}
