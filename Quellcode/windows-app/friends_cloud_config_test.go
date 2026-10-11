package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestFriendsCloudURLValidation(t *testing.T) {
	for _, c := range []struct {
		url  string
		good bool
	}{
		{"https://abc.workers.dev", true}, {"http://127.0.0.1:8787", true}, {"http://localhost:8787", true}, {"", true},
		{"http://evil.example", false}, {"https://evil.example/path", false}, {"https://evil.example?token=bad", false},
		{"javascript:alert(1)", false}, {"https://user:pass@example.com", false},
	} {
		if validFriendsURL(c.url) != c.good {
			t.Errorf("URL %q unexpected validation", c.url)
		}
	}
}
func TestFriendsCloudConfigPersistence(t *testing.T) {
	a := &App{folder: t.TempDir()}
	body := `{"url":"https://example.workers.dev"}`
	w := httptest.NewRecorder()
	a.friendsCloudConfigAPI(w, httptest.NewRequest(http.MethodPost, "/api/friends-cloud-config", strings.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("POST code %d: %s", w.Code, w.Body.String())
	}
	b := &App{folder: a.folder}
	w = httptest.NewRecorder()
	b.friendsCloudConfigAPI(w, httptest.NewRequest(http.MethodGet, "/api/friends-cloud-config", nil))
	if !strings.Contains(w.Body.String(), "https://example.workers.dev") {
		t.Fatalf("saved URL missing: %s", w.Body.String())
	}
	if _, err := filepath.Abs(b.folder); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	b.friendsCloudConfigAPI(w, httptest.NewRequest(http.MethodPost, "/api/friends-cloud-config", strings.NewReader(`{"url":"http://evil.example"}`)))
	if w.Code != 400 {
		t.Fatal("HTTP internet URL allowed")
	}
}
