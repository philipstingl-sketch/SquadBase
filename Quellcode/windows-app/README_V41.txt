SQUAD BASE v4.1 – Cozy Workspace (Windows 10/11 x64)
========================================================
Built on v4.0, preserving the original launcher, game library, Cube Rush,
reaction mini-game, hardware sensors, overlays, Spotify, friends/voice,
Discord, news, theme studio, wallpapers, and local AI integrations.

NEW
- Dashboard presets: Gaming, Cozy, Minimal, Social and individually saved.
  Edit mode moves cards, changes widths and hides cards. All changes in
  %AppData%\SquadBase\workspace-v41.json, independently per Windows user.
- Sidebar: only Games / Library / News and a few further tools are grouped.
  Dashboard, Ping, Friends, Arcade, AI, Spotify, Overlay and Settings remain
  directly accessible.
- Spotify miniplayer in a translucent bottom bar, optional floating compact
  player or disabled. Album art, current track, position, previous/play/next,
  seeking and device volume via official Spotify API where Spotify permits it.
  Requires an authorized Spotify account and supported playback device; some
  controls need Premium. No simulated music or playlists.
- Assistant dictation when Microsoft Edge speech recognition is available.
  Explicit microphone click & permission required. Recognition can use online
  recognition services. Offline fallback: Windows dictation via Win+H.
  Text-to-speech uses installed Edge/Windows voices; optional and off by default.
- Local diagnostics, automatic checking for configured official release feeds,
  settings backup and manual restore of the latest local settings backup.
  Restore never includes Spotify auth tokens or private chat messages.
- Simplified full UI with coherent spacing and Liquid Glass 2.0.

INSTALL/USE
- Portable EXE: just launch SquadBase_v4.1_CozyWorkspace.exe.
- Optional per-user start-menu installer is included with the release packet;
  it copies the same EXE and does not attempt to bypass Windows protection.
- Retain your previous functioning version as a fallback.
- Do not share your personal %AppData%\SquadBase folder.

TESTED
- go test ./...; go test -race ./...; go vet ./...
- Windows x64 GUI cross-compilation, PE32+ checked.
- Chromium-based UI tests using stubbed local API: card editor, profile
  selection, grouped navigation, AI voice preferences, Spotify bar/dock,
  settings, JavaScript console.
- Not tested on a real Windows PC. This binary is unsigned and can still be
  blocked by Microsoft Smart App Control; no workaround or bypass is provided.
- Microphone and Spotify live authorization cannot be validated in this sandbox.
