//go:build windows

package main

import (
	"fmt"
	"runtime"
	"syscall"
)

// Windows multimedia keys control the current system media session.  This
// does not require Spotify OAuth and deliberately does not claim to read
// Spotify playlists, metadata, accounts, or device state.
var mediaUser32 = syscall.NewLazyDLL("user32.dll")
var mediaKeyEvent = mediaUser32.NewProc("keybd_event")

func desktopMediaAction(action string) error {
	var key uintptr
	switch action {
	case "toggle":
		key = 0xB3 // VK_MEDIA_PLAY_PAUSE
	case "next":
		key = 0xB0 // VK_MEDIA_NEXT_TRACK
	case "previous":
		key = 0xB1 // VK_MEDIA_PREV_TRACK
	default:
		return fmt.Errorf("Unbekannte Medienaktion")
	}
	if err := mediaKeyEvent.Find(); err != nil {
		return fmt.Errorf("Windows-Mediensteuerung nicht verfügbar: %w", err)
	}
	// keybd_event is an OS multimedia key event. It works with Spotify Desktop
	// and other Windows media sessions, and may control another active player.
	mediaKeyEvent.Call(key, 0, 0, 0)
	mediaKeyEvent.Call(key, 0, 0x0002, 0) // KEYEVENTF_KEYUP
	runtime.KeepAlive(mediaKeyEvent)
	return nil
}

func desktopMediaAvailable() bool { return mediaKeyEvent.Find() == nil }
