from pathlib import Path
base=Path(__file__).parent
p=base/'web/index.html'
s=p.read_text()
s=s.replace('SQUAD BASE · 4.0 · Liquid Glass','SQUAD BASE · 4.1 · Cozy Workspace')
s=s.replace('<link rel="stylesheet" href="v4.css">','<link rel="stylesheet" href="v4.css"><link rel="stylesheet" href="v41.css">')
s=s.replace('<span class="version">v4.0</span>','<span class="version">v4.1</span>')
s=s.replace('>Cube Rush 3.6</h2>','>Cube Rush 4.1</h2>')
s=s.replace('>◆ Cube Rush 3.6</h2>','>◆ Cube Rush 4.1</h2>')
# dashboard personalization toolbar inserted after intro before grid, only first occurrence
needle='<div class="grid two"><article class="panel"><div class="section-head"><div><h2>▣ System & Hardware</h2>'
repl='''<div class="v41-dashboard-toolbar" aria-label="Dashboard anpassen">
 <div class="v41-dashboard-toolbar-info"><span class="v41-edit-icon">✧</span><div><strong>Dein Dashboard</strong><small id="v41-layout-status">Gaming · Layout gespeichert</small></div></div>
 <div class="v41-dashboard-toolbar-actions"><label for="v41-layout-preset">Layout</label><select id="v41-layout-preset"><option value="gaming">Gaming</option><option value="cozy">Cozy</option><option value="minimal">Minimal</option><option value="social">Social</option><option value="custom">Individuell</option></select><button class="outline" type="button" id="v41-edit-toggle" aria-pressed="false">✎ Dashboard bearbeiten</button><button type="button" class="outline" id="v41-layout-reset" hidden>↶ Zurücksetzen</button></div>
</div>
<div class="v41-edit-help" id="v41-edit-help" hidden>Karten am Griff verschieben, ein-/ausblenden oder verbreitern. Mit ↑ und ↓ funktioniert die Reihenfolge auch ohne Maus.</div>
<div class="grid two"><article class="panel"><div class="section-head"><div><h2>▣ System & Hardware</h2>'''
assert needle in s
s=s.replace(needle,repl,1)
# assistant voice controls added before chat form
needle='<form id="assistant-form" class="assistant-compose">'
voice='''<div class="v41-voice-controls" aria-label="Sprachassistent"><button class="outline" id="v41-voice-record" type="button" aria-pressed="false">🎙 Spracheingabe</button><button class="outline" id="v41-voice-stop" type="button">◼ Stimme stoppen</button><label class="v41-check"><input id="v41-read-aloud" type="checkbox"> Antworten vorlesen</label><span class="v41-voice-indicator" id="v41-voice-indicator" role="status">Mikrofon nur nach Klick aktiv</span></div>'''
assert needle in s
s=s.replace(needle,voice+needle)
# create voice settings next to assistant settings below
needle='<article class="panel assistant-tips"><h2>Häufige Fragen</h2>'
assert needle in s
s=s.replace(needle,'''<article class="panel assistant-tips"><h2>Häufige Fragen</h2><details class="v41-voice-settings"><summary>🎤 Sprachmodus anpassen</summary><label>Vorlesestimme <select id="v41-voice-name"><option value="">Windows-Standard</option></select></label><label>Sprechtempo <output id="v41-voice-rate-value">1,0×</output><input type="range" min="0.7" max="1.4" step="0.1" value="1" id="v41-voice-rate"></label><p class="hint">Sprachausgabe nutzt Windows-/Edge-Stimmen. Spracheingabe benötigt eine Mikrofonfreigabe; Edge kann für die Erkennung einen Onlinedienst verwenden. Es werden keine Mitschnitte in SQUAD BASE gespeichert.</p></details>''',1)
# Spotify config controls add to spotify section after page title by replacing one section markup
idx=s.index('<section class="page" id="spotify">')
point=s.index('<div class="spotify-layout">',idx)
s=s[:point]+'''<div class="v41-spotify-toolbar"><span>♫ Spotify-Musikleiste</span><label>Ansicht <select id="v41-spotify-dock-select"><option value="bar">Schmale Leiste unten</option><option value="floating">Schwebender Player</option><option value="hidden">Nur im Spotify-Reiter</option></select></label><small>Steuerung und Playlists nur nach offizieller Spotify-Anmeldung · Premium/Freigabe erforderlich</small></div>\n'''+s[point:]
# Append seek volume to existing mini before mini-controls div for progress, volume after
needle='<div class="spotify-mini-controls"><button id="spotify-mini-prev"'
assert needle in s
s=s.replace(needle,'''<div class="v41-mini-timeline"><input id="v41-spotify-seek" type="range" min="0" max="1000" value="0" disabled aria-label="Songposition ändern"><div class="v41-mini-times"><span id="v41-mini-current">0:00</span><span id="v41-mini-total">0:00</span></div></div><div class="spotify-mini-controls"><button id="spotify-mini-prev"''',1)
needle='</div></div></aside><footer>'
assert needle in s
s=s.replace(needle,'''</div><div class="v41-mini-volume"><label for="v41-spotify-volume">◖ Lautstärke</label><input id="v41-spotify-volume" type="range" min="0" max="100" value="40" disabled></div></div></aside><footer>''',1)
# settings add local layout and speech options panel just before end of settings (can search final config panel)
needle='<article class="panel settings-panel"><h2>Server & Spielpfade</h2>'
assert needle in s
s=s.replace(needle,'''<article class="panel v41-workspace-settings"><div class="section-head"><div><h2>▧ Arbeitsbereich & Bedienung</h2><p>Alle Änderungen bleiben in deinem eigenen Windows-Profil.</p></div><span class="tag">v4.1</span></div><div class="v41-settings-grid"><div><strong>Dashboard</strong><p class="hint">Karten verschieben, verkleinern oder ausblenden.</p><button class="outline" type="button" data-goto="dashboard">Dashboard bearbeiten →</button></div><div><strong>Spotify-Miniplayer</strong><p class="hint">Musikleiste, schwebender Player oder keine Einblendung.</p><button class="outline" type="button" data-goto="spotify">Musikleiste einstellen →</button></div><div><strong>Sprachassistent</strong><p class="hint">Mikrofon auf Knopfdruck und optionale Sprachausgabe.</p><button class="outline" type="button" data-goto="assistant">Sprachmodus öffnen →</button></div><div><strong>Wartung & Backups</strong><p class="hint">Sicherung, Versionsprüfung und lokale Fehlerdiagnose.</p><button class="outline" type="button" data-goto="maintenance">Wartung öffnen →</button></div></div></article>
<article class="panel settings-panel"><h2>Server & Spielpfade</h2>''',1)
# append after v4
s=s.replace('<script src="v4.js"></script>','<script src="v4.js"></script><script src="v41.js"></script>')
p.write_text(s)
print('HTML v41 done')
