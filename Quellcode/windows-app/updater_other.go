//go:build !windows

package main

import "errors"

func startExecutableReplacement(targetPath, downloadedPath string, currentPID int) error {
	return errors.New("automatische EXE-Aktualisierung ist nur unter Windows verfügbar")
}
