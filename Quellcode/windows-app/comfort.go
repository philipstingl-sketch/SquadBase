package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ComfortConfig stores convenience settings separately from gameplay settings.
// The file contains preferences only, never authentication data.
type ComfortConfig struct {
	StartWithWindows bool   `json:"startWithWindows"`
	MinimizeToTray   bool   `json:"minimizeToTray"`
	ThemeMode        string `json:"themeMode"`
}

func defaultComfort() ComfortConfig {
	return ComfortConfig{ThemeMode: "dark"}
}

func validComfort(v ComfortConfig) bool {
	switch v.ThemeMode {
	case "dark", "light", "system":
		return true
	default:
		return false
	}
}

func (a *App) comfortAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		settings := a.comfort
		a.mu.RUnlock()
		if !validComfort(settings) {
			settings = defaultComfort()
		}
		send(w, http.StatusOK, map[string]any{
			"settings": settings,
			"windows":  runtime.GOOS == "windows",
			"version":  squadVersion,
		})
	case http.MethodPost:
		var q struct {
			Action   string        `json:"action"`
			Settings ComfortConfig `json:"settings"`
		}
		if json.NewDecoder(r.Body).Decode(&q) != nil {
			failure(w, http.StatusBadRequest, "Ungültige Anfrage")
			return
		}
		switch q.Action {
		case "configure":
			if !validComfort(q.Settings) {
				failure(w, http.StatusUnprocessableEntity, "Bitte ein gültiges Design auswählen")
				return
			}
			if runtime.GOOS != "windows" && (q.Settings.StartWithWindows || q.Settings.MinimizeToTray) {
				failure(w, http.StatusNotImplemented, "Autostart und Infobereich sind nur unter Windows verfügbar")
				return
			}
			a.mu.RLock()
			old := a.comfort
			a.mu.RUnlock()

			startupCommandChanged := old.StartWithWindows != q.Settings.StartWithWindows || (q.Settings.StartWithWindows && old.MinimizeToTray != q.Settings.MinimizeToTray)
			trayChanged := old.MinimizeToTray != q.Settings.MinimizeToTray
			if startupCommandChanged {
				if err := configureWindowsAutostart(q.Settings.StartWithWindows, q.Settings.MinimizeToTray); err != nil {
					failure(w, http.StatusInternalServerError, "Windows-Autostart konnte nicht geändert werden: "+err.Error())
					return
				}
			}
			if err := saveJSON(filepath.Join(a.folder, "comfort.json"), q.Settings); err != nil {
				if startupCommandChanged {
					_ = configureWindowsAutostart(old.StartWithWindows, old.MinimizeToTray)
				}
				failure(w, http.StatusInternalServerError, "Einstellungen konnten nicht gespeichert werden")
				return
			}
			if trayChanged {
				if err := setTrayVisible(q.Settings.MinimizeToTray); err != nil {
					_ = saveJSON(filepath.Join(a.folder, "comfort.json"), old)
					if startupCommandChanged {
						_ = configureWindowsAutostart(old.StartWithWindows, old.MinimizeToTray)
					}
					failure(w, http.StatusServiceUnavailable, "Infobereich konnte nicht eingerichtet werden: "+err.Error())
					return
				}
			}
			a.mu.Lock()
			a.comfort = q.Settings
			a.mu.Unlock()
			send(w, http.StatusOK, map[string]any{"saved": true, "settings": q.Settings, "message": "Einstellungen gespeichert."})
		case "hide":
			a.mu.RLock()
			enabled := a.comfort.MinimizeToTray
			a.mu.RUnlock()
			if !enabled {
				failure(w, http.StatusConflict, "Aktiviere zuerst „Im Infobereich behalten“ und speichere die Einstellung.")
				return
			}
			if err := hideAppWindow(); err != nil {
				failure(w, http.StatusServiceUnavailable, err.Error())
				return
			}
			send(w, http.StatusOK, map[string]any{"hidden": true})
		default:
			failure(w, http.StatusUnprocessableEntity, "Unbekannte Aktion")
		}
	default:
		w.Header().Set("Allow", "GET, POST")
		failure(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
	}
}

func loadComfort(path string) ComfortConfig {
	value := defaultComfort()
	loadJSON(path, &value)
	if !validComfort(value) {
		return defaultComfort()
	}
	// Avoid preserving a startup checkbox after a preferences file was edited
	// manually; the next save reconciles the checkbox with the Windows Run key.
	value.ThemeMode = strings.TrimSpace(value.ThemeMode)
	if _, err := os.Stat(path); err != nil {
		return defaultComfort()
	}
	return value
}
