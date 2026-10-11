package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestV43PlaybackOptions(t *testing.T) {
	for _, tc := range []struct {
		action, uri, rep, method, path string
		enabled                        bool
		ok                             bool
	}{
		{"shuffle", "", "", "PUT", "/me/player/shuffle?state=true", true, true},
		{"repeat", "", "track", "PUT", "/me/player/repeat?state=track", false, true},
		{"enqueue", "spotify:track:5yEPktRqvIhko5QFF3aBH3", "", "POST", "/me/player/queue?uri=spotify%3Atrack%3A5yEPktRqvIhko5QFF3aBH3", false, true},
		{"enqueue", "spotify:album:5yEPktRqvIhko5QFF3aBH3", "", "", "", false, false},
		{"repeat", "", "everything", "", "", false, false},
		{"system", "", "", "", "", false, false},
	} {
		m, p, _, err := spotifyPlaybackAction(tc.action, tc.uri, tc.enabled, tc.rep)
		if (err == nil) != tc.ok {
			t.Fatalf("%s expected ok=%t got error %v", tc.action, tc.ok, err)
		}
		if tc.ok && (m != tc.method || p != tc.path) {
			t.Fatalf("%s: %q %q", tc.action, m, p)
		}
	}
}
func TestV43LocalLibraryTracker(t *testing.T) {
	dir := t.TempDir()
	tracker := newLibraryTracker(dir)
	names := parseTasklistCSV("\"noriskclient.exe\",\"4321\",\"Console\"\n\"explorer.exe\",\"4\",\"Console\"\n")
	if !names["noriskclient.exe"] || names["noriskclient.exe\"bad"] {
		t.Fatalf("unexpected processes %v", names)
	}
	games := []LibraryGame{{ID: "custom-one", Name: "Minecraft", Source: "Eigene EXE", Path: `C:\\Games\\noriskclient.exe`}}
	// Windows paths under Linux are not interpreted as native paths. Use an
	// OS-neutral executable basename for this test instead.
	games[0].Path = filepath.Join(dir, "noriskclient.exe")
	tracker.tick(games, names, time.Now())
	tracker.tick(games, names, time.Now().Add(30*time.Second))
	a := tracker.snapshot()["custom-one"]
	if a.Seconds < 29 || a.Seconds > 31 {
		t.Fatalf("tracked seconds %d", a.Seconds)
	}
	tracker.markLaunched("custom-one")
	if newLibraryTracker(dir).snapshot()["custom-one"].LastLaunched == "" {
		t.Fatal("launch not saved")
	}
	// Unknown Steam process must not create an invented playtime.
	tracker.tick([]LibraryGame{{ID: "steam-1", Source: "Steam", Path: games[0].Path}}, names, time.Now().Add(60*time.Second))
	if tracker.snapshot()["steam-1"].Seconds != 0 {
		t.Fatal("Steam guess recorded")
	}
}
func TestV43ThemeAndPrivacy(t *testing.T) {
	a := &App{folder: t.TempDir(), themePro: defaultThemePro(), libraryTracker: newLibraryTracker(t.TempDir()), spotify: newSpotify(t.TempDir())}
	if !validThemePro(a.themePro) || validThemePro(ThemePro{NightFrom: 26, DayFrom: 7, NightGlass: "warm"}) {
		t.Fatal("theme validation")
	}
	w := httptest.NewRecorder()
	a.themeProAPI(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"autoDayNight":true,"nightFrom":21,"dayFrom":8,"nightGlass":"warm","reducedMotion":true}`)))
	if w.Code != 200 {
		t.Fatalf("theme save %d %s", w.Code, w.Body.String())
	}
	var read ThemePro
	loadJSON(filepath.Join(a.folder, "theme-pro.json"), &read)
	if !read.AutoDayNight || read.NightGlass != "warm" || !read.ReducedMotion {
		t.Fatalf("not persisted %+v", read)
	}
	w = httptest.NewRecorder()
	a.assistantContextAPI(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusOK || !json.Valid(w.Body.Bytes()) {
		t.Fatalf("context failed: %s", w.Body.String())
	}
	content := w.Body.String()
	if strings.Contains(content, a.folder) || strings.Contains(content, "AccessToken") || strings.Contains(content, "ClientID") {
		t.Fatal("private field exposed")
	}
}
func TestV43OverlayDragSafety(t *testing.T) {
	dir := t.TempDir()
	a := &App{folder: dir, overlay: defaultOverlay()}
	a.overlay.ClickThrough = true
	w := httptest.NewRecorder()
	a.overlayPositionAPI(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"x":99,"y":33}`)))
	if w.Code != 409 {
		t.Fatalf("click through must block dragging: %d", w.Code)
	}
	a.overlay.ClickThrough = false
	w = httptest.NewRecorder()
	a.overlayPositionAPI(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"x":125,"y":44}`)))
	if w.Code != 200 || a.overlay.X != 125 || a.overlay.Y != 44 {
		t.Fatalf("valid drag: %d %v", w.Code, a.overlay)
	}
	if _, err := os.Stat(filepath.Join(dir, "overlay.json")); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	a.overlayPositionAPI(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"x":999999,"y":-3}`)))
	if w.Code != 422 {
		t.Fatal("invalid drag accepted")
	}
}
func TestV43AssetsEmbedded(t *testing.T) {
	for _, name := range []string{"web/v43.css", "web/v43.js", "web/overlay-drag.js", "web/spotify-library.js"} {
		d, e := assets.ReadFile(name)
		if e != nil || len(d) < 100 {
			t.Fatalf("missing %s: %v", name, e)
		}
	}
}

func TestV43ThemeBackupRestore(t *testing.T) {
	dir := t.TempDir()
	a := testAppWithSettings(dir)
	a.themePro = ThemePro{AutoDayNight: true, NightFrom: 21, DayFrom: 7, NightGlass: "warm", ReducedMotion: true}
	report := httptest.NewRecorder()
	a.diagnosticsAPI(report, httptest.NewRequest("POST", "/api/diagnostics", strings.NewReader(`{"action":"backup"}`)))
	if report.Code != 200 {
		t.Fatalf("backup %d: %s", report.Code, report.Body.String())
	}
	a.themePro = defaultThemePro()
	restored := httptest.NewRecorder()
	a.diagnosticsAPI(restored, httptest.NewRequest("POST", "/api/diagnostics", strings.NewReader(`{"action":"restore-latest"}`)))
	if restored.Code != 200 {
		t.Fatalf("restore %d: %s", restored.Code, restored.Body.String())
	}
	if !a.themePro.AutoDayNight || a.themePro.NightGlass != "warm" || !a.themePro.ReducedMotion {
		t.Fatalf("theme restore failed: %#v", a.themePro)
	}
}
