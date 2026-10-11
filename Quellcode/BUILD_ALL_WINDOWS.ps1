# SQUAD BASE – baut Programm, Uninstaller und Installer für Windows (64 Bit).
# Aufruf in PowerShell:   .\BUILD_ALL_WINDOWS.ps1
# Ergebnis liegt im Ordner "dist": Friends_Tools.exe (ohne Installation), Installer.exe und SHA256SUMS.txt.
param([string]$Version = '4.7.0', [switch]$SkipTests)
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$dist = Join-Path $root 'dist'
$payload = Join-Path $root 'installer\payload'
New-Item -ItemType Directory -Force -Path $dist, $payload | Out-Null

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw 'Go ist nicht installiert. Installieren mit:  winget install GoLang.Go   (danach PowerShell neu öffnen)'
}

if (-not $SkipTests) {
  Push-Location (Join-Path $root 'windows-app')
  try { go test ./...; if ($LASTEXITCODE -ne 0) { throw 'Die Go-Tests sind fehlgeschlagen. Es wurde nichts gebaut.' } } finally { Pop-Location }
}

$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
function Build-Part([string]$folder, [string]$output) {
  Push-Location (Join-Path $root $folder)
  try {
    go build -trimpath -ldflags '-H windowsgui -s -w' -o $output .
    if ($LASTEXITCODE -ne 0) { throw "Build fehlgeschlagen: $folder" }
  } finally { Pop-Location }
}

# Reihenfolge ist wichtig: Programm und Uninstaller werden in den Installer eingebettet.
Build-Part 'windows-app'  (Join-Path $payload 'SquadBase.exe')
Build-Part 'uninstaller'  (Join-Path $payload 'SquadBaseUninstall.exe')
Build-Part 'installer'    (Join-Path $dist "SquadBase_v${Version}_Installer.exe")
Copy-Item (Join-Path $payload 'SquadBase.exe') (Join-Path $dist "SquadBase_v${Version}_Friends_Tools.exe") -Force

Get-ChildItem $dist -Filter *.exe | ForEach-Object {
  '{0}  {1}' -f (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower(), $_.Name
} | Set-Content (Join-Path $dist 'SHA256SUMS.txt') -Encoding ascii

Write-Host ''
Write-Host "Fertig. Dateien im Ordner: $dist" -ForegroundColor Green
Get-ChildItem $dist | Format-Table Name, @{n='MB';e={[math]::Round($_.Length/1MB,1)}} -AutoSize
Write-Warning 'Die EXE-Dateien sind NICHT signiert: Windows SmartScreen zeigt beim ersten Start eine Warnung (Weitere Informationen > Trotzdem ausführen).'
