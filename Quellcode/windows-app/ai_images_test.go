package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTestPng(t *testing.T) string {
	t.Helper()
	b := new(bytes.Buffer)
	pic := image.NewRGBA(image.Rect(0, 0, 128, 128))
	pic.Set(10, 10, color.RGBA{11, 150, 88, 255})
	if err := png.Encode(b, pic); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestImagePromptAndStorage(t *testing.T) {
	pngB64 := makeTestPng(t)
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/txt2img" || r.Method != "POST" {
			t.Errorf("unexpected request %v", r.URL)
			return
		}
		var v map[string]any
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			t.Error(err)
		}
		if !strings.Contains(v["prompt"].(string), "pixel art") {
			t.Error("image style not forwarded")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"images": []string{pngB64}})
	}))
	defer engine.Close()
	old := imageAPIBase
	imageAPIBase = engine.URL
	defer func() { imageAPIBase = old }()
	dir := t.TempDir()
	result, e := generateLocalImage(context.Background(), dir, imageRequest{Prompt: "Ein niedlicher Würfel im Wald", Style: "pixel", Width: 512, Height: 512})
	if e != nil {
		t.Fatalf("image generation: %v", e)
	}
	if !safeImageID.MatchString(result.Name) {
		t.Fatal("unsafe image name")
	}
	f, e := os.ReadFile(filepath.Join(dir, "generated-images", result.Name))
	if e != nil {
		t.Fatal(e)
	}
	_, fmt, e := image.DecodeConfig(bytes.NewReader(f))
	if fmt != "png" || e != nil {
		t.Fatalf("saved image invalid: %v", e)
	}
	a := &App{folder: dir}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/ai-image/file?id="+result.Name, nil)
	a.localImageAPI(w, r, "api/ai-image/file")
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("image retrieval: %d", w.Code)
	}
	w2 := httptest.NewRecorder()
	a.localImageAPI(w2, httptest.NewRequest("GET", "/api/ai-image/file?id=../../config.json", nil), "api/ai-image/file")
	if w2.Code != 400 {
		t.Errorf("traversal not rejected: %d", w2.Code)
	}
	w3 := httptest.NewRecorder()
	a.localImageAPI(w3, httptest.NewRequest("GET", "/api/ai-image/gallery", nil), "api/ai-image/gallery")
	if w3.Code != 200 || !strings.Contains(w3.Body.String(), result.Name) {
		t.Fatal("gallery lacks generated image")
	}
}
func TestImageRejectsInvalidInput(t *testing.T) {
	for _, q := range []imageRequest{
		{Prompt: "", Style: "pixel", Width: 512, Height: 512},
		{Prompt: "cool", Style: "unknown", Width: 512, Height: 512},
		{Prompt: "cool", Style: "pixel", Width: 8192, Height: 512},
	} {
		if _, err := generateLocalImage(context.Background(), t.TempDir(), q); err == nil {
			t.Errorf("accepted invalid input %+v", q)
		}
	}
}
