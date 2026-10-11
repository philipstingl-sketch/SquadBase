//go:build !windows

package main

import "errors"

func desktopMediaAction(string) error {
	return errors.New("Die Desktop-Musiksteuerung benötigt Windows")
}
func desktopMediaAvailable() bool { return false }
