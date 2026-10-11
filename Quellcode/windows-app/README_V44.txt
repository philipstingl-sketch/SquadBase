SQUAD BASE v4.4 – Cozy Control
================================

Windows 10/11, x64. Build: Go 1.22+ (stdlib only; no dependencies).
Application remains the existing Go backend + Microsoft Edge app window,
not an HTML-only replacement. All earlier features and settings remain.

Changes from v4.3:
- Soundboard: up to 12 imported MP3, WAV and OGG clips, up to 8 MiB each;
  play in the desktop app, volume, remove, keyboard Ctrl+Alt+1..9 (only when
  SQUAD BASE has keyboard focus). Sounds are local to Windows user profile.
- Remote: optional temporary WLAN server bound to ONE private IPv4 address;
  disabled on every launch; six-digit expiring code (5 minutes), five
  attempts per minute/IP, a 192-bit session token and an explicit allowlist
  of only media controls, app tab changes and sound play. Remotely accessible
  endpoints do not expose local configuration, Spotify tokens, chat, or game
  paths. Sessions are revoked on disabling. Remote audio keys may control
  another current Windows media app, not necessarily Spotify.
- Liquid Glass 3.0: four extra decorative lighting presets (Aurora, Forest,
  Frost and Ember). Older appearance preferences are not overwritten.
- UI fixes: compact audio clips at narrow widths, cleaner secondary sidebar
  menu and a sessionStorage failure fallback in the phone interface.
- All prior functionality of v4.3 retained: Spotify Music Center, favorites,
  playable playlist songs where supported, desktop overlay, game library,
  AI, dashboard editor, theme automation, Cube Rush and Friends relay.

LOCAL CONFIGURATION
%APPDATA%\SquadBase\ on standard Windows profiles. New sounds: subfolder
soundboard. Existing settings files are never reset or deleted by v4.4.
Keep v4.3 as backup until testing on your own PC.

SECURITY
The WLAN server uses HTTP (unencrypted) and must be used only on a trusted
private WLAN. Do not forward the port or use public/hotel/school Wi-Fi.
When Windows prompts for network access, only grant private networks if
appropriate. Disable the remote immediately when not needed. Do not share
PINs or photos of the pairing panel.

Discord voice chat cannot hear system output automatically. A virtual audio
cable or audio routing needs separate user configuration. No Discord audio
integration is claimed.

LIMITATIONS
No live Windows machine was available in the Linux build environment; actual
network/firewall behavior, overlay game compatibility, audio output, Spotify
OAuth and Smart App Control remain to be tested by the end user.
The unsigned build is not guaranteed to be allowed by Smart App Control.

BUILD
PowerShell: .\BUILD_WINDOWS.ps1
Or run:
  go test ./...
  $env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
  go build -trimpath -ldflags '-H windowsgui -s -w' -o SquadBase_v4.4_Cozy_Control.exe .
For the optional Friends relay:
  go build -trimpath -ldflags '-s -w' -o SquadBaseRelay_v4.4_Windows.exe ./relay
