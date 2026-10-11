//go:build windows

package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var k32 = syscall.NewLazyDLL("kernel32.dll")
var createMutexW = k32.NewProc("CreateMutexW")
var closeMutex = k32.NewProc("CloseHandle")

// Prevent conflicting simultaneous launches (particularly Spotify's fixed callback port).
func acquireInstanceLock() (func(), bool) {
	name, err := syscall.UTF16PtrFromString(`Local\SquadBase_v4_App_Instance`)
	if err != nil {
		return func() {}, false
	}
	h, _, callErr := createMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return func() {}, false
	} // fail open if mutex API itself fails
	if callErr == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		closeMutex.Call(h)
		return func() {}, true
	}
	return func() { closeMutex.Call(h) }, false
}

var sh32 = syscall.NewLazyDLL("shell32.dll")
var u32 = syscall.NewLazyDLL("user32.dll")
var pTimes = k32.NewProc("GetSystemTimes")
var pMem = k32.NewProc("GlobalMemoryStatusEx")
var pOpen = sh32.NewProc("ShellExecuteW")
var pMsg = u32.NewProc("MessageBoxW")
var pForeground = u32.NewProc("GetForegroundWindow")
var comDlg = syscall.NewLazyDLL("comdlg32.dll")
var pOpenFileName = comDlg.NewProc("GetOpenFileNameW")
var pDialogError = comDlg.NewProc("CommDlgExtendedError")

type memStatus struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPage            uint64
	AvailPage            uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var prevIdle, prevKernel, prevUser uint64

func valTime(f syscall.Filetime) uint64 { return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime) }
func readHardware() (float64, float64, float64, float64, *float64, *int, string) {
	var idle, kernel, user syscall.Filetime
	ok, _, _ := pTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
	cpu := 0.0
	if ok != 0 {
		a, b, c := valTime(idle), valTime(kernel), valTime(user)
		if prevKernel != 0 && b >= prevKernel && c >= prevUser && a >= prevIdle {
			dt := (b - prevKernel) + (c - prevUser)
			di := a - prevIdle
			if dt > 0 && di <= dt {
				cpu = 100 * float64(dt-di) / float64(dt)
			}
		}
		prevIdle, prevKernel, prevUser = a, b, c
	}
	var m memStatus
	m.Length = uint32(unsafe.Sizeof(m))
	success, _, _ := pMem.Call(uintptr(unsafe.Pointer(&m)))
	ram, total, used := 0.0, 0.0, 0.0
	if success != 0 && m.TotalPhys > 0 {
		total = float64(m.TotalPhys) / 1073741824
		used = float64(m.TotalPhys-m.AvailPhys) / 1073741824
		ram = 100 * used / total
	}
	gpu, temp, name := queryNvidia()
	return cpu, ram, used, total, gpu, temp, name
}

var cachedExtraVRAMUsed, cachedExtraVRAMTotal, cachedExtraFan *int

