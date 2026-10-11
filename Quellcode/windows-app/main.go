package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func hasArg(target string) bool {
	for _, arg := range os.Args[1:] {
		if arg == target {
			return true
		}
	}
	return false
}

//go:embed web/*
var assets embed.FS

type Config struct {
	Server    string            `json:"server"`
	GamePaths map[string]string `json:"gamePaths"`
}
type Hardware struct {
	CPU          float64  `json:"cpu"`
	RAM          float64  `json:"ram"`
	UsedRAM      float64  `json:"usedRam"`
	TotalRAM     float64  `json:"totalRam"`
	GPU          *float64 `json:"gpu"`
	GPUTemp      *int     `json:"gpuTemp"`
	GPUVRAMUsed  *int     `json:"gpuVramUsed"`
	GPUVRAMTotal *int     `json:"gpuVramTotal"`
	GPUFan       *int     `json:"gpuFan"`
	DiskFreeGB   *float64 `json:"diskFreeGB"`
	DiskTotalGB  *float64 `json:"diskTotalGB"`
	CPUTemp      *int     `json:"cpuTemp"`
	FPS          *float64 `json:"fps"`
	GPUName      string   `json:"gpuName"`
	CPUName      string   `json:"cpuName"`
	Host         string   `json:"host"`
}
type SpeedResult struct {
	Running  bool     `json:"running"`
	Phase    string   `json:"phase"`
	Ping     *float64 `json:"ping"`
	Download *float64 `json:"download"`
	Upload   *float64 `json:"upload"`
	Error    string   `json:"error"`
	TestedAt string   `json:"testedAt"`
}
type ServerStatus struct {
	Online     bool   `json:"online"`
	Players    int    `json:"players"`
	MaxPlayers int    `json:"maxPlayers"`
	Latency    int64  `json:"latency"`
	Version    string `json:"version"`
	Error      string `json:"error"`
	CheckedAt  string `json:"checkedAt"`
}
type App struct {
	appearance      AppearanceConfig
	clock           ClockConfig
	overlay         OverlayConfig
	overlayVisible  bool
	overlayURL      string
	arcade          map[string]json.RawMessage
	mu              sync.RWMutex
	cfg             Config
	usage           map[string]int64
	hw              Hardware
	speed           SpeedResult
	ping            TargetPing
	server          ServerStatus
	playing         map[string]bool
	folder          string
	prefix          string
	lastSeen        time.Time
	libraryPrefs    LibraryPrefs
	libraryCache    []LibraryGame
	libraryScanAt   time.Time
	maintenance     Maintenance
	comfort         ComfortConfig
	studio          StudioConfig
	workspace       WorkspacePrefs
	libraryTracker  *LibraryTracker
	themePro        ThemePro
	glass45         Glass45
	spotify         *SpotifyEngine
	remote          RemoteController
	soundboard      *SoundBoard
	clicker         *AutoClicker
	friendPrefs     *FriendPrefStore
	news            *NewsEngine
	shutdownRequest chan struct{}
}

var gameIDs = []string{"minecraft", "rocket", "gta", "valorant", "fortnite"}

