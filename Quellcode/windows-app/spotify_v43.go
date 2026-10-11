package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Requests are routed through the existing PKCE session and Spotify's official
// Web API. No songs are downloaded or streamed by SQUAD BASE.
var spotifyTrackURI = regexp.MustCompile(`^spotify:track:[A-Za-z0-9]{22}$`)

func spotifyPlaybackAction(action, uri string, enabled bool, repeat string) (method, path string, payload []byte, err error) {
	switch action {
	case "shuffle":
		return "PUT", fmt.Sprintf("/me/player/shuffle?state=%t", enabled), nil, nil
	case "repeat":
		if repeat != "off" && repeat != "context" && repeat != "track" {
			return "", "", nil, fmt.Errorf("Ungültiger Wiederholungsmodus")
		}
		return "PUT", "/me/player/repeat?state=" + url.QueryEscape(repeat), nil, nil
	case "enqueue":
		if !spotifyTrackURI.MatchString(uri) {
			return "", "", nil, fmt.Errorf("Nur gültige Spotify-Titel können hinzugefügt werden")
		}
		return "POST", "/me/player/queue?uri=" + url.QueryEscape(uri), nil, nil
	default:
		return "", "", nil, fmt.Errorf("Unbekannte Wiedergabe-Aktion")
	}
}

func (s *SpotifyEngine) musicCenterAPI(w http.ResponseWriter, r *http.Request, name string) {
	switch name {
	case "spotify/queue":
		if r.Method != http.MethodGet {
			failure(w, 405, "Nur GET erlaubt")
			return
		}
		data, _, err := s.spotifyRequest(r.Context(), "GET", "/me/player/queue", nil)
		if err != nil {
			failure(w, 502, "Warteschlange nicht verfügbar: "+err.Error())
			return
		}
		var raw struct {
			CurrentlyPlaying json.RawMessage   `json:"currently_playing"`
			Queue            []json.RawMessage `json:"queue"`
		}
		if json.Unmarshal(data, &raw) != nil {
			failure(w, 502, "Ungültige Spotify-Warteschlange")
			return
		}
		items := make([]map[string]string, 0, min(50, len(raw.Queue)))
		for _, entry := range raw.Queue {
			if len(items) >= 50 {
				break
			}
			var t struct {
				Name    string `json:"name"`
				URI     string `json:"uri"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			}
			if json.Unmarshal(entry, &t) != nil || !spotifyTrackURI.MatchString(t.URI) {
				continue
			}
			names := make([]string, 0, len(t.Artists))
			for _, a := range t.Artists {
				names = append(names, a.Name)
			}
			items = append(items, map[string]string{"title": t.Name, "artists": strings.Join(names, ", "), "uri": t.URI})
		}
		send(w, 200, map[string]any{"items": items, "note": "Spotify kann die Warteschlangen-API je nach App-Zugang einschränken."})
	case "spotify/playback-options":
		if r.Method != http.MethodPost {
			failure(w, 405, "Nur POST erlaubt")
			return
		}
		var input struct {
			Action  string `json:"action"`
			URI     string `json:"uri"`
			Enabled bool   `json:"enabled"`
			Repeat  string `json:"repeat"`
		}
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			failure(w, 400, "Ungültige Eingabe")
			return
		}
		method, path, payload, err := spotifyPlaybackAction(input.Action, input.URI, input.Enabled, input.Repeat)
		if err != nil {
			failure(w, 422, err.Error())
			return
		}
		_, _, err = s.spotifyRequest(r.Context(), method, path, payload)
		if err != nil {
			failure(w, 502, "Spotify konnte den Befehl nicht ausführen: "+err.Error())
			return
		}
		send(w, 200, map[string]string{"message": "An Spotify übergeben"})
	default:
		failure(w, 404, "Unbekannte Spotify-Funktion")
	}
}
