package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Local image generation is opt-in and calls ONLY the user's own local
// Stable Diffusion WebUI API. It NEVER uses a fake placeholder or an API key.
// The URL is not configurable from the UI to avoid arbitrary network requests.
var imageAPIBase = "http://127.0.0.1:7860"

var safeImageID = regexp.MustCompile(`^[a-f0-9]{24}\.png$`)

type imageRequest struct {
	Prompt string `json:"prompt"`
	Style  string `json:"style"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type imageItem struct {
	Name    string `json:"name"`
	Created string `json:"created"`
}

func generateLocalImage(ctx context.Context, root string, q imageRequest) (imageItem, error) {
	prompt := strings.TrimSpace(q.Prompt)
	if len([]rune(prompt)) < 3 || len([]rune(prompt)) > 500 {
		return imageItem{}, errors.New("Bildbeschreibung muss 3 bis 500 Zeichen haben")
	}
	styles := map[string]string{
		"natural":   "high quality, detailed, natural lighting",
		"pixel":     "pixel art, crisp pixels, retro video game sprite art",
		"fantasy":   "fantasy illustration, atmospheric, painterly lighting",
		"anime":     "colorful 2D anime-inspired illustration, clean lineart",
		"wallpaper": "landscape wallpaper, cinematic wide composition, no text",
	}
	suffix, ok := styles[q.Style]
	if !ok {
		return imageItem{}, errors.New("Unbekannter Bildstil")
	}
	if q.Width != 512 && q.Width != 768 {
		return imageItem{}, errors.New("Ungültige Bildbreite")
	}
	if q.Height != 512 && q.Height != 768 {
		return imageItem{}, errors.New("Ungültige Bildhöhe")
	}
	request := map[string]any{
		"prompt":          prompt + ", " + suffix,
		"negative_prompt": "blurry, low resolution, text artifacts, watermark",
		"steps":           20, "width": q.Width, "height": q.Height,
		"batch_size": 1, "n_iter": 1, "cfg_scale": 7,
		"send_images": true, "save_images": false,
	}
	body, _ := json.Marshal(request)
	timeout, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(timeout, http.MethodPost, imageAPIBase+"/sdapi/v1/txt2img", bytes.NewReader(body))
	if err != nil {
		return imageItem{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Minute}
	rsp, err := client.Do(req)
	if err != nil {
		return imageItem{}, errors.New("Lokaler Bildgenerator ist nicht erreichbar. Starte Stable Diffusion WebUI mit --api (Port 7860).")
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return imageItem{}, fmt.Errorf("Bildgenerator antwortet mit HTTP %d; prüfe, ob ein Modell geladen ist", rsp.StatusCode)
	}
	var output struct {
		Images []string `json:"images"`
	}
	if err = json.NewDecoder(io.LimitReader(rsp.Body, 22<<20)).Decode(&output); err != nil || len(output.Images) == 0 {
		return imageItem{}, errors.New("Bildgenerator lieferte kein gültiges Bild")
	}
	encoded := output.Images[0]
	if j := strings.Index(encoded, ","); strings.HasPrefix(encoded, "data:") && j >= 0 {
		encoded = encoded[j+1:]
	}
	if len(encoded) > 20<<20 {
		return imageItem{}, errors.New("Bilddatei ist zu groß")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) > 15<<20 {
		return imageItem{}, errors.New("Ungültige Bilddaten")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" || cfg.Width < 64 || cfg.Height < 64 || cfg.Width > 2048 || cfg.Height > 2048 {
		return imageItem{}, errors.New("Bildgenerator lieferte kein unterstütztes PNG")
	}
	dir := filepath.Join(root, "generated-images")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return imageItem{}, err
	}
	idBytes := make([]byte, 12)
	if _, err = rand.Read(idBytes); err != nil {
		return imageItem{}, err
	}
	name := hex.EncodeToString(idBytes) + ".png"
	if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
		return imageItem{}, err
	}
	return imageItem{Name: name, Created: time.Now().Format(time.RFC3339)}, nil
}

func pruneImages(dir string, max int) {
	entries, _ := os.ReadDir(dir)
	var images []imageItem
	for _, e := range entries {
		if e.IsDir() || !safeImageID.MatchString(e.Name()) {
			continue
		}
		f, err := e.Info()
		if err == nil {
			images = append(images, imageItem{e.Name(), f.ModTime().Format(time.RFC3339Nano)})
		}
	}
	sort.Slice(images, func(i, j int) bool { return images[i].Created > images[j].Created })
	for _, it := range images[min(max, len(images)):] {
		_ = os.Remove(filepath.Join(dir, it.Name))
	}
}

func (a *App) localImageAPI(w http.ResponseWriter, r *http.Request, name string) {
	dir := filepath.Join(a.folder, "generated-images")
	switch name {
	case "api/ai-image/generate":
		if r.Method != http.MethodPost {
			failure(w, 405, "Nur POST")
			return
		}
		var q imageRequest
		if json.NewDecoder(r.Body).Decode(&q) != nil {
			failure(w, 400, "Ungültige Bildanfrage")
			return
		}
		img, err := generateLocalImage(r.Context(), a.folder, q)
		if err != nil {
			failure(w, 503, err.Error())
			return
		}
		send(w, 200, img)
	case "api/ai-image/gallery":
		if r.Method != http.MethodGet {
			failure(w, 405, "Nur GET")
			return
		}
		entries, _ := os.ReadDir(dir)
		out := make([]imageItem, 0)
		for _, e := range entries {
			if e.IsDir() || !safeImageID.MatchString(e.Name()) {
				continue
			}
			info, err := e.Info()
			if err == nil {
				out = append(out, imageItem{e.Name(), info.ModTime().Format(time.RFC3339)})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
		if len(out) > 24 {
			out = out[:24]
		}
		send(w, 200, map[string]any{"images": out})
	case "api/ai-image/file":
		if r.Method != http.MethodGet {
			failure(w, 405, "Nur GET")
			return
		}
		id := r.URL.Query().Get("id")
		if !safeImageID.MatchString(id) {
			failure(w, 400, "Ungültige Bild-ID")
			return
		}
		f, err := os.Open(filepath.Join(dir, id))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", id))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = io.CopyN(w, f, 16<<20)
	default:
		http.NotFound(w, r)
	}
}