func queryNvidia() (*float64, *int, string) {
	cachedExtraVRAMUsed, cachedExtraVRAMTotal, cachedExtraFan = nil, nil, nil
	exe := "nvidia-smi.exe"
	if _, err := exec.LookPath(exe); err != nil {
		p := filepath.Join(os.Getenv("ProgramFiles"), "NVIDIA Corporation", "NVSMI", "nvidia-smi.exe")
		if _, e := os.Stat(p); e != nil {
			return nil, nil, "Keine NVIDIA-Messung verfügbar"
		}
		exe = p
	}
	ctx, c := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer c()
	cmd := exec.CommandContext(ctx, exe, "--query-gpu=utilization.gpu,temperature.gpu,name,memory.used,memory.total,fan.speed", "--format=csv,noheader,nounits")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	data, err := cmd.Output()
	if err != nil {
		fallback := exec.CommandContext(ctx, exe, "--query-gpu=utilization.gpu,temperature.gpu,name", "--format=csv,noheader,nounits")
		fallback.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		data, err = fallback.Output()
		if err != nil {
			return nil, nil, "NVIDIA-Messung nicht verfügbar"
		}
	}
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil || len(rows) < 1 || len(rows[0]) < 3 {
		return nil, nil, "NVIDIA-GPU"
	}
	if len(rows[0]) >= 6 {
		parse := func(t string) *int {
			v, e := strconv.Atoi(strings.TrimSpace(t))
			if e != nil {
				return nil
			}
			return &v
		}
		cachedExtraVRAMUsed = parse(rows[0][3])
		cachedExtraVRAMTotal = parse(rows[0][4])
		cachedExtraFan = parse(rows[0][5])
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(rows[0][0]), 64)
	if err != nil {
		return nil, nil, strings.TrimSpace(rows[0][2])
	}
	t, err := strconv.Atoi(strings.TrimSpace(rows[0][1]))
	if err != nil {
		return &v, nil, strings.TrimSpace(rows[0][2])
	}
	return &v, &t, strings.TrimSpace(rows[0][2])
}
func cpuName() string {
	ctx, c := context.WithTimeout(context.Background(), 3*time.Second)
	defer c()
	cmd := exec.CommandContext(ctx, "reg.exe", "query", `HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0`, "/v", "ProcessorNameString")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	b, e := cmd.Output()
	if e != nil {
		return "Windows-Prozessor"
	}
	for _, line := range strings.Split(string(b), "\n") {
		if n := strings.Index(line, "REG_SZ"); n >= 0 {
			return strings.TrimSpace(line[n+6:])
		}
	}
	return "Windows-Prozessor"
}
func shellOpen(value string) error {
	v, e := syscall.UTF16PtrFromString(value)
	if e != nil {
		return e
	}
	verb, _ := syscall.UTF16PtrFromString("open")
	status, _, _ := pOpen.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(v)), 0, 0, 1)
	if status <= 32 {
		return fmt.Errorf("Windows ShellExecute Fehler %d", status)
	}
	return nil
}
func openFolder(value string) error {
	if _, e := os.Stat(value); e != nil {
		return e
	}
	return shellOpen(value)
}
func openAppWindow(url string) error {
	// Use the same Edge discovery for the main window and the native overlay.
	// This also covers per-user Edge installs under LOCALAPPDATA, unlike the
	// previous Program Files-only discovery.
	if edge := findEdgeExe(); edge != "" {
		cmd := exec.Command(edge, "--app="+url, "--new-window")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	// When Edge is missing or cannot launch, keep the backend usable via the
	// user's default browser rather than silently closing the application.
	return shellOpen(url)
}
func msgBox(s string) {
	ptr, _ := syscall.UTF16PtrFromString(s)
	title, _ := syscall.UTF16PtrFromString("SQUAD BASE")
	pMsg.Call(0, uintptr(unsafe.Pointer(ptr)), uintptr(unsafe.Pointer(title)), 0)
}

// openFileName mirrors the Windows OPENFILENAMEW structure (including its 64-bit padding).
// Using the native Windows file picker avoids launching PowerShell just to select a game.
type openFileName struct {
	StructSize       uint32
	Owner            uintptr
	Instance         uintptr
	Filter           *uint16
	CustomFilter     *uint16
	MaxCustomFilter  uint32
	FilterIndex      uint32
	File             *uint16
	MaxFile          uint32
	FileTitle        *uint16
	MaxFileTitle     uint32
	InitialDir       *uint16
	Title            *uint16
	Flags            uint32
	FileOffset       uint16
	FileExtension    uint16
	DefaultExtension *uint16
	CustomData       uintptr
	Hook             uintptr
	TemplateName     *uint16
	Reserved         uintptr
	ReservedDWORD    uint32
	FlagsEx          uint32
}

func chooseEXE() (string, error) {
	// Dialogs need a single locked thread. Foreground window ownership helps
	// keep the picker visible above SQUAD BASE on multi-monitor Windows setups.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	filter := syscall.StringToUTF16("Anwendungen (*.exe)\x00*.exe\x00Alle Dateien (*.*)\x00*.*\x00\x00")
	title := syscall.StringToUTF16("Spiel-EXE auswählen")
	file := make([]uint16, 32768)
	owner, _, _ := pForeground.Call()
	var o openFileName
	o.StructSize = uint32(unsafe.Sizeof(o))
	o.Owner = owner
	o.Filter = &filter[0]
	o.File = &file[0]
	o.MaxFile = uint32(len(file))
	o.Title = &title[0]
	o.Flags = 0x00080000 | 0x00001000 | 0x00000800 | 0x00000008 | 0x00800000
	result, _, _ := pOpenFileName.Call(uintptr(unsafe.Pointer(&o)))
	if result != 0 {
		return syscall.UTF16ToString(file), nil
	}
	code, _, _ := pDialogError.Call()
	// Zero means the user pressed Cancel. Other codes indicate dialog failure.
	if code == 0 {
		return "", nil
	}
	return "", fmt.Errorf("Windows-Dateidialog Fehlercode %d", code)
}

// Native image picker. Selected files are validated and copied into the user's
// own SquadBase configuration folder; the original file is never modified.
func chooseImage() (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	filter := syscall.StringToUTF16("Bilder (*.png;*.jpg;*.jpeg;*.gif)\x00*.png;*.jpg;*.jpeg;*.gif\x00\x00")
	title := syscall.StringToUTF16("Hintergrundbild wählen")
	file := make([]uint16, 32768)
	owner, _, _ := pForeground.Call()
	o := openFileName{StructSize: uint32(unsafe.Sizeof(openFileName{})), Owner: owner, Filter: &filter[0], File: &file[0], MaxFile: uint32(len(file)), Title: &title[0], Flags: 0x00080000 | 0x00001000 | 0x00000800 | 0x00000008 | 0x00800000}
	ok, _, _ := pOpenFileName.Call(uintptr(unsafe.Pointer(&o)))
	if ok != 0 {
		return syscall.UTF16ToString(file), nil
	}
	code, _, _ := pDialogError.Call()
	if code == 0 {
		return "", nil
	}
	return "", fmt.Errorf("Windows Bildauswahl Fehler %d", code)
}

// Run console utilities without popping up console windows from our GUI process.
func hiddenOutput(ctx context.Context, program string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.CombinedOutput()
}

// --- Optional Windows startup and system-tray support -----------------------
// The tray host has no visible window. It keeps the local server alive while
// the user hides the Edge app window, and can reopen that window on demand.
var (
	advapi32           = syscall.NewLazyDLL("advapi32.dll")
	pRegCreateKeyExW   = advapi32.NewProc("RegCreateKeyExW")
	pRegOpenKeyExW     = advapi32.NewProc("RegOpenKeyExW")
	pRegSetValueExW    = advapi32.NewProc("RegSetValueExW")
	pRegDeleteValueW   = advapi32.NewProc("RegDeleteValueW")
	pRegCloseKey       = advapi32.NewProc("RegCloseKey")
	pGetModuleHandleW  = k32.NewProc("GetModuleHandleW")
	pRegisterClassExW  = u32.NewProc("RegisterClassExW")
	pCreateWindowExW   = u32.NewProc("CreateWindowExW")
	pDefWindowProcW    = u32.NewProc("DefWindowProcW")
	pGetMessageW       = u32.NewProc("GetMessageW")
	pTranslateMessage  = u32.NewProc("TranslateMessage")
	pDispatchMessageW  = u32.NewProc("DispatchMessageW")
	pPostQuitMessage   = u32.NewProc("PostQuitMessage")
	pDestroyWindow     = u32.NewProc("DestroyWindow")
	pShowWindow        = u32.NewProc("ShowWindow")
	pSetForeground     = u32.NewProc("SetForegroundWindow")
	pEnumWindows       = u32.NewProc("EnumWindows")
	pGetWindowTextLenW = u32.NewProc("GetWindowTextLengthW")
	pGetWindowTextW    = u32.NewProc("GetWindowTextW")
	pCreatePopupMenu   = u32.NewProc("CreatePopupMenu")
	pAppendMenuW       = u32.NewProc("AppendMenuW")
	pTrackPopupMenu    = u32.NewProc("TrackPopupMenu")
	pDestroyMenu       = u32.NewProc("DestroyMenu")
	pGetCursorPos      = u32.NewProc("GetCursorPos")
	pPostMessageW      = u32.NewProc("PostMessageW")
	pLoadIconW         = u32.NewProc("LoadIconW")
	pShellNotifyIconW  = sh32.NewProc("Shell_NotifyIconW")
	trayMu             sync.Mutex
	trayHWnd           uintptr
	trayIconAdded      bool
	trayAppURL         string
)

const (
	trayClassName       = "SquadBaseTrayHostWindow"
	trayCallbackMessage = 0x8031 // WM_APP + 49
	trayIconID          = 1
	trayMenuOpen        = 4101
	trayMenuExit        = 4102
	wmClose             = 0x0010
	wmDestroy           = 0x0002
	wmCommand           = 0x0111
	wmLButtonUp         = 0x0202
	wmLButtonDouble     = 0x0203
	wmRButtonUp         = 0x0205
	nimAdd              = 0x00000000
	nimDelete           = 0x00000002
	nifMessage          = 0x00000001
	nifIcon             = 0x00000002
	nifTip              = 0x00000004
	idiApplication      = 32512
	wsPopup             = 0x80000000
	swHide              = 0
	swShow              = 5
	swRestore           = 9
	mfString            = 0x00000000
	mfSeparator         = 0x00000800
	tpmRightButton      = 0x0002
	tpmReturnCommand    = 0x0100
	regCurrentUser      = 0x80000001
	regSz               = 1
	keySetValue         = 0x0002
	keyQueryValue       = 0x0001
	keyWow64_64         = 0x0100
)

type trayWndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSmall  uintptr
}

