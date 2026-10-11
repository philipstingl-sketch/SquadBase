package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const defaultFriendsCloudURL = "https://squadbase-friends-cloud.squadbase-gaming.workers.dev"

type FriendsCloudConfig struct {
	URL string `json:"url"`
}

func validFriendsURL(raw string) bool {
	if raw == "" {
		return true
	}
	if len(raw) > 240 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	return u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")
}
func (a *App) friendsCloudConfigAPI(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(a.folder, "friends-cloud.json")
	switch r.Method {
	case http.MethodGet:
		var state FriendsCloudConfig
		loadJSON(path, &state)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			state.URL = defaultFriendsCloudURL
		}
		if !validFriendsURL(state.URL) {
			state.URL = ""
		}
		send(w, 200, state)
	case http.MethodPost:
		var state FriendsCloudConfig
		if json.NewDecoder(r.Body).Decode(&state) != nil {
			failure(w, 400, "Ungültige Daten")
			return
		}
		state.URL = strings.TrimSuffix(strings.TrimSpace(state.URL), "/")
		if !validFriendsURL(state.URL) {
			failure(w, 400, "Nur HTTPS-Adressen oder lokale HTTP-Adressen sind erlaubt")
			return
		}
		if err := saveJSON(path, state); err != nil {
			failure(w, 500, "Cloud-Adresse konnte nicht gespeichert werden")
			return
		}
		send(w, 200, state)
	default:
		failure(w, 405, "Methode nicht erlaubt")
	}
}
