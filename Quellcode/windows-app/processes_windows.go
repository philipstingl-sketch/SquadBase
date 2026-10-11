//go:build windows

package main

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

func runningProcessNames() map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tasklist.exe", "/fo", "csv", "/nh")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	raw, e := cmd.Output()
	if e != nil {
		return map[string]bool{}
	}
	return parseTasklistCSV(string(raw))
}
