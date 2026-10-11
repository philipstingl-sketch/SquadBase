package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const playlistID = "3cEYpjA9oz9GiPac4AsH4n"
const trackID = "5yEPktRqvIhko5QFF3aBH3"

func newMockSpotify(t *testing.T, handler func(*http.Request) (int, string)) *SpotifyEngine {
	t.Helper()
	s := newSpotify(t.TempDir())
	s.clientID = strings.Repeat("a", 32)
	s.token = SpotifyToken{AccessToken: "mock", RefreshToken: "refresh", ExpiresAt: time.Now().Add(time.Hour)}
	s.client = &http.Client{Transport: apiTransport(func(req *http.Request) (*http.Response, error) {
		code, body := handler(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	return s
}
func TestSpotifyPlaylistItemParseAndPagination(t *testing.T) {
	raw := `{"total":81,"items":[{"track":{"id":"5yEPktRqvIhko5QFF3aBH3","name":"Erster Song","type":"track","duration_ms":200000,"artists":[{"name":"Künstler A"}],"album":{"name":"Album","images":[{"url":"https://i.scdn.co/image/cover"}]}}},{"track":null},{"item":{"id":"5yEPktRqvIhko5QFF3aBH3","name":"Neues Schema","type":"track","duration_ms":60000,"artists":[{"name":"B"}]}},{"track":{"id":"5yEPktRqvIhko5QFF3aBH3","name":"Lokal","type":"track","is_local":true}},{"track":{"id":"5yEPktRqvIhko5QFF3aBH3","name":"Podcast","type":"episode"}}]}`
	got, err := parseSpotifyPlaylistPage(playlistID, 50, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 81 || got.NextOffset == nil || *got.NextOffset != 55 || len(got.Tracks) != 5 {
		t.Fatalf("pagination: %+v", got)
	}
	if !got.Tracks[0].Playable || got.Tracks[0].Position != 50 || got.Tracks[0].Artists != "Künstler A" || got.Tracks[0].Duration != 200000 {
		t.Fatalf("song: %+v", got.Tracks[0])
	}
	if got.Tracks[1].Playable || got.Tracks[3].Playable || got.Tracks[4].Playable {
		t.Fatal("unavailable/episode/local should not be playable")
	}
	if got.Tracks[2].Position != 52 || got.Tracks[2].Name != "Neues Schema" || !got.Tracks[2].Playable {
		t.Fatalf("new schema: %+v", got.Tracks[2])
	}
	if _, err := parseSpotifyPlaylistPage(playlistID, 0, []byte(`{"total":5}`)); err == nil {
		t.Fatal("missing items accepted")
	}
}
func TestSpotifyPlaylistItemsHTTPAndPlayback(t *testing.T) {
	calls := 0
	s := newMockSpotify(t, func(req *http.Request) (int, string) {
		calls++
		if req.URL.Path == "/v1/playlists/"+playlistID+"/items" {
			if req.URL.Query().Get("offset") != "50" || req.URL.Query().Get("limit") != "50" {
				t.Errorf("wrong pagination: %s", req.URL.String())
			}
			return 200, `{"total":51,"items":[{"item":{"id":"5yEPktRqvIhko5QFF3aBH3","name":"Song","type":"track"}}]}`
		}
		if req.URL.Path == "/v1/me/player/play" {
			data, _ := io.ReadAll(req.Body)
			var body struct {
				ContextURI string `json:"context_uri"`
				Offset     struct {
					Position int `json:"position"`
				} `json:"offset"`
			}
			if json.Unmarshal(data, &body) != nil || body.ContextURI != "spotify:playlist:"+playlistID || body.Offset.Position != 50 {
				t.Errorf("wrong playback payload: %s", string(data))
			}
			return 204, ""
		}
		t.Errorf("unexpected Spotify request: %s", req.URL.String())
		return 404, `{"error":{"message":"missing"}}`
	})
	req := httptest.NewRequest("GET", "http://127.0.0.1:8901/api/spotify/playlist-items?id="+playlistID+"&offset=50", nil)
	w := httptest.NewRecorder()
	s.handleAPI(w, req, "spotify/playlist-items")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"position":50`) {
		t.Fatalf("page: %d %s", w.Code, w.Body.String())
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1:8901/api/spotify/control", strings.NewReader(`{"action":"playlist-track","id":"`+playlistID+`","position":50}`))
	w = httptest.NewRecorder()
	s.handleAPI(w, req, "spotify/control")
	if w.Code != 200 {
		t.Fatalf("play: %d %s", w.Code, w.Body.String())
	}
	if calls != 2 {
		t.Fatalf("request count=%d", calls)
	}
}
func TestSpotifyPlaylistInvalidInputsDontReachAPI(t *testing.T) {
	s := newMockSpotify(t, func(req *http.Request) (int, string) { t.Errorf("unexpected request: %s", req.URL); return 500, "" })
	for _, raw := range []string{"id=../../config", "id=" + playlistID + "&offset=-3", "id=" + playlistID + "&offset=not-number"} {
		req := httptest.NewRequest("GET", "http://127.0.0.1/api/spotify/playlist-items?"+raw, nil)
		w := httptest.NewRecorder()
		s.handleAPI(w, req, "spotify/playlist-items")
		if w.Code != 422 {
			t.Errorf("expected 422 for %s got %d", raw, w.Code)
		}
	}
	for _, raw := range []string{`{"action":"playlist-track","id":"` + playlistID + `","position":-1}`, `{"action":"playlist-track","id":"` + playlistID + `"}`, `{"action":"playlist-track","id":"bad","position":5}`} {
		req := httptest.NewRequest("POST", "http://127.0.0.1/api/spotify/control", strings.NewReader(raw))
		w := httptest.NewRecorder()
		s.handleAPI(w, req, "spotify/control")
		if w.Code != 422 {
			t.Errorf("expected 422 for %s got %d %s", raw, w.Code, w.Body.String())
		}
	}
}
func TestSpotifyPlaylist403IsFriendly(t *testing.T) {
	s := newMockSpotify(t, func(req *http.Request) (int, string) { return 403, `{"error":{"message":"Forbidden"}}` })
	req := httptest.NewRequest("GET", "http://127.0.0.1/api/spotify/playlist-items?id="+playlistID, nil)
	w := httptest.NewRecorder()
	s.handleAPI(w, req, "spotify/playlist-items")
	if w.Code != 403 || !strings.Contains(w.Body.String(), "eigene Playlists") {
		t.Fatalf("unhelpful Spotify 403: %d %s", w.Code, w.Body.String())
	}
}
func TestSpotifyLibraryAssetsServed(t *testing.T) {
	a := &App{prefix: "/s/test/", lastSeen: time.Now()}
	for _, name := range []string{"spotify-library.js", "v423.css", "index.html"} {
		req := httptest.NewRequest("GET", "http://127.0.0.1:8090/s/test/"+name, nil)
		req.RemoteAddr = "127.0.0.1:20000"
		rec := httptest.NewRecorder()
		a.handle(rec, req)
		if rec.Code != 200 {
			t.Fatalf("asset %s missing: %d", name, rec.Code)
		}
		if name == "index.html" && !strings.Contains(rec.Body.String(), `spotify-library.js`) {
			t.Fatal("playlist browser missing from frontend")
		}
	}
}
