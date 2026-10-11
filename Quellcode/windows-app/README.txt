SQUAD BASE v2.3 | NATIVE WINDOWS ARCADE
=======================================
- Minispiele und TCP-Ping-Auswahl fest eingebaut (kein Browser-Add-on).
- Reaktionszeit: 5-Runden-Challenge, Fehlstartanzeige, Bestzeit.
- Cube Rush: 3 Modi, Kristalle, Sprungpuffer, faires Spawning, Sound, pausierbar.
- Optimierter Canvas: vorberechneter Hintergrund, Fixed-Timestep, reduzierte HUD-Zugriffe.
- Verbesserte Datei-EXE-Auswahl sowie Entfernen von Anführungszeichen bei Pfaden.
- Smart App Control kann die *unsignierte* EXE weiterhin blockieren.

SQUAD BASE v2.2 (Minispiele + Ping-Zielauswahl)
================================================
Aufbauend auf v2.1; alles aus dem bisherigen Quellcode ist weiterhin enthalten.

NEU:
- "Minispiele": Reaktionszeit (Fehlstart, Bestzeit, Durchschnitt) und
  Cube Rush (springender Würfel, Hindernisse, steigendes Tempo, Pause, Rekord).
  Highscores werden lokal über Browser-LocalStorage gespeichert.
- "Internet & Ping": Separater realer TCP-Latenztest mit wählbaren Zielen
  Cloudflare, Google, Quad9, Minecraft-Server oder eigener Host:Port.
  Die Messung ist KEIN ICMP-Ping und KEIN Ingame-Ping.
- Go-Backend in ping.go, UI/Spiele in web/minigames.js, Go-Tests in ping_test.go.
- Programmversion v2.2; bestehende Windows-Einstellungen bleiben erhalten.

Siehe SquadBase_Arcade_Ping_Anleitung.txt im Paket.

SQUAD BASE v2.1 (Spotify + Spiele-Updates)
=======================================
Diese Version ergänzt die originale v2-Freunde-App. Spotify OAuth PKCE, Playlist-Anzeige und -Steuerung sowie echte Minecraft-RSS- und Steam-News sind hinzugefügt. Alle vorherigen Funktionen sind im Quellcode weiter enthalten. Details: SquadBase_Spotify_Updates_Anleitung.txt im Paket.

SQUAD BASE – COZY OAK EDITION v2: FREUNDE & CHAT
================================================
Windows 10 / Windows 11, 64-Bit

WAS IM PAKET STECKT
-------------------
1. SquadBase_Freunde.exe       - die erweiterte Windows-App mit allen bisherigen
                                 Funktionen (Hardware, Games, Speedtest, Minecraft)
                                 und dem neuen Reiter "Freunde & Chat".
2. SquadBaseRelay_Windows.exe - separates Serverprogramm für Windows.
3. SquadBaseRelay_Linux       - separates Serverprogramm für Linux x64.
4. source/                    - vollständiger Quellcode (Go, HTML, CSS, JS).

FREUNDESYSTEM: ECHTE FUNKTIONEN
-------------------------------
- Erstelle ein Benutzerkonto mit Namen und Passwort auf einem Freunde-Server.
- Freundschaftsanfragen senden, annehmen oder ablehnen.
- Nur bestätigte Freunde können private Nachrichten, Dateien und Anrufe senden.
- Freund entfernen, blockieren und Blockierung wieder aufheben.
- Nachrichten bleiben auf dem Server gespeichert und sind später abrufbar.
- Dateien bis 12 MB senden und herunterladen. Dateien nur von Personen öffnen,
  denen du vertraust. Die App scannt keine Anhänge nach Schadsoftware.
- Online-Status (aktiv in der App innerhalb der letzten 12 Sekunden).
- 1-zu-1-Sprachanrufe und Videoanrufe über WebRTC.
- Bildschirmfreigabe (du wählst selbst das Fenster/den Monitor). Mikrofon,
  Kamera und Bildschirm werden nur nach einer Freigabe benutzt.
- Für schwierige Netzwerkverbindungen unterstützt der Server optionale
  TURN-Konfiguration; ohne TURN können manche Anrufe scheitern.

WICHTIG: OHNE SERVER KEIN INTERNET-CHAT
---------------------------------------
Die Windows-EXE enthält kein öffentlich betriebenes Chatnetzwerk. Damit zwei
Freunde auf verschiedenen PCs kommunizieren, müssen beide die Adresse DESSELBEN
laufenden Squad-Base-Freundeservers eintragen. Eine öffentliche Domain mit
HTTPS und ein dauerhaft erreichbarer Rechner/Server sind erforderlich.
Ein Konto existiert nur auf dem jeweiligen Server; es ist kein Epic-,
Microsoft- oder Discord-Konto. Bei Serverneustart ist eine erneute Anmeldung
notwendig (Sitzungen sind absichtlich nicht dauerhaft gespeichert).

SCHNELLTEST AUF EINEM EINZELNEN PC (kein Hosting notwendig)
----------------------------------------------------------
1. SquadBaseRelay_Windows.exe per Doppelklick starten (Konsole offen lassen).
   Der Server lauscht standardmäßig nur auf 127.0.0.1:8787.
2. SquadBase_Freunde.exe starten, Reiter "Freunde & Chat" öffnen.
3. Als Serveradresse eintragen: http://127.0.0.1:8787
4. "Verbinden" und "Neuen Account erstellen" anklicken.
5. Zum Test eines zweiten Kontos ein zweites Browserprofil oder ein zweites
   Windows-Benutzerprofil nutzen. Es braucht ein eigenes Browser-Sitzungsfenster.
