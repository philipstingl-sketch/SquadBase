package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Spotify uses OAuth Authorization Code + PKCE. No password, client secret,
// or access token is ever returned to the browser UI.
type SpotifyEngine struct {
	mu            sync.Mutex
	refreshMu     sync.Mutex
	folder        string
	client        *http.Client
	clientID      string
	bookmarks     []SpotifyBookmark
	token         SpotifyToken
	authState     string
	verifier      string
	authUntil     time.Time
	redirectURI   string
	callbackError string
}
type SpotifyBookmark struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}
type SpotifyToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expiresAt"`
}
type spotifySettings struct {
	ClientID  string            `json:"clientId"`
	Bookmarks []SpotifyBookmark `json:"bookmarks"`
}

var spotifyIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
var spotifyPlaylistPattern = regexp.MustCompile(`^[A-Za-z0-9]{22}$`)

func newSpotify(folder string) *SpotifyEngine {
	s := &SpotifyEngine{folder: folder, client: &http.Client{Timeout: 14 * time.Second}, bookmarks: []SpotifyBookmark{}}
	var cfg spotifySettings
	loadJSON(filepath.Join(folder, "spotify_settings.json"), &cfg)
	s.clientID = cfg.ClientID
	if cfg.Bookmarks != nil {
		s.bookmarks = cfg.Bookmarks
	}
	loadJSON(filepath.Join(folder, "spotify_session.json"), &s.token)
	return s
}
func (s *SpotifyEngine) saveSettings() error {
	s.mu.Lock()
	cfg := spotifySettings{ClientID: s.clientID, Bookmarks: append([]SpotifyBookmark{}, s.bookmarks...)}
	s.mu.Unlock()
	return saveJSON(filepath.Join(s.folder, "spotify_settings.json"), cfg)
}
func (s *SpotifyEngine) saveToken(t SpotifyToken) error {
	// Personal OAuth tokens are saved in the user's local app-data folder.
	return saveJSON(filepath.Join(s.folder, "spotify_session.json"), t)
}
func randomURLToken(n int) (string, error) {
	v := make([]byte, n)
	if _, e := rand.Read(v); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(v), nil
}
func (s *SpotifyEngine) authURL() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !spotifyIDPattern.MatchString(s.clientID) {
		return "", errors.New("Trage zuerst deine Spotify Client-ID ein (32 Zeichen).")
	}
	if s.redirectURI == "" {
		if s.callbackError != "" {
			return "", errors.New(s.callbackError)
		}
		return "", errors.New("Spotify-Callback ist noch nicht bereit.")
	}
	state, e := randomURLToken(24)
	if e != nil {
		return "", e
	}
	verifier, e := randomURLToken(48)
	if e != nil {
		return "", e
	}
	s.authState = state
	s.verifier = verifier
	s.authUntil = time.Now().Add(10 * time.Minute)
	digest := sha256.Sum256([]byte(verifier))
	q := url.Values{}
	q.Set("client_id", s.clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", s.redirectURI)
	q.Set("state", state)
	q.Set("scope", "playlist-read-private playlist-read-collaborative user-read-playback-state user-modify-playback-state")
	q.Set("code_challenge_method", "S256")
	q.Set("code_challenge", base64.RawURLEncoding.EncodeToString(digest[:]))
	return "https://accounts.spotify.com/authorize?" + q.Encode(), nil
}
func (s *SpotifyEngine) exchangeToken(ctx context.Context, params url.Values) (SpotifyToken, error) {
	req, e := http.NewRequestWithContext(ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(params.Encode()))
	if e != nil {
		return SpotifyToken{}, e
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, e := s.client.Do(req)
	if e != nil {
		return SpotifyToken{}, fmt.Errorf("Spotify nicht erreichbar: %w", e)
	}
	defer res.Body.Close()
	body, e := io.ReadAll(io.LimitReader(res.Body, 32*1024))
	if e != nil {
		return SpotifyToken{}, e
	}
	if res.StatusCode != 200 {
		return SpotifyToken{}, fmt.Errorf("Spotify-Anmeldung HTTP %d: %s", res.StatusCode, safeSpotifyError(body))
	}
	var t struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if e = json.Unmarshal(body, &t); e != nil {
		return SpotifyToken{}, e
	}
	if t.AccessToken == "" {
		return SpotifyToken{}, errors.New("Spotify hat keinen Zugriffstoken zurückgegeben")
	}
	return SpotifyToken{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, ExpiresAt: time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)}, nil
}
func safeSpotifyError(body []byte) string {
	var v struct {
		Error            any    `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &v)
	if v.ErrorDescription != "" {
		return v.ErrorDescription
	}
	switch err := v.Error.(type) {
	case string:
		return err
	case map[string]any:
		if s, ok := err["message"].(string); ok {
			return s
		}
	}
	return "Prüfe Berechtigungen, Spotify Premium und App-Freigabe"
}
func (s *SpotifyEngine) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	if r.Method != "GET" {
		http.Error(w, "Nur GET", 405)
		return
	}
	host := r.Host
	if !strings.HasPrefix(host, "127.0.0.1:") {
		http.Error(w, "Ungültiger Host", 403)
		return
	}
	s.mu.Lock()
	valid := s.authState != "" && r.URL.Query().Get("state") == s.authState && time.Now().Before(s.authUntil)
	v := s.verifier
	cid := s.clientID
	redirect := s.redirectURI
	if valid {
		s.authState = ""
		s.verifier = ""
	}
	s.mu.Unlock()
	if !valid {
		http.Error(w, "Anmeldung abgelaufen oder Sicherheitsprüfung fehlgeschlagen", 403)
		return
	}
	if errText := r.URL.Query().Get("error"); errText != "" {
		fmt.Fprint(w, spotifyCallbackPage("Spotify-Verbindung abgebrochen.", false))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Kein Spotify-Code", 400)
		return
	}
	p := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirect}, "client_id": {cid}, "code_verifier": {v}}
	t, e := s.exchangeToken(r.Context(), p)
	if e != nil {
		fmt.Fprint(w, spotifyCallbackPage("Spotify-Anmeldung fehlgeschlagen. Bitte in Squad Base erneut versuchen.", false))
		return
	}
	s.mu.Lock()
	s.token = t
	s.mu.Unlock()
	if e = s.saveToken(t); e != nil {
		fmt.Fprint(w, spotifyCallbackPage("Verbunden, aber Sitzung konnte nicht gespeichert werden.", false))
		return
	}
	fmt.Fprint(w, spotifyCallbackPage("Spotify ist verbunden. Du kannst zu SQUAD BASE zurückkehren.", true))
}
func spotifyCallbackPage(msg string, ok bool) string {
	color := "#f09292"
	if ok {
		color = "#93dfac"
	}
	return `<!doctype html><html lang="de"><head><meta charset="utf-8"><title>Squad Base · Spotify</title></head><body style="background:#101513;color:#eff6f0;font:18px system-ui;display:grid;place-items:center;min-height:90vh;text-align:center"><div><h1 style="color:` + color + `">SQUAD BASE × Spotify</h1><p>` + msg + `</p><small>Du kannst dieses Fenster jetzt schließen.</small></div></body></html>`
}

// Refreshing is serialized so simultaneous playback and playlist requests do
// not race when an access token expires. OAuth tokens stay on the local Go side.
func (s *SpotifyEngine) refreshToken(ctx context.Context, rejectedToken string) (string, error) {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	s.mu.Lock()
	t, cid := s.token, s.clientID
	s.mu.Unlock()
	// Another request may already have refreshed this token.
	if rejectedToken != "" && t.AccessToken != "" && t.AccessToken != rejectedToken && time.Until(t.ExpiresAt) > 0 {
		return t.AccessToken, nil
	}
	if rejectedToken == "" && t.AccessToken != "" && time.Until(t.ExpiresAt) > 50*time.Second {
		return t.AccessToken, nil
	}
	if t.RefreshToken == "" {
		return "", errors.New("Spotify ist nicht verbunden. Bitte anmelden.")
	}
	params := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {t.RefreshToken}, "client_id": {cid}}
	refreshed, err := s.exchangeToken(ctx, params)
	if err != nil {
		return "", fmt.Errorf("Spotify-Zugang konnte nicht erneuert werden: %w", err)
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = t.RefreshToken
	}
	s.mu.Lock()
	s.token = refreshed
	s.mu.Unlock()
	if err = s.saveToken(refreshed); err != nil {
		return "", fmt.Errorf("Spotify-Sitzung konnte nicht gespeichert werden: %w", err)
	}
	return refreshed.AccessToken, nil
}
func (s *SpotifyEngine) tokenFor(ctx context.Context) (string, error) {
	s.mu.Lock()
	t := s.token
	s.mu.Unlock()
	if t.AccessToken != "" && time.Until(t.ExpiresAt) > 50*time.Second {
		return t.AccessToken, nil
	}
	return s.refreshToken(ctx, "")
}
func (s *SpotifyEngine) spotifyRequest(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	tk, err := s.tokenFor(ctx)
	if err != nil {
		return nil, 0, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, "https://api.spotify.com/v1"+path, bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+tk)
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := s.client.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("Spotify-API nicht erreichbar: %w", err)
		}
		data, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		_ = res.Body.Close()
		if readErr != nil {
			return nil, 0, readErr
		}
		if res.StatusCode == 401 && attempt == 0 {
			// Spotify sometimes revokes a token before its expected expiry.
			tk, err = s.refreshToken(ctx, tk)
			if err != nil {
				return nil, 401, err
			}
			continue
		}
		if res.StatusCode >= 400 {
			switch res.StatusCode {
			case 401:
				return nil, res.StatusCode, errors.New("Spotify-Anmeldung ist abgelaufen. Bitte Verbindung trennen und neu anmelden.")
			case 403:
				return nil, res.StatusCode, errors.New("Spotify meldet 403: Prüfe Premium des Developer-App-Besitzers, ob dein Spotify-Konto in der Benutzerliste freigegeben ist und ob die App-Berechtigungen erteilt wurden.")
			case 404:
				return nil, res.StatusCode, errors.New("Kein aktives Spotify-Wiedergabegerät. Öffne Spotify und starte dort kurz einen Song.")
			case 429:
				return nil, res.StatusCode, errors.New("Spotify-Limit erreicht. Bitte später erneut versuchen.")
			}
			return nil, res.StatusCode, fmt.Errorf("Spotify-API HTTP %d: %s", res.StatusCode, safeSpotifyError(data))
		}
		return data, res.StatusCode, nil
	}
	return nil, 401, errors.New("Spotify-Zugang konnte nicht aktualisiert werden")
}
func (s *SpotifyEngine) status() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	bk := append([]SpotifyBookmark{}, s.bookmarks...)
	return map[string]any{"hasClientId": spotifyIDPattern.MatchString(s.clientID), "connected": s.token.RefreshToken != "" || s.token.AccessToken != "", "redirectRegister": spotifyCallbackURI, "callbackReady": s.redirectURI != "", "callbackError": s.callbackError, "bookmarks": bk,
		"desktopControls": desktopMediaAvailable(), "platform": runtime.GOOS,
		"integration": "Spotify Web API (Premium/Entwicklerfreigabe) oder Windows-Medientasten (ohne Konto-Anbindung)"}
}
func (s *SpotifyEngine) handleAPI(w http.ResponseWriter, r *http.Request, name string) {
	if s == nil {
		failure(w, 503, "Spotify noch nicht eingerichtet")
		return
	}
	switch name {
	case "spotify/queue", "spotify/playback-options":
		s.musicCenterAPI(w, r, name)
		return
	case "spotify/desktop":
		if r.Method == "POST" {
			var q struct {
				Action string `json:"action"`
			}
			if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
				failure(w, 400, "Ungültige Eingabe")
				return
			}
			if q.Action == "open" {
				// First try the installed Spotify protocol handler. A browser
				// fallback is useful if Spotify is not installed.
				if err := shellOpen("spotify:"); err != nil {
					if err2 := shellOpen("https://open.spotify.com/"); err2 != nil {
						failure(w, 500, "Spotify konnte nicht geöffnet werden")
						return
					}
				}
				send(w, 200, map[string]string{"message": "Spotify öffnen angefordert"})
				return
			}
			if err := desktopMediaAction(q.Action); err != nil {
				failure(w, 422, err.Error())
				return
			}
			send(w, 200, map[string]string{"message": "Windows-Medientaste gesendet. Falls eine andere Musik-App aktiv ist, kann diese reagieren."})
			return
		}
	case "spotify/status":
		if r.Method == "GET" {
			send(w, 200, s.status())
			return
		}
	case "spotify/client":
		if r.Method == "POST" {
			var q struct {
				ClientID string `json:"clientId"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil {
				failure(w, 400, "Ungültige Eingabe")
				return
			}
			q.ClientID = strings.TrimSpace(q.ClientID)
			if q.ClientID != "" && !spotifyIDPattern.MatchString(q.ClientID) {
				failure(w, 422, "Client-ID muss 32 hexadezimale Zeichen enthalten")
				return
			}
			s.mu.Lock()
			changed := s.clientID != q.ClientID
			if changed {
				s.token = SpotifyToken{}
			}
			s.clientID = q.ClientID
			s.mu.Unlock()
			if changed {
				_ = os.Remove(filepath.Join(s.folder, "spotify_session.json"))
			}
			if e := s.saveSettings(); e != nil {
				failure(w, 500, "Client-ID konnte nicht gespeichert werden")
				return
			}
			send(w, 200, map[string]string{"message": "Spotify Client-ID gespeichert"})
			return
		}
	case "spotify/start":
		if r.Method == "POST" {
			target, e := s.authURL()
			if e != nil {
				failure(w, 422, e.Error())
				return
			}
			if e = shellOpen(target); e != nil {
				failure(w, 500, "Browser konnte nicht geöffnet werden: "+e.Error())
				return
			}
			send(w, 200, map[string]string{"message": "Spotify-Anmeldung im Browser geöffnet"})
			return
		}
	case "spotify/logout":
		if r.Method == "POST" {
			s.mu.Lock()
			s.token = SpotifyToken{}
			s.mu.Unlock()
			_ = os.Remove(filepath.Join(s.folder, "spotify_session.json"))
			send(w, 200, map[string]string{"message": "Spotify getrennt"})
			return
		}
	case "spotify/bookmarks":
		if r.Method == "POST" {
			var q struct {
				Action string `json:"action"`
				URL    string `json:"url"`
				Name   string `json:"name"`
				ID     string `json:"id"`
				Value  int    `json:"value"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil {
				failure(w, 400, "Ungültige Eingabe")
				return
			}
			if q.Action == "add" {
				id, e := extractSpotifyPlaylistID(strings.TrimSpace(q.URL))
				if e != nil {
					failure(w, 422, e.Error())
					return
				}
				q.Name = strings.TrimSpace(q.Name)
				if q.Name == "" {
					q.Name = "Spotify Playlist"
				}
				if len([]rune(q.Name)) > 70 {
					failure(w, 422, "Name zu lang")
					return
				}
				s.mu.Lock()
				if len(s.bookmarks) >= 50 {
					s.mu.Unlock()
					failure(w, 422, "Maximal 50 Lesezeichen")
					return
				}
				found := false
				for _, b := range s.bookmarks {
					if b.ID == id {
						found = true
						break
					}
				}
				if !found {
					s.bookmarks = append(s.bookmarks, SpotifyBookmark{ID: id, Name: q.Name, URL: "https://open.spotify.com/playlist/" + id})
				}
				s.mu.Unlock()
			} else if q.Action == "remove" {
				s.mu.Lock()
				out := make([]SpotifyBookmark, 0, len(s.bookmarks))
				for _, b := range s.bookmarks {
					if b.ID != q.ID {
						out = append(out, b)
					}
				}
				s.bookmarks = out
				s.mu.Unlock()
			} else {
				failure(w, 422, "Unbekannte Aktion")
				return
			}
			if e := s.saveSettings(); e != nil {
				failure(w, 500, "Konnte Playlist-Lesezeichen nicht speichern")
				return
			}
			send(w, 200, s.status())
			return
		}
	case "spotify/playlists":
		if r.Method == "GET" {
			// Multiple real account playlists, up to 150 items in pages of 50.
			result := []map[string]any{}
			for offset := 0; offset < 150; offset += 50 {
				data, _, e := s.spotifyRequest(r.Context(), "GET", fmt.Sprintf("/me/playlists?limit=50&offset=%d", offset), nil)
				if e != nil {
					failure(w, 502, e.Error())
					return
				}
				var x struct {
					Items []struct {
						ID     string `json:"id"`
						Name   string `json:"name"`
						URI    string `json:"uri"`
						Images []struct {
							URL string `json:"url"`
						} `json:"images"`
						External struct {
							Spotify string `json:"spotify"`
						} `json:"external_urls"`
						Items struct {
							Total int `json:"total"`
						} `json:"items"`
					} `json:"items"`
					Total int `json:"total"`
				}
				if e = json.Unmarshal(data, &x); e != nil {
					failure(w, 502, "Ungültige Spotify-Antwort")
					return
				}
				for _, p := range x.Items {
					if !spotifyPlaylistPattern.MatchString(p.ID) {
						continue
					}
					cover := ""
					if len(p.Images) > 0 && strings.HasPrefix(p.Images[0].URL, "https://") {
						cover = p.Images[0].URL
					}
					result = append(result, map[string]any{"id": p.ID, "name": p.Name, "uri": "spotify:playlist:" + p.ID, "url": "https://open.spotify.com/playlist/" + p.ID, "cover": cover, "total": p.Items.Total})
				}
				if len(x.Items) < 50 || offset+len(x.Items) >= x.Total {
					break
				}
			}
			send(w, 200, map[string]any{"playlists": result})
			return
		}
	case "spotify/playlist-items":
		if r.Method == "GET" {
			id := strings.TrimSpace(r.URL.Query().Get("id"))
			if !spotifyPlaylistPattern.MatchString(id) {
				failure(w, 422, "Ungültige Spotify-Playlist-ID")
				return
			}
			offset := 0
			if raw := r.URL.Query().Get("offset"); raw != "" {
				var parseErr error
				offset, parseErr = strconv.Atoi(raw)
				if parseErr != nil || offset < 0 || offset > 100000 {
					failure(w, 422, "Ungültiger Playlist-Abschnitt")
					return
				}
			}
			page, status, err := s.fetchPlaylistPage(r.Context(), id, offset)
			if err != nil {
				if status == 403 {
					failure(w, 403, "Spotify erlaubt die komplette Songliste nur für eigene Playlists oder solche, bei denen du mitarbeitest. Öffne diese Playlist stattdessen direkt in Spotify.")
					return
				}
				failure(w, 502, err.Error())
				return
			}
			send(w, 200, page)
			return
		}
	case "spotify/now":
		if r.Method == "GET" {
			data, status, e := s.spotifyRequest(r.Context(), "GET", "/me/player", nil)
			if e != nil {
				failure(w, 502, e.Error())
				return
			}
			if status == 204 || len(data) == 0 {
				send(w, 200, map[string]any{"active": false, "message": "Keine aktive Spotify-Wiedergabe"})
				return
			}
			var p struct {
				IsPlaying    bool   `json:"is_playing"`
				ShuffleState bool   `json:"shuffle_state"`
				RepeatState  string `json:"repeat_state"`
				ProgressMS   int    `json:"progress_ms"`
				Device       struct {
					Name          string `json:"name"`
					VolumePercent *int   `json:"volume_percent"`
				} `json:"device"`
				Item struct {
					Name       string `json:"name"`
					DurationMS int    `json:"duration_ms"`
					External   struct {
						Spotify string `json:"spotify"`
					} `json:"external_urls"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
					Album struct {
						Images []struct {
							URL string `json:"url"`
						} `json:"images"`
					} `json:"album"`
				} `json:"item"`
			}
			if json.Unmarshal(data, &p) != nil {
				failure(w, 502, "Ungültige Spotify-Wiedergabedaten")
				return
			}
			artists := make([]string, 0, len(p.Item.Artists))
			for _, v := range p.Item.Artists {
				artists = append(artists, v.Name)
			}
			art := ""
			if len(p.Item.Album.Images) > 0 && strings.HasPrefix(p.Item.Album.Images[0].URL, "https://") {
				art = p.Item.Album.Images[0].URL
			}
			send(w, 200, map[string]any{"active": true, "playing": p.IsPlaying, "title": p.Item.Name, "artists": strings.Join(artists, ", "), "cover": art, "url": p.Item.External.Spotify, "duration": p.Item.DurationMS, "progress": p.ProgressMS, "device": p.Device.Name, "volume": p.Device.VolumePercent, "shuffle": p.ShuffleState, "repeat": p.RepeatState})
			return
		}
	case "spotify/control":
		if r.Method == "POST" {
			var q struct {
				Action   string `json:"action"`
				ID       string `json:"id"`
				Value    int    `json:"value"`
				Position *int   `json:"position"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil {
				failure(w, 400, "Ungültige Eingabe")
				return
			}
			method, path := "", ""
			var payload []byte
			switch q.Action {
			case "play":
				method, path = "PUT", "/me/player/play"
			case "pause":
				method, path = "PUT", "/me/player/pause"
			case "next":
				method, path = "POST", "/me/player/next"
			case "previous":
				method, path = "POST", "/me/player/previous"
			case "volume":
				if q.Value < 0 || q.Value > 100 {
					failure(w, 422, "Lautstärke muss zwischen 0 und 100 liegen")
					return
				}
				method, path = "PUT", fmt.Sprintf("/me/player/volume?volume_percent=%d", q.Value)
			case "seek":
				if q.Value < 0 || q.Value > 24*60*60*1000 {
					failure(w, 422, "Ungültige Position")
					return
				}
				method, path = "PUT", fmt.Sprintf("/me/player/seek?position_ms=%d", q.Value)
			case "playlist", "playlist-track":
				if !spotifyPlaylistPattern.MatchString(q.ID) {
					failure(w, 422, "Ungültige Spotify-Playlist-ID")
					return
				}
				method, path = "PUT", "/me/player/play"
				if q.Action == "playlist-track" {
					if q.Position == nil || *q.Position < 0 || *q.Position > 100000 {
						failure(w, 422, "Ungültige Songposition in der Playlist")
						return
					}
					payload, _ = json.Marshal(map[string]any{
						"context_uri": "spotify:playlist:" + q.ID,
						"offset":      map[string]int{"position": *q.Position},
					})
				} else {
					payload = []byte(`{"context_uri":"spotify:playlist:` + q.ID + `"}`)
				}
			default:
				failure(w, 422, "Unbekannter Spotify-Befehl")
				return
			}
			_, _, e := s.spotifyRequest(r.Context(), method, path, payload)
			if e != nil {
				failure(w, 502, e.Error())
				return
			}
			send(w, 200, map[string]string{"message": "Spotify-Befehl gesendet"})
			return
		}
	case "spotify/open":
		if r.Method == "POST" {
			var q struct {
				ID string `json:"id"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil {
				failure(w, 400, "Ungültige Eingabe")
				return
			}
			link := "https://open.spotify.com/"
			if q.ID != "" {
				if !spotifyPlaylistPattern.MatchString(q.ID) {
					failure(w, 422, "Ungültige Playlist-ID")
					return
				}
				link = "https://open.spotify.com/playlist/" + q.ID
			}
			if e := shellOpen(link); e != nil {
				failure(w, 500, e.Error())
				return
			}
			send(w, 200, map[string]string{"message": "Spotify geöffnet"})
			return
		}
	}
	failure(w, 405, "Nicht unterstützte Spotify-Aktion")
}

