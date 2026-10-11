package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type startupDiagnostic struct {
	mu   sync.Mutex
	path string
	file *os.File
}

func newStartupDiagnostic(folder string) *startupDiagnostic {
	p := filepath.Join(folder, "startdiagnose.log")
	// Keep a short log; do not leave infinite historical data or copy credentials.
	if info, err := os.Stat(p); err == nil && info.Size() > 65536 {
		_ = os.Rename(p, p+".old")
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return &startupDiagnostic{path: p}
	}
	return &startupDiagnostic{path: p, file: f}
}
func (d *startupDiagnostic) note(phase string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.file != nil {
		_, _ = fmt.Fprintf(d.file, "%s | %s\n", time.Now().Format(time.RFC3339), phase)
	}
}
func (d *startupDiagnostic) close() {
	d.note("Prozess beendet")
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.file != nil {
		_ = d.file.Close()
		d.file = nil
	}
}