type trayPoint struct{ X, Y int32 }
type trayMSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   trayPoint
	Private uint32
}

type notifyIconDataW struct {
	Size             uint32
	HWnd             uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GUID             [16]byte
	BalloonIcon      uintptr
}

var trayWndCallback = syscall.NewCallback(trayWndProc)
var enumWindowCallback = syscall.NewCallback(enumSquadBaseWindows)

var enumWindowSearchMu sync.Mutex
var enumWindowFound atomic.Uintptr

func enumSquadBaseWindows(hwnd uintptr, _ uintptr) uintptr {
	length, _, _ := pGetWindowTextLenW.Call(hwnd)
	if length == 0 || length > 1024 {
		return 1
	}
	buf := make([]uint16, int(length)+1)
	pGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	title := strings.ToLower(syscall.UTF16ToString(buf))
	if strings.Contains(title, "squad base") {
		enumWindowFound.Store(hwnd)
		return 0
	}
	return 1
}

func findSquadBaseWindow() uintptr {
	enumWindowSearchMu.Lock()
	defer enumWindowSearchMu.Unlock()
	enumWindowFound.Store(0)
	pEnumWindows.Call(enumWindowCallback, 0)
	return enumWindowFound.Load()
}

func hideAppWindow() error {
	hwnd := findSquadBaseWindow()
	if hwnd == 0 {
		return fmt.Errorf("SQUAD BASE-Fenster wurde nicht gefunden. Öffne das Fenster zuerst normal.")
	}
	pShowWindow.Call(hwnd, swHide)
	return nil
}

