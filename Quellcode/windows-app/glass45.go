package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
)

// Glass45 is stored separately from all previous themes and workspace settings.
// "original" leaves all v4.4 design choices untouched.
type Glass45 struct {
	Preset  string `json:"preset"`
	Blur    int    `json:"blur"`
	Opacity int    `json:"opacity"`
	Radius  int    `json:"radius"`
	Glow    bool   `json:"glow"`
	Ambient bool   `json:"ambient"`
}

func defaultGlass45() Glass45 {
	return Glass45{Preset: "original", Blur: 18, Opacity: 68, Radius: 20, Glow: true, Ambient: true}
}
func validGlass45(g Glass45) bool {
	switch g.Preset {
	case "original", "moonstone", "boreal", "rosewood", "starlight":
	default:
		return false
	}
	return g.Blur >= 0 && g.Blur <= 30 && g.Opacity >= 35 && g.Opacity <= 92 && g.Radius >= 8 && g.Radius <= 32
}
func (a *App) glass45API(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		config := a.glass45
		a.mu.RUnlock()
		send(w, 200, config)
	case http.MethodPost:
		var config Glass45
		if json.NewDecoder(r.Body).Decode(&config) != nil || !validGlass45(config) {
			failure(w, 422, "Ungültiges Glass-4.0-Profil")
			return
		}
		a.mu.Lock()
		err := saveJSON(filepath.Join(a.folder, "glass45.json"), config)
		if err == nil {
			a.glass45 = config
		}
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Glasdesign konnte nicht gespeichert werden")
			return
		}
		send(w, 200, map[string]any{"saved": true, "glass45": config})
	default:
		failure(w, 405, "Nicht unterstützt")
	}
}
