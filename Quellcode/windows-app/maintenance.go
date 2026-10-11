package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const squadVersion = "4.7.0"

const (
	maxReleaseJSONBytes = 1 << 20
	maxChecksumBytes    = 1 << 20
	maxUpdateAssetBytes = 180 << 20
)

var repoPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,70}/[a-zA-Z0-9][a-zA-Z0-9_.-]{0,90}$`)

type Maintenance struct {
	Repository      string    `json:"repository"`
	AutoCheck       bool      `json:"autoCheck"`
	LastCheck       time.Time `json:"lastCheck"`
	Latest          string    `json:"latest"`
	Available       bool      `json:"available"`
	CanInstall      bool      `json:"canInstall"`
	LatestNotes     string    `json:"latestNotes"`
	LatestPublished string    `json:"latestPublished"`
}

type ReleaseInfo struct {
	Available      bool   `json:"available"`
	CanInstall     bool   `json:"canInstall"`
	CurrentVersion string `json:"currentVersion"`
	Notes          string `json:"notes"`
	Version        string `json:"version"`
	URL            string `json:"url"`
	Published      string `json:"published"`
	Message        string `json:"message"`
}

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Published  string        `json:"published_at"`
	Body       string        `json:"body"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

func parseVersion(version string) ([3]int, bool) {
	var nums [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(version), "v"), ".")
	if len(parts) < 2 || len(parts) > 3 {
		return nums, false
	}
	for i, p := range parts {
		p = strings.SplitN(p, "-", 2)[0]
		v, e := strconv.Atoi(p)
		if e != nil || v < 0 || v > 100000 {
			return nums, false
		}
		nums[i] = v
	}
	return nums, true
}

func newerVersion(current, latest string) bool {
	a, ok := parseVersion(current)
	b, ok2 := parseVersion(latest)
	if !ok || !ok2 {
		return false
	}
	for i := range a {
		if b[i] != a[i] {
			return b[i] > a[i]
		}
	}
	return false
}

