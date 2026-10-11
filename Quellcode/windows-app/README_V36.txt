SQUAD BASE v3.6 – Cozy Studio
============================
Basiert auf SQUAD BASE v3.5 und nutzt dieselbe Windows-GUI-Starttechnik.

Verbesserungen:
- Drei neue Hintergründe (Snowfall, Cloud Drift, Golden Hour)
- Helligkeit, Dunkelheit, Randabdunklung, Partikeldichte
- Speicherung im alten Windows-Profil, vorhandene Konfiguration bleibt gültig
- Cube Rush Menüs einklappbar und logisch gruppiert
- Fehlerkorrekturen für Tastatur/Pause und bezahlbare Wiederbelebungen
- Zusätzliche UI- und Go-Tests, neuer Stylesheet-Layer

Windows-Build:
  go test ./...
  $env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
  go build -trimpath -ldflags '-H windowsgui -s -w' -o SquadBase_v3.6_CozyStudio.exe .

Nicht digital signiert: Windows Smart App Control kann den Start blockieren.
Eine vertrauenswürdige Codesignatur erfordert einen verifizierten Herausgeber.