// spotifyPlaylistTrack is a normalized entry for the UI. Position is the
// original Spotify playlist index, so clicking a song preserves playlist order.
type spotifyPlaylistTrack struct {
	Position int    `json:"position"`
	ID       string `json:"id"`
	URI      string `json:"uri"`
	Name     string `json:"name"`
	Artists  string `json:"artists"`
	Album    string `json:"album"`
	Cover    string `json:"cover"`
	URL      string `json:"url"`
	Duration int    `json:"duration"`
	Playable bool   `json:"playable"`
}
type spotifyPlaylistPage struct {
	ID         string                 `json:"id"`
	Total      int                    `json:"total"`
	Offset     int                    `json:"offset"`
	NextOffset *int                   `json:"nextOffset"`
	Tracks     []spotifyPlaylistTrack `json:"tracks"`
}

func parseSpotifyPlaylistPage(id string, offset int, raw []byte) (spotifyPlaylistPage, error) {
	var response struct {
		Total int `json:"total"`
		Items []struct {
			Track   json.RawMessage `json:"track"`
			Item    json.RawMessage `json:"item"`
			IsLocal bool            `json:"is_local"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &response); err != nil || response.Items == nil {
		return spotifyPlaylistPage{}, errors.New("Spotify hat eine unvollständige Songliste zurückgegeben")
	}
	out := spotifyPlaylistPage{ID: id, Total: response.Total, Offset: offset, Tracks: make([]spotifyPlaylistTrack, 0, len(response.Items))}
	for i, entry := range response.Items {
		n := spotifyPlaylistTrack{Position: offset + i, Name: "Nicht verfügbar", Artists: "In Spotify öffnen"}
		item := entry.Track
		if len(item) == 0 || string(item) == "null" {
			item = entry.Item // tolerate both Spotify item field layouts
		}
		var track struct {
			ID         string `json:"id"`
			URI        string `json:"uri"`
			Name       string `json:"name"`
			Type       string `json:"type"`
			Duration   int    `json:"duration_ms"`
			IsPlayable *bool  `json:"is_playable"`
			IsLocal    bool   `json:"is_local"`
			Artists    []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Name   string `json:"name"`
				Images []struct {
					URL string `json:"url"`
				} `json:"images"`
			} `json:"album"`
		}
		if len(item) != 0 && string(item) != "null" && json.Unmarshal(item, &track) == nil {
			if track.Name != "" {
				n.Name = track.Name
			}
			if track.Type == "episode" {
				n.Artists = "Podcast-Folge"
			}
			if len(track.Artists) > 0 {
				artists := make([]string, 0, len(track.Artists))
				for _, artist := range track.Artists {
					if artist.Name != "" {
						artists = append(artists, artist.Name)
					}
				}
				n.Artists = strings.Join(artists, ", ")
			}
			n.Album = track.Album.Name
			n.Duration = track.Duration
			if len(track.Album.Images) > 0 && strings.HasPrefix(track.Album.Images[0].URL, "https://i.scdn.co/") {
				n.Cover = track.Album.Images[0].URL
			}
			if spotifyPlaylistPattern.MatchString(track.ID) && track.Type == "track" && !entry.IsLocal && !track.IsLocal && (track.IsPlayable == nil || *track.IsPlayable) {
				n.ID = track.ID
				n.URI = "spotify:track:" + track.ID
				n.URL = "https://open.spotify.com/track/" + track.ID
				n.Playable = true
			} else if spotifyPlaylistPattern.MatchString(track.ID) && track.Type == "episode" {
				n.URL = "https://open.spotify.com/episode/" + track.ID
			}
		}
		out.Tracks = append(out.Tracks, n)
	}
	if offset+len(response.Items) < response.Total && len(response.Items) > 0 {
		next := offset + len(response.Items)
		out.NextOffset = &next
	}
	return out, nil
}

func (s *SpotifyEngine) fetchPlaylistPage(ctx context.Context, id string, offset int) (spotifyPlaylistPage, int, error) {
	if !spotifyPlaylistPattern.MatchString(id) || offset < 0 || offset > 100000 {
		return spotifyPlaylistPage{}, 422, errors.New("Ungültige Spotify-Playlist")
	}
	path := fmt.Sprintf("/playlists/%s/items?limit=50&offset=%d", id, offset)
	data, code, err := s.spotifyRequest(ctx, "GET", path, nil)
	if err != nil {
		return spotifyPlaylistPage{}, code, err
	}
	page, err := parseSpotifyPlaylistPage(id, offset, data)
	return page, code, err
}

func extractSpotifyPlaylistID(raw string) (string, error) {
	if strings.HasPrefix(raw, "spotify:playlist:") {
		id := strings.TrimPrefix(raw, "spotify:playlist:")
		if spotifyPlaylistPattern.MatchString(id) {
			return id, nil
		}
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || (u.Host != "open.spotify.com" && u.Host != "spotify.com") {
		return "", errors.New("Nutze einen echten Spotify-Playlist-Link (https://open.spotify.com/playlist/...)")
	}
	bits := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(bits) != 2 || bits[0] != "playlist" || !spotifyPlaylistPattern.MatchString(bits[1]) {
		return "", errors.New("Kein gültiger Spotify-Playlist-Link")
	}
	return bits[1], nil
}
