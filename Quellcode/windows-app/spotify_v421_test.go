package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSpotifyRefreshOnUnexpected401(t *testing.T) {
	s := newSpotify(t.TempDir())
	s.clientID = strings.Repeat("a", 32)
	s.token = SpotifyToken{AccessToken: "old", RefreshToken: "refresh-old", ExpiresAt: time.Now().Add(time.Hour)}
	tokenRequests, playerRequests := 0, 0
	s.client = &http.Client{Transport: apiTransport(func(r *http.Request) (*http.Response, error) {
		code, body := 200, `{}`
		switch r.URL.Host {
		case "api.spotify.com":
			playerRequests++
			if playerRequests == 1 {
				code = 401
				body = `{"error":{"message":"expired"}}`
			}
			if playerRequests == 2 && r.Header.Get("Authorization") != "Bearer new" {
				t.Errorf("refreshed token not used: %q", r.Header.Get("Authorization"))
			}
		case "accounts.spotify.com":
			tokenRequests++
			body = `{"access_token":"new","expires_in":3600}`
		default:
			t.Errorf("unexpected service %s", r.URL.Host)
		}
		return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	_, _, err := s.spotifyRequest(context.Background(), "GET", "/me/player", nil)
	if err != nil {
		t.Fatal(err)
	}
	if tokenRequests != 1 || playerRequests != 2 {
		t.Fatalf("unexpected request counts token=%d player=%d", tokenRequests, playerRequests)
	}
	if s.token.RefreshToken != "refresh-old" {
		t.Fatal("refresh token should be preserved when Spotify omits it")
	}
}

func TestSpotifyFallbackAPIWithNoUserSecrets(t *testing.T) {
	a := &App{spotify: newSpotify(t.TempDir()), prefix: "/s/test/", lastSeen: time.Now()}
	get := httptest.NewRequest("GET", "http://127.0.0.1:7890/s/test/api/spotify/status", nil)
	get.RemoteAddr = "127.0.0.1:60000"
	w := httptest.NewRecorder()
	a.handle(w, get)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "desktopControls") {
		t.Fatalf("bad desktop status %d %s", w.Code, w.Body.String())
	}
	post := httptest.NewRequest("POST", "http://127.0.0.1:7890/s/test/api/spotify/desktop", strings.NewReader(`{"action":"not-supported"}`))
	post.RemoteAddr = "127.0.0.1:60000"
	post.Header.Set("Origin", "http://127.0.0.1:7890")
	post.Header.Set("X-Squad-Base", "1")
	w = httptest.NewRecorder()
	a.handle(w, post)
	if w.Code != 422 {
		t.Fatalf("should reject unknown desktop action: %d %s", w.Code, w.Body.String())
	}
}
