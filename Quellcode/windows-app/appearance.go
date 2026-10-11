package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
)

// Appearance is intentionally separate from the legacy game configuration:
// older saved launcher paths and account preferences remain untouched.
type AppearanceConfig struct {
	Color      string `json:"color"`
	Scene      string `json:"scene"`
	Motion     bool   `json:"motion"`
	Speed      string `json:"speed"`
	Cozy       string `json:"cozy"`
	Brightness int    `json:"brightness"`
	Dim        int    `json:"dim"`
	Vignette   int    `json:"vignette"`
	Density    string `json:"density"`
}

func defaultAppearance() AppearanceConfig {
	return AppearanceConfig{Color: "oak", Scene: "oak", Motion: true, Speed: "slow", Cozy: "cozy", Brightness: 100, Dim: 0, Vignette: 20, Density: "normal"}
}
func validAppearance(v AppearanceConfig) bool {
	switch v.Cozy {
	case "", "classic", "cozy", "extra":
	default:
		return false
	}
	switch v.Color {
	case "oak", "violet", "ocean", "sunset", "rose", "ice":
	default:
		return false
	}
	switch v.Scene {
	case "oak", "aurora", "stars", "rain", "embers", "snow", "clouds", "dusk":
	default:
		return false
	}
	switch v.Speed {
	case "slow", "normal", "fast":
	default:
		return false
	}
	if v.Brightness != 0 && (v.Brightness < 55 || v.Brightness > 140) {
		return false
	}
	if v.Dim < 0 || v.Dim > 65 || v.Vignette < 0 || v.Vignette > 65 {
		return false
	}
	switch v.Density {
	case "", "low", "normal", "high":
	default:
		return false
	}
	return true
}
func (a *App) appearanceAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		value := a.appearance
		a.mu.RUnlock()
		if !validAppearance(value) {
			value = defaultAppearance()
		}
		send(w, http.StatusOK, value)
	case http.MethodPost:
		var value AppearanceConfig
		if json.NewDecoder(r.Body).Decode(&value) != nil || !validAppearance(value) {
			failure(w, http.StatusUnprocessableEntity, "Ungültiges Farbschema oder Hintergrund")
			return
		}
		a.mu.Lock()
		err := saveJSON(filepath.Join(a.folder, "appearance.json"), value)
		if err == nil {
			a.appearance = value
		}
		a.mu.Unlock()
		if err != nil {
			failure(w, http.StatusInternalServerError, "Design konnte nicht gespeichert werden")
			return
		}
		send(w, http.StatusOK, map[string]any{"saved": true, "appearance": value})
	default:
		w.Header().Set("Allow", "GET, POST")
		failure(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
	}
}
