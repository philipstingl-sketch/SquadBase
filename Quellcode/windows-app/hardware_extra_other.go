//go:build !windows

package main

func readExtraHardware() (*int, *int, *int, *float64, *float64) { return nil, nil, nil, nil, nil }
