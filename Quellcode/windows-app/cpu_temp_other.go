//go:build !windows

package main

func optionalCPUTemp() *int { return nil }
