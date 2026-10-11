package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"
)

type FriendPrefs struct {
	Messages bool           `json:"messages"`
	Calls    bool           `json:"calls"`
	Sent     bool           `json:"sent"`
	Invites  bool           `json:"invites"`
	Volume   int            `json:"volume"`
	Volumes  map[string]int `json:"volumes"`
}
type FriendPrefStore struct {
	mu     sync.Mutex
	folder string
	prefs  FriendPrefs
}

var friendPrefName = regexp.MustCompile(`^[a-z0-9_]{3,20}$`)

func validFriendPrefs(p FriendPrefs) bool {
	if p.Volume < 0 || p.Volume > 100 || len(p.Volumes) > 200 {
		return false
	}
	for n, v := range p.Volumes {
		if !friendPrefName.MatchString(n) || v < 0 || v > 100 {
			return false
		}
	}
	return true
}
func newFriendPrefStore(folder string) *FriendPrefStore {
	p := &FriendPrefStore{folder: folder, prefs: FriendPrefs{Messages: true, Calls: true, Invites: true, Volume: 55, Volumes: map[string]int{}}}
	loadJSON(filepath.Join(folder, "friend-preferences.json"), &p.prefs)
	if !validFriendPrefs(p.prefs) {
		p.prefs = FriendPrefs{Messages: true, Calls: true, Invites: true, Volume: 55, Volumes: map[string]int{}}
	}
	if p.prefs.Volumes == nil {
		p.prefs.Volumes = map[string]int{}
	}
	return p
}
func (p *FriendPrefStore) api(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if r.Method == http.MethodGet {
		send(w, 200, p.prefs)
		return
	}
	if r.Method != http.MethodPost {
		failure(w, 405, "Nur GET/POST")
		return
	}
	var data FriendPrefs
	if json.NewDecoder(r.Body).Decode(&data) != nil || !validFriendPrefs(data) {
		failure(w, 400, "Ungültige Benachrichtigungseinstellungen")
		return
	}
	if data.Volumes == nil {
		data.Volumes = map[string]int{}
	}
	if err := saveJSON(filepath.Join(p.folder, "friend-preferences.json"), data); err != nil {
		failure(w, 500, "Einstellungen konnten nicht gespeichert werden")
		return
	}
	p.prefs = data
	send(w, 200, p.prefs)
}
