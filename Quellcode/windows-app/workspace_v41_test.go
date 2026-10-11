package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestWorkspacePersistence(t *testing.T) {
	a := &App{folder: t.TempDir(), workspace: defaultWorkspace()}
	x := defaultWorkspace()
	x.Preset = "custom"
	x.Order = []string{"network", "monitor", "games", "system", "news", "clockwidget", "musicwidget", "friendswidget", "sessionwidget"}
	x.Hidden = []string{"news"}
	x.Wide = []string{"system"}
	x.VoiceRate = 1.2
	x.ReadAloud = true
	x.SpotifyDock = "floating"
	b, _ := json.Marshal(x)
	req := httptest.NewRequest(http.MethodPost, "/api/workspace", bytes.NewReader(b))
	w := httptest.NewRecorder()
	a.workspaceAPI(w, req)
	if w.Code != 200 {
		t.Fatalf("save: %v %s", w.Code, w.Body.String())
	}
	var fromDisk WorkspacePrefs
	loadJSON(filepath.Join(a.folder, "workspace-v41.json"), &fromDisk)
	if fromDisk.Preset != "custom" || fromDisk.VoiceRate != 1.2 || fromDisk.Order[0] != "network" {
		t.Fatalf("lost settings %#v", fromDisk)
	}
	w = httptest.NewRecorder()
	a.workspaceAPI(w, httptest.NewRequest(http.MethodGet, "/api/workspace", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
func TestWorkspaceRejectsUnexpectedCardsAndModes(t *testing.T) {
	a := &App{folder: t.TempDir(), workspace: defaultWorkspace()}
	for _, which := range []string{"evil", "duplicate", "allhidden", "voice", "dock"} {
		x := defaultWorkspace()
		switch which {
		case "evil":
			x.Order[0] = "invalid"
		case "duplicate":
			x.Order[0] = "monitor"
		case "allhidden":
			x.Hidden = dashboardCards
		case "voice":
			x.VoiceRate = 50
		case "dock":
			x.SpotifyDock = "file://"
		}
		b, _ := json.Marshal(x)
		w := httptest.NewRecorder()
		a.workspaceAPI(w, httptest.NewRequest(http.MethodPost, "/api/workspace", bytes.NewReader(b)))
		if w.Code != 422 {
			t.Errorf("%s: expected 422 got %d", which, w.Code)
		}
	}
}
