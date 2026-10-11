SQUAD BASE v3.5 – COZY GLASS
============================
Dieses Projekt basiert auf v3.4, ohne die bestehenden Spiele-, Chat-, AI-, Musik- und Hardware-Schnittstellen zu ersetzen.

Neue UI-Pakete:
- web/cozy-v35.css
- web/clock.js
- clock.go (per-user storage unter clock.json)
- überarbeitete Overlay-Konfiguration (web/overlay-settings.js)
- AppearanceConfig.Cozy: classic | cozy | extra (leere Legacy-Werte bleiben gültig)

Windows-Build: Powershell BUILD_WINDOWS.ps1 ausführen. Go >= 1.22 erforderlich.
Unsignierte EXE kann von Windows Smart App Control blockiert werden. Keine Sicherheitsfunktionen deaktivieren.

Release-Tests siehe SquadBase_v3.5_Testbericht.txt im Komplettpaket.
