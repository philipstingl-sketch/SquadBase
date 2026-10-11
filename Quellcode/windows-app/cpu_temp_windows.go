//go:build windows

package main

import (
	"context"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Only reads CPU Package from a genuinely available LibreHardwareMonitor WMI
// provider. ACPI thermal-zone data is deliberately not presented as CPU temp.
var cpuSensorCache struct {
	sync.Mutex
	last  time.Time
	value *int
}

func optionalCPUTemp() *int {
	cpuSensorCache.Lock()
	defer cpuSensorCache.Unlock()
	if time.Since(cpuSensorCache.last) < 30*time.Second {
		return cpuSensorCache.value
	}
	cpuSensorCache.last = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2200*time.Millisecond)
	defer cancel()
	script := `$ErrorActionPreference='Stop'; $x=Get-CimInstance -Namespace 'root/LibreHardwareMonitor' -ClassName Sensor | Where-Object { $_.SensorType -eq 'Temperature' -and ($_.Name -match 'CPU Package|Core Average|CPU Core') } | Select-Object -First 1 -ExpandProperty Value; if($null -ne $x){ [Console]::Write([string]$x) }`
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, e := cmd.Output()
	if e != nil {
		cpuSensorCache.value = nil
		return nil
	}
	raw := strings.TrimSpace(strings.ReplaceAll(string(out), ",", "."))
	f, e := strconv.ParseFloat(raw, 64)
	if e != nil || f < 5 || f > 125 {
		cpuSensorCache.value = nil
		return nil
	}
	val := int(math.Round(f))
	cpuSensorCache.value = &val
	return &val
}
