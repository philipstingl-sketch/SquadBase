package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SettingsBackup never includes credentials, OAuth tokens, private chat, or files.
type SettingsBackup struct {
	Format      string                     `json:"format"`
	Version     string                     `json:"version"`
	Appearance  AppearanceConfig           `json:"appearance"`
	Clock       ClockConfig                `json:"clock"`
	Overlay     OverlayConfig              `json:"overlay"`
	GamePaths   map[string]string          `json:"gamePaths"`
	Server      string                     `json:"server,omitempty"`
	Arcade      map[string]json.RawMessage `json:"arcade"`
	Studio      StudioConfig               `json:"studio"`
	Library     LibraryPrefs               `json:"library"`
	Workspace   *WorkspacePrefs            `json:"workspace,omitempty"`
	ThemePro    *ThemePro                  `json:"themePro,omitempty"`
	Glass45     *Glass45                   `json:"glass45,omitempty"`
	AutoClicker *ClickerSettings           `json:"autoclicker,omitempty"`
	FriendPrefs *FriendPrefs               `json:"friendPrefs,omitempty"`
	Comfort     *ComfortConfig             `json:"comfort,omitempty"`
}

func (a *App) settingsSnapshot() SettingsBackup {
	b := SettingsBackup{Format: "SquadBaseSettingsBackupV1", Version: squadVersion, Appearance: a.appearance, Clock: a.clock, Overlay: a.overlay, GamePaths: a.cfg.GamePaths, Server: a.cfg.Server, Arcade: a.arcade, Studio: a.studio, Library: a.libraryPrefs}
	comfort := a.comfort
	if validComfort(comfort) {
		b.Comfort = &comfort
	}
	w := a.workspace
	b.Workspace = &w
	g := a.glass45
	if validGlass45(g) {
		b.Glass45 = &g
	}
	if a.clicker != nil {
		prefs := a.clicker.snapshot().ClickerSettings
		b.AutoClicker = &prefs
	}
	if a.friendPrefs != nil {
		a.friendPrefs.mu.Lock()
		prefs := a.friendPrefs.prefs
		a.friendPrefs.mu.Unlock()
		b.FriendPrefs = &prefs
	}
	th := a.themePro
	if validThemePro(th) {
		b.ThemePro = &th
	}
	return b
}
func validSettingsBackup(b SettingsBackup) bool {
	if b.Format != "SquadBaseSettingsBackupV1" || !validAppearance(b.Appearance) || !validClock(b.Clock) || !validOverlay(b.Overlay) || !validStudio(b.Studio) {
		return false
	}
	if b.AutoClicker != nil && validateClickerSettings(*b.AutoClicker) != nil {
		return false
	}
	if b.FriendPrefs != nil && !validFriendPrefs(*b.FriendPrefs) {
		return false
	}
	if b.Comfort != nil && !validComfort(*b.Comfort) {
		return false
	}
	if b.Glass45 != nil && !validGlass45(*b.Glass45) {
		return false
	}
	if b.ThemePro != nil && !validThemePro(*b.ThemePro) {
		return false
	}
	if b.Workspace != nil && !validWorkspace(migrateWorkspace(*b.Workspace)) {
		return false
	}
	if len(b.GamePaths) > len(gameIDs) || len(b.Arcade) > 500 {
		return false
	}
	for _, raw := range b.Arcade {
		if len(raw) > 64<<10 {
			return false
		}
	}
	for id, p := range b.GamePaths {
		if !validGame(id) || len(p) > 1024 || strings.ContainsAny(p, "\r\n\x00") || (p != "" && !strings.HasSuffix(strings.ToLower(p), ".exe")) {
			return false
		}
	}
	if len(b.Server) > 200 || strings.ContainsAny(b.Server, "\r\n\x00") {
		return false
	}
	return true
}
func (a *App) restoreLatest(w http.ResponseWriter) {
	files, err := filepath.Glob(filepath.Join(a.folder, "SQUAD_BASE_Einstellungen_*.json"))
	if err != nil || len(files) == 0 {
		failure(w, 404, "Keine gespeicherte Sicherung gefunden. Zuerst Einstellungen sichern.")
		return
	}
	sort.Strings(files)
	path := files[len(files)-1]
	fi, err := os.Stat(path)
	if err != nil || fi.Size() > 2<<20 {
		failure(w, 422, "Sicherungsdatei zu groß oder nicht lesbar")
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		failure(w, 500, "Sicherung konnte nicht gelesen werden")
		return
	}
	a.restoreBackupBytes(w, raw)
}

