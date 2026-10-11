//go:build windows

package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// startExecutableReplacement launches a tiny PowerShell script that waits for
// the current process to exit, installs the already checksum-verified EXE, then
// relaunches it. The script does not download or execute any code from the web.
func startExecutableReplacement(targetPath, downloadedPath string, currentPID int) error {
	psQuote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$target = %s
$download = %s
$targetPid = %d
$backup = $null
$staged = $target + '.squadbase-new'
try {
  $deadline = (Get-Date).AddSeconds(90)
  while (Get-Process -Id $targetPid -ErrorAction SilentlyContinue) {
    if ((Get-Date) -gt $deadline) { throw 'SquadBase wurde nicht rechtzeitig geschlossen.' }
    Start-Sleep -Milliseconds 250
  }
  if (!(Test-Path -LiteralPath $download -PathType Leaf)) { throw 'Die geprüfte Update-Datei fehlt.' }
  Copy-Item -LiteralPath $download -Destination $staged -Force
  if ((Get-Item -LiteralPath $staged).Length -lt 524288) { throw 'Die Update-Datei ist unerwartet klein.' }
  if (Test-Path -LiteralPath $target -PathType Leaf) {
    $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
    $backup = $target + '.previous-' + $stamp
    Copy-Item -LiteralPath $target -Destination $backup -Force
  }
  Move-Item -LiteralPath $staged -Destination $target -Force
  Start-Process -FilePath $target
  Remove-Item -LiteralPath $download -Force -ErrorAction SilentlyContinue
} catch {
  if ($backup -and (Test-Path -LiteralPath $backup -PathType Leaf)) {
    try { Copy-Item -LiteralPath $backup -Destination $target -Force } catch { }
  }
  try { Add-Content -LiteralPath (Join-Path $env:TEMP 'SquadBase-update.log') -Value (Get-Date -Format o) -ErrorAction SilentlyContinue; Add-Content -LiteralPath (Join-Path $env:TEMP 'SquadBase-update.log') -Value $_ -ErrorAction SilentlyContinue } catch { }
  exit 1
}
`, psQuote(targetPath), psQuote(downloadedPath), currentPID)
	// PowerShell -EncodedCommand expects UTF-16LE, represented as base64.
	utf16 := make([]byte, 0, len(script)*2)
	for _, r := range script {
		if r <= 0xFFFF {
			b := make([]byte, 2)
			binary.LittleEndian.PutUint16(b, uint16(r))
			utf16 = append(utf16, b...)
		} else {
			// File paths and this script should be BMP text, but encode surrogate
			// pairs correctly if a path contains a supplementary-plane character.
			r -= 0x10000
			for _, unit := range []uint16{uint16(0xD800 + (r >> 10)), uint16(0xDC00 + (r & 0x3FF))} {
				b := make([]byte, 2)
				binary.LittleEndian.PutUint16(b, unit)
				utf16 = append(utf16, b...)
			}
		}
	}
	encoded := base64.StdEncoding.EncodeToString(utf16)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-EncodedCommand", encoded)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("PowerShell-Updatehelfer konnte nicht gestartet werden")
	}
	return nil
}
