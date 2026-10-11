SQUAD BASE v4.0 – Liquid Glass 2.0
================================
Windows 10/11, 64-bit. Built directly on the previous v3.6 source.

NEW
- Liquid Glass 2.0: unified cards, theme studio, adjustable blur/radius/tint/outline.
- Optional personal JPG/PNG/GIF background (copied to user config folder; never uploaded).
- Overlay 2.0: compact view, minimal header, presets, position, scale, opacity and widgets.
- Intelligent game library: local Steam appmanifest_*.acf files, Epic .item manifests,
  add personal .exe using the native Windows file picker, favorites and launch.
- Assistant 2.0: optional locally stored conversation; locally generated AI images
  still require a separate Stable Diffusion model, free chat answers require Ollama.
- Automatic release checks once every 12-24 hours after the user configures a valid
  GitHub owner/repo and enables the checkbox; no automatic program downloads/runs.
- Private diagnostics and backup: locally saved files, no account secrets, no passwords.

PRESERVED
Dashboard, Hardware Monitor Pro, Cube Rush, Reaktionstest, NoRisk launcher path,
Spotify, friends/voice/file sharing, ping and speed, Minecraft, news, clock,
soundtrack, original backgrounds.

DATA
By default, settings live in %AppData%\SquadBase on Windows (os.UserConfigDir).
This EXE DOES NOT include its builder's personal paths or account data. If you start
it on the same Windows profile, it reads that profile's existing SQUAD BASE settings.

LIMITATIONS
The GUI launches via the Microsoft Edge App Mode, bundled web resources are served
only to 127.0.0.1. Native overlay works best with borderless/windowed games.
The app build is unsigned and may be blocked by Windows Smart App Control.
No Windows device was available here for end-to-end practical testing; Go and
browser-based UI tests are provided separately.
Steam/Epic autodetection is best effort and requires manifests on the user's PC.
Account/voice communications need a separate, correctly configured Relay server.
Spotify control requires official authorization; image AI requires a local model.

Build from source on a Windows PC:
  powershell -ExecutionPolicy Bypass -File .\BUILD_WINDOWS.ps1
For public distribution, use trusted code signing / Microsoft Store.
