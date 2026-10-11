//go:build !windows

package main

import "errors"

func overlaySupported() bool { return false }
func showOverlayWindow(_ string, _ OverlayConfig) error {
	return errors.New("Overlay ist nur unter Windows verfügbar")
}
func hideOverlayWindow()                 {}
func applyOverlayWindow(_ OverlayConfig) {}
func startOverlayHotkey(_ func())        {}
