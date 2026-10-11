# SQUAD BASE 4.5.3 – kostenloser Freunde-Cloud-Server

Dieser Ordner enthält **fertigen Cloudflare-Workers-Code**, SQL-Migrationen und automatisierte Tests. Die Anmeldung in deinem Cloudflare-Konto und die Veröffentlichung musst du selbst bestätigen. Bis dahin gibt es **noch keine öffentliche Serveradresse**.

## Voraussetzungen

- Ein eigenes Cloudflare-Konto mit Workers und D1 im Free-Tarif.
- Node.js LTS + npm (https://nodejs.org/).
- Ein Windows-PC mit PowerShell, alternativ ein anderes Terminal.
- SQUAD BASE **v4.5.3 Cloud Friends** auf allen beteiligten PCs.

Die grundlegenden Funktionen (Konten, Freunde, Chats, Anrufe-Signalisierung) brauchen keine kostenpflichtige VM. Freie Angebote haben Tages-/Speicherlimits und können sich ändern. Der Workers-Free-Tarif hat ein enges CPU-Limit pro Anfrage; die tatsächliche Cloud-Performance insbesondere bei der Passwortprüfung muss nach dem Deployment getestet werden.

## Installation: 7 Schritte

Öffne Windows PowerShell **im Ordner `cloud-server`**, nicht im Quellcode-Ordner der Windows-App:

**1.** Cloudflare-Werkzeug installieren (benötigt Internet):

```powershell
npm install
```

**2.** Bei Cloudflare anmelden:

```powershell
npx wrangler login
```

**3.** D1-Datenbank anlegen:

```powershell
npx wrangler d1 create squadbase-friends-db
```

Kopiere die ausgegebene `database_id` (UUID). In der Datei `wrangler.toml` ersetze **genau** `REPLACE_WITH_YOUR_D1_DATABASE_ID` durch diese ID. Wenn Wrangler anbietet, die Datenbank noch einmal automatisch in die Konfiguration einzutragen, wähle **Nein** – die Bindung `DB` steht bereits in `wrangler.toml`.

**4.** Tabellen für Konten und Chatnachrichten anlegen:

```powershell
npx wrangler d1 migrations apply squadbase-friends-db --remote
```

**5.** Privaten Einladungscode einstellen:

```powershell
npx wrangler secret put INVITE_CODE
```

Wähle einen **zufälligen, langen Code mit mindestens 16 Zeichen**. Er wird in Cloudflare gespeichert, nicht im Windows-Programm. Teile ihn nur deinen vertrauenswürdigen Freunden mit. **Nicht auf GitHub, die Download-Website oder in Screenshots hochladen.** Ohne diesen Code lässt der Server keine Kontoregistrierungen zu.

**6.** Cloud-Server veröffentlichen:

```powershell
npx wrangler deploy
```

Cloudflare gibt eine HTTPS-Adresse aus, die gewöhnlich mit `.workers.dev` endet. **Diese Adresse wird bei jedem Freund im SQUAD-BASE-Freunde-Reiter eingetragen.** Teste im Browser `DEINE_ADRESSE/api/health` – dort sollte `"status":"ok"` erscheinen.

**7.** Starte `SquadBase_v4.5.3_Cloud_Friends.exe` oder den Installer, öffne **Freunde → Serveradresse → Verbinden**, gib die genaue HTTPS-Adresse ein, erstelle mit dem Einladungscode einen Account. Danach kannst du Freunde über den Benutzernamen hinzufügen. Die Cloud-Adresse wird im Windows-Konfigurationsordner gespeichert und bleibt nach Updates erhalten.

Die beiden Benutzerkonten müssen am **gleichen Cloud-Server** registriert sein. Die alte lokale Serverdatei `users_and_messages.json` wird nicht automatisch in D1 übernommen. Alte Chatverläufe bleiben in deiner lokalen Serverinstallation und sollten privat gesichert werden.

## Optional: Dateianhänge mit R2

Wenn du die Dateiübertragung (bis 12 MB pro Datei) nutzen möchtest, muss **R2 separat aktiviert** werden. Cloudflare verlangt dafür unter Umständen eine Zahlungsmethode/ein R2-Abonnement, auch wenn es ein kostenloses monatliches Kontingent gibt. Die App zeigt ohne R2 verständlich an, dass Anhänge deaktiviert sind; Textchat und Anrufe-Signalisierung funktionieren weiter.

Nach Aktivierung:

```powershell
npx wrangler r2 bucket create squadbase-friends-files
```

Entferne in `wrangler.toml` die Kommentarzeichen vor:

```toml
[[r2_buckets]]
binding = "FILES"
bucket_name = "squadbase-friends-files"
```

Danach `npx wrangler deploy` erneut ausführen. Der `GET /api/health`-Endpunkt zeigt `capabilities.files: true`.

## Optional: Bessere Anrufe mit TURN

Sprach- und Videoanrufe laufen per WebRTC direkt zwischen den PCs. Die Cloud vermittelt nur die Signale. Strenge Router und Firewalls (zum Beispiel Mobilfunk, Firmen- oder Uninetze) blockieren die direkte Verbindung. Dann hilft nur ein **TURN-Server**, der den Datenverkehr weiterleitet. Bei **Gruppenanrufen** ist das wichtiger, weil jede einzelne Verbindung zwischen zwei Personen klappen muss. Ohne TURN sind Anrufe nicht in allen Netzwerken garantiert.

Der Server wertet die Einstellungen in dieser Reihenfolge aus. Die App holt die Zugangsdaten vor **jedem Anruf** frisch ab.

### Variante A (empfohlen): Cloudflare TURN

1. Im Cloudflare-Dashboard unter **Realtime → TURN** einen **TURN-Key** erstellen (je nach Dashboard-Version heißt der Bereich noch „Calls“). Notiere die **Key ID** und den **API-Token** und bewahre sie sicher auf. Falls die Menünamen abweichen, hilft die Cloudflare-Anleitung: https://developers.cloudflare.com/realtime/turn/generate-credentials/
2. Im Ordner `cloud-server` ausführen:

```powershell
npx wrangler secret put TURN_KEY_ID
npx wrangler secret put TURN_KEY_API_TOKEN
npx wrangler deploy
```

3. In der App unter **Benachrichtigungen & Mikrofontest → 🌐 Verbindung testen** prüfen. Erwartet wird: „✓ Relay-Server (TURN) erreichbar“.

Kosten (Stand der Prüfung, Oktober 2026, bitte aktuell nachlesen unter https://developers.cloudflare.com/realtime/sfu/pricing/): 0,05 US-Dollar pro GB ausgehender Daten, die ersten 1.000 GB pro Monat sind frei. Nur Anrufe, die nicht direkt zustande kommen, laufen überhaupt über TURN. Grobe Schätzung (nicht gemessen): ein Sprachanruf braucht je Verbindung etwa 15 bis 25 MB pro Stunde, Video ein Vielfaches davon.

Optional: `TURN_TTL` (Sekunden, 3600 bis 172800, Standard 86400) legt fest, wie lange die Zugangsdaten gültig sind. Der Wert sollte länger sein als dein längster Anruf.

### Variante B: eigener coturn-Server

coturn mit `use-auth-secret` und `static-auth-secret=<geheimnis>` betreiben. Dann hinterlegen:

```powershell
npx wrangler secret put TURN_URL      # z. B. turn:turn.deine-domain.de:3478?transport=udp,turns:turn.deine-domain.de:5349
npx wrangler secret put TURN_SECRET   # dasselbe Geheimnis wie static-auth-secret
npx wrangler deploy
```

Jeder angemeldete Benutzer bekommt eigene, befristete Zugangsdaten (Benutzername `Ablaufzeit:name`, Passwort per HMAC-SHA1 – das übliche coturn-Verfahren).

### Variante C: feste Zugangsdaten (wie bisher)

`TURN_URL`, `TURN_USER` und `TURN_PASSWORD` einzeln als Secrets setzen. Nachteil: Die Zugangsdaten sind dauerhaft gültig, und jeder angemeldete Benutzer kann sie auslesen.

### Verhalten bei Problemen

- Fällt der TURN-Dienst aus oder sind die Zugangsdaten falsch, liefert der Server weiter nur STUN (`GET /api/ice` meldet dann `turnMode: "error"`). Anrufe laufen dann nur direkt. Der Fehler wird 30 Sekunden gemerkt, damit die Cloudflare-API nicht bei jedem Anruf angefragt wird.
- Der API-Token und der Key bleiben auf dem Server und gelangen nie in die App.
- Port 53 filtert der Server heraus, weil Browser ihn blockieren.
- Für Gruppenanrufe ist kein Server-Update nötig. Das Update ist nur für TURN erforderlich. Es gibt keine neue D1-Migration.

## Backups und Datenschutz

- Chats, Kontakte und Konten liegen in D1 und bleiben normalerweise bei Worker-Neustarts und Updates erhalten – **innerhalb der Cloudflare-Speicherlimits**.
- Chats sind **nicht Ende-zu-Ende-verschlüsselt**. Personen mit Datenbank-Adminzugriff können die gespeicherten Nachrichten einsehen. Nutze nur einen Server von Personen, denen du vertraust.
- Für private Backups: `npx wrangler d1 export squadbase-friends-db --remote --output=./squadbase-private-backup.sql`. Diese Datei enthält Nachrichten und Passwort-Hashes: **niemals veröffentlichen oder in ein öffentliches GitHub-Repository hochladen**.
- Backup der optionalen R2-Dateien ist separat nötig. D1-Export umfasst die R2-Dateiinhalte nicht.
- Im Standardmodus werden alte Anrufsignale und abgelaufene Sitzungen täglich gelöscht; Chatnachrichten werden dadurch nicht gelöscht.
- Cloudflare hat Freikontingent-Grenzen, etwa für Anfragen und Datenbank-Speicherung; bei Erreichen kann der Dienst vorübergehend nicht funktionieren. Kein garantierter 24/7-Service.
- Die aktuelle Lösung ist für einen **kleinen privaten Freundeskreis** gebaut, nicht als öffentliche Chatplattform für Tausende Personen.

## Entwicklung und Tests

```powershell
npm test
```

Die Tests verwenden das lokale SQLite-Test-Doppel (`node:sqlite` unter Node.js 22+) und simulieren Cloudflare D1/R2. Ein echter End-to-End-Test in Cloudflare ist erst nach deinem Deployment möglich. `wrangler dev` kann die Cloudflare-Laufzeit lokal bereitstellen, sobald die npm-Abhängigkeiten installiert sind. Für produktiven Betrieb bitte die Anmeldung und das Cloudflare-Kontingent überprüfen.

## Serveradresse ändern

Wenn sich deine Adresse ändert: SQUAD BASE → Freunde → Abmelden → neue Serveradresse eingeben → Verbinden. Die Anmeldung gilt nur für den jeweils verbundenen Server. Alte Freunde-Server bleiben unabhängig.


## UPDATE bestehender v4.5.1-Cloud-Server
Beim Update nur src/worker.mjs und migrations/0002_chat_privacy.sql austauschen. wrangler.toml beibehalten! Dann `npx.cmd wrangler d1 migrations apply squadbase-friends-db --remote` und `npx.cmd wrangler deploy`. Achtung: PBKDF2 100000 ist erforderlich, um bestehende 4.5.1-Cloud-Konten zu verwenden.
