package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type StudioConfig struct {
	Blur        int  `json:"blur"`
	Radius      int  `json:"radius"`
	Tint        int  `json:"tint"`
	Outline     int  `json:"outline"`
	CustomImage bool `json:"customImage"`
}

func defaultStudio() StudioConfig { return StudioConfig{Blur: 18, Radius: 18, Tint: 55, Outline: 18} }
func validStudio(x StudioConfig) bool {
	return x.Blur >= 0 && x.Blur <= 40 && x.Radius >= 8 && x.Radius <= 34 && x.Tint >= 15 && x.Tint <= 95 && x.Outline >= 0 && x.Outline <= 75
}
func (a *App) wallpaperPath() string {
	for _, ext := range []string{"png", "jpg", "gif"} {
		p := filepath.Join(a.folder, "custom-wallpaper."+ext)
		if _, e := os.Stat(p); e == nil {
			return p
		}
	}
	return ""
}
func (a *App) studioAPI(w http.ResponseWriter, r *http.Request, name string) {
	if name == "api/studio/wallpaper" {
		if r.Method != http.MethodGet {
			failure(w, 405, "Methode nicht erlaubt")
			return
		}
		p := a.wallpaperPath()
		if p == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".gif": "image/gif"}[filepath.Ext(p)])
		http.ServeFile(w, r, p)
		return
	}
	if r.Method == http.MethodGet {
		a.mu.RLock()
		x := a.studio
		a.mu.RUnlock()
		x.CustomImage = a.wallpaperPath() != ""
		send(w, 200, x)
		return
	}
	if r.Method != http.MethodPost {
		failure(w, 405, "Methode nicht erlaubt")
		return
	}
	var q struct {
		Action string `json:"action"`
		StudioConfig
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		failure(w, 400, "Ungültige Eingabe")
		return
	}
	switch q.Action {
	case "save":
		if !validStudio(q.StudioConfig) {
			failure(w, 422, "Ungültige Glas-Einstellungen")
			return
		}
		a.mu.Lock()
		q.StudioConfig.CustomImage = false
		err := saveJSON(filepath.Join(a.folder, "studio.json"), q.StudioConfig)
		if err == nil {
			a.studio = q.StudioConfig
		}
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Konnte nicht speichern")
			return
		}
		send(w, 200, map[string]any{"saved": true})
		return
	case "choose-wallpaper":
		p, e := chooseImage()
		if e != nil {
			failure(w, 503, e.Error())
			return
		}
		if p == "" {
			send(w, 200, map[string]any{"cancelled": true})
			return
		}
		if e = importWallpaper(a.folder, p); e != nil {
			failure(w, 422, e.Error())
			return
		}
		send(w, 200, map[string]any{"saved": true})
		return
	case "clear-wallpaper":
		for _, ext := range []string{"png", "jpg", "gif"} {
			_ = os.Remove(filepath.Join(a.folder, "custom-wallpaper."+ext))
		}
		send(w, 200, map[string]any{"removed": true})
		return
	default:
		failure(w, 422, "Unbekannte Aktion")
	}
}
func importWallpaper(folder, p string) error {
	f, e := os.Open(p)
	if e != nil {
		return fmt.Errorf("Bild nicht lesbar")
	}
	defer f.Close()
	stat, e := f.Stat()
	if e != nil || stat.Size() > 12<<20 || stat.Size() < 50 {
		return fmt.Errorf("Bitte ein Bild bis 12 MB wählen")
	}
	cfg, format, e := image.DecodeConfig(io.LimitReader(f, 12<<20))
	if e != nil || cfg.Width < 16 || cfg.Height < 16 || cfg.Width > 8192 || cfg.Height > 8192 {
		return fmt.Errorf("Kein unterstütztes Bild (JPG/PNG/GIF, maximal 8192 Pixel)")
	}
	ext := map[string]string{"jpeg": "jpg", "png": "png", "gif": "gif"}[strings.ToLower(format)]
	if ext == "" {
		return fmt.Errorf("Nur JPG, PNG oder GIF erlaubt")
	}
	if _, e = f.Seek(0, 0); e != nil {
		return e
	}
	tmp := filepath.Join(folder, "custom-wallpaper.tmp")
	out, e := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	_, e = io.Copy(out, io.LimitReader(f, 12<<20))
	closeErr := out.Close()
	if e != nil || closeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("Kopieren fehlgeschlagen")
	}
	for _, other := range []string{"png", "jpg", "gif"} {
		_ = os.Remove(filepath.Join(folder, "custom-wallpaper."+other))
	}
	return os.Rename(tmp, filepath.Join(folder, "custom-wallpaper."+ext))
}
