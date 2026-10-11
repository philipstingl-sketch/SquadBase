SQUAD BASE 4.5.5 – Gruppenanrufe und TURN
=========================================

NEU
- Gruppenanrufe bis 5 Personen (Mesh: jeder verbindet sich direkt mit jedem), Einladen im laufenden Anruf.
- Kachel pro Person mit eigenem Lautstärkeregler; bei einer Person bleibt der bekannte Regler.
- TURN-Unterstützung im Cloud-Server (Cloudflare TURN, eigener coturn mit Shared Secret, statisch wie bisher).
- Die App holt TURN-Zugangsdaten vor jedem Anruf frisch ab. „Verbindung testen“ unter
  „Benachrichtigungen & Mikrofontest“ prüft, ob der Relay-Server erreichbar ist.

DATEIEN
- web/calls.js (neu): komplette Anruf-Engine. web/friends.js reicht nur noch Zustand und Signale weiter.
- web/v455.css (neu), web/index.html (Kacheln, Einladen-Zeile, Testknopf).
- main.go: EINE Zeile. Statische Dateien kommen aus einer festen Namensliste; calls.js und v455.css stehen jetzt darin.
  Tests/assets_check.py findet solche Lücken künftig automatisch.
- cloud-server/src/worker.mjs: /api/ice (TURN), Version 4.5.5. Keine D1-Migration.

SIGNALISIERUNG (ohne Server-Änderung für Gruppenanrufe)
Alles läuft über die vorhandenen /api/signals-Endpunkte; Zusatzinfos stehen im frei formbaren "data"-Feld.
  Einladung  offer   data.squadCall = {id, members, join:false}   klingelt beim Empfänger
  Beitritt   offer   data.squadCall = {id, members, join:true}    wird nur angenommen, wenn die id zum eigenen Anruf passt
  Antwort    answer  data.squadCall = {id, members}
  Roster     ice     data = {squadCtl:'roster', id, members, fresh} nur an Teilnehmer, die squadCall kennen
  Auflegen   hangup / busy tragen die Anruf-ID; Signale aus einem früheren Anruf werden ignoriert.
Die alphabetisch größere Person baut die Verbindung zur kleineren auf, dadurch gibt es nie zwei Angebote für dasselbe
Paar. Die Teilnehmerliste verbreitet sich per Roster-Nachricht. Alles von anderen Clients wird geprüft
(Benutzernamen, ID-Format, höchstens 8 Namen). Ältere Clients (4.5.4) ignorieren die Zusatzfelder, funktionieren
1:1 weiter und werden bei einer Gruppeneinladung sauber abgelehnt.

BEHOBENE FEHLER (vom Mehrfenster-Test gefunden, Punkte 2 bis 4 gab es schon in 4.5.4)
1. Wiedereintritt innerhalb von 60 Sekunden wurde von einigen Teilnehmern ignoriert.
2. Ein verspätetes „aufgelegt“ aus einem früheren Anruf konnte einen neuen Anruf mit derselben Person beenden.
3. Ton und Bild einer Person landeten in getrennten Streams; je nach Reihenfolge fehlte Ton oder Bild.
4. Schaltete der Angerufene die Kamera erst nach der Annahme ein, kam sie beim Anrufer nie an
   (eigener Video-Kanal vor dem Angebot, den Chromium nicht zuordnet).
5. Der Client sprang nach jeder Abfrage auf „latest“ vor; ein zwischenzeitlich eingetroffenes Signal ging dabei
   verloren. Ohne Sprung gibt es diesen Verlust nicht. (Beim Lesen des Codes gefunden, kein eigener Test.)
6. Ein TURN, das der Server nicht mehr anbietet, blieb im Client gespeichert; jetzt gilt immer die aktuelle Antwort.

VERHALTEN, DAS SICH GEÄNDERT HAT
- Ein eingehender Anruf klingelt höchstens 60 Sekunden; eine Einladung wartet höchstens 60 Sekunden auf Antwort.
- Das Element #friend-video gibt es nicht mehr (Kacheln pro Person).
- Maximal 5 Personen. Teilnehmer müssen untereinander befreundet sein, sonst gibt es einen Hinweis.

GRENZEN
- Mesh: Jeder sendet an alle anderen. Bei 4 bis 5 Personen mit Kamera wird die Upload-Leitung stark belastet.
- Nicht automatisch getestet: echtes Netz mit NAT (dafür gibt es den Verbindungstest und TURN), Windows mit echten
  Geräten, Bildschirmfreigabe und Mikrofonwechsel in der Gruppe.

TESTS (aus dem Paket-Ordner starten; Playwright + Chromium bzw. Node nötig)
  python3 Tests/mesh_v455.py      Gruppenanrufe mit echtem WebRTC in 5 Browserfenstern, ca. 90 Sekunden
  python3 Tests/turn_v455.py      TURN im Client
  python3 Tests/assets_check.py   Go-Dateiliste gegen index.html
  python3 Tests/webrtc_v454.py    bisheriger 1:1-Test (Pfade ggf. an deinen Chromium anpassen)
  cd Quellcode/cloud-server && node --test test/*.test.mjs     Server-Tests (13)
  Go-Tests laufen wie bisher über BUILD_WINDOWS.ps1.
