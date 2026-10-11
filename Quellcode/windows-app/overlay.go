package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
)

// Overlay is an optional native, always-on-top Edge app window, not an extension.
// It works best with windowed/borderless games; exclusive fullscreen and anti-cheat
// compatibility cannot be guaranteed.
var overlayOperation sync.Mutex

type OverlayConfig struct {
	Enabled       bool `json:"enabled"`
	Opacity       int  `json:"opacity"`
	Scale         int  `json:"scale"`
	X             int  `json:"x"`
	Y             int  `json:"y"`
	CPU           bool `json:"cpu"`
	GPU           bool `json:"gpu"`
	RAM           bool `json:"ram"`
	GPUTemp       bool `json:"gpuTemp"`
	FPS           bool `json:"fps"`
	Ping          bool `json:"ping"`
	Music         bool `json:"music"`
	Friends       bool `json:"friends"`
	Clock         bool `json:"clock"`
	ClickThrough  bool `json:"clickThrough"`
	Compact       bool `json:"compact"`
	MinimalHeader bool `json:"minimalHeader"`
}

func defaultOverlay() OverlayConfig {
	return OverlayConfig{Enabled: true, Opacity: 88, Scale: 100, X: 18, Y: 18, CPU: true, GPU: true, RAM: true, GPUTemp: true, FPS: false, Ping: true, Music: true, Friends: false, Clock: true}
}
func validOverlay(o OverlayConfig) bool {
	return o.Opacity >= 35 && o.Opacity <= 100 && o.Scale >= 70 && o.Scale <= 150 && o.X >= 0 && o.X <= 8000 && o.Y >= 0 && o.Y <= 8000
}
func (a *App) overlayAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		config := a.overlay
		visible := a.overlayVisible
		a.mu.RUnlock()
		send(w, 200, map[string]any{"config": config, "visible": visible, "supported": overlaySupported()})
	case http.MethodPost:
		var o OverlayConfig
		if json.NewDecoder(r.Body).Decode(&o) != nil || !validOverlay(o) {
			failure(w, 422, "Ungültige Overlay-Einstellungen")
			return
		}
		a.mu.Lock()
		err := saveJSON(filepath.Join(a.folder, "overlay.json"), o)
		if err == nil {
			a.overlay = o
		}
		visible := a.overlayVisible
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Overlay-Einstellungen konnten nicht gespeichert werden")
			return
		}
		if visible {
			applyOverlayWindow(o)
		}
		send(w, 200, map[string]any{"saved": true})
	default:
		failure(w, 405, "Methode nicht erlaubt")
	}
}
func (a *App) toggleOverlay() error {
	overlayOperation.Lock()
	defer overlayOperation.Unlock()
	a.mu.Lock()
	a.overlayVisible = !a.overlayVisible
	visible := a.overlayVisible
	cfg := a.overlay
	url := a.overlayURL
	a.mu.Unlock()
	if !visible {
		hideOverlayWindow()
		return nil
	}
	if err := showOverlayWindow(url, cfg); err != nil {
		a.mu.Lock()
		a.overlayVisible = false
		a.mu.Unlock()
		return err
	}
	return nil
}

// Reposition via dragging the visible overlay header. Disabled while
// click-through is enabled, to avoid intercepting input in games.
func (a *App) overlayPositionAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		failure(w, 405, "Nur POST")
		return
	}
	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.X < 0 || req.Y < 0 || req.X > 8000 || req.Y > 8000 {
		failure(w, 422, "Ungültige Position")
		return
	}
	a.mu.Lock()
	if a.overlay.ClickThrough {
		a.mu.Unlock()
		failure(w, 409, "Ziehfunktion ist im Klick-durchlassen-Modus deaktiviert")
		return
	}
	cfg := a.overlay
	cfg.X = req.X
	cfg.Y = req.Y
	if err := saveJSON(filepath.Join(a.folder, "overlay.json"), cfg); err != nil {
		a.mu.Unlock()
		failure(w, 500, "Position konnte nicht gespeichert werden")
		return
	}
	a.overlay = cfg
	visible := a.overlayVisible
	a.mu.Unlock()
	if visible {
		applyOverlayWindow(cfg)
	}
	send(w, 200, map[string]any{"x": cfg.X, "y": cfg.Y})
}
