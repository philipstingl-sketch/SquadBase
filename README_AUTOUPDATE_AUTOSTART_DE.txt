SQUAD BASE v4.7.2 – AUTO-UPDATE UND AUTOSTART
=====================================================

INTEGRATION
-----------
Diese Projektkopie basiert auf dem vollständigen v4.7.0-Quellstand. Die Update-
Installation aus dem bereitgestellten v4.5.5-Paket wurde auf v4.7 portiert;
v4.7-eigene Diagnose-, Versionshinweis- und Komfortfunktionen bleiben erhalten.

APP-UPDATES EINRICHTEN
----------------------
1. Erstelle ein öffentliches GitHub-Repository und lade den Inhalt dieses
   Projektordners hoch. Die App-Quellen liegen unter Quellcode/windows-app.
2. Prüfe den Quellcode vor dem Upload. Repository und Release-Dateien sind dann
   öffentlich sichtbar. Lade keine Zugangsdaten, privaten Chatdaten oder lokalen
   Backups hoch.
3. Erstelle für zukünftige Builds einen Patch-Tag im Format v4.7.3, v4.7.4 usw.
   GitHub Actions baut SquadBase_vX.Y.Z_Friends_Tools.exe und SHA256SUMS.txt und
   hängt beides an das Release. GitHub Actions muss aktiviert sein.
4. Installiere die erste updaterfähige Version manuell. Öffne in SquadBase
   „Updates & Diagnose“, trage dein Repository als owner/repository ein und
   aktiviere die Prüfung beim Start und frühestens alle 12 Stunden.
5. Verfügbare Updates werden angezeigt, aber nie ohne deine Bestätigung
   installiert. Beim Installieren wird ein lokales Einstellungs-Backup erstellt.
   Der Updater prüft Release-Zuordnung, Dateigröße, MZ-Kennung und SHA-256,
   schließt SquadBase, ersetzt die EXE und startet sie erneut.

Der Updater benötigt Schreibrechte im Installationsordner. Bei Installationen
unter einem geschützten Ordner wie „Programme“ kann das Update fehlschlagen.
Nutze dann die Release-Seite oder installiere SquadBase in einem Ordner, in dem
dein Windows-Benutzer Änderungen vornehmen darf.

SPOTIFY FÜR FREUNDE
-------------------
Die Spotify-App-Kennung ist in SquadBase hinterlegt. Freunde müssen keine eigene
Client-ID erstellen: Sie klicken auf „Mit Spotify anmelden“ und stimmen dem
Zugriff auf ihr eigenes Konto zu. OAuth PKCE wird ohne Client-Secret verwendet;
persönliche Tokens bleiben lokal auf dem jeweiligen PC.

Der App-Inhaber muss einmalig im Spotify Developer Dashboard die Redirect-URI
http://127.0.0.1:8888/spotify/callback exakt eintragen und die Spotify-Konten der
Freunde freigeben. Im Developer-Modus sind höchstens fünf Nutzer zulässig; der
App-Inhaber benötigt Spotify Premium. Die Benutzerfreigabe wird nicht durch
SquadBase automatisiert.
WINDOWS-AUTOSTART
-----------------
Der Windows-Autostart ist in v4.7 bereits unter „Windows & Komfort“ integriert.
Dort kannst du „Mit Windows starten“ aktivieren und optional festlegen, dass
SquadBase minimiert im Infobereich startet. Diese vorhandene Einstellung
verwaltet den Autostart-Eintrag und bleibt bei Updates bestehen, solange die EXE
am gleichen Pfad ersetzt wird.

SICHERHEITSHINWEIS
------------------
SHA-256 erkennt beschädigte oder veränderte Dateien, bestätigt aber keine digitale
Herausgeber-Signatur. Nutze nur ein Release-Repository, das du selbst kontrollierst
und dessen Dateien du geprüft hast. Die Update-Installation startet erst nach
deiner Bestätigung in SquadBase.
