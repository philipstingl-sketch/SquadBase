package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type AssistantSaved struct {
	Messages []chatTurn `json:"messages"`
}

func (a *App) assistantHistoryAPI(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(a.folder, "assistant-history.json")
	switch r.Method {
	case http.MethodGet:
		var h AssistantSaved
		loadJSON(path, &h)
		if len(h.Messages) > 40 {
			h.Messages = h.Messages[len(h.Messages)-40:]
		}
		send(w, 200, h)
	case http.MethodPost:
		var h AssistantSaved
		if json.NewDecoder(r.Body).Decode(&h) != nil || len(h.Messages) > 40 {
			failure(w, 422, "Maximal 40 Chatnachrichten")
			return
		}
		for _, m := range h.Messages {
			if (m.Role != "assistant" && m.Role != "user") || len([]rune(m.Content)) > 2000 || strings.TrimSpace(m.Content) == "" {
				failure(w, 422, "Ungültiger Chatverlauf")
				return
			}
		}
		if err := saveJSON(path, h); err != nil {
			failure(w, 500, "Chat nicht gespeichert")
			return
		}
		send(w, 200, map[string]any{"saved": true})
	case http.MethodDelete:
		// Explicit delete is local only and does not affect any friend messages.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			failure(w, 500, "Chat konnte nicht gelöscht werden")
			return
		}
		send(w, 200, map[string]any{"cleared": true})
	default:
		failure(w, 405, "Methode nicht erlaubt")
	}
}
