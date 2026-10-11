package main

import (
	"net/http"
	"runtime"
)

// Only non-sensitive, app-specific diagnostics are exposed. No usernames,
// account IDs, paths, tokens, or messages are returned.
func (a *App) assistantContextAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		failure(w, 405, "Nur GET erlaubt")
		return
	}
	a.mu.RLock()
	out := map[string]any{
		"platform": runtime.GOOS, "version": "4.3.0",
		"cpuPercent": a.hw.CPU, "ramPercent": a.hw.RAM,
		"gpuAvailable": a.hw.GPU != nil, "gpuPercent": a.hw.GPU,
		"overlayEnabled":     a.overlayVisible,
		"spotifyConnected":   a.spotify.status()["connected"],
		"smartGamingEnabled": a.workspace.SmartGaming,
		"libraryGames":       len(a.libraryCache),
		"speedtestRunning":   a.speed.Running,
	}
	a.mu.RUnlock()
	send(w, 200, out)
}
