package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
)

func arcadeKey(key string) bool {
	if len(key) > 85 || len(key) < 8 {
		return false
	}
	if !(strings.HasPrefix(key, "squad-arcade-") || strings.HasPrefix(key, "squad-cube-")) {
		return false
	}
	for _, c := range key {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}
func (a *App) arcadeAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		v := make(map[string]json.RawMessage, len(a.arcade))
		for k, x := range a.arcade {
			v[k] = x
		}
		a.mu.RUnlock()
		send(w, 200, v)
	case http.MethodPost:
		var req struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || !arcadeKey(req.Key) || len(req.Value) > 4000 || !json.Valid(req.Value) {
			failure(w, 422, "Ungültige Arcade-Einstellung")
			return
		}
		a.mu.Lock()
		if a.arcade == nil {
			a.arcade = make(map[string]json.RawMessage)
		}
		a.arcade[req.Key] = req.Value
		err := saveJSON(filepath.Join(a.folder, "arcade.json"), a.arcade)
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Spielstand konnte nicht gespeichert werden")
			return
		}
		send(w, 200, map[string]bool{"saved": true})
	default:
		failure(w, 405, "Methode nicht erlaubt")
	}
}
