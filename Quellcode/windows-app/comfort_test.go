package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestValidComfortModes(t *testing.T) {
	for _, mode := range []string{"dark", "light", "system"} {
		if !validComfort(ComfortConfig{ThemeMode: mode}) {
			t.Fatalf("expected mode %q to be valid", mode)
		}
	}
	if validComfort(ComfortConfig{ThemeMode: "sepia"}) {
		t.Fatal("unknown theme mode must be rejected")
	}
}

func TestComfortAPISavesThemeLocally(t *testing.T) {
	dir := t.TempDir()
	a := testAppWithSettings(dir)
	payload := []byte(`{"action":"configure","settings":{"startWithWindows":false,"minimizeToTray":false,"themeMode":"light"}}`)
	response := httptest.NewRecorder()
	a.comfortAPI(response, httptest.NewRequest(http.MethodPost, "/api/comfort", bytes.NewReader(payload)))
	if response.Code != http.StatusOK {
		t.Fatalf("configure returned %d: %s", response.Code, response.Body.String())
	}
	if a.comfort.ThemeMode != "light" {
		t.Fatalf("theme mode was not saved: %#v", a.comfort)
	}
	var disk ComfortConfig
	b, err := os.ReadFile(filepath.Join(dir, "comfort.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &disk); err != nil || disk.ThemeMode != "light" {
		t.Fatalf("disk settings mismatch: %#v, %v", disk, err)
	}
}

func TestComfortAPIRejectsInvalidTheme(t *testing.T) {
	a := testAppWithSettings(t.TempDir())
	payload := []byte(`{"action":"configure","settings":{"startWithWindows":false,"minimizeToTray":false,"themeMode":"invisible"}}`)
	response := httptest.NewRecorder()
	a.comfortAPI(response, httptest.NewRequest(http.MethodPost, "/api/comfort", bytes.NewReader(payload)))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("wanted 422, got %d: %s", response.Code, response.Body.String())
	}
}

func TestUploadedSettingsBackupRestoresAfterValidation(t *testing.T) {
	a := testAppWithSettings(t.TempDir())
	backup := a.settingsSnapshot()
	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatal(err)
	}
	a.appearance.Color = "violet"
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(map[string]any{"action": "restore-upload", "backup": json.RawMessage(data)}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	a.diagnosticsAPI(response, httptest.NewRequest(http.MethodPost, "/api/diagnostics", &payload))
	if response.Code != http.StatusOK {
		t.Fatalf("restore-upload returned %d: %s", response.Code, response.Body.String())
	}
	if a.appearance.Color != "oak" {
		t.Fatalf("backup did not restore appearance: %#v", a.appearance)
	}
	if _, err := os.Stat(filepath.Join(a.folder, "SQUAD_BASE_Vorher_Wiederherstellung.json")); err != nil {
		t.Fatalf("missing rollback backup: %v", err)
	}
}

func TestUploadedSettingsBackupRejectsInvalidData(t *testing.T) {
	a := testAppWithSettings(t.TempDir())
	payload := []byte(`{"action":"restore-upload","backup":{"format":"not-a-squad-backup"}}`)
	response := httptest.NewRecorder()
	a.diagnosticsAPI(response, httptest.NewRequest(http.MethodPost, "/api/diagnostics", bytes.NewReader(payload)))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("wanted 422, got %d: %s", response.Code, response.Body.String())
	}
	if a.appearance.Color != "oak" {
		t.Fatal("settings changed despite invalid backup")
	}
}