func restoreAppWindow(url string) {
	hwnd := findSquadBaseWindow()
	if hwnd != 0 {
		pShowWindow.Call(hwnd, swRestore)
		pShowWindow.Call(hwnd, swShow)
		pSetForeground.Call(hwnd)
		return
	}
	if url != "" {
		_ = openAppWindow(url)
	}
}

func notifyData(hwnd uintptr) notifyIconDataW {
	var data notifyIconDataW
	data.Size = uint32(unsafe.Sizeof(data))
	data.HWnd = hwnd
	data.ID = trayIconID
	data.Flags = nifMessage | nifIcon | nifTip
	data.CallbackMessage = trayCallbackMessage
	icon, _, _ := pLoadIconW.Call(0, uintptr(idiApplication))
	data.Icon = icon
	tip := syscall.StringToUTF16("SQUAD BASE · Gaming Hub")
	copy(data.Tip[:], tip)
	return data
}

func setTrayVisible(visible bool) error {
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayHWnd == 0 {
		if visible {
			return fmt.Errorf("Tray-Fenster ist noch nicht bereit")
		}
		return nil
	}
	if visible == trayIconAdded {
		return nil
	}
	data := notifyData(trayHWnd)
	if visible {
		result, _, _ := pShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
		if result == 0 {
			return fmt.Errorf("Windows konnte das Infobereich-Symbol nicht hinzufügen")
		}
		trayIconAdded = true
		return nil
	}
	pShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	trayIconAdded = false
	return nil
}

func startSystemTray(url string, visible bool) (<-chan struct{}, error) {
	ready := make(chan error, 1)
	exited := make(chan struct{})
	trayMu.Lock()
	trayAppURL = url
	trayMu.Unlock()
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(exited)

		instance, _, _ := pGetModuleHandleW.Call(0)
		className, _ := syscall.UTF16PtrFromString(trayClassName)
		wc := trayWndClassEx{Size: uint32(unsafe.Sizeof(trayWndClassEx{})), WndProc: trayWndCallback, Instance: instance, ClassName: className}
		atom, _, classErr := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		if atom == 0 {
			ready <- fmt.Errorf("Tray-Fensterklasse konnte nicht registriert werden: %v", classErr)
			return
		}
		hwnd, _, createErr := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), 0, wsPopup, 0, 0, 0, 0, 0, 0, instance, 0)
		if hwnd == 0 {
			ready <- fmt.Errorf("Tray-Fenster konnte nicht erstellt werden: %v", createErr)
			return
		}
		trayMu.Lock()
		trayHWnd = hwnd
		trayIconAdded = false
		trayMu.Unlock()
		if visible {
			if err := setTrayVisible(true); err != nil {
				ready <- err
				pDestroyWindow.Call(hwnd)
				return
			}
		}
		ready <- nil
		var message trayMSG
		for {
			ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
			if int32(ret) <= 0 {
				break
			}
			pTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
			pDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
		}
		trayMu.Lock()
		trayHWnd = 0
		trayIconAdded = false
		trayMu.Unlock()
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	return exited, nil
}

