SQUAD BASE v3.0 – Liquid Glass / Arcade Edition
===============================================

Dieses Paket ist eine echte Windows-Desktop-App als 64-Bit-EXE. Die Oberfläche wird
wie bei v2.6 in einem eigenen Microsoft-Edge-App-Fenster dargestellt; es ist
keine Browser-Erweiterung nötig. Das Overlay ist ein zweites Topmost-App-Fenster
der gleichen laufenden Anwendung.

NEU IN 3.0
-----------
* Hardware Monitor Pro auf dem Dashboard RECHTS anstelle von Schnellstart & Tools
  - CPU, RAM, GPU, GPU-Temperatur, VRAM, GPU-Lüfter, Systemlaufwerk
  - CPU-Temperatur nur bei real verfügbarem LibreHardwareMonitor-WMI-Provider
  - FPS nur bei real verfügbarer Datenquelle (derzeit ohne FPS-Collector: n/v)
  - gemessene Zeitreihen, Auswahl 1/3/10 Minuten, einzeln ein-/ausschaltbar
  - bisherige Schnellstart-Tools weiterhin aufklappbar
* Liquid-Glass-Ingame-Overlay OBEN LINKS
  - Hotkey Strg+Umschalt+O, aktivieren auch über Einstellungen
  - Windows Topmost-Fenster (Edge installiert vorausgesetzt)
  - CPU/GPU/RAM/GPU-Temp/TCP-Ping, Spotify-Steuerung, Uhrzeit,
    Freunde-Shortcut, ein-/ausblenden je Anzeige
  - Position, Größe, Transparenz und optionale Klickdurchlässigkeit
  - funktioniert am besten bei Fenstermodus / randlosem Vollbild;
    exklusives Vollbild und Anti-Cheat-Titel können Overlay verhindern
  - keine Code-Injektion in Spiele
* Cube Rush 2.0
  - 4 Schwierigkeitsstufen, Schwierigkeit und Tempo steigen in jedem Run
  - Kampagne (5 Stufen), Endlos, Practice-Checkpoints
  - Sägen, Blöcke, Mehrfachspikes, Bounce-Pads, Sprung-Orbs,
    Tempo-Portale und Schwerkraft-Portale mit Decken-Hindernissen
  - Wiederbelebung 5/8/11/... Kristalle oder gratis Practice-Restart
  - persönliches Kristallkonto, vier freischaltbare Würfel-Skins
  - SFX, Synth-/Original-Soundtracks, Bestzeiten
* Spielstände (Arcade), UI-Farben, Animationsauswahl und Spielpfade
  werden im Windows-Konfigurationsverzeichnis gespeichert:
  %AppData%\SquadBase\

BEHALTENE FUNKTIONEN
-------------------
Game Launcher / NoRisk-EXE-Pfade, realer Internet-Test und TCP-Ping,
Minecraft-Server-Abfrage, Spotify-OAuth/Playlist-Integration, Freunde & Chat,
WebRTC Anrufe, Dateiübertragung und Bildschirmfreigabe (über separaten
Relay-Server), KI-Hilfe, Spiele-News, Musik, Reaktionszeit-Test und Designs.

START
-----
1. Windows 10/11 64-Bit mit Microsoft Edge (üblich bei Windows) benötigt.
2. Das Programm aus dem ZIP entpacken.
3. SquadBase_v3.0_LiquidGlass.exe starten.
4. Minispiele: Reiter "Minispiele" öffnen.
5. Overlay: Einstellungen > Liquid Glass – Ingame-Overlay > "Overlay starten".
   Hotkey: Strg + Umschalt + O. Kein Add-on und keine Browser-Erweiterung.
6. Für Freunde im Internet: Relay-Server auf einem eigenen Server mit HTTPS
   einrichten, Anleitung aus den bisherigen Versionen beachten.

WICHTIG – SMART APP CONTROL / SIGNIERUNG
--------------------------------------
DIESE EXE IST NICHT DIGITAL SIGNIERT. Wenn Windows Smart App Control sie
blockiert, ist das keine durch CSS, Go-Build-Flags oder Dateinamen lösbare
Meldung. Auch v3.0 kann deshalb auf deinem PC wieder BLOCKIERT werden.

Smart App Control bitte nicht deaktivieren. Für einen allgemein vertrauenswürdigen
Release muss ein berechtigter Herausgeber die Anwendung signieren bzw. offiziell
veröffentlichen, etwa über den Microsoft Store. Ohne Signierung kann eine
zuverlässige Freigabe nicht garantiert werden. Der vorliegende Download ist ein
Entwickler-/Testbuild, keine signierte Endkundenveröffentlichung.

TESTSTATUS
----------
PASS: Go-Unit-Tests Hauptanwendung und Relay, Go-Windows-Build 64-Bit.
PASS: JS-Tests für Wiederbelebungen, Practice, Kampagne, Skin-Ökonomie.
PASS: Browser-Oberflächentest für Dashboard/Hardwaregraph, Einstellungen,
      Cube Rush Start/Pause und Overlay-Datenansicht.
NICHT AUF ECHTEM WINDOWS-PC BESTÄTIGT: Topmost-Overlay über Spielen,
      Registrierung des globalen Hotkeys, Edge-Fenstertitel-Erkennung.
NICHT BESTÄTIGT: Spotify-OAuth mit echtem Konto, Voice/Video zwischen zwei
      PCs, Anti-Cheat/exklusiver Vollbildmodus, alle Sensoren auf jeder GPU.
NICHT ERLEDIGT: vertrauenswürdige digitale Signatur und Store-Freigabe.

Keine erfundenen Hardware-Messwerte: Fehlende Sensoren / FPS werden als
nicht verfügbar angezeigt. Für Spotify steuert die App nur über genehmigte
Spotify-APIs ein aktives Gerät, sie kopiert/streamt keine Spotify-Musik.
