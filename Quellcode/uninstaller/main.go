//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var user32 = syscall.NewLazyDLL("user32.dll")
var message = user32.NewProc("MessageBoxW")

func notify(msg string, style uintptr) uintptr {
	p, _ := syscall.UTF16PtrFromString(msg)
	title, _ := syscall.UTF16PtrFromString("SQUAD BASE · Deinstallation")
	r, _, _ := message.Call(0, uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(title)), style)
	return r
}
func baseDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return ""
	}
	return filepath.Join(base, "Programs", "SquadBase")
}
func main() {
	dir := baseDir()
	if dir == "" {
		notify("Windows-Installationspfad nicht verfügbar.", 0x10)
		return
	}
	exe, err := os.Executable()
	if err != nil || !strings.EqualFold(filepath.Clean(filepath.Dir(exe)), filepath.Clean(dir)) {
		notify("Bitte über Windows > Installierte Apps deinstallieren.", 0x10)
		return
	}
	if notify("SQUAD BASE 4.7.0 wirklich deinstallieren?\n\nDeine persönlichen Einstellungen, Spotify-Verknüpfungen und Spielstände bleiben im Benutzerprofil erhalten.", 0x24) != 6 {
		return
	}
	target := filepath.Join(dir, "SquadBase.exe")
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		notify("Bitte SQUAD BASE vollständig schließen und erneut deinstallieren.\n"+err.Error(), 0x10)
		return
	}
	start := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "SQUAD BASE.lnk")
	_ = os.Remove(start)
	_ = exec.Command("reg.exe", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\SquadBase`, "/f").Run()
	// Uninstaller stays in the folder until the current process has exited.
	// A delayed cmd cleanup removes the *known* user-local install directory.
	cmd := `ping 127.0.0.1 -n 3 >nul & rmdir /s /q "` + dir + `"`
	if strings.ContainsAny(dir, "\"&|<>^%\r\n") {
		notify("Deinstallation abgeschlossen. Den Installationsordner kannst du manuell entfernen.", 0x40)
		return
	}
	notify("SQUAD BASE wurde deinstalliert. Persönliche Einstellungen bleiben erhalten.", 0x40)
	c := exec.Command("cmd.exe", "/C", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = c.Start()
}
