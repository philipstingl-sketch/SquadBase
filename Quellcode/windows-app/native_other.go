//go:build !windows

package main

import (
	"context"
	"errors"
)

func readHardware() (float64, float64, float64, float64, *float64, *int, string) {
	return 0, 0, 0, 0, nil, nil, "Nur Windows"
}
func cpuName() string            { return "Nur Windows" }
func shellOpen(string) error     { return errors.New("Nur Windows") }
func openFolder(string) error    { return errors.New("Nur Windows") }
func openAppWindow(string) error { return errors.New("Nur Windows") }
func msgBox(string)              {}
func chooseEXE() (string, error) { return "", errors.New("Nur Windows") }

func hiddenOutput(ctx context.Context, program string, args ...string) ([]byte, error) {
	return nil, errors.New("Nur Windows")
}

func chooseImage() (string, error) { return "", errors.New("Nur Windows") }

func acquireInstanceLock() (func(), bool) { return func() {}, false }

func configureWindowsAutostart(enabled, minimized bool) error {
	if enabled || minimized {
		return errors.New("Nur Windows")
	}
	return nil
}
func startSystemTray(url string, visible bool) (<-chan struct{}, error) { return nil, nil }
func setTrayVisible(visible bool) error {
	if visible {
		return errors.New("Nur Windows")
	}
	return nil
}
func hideAppWindow() error { return errors.New("Nur Windows") }
func stopSystemTray()      {}