func main() {
	logDir, e := os.UserConfigDir()
	if e != nil {
		logDir = os.TempDir()
	}
	logDir = filepath.Join(logDir, "SquadBase")
	if os.MkdirAll(logDir, 0700) != nil {
		msgBox("Konnte Konfigurationsordner nicht erstellen")
		return
	}
	diag := newStartupDiagnostic(logDir)
	defer diag.close()
	diag.note("Start")
	defer func() {
		if v := recover(); v != nil {
			diag.note("Unerwarteter Programmfehler (Details absichtlich nicht protokolliert)")
			msgBox("SQUAD BASE ist unerwartet beendet worden.\nStartdiagnose: " + diag.path)
		}
	}()
	releaseInstance, duplicate := acquireInstanceLock()
	if duplicate {
		diag.note("Zweite Instanz abgewiesen")
		msgBox("SQUAD BASE laeuft bereits. Bitte das bestehende Fenster verwenden.")
		return
	}
	defer releaseInstance()
	a := &App{folder: logDir, appearance: defaultAppearance(), clock: defaultClock(), overlay: defaultOverlay(), cfg: Config{Server: "play.hugosmp.de", GamePaths: map[string]string{}}, usage: map[string]int64{}, playing: map[string]bool{}, lastSeen: time.Now(), shutdownRequest: make(chan struct{}, 1)}
	loadJSON(filepath.Join(logDir, "config.json"), &a.cfg)
	loadJSON(filepath.Join(logDir, "arcade.json"), &a.arcade)
	loadJSON(filepath.Join(logDir, "usage.json"), &a.usage)
	loadJSON(filepath.Join(logDir, "appearance.json"), &a.appearance)
	loadJSON(filepath.Join(logDir, "clock.json"), &a.clock)
	if !validClock(a.clock) {
		a.clock = defaultClock()
	}
	loadJSON(filepath.Join(logDir, "overlay.json"), &a.overlay)
	if !validOverlay(a.overlay) {
		a.overlay = defaultOverlay()
	}
	if !validAppearance(a.appearance) {
		a.appearance = defaultAppearance()
	}
	a.loadLibrary()
	a.libraryTracker = newLibraryTracker(logDir)
	a.themePro = defaultThemePro()
	loadJSON(filepath.Join(logDir, "theme-pro.json"), &a.themePro)
	if !validThemePro(a.themePro) {
		a.themePro = defaultThemePro()
	}
	a.workspace = defaultWorkspace()
	var previousWorkspace WorkspacePrefs
	loadJSON(filepath.Join(logDir, "workspace-v41.json"), &previousWorkspace)
	if len(previousWorkspace.Order) > 0 {
		a.workspace = migrateWorkspace(previousWorkspace)
	}
	if !validWorkspace(a.workspace) {
		a.workspace = defaultWorkspace()
	}
	a.glass45 = defaultGlass45()
	loadJSON(filepath.Join(logDir, "glass45.json"), &a.glass45)
	if !validGlass45(a.glass45) {
		a.glass45 = defaultGlass45()
	}
	a.studio = defaultStudio()
	loadJSON(filepath.Join(logDir, "studio.json"), &a.studio)
	if !validStudio(a.studio) {
		a.studio = defaultStudio()
	}
	loadJSON(filepath.Join(logDir, "maintenance.json"), &a.maintenance)
	a.comfort = loadComfort(filepath.Join(logDir, "comfort.json"))
	if a.maintenance.Repository != "" && !repoPattern.MatchString(a.maintenance.Repository) {
		a.maintenance.Repository = ""
	}
	a.soundboard = newSoundBoard(logDir)
	a.friendPrefs = newFriendPrefStore(logDir)
	a.clicker = newAutoClicker(logDir)
	defer a.clicker.shutdown()
	a.remote.sounds = a.soundboard
	a.spotify = newSpotify(logDir)
	a.news = newNews()
	go a.releaseWatch()
	if a.cfg.GamePaths == nil {
		a.cfg.GamePaths = map[string]string{}
	}
	if a.usage == nil {
		a.usage = map[string]int64{}
	}
	if a.cfg.Server == "" {
		a.cfg.Server = "play.hugosmp.de"
	}
	t := make([]byte, 16)
	if _, e = rand.Read(t); e != nil {
		return
	}
	a.prefix = "/s/" + hex.EncodeToString(t) + "/"
	a.hw.Host, _ = os.Hostname()
	a.hw.CPUName = cpuName()
	go a.collectHardware()
	go a.monitorGames()
	go a.trackLibraryLoop()
	go a.serverLoop()
	mux := http.NewServeMux()
	mux.HandleFunc(a.prefix, a.handle)
	listener, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		diag.note("Lokaler Server konnte nicht starten")
		msgBox("Lokalen Server nicht erreichbar")
		return
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	go func() { _ = srv.Serve(listener) }()
	diag.note("Lokaler Server gestartet")
	// Keep the existing randomly selected UI port, but use a separate, stable
	// Spotify callback port: Spotify's dashboard often rejects port-less
	// loopback redirect URIs even though they are documented as valid.
	spotifyCallbackServer, callbackErr := startSpotifyCallbackServer(a.spotify)
	if callbackErr != nil {
		a.spotify.mu.Lock()
		a.spotify.callbackError = fmt.Sprintf("Spotify-Port 8888 konnte nicht geöffnet werden: %v. Schließe eine andere SQUAD BASE oder eine App, die Port 8888 benutzt.", callbackErr)
		a.spotify.mu.Unlock()
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = spotifyCallbackServer.Shutdown(ctx)
		}()
	}
	url := "http://" + listener.Addr().String() + a.prefix
	a.overlayURL = url + "overlay.html"
	trayExit, trayErr := startSystemTray(url, a.comfort.MinimizeToTray)
	if trayErr != nil {
		diag.note("Infobereich konnte nicht gestartet werden")
		// The window remains usable even if the optional tray host is unavailable.
		a.mu.Lock()
		a.comfort.MinimizeToTray = false
		a.mu.Unlock()
	}
	startOverlayHotkey(a.toggleOverlayShortcut)
	backgroundLaunch := hasArg("--minimized") && a.comfort.MinimizeToTray && trayErr == nil
	if backgroundLaunch {
		diag.note("Im Infobereich gestartet")
	} else if openAppWindow(url) != nil {
		diag.note("Fensterstart fehlgeschlagen; Browseradresse angezeigt")
		msgBox("Browser öffnen und Adresse eingeben:\n" + url)
	} else {
		diag.note("App-Fenster gestartet")
	}
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	defer a.remote.stop()
	for {
		select {
		case <-ticker.C:
			a.mu.RLock()
			idle := time.Since(a.lastSeen)
			keepInTray := a.comfort.MinimizeToTray && trayErr == nil
			a.mu.RUnlock()
			if idle > 100*time.Second && !keepInTray {
				diag.note("Normales Beenden nach Inaktivitaet")
				ctx, c := context.WithTimeout(context.Background(), 3*time.Second)
				_ = srv.Shutdown(ctx)
				c()
				stopSystemTray()
				return
			}
		case <-a.shutdownRequest:
			diag.note("Beenden für bestätigtes App-Update")
			ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
			_ = srv.Shutdown(ctx)
			c()
			stopSystemTray()
			return
		case <-trayExit:
			diag.note("Beenden über den Infobereich")
			ctx, c := context.WithTimeout(context.Background(), 3*time.Second)
			_ = srv.Shutdown(ctx)
			c()
			return
		}
	}
}
func loadJSON(path string, to any) {
	b, e := os.ReadFile(path)
	if e == nil {
		_ = json.Unmarshal(b, to)
	}
}
func saveJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	p := path + ".tmp"
	if e = os.WriteFile(p, b, 0600); e != nil {
		return e
	}
	// Atomic replace on POSIX; Windows may reject replacing an existing target.
	if err := os.Rename(p, path); err == nil {
		return nil
	}
	backup := path + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(path, backup); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(p)
		return err
	}
	if err := os.Rename(p, path); err != nil {
		_ = os.Rename(backup, path)
		return err
	}
	_ = os.Remove(backup)
	return nil
}
func send(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func failure(w http.ResponseWriter, status int, msg string) {
	send(w, status, map[string]string{"error": msg})
}
func (a *App) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://i.scdn.co https://mosaic.scdn.co https://cdn.akamai.steamstatic.com; connect-src 'self' https: http://127.0.0.1:* http://localhost:*; media-src 'self' blob:; frame-ancestors 'none'; object-src 'none'")
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host != "127.0.0.1" {
		failure(w, 403, "Nur localhost")
		return
	}
	name := strings.TrimPrefix(r.URL.Path, a.prefix)
	if r.Method == "GET" && (name == "" || name == "index.html" || name == "style.css" || name == "cozy-v35.css" || name == "cozy-v36.css" || name == "app.js" || name == "friends.js" || name == "spotify.js" || name == "spotify-library.js" || name == "updates.js" || name == "minigames.js" || name == "assistant.js" || name == "ai-images.js" || name == "soundtrack.js" || name == "hardware-chart.js" || name == "themes.js" || name == "clock.js" || name == "squad_cozy_orbit.ogg" || name == "squad_neon_sprint.ogg" || name == "overlay.html" || name == "overlay.css" || name == "overlay.js" || name == "hardware-pro.js" || name == "overlay-settings.js" || name == "v4.css" || name == "v4.js" || name == "v41.js" || name == "v41.css" || name == "v42.css" || name == "v42.js" || name == "v423.css" || name == "v43.css" || name == "v43.js" || name == "overlay-drag.js" || name == "v44.css" || name == "v44.js" || name == "v45.css" || name == "v45.js" || name == "v453.css" || name == "v454.css" || name == "v454.js" || name == "friend-alerts.js" || name == "calls.js" || name == "v455.css" || name == "comfort.css" || name == "comfort.js") {
		if name == "" {
			name = "index.html"
		}
		b, e := assets.ReadFile("web/" + name)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		switch filepath.Ext(name) {
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		case ".ogg":
			w.Header().Set("Content-Type", "audio/ogg")
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
		return
	}
	if !strings.HasPrefix(name, "api/") {
		http.NotFound(w, r)
		return
	}
	a.mu.Lock()
	a.lastSeen = time.Now()
	a.mu.Unlock()
	if r.Method == "POST" || r.Method == "DELETE" {
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
			failure(w, 403, "Ungültiger Origin")
			return
		}
		if r.Header.Get("X-Squad-Base") != "1" {
			failure(w, 403, "Sicherheitsprüfung fehlgeschlagen")
			return
		}
		maxBody := int64(8192)
		if name == "api/soundboard/upload" {
			maxBody = maxSoundBytes + 1
		}
		if name == "api/diagnostics" {
			maxBody = 2 << 20
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	}
	if name == "api/friends-cloud-config" {
		a.friendsCloudConfigAPI(w, r)
		return
	}
	if name == "api/remote" || name == "api/remote/events" {
		a.remote.handleLocal(a, w, r, name == "api/remote/events")
		return
	}
	if name == "api/friend-prefs" {
		a.friendPrefs.api(w, r)
		return
	}
	if name == "api/autoclicker" {
		a.clickerAPI(w, r)
		return
	}
	if name == "api/soundboard" || strings.HasPrefix(name, "api/soundboard/") {
		a.soundboard.api(w, r, name)
		return
	}
	if strings.HasPrefix(name, "api/spotify/") {
		a.spotify.handleAPI(w, r, strings.TrimPrefix(name, "api/"))
		return
	}
	if strings.HasPrefix(name, "api/ai-image/") {
		a.localImageAPI(w, r, name)
		return
	}
	if name == "api/news" || name == "api/news/refresh" {
		a.news.handleAPI(w, r, strings.TrimPrefix(name, "api/"))
		return
	}
	switch name {
	case "api/workspace":
		a.workspaceAPI(w, r)
		return
	case "api/library/pro":
		a.libraryProAPI(w, r)
		return
	case "api/glass45":
		a.glass45API(w, r)
		return
	case "api/theme-pro":
		a.themeProAPI(w, r)
		return
	case "api/assistant/context":
		a.assistantContextAPI(w, r)
		return
	case "api/overlay/position":
		a.overlayPositionAPI(w, r)
		return
	case "api/library":
		a.libraryAPI(w, r)
		return
	case "api/studio", "api/studio/wallpaper":
		a.studioAPI(w, r, name)
		return
	case "api/maintenance":
		a.updateAPI(w, r)
		return
	case "api/diagnostics":
		a.diagnosticsAPI(w, r)
		return
	case "api/comfort":
		a.comfortAPI(w, r)
		return
	case "api/arcade":
		a.arcadeAPI(w, r)
		return
	case "api/overlay":
		a.overlayAPI(w, r)
		return
	case "api/overlay/toggle":
		if r.Method != http.MethodPost {
			failure(w, 405, "Methode nicht erlaubt")
			return
		}
		if err := a.toggleOverlay(); err != nil {
			failure(w, 503, err.Error())
			return
		}
		a.mu.RLock()
		visible := a.overlayVisible
		a.mu.RUnlock()
		send(w, 200, map[string]any{"visible": visible})
		return
	case "api/appearance":
		a.appearanceAPI(w, r)
		return
	case "api/clock":
		a.clockAPI(w, r)
		return
	case "api/assistant/history":
		a.assistantHistoryAPI(w, r)
		return
	case "api/assistant/models":
		if r.Method != http.MethodGet {
			failure(w, 405, "Nur GET")
			return
		}
		models, e := availableLocalModels(r.Context())
		if e != nil {
			failure(w, 503, e.Error())
			return
		}
		send(w, 200, map[string]any{"models": models})
		return
	case "api/assistant":
		if r.Method == "POST" {
			var q struct {
				Message string     `json:"message"`
				History []chatTurn `json:"history"`
				Model   string     `json:"model"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil || len([]rune(q.Message)) > 600 || len(strings.TrimSpace(q.Message)) < 1 {
				failure(w, 400, "Bitte eine Frage mit maximal 600 Zeichen eingeben")
				return
			}
			if len(q.History) > 8 {
				failure(w, 400, "Zu viele Chatnachrichten")
				return
			}
			answer, model, e := localAIChat(r.Context(), q.Message, q.History, q.Model)
			if e != nil {
				failure(w, 503, e.Error())
				return
			}
			send(w, 200, map[string]string{"answer": answer, "model": model})
			return
		}
	case "api/state":
		if r.Method == "GET" {
			a.mu.RLock()
			v := struct {
				Hardware Hardware         `json:"hardware"`
				Config   Config           `json:"config"`
				Usage    map[string]int64 `json:"usage"`
				Playing  map[string]bool  `json:"playing"`
				Speed    SpeedResult      `json:"speed"`
				Ping     TargetPing       `json:"ping"`
				Server   ServerStatus     `json:"server"`
			}{a.hw, a.cfg, a.usage, a.playing, a.speed, a.ping, a.server}
			a.mu.RUnlock()
			send(w, 200, v)
			return
		}
	case "api/ping":
		if r.Method == "POST" {
			var q struct {
				Target string `json:"target"`
				Custom string `json:"custom"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil {
				failure(w, 400, "Ungültige Eingabe")
				return
			}
			a.mu.RLock()
			minecraft := a.cfg.Server
			a.mu.RUnlock()
			label, endpoint, e := pingEndpoint(q.Target, q.Custom, minecraft)
			if e != nil {
				failure(w, 422, e.Error())
				return
			}
			a.mu.Lock()
			if a.ping.Running {
				a.mu.Unlock()
				failure(w, 409, "Ping-Messung läuft bereits")
				return
			}
			a.ping = TargetPing{Running: true, Label: label, Endpoint: endpoint, Attempts: 5}
			a.mu.Unlock()
			go a.runPing(label, endpoint)
			send(w, 202, map[string]string{"message": "Ping-Test gestartet"})
			return
		}
	case "api/speed":
		if r.Method == "POST" {
			a.mu.Lock()
			if a.speed.Running {
				a.mu.Unlock()
				failure(w, 409, "Test läuft bereits")
				return
			}
			a.speed = SpeedResult{Running: true, Phase: "Ping messen"}
			a.mu.Unlock()
			go a.speedTest()
			send(w, 202, map[string]string{"message": "Test gestartet"})
			return
		}
	case "api/dns":
		if r.Method == "POST" {
			ctx, c := context.WithTimeout(r.Context(), 12*time.Second)
			defer c()
			out, e := hiddenOutput(ctx, "ipconfig.exe", "/flushdns")
			if e != nil {
				failure(w, 500, "DNS-Cache nicht geleert: "+strings.TrimSpace(string(out)))
				return
			}
			send(w, 200, map[string]string{"message": "Windows-DNS-Cache wirklich geleert."})
			return
		}
	case "api/temp":
		if r.Method == "POST" {
			if e := openFolder(os.TempDir()); e != nil {
				failure(w, 500, e.Error())
				return
			}
			send(w, 200, map[string]string{"message": "Temp-Ordner geöffnet; nichts automatisch gelöscht."})
			return
		}
	case "api/launch":
		if r.Method == "POST" {
			var q struct {
				ID string `json:"id"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil {
				failure(w, 400, "Ungültige Eingabe")
				return
			}
			if e := a.launch(q.ID); e != nil {
				failure(w, 422, e.Error())
				return
			}
			send(w, 200, map[string]string{"message": "Windows-Startbefehl ausgeführt."})
			return
		}
	case "api/open":
		if r.Method == "POST" {
			var q struct {
				ID string `json:"id"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil {
				failure(w, 400, "Ungültige Eingabe")
				return
			}
			if e := openLink(q.ID); e != nil {
				failure(w, 422, e.Error())
				return
			}
			send(w, 200, map[string]string{"message": "Geöffnet."})
			return
		}
	case "api/browse":
		if r.Method == "POST" {
			var q struct {
				ID string `json:"id"`
			}
			if json.NewDecoder(r.Body).Decode(&q) != nil || !validGame(q.ID) {
				failure(w, 400, "Unbekanntes Spiel")
				return
			}
			p, e := chooseEXE()
			if e != nil {
				failure(w, 500, e.Error())
				return
			}
			if p != "" {
				a.mu.Lock()
				a.cfg.GamePaths[q.ID] = p
				a.mu.Unlock()
				a.saveCfg()
			}
			send(w, 200, map[string]string{"path": p})
			return
		}
	case "api/config":
		if r.Method == "POST" {
			var q Config
			if json.NewDecoder(r.Body).Decode(&q) != nil {
				failure(w, 400, "Ungültige Einstellungen")
				return
			}
			q.Server = strings.TrimSpace(q.Server)
			if q.Server == "" || len(q.Server) > 180 || strings.ContainsAny(q.Server, " /\\?&#@") {
				failure(w, 422, "Ungültige Minecraft-Adresse")
				return
			}
			for k, v := range q.GamePaths {
				v = strings.Trim(strings.TrimSpace(v), "\"")
				q.GamePaths[k] = v
				if !validGame(k) || v != "" && (!filepath.IsAbs(v) || !strings.EqualFold(filepath.Ext(v), ".exe")) {
					failure(w, 422, "Nur absolute .exe-Pfade erlaubte Spiele")
					return
				}
			}
			a.mu.Lock()
			a.cfg.Server = q.Server
			for k, v := range q.GamePaths {
				a.cfg.GamePaths[k] = v
			}
			a.mu.Unlock()
			a.saveCfg()
			go a.updateServer()
			send(w, 200, map[string]string{"message": "Gespeichert"})
			return
		}
	}
	failure(w, 404, "Unbekannter Aufruf")
}
func (a *App) saveCfg() {
	a.mu.RLock()
	c := Config{Server: a.cfg.Server, GamePaths: map[string]string{}}
	for k, v := range a.cfg.GamePaths {
		c.GamePaths[k] = v
	}
	a.mu.RUnlock()
	_ = saveJSON(filepath.Join(a.folder, "config.json"), c)
}
func validGame(id string) bool {
	for _, s := range gameIDs {
		if s == id {
			return true
		}
	}
	return false
}
func (a *App) launch(id string) error {
	if !validGame(id) {
		return errors.New("Spiel nicht bekannt")
	}
	a.mu.RLock()
	p := a.cfg.GamePaths[id]
	a.mu.RUnlock()
	if p != "" {
		if filepath.Ext(p) != ".exe" && filepath.Ext(p) != ".EXE" {
			return errors.New("Nur ausführbare .exe-Dateien")
		}
		if _, e := os.Stat(p); e != nil {
			return errors.New("Spielpfad nicht gefunden. Bitte neu auswählen.")
		}
		return shellOpen(p)
	}
	defaults := map[string]string{"minecraft": "minecraft://", "rocket": "com.epicgames.launcher://apps/Sugar?action=launch&silent=true", "gta": "steam://rungameid/271590", "valorant": "riotclient://launch?raku=valorant", "fortnite": "com.epicgames.launcher://apps/Fortnite?action=launch&silent=true"}
	if e := shellOpen(defaults[id]); e != nil {
		return fmt.Errorf("Launcher nicht gefunden. Wähle die Spiel-EXE in Einstellungen aus. (%v)", e)
	}
	return nil
}
func openLink(id string) error {
	urls := map[string]string{"discord": "discord://", "discord-web": "https://discord.com/app", "steam": "https://steamcommunity.com/my/games/", "riot": "https://tracker.gg/valorant", "rocket": "https://rocketleague.tracker.network/", "fortnite": "https://fortnitetracker.com/", "epic": "https://www.epicgames.com/account/", "minecraft": "https://www.minecraft.net/"}
	if s, ok := urls[id]; ok {
		return shellOpen(s)
	}
	return errors.New("Link nicht bekannt")
}
func (a *App) collectHardware() {
	for {
		cpu, ram, used, total, gpu, temp, name := readHardware()
		usedVRAM, totalVRAM, fan, freeDisk, totalDisk := readExtraHardware()
		cpuTemp := optionalCPUTemp()
		a.mu.Lock()
		a.hw.CPU = cpu
		a.hw.RAM = ram
		a.hw.UsedRAM = used
		a.hw.TotalRAM = total
		a.hw.GPU = gpu
		a.hw.GPUTemp = temp
		a.hw.GPUVRAMUsed = usedVRAM
		a.hw.GPUVRAMTotal = totalVRAM
		a.hw.GPUFan = fan
		a.hw.DiskFreeGB = freeDisk
		a.hw.DiskTotalGB = totalDisk
		a.hw.CPUTemp = cpuTemp
		if name != "" {
			a.hw.GPUName = name
		}
		a.mu.Unlock()
		time.Sleep(3 * time.Second)
	}
}
func (a *App) monitorGames() {
	last := time.Now()
	for {
		now := time.Now()
		p := activeProcesses()
		elapsed := int64(now.Sub(last).Seconds())
		if elapsed < 0 || elapsed > 30 {
			elapsed = 0
		}
		a.mu.Lock()
		for _, id := range gameIDs {
			if a.playing[id] {
				a.usage[id] += elapsed
			}
		}
		a.playing = p
		usage := map[string]int64{}
		for k, v := range a.usage {
			usage[k] = v
		}
		a.mu.Unlock()
		_ = saveJSON(filepath.Join(a.folder, "usage.json"), usage)
		last = now
		time.Sleep(15 * time.Second)
	}
}
func (a *App) serverLoop() {
	for {
		a.updateServer()
		time.Sleep(30 * time.Second)
	}
}
func (a *App) updateServer() {
	a.mu.RLock()
	host := a.cfg.Server
	a.mu.RUnlock()
	s := probeMinecraft(host)
	a.mu.Lock()
	a.server = s
	a.mu.Unlock()
}
func (a *App) setSpeed(s SpeedResult) { a.mu.Lock(); a.speed = s; a.mu.Unlock() }

func (a *App) toggleOverlayShortcut() { _ = a.toggleOverlay() }
