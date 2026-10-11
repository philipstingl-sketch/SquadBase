package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const maxSoundBytes int64 = 8 << 20
const maxSoundSlots = 12

var soundIDPattern = regexp.MustCompile(`^[a-f0-9]{20}$`)

type SoundClip struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}
type SoundBoard struct {
	mu     sync.RWMutex
	folder string
	clips  []SoundClip
}

func newSoundBoard(folder string) *SoundBoard {
	s := &SoundBoard{folder: filepath.Join(folder, "soundboard")}
	_ = os.MkdirAll(s.folder, 0700)
	loadJSON(filepath.Join(s.folder, "clips.json"), &s.clips)
	valid := make([]SoundClip, 0, len(s.clips))
	for _, v := range s.clips {
		if validSoundID(v.ID) && len(v.Name) > 0 && len(v.Name) <= 70 && validSoundType(v.Type) {
			if _, e := os.Stat(s.soundPath(v.ID, v.Type)); e == nil {
				valid = append(valid, v)
			}
		}
	}
	s.clips = valid
	return s
}
func validSoundID(id string) bool  { return soundIDPattern.MatchString(id) }
func validSoundType(t string) bool { return t == "audio/mpeg" || t == "audio/wav" || t == "audio/ogg" }
func (s *SoundBoard) soundPath(id, t string) string {
	ext := map[string]string{"audio/mpeg": ".mp3", "audio/wav": ".wav", "audio/ogg": ".ogg"}[t]
	return filepath.Join(s.folder, id+ext)
}
func clipName(name string) string {
	name = strings.TrimSpace(name)
	r := []rune(name)
	if len(r) > 60 {
		name = string(r[:60])
	}
	if name == "" {
		name = "Neuer Sound"
	}
	return name
}
func (s *SoundBoard) save() error { return saveJSON(filepath.Join(s.folder, "clips.json"), s.clips) }
func (s *SoundBoard) add(name string, body []byte, ext string) (SoundClip, error) {
	if len(body) == 0 || int64(len(body)) > maxSoundBytes {
		return SoundClip{}, errors.New("Audio ist leer oder größer als 8 MB")
	}
	kind := ""
	switch strings.ToLower(ext) {
	case ".mp3":
		if len(body) > 3 && (string(body[:3]) == "ID3" || (body[0] == 0xff && body[1]&0xe0 == 0xe0)) {
			kind = "audio/mpeg"
		}
	case ".wav":
		if len(body) > 12 && string(body[:4]) == "RIFF" && string(body[8:12]) == "WAVE" {
			kind = "audio/wav"
		}
	case ".ogg":
		if len(body) > 4 && string(body[:4]) == "OggS" {
			kind = "audio/ogg"
		}
	}
	if kind == "" {
		return SoundClip{}, errors.New("Nur gültige MP3-, WAV- oder OGG-Dateien sind erlaubt")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.clips) >= maxSoundSlots {
		return SoundClip{}, errors.New("Maximal 12 Sounds. Lösche zuerst einen Sound.")
	}
	b := make([]byte, 10)
	if _, e := rand.Read(b); e != nil {
		return SoundClip{}, e
	}
	id := hex.EncodeToString(b)
	clip := SoundClip{ID: id, Name: clipName(name), Type: kind, Size: int64(len(body))}
	if e := os.WriteFile(s.soundPath(id, kind), body, 0600); e != nil {
		return SoundClip{}, e
	}
	s.clips = append(s.clips, clip)
	if e := s.save(); e != nil {
		s.clips = s.clips[:len(s.clips)-1]
		_ = os.Remove(s.soundPath(id, kind))
		return SoundClip{}, e
	}
	return clip, nil
}
func (s *SoundBoard) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.clips {
		if c.ID == id {
			old := append([]SoundClip(nil), s.clips...)
			s.clips = append(s.clips[:i], s.clips[i+1:]...)
			if e := s.save(); e != nil {
				s.clips = old
				return e
			}
			_ = os.Remove(s.soundPath(id, c.Type))
			return nil
		}
	}
	return errors.New("Sound nicht gefunden")
}
func (s *SoundBoard) all() []SoundClip {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SoundClip{}, s.clips...)
}
func (s *SoundBoard) serve(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		failure(w, 405, "Nur GET")
		return
	}
	s.mu.RLock()
	var clip *SoundClip
	for _, c := range s.clips {
		if c.ID == id {
			cc := c
			clip = &cc
			break
		}
	}
	s.mu.RUnlock()
	if clip == nil {
		http.NotFound(w, r)
		return
	}
	file, e := os.Open(s.soundPath(id, clip.Type))
	if e != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	st, e := file.Stat()
	if e != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", clip.Type)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=60")
	http.ServeContent(w, r, clip.Name, st.ModTime(), file)
}
func (s *SoundBoard) api(w http.ResponseWriter, r *http.Request, name string) {
	if strings.HasPrefix(name, "api/soundboard/file/") {
		s.serve(w, r, strings.TrimPrefix(name, "api/soundboard/file/"))
		return
	}
	if name == "api/soundboard" {
		if r.Method != http.MethodGet {
			failure(w, 405, "Nur GET")
			return
		}
		send(w, 200, map[string]any{"clips": s.all(), "maxClips": maxSoundSlots, "maxBytes": maxSoundBytes})
		return
	}
	if name == "api/soundboard/upload" {
		if r.Method != http.MethodPost {
			failure(w, 405, "Nur POST")
			return
		}
		// Binary audio is uploaded only through this local endpoint, not the remote.
		r.Body = http.MaxBytesReader(w, r.Body, maxSoundBytes+1)
		b, e := io.ReadAll(r.Body)
		if e != nil {
			failure(w, 413, "Sound zu groß (maximal 8 MB)")
			return
		}
		clip, e := s.add(r.URL.Query().Get("name"), b, strings.ToLower(filepath.Ext(r.URL.Query().Get("filename"))))
		if e != nil {
			failure(w, 422, e.Error())
			return
		}
		send(w, 201, map[string]any{"clip": clip})
		return
	}
	if name == "api/soundboard/delete" {
		if r.Method != http.MethodPost {
			failure(w, 405, "Nur POST")
			return
		}
		var q struct {
			ID string `json:"id"`
		}
		if json.NewDecoder(r.Body).Decode(&q) != nil || !validSoundID(q.ID) {
			failure(w, 400, "Ungültige ID")
			return
		}
		if e := s.remove(q.ID); e != nil {
			failure(w, 404, e.Error())
			return
		}
		send(w, 200, map[string]string{"status": "deleted"})
		return
	}
	failure(w, 404, fmt.Sprintf("Soundboard-Route %q existiert nicht", name))
}
