//go:build !windows

package main

func clickerNativeAvailable() bool     { return false }
func clickerHotkeyPressed(string) bool { return false }
func clickerEmergencyStop() bool       { return false }
func clickerEmit(string) bool          { return false }
