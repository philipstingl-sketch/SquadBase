//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var clickerUser32 = syscall.NewLazyDLL("user32.dll")
var clickerKeyState = clickerUser32.NewProc("GetAsyncKeyState")
var clickerSendInput = clickerUser32.NewProc("SendInput")

func clickerNativeAvailable() bool { return true }
func clickerKeyDown(k int) bool    { r, _, _ := clickerKeyState.Call(uintptr(k)); return r&0x8000 != 0 }
func clickerHotkeyPressed(h string) bool {
	if len(h) == 2 && h[0] == 'F' && h[1] >= '6' && h[1] <= '9' {
		return clickerKeyDown(0x75 + int(h[1]-'6'))
	}
	if h == "F10" {
		return clickerKeyDown(0x79)
	}
	if h == "F11" {
		return clickerKeyDown(0x7A)
	}
	if h == "F12" {
		return clickerKeyDown(0x7B)
	}
	if len(h) == 10 && h[:9] == "CTRL+ALT+" {
		return clickerKeyDown(0x11) && clickerKeyDown(0x12) && clickerKeyDown(int(h[9]))
	}
	return false
}
func clickerEmergencyStop() bool {
	return clickerKeyDown(0x11) && clickerKeyDown(0x10) && clickerKeyDown(0x7B)
} // Ctrl+Shift+F12
// x64 INPUT contains an 8-byte aligned MOUSEINPUT union.
type clickInput struct {
	Type      uint32
	Pad       uint32
	Dx        int32
	Dy        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	Pad2      uint32
	ExtraInfo uintptr
}

func clickerEmit(button string) bool {
	var down, up uint32 = 0x0002, 0x0004
	if button == "right" {
		down, up = 0x0008, 0x0010
	}
	clicks := [2]clickInput{{Flags: down}, {Flags: up}}
	r, _, _ := clickerSendInput.Call(2, uintptr(unsafe.Pointer(&clicks[0])), unsafe.Sizeof(clicks[0]))
	return r == 2
}
