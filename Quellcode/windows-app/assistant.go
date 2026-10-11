package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// A real local model is optional. All URLs are fixed to loopback, never user supplied.
// No API keys, accounts, prompts, friend messages, or files are sent to an external service.
const ollamaBase = "http://127.0.0.1:11434"

type chatTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func localAI(ctx context.Context, question string) (string, string, error) {
	return localAIChat(ctx, question, nil, "")
}
func localAIChat(ctx context.Context, question string, history []chatTurn, preferred string) (string, string, error) {
	question = strings.TrimSpace(question)
	if question == "" || len(question) > 1200 {
		return "", "", errors.New("Frage muss 1–600 Zeichen lang sein")
	}
	client := &http.Client{Timeout: 32 * time.Second}
	ctx, cancel := context.WithTimeout(ctx, 32*time.Second)
	defer cancel()
	// Find available installed model; do NOT download large models without consent.
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, ollamaBase+"/api/tags", nil)
	if e != nil {
		return "", "", e
	}
	rsp, e := client.Do(req)
	if e != nil {
		return "", "", errors.New("Ollama läuft nicht auf diesem PC")
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return "", "", errors.New("Ollama hat keine Modelle geliefert")
	}
	var models struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if json.NewDecoder(io.LimitReader(rsp.Body, 120000)).Decode(&models) != nil || len(models.Models) == 0 {
		return "", "", errors.New("Noch kein lokales Ollama-Modell installiert")
	}
	model := models.Models[0].Name
	for _, m := range models.Models {
		if strings.HasPrefix(m.Name, "llama3.2") {
			model = m.Name
			break
		}
	}
	if preferred != "" && preferred != "auto" {
		found := false
		for _, m := range models.Models {
			if m.Name == preferred {
				found = true
				model = preferred
				break
			}
		}
		if !found {
			return "", "", errors.New("Das ausgewählte KI-Modell ist nicht installiert")
		}
	}
	messages := []map[string]string{{"role": "system", "content": "Du bist SQUAD, der deutschsprachige Gaming-Assistent in SQUAD BASE. Antworte hilfreich und konkret; erinnere dich an den kurzen Gesprächskontext. Du hast keinen Live-Zugriff auf private Daten, andere Apps, Freunde-Chats oder Windows. Behaupte nie, Aktionen ausgeführt zu haben. Erkläre ehrlich, was ohne verbundene Dienste nicht geht. Cube Rush hat Übungsmodus, Schwerkraftwechsel, Kristall-Rettungen und unterschiedliche Spielmodi. Bildgenerierung funktioniert über eine vom Nutzer lokal installierte Stable Diffusion WebUI mit API; ohne Bildmodell erfinde keine erzeugten Bilder. Keine privaten Kontodaten anfordern."}}
	for _, turn := range history {
		role := turn.Role
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(turn.Content)
		if len([]rune(content)) > 650 {
			continue
		}
		messages = append(messages, map[string]string{"role": role, "content": content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": question})
	data, _ := json.Marshal(map[string]any{
		"model":      model,
		"stream":     false,
		"keep_alive": "5m",
		"options":    map[string]any{"temperature": 0.45, "num_predict": 240},
		"messages":   messages,
	})
	req, e = http.NewRequestWithContext(ctx, http.MethodPost, ollamaBase+"/api/chat", bytes.NewReader(data))
	if e != nil {
		return "", "", e
	}
	req.Header.Set("Content-Type", "application/json")
	rsp2, e := client.Do(req)
	if e != nil {
		return "", "", errors.New("Antwort des lokalen Modells nicht verfügbar")
	}
	defer rsp2.Body.Close()
	if rsp2.StatusCode != 200 {
		return "", "", errors.New("Lokales Modell konnte nicht antworten")
	}
	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if json.NewDecoder(io.LimitReader(rsp2.Body, 100000)).Decode(&result) != nil || strings.TrimSpace(result.Message.Content) == "" {
		return "", "", errors.New("Lokales Modell hat keine Antwort geliefert")
	}
	answer := strings.TrimSpace(result.Message.Content)
	if len(answer) > 4500 {
		answer = string([]rune(answer)[:min(len([]rune(answer)), 1800)])
	}
	return answer, model, nil
}

// List installed local chat models. Nothing is downloaded automatically.
func availableLocalModels(ctx context.Context) ([]string, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, ollamaBase+"/api/tags", nil)
	if e != nil {
		return nil, e
	}
	client := &http.Client{Timeout: 4 * time.Second}
	rsp, e := client.Do(req)
	if e != nil {
		return nil, errors.New("Ollama ist nicht gestartet")
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return nil, errors.New("Ollama antwortet nicht")
	}
	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if json.NewDecoder(io.LimitReader(rsp.Body, 120000)).Decode(&out) != nil {
		return nil, errors.New("Ungültige Ollama-Modellliste")
	}
	result := make([]string, 0, min(25, len(out.Models)))
	for _, m := range out.Models {
		if len(m.Name) > 0 && len(m.Name) < 100 && len(result) < 25 {
			result = append(result, m.Name)
		}
	}
	return result, nil
}
