//go:build windows

package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed payload/SquadBase.exe payload/SquadBaseUninstall.exe
var payload embed.FS
var msgProc = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")

func show(text string, flags uintptr) uintptr {
	p, _ := syscall.UTF16PtrFromString(text)
	title, _ := syscall.UTF16PtrFromString("SQUAD BASE 4.7.0 · Installer")
	r, _, _ := msgProc.Call(0, uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(title)), flags)
	return r
}
func saveReplace(dir, file string, data []byte) error {
	path := filepath.Join(dir, file)
	f, e := os.CreateTemp(dir, ".squad-new-*.exe")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(data); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	old := path + ".before-update"
	_ = os.Remove(old)
	if _, e = os.Stat(path); e == nil {
		if e = os.Rename(path, old); e != nil {
			return fmt.Errorf("Alte Version ist wahrscheinlich noch geöffnet: %w", e)
		}
	}
	if e = os.Rename(tmp, path); e != nil {
		_ = os.Rename(old, path)
		return e
	}
	_ = os.Remove(old)
	return nil
}
func psQuoted(p string) string { return "'" + strings.ReplaceAll(p, "'", "''") + "'" }
func makeShortcut(target, lnk string) error {
	script := "$shell=New-Object -ComObject WScript.Shell; $shortcut=$shell.CreateShortcut(" + psQuoted(lnk) + "); $shortcut.TargetPath=" + psQuoted(target) + "; $shortcut.WorkingDirectory=" + psQuoted(filepath.Dir(target)) + "; $shortcut.Description='SQUAD BASE 4.7.0'; $shortcut.Save()"
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("Startmenü-Verknüpfung: %w (%s)", e, string(out))
	}
	return nil
}
func registry(name, kind, value string) error {
	return exec.Command("reg.exe", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\SquadBase`, "/v", name, "/t", kind, "/d", value, "/f").Run()
}
func install() error {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return fmt.Errorf("LOCALAPPDATA fehlt")
	}
	dir := filepath.Join(base, "Programs", "SquadBase")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	app, e := payload.ReadFile("payload/SquadBase.exe")
	if e != nil {
		return e
	}
	if e = saveReplace(dir, "SquadBase.exe", app); e != nil {
		return e
	}
	uninstall, e := payload.ReadFile("payload/SquadBaseUninstall.exe")
	if e != nil {
		return e
	}
	if e = saveReplace(dir, "SquadBaseUninstall.exe", uninstall); e != nil {
		return e
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return fmt.Errorf("APPDATA fehlt; Startmenü ist nicht erreichbar")
	}
	start := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs")
	if e = os.MkdirAll(start, 0700); e != nil {
		return e
	}
	shortcutError := makeShortcut(filepath.Join(dir, "SquadBase.exe"), filepath.Join(start, "SQUAD BASE.lnk"))
	// Register a per-user uninstall entry in Windows Settings (no admin rights).
	_ = registry("DisplayName", "REG_SZ", "SQUAD BASE")
	_ = registry("DisplayVersion", "REG_SZ", "4.7.0")
	_ = registry("Publisher", "REG_SZ", "SQUAD BASE Community")
	_ = registry("InstallLocation", "REG_SZ", dir)
	_ = registry("UninstallString", "REG_SZ", filepath.Join(dir, "SquadBaseUninstall.exe"))
	_ = registry("NoModify", "REG_DWORD", "1")
	_ = registry("NoRepair", "REG_DWORD", "1")
	if shortcutError != nil {
		return fmt.Errorf("App installiert, aber Startmenü-Verknüpfung fehlgeschlagen: %w", shortcutError)
	}
	return nil
}
func main() {
	if show("SQUAD BASE 4.7.0 für diesen Windows-Benutzer installieren?\n\nInstallationsort: %LOCALAPPDATA%\\Programs\\SquadBase\n\nVorhandene persönliche Einstellungen und Spielstände werden nicht gelöscht.\n\nDie App ist nicht digital signiert.", 0x24) != 6 {
		return
	}
	if e := install(); e != nil {
		show("Installation nicht vollständig: "+e.Error()+"\n\nDie portable EXE im Downloadpaket kann alternativ verwendet werden.", 0x10)
		return
	}
	show("SQUAD BASE 4.7.0 wurde installiert!\n\nÖffne SQUAD BASE über das Startmenü.\n\nDeine bisherigen Einstellungen bleiben erhalten.", 0x40)
}
