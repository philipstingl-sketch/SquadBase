SQUAD BASE v4.2.1 SPOTIFY FIX
============================
Based on v4.2 Cozy Studio, preserving all current features and per-user settings.

Changes:
- Native Windows media keys (VK_MEDIA_PLAY_PAUSE/NEXT_TRACK/PREV_TRACK) for
  controlling an existing Spotify desktop session without a Spotify Developer ID.
  The events can control another currently active media player. The app never
  fabricates current-song metadata from this Windows fallback.
- Spotify opening through registered spotify: URI with HTTPS fallback.
- Mini-player accessible without OAuth; metadata/cover/playlists remain API-only.
- Better explanation of developer API restrictions / Spotify 403 errors.
- Correct refresh and single retry on Spotify API 401, serialized token refresh.
- API tokens remain only in per-user config, not returned to frontend.

Testing: go test ./..., go test -race ./..., go vet ./..., Chromium mocked UI
flow (both desktop fallback and authenticated API), JavaScript syntax, Windows
x64 GUI cross-build. NOT live-tested with user's Spotify account or on Windows.
The unsigned EXE may be blocked by Windows Smart App Control.

Official Spotify docs:
https://developer.spotify.com/documentation/web-api/concepts/redirect_uri
https://developer.spotify.com/documentation/web-api/concepts/quota-modes

Build: PowerShell BUILD_WINDOWS.ps1
