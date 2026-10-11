from pathlib import Path
p=Path(__file__).parent
s=(p/'workspace_v41.go').read_text()
s=s.replace('SpotifyDock string   `json:"spotifyDock"`','SpotifyDock string   `json:"spotifyDock"`\n\tVersion     int      `json:"version"`\n\tCozyStart   bool     `json:"cozyStart"`\n\tSmartGaming bool    `json:"smartGaming"`')
s=s.replace('var dashboardCards = []string{"system", "monitor", "games", "network", "news"}', 'var dashboardCards = []string{"system", "monitor", "games", "network", "news", "clockwidget", "musicwidget", "friendswidget", "sessionwidget"}\nvar legacyDashboardCards = []string{"system", "monitor", "games", "network", "news"}\nvar optionalDashboardCards = []string{"clockwidget", "musicwidget", "friendswidget", "sessionwidget"}')
s=s.replace('return WorkspacePrefs{Preset: "gaming", Order: append([]string(nil), dashboardCards...), Hidden: []string{}, Wide: []string{"news"}, VoiceRate: 1, SpotifyDock: "bar"}', 'return WorkspacePrefs{Preset: "gaming", Order: append([]string(nil), dashboardCards...), Hidden: append([]string(nil), optionalDashboardCards...), Wide: []string{"news"}, VoiceRate: 1, SpotifyDock: "bar", Version: 2, CozyStart: true, SmartGaming: true}')
# Migration of old 5-card dashboard without losing saved choices.
needle='func uniqueCardList(xs []string) ([]string, bool) {'
addon='''// The previous version stored five cards. Append widgets without altering personal
// order, hidden cards, Spotify settings or voice preferences.
func migrateWorkspace(x WorkspacePrefs) WorkspacePrefs {
 if x.Version >= 2 { return x }
 // Recognize only a valid legacy card permutation; corrupted files still fall
 // back to defaultWorkspace during the normal validation step.
 if len(x.Order) != len(legacyDashboardCards) { return x }
 seen := map[string]bool{}
 for _, id := range x.Order { if !slices.Contains(legacyDashboardCards, id) || seen[id] { return x }; seen[id] = true }
 x.Order = append(x.Order, optionalDashboardCards...)
 x.Hidden = append(x.Hidden, optionalDashboardCards...)
 x.Version = 2
 x.CozyStart = true
 x.SmartGaming = true
 return x
}
'''
s=s.replace(needle,addon+needle)
s=s.replace('if !slices.Contains([]string{"gaming", "cozy", "minimal", "social", "custom"}, x.Preset) {','if x.Version != 2 || !slices.Contains([]string{"gaming", "cozy", "minimal", "social", "custom"}, x.Preset) {')
(p/'workspace_v41.go').write_text(s)
s=(p/'main.go').read_text().replace('if !validWorkspace(a.workspace) {','a.workspace = migrateWorkspace(a.workspace)\n\tif !validWorkspace(a.workspace) {')
s=s.replace('|| name == "v41.css") {','|| name == "v41.css" || name == "v42.css" || name == "v42.js") {')
(p/'main.go').write_text(s)
# rewrite legacy tests case length to account for 9 cards
s=(p/'workspace_v41_test.go').read_text()
s=s.replace('x.Order = []string{"network", "monitor", "games", "system", "news"}', 'x.Order = []string{"network", "monitor", "games", "system", "news", "clockwidget", "musicwidget", "friendswidget", "sessionwidget"}')
(p/'workspace_v41_test.go').write_text(s)
# html: shift editing selections behind a dedicated mode
h=(p/'web/index.html').read_text()
h=h.replace('· 4.1 · Cozy Workspace','· 4.2 · Cozy Studio')
h=h.replace('<link rel="stylesheet" href="v41.css">','<link rel="stylesheet" href="v41.css"><link rel="stylesheet" href="v42.css">')
h=h.replace('<script src="v41.js"></script>','<script src="v42.js"></script>')
h=h.replace('''<div class="v41-dashboard-toolbar-actions"><label for="v41-layout-preset">Layout</label><select id="v41-layout-preset">''','''<div class="v41-dashboard-toolbar-actions"><div id="v42-edit-options" hidden><label for="v41-layout-preset">Layout</label><select id="v41-layout-preset">''')
h=h.replace('''<option value="custom">Individuell</option></select><button class="outline" type="button" id="v41-edit-toggle" aria-pressed="false">✎ Dashboard bearbeiten</button><button type="button" class="outline" id="v41-layout-reset" hidden>↶ Zurücksetzen</button></div>''','''<option value="custom">Individuell</option></select><select id="v42-widget-select" aria-label="Widget auswählen"><option value="">+ Widget auswählen</option><option value="clockwidget">◷ Uhr</option><option value="musicwidget">♫ Spotify</option><option value="friendswidget">☻ Freunde</option><option value="sessionwidget">⚡ Session & PC</option></select><button class="outline" id="v42-widget-add" type="button">+ Hinzufügen</button><button class="outline" type="button" id="v41-layout-reset" hidden>↶ Zurücksetzen</button></div><button class="outline" id="v42-cancel" type="button" hidden>Abbrechen</button><button class="primary" id="v42-save" type="button" hidden>✓ Speichern & Fertig</button><button class="outline" type="button" id="v41-edit-toggle" aria-pressed="false">✎ Bearbeiten</button></div>''')
h=h.replace('Karten am Griff verschieben, ein-/ausblenden oder verbreitern. Mit ↑ und ↓ funktioniert die Reihenfolge auch ohne Maus.', 'Nur im Bearbeitungsmodus: Widgets hinzufügen, Karten verschieben oder ausblenden. Mit Speichern & Fertig übernimmst du die Änderungen; Abbrechen verwirft sie.')
# insert cozy start section, no simulated data
needle='<div class="v41-dashboard-toolbar"'
hero='''<section class="v42-welcome" id="v42-cozy-welcome" aria-label="Cozy-Startbildschirm"><div class="v42-welcome-art"><span class="v42-orbit v42-orbit-one"></span><span class="v42-orbit v42-orbit-two"></span></div><div class="v42-welcome-main"><span class="v42-kicker">✦ DEINE COZY ECKE</span><h2 id="v42-greeting">Willkommen zurück</h2><p>Musik an, Freunde online und die Spiele warten. Mach es dir gemütlich.</p><div class="v42-welcome-actions"><button class="primary" type="button" data-goto="library">▣ Spielebibliothek</button><button class="outline" type="button" data-goto="spotify">♫ Musik öffnen</button></div></div><div class="v42-hero-side"><div class="v42-hero-clock" id="v42-hero-clock">--:--</div><div id="v42-hero-date">Datum wird geladen</div><div class="v42-hero-divider"></div><small>AKTUELLER SPIELSTATUS</small><strong id="v42-hero-game">Kein Spiel erkannt</strong><small id="v42-hero-system">System bereit</small></div></section>\n'''
h=h.replace(needle,hero+needle,1)
# add settings with controls
needle='<article class="panel v41-workspace-settings">'
block='''<article class="panel v42-workspace-settings"><div class="section-head"><div><h2>✦ Cozy Start & Smart Gaming</h2><p>Willkommensansicht, zusätzliche Widgets und automatische Leistungsanpassung.</p></div><span class="tag">v4.2</span></div><div class="v42-setting-pairs"><label class="v42-setting-option"><input type="checkbox" id="v42-start-toggle" checked><span><strong>Cozy-Startbildschirm</strong><small>Begrüßung, große Uhr, Datum und aktueller Spielstatus auf dem Dashboard</small></span></label><label class="v42-setting-option"><input type="checkbox" id="v42-smart-toggle" checked><span><strong>Smart Gaming-Modus</strong><small>Automatisch weniger Animationen und Hintergrundbewegung bei erkanntem Spiel</small></span></label></div><p id="v42-smart-status" class="hint" role="status">Smart Gaming wartet auf einen erkannten Spielprozess. Es verändert weder Windows-Einstellungen noch Spiele.</p><button class="outline" type="button" id="v42-open-edit">Dashboard-Widgets verwalten →</button></article>\n'''
h=h.replace(needle,block+needle,1)
h=h.replace('<span class="tag">v4.1</span></div><div class="v41-settings-grid">','<span class="tag">v4.2</span></div><div class="v41-settings-grid">')
(p/'web/index.html').write_text(h)