func trayWndProc(hwnd uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	switch message {
	case trayCallbackMessage:
		event := uint32(lParam & 0xffff)
		if event == wmLButtonUp || event == wmLButtonDouble {
			trayMu.Lock()
			url := trayAppURL
			trayMu.Unlock()
			restoreAppWindow(url)
			return 0
		}
		if event == wmRButtonUp {
			menu, _, _ := pCreatePopupMenu.Call()
			if menu == 0 {
				return 0
			}
			openText, _ := syscall.UTF16PtrFromString("SQUAD BASE öffnen")
			exitText, _ := syscall.UTF16PtrFromString("SQUAD BASE beenden")
			pAppendMenuW.Call(menu, mfString, trayMenuOpen, uintptr(unsafe.Pointer(openText)))
			pAppendMenuW.Call(menu, mfSeparator, 0, 0)
			pAppendMenuW.Call(menu, mfString, trayMenuExit, uintptr(unsafe.Pointer(exitText)))
			var point trayPoint
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
			pSetForeground.Call(hwnd)
			selected, _, _ := pTrackPopupMenu.Call(menu, tpmReturnCommand|tpmRightButton, uintptr(point.X), uintptr(point.Y), 0, hwnd, 0)
			pDestroyMenu.Call(menu)
			if selected == trayMenuOpen {
				trayMu.Lock()
				url := trayAppURL
				trayMu.Unlock()
				restoreAppWindow(url)
			} else if selected == trayMenuExit {
				pDestroyWindow.Call(hwnd)
			}
			return 0
		}
	case wmClose:
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		trayMu.Lock()
		added := trayIconAdded
		trayIconAdded = false
		trayMu.Unlock()
		if added {
			data := notifyData(hwnd)
			pShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
		}
		pPostQuitMessage.Call(0)
		return 0
	}
	result, _, _ := pDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func stopSystemTray() {
	trayMu.Lock()
	hwnd := trayHWnd
	trayMu.Unlock()
	if hwnd != 0 {
		pPostMessageW.Call(hwnd, wmClose, 0, 0)
	}
}

func configureWindowsAutostart(enabled, minimized bool) error {
	const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
	keyPath, _ := syscall.UTF16PtrFromString(runKey)
	valueName, _ := syscall.UTF16PtrFromString("SquadBase")
	var key uintptr
	if !enabled {
		ret, _, _ := pRegOpenKeyExW.Call(regCurrentUser, uintptr(unsafe.Pointer(keyPath)), 0, keySetValue|keyWow64_64, uintptr(unsafe.Pointer(&key)))
		if ret == 2 || ret == 3 {
			return nil // Run key doesn't exist; desired state is already achieved.
		}
		if ret != 0 {
			return fmt.Errorf("Registry-Zugriff fehlgeschlagen (%d)", ret)
		}
		defer pRegCloseKey.Call(key)
		ret, _, _ = pRegDeleteValueW.Call(key, uintptr(unsafe.Pointer(valueName)))
		if ret == 2 {
			return nil
		}
		if ret != 0 {
			return fmt.Errorf("Autostart konnte nicht entfernt werden (%d)", ret)
		}
		return nil
	}
	disposition := uint32(0)
	ret, _, _ := pRegCreateKeyExW.Call(regCurrentUser, uintptr(unsafe.Pointer(keyPath)), 0, 0, 0, keySetValue|keyWow64_64, 0, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&disposition)))
	if ret != 0 {
		return fmt.Errorf("Autostart-Schlüssel konnte nicht geöffnet werden (%d)", ret)
	}
	defer pRegCloseKey.Call(key)
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("EXE-Pfad ist nicht verfügbar")
	}
	command := `"` + filepath.Clean(exe) + `"`
	if minimized {
		command += " --minimized"
	}
	value, err := syscall.UTF16FromString(command)
	if err != nil {
		return fmt.Errorf("Autostart-Befehl ist ungültig")
	}
	ret, _, _ = pRegSetValueExW.Call(key, uintptr(unsafe.Pointer(valueName)), 0, regSz, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)*2))
	if ret != 0 {
		return fmt.Errorf("Autostart konnte nicht gespeichert werden (%d)", ret)
	}
	return nil
}
