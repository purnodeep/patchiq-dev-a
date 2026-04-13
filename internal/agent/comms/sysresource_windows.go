//go:build windows

package comms

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

// systemResourceUsage returns OS-level CPU percent, memory used (bytes) and disk used (bytes).
func systemResourceUsage(_ context.Context) (cpuPct float64, memUsed uint64, diskUsed uint64) {
	cpuPct = readCPUPercentWin()
	memUsed = readMemUsedWin()
	diskUsed = readDiskUsedWin()
	return
}

func runPowerShell(command string) ([]byte, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(stdout.Bytes()), nil
}

func readCPUPercentWin() float64 {
	out, err := runPowerShell("(Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average")
	if err != nil {
		slog.Warn("sysresource: cpu query failed", "error", err)
		return 0
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0
	}
	return v
}

func readMemUsedWin() uint64 {
	out, err := runPowerShell("Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize,FreePhysicalMemory | ConvertTo-Json")
	if err != nil {
		slog.Warn("sysresource: memory query failed", "error", err)
		return 0
	}
	var info struct {
		TotalVisibleMemorySize uint64 `json:"TotalVisibleMemorySize"`
		FreePhysicalMemory     uint64 `json:"FreePhysicalMemory"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return 0
	}
	if info.TotalVisibleMemorySize > info.FreePhysicalMemory {
		return (info.TotalVisibleMemorySize - info.FreePhysicalMemory) * 1024 // KB to bytes
	}
	return 0
}

func readDiskUsedWin() uint64 {
	out, err := runPowerShell("Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | Select-Object Size,FreeSpace | ConvertTo-Json")
	if err != nil {
		slog.Warn("sysresource: disk query failed", "error", err)
		return 0
	}

	// PowerShell returns object for single disk, array for multiple
	type diskInfo struct {
		Size      uint64 `json:"Size"`
		FreeSpace uint64 `json:"FreeSpace"`
	}

	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return 0
	}

	var disks []diskInfo
	if out[0] == '[' {
		if err := json.Unmarshal(out, &disks); err != nil {
			return 0
		}
	} else {
		var single diskInfo
		if err := json.Unmarshal(out, &single); err != nil {
			return 0
		}
		disks = []diskInfo{single}
	}

	var totalUsed uint64
	for _, d := range disks {
		if d.Size > d.FreeSpace {
			totalUsed += d.Size - d.FreeSpace
		}
	}
	return totalUsed
}
