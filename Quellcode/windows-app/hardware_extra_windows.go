//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var diskSpaceProc = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

func readExtraHardware() (*int, *int, *int, *float64, *float64) {
	used, total, fan := cachedExtraVRAMUsed, cachedExtraVRAMTotal, cachedExtraFan
	root := os.Getenv("SystemDrive") + `\`
	if root == `\` {
		root = `C:\`
	}
	rootPtr, e := syscall.UTF16PtrFromString(root)
	if e != nil {
		return used, total, fan, nil, nil
	}
	var available, totalBytes, freeBytes uint64
	ok, _, _ := diskSpaceProc.Call(uintptr(unsafe.Pointer(rootPtr)), uintptr(unsafe.Pointer(&available)), uintptr(unsafe.Pointer(&totalBytes)), uintptr(unsafe.Pointer(&freeBytes)))
	if ok == 0 || totalBytes == 0 {
		return used, total, fan, nil, nil
	}
	freeGB := float64(available) / 1073741824.0
	totalGB := float64(totalBytes) / 1073741824.0
	return used, total, fan, &freeGB, &totalGB
}
