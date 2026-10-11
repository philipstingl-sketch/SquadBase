SQUAD BASE v2.6 – Design- und Hintergrund-Update
================================================

Neue Dateien:
  appearance.go          Server API mit dauerhaftem Speicher in appearance.json
  appearance_test.go     Persistenz-/Zugriffstests
  ui_v26_test.go         Asset- und UI-Routentests
  web/themes.js          Theme-Auswahl und Canvas-Animation

Verändert:
  main.go                Web-Asset und sichere appearance API
  web/index.html         Design-Einstellungen innerhalb der Windows-App
  web/style.css          Farbvariablen, Szenen, responsive UI
  web/hardware-chart.js  Grafikkurven verwenden gewählte Farben

Build unter Windows mit installiertem Go:
  cd <Quellcodeordner>
  go test ./...
  go build -trimpath -buildvcs=false -ldflags="-s -w -H=windowsgui" -o SquadBase.exe .

Die erstellte EXE muss durch einen verifizierten Herausgeber digital signiert
oder aus dem Microsoft Store ausgeliefert werden, um Smart-App-Control-Probleme
zuverlässig zu vermeiden. Keine Anleitung zum Abschalten von Windows-Schutz.
