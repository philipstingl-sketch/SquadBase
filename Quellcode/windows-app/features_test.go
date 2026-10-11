package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSpotifyPlaylistLinkValidation(t *testing.T) {
	id := "3cEYpjA9oz9GiPac4AsH4n"
	for _, s := range []string{"https://open.spotify.com/playlist/" + id, "https://open.spotify.com/playlist/" + id + "?si=testing", "spotify:playlist:" + id} {
		got, e := extractSpotifyPlaylistID(s)
		if e != nil || got != id {
			t.Fatalf("valid Spotify ID failed for %q: %s %v", s, got, e)
		}
	}
	for _, s := range []string{"http://open.spotify.com/playlist/" + id, "https://evil.example/playlist/" + id, "https://open.spotify.com.evil.example/playlist/" + id, "https://open.spotify.com/track/" + id, "file:///etc/passwd", "https://open.spotify.com/playlist/x"} {
		if _, e := extractSpotifyPlaylistID(s); e == nil {
			t.Errorf("unsafe URI accepted: %s", s)
		}
	}
}
func TestSpotifyPKCEAndStatusNoTokens(t *testing.T) {
	s := newSpotify(t.TempDir())
	s.clientID = strings.Repeat("a", 32)
	s.redirectURI = "http://127.0.0.1:5555/spotify/callback"
	target, e := s.authURL()
	if e != nil {
		t.Fatal(e)
	}
	u, e := url.Parse(target)
	if e != nil {
		t.Fatal(e)
	}
	if u.Host != "accounts.spotify.com" || u.Query().Get("code_challenge_method") != "S256" || len(u.Query().Get("state")) < 30 || u.Query().Get("code_verifier") != "" {
		t.Fatalf("bad authorize url: %s", target)
	}
	s.token = SpotifyToken{AccessToken: "PRIVATE", RefreshToken: "SUPERSECRET", ExpiresAt: time.Now().Add(time.Hour)}
	v, _ := json.Marshal(s.status())
	if strings.Contains(string(v), "PRIVATE") || strings.Contains(string(v), "SUPERSECRET") {
		t.Fatal("OAuth tokens exposed in status JSON")
	}
}
func TestSpotifySavedPlaylistBookmarkAPI(t *testing.T) {
	dir := t.TempDir()
	a := &App{prefix: "/s/test/", lastSeen: time.Now(), spotify: newSpotify(dir), news: newNews()}
	a.spotify.redirectURI = "http://127.0.0.1:9988/spotify/callback"
	req := httptest.NewRequest("POST", "http://127.0.0.1:9988/s/test/api/spotify/bookmarks", strings.NewReader(`{"action":"add","url":"https://open.spotify.com/playlist/3cEYpjA9oz9GiPac4AsH4n","name":"Meine Musik"}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Squad-Base", "1")
	req.Header.Set("Origin", "http://127.0.0.1:9988")
	w := httptest.NewRecorder()
	a.handle(w, req)
	if w.Code != 200 {
		t.Fatalf("bookmark failed %d: %s", w.Code, w.Body.String())
	}
	a.spotify = newSpotify(dir)
	if len(a.spotify.bookmarks) != 1 || a.spotify.bookmarks[0].Name != "Meine Musik" {
		t.Fatal("Spotify bookmarks not saved and loaded")
	}
}
func TestNewsDataAndSafeText(t *testing.T) {
	if x := cleanNews("<b>Patch</b> &amp; <div>News</div>"); x != "Patch & News" {
		t.Fatalf("news HTML filtering broken: %q", x)
	}
	n := newNews()
	n.data = NewsData{Items: []GameNews{{Game: "gta", Title: "Test", URL: "https://store.steampowered.com/", Date: time.Now().Format(time.RFC3339)}}}
	n.next = time.Now().Add(time.Hour)
	rec := httptest.NewRecorder()
	n.handleAPI(rec, httptest.NewRequest(http.MethodGet, "/api/news", nil), "news")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Test") {
		t.Fatalf("news cache response: %d %s", rec.Code, rec.Body.String())
	}
}
func TestNewWebAssets(t *testing.T) {
	a := &App{prefix: "/s/test/", lastSeen: time.Now()}
	for _, name := range []string{"spotify.js", "updates.js", "index.html", "style.css"} {
		r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:7733/s/test/"+name, nil)
		r.RemoteAddr = "127.0.0.1:10000"
		w := httptest.NewRecorder()
		a.handle(w, r)
		if w.Code != 200 {
			t.Errorf("missing %s: %d", name, w.Code)
		}
	}
}

type apiTransport func(*http.Request) (*http.Response, error)

func (f apiTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSpotifyPlaybackAPIProxy(t *testing.T) {
	s := newSpotify(t.TempDir())
	s.clientID = strings.Repeat("b", 32)
	s.token = SpotifyToken{AccessToken: "mock", RefreshToken: "refresh", ExpiresAt: time.Now().Add(time.Hour)}
	count := 0
	s.client = &http.Client{Transport: apiTransport(func(r *http.Request) (*http.Response, error) {
		count++
		if got := r.Header.Get("Authorization"); got != "Bearer mock" {
			t.Errorf("missing Spotify bearer token: %s", got)
		}
		body := `{}`
		if r.URL.Path == "/v1/me/playlists" {
			body = `{"total":1,"items":[{"id":"3cEYpjA9oz9GiPac4AsH4n","name":"Actual Spotify Test","images":[],"items":{"total":20}}]}`
		}
		if r.URL.Path == "/v1/me/player" {
			body = `{"is_playing":true,"item":{"name":"Test Track","artists":[{"name":"Test Artist"}],"duration_ms":185000},"progress_ms":5000,"device":{"name":"Windows Spotify"}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	a := &App{spotify: s, prefix: "/s/test/", lastSeen: time.Now()}
	for _, endpoint := range []string{"playlists", "now"} {
		req := httptest.NewRequest("GET", "http://127.0.0.1:8989/s/test/api/spotify/"+endpoint, nil)
		req.RemoteAddr = "127.0.0.1:2222"
		rec := httptest.NewRecorder()
		a.handle(rec, req)
		if rec.Code != 200 {
			t.Fatalf("Spotify %s: %s", endpoint, rec.Body.String())
		}
		if endpoint == "now" && !strings.Contains(rec.Body.String(), "Test Track") {
			t.Fatalf("Spotify song missing: %s", rec.Body.String())
		}
	}
	if count != 2 {
		t.Errorf("expected exactly two Spotify requests, got %d", count)
	}
}
func TestOfficialMinecraftRSSFeed(t *testing.T) {
	n := newNews()
	n.client = &http.Client{Transport: apiTransport(func(req *http.Request) (*http.Response, error) {
		data := `<?xml version="1.0"?><rss version="2.0"><channel><item><title>Game Update</title><link>https://www.minecraft.net/en-us/article/test-update</link><pubDate>Fri, 09 Oct 2026 12:00:00 +0000</pubDate><description>Some &amp; cool news</description></item><item><title>Unsafe</title><link>https://evil.example/</link><pubDate>Fri, 09 Oct 2026 12:00:00 +0000</pubDate></item></channel></rss>`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(data)), Header: make(http.Header)}, nil
	})}
	it, e := n.fetchMinecraft()
	if e != nil {
		t.Fatal(e)
	}
	if len(it) != 1 || it[0].Title != "Game Update" || it[0].Source != "Minecraft.net · RSS" {
		t.Fatalf("unexpected RSS parsing: %+v", it)
	}
}
