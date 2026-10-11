//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var winUser = syscall.NewLazyDLL("user32.dll")
var enumWin = winUser.NewProc("EnumWindows")
var winText = winUser.NewProc("GetWindowTextW")
var winLength = winUser.NewProc("GetWindowTextLengthW")
var winShow = winUser.NewProc("ShowWindow")
var winIsWindow = winUser.NewProc("IsWindow")
var winPos = winUser.NewProc("SetWindowPos")
var winStyleGet = winUser.NewProc("GetWindowLongPtrW")
var winStyleSet = winUser.NewProc("SetWindowLongPtrW")
var winAlpha = winUser.NewProc("SetLayeredWindowAttributes")
var winRegisterHotkey = winUser.NewProc("RegisterHotKey")
var winUnregisterHotkey = winUser.NewProc("UnregisterHotKey")
var winGetMessage = winUser.NewProc("GetMessageW")
var overlayHandle uintptr

const (
	styleEx       = -20 // GWL_EXSTYLE
	exLayered     = 0x00080000
	exTransparent = 0x00000020
	exToolwindow  = 0x00000080
	lWAAlpha      = 0x2
	showNormal    = 5
	hide          = 0
	topmost       = ^uintptr(0) // HWND_TOPMOST = -1
	setNoActivate = 0x0010
	modControl    = 0x0002
	modShift      = 0x0004
	modNoRepeat   = 0x4000
	wmHotkey      = 0x0312
)

func overlaySupported() bool { return findEdgeExe() != "" }
func findEdgeExe() string {
	for _, root := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
		if root == "" {
			continue
		}
		p := filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
func findOverlayWindow() uintptr {
	var found uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		n, _, _ := winLength.Call(hwnd)
		if n < 10 || n > 1024 {
			return 1
		}
		buf := make([]uint16, n+1)
		winText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if strings.Contains(strings.ToUpper(syscall.UTF16ToString(buf)), "SQUAD BASE OVERLAY") {
			found = hwnd
			return 0
		}
		return 1
	})
	enumWin.Call(cb, 0)
	return found
}
func showOverlayWindow(url string, cfg OverlayConfig) error {
	hwnd := overlayHandle
	if hwnd != 0 {
		ok, _, _ := winIsWindow.Call(hwnd)
		if ok == 0 {
			hwnd = 0
			overlayHandle = 0
		}
	}
	if hwnd == 0 {
		hwnd = findOverlayWindow()
	}
	if hwnd == 0 {
		edge := findEdgeExe()
		if edge == "" {
			return errors.New("Microsoft Edge ist für das Overlay nicht installiert")
		}
		cmd := exec.Command(edge, "--app="+url, "--new-window", "--window-position=18,18", "--window-size=440,300", "--disable-features=msEdgeSidebarV2")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("Overlay-Fenster startet nicht: %w", err)
		}
		for i := 0; i < 70; i++ {
			time.Sleep(100 * time.Millisecond)
			hwnd = findOverlayWindow()
			if hwnd != 0 {
				break
			}
		}
		if hwnd == 0 {
			return errors.New("Overlay-Fenster konnte nicht erkannt werden (Edge-App-Modus)")
		}
	}
	overlayHandle = hwnd
	winShow.Call(hwnd, showNormal)
	applyOverlayWindow(cfg)
	return nil
}
func applyOverlayWindow(cfg OverlayConfig) {
	hwnd := overlayHandle
	if hwnd == 0 {
		return
	}
	idx := int32(styleEx)
	current, _, _ := winStyleGet.Call(hwnd, uintptr(idx))
	flags := current | exLayered | exToolwindow
	if cfg.ClickThrough {
		flags |= exTransparent
	} else {
		flags &^= exTransparent
	}
	winStyleSet.Call(hwnd, uintptr(idx), flags)
	winAlpha.Call(hwnd, 0, uintptr(uint8(cfg.Opacity*255/100)), lWAAlpha)
	width := int32(440 * cfg.Scale / 100)
	visible := 0
	for _, enabled := range []bool{cfg.CPU, cfg.GPU, cfg.RAM, cfg.GPUTemp, cfg.FPS, cfg.Ping} {
		if enabled {
			visible++
		}
	}
	rows := (visible + 2) / 3
	baseHeight := 108 + rows*85
	if cfg.Music {
		baseHeight += 67
	}
	if cfg.Friends {
		baseHeight += 48
	}
	height := int32((baseHeight + 34) * cfg.Scale / 100)
	winPos.Call(hwnd, topmost, uintptr(cfg.X), uintptr(cfg.Y), uintptr(width), uintptr(height), setNoActivate)
}
func hideOverlayWindow() {
	hwnd := overlayHandle
	if hwnd == 0 {
		hwnd = findOverlayWindow()
	}
	if hwnd != 0 {
		winShow.Call(hwnd, hide)
	}
}

type nativeMSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	PtX     int32
	PtY     int32
	Private uint32
}

func startOverlayHotkey(toggle func()) {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		ok, _, _ := winRegisterHotkey.Call(0, 0x5342, modControl|modShift|modNoRepeat, uintptr('O'))
		if ok == 0 {
			return
		}
		defer winUnregisterHotkey.Call(0, 0x5342)
		var msg nativeMSG
		for {
			ret, _, _ := winGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(ret) <= 0 {
				return
			}
			if msg.Message == wmHotkey && msg.WParam == 0x5342 {
				go toggle()
			}
		}
	}()
}
