package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV452VersionAndStartupDiagnostics(t *testing.T) {
	if squadVersion != "4.7.0" {
		t.Fatalf("wrong version: %s", squadVersion)
	}
	d := newStartupDiagnostic(t.TempDir())
	d.note("Testphase")
	path := d.path
	d.close()
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), "Testphase") {
		t.Fatalf("diagnostics missing %v", err)
	}
}
func TestV452CloudDefaultAndExplicitSelection(t *testing.T) {
	a := newV4TestApp(t)
	a.folder = t.TempDir()
	req := httptest.NewRequest("GET", "http://127.0.0.1/api/friends-cloud-config", nil)
	resp := httptest.NewRecorder()
	a.friendsCloudConfigAPI(resp, req)
	var got FriendsCloudConfig
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.URL != defaultFriendsCloudURL {
		t.Fatalf("default server: %q", got.URL)
	}
	if err := os.WriteFile(filepath.Join(a.folder, "friends-cloud.json"), []byte(`{"url":""}`), 0600); err != nil {
		t.Fatal(err)
	}
	resp = httptest.NewRecorder()
	a.friendsCloudConfigAPI(resp, req)
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.URL != "" {
		t.Fatalf("intentional empty url overwritten: %q", got.URL)
	}
}
