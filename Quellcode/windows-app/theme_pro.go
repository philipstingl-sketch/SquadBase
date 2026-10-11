package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
)

type ThemePro struct {
	AutoDayNight  bool   `json:"autoDayNight"`
	NightFrom     int    `json:"nightFrom"`
	DayFrom       int    `json:"dayFrom"`
	NightGlass    string `json:"nightGlass"`
	ReducedMotion bool   `json:"reducedMotion"`
}

func defaultThemePro() ThemePro { return ThemePro{NightFrom: 20, DayFrom: 7, NightGlass: "midnight"} }
func validThemePro(t ThemePro) bool {
	return t.NightFrom >= 0 && t.NightFrom < 24 && t.DayFrom >= 0 && t.DayFrom < 24 && t.NightFrom != t.DayFrom && (t.NightGlass == "midnight" || t.NightGlass == "warm" || t.NightGlass == "crystal" || t.NightGlass == "frosted")
}
func (a *App) themeProAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		v := a.themePro
		a.mu.RUnlock()
		send(w, 200, v)
	case http.MethodPost:
		var t ThemePro
		if json.NewDecoder(r.Body).Decode(&t) != nil || !validThemePro(t) {
			failure(w, 422, "Ungültige Theme-Einstellungen")
			return
		}
		a.mu.Lock()
		err := saveJSON(filepath.Join(a.folder, "theme-pro.json"), t)
		if err == nil {
			a.themePro = t
		}
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Theme konnte nicht gespeichert werden")
			return
		}
		send(w, 200, map[string]bool{"saved": true})
	default:
		failure(w, 405, "Nicht unterstützt")
	}
}