func (a *App) restoreBackupBytes(w http.ResponseWriter, raw []byte) {
	var b SettingsBackup
	if len(raw) == 0 || len(raw) > 2<<20 || json.Unmarshal(raw, &b) != nil || !validSettingsBackup(b) {
		failure(w, 422, "Ungültige Sicherung: Keine Einstellungen übernommen")
		return
	}
	if b.Workspace != nil {
		migrated := migrateWorkspace(*b.Workspace)
		b.Workspace = &migrated
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// Keep an extra recovery copy before replacing any of the current user settings.
	old := a.settingsSnapshot()
	prev := filepath.Join(a.folder, "SQUAD_BASE_Vorher_Wiederherstellung.json")
	var err error
	if err = saveJSON(prev, old); err != nil {
		failure(w, 500, "Rückfallsicherung konnte nicht erstellt werden")
		return
	}
	toWrite := []struct {
		name string
		data any
	}{
		{"appearance.json", b.Appearance}, {"clock.json", b.Clock}, {"overlay.json", b.Overlay}, {"studio.json", b.Studio}, {"arcade.json", b.Arcade}, {"library.json", b.Library},
	}
	if b.AutoClicker != nil {
		toWrite = append(toWrite, struct {
			name string
			data any
		}{"autoclicker.json", *b.AutoClicker})
	}
	if b.FriendPrefs != nil {
		toWrite = append(toWrite, struct {
			name string
			data any
		}{"friend-preferences.json", *b.FriendPrefs})
	}
	if b.ThemePro != nil {
		toWrite = append(toWrite, struct {
			name string
			data any
		}{"theme-pro.json", *b.ThemePro})
	}
	if b.Glass45 != nil {
		toWrite = append(toWrite, struct {
			name string
			data any
		}{"glass45.json", *b.Glass45})
	}
	if b.Workspace != nil {
		toWrite = append(toWrite, struct {
			name string
			data any
		}{"workspace-v41.json", *b.Workspace})
	}
	if b.Comfort != nil {
		toWrite = append(toWrite, struct {
			name string
			data any
		}{"comfort.json", *b.Comfort})
	}
	for _, item := range toWrite {
		if err = saveJSON(filepath.Join(a.folder, item.name), item.data); err != nil {
			failure(w, 500, "Wiederherstellung unvollständig, vorherige Einstellungen sind als Rückfall gesichert. Bitte nicht neu starten.")
			return
		}
	}
	c := a.cfg
	c.GamePaths = b.GamePaths
	if b.Server != "" {
		c.Server = b.Server
	}
	if err = saveJSON(filepath.Join(a.folder, "config.json"), c); err != nil {
		if b.Comfort != nil {
			_ = saveJSON(filepath.Join(a.folder, "comfort.json"), a.comfort)
		}
		failure(w, 500, "Launcher-Pfade konnten nicht wiederhergestellt werden")
		return
	}
	if b.Comfort != nil {
		oldComfort := a.comfort
		startupCommandChanged := oldComfort.StartWithWindows != b.Comfort.StartWithWindows || (b.Comfort.StartWithWindows && oldComfort.MinimizeToTray != b.Comfort.MinimizeToTray)
		trayChanged := oldComfort.MinimizeToTray != b.Comfort.MinimizeToTray
		if startupCommandChanged {
			if err = configureWindowsAutostart(b.Comfort.StartWithWindows, b.Comfort.MinimizeToTray); err != nil {
				_ = saveJSON(filepath.Join(a.folder, "comfort.json"), oldComfort)
				_ = configureWindowsAutostart(oldComfort.StartWithWindows, oldComfort.MinimizeToTray)
				failure(w, 500, "Windows-Komforteinstellungen konnten nicht angewendet werden")
				return
			}
		}
		if trayChanged {
			if err = setTrayVisible(b.Comfort.MinimizeToTray); err != nil {
				_ = saveJSON(filepath.Join(a.folder, "comfort.json"), oldComfort)
				if startupCommandChanged {
					_ = configureWindowsAutostart(oldComfort.StartWithWindows, oldComfort.MinimizeToTray)
				}
				_ = setTrayVisible(oldComfort.MinimizeToTray)
				failure(w, 500, "Infobereich-Einstellung konnte nicht angewendet werden")
				return
			}
		}
		a.comfort = *b.Comfort
	}
	a.appearance = b.Appearance
	a.clock = b.Clock
	a.overlay = b.Overlay
	a.studio = b.Studio
	a.arcade = b.Arcade
	a.libraryPrefs = b.Library
	if b.Glass45 != nil {
		a.glass45 = *b.Glass45
	}
	if b.ThemePro != nil {
		a.themePro = *b.ThemePro
	}
	a.cfg = c
	if b.AutoClicker != nil && a.clicker != nil {
		a.clicker.mu.Lock()
		a.clicker.settings = *b.AutoClicker
		a.clicker.active = false
		a.clicker.mu.Unlock()
	}
	if b.FriendPrefs != nil && a.friendPrefs != nil {
		a.friendPrefs.mu.Lock()
		a.friendPrefs.prefs = *b.FriendPrefs
		a.friendPrefs.mu.Unlock()
	}
	if b.Workspace != nil {
		a.workspace = *b.Workspace
	}
	send(w, 200, map[string]any{"restored": true, "message": "Letzte Sicherung wiederhergestellt. SQUAD BASE bitte vollständig schließen und neu starten. Spotify- und Freunde-Anmeldungen wurden nicht verändert."})
}
