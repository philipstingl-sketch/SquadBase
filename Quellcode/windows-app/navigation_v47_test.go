package main

import (
	"strings"
	"testing"
)

func TestV47CleanNavigationAndOverlaySettings(t *testing.T) {
	html, err := assets.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(html)
	if !strings.Contains(page, `data-tab="minigames"`) {
		t.Fatal("Minispiele must remain a top-level tab")
	}
	if strings.Contains(page, `data-tab="overlay"`) {
		t.Fatal("Overlay must not be a top-level tab")
	}
	if strings.Contains(page, `<section class="page" id="overlay">`) {
		t.Fatal("Overlay must no longer be a separate page")
	}
	if !strings.Contains(page, `id="overlay-settings-group"`) || !strings.Contains(page, `id="overlay-settings"`) {
		t.Fatal("overlay settings accordion or controls missing")
	}
	if !strings.Contains(page, `data-tab="assistant"`) || !strings.Contains(page, `data-tab="soundboard"`) {
		t.Fatal("Tools routes must preserve existing AI and Soundboard pages")
	}
	nav, err := assets.ReadFile("web/v42.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(nav)
	if !strings.Contains(js, `label:'🧰 Tools',ids:['assistant','soundboard'`) {
		t.Fatal("AI and Soundboard must be grouped under Tools")
	}
	if squadVersion != "4.7.0" {
		t.Fatalf("version: %s", squadVersion)
	}
}
