package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// The autoclicker only sends ordinary mouse clicks via Windows SendInput.
// It never starts automatically and has hard rate limits and an emergency stop.
type ClickerSettings struct {
	IntervalMs  int    `json:"intervalMs"`
	MouseButton string `json:"mouseButton"`
	Hotkey      string `json:"hotkey"`
}
type ClickerStatus struct {
	ClickerSettings
	Active    bool   `json:"active"`
	Clicks    uint64 `json:"clicks"`
	Supported bool   `json:"supported"`
}
type AutoClicker struct {
	mu        sync.Mutex
	folder    string
	settings  ClickerSettings
	active    bool
	clicks    uint64
	lastDown  bool
	nextClick time.Time
	done      chan struct{}
	once      sync.Once
}

var allowedClickerHotkeys = map[string]bool{"F6": true, "F7": true, "F8": true, "F9": true, "F10": true, "F11": true, "F12": true}

func validClickerHotkey(key string) bool {
	if allowedClickerHotkeys[key] {
		return true
	}
	return len(key) == 10 && strings.HasPrefix(key, "CTRL+ALT+") && key[9] >= 'A' && key[9] <= 'Z'
}
func validateClickerSettings(s ClickerSettings) error {
	if s.IntervalMs < 85 || s.IntervalMs > 5000 {
		return errors.New("Intervall zwischen 85 und 5000 ms wählen")
	}
	if s.MouseButton != "left" && s.MouseButton != "right" {
		return errors.New("Nur linke oder rechte Maustaste")
	}
	if !validClickerHotkey(s.Hotkey) {
		return errors.New("Hotkey: F6–F12 oder Strg+Alt+Buchstabe")
	}
	return nil
}
func newAutoClicker(folder string) *AutoClicker {
	c := &AutoClicker{folder: folder, settings: ClickerSettings{IntervalMs: 200, MouseButton: "left", Hotkey: "F8"}, done: make(chan struct{})}
	var stored ClickerSettings
	loadJSON(filepath.Join(folder, "autoclicker.json"), &stored)
	if validateClickerSettings(stored) == nil {
		c.settings = stored
	}
	go c.loop()
	return c
}
func (c *AutoClicker) shutdown() {
	c.once.Do(func() { close(c.done) })
	c.mu.Lock()
	c.active = false
	c.mu.Unlock()
}
func (c *AutoClicker) snapshot() ClickerStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return ClickerStatus{ClickerSettings: c.settings, Active: c.active, Clicks: c.clicks, Supported: clickerNativeAvailable()}
}
func (c *AutoClicker) action(act string, settings ClickerSettings) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch act {
	case "configure":
		if err := validateClickerSettings(settings); err != nil {
			return err
		}
		if err := saveJSON(filepath.Join(c.folder, "autoclicker.json"), settings); err != nil {
			return errors.New("Einstellungen konnten nicht gespeichert werden")
		}
		c.settings = settings
		c.active = false
		c.lastDown = false
	case "start":
		if !clickerNativeAvailable() {
			return errors.New("Autoclicker benötigt Windows")
		}
		c.active = true
		c.nextClick = time.Now()
		c.clicks = 0
	case "stop":
		c.active = false
	default:
		return errors.New("Unbekannte Aktion")
	}
	return nil
}
func (c *AutoClicker) loop() {
	ticker := time.NewTicker(22 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			key := c.settings.Hotkey
			down := clickerHotkeyPressed(key)
			emergency := clickerEmergencyStop()
			if emergency {
				c.active = false
			}
			if down && !c.lastDown && !emergency {
				c.active = !c.active
				c.nextClick = time.Now()
				if c.active {
					c.clicks = 0
				}
			}
			c.lastDown = down
			doClick := c.active && !emergency && !time.Now().Before(c.nextClick)
			button := c.settings.MouseButton
			if doClick {
				c.nextClick = time.Now().Add(time.Duration(c.settings.IntervalMs) * time.Millisecond)
			}
			c.mu.Unlock()
			if doClick && clickerEmit(button) {
				c.mu.Lock()
				c.clicks++
				c.mu.Unlock()
			}
		}
	}
}
func (a *App) clickerAPI(w http.ResponseWriter, r *http.Request) {
	if a.clicker == nil {
		failure(w, 503, "Autoclicker nicht geladen")
		return
	}
	if r.Method == http.MethodGet {
		send(w, 200, a.clicker.snapshot())
		return
	}
	if r.Method != http.MethodPost {
		failure(w, 405, "Nur GET/POST")
		return
	}
	var q struct {
		Action      string `json:"action"`
		IntervalMs  int    `json:"intervalMs"`
		MouseButton string `json:"mouseButton"`
		Hotkey      string `json:"hotkey"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		failure(w, 400, "Ungültige Einstellungen")
		return
	}
	if err := a.clicker.action(q.Action, ClickerSettings{q.IntervalMs, q.MouseButton, strings.ToUpper(q.Hotkey)}); err != nil {
		failure(w, 400, err.Error())
		return
	}
	send(w, 200, a.clicker.snapshot())
}
