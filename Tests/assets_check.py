"""Prüft, dass jede Datei, die index.html einbindet, auch von der Go-App ausgeliefert wird.

Die App liefert statische Dateien nur aus einer festen Namensliste in main.go aus. Eine neue Datei, die dort
fehlt, würde im laufenden Programm unbemerkt mit 404 fehlschlagen. Start:  python3 Tests/assets_check.py
"""
import re, sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
app = next(p for p in (root / 'Quellcode' / 'windows-app', root / 'windows-app') if p.exists())
html = (app / 'web' / 'index.html').read_text(encoding='utf-8')
go = (app / 'main.go').read_text(encoding='utf-8')
start = go.index('if r.Method == "GET" && (name == ""')
allowed = set(re.findall(r'name == "([^"]+)"', go[start:go.index('{\n', start)]))
needed = set(re.findall(r'<script src="([^"]+)"', html)) | set(re.findall(r'<link rel="stylesheet" href="([^"]+)"', html))
missing = sorted(n for n in needed if n not in allowed)
absent = sorted(n for n in needed if not (app / 'web' / n).exists())
if missing or absent:
    print('FEHLER: nicht in der Go-Liste:', missing, '| Datei fehlt:', absent)
    sys.exit(1)
print(f'PASS Asset-Prüfung: alle {len(needed)} eingebundenen Dateien existieren und stehen in der Go-Liste')
