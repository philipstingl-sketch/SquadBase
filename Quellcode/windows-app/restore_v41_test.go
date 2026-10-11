package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testAppWithSettings(dir string) *App {
	return &App{folder: dir, cfg: Config{Server: "test.example", GamePaths: map[string]string{}}, appearance: defaultAppearance(), clock: defaultClock(), overlay: defaultOverlay(), studio: defaultStudio(), workspace: defaultWorkspace(), arcade: map[string]json.RawMessage{}, libraryPrefs: LibraryPrefs{Favorites: map[string]bool{}}, comfort: defaultComfort()}
}
func TestRestoreLatestSettings(t *testing.T) {
	a := testAppWithSettings(t.TempDir())
	request := httptest.NewRequest(http.MethodPost, "/api/diagnostics", strings.NewReader(`{"action":"backup"}`))
	record := httptest.NewRecorder()
	a.diagnosticsAPI(record, request)
	if record.Code != 200 {
		t.Fatalf("backup: %v %s", record.Code, record.Body.String())
	}
	raw, _ := os.ReadFile(filepath.Join(a.folder, "SQUAD_BASE_Einstellungen_"+"invalid.json"))
	_ = raw
	a.appearance.Color = "violet"
	a.workspace.SpotifyDock = "hidden"
	record = httptest.NewRecorder()
	a.diagnosticsAPI(record, httptest.NewRequest(http.MethodPost, "/api/diagnostics", strings.NewReader(`{"action":"restore-latest"}`)))
	if record.Code != 200 {
		t.Fatalf("restore: %v %s", record.Code, record.Body.String())
	}
	if a.appearance.Color != "oak" || a.workspace.SpotifyDock != "bar" {
		t.Fatalf("not restored: %#v %#v", a.appearance, a.workspace)
	}
	if _, err := os.Stat(filepath.Join(a.folder, "SQUAD_BASE_Vorher_Wiederherstellung.json")); err != nil {
		t.Fatal(err)
	}
}
func TestRestoreRejectsInvalidBackup(t *testing.T) {
	a := testAppWithSettings(t.TempDir())
	p := filepath.Join(a.folder, "SQUAD_BASE_Einstellungen_20260101_120000.json")
	if e := os.WriteFile(p, []byte(`{"format":"NotSquad","version":"0"}`), 0600); e != nil {
		t.Fatal(e)
	}
	record := httptest.NewRecorder()
	a.restoreLatest(record)
	if record.Code != 422 {
		t.Fatalf("wanted 422 got %d", record.Code)
	}
	if a.appearance.Color != "oak" {
		t.Fatal("changed despite invalid backup")
	}
}
