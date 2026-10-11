package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Library entries are discovered only from local Steam/Epic manifests or explicitly
// selected Windows executables. We never upload a directory listing.
type LibraryGame struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	Path      string `json:"path,omitempty"`
	AppID     string `json:"appId,omitempty"`
	Favorite  bool   `json:"favorite"`
	Installed bool   `json:"installed"`
}
type LibraryPrefs struct {
	Custom    []LibraryGame   `json:"custom"`
	Favorites map[string]bool `json:"favorites"`
}

var gameSafeID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,90}$`)
var steamAppPattern = regexp.MustCompile(`(?i)"appid"\s+"(\d{1,15})"`)
var steamNamePattern = regexp.MustCompile(`(?i)"name"\s+"([^"]{1,180})"`)
var steamPathPattern = regexp.MustCompile(`(?i)"path"\s+"([^"]+)"`)

func manifestString(data []byte, re *regexp.Regexp) string {
	match := re.FindSubmatch(data)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(string(match[1]))
}
func detectSteamManifest(path string) (LibraryGame, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 1<<20 {
		return LibraryGame{}, false
	}
	id := manifestString(data, steamAppPattern)
	name := manifestString(data, steamNamePattern)
	if id == "" || name == "" {
		return LibraryGame{}, false
	}
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return LibraryGame{}, false
	}
	return LibraryGame{ID: "steam-" + id, Name: name, Source: "Steam", AppID: id, Installed: true}, true
}
func steamRoots() []string {
	roots := []string{}
	for _, p := range []string{filepath.Join(os.Getenv("ProgramFiles(x86)"), "Steam"), filepath.Join(os.Getenv("ProgramFiles"), "Steam"), filepath.Join(os.Getenv("LOCALAPPDATA"), "Steam"), filepath.Join(os.Getenv("HOME"), ".steam", "steam")} {
		if p != "" {
			if s, e := os.Stat(p); e == nil && s.IsDir() {
				roots = append(roots, p)
			}
		}
	}
	return roots
}
func findSteamGames() []LibraryGame {
	roots := steamRoots()
	dirs := []string{}
	for _, root := range roots {
		dirs = append(dirs, filepath.Join(root, "steamapps"))
		b, e := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
		if e != nil {
			continue
		}
		for _, m := range steamPathPattern.FindAllSubmatch(b, 35) {
			p := strings.ReplaceAll(string(m[1]), `\\`, `\`)
			if filepath.IsAbs(p) {
				dirs = append(dirs, filepath.Join(p, "steamapps"))
			}
		}
	}
	out := []LibraryGame{}
	seen := map[string]bool{}
	for _, dir := range dirs {
		matches, _ := filepath.Glob(filepath.Join(dir, "appmanifest_*.acf"))
		for _, m := range matches {
			g, ok := detectSteamManifest(m)
			if ok && !seen[g.ID] {
				out = append(out, g)
				seen[g.ID] = true
			}
		}
	}
	return out
}
func findEpicGames() []LibraryGame {
	folder := filepath.Join(os.Getenv("PROGRAMDATA"), "Epic", "EpicGamesLauncher", "Data", "Manifests")
	matches, _ := filepath.Glob(filepath.Join(folder, "*.item"))
	out := []LibraryGame{}
	for _, p := range matches {
		b, e := os.ReadFile(p)
		if e != nil || len(b) > 1<<20 {
			continue
		}
		var x struct {
			DisplayName      string
			InstallLocation  string
			LaunchExecutable string
			AppName          string
		}
		if json.Unmarshal(b, &x) != nil || x.DisplayName == "" || x.InstallLocation == "" || x.LaunchExecutable == "" {
			continue
		}
		base := filepath.Clean(x.InstallLocation)
		if !filepath.IsAbs(base) {
			continue
		}
		rel := filepath.Clean(x.LaunchExecutable)
		if filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") || strings.EqualFold(filepath.Ext(rel), ".exe") == false {
			continue
		}
		exe := filepath.Join(base, rel)
		if _, e := os.Stat(exe); e != nil {
			continue
		}
		sum := sha256.Sum256([]byte(x.AppName + "|" + exe))
		out = append(out, LibraryGame{ID: "epic-" + hex.EncodeToString(sum[:6]), Name: x.DisplayName, Source: "Epic", Path: exe, Installed: true})
	}
	return out
}
func defaultLibraryPrefs() LibraryPrefs {
	return LibraryPrefs{Custom: []LibraryGame{}, Favorites: map[string]bool{}}
}
func (a *App) loadLibrary() {
	a.libraryPrefs = defaultLibraryPrefs()
	loadJSON(filepath.Join(a.folder, "library.json"), &a.libraryPrefs)
	if a.libraryPrefs.Favorites == nil {
		a.libraryPrefs.Favorites = map[string]bool{}
	}
	if len(a.libraryPrefs.Custom) > 200 {
		a.libraryPrefs.Custom = a.libraryPrefs.Custom[:200]
	}
}
func (a *App) libraryGames(refresh bool) []LibraryGame {
	a.mu.RLock()
	cached := append([]LibraryGame(nil), a.libraryCache...)
	scanAt := a.libraryScanAt
	a.mu.RUnlock()
	if !refresh && time.Since(scanAt) < 5*time.Minute {
		return cached
	}
	games := append(findSteamGames(), findEpicGames()...)
	a.mu.RLock()
	custom := append([]LibraryGame(nil), a.libraryPrefs.Custom...)
	fav := map[string]bool{}
	for k, v := range a.libraryPrefs.Favorites {
		fav[k] = v
	}
	a.mu.RUnlock()
	games = append(games, custom...)
	uniq := map[string]bool{}
	result := []LibraryGame{}
	for _, g := range games {
		if !gameSafeID.MatchString(g.ID) || g.Name == "" || uniq[g.ID] {
			continue
		}
		uniq[g.ID] = true
		g.Favorite = fav[g.ID]
		if g.Path != "" {
			if _, e := os.Stat(g.Path); e != nil {
				g.Installed = false
			}
		}
		result = append(result, g)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Favorite != result[j].Favorite {
			return result[i].Favorite
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	a.mu.Lock()
	a.libraryCache = result
	a.libraryScanAt = time.Now()
	a.mu.Unlock()
	return result
}
func (a *App) libraryAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		send(w, 200, map[string]any{"games": a.libraryGames(false), "scannedAt": a.libraryScanAt.Format(time.RFC3339)})
		return
	}
	if r.Method != http.MethodPost {
		failure(w, 405, "Methode nicht erlaubt")
		return
	}
	var q struct {
		Action   string `json:"action"`
		ID       string `json:"id"`
		Favorite bool   `json:"favorite"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		failure(w, 400, "Ungültige Anfrage")
		return
	}
	switch q.Action {
	case "refresh":
		send(w, 200, map[string]any{"games": a.libraryGames(true)})
		return
	case "add":
		path, e := chooseEXE()
		if e != nil {
			failure(w, 503, e.Error())
			return
		}
		if path == "" {
			send(w, 200, map[string]any{"cancelled": true})
			return
		}
		path = filepath.Clean(path)
		if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".exe") {
			failure(w, 422, "Nur absolute EXE-Pfade")
			return
		}
		s, e := os.Stat(path)
		if e != nil || s.IsDir() {
			failure(w, 422, "EXE nicht gefunden")
			return
		}
		sum := sha256.Sum256([]byte(strings.ToLower(path)))
		id := "custom-" + hex.EncodeToString(sum[:9])
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		a.mu.Lock()
		found := false
		for _, g := range a.libraryPrefs.Custom {
			if g.ID == id {
				found = true
			}
		}
		if !found && len(a.libraryPrefs.Custom) < 200 {
			a.libraryPrefs.Custom = append(a.libraryPrefs.Custom, LibraryGame{ID: id, Name: name, Source: "Eigene EXE", Path: path, Installed: true})
		}
		err := saveJSON(filepath.Join(a.folder, "library.json"), a.libraryPrefs)
		a.libraryScanAt = time.Time{}
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Spiel konnte nicht gespeichert werden")
			return
		}
		send(w, 200, map[string]any{"added": !found, "id": id})
		return
	case "favorite", "remove", "launch":
		if !gameSafeID.MatchString(q.ID) {
			failure(w, 422, "Ungültige Spiel-ID")
			return
		}
		match := LibraryGame{}
		found := false
		for _, g := range a.libraryGames(false) {
			if g.ID == q.ID {
				match = g
				found = true
				break
			}
		}
		if !found {
			failure(w, 404, "Spiel nicht gefunden")
			return
		}
		if q.Action == "launch" {
			if !match.Installed {
				failure(w, 422, "Spiel ist nicht mehr installiert")
				return
			}
			var err error
			if match.Source == "Steam" {
				err = shellOpen("steam://rungameid/" + match.AppID)
			} else if match.Path != "" && filepath.IsAbs(match.Path) && strings.EqualFold(filepath.Ext(match.Path), ".exe") {
				err = shellOpen(match.Path)
			} else {
				err = errors.New("Kein gültiger Startpfad")
			}
			if err != nil {
				failure(w, 503, err.Error())
				return
			}
			if a.libraryTracker != nil {
				a.libraryTracker.markLaunched(q.ID)
			}
			send(w, 200, map[string]string{"message": "Start angefordert"})
			return
		}
		a.mu.Lock()
		if q.Action == "favorite" {
			a.libraryPrefs.Favorites[q.ID] = q.Favorite
		} else {
			if match.Source != "Eigene EXE" {
				a.mu.Unlock()
				failure(w, 422, "Nur selbst hinzugefügte Spiele entfernbar")
				return
			}
			filtered := []LibraryGame{}
			for _, g := range a.libraryPrefs.Custom {
				if g.ID != q.ID {
					filtered = append(filtered, g)
				}
			}
			a.libraryPrefs.Custom = filtered
			delete(a.libraryPrefs.Favorites, q.ID)
		}
		err := saveJSON(filepath.Join(a.folder, "library.json"), a.libraryPrefs)
		a.libraryScanAt = time.Time{}
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Bibliothek nicht gespeichert")
			return
		}
		send(w, 200, map[string]any{"saved": true})
		return
	default:
		failure(w, 422, fmt.Sprintf("Unbekannte Aktion: %s", q.Action))
	}
}
