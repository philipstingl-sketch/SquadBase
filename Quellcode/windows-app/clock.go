package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
)

// ClockConfig is independent from launcher, appearance and friends settings.
type ClockConfig struct {
	Style      string `json:"style"`
	Seconds    bool   `json:"seconds"`
	TwentyFour bool   `json:"twentyFour"`
	Date       bool   `json:"date"`
	Weekday    bool   `json:"weekday"`
	Session    bool   `json:"session"`
	LocalBadge bool   `json:"localBadge"`
}

func defaultClock() ClockConfig {
	return ClockConfig{Style: "digital", Seconds: true, TwentyFour: true, Date: true, Weekday: true, Session: true, LocalBadge: true}
}
func validClock(c ClockConfig) bool {
	return c.Style == "digital" || c.Style == "flip" || c.Style == "minimal"
}
func (a *App) clockAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		v := a.clock
		a.mu.RUnlock()
		if !validClock(v) {
			v = defaultClock()
		}
		send(w, 200, v)
	case http.MethodPost:
		var v ClockConfig
		if json.NewDecoder(r.Body).Decode(&v) != nil || !validClock(v) {
			failure(w, 422, "Ungültige Uhreinstellung")
			return
		}
		a.mu.Lock()
		err := saveJSON(filepath.Join(a.folder, "clock.json"), v)
		if err == nil {
			a.clock = v
		}
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Uhreinstellungen konnten nicht gespeichert werden")
			return
		}
		send(w, 200, map[string]any{"saved": true, "clock": v})
	default:
		w.Header().Set("Allow", "GET, POST")
		failure(w, 405, "Methode nicht erlaubt")
	}
}
