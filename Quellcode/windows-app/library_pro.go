package main

import (
	"encoding/csv"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type LibraryActivity struct {
	Seconds      int64  `json:"seconds"`
	LastLaunched string `json:"lastLaunched,omitempty"`
	LastSeen     string `json:"lastSeen,omitempty"`
}
type LibraryTracker struct {
	mu       sync.RWMutex
	activity map[string]LibraryActivity
	lastTick time.Time
	folder   string
}

func newLibraryTracker(folder string) *LibraryTracker {
	t := &LibraryTracker{folder: folder, activity: map[string]LibraryActivity{}}
	loadJSON(filepath.Join(folder, "library-activity.json"), &t.activity)
	if t.activity == nil {
		t.activity = map[string]LibraryActivity{}
	}
	return t
}
func (t *LibraryTracker) markLaunched(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	v := t.activity[id]
	v.LastLaunched = time.Now().Format(time.RFC3339)
	t.activity[id] = v
	_ = saveJSON(filepath.Join(t.folder, "library-activity.json"), t.activity)
}

// Parse Windows tasklist /FO CSV output (supports localized header names by
// ignoring the header). User-added EXE and Epic process names are compared only
// with the actual executable name; Steam titles cannot reliably be identified.
func parseTasklistCSV(raw string) map[string]bool {
	out := map[string]bool{}
	r := csv.NewReader(strings.NewReader(raw))
	r.FieldsPerRecord = -1
	lines, e := r.ReadAll()
	if e != nil {
		return out
	}
	for _, row := range lines {
		if len(row) < 2 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(row[0]))
		if strings.HasSuffix(name, ".exe") {
			out[name] = true
		}
	}
	return out
}
func (t *LibraryTracker) tick(games []LibraryGame, running map[string]bool, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delta := now.Sub(t.lastTick)
	t.lastTick = now
	if delta <= 0 || delta > 90*time.Second {
		return
	}
	changed := false
	for _, g := range games {
		if g.Source == "Steam" || g.Path == "" {
			continue
		}
		exe := strings.ToLower(filepath.Base(g.Path))
		if !running[exe] {
			continue
		}
		v := t.activity[g.ID]
		v.Seconds += int64(delta.Seconds())
		v.LastSeen = now.Format(time.RFC3339)
		t.activity[g.ID] = v
		changed = true
	}
	if !changed {
		return
	}
	_ = saveJSON(filepath.Join(t.folder, "library-activity.json"), t.activity)
}
func (t *LibraryTracker) snapshot() map[string]LibraryActivity {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make(map[string]LibraryActivity, len(t.activity))
	for k, v := range t.activity {
		out[k] = v
	}
	return out
}
func (a *App) libraryProAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		failure(w, 405, "Nur GET")
		return
	}
	send(w, 200, map[string]any{"activity": a.libraryTracker.snapshot(), "note": "Erfasste Zeiten gelten nur, während SQUAD BASE läuft und den EXE-Prozess erkennt. Steam-Spiele ohne bekannte EXE werden nicht geschätzt."})
}
func (a *App) trackLibraryLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		a.libraryTracker.tick(a.libraryGames(false), runningProcessNames(), time.Now())
	}
}