6. Mit Freundesnamen eine Anfrage senden und auf dem anderen Konto annehmen.
7. Dann schreiben / Datei senden / Anruf starten.

ACHTUNG: Ein Server auf 127.0.0.1 ist nur auf DIESEM PC erreichbar.
Ein HTTP-Server über öffentliches Internet ist NICHT sicher. NICHT den
Server einfach direkt ins Internet auf Port 8787 freigeben.

SERVER ÜBER INTERNET HOSTEN – BEISPIEL LINUX
--------------------------------------------
Du brauchst einen Linux-x64-Server, eine Domain und einen HTTPS-Reverse-Proxy.
Beispiel mit Caddy und chat.deine-domain.de (DNS A/AAAA auf Server-IP):

1. SquadBaseRelay_Linux auf den Linux-Server kopieren, ausführbar machen:
   chmod +x SquadBaseRelay_Linux
2. Starten, z.B.:
   ./SquadBaseRelay_Linux -listen 127.0.0.1:8787 -data /srv/squadbase/data
   (Ordner muss für den ausführenden Benutzer beschreibbar sein.)
3. Caddy installieren; Caddy-Konfiguration:
   chat.deine-domain.de {
       reverse_proxy 127.0.0.1:8787
   }
   Caddy kümmert sich bei richtig eingerichtetem DNS und freigegebenen
   Ports 80/443 um HTTPS. Den Relay-Port 8787 NICHT öffentlich freigeben.
4. Das Serverprogramm als Dienst einrichten, damit es dauerhaft läuft.
5. Auf JEDEM PC in Squad Base die Adresse
   https://chat.deine-domain.de eingeben.

Der komplette produktive Serverbetrieb (Hosting, DNS, TLS, Firewalls, Backups,
Aktualisierungen, Missbrauchsschutz) muss vom Betreiber durchgeführt werden.
Diese Version ist ein kleiner Self-Hosting-Prototyp und kein vollständig
geprüfter öffentlicher Messenger-Dienst.

OPTIONAL: TURN-SERVER FÜR ANRUFE
--------------------------------
Manche NATs oder Firewalls blockieren direkte Peer-Verbindungen. Dann muss
zusätzlich ein TURN-Server betrieben werden. Um ihn zu verwenden, beim Start
des Relay folgende Umgebungsvariablen setzen:
   SQUAD_TURN_URL        z.B. turn:turn.deine-domain.de:3478
   SQUAD_TURN_USER       TURN-Benutzername
   SQUAD_TURN_PASSWORD   TURN-Passwort
   (In HTTPS-Umgebungen ggf. turns:-Adresse nutzen.)
Diese TURN-Daten werden nur angemeldeten Nutzern ausgegeben. Für einen
öffentlichen Produktionsbetrieb sollten zeitlich begrenzte TURN-Credentials
verwendet werden, nicht langzeitige feste Passwörter.

DATENSCHUTZ UND GRENZEN
-----------------------
- Server empfängt Nachrichten und speichert Texte/Dateien dauerhaft im
  angegebenen Serverdatenordner. Der Serverbetreiber hat darauf Zugriff.
  CHAT-NACHRICHTEN SIND NICHT ENDE-ZU-ENDE-VERSCHLÜSSELT.
- WebRTC verschlüsselt Medien während der Übertragung (DTLS/SRTP), aber die
  Signalvermittlung läuft über den Server. TURN transportiert ggf. Medien.
- Passwort-Hashes verwenden PBKDF2-HMAC-SHA256 mit individuellem Salt.
- Kein Cloudanbieter, keine automatische Kontosynchronisierung, kein
  Drittanbieter-Login. Kein automatischer Virenscan für Dateien.
- Benutze ein neues, starkes Passwort. Versende keine sensiblen Dokumente.
- Keine Push-Benachrichtigungen, wenn die Windows-App geschlossen ist.
- Es sind 1-zu-1-Anrufe, keine Gruppenanrufe; keine fertigen Voice-Server.
- Keine E2EE für Nachrichten und keinen integrierten Dateilöschdialog.
- Ohne selbst betriebenen HTTPS-Server bleibt der Chat nur lokal testbar.
- Keine Garantie, dass Anrufe über jede Internetverbindung ohne TURN gehen.

ANDERE FUNKTIONEN
-----------------
Das bereits vorhandene Dashboard behält echte Windows-CPU-/RAM-Werte,
NVIDIA-GPU-Informationen (wenn Treibersensoren verfügbar sind), Cloudflare-
Speedtest, Minecraft-Status, Spiele-Launcher, Spielzeiterfassung und Tools.
Keine erfundenen Game-Ranks oder Epic-/Riot-Verknüpfungen.

QUELLCODE UND BUILD
-------------------
Go 1.22+; der Desktop-Client hat keine externen Go-Modul-Abhängigkeiten.
Mit installiertem Go im Quellordner:
  go test ./...
  go build -ldflags "-H=windowsgui" -o SquadBase_Freunde.exe .
  go build -o SquadBaseRelay_Windows.exe ./relay

PRÜFUNG
-------
Die Windows-EXEs wurden auf Linux für Windows x64 kompiliert.
Automatisierte Go-Tests für Registrierung, Authentifizierung, Freundschaft,
Nachrichten, Zugriffsschutz von Dateien, Blockierungen, Signale und die
ursprünglichen Funktionen liefen erfolgreich durch. JavaScript-Syntax wurde
geprüft. Ein vollständiger Test auf zwei echten Windows-PCs – vor allem
Mikrofon, Kamera und Bildschirmteilen – ist hier nicht möglich gewesen.

WICHTIG: Die EXE ist nicht digital signiert. Nur Software aus vertrauenswürdigen
Quellen starten. Microsoft Edge wird als App-Fenster verwendet.
