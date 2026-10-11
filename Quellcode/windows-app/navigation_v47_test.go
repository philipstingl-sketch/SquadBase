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
	for _, id := range []string{"dashboard", "games", "library", "friends", "minigames", "assistant", "spotify", "soundboard", "remote", "updates", "discord", "maintenance", "settings", "network", "stats"} {
		if !strings.Contains(page, `data-tab="`+id+`"`) {
			t.Fatalf("navigation destination %q is missing", id)
		}
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
	for _, group := range []string{
		`key:'games',label:'🎮 Spiele',ids:['games','library','minigames','updates']`,
		`key:'media',label:'♫ Medien',ids:['spotify','soundboard']`,
		`key:'tools',label:'🧰 Extras',ids:['assistant','network','stats','discord','remote']`,
		`key:'system',label:'⚙ System',ids:['maintenance','settings']`,
	} {
		if !strings.Contains(js, group) {
			t.Fatalf("expected compact navigation group %q", group)
		}
	}
	if !strings.Contains(js, `nav.replaceChildren(dashboard,groupNodes[0],friends,...groupNodes.slice(1));`) {
		t.Fatal("navigation should keep only Dashboard and Friends directly visible")
	}
	if squadVersion != "4.7.0" {
		t.Fatalf("version: %s", squadVersion)
	}
}