func (a *App) updateAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.mu.RLock()
		m := a.maintenance
		a.mu.RUnlock()
		send(w, 200, map[string]any{
			"version": squadVersion, "repository": m.Repository, "autoCheck": m.AutoCheck,
			"lastCheck": m.LastCheck, "latest": m.Latest, "available": m.Available,
			"canInstall": m.CanInstall, "latestNotes": m.LatestNotes, "latestPublished": m.LatestPublished,
		})
		return
	}
	if r.Method != http.MethodPost {
		failure(w, 405, "Methode nicht erlaubt")
		return
	}
	var q struct {
		Action     string `json:"action"`
		Repository string `json:"repository"`
		AutoCheck  bool   `json:"autoCheck"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		failure(w, 400, "Ungültige Anfrage")
		return
	}
	switch q.Action {
	case "configure":
		q.Repository = strings.TrimSpace(q.Repository)
		if q.Repository != "" && !repoPattern.MatchString(q.Repository) {
			failure(w, 422, "Bitte nur owner/repo von GitHub eingeben")
			return
		}
		a.mu.Lock()
		a.maintenance.Repository = q.Repository
		a.maintenance.AutoCheck = q.AutoCheck && q.Repository != ""
		a.maintenance.LastCheck = time.Time{}
		a.maintenance.Latest = ""
		a.maintenance.Available = false
		a.maintenance.CanInstall = false
		a.maintenance.LatestNotes = ""
		a.maintenance.LatestPublished = ""
		err := saveJSON(filepath.Join(a.folder, "maintenance.json"), a.maintenance)
		a.mu.Unlock()
		if err != nil {
			failure(w, 500, "Nicht gespeichert")
			return
		}
		send(w, 200, map[string]any{"saved": true})
		return
	case "check":
		a.mu.RLock()
		repo := a.maintenance.Repository
		a.mu.RUnlock()
		if repo == "" {
			send(w, 200, ReleaseInfo{Message: "Kein offizielles Release-Repository eingerichtet. Trage unter Updates & Diagnose dein öffentliches GitHub-Repository ein."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		info, err := fetchLatestRelease(ctx, repo)
		if err != nil {
			failure(w, 503, err.Error())
			return
		}
		a.mu.Lock()
		if a.maintenance.Repository == repo {
			a.maintenance.LastCheck = time.Now()
			a.maintenance.Latest = info.Version
			a.maintenance.Available = info.Available
			a.maintenance.CanInstall = info.CanInstall
			a.maintenance.LatestNotes = info.Notes
			a.maintenance.LatestPublished = info.Published
			_ = saveJSON(filepath.Join(a.folder, "maintenance.json"), a.maintenance)
		}
		a.mu.Unlock()
		send(w, 200, info)
		return
	case "install":
		a.mu.RLock()
		repo := a.maintenance.Repository
		a.mu.RUnlock()
		if repo == "" {
			failure(w, 422, "Kein GitHub-Release-Repository eingerichtet")
			return
		}
		if runtime.GOOS != "windows" {
			failure(w, 501, "Die automatische EXE-Aktualisierung funktioniert nur unter Windows")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
		defer cancel()
		version, err := downloadAndScheduleUpdate(ctx, repo)
		if err != nil {
			failure(w, 503, err.Error())
			return
		}
		a.mu.Lock()
		a.maintenance.LastCheck = time.Now()
		a.maintenance.Latest = version
		a.maintenance.Available = false
		a.maintenance.CanInstall = false
		_ = saveJSON(filepath.Join(a.folder, "maintenance.json"), a.maintenance)
		a.mu.Unlock()
		send(w, 200, map[string]any{"scheduled": true, "version": version, "message": "Update geprüft und vorbereitet. SquadBase wird geschlossen, ersetzt und danach neu gestartet."})
		// Give the browser enough time to receive the successful API response.
		go func() {
			time.Sleep(1400 * time.Millisecond)
			a.requestShutdown()
		}()
		return
	case "open":
		a.mu.RLock()
		repo := a.maintenance.Repository
		a.mu.RUnlock()
		if !repoPattern.MatchString(repo) {
			failure(w, 422, "Kein Release-Repository hinterlegt")
			return
		}
		if err := shellOpen("https://github.com/" + repo + "/releases/latest"); err != nil {
			failure(w, 503, err.Error())
			return
		}
		send(w, 200, map[string]any{"opened": true})
		return
	default:
		failure(w, 422, "Unbekannte Aktion")
	}
}

func githubAPIClient() *http.Client {
	return &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if r.URL.Scheme != "https" || !strings.EqualFold(r.URL.Hostname(), "api.github.com") {
				return http.ErrUseLastResponse
			}
			if len(via) >= 3 {
				return fmt.Errorf("Zu viele Weiterleitungen")
			}
			return nil
		},
	}
}

func fetchGitHubRelease(ctx context.Context, repo string) (githubRelease, error) {
	if !repoPattern.MatchString(repo) {
		return githubRelease{}, fmt.Errorf("Kein gültiges GitHub-Repository")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "SquadBase/"+squadVersion+" (release updater)")
	res, err := githubAPIClient().Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("GitHub-Update-Dienst nicht erreichbar")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusNotFound {
			return githubRelease{}, fmt.Errorf("Kein öffentliches, veröffentlichtes Release gefunden. Repository und Release prüfen.")
		}
		return githubRelease{}, fmt.Errorf("GitHub hat die Release-Abfrage abgelehnt (HTTP %d)", res.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(res.Body, maxReleaseJSONBytes)).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("Ungültige Release-Daten von GitHub")
	}
	if release.Draft || release.Prerelease || release.TagName == "" {
		return githubRelease{}, fmt.Errorf("Das neueste GitHub-Release ist kein stabiles, veröffentlichtes Release")
	}
	if strings.ContainsAny(strings.TrimPrefix(release.TagName, "v"), "-+") {
		return githubRelease{}, fmt.Errorf("Das Release-Tag muss eine stabile Version wie v4.8.0 sein")
	}
	if _, ok := parseVersion(release.TagName); !ok {
		return githubRelease{}, fmt.Errorf("Das Release besitzt kein gültiges Versions-Tag (z. B. v4.8.0)")
	}
	return release, nil
}

func fetchLatestRelease(ctx context.Context, repo string) (ReleaseInfo, error) {
	release, err := fetchGitHubRelease(ctx, repo)
	if err != nil {
		return ReleaseInfo{}, err
	}
	info := ReleaseInfo{
		Available:      newerVersion(squadVersion, release.TagName),
		CurrentVersion: squadVersion,
		Version:        release.TagName,
		URL:            "https://github.com/" + repo + "/releases/latest",
		Published:      release.Published,
		Notes:          strings.TrimSpace(release.Body),
	}
	runes := []rune(info.Notes)
	if len(runes) > 6000 {
		info.Notes = string(runes[:6000]) + "…"
	}
	if !info.Available {
		info.Message = "Du nutzt bereits die neueste veröffentlichte Version."
		return info, nil
	}
	asset, assetErr := portableUpdateAsset(repo, release)
	if assetErr == nil {
		if _, checksumErr := resolveAssetSHA256(ctx, repo, release, asset); checksumErr == nil {
			info.CanInstall = true
			info.Message = "Neue Version verfügbar. Die EXE und ihre SHA-256-Prüfsumme wurden im Release gefunden. Installation startet erst nach deinem Klick."
			return info, nil
		}
	}
	info.Message = "Neue Version verfügbar, aber es konnte keine passende EXE mit verifizierbarer SHA-256-Prüfsumme gefunden werden. Öffne die Release-Seite und prüfe das Paket manuell."
	return info, nil
}

func portableUpdateAsset(repo string, release githubRelease) (githubAsset, error) {
	normalizedTag := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	if _, ok := parseVersion(normalizedTag); !ok {
		return githubAsset{}, fmt.Errorf("Ungültiges Versions-Tag")
	}
	expectedName := "SquadBase_v" + normalizedTag + "_Friends_Tools.exe"
	for _, asset := range release.Assets {
		if !strings.EqualFold(asset.Name, expectedName) {
			continue
		}
		if asset.Size < 512*1024 || asset.Size > maxUpdateAssetBytes {
			return githubAsset{}, fmt.Errorf("Die Release-EXE hat eine ungültige Dateigröße")
		}
		if _, err := safeGitHubAssetURL(repo, asset.BrowserDownloadURL); err != nil {
			return githubAsset{}, err
		}
		return asset, nil
	}
	return githubAsset{}, fmt.Errorf("Die portable EXE %s fehlt im Release", expectedName)
}

func safeGitHubAssetURL(repo, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("Unsichere Download-URL im Release")
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || !strings.EqualFold(u.Hostname(), "github.com") {
		return nil, fmt.Errorf("Download muss von github.com stammen")
	}
	expectedPrefix := "/" + parts[0] + "/" + parts[1] + "/releases/download/"
	if !strings.HasPrefix(strings.ToLower(u.EscapedPath()), strings.ToLower(expectedPrefix)) || u.User != nil {
		return nil, fmt.Errorf("Download-URL gehört nicht zu diesem GitHub-Repository")
	}
	return u, nil
}

func allowedGitHubDownloadHost(host string) bool {
	switch strings.ToLower(host) {
	case "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com", "github-releases.githubusercontent.com":
		return true
	default:
		return false
	}
}

func githubAssetClient() *http.Client {
	return &http.Client{
		Timeout: 90 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if r.URL.Scheme != "https" || !allowedGitHubDownloadHost(r.URL.Hostname()) {
				return fmt.Errorf("Unsichere Weiterleitung beim Update-Download")
			}
			if len(via) >= 5 {
				return fmt.Errorf("Zu viele Weiterleitungen beim Update-Download")
			}
			return nil
		},
	}
}

func downloadAssetBytes(ctx context.Context, repo string, asset githubAsset, maxBytes int64) ([]byte, error) {
	if _, err := safeGitHubAssetURL(repo, asset.BrowserDownloadURL); err != nil {
		return nil, err
	}
	if asset.Size <= 0 || asset.Size > maxBytes {
		return nil, fmt.Errorf("Ungültige Größe der Release-Datei %s", asset.Name)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "SquadBase/"+squadVersion+" (release updater)")
	res, err := githubAssetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("Release-Datei konnte nicht heruntergeladen werden")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Download fehlgeschlagen (HTTP %d)", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("Release-Datei konnte nicht gelesen werden")
	}
	if int64(len(data)) > maxBytes || (asset.Size > 0 && int64(len(data)) != asset.Size) {
		return nil, fmt.Errorf("Dateigröße von %s stimmt nicht mit GitHub überein", asset.Name)
	}
	return data, nil
}

func parseSHA256(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "sha256:") {
		raw = raw[len("sha256:"):]
	}
	if len(raw) != 64 {
		return ""
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil || len(decoded) != sha256.Size {
		return ""
	}
	return strings.ToLower(raw)
}

func sha256FromSums(content, fileName string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimLeft(fields[1], "*")
		if filepath.Base(name) == fileName {
			return parseSHA256(fields[0])
		}
	}
	return ""
}

func resolveAssetSHA256(ctx context.Context, repo string, release githubRelease, asset githubAsset) (string, error) {
	if digest := parseSHA256(asset.Digest); digest != "" {
		return digest, nil
	}
	for _, checksum := range release.Assets {
		if !strings.EqualFold(checksum.Name, "SHA256SUMS.txt") && !strings.EqualFold(checksum.Name, "SHA256SUMS") {
			continue
		}
		data, err := downloadAssetBytes(ctx, repo, checksum, maxChecksumBytes)
		if err != nil {
			continue
		}
		if hash := sha256FromSums(string(data), asset.Name); hash != "" {
			return hash, nil
		}
	}
	return "", fmt.Errorf("Für %s fehlt eine gültige SHA-256-Prüfsumme", asset.Name)
}

func downloadAndScheduleUpdate(ctx context.Context, repo string) (string, error) {
	release, err := fetchGitHubRelease(ctx, repo)
	if err != nil {
		return "", err
	}
	if !newerVersion(squadVersion, release.TagName) {
		return "", fmt.Errorf("Es gibt keine neuere Version zum Installieren")
	}
	asset, err := portableUpdateAsset(repo, release)
	if err != nil {
		return "", err
	}
	expectedHash, err := resolveAssetSHA256(ctx, repo, release, asset)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(expectedHash) == "" {
		return "", fmt.Errorf("Die Prüfsumme der EXE fehlt")
	}
	tmp, err := os.CreateTemp("", "SquadBase-update-*.exe")
	if err != nil {
		return "", fmt.Errorf("Temporäre Update-Datei konnte nicht erstellt werden")
	}
	tmpPath := tmp.Name()
	keepTemp := false
	defer func() {
		_ = tmp.Close()
		if !keepTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0700); err != nil {
		return "", fmt.Errorf("Temporäre Update-Datei konnte nicht geschützt werden")
	}
	if _, err := safeGitHubAssetURL(repo, asset.BrowserDownloadURL); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "SquadBase/"+squadVersion+" (release updater)")
	res, err := githubAssetClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("Update-Download fehlgeschlagen. Prüfe die Internetverbindung.")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Update-Download fehlgeschlagen (HTTP %d)", res.StatusCode)
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(res.Body, maxUpdateAssetBytes+1))
	if err != nil {
		return "", fmt.Errorf("Update-Download konnte nicht gespeichert werden")
	}
	if n > maxUpdateAssetBytes || n != asset.Size {
		return "", fmt.Errorf("Die heruntergeladene EXE hat eine unerwartete Dateigröße")
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), expectedHash) {
		return "", fmt.Errorf("Die SHA-256-Prüfsumme stimmt nicht. Update aus Sicherheitsgründen abgebrochen.")
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("Update-Datei konnte nicht geprüft werden")
	}
	magic := make([]byte, 2)
	if nMagic, readErr := tmp.Read(magic); readErr != nil || nMagic != 2 || string(magic) != "MZ" {
		return "", fmt.Errorf("Die heruntergeladene Datei ist keine gültige Windows-EXE")
	}
	if err := tmp.Sync(); err != nil {
		return "", fmt.Errorf("Update-Datei konnte nicht fertig geschrieben werden")
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("Update-Datei konnte nicht geschlossen werden")
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("Pfad der laufenden App konnte nicht ermittelt werden")
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", fmt.Errorf("Pfad der laufenden App ist ungültig")
	}
	probe, err := os.CreateTemp(filepath.Dir(executable), ".squadbase-update-check-*")
	if err != nil {
		return "", fmt.Errorf("Der App-Ordner ist nicht beschreibbar. Nutze bei installierter Version bitte die offizielle Release-Seite bzw. den Installer.")
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	if err := startExecutableReplacement(executable, tmpPath, os.Getpid()); err != nil {
		return "", fmt.Errorf("Update wurde geladen, kann aber nicht sicher angewendet werden: %v", err)
	}
	keepTemp = true
	return release.TagName, nil
}

func (a *App) requestShutdown() {
	if a.shutdownRequest == nil {
		return
	}
	select {
	case a.shutdownRequest <- struct{}{}:
	default:
	}
}

func (a *App) releaseWatch() {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	check := func() {
		a.mu.RLock()
		m := a.maintenance
		a.mu.RUnlock()
		if !m.AutoCheck || m.Repository == "" || time.Since(m.LastCheck) < 12*time.Hour {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		info, err := fetchLatestRelease(ctx, m.Repository)
		if err != nil {
			return
		}
		a.mu.Lock()
		if a.maintenance.Repository == m.Repository && a.maintenance.AutoCheck {
			a.maintenance.LastCheck = time.Now()
			a.maintenance.Latest = info.Version
			a.maintenance.Available = info.Available
			a.maintenance.CanInstall = info.CanInstall
			a.maintenance.LatestNotes = info.Notes
			a.maintenance.LatestPublished = info.Published
			_ = saveJSON(filepath.Join(a.folder, "maintenance.json"), a.maintenance)
		}
		a.mu.Unlock()
	}
	time.Sleep(5 * time.Second)
	check()
	for range ticker.C {
		check()
	}
}

func (a *App) diagnosticsSnapshot() map[string]any {
	a.mu.RLock()
	hardware := a.hw
	cfg := a.cfg
	playing := a.playing
	overlay := a.overlay
	comfort := a.comfort
	prefs := a.libraryPrefs
	a.mu.RUnlock()
	installed := 0
	for _, g := range prefs.Custom {
		if g.Path != "" {
			installed++
		}
	}
	configured := 0
	for _, v := range cfg.GamePaths {
		if v != "" {
			configured++
		}
	}
	active := 0
	for _, v := range playing {
		if v {
			active++
		}
	}
	return map[string]any{"version": squadVersion, "system": runtime.GOOS + "/" + runtime.GOARCH, "time": time.Now().Format(time.RFC3339), "hardwareDetected": hardware.CPUName != "", "gpuDataAvailable": hardware.GPU != nil, "cpuTemperatureAvailable": hardware.CPUTemp != nil, "gpuTemperatureAvailable": hardware.GPUTemp != nil, "configuredLauncherCount": configured, "userAddedGameCount": installed, "activeKnownGames": active, "overlayEnabled": overlay.Enabled, "startWithWindows": comfort.StartWithWindows, "minimizeToTray": comfort.MinimizeToTray, "themeMode": comfort.ThemeMode, "localServer": "127.0.0.1 (nur dieses Gerät)", "updateSigning": "Nicht digital signiert – Smart App Control kann sperren", "personalAccounts": "Nicht in Diagnose enthalten", "secretsIncluded": false}
}
func (a *App) diagnosticsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		send(w, 200, a.diagnosticsSnapshot())
		return
	}
	if r.Method != http.MethodPost {
		failure(w, 405, "Methode nicht erlaubt")
		return
	}
	var q struct {
		Action string          `json:"action"`
		Backup *SettingsBackup `json:"backup,omitempty"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		failure(w, 400, "Ungültige Anfrage")
		return
	}
	switch q.Action {
	case "restore-latest":
		a.restoreLatest(w)
		return
	case "restore-upload":
		if q.Backup == nil {
			failure(w, 400, "Keine Sicherungsdaten gefunden")
			return
		}
		raw, err := json.Marshal(q.Backup)
		if err != nil {
			failure(w, 422, "Sicherungsdatei ist ungültig")
			return
		}
		a.restoreBackupBytes(w, raw)
		return
	case "report":
		b, e := json.MarshalIndent(a.diagnosticsSnapshot(), "", "  ")
		if e != nil {
			failure(w, 500, "Bericht fehlgeschlagen")
			return
		}
		report := filepath.Join(a.folder, "SQUAD_BASE_Diagnose_"+time.Now().Format("20060102_150405")+".json")
		if e = os.WriteFile(report, b, 0600); e != nil {
			failure(w, 500, "Bericht konnte nicht gespeichert werden")
			return
		}
		send(w, 200, map[string]string{"message": "Diagnosebericht lokal gespeichert. Keine Passwörter oder Account-Tokens enthalten.", "file": report})
		return
	case "backup":
		a.mu.RLock()
		b := a.settingsSnapshot()
		a.mu.RUnlock()
		data, e := json.MarshalIndent(b, "", "  ")
		if e != nil {
			failure(w, 500, "Backup fehlgeschlagen")
			return
		}
		p := filepath.Join(a.folder, "SQUAD_BASE_Einstellungen_"+time.Now().Format("20060102_150405")+".json")
		if e = os.WriteFile(p, data, 0600); e != nil {
			failure(w, 500, "Backup nicht gespeichert")
			return
		}
		send(w, 200, map[string]string{"message": "Einstellungen lokal gesichert; keine Spotify-Tokens oder Chatdateien enthalten. Achtung: Spielpfade und Spielstände sind privat.", "file": p})
		return
	case "export":
		a.mu.RLock()
		b := a.settingsSnapshot()
		a.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Disposition", `attachment; filename="SQUAD_BASE_Einstellungen_`+time.Now().Format("20060102_150405")+`.json"`)
		if err := json.NewEncoder(w).Encode(b); err != nil {
			return
		}
		return
	case "folder":
		if err := openFolder(a.folder); err != nil {
			failure(w, 503, err.Error())
			return
		}
		send(w, 200, map[string]any{"opened": true})
		return
	default:
		failure(w, 422, "Unbekannte Aktion")
	}
}
