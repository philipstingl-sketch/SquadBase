# Kompatibler Einstiegspunkt für ältere Anleitungen.
# Verwendet den zentralen v4.7.0-Build inklusive App, Installer und Uninstaller.
param([switch]$SkipTests)
$ErrorActionPreference = 'Stop'
$buildScript = Join-Path (Split-Path $PSScriptRoot -Parent) 'BUILD_ALL_WINDOWS.ps1'
if (-not (Test-Path -LiteralPath $buildScript -PathType Leaf)) {
    throw "Zentrales Build-Skript nicht gefunden: $buildScript"
}
if ($SkipTests) {
    & $buildScript -Version '4.7.0' -SkipTests
} else {
    & $buildScript -Version '4.7.0'
}
