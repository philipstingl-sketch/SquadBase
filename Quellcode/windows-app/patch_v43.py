from pathlib import Path
p=Path('main.go');s=p.read_text()
s=s.replace('workspace      WorkspacePrefs\n', 'workspace      WorkspacePrefs\n\tlibraryTracker *LibraryTracker\n\tthemePro ThemePro\n')
s=s.replace('a.loadLibrary()','a.loadLibrary()\n\ta.libraryTracker = newLibraryTracker(logDir)\n\ta.themePro = defaultThemePro()\n\tloadJSON(filepath.Join(logDir, "theme-pro.json"), &a.themePro)\n\tif !validThemePro(a.themePro) { a.themePro=defaultThemePro() }',1)
s=s.replace('go a.monitorGames()', 'go a.monitorGames()\n\tgo a.trackLibraryLoop()',1)
s=s.replace('name == "v423.css")','name == "v423.css" || name == "v43.css" || name == "v43.js")')
s=s.replace('https://mosaic.scdn.co; connect-src','https://mosaic.scdn.co https://cdn.akamai.steamstatic.com; connect-src')
s=s.replace('case "api/library":\n', 'case "api/library/pro":\n\t\ta.libraryProAPI(w,r)\n\t\treturn\n\tcase "api/theme-pro":\n\t\ta.themeProAPI(w,r)\n\t\treturn\n\tcase "api/assistant/context":\n\t\ta.assistantContextAPI(w,r)\n\t\treturn\n\tcase "api/overlay/position":\n\t\ta.overlayPositionAPI(w,r)\n\t\treturn\n\tcase "api/library":\n')
s=s.replace('func saveJSON(path string, v any) error {\n', 'func saveJSON(path string, v any) error {\n')
# ensure old config survives failure when replacing on Windows; rename old config to rollback until temp is in place
s=s.replace('''\t_ = os.Remove(path)
\treturn os.Rename(p, path)
}''','''\t// Atomic replace on POSIX; Windows may reject replacing an existing target.
\tif err := os.Rename(p,path); err == nil { return nil }
\tbackup := path + ".old"
\t_ = os.Remove(backup)
\tif err := os.Rename(path, backup); err != nil && !os.IsNotExist(err) { _ = os.Remove(p); return err }
\tif err := os.Rename(p, path); err != nil { _ = os.Rename(backup,path);return err }
\t_ = os.Remove(backup)
\treturn nil
}''')
p.write_text(s)
p=Path('spotify.go');s=p.read_text()
s=s.replace('''\tcase "spotify/desktop":''','''\tcase "spotify/queue", "spotify/playback-options":
\t\ts.musicCenterAPI(w,r,name)
\t\treturn
\tcase "spotify/desktop":''',1)
s=s.replace('''\t\t\tIsPlaying  bool `json:"is_playing"`''','''\t\t\tIsPlaying  bool `json:"is_playing"`
\t\t\tShuffleState bool `json:"shuffle_state"`
\t\t\tRepeatState string `json:"repeat_state"`''')
s=s.replace('''"volume": p.Device.VolumePercent})''','''"volume": p.Device.VolumePercent, "shuffle": p.ShuffleState, "repeat": p.RepeatState})''')
p.write_text(s)
p=Path('library.go');s=p.read_text().replace('''\t\t\tsend(w, 200, map[string]string{"message": "Start angefordert"})''','''\t\t\tif a.libraryTracker != nil {a.libraryTracker.markLaunched(q.ID)}
\t\t\tsend(w, 200, map[string]string{"message": "Start angefordert"})''');p.write_text(s)
p=Path('overlay.go');s=p.read_text();s += '''
// Reposition via dragging the visible overlay header. Disabled while
// click-through is enabled, to avoid intercepting input in games.
func (a *App) overlayPositionAPI(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodPost {failure(w,405,"Nur POST");return}
 var req struct {X int `json:"x"`;Y int `json:"y"`}
 if json.NewDecoder(r.Body).Decode(&req)!=nil || req.X<0||req.Y<0||req.X>8000||req.Y>8000 {failure(w,422,"Ungültige Position");return}
 a.mu.Lock()
 if a.overlay.ClickThrough {a.mu.Unlock();failure(w,409,"Ziehfunktion ist im Klick-durchlassen-Modus deaktiviert");return}
 cfg:=a.overlay;cfg.X=req.X;cfg.Y=req.Y
 if err:=saveJSON(filepath.Join(a.folder,"overlay.json"),cfg);err!=nil {a.mu.Unlock();failure(w,500,"Position konnte nicht gespeichert werden");return}
 a.overlay=cfg;visible:=a.overlayVisible;a.mu.Unlock()
 if visible {applyOverlayWindow(cfg)}
 send(w,200,map[string]any{"x":cfg.X,"y":cfg.Y})
}
''';p.write_text(s)
# maintenance version
p=Path('maintenance.go');s=p.read_text();s=s.replace('4.2.3','4.3.0').replace('4.2.2','4.3.0') if '4.2.3' in s or '4.2.2' in s else s
p.write_text(s)
# remove unused imports library_pro
p=Path('library_pro.go');s=p.read_text().replace('  "encoding/json"\n','').replace('  "os"\n','').replace('var _ = json.Valid\nvar _ = os.ErrNotExist\n','');p.write_text(s)
