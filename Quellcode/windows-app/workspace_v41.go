package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
)

// Workspace layout is deliberately local to the Windows user profile.
// Only a bounded allowlist of dashboard identifiers can be stored.
type WorkspacePrefs struct {
	Preset      string   `json:"preset"`
	Order       []string `json:"order"`
	Hidden      []string `json:"hidden"`
	Wide        []string `json:"wide"`
	ReadAloud   bool     `json:"readAloud"`
	VoiceRate   float64  `json:"voiceRate"`
	VoiceName   string   `json:"voiceName"`
	SpotifyDock string   `json:"spotifyDock"`
	Version     int      `json:"version"`
	CozyStart   bool     `json:"cozyStart"`
	SmartGaming bool     `json:"smartGaming"`
	GlassStyle  string   `json:"glassStyle"`
}

var dashboardCards = []string{"system", "monitor", "games", "network", "news", "clockwidget", "musicwidget", "friendswidget", "sessionwidget"}
var legacyDashboardCards = []string{"system", "monitor", "games", "network", "news"}
var optionalDashboardCards = []string{"clockwidget", "musicwidget", "friendswidget", "sessionwidget"}

func defaultWorkspace() WorkspacePrefs {
	return WorkspacePrefs{Preset: "gaming", Order: append([]string(nil), dashboardCards...), Hidden: append([]string(nil), optionalDashboardCards...), Wide: []string{"news"}, VoiceRate: 1, SpotifyDock: "bar", Version: 2, CozyStart: true, SmartGaming: true, GlassStyle: "frosted"}
}

// The previous version stored five cards. Append widgets without altering personal
// order, hidden cards, Spotify settings or voice preferences.
func migrateWorkspace(x WorkspacePrefs) WorkspacePrefs {
	if x.Version >= 2 {
		return x
	}
	// Recognize only a valid legacy card permutation; corrupted files still fall
	// back to defaultWorkspace during the normal validation step.
	if len(x.Order) != len(legacyDashboardCards) {
		return x
	}
	seen := map[string]bool{}
	for _, id := range x.Order {
		if !slices.Contains(legacyDashboardCards, id) || seen[id] {
			return x
		}
		seen[id] = true
	}
	x.Order = append(x.Order, optionalDashboardCards...)
	x.Hidden = append(x.Hidden, optionalDashboardCards...)
	x.Version = 2
	x.CozyStart = true
	x.SmartGaming = true
	x.GlassStyle = "frosted"
	return x
}
func uniqueCardList(xs []string) ([]string, bool) {
	if len(xs) > len(dashboardCards) {
		return nil, false
	}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if !slices.Contains(dashboardCards, x) || slices.Contains(out, x) {
			return nil, false
		}
		out = append(out, x)
	}
	return out, true
}
func validWorkspace(x WorkspacePrefs) bool {
	if x.Version != 2 || !slices.Contains([]string{"frosted", "crystal", "warm", "midnight"}, x.GlassStyle) || !slices.Contains([]string{"gaming", "cozy", "minimal", "social", "custom"}, x.Preset) {
		return false
	}
	order, ok := uniqueCardList(x.Order)
	if !ok || len(order) != len(dashboardCards) {
		return false
	}
	for _, c := range dashboardCards {
		if !slices.Contains(order, c) {
			return false
		}
	}
	hide, ok := uniqueCardList(x.Hidden)
	if !ok || len(hide) == len(dashboardCards) {
		return false
	}
	_, ok = uniqueCardList(x.Wide)
	if !ok {
		return false
	}
	if x.VoiceRate < .7 || x.VoiceRate > 1.4 || len(x.VoiceName) > 120 || strings.ContainsAny(x.VoiceName, "\r\n\x00") {
		return false
	}
	return slices.Contains([]string{"bar", "floating", "hidden"}, x.SpotifyDock)
}
func (a *App) workspaceAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		cfg := a.workspace
		a.mu.RUnlock()
		send(w, 200, cfg)
	case http.MethodPost:
		var data WorkspacePrefs
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			failure(w, 400, "Ungültige Arbeitsbereich-Einstellungen")
			return
		}
		if !validWorkspace(data) {
			failure(w, 422, "Ungültiges Dashboard-, Sprach- oder Player-Layout")
			return
		}
		a.mu.Lock()
		err := saveJSON(filepath.Join(a.folder, "workspace-v41.json"), data)
		if err == nil {
			a.workspace = data
		}
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Konnte Arbeitsbereich nicht speichern")
			return
		}
		send(w, 200, map[string]any{"saved": true})
	default:
		failure(w, 405, "Methode nicht erlaubt")
	}
}
