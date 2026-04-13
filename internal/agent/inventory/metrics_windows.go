//go:build windows

package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// CollectMetrics gathers real-time system performance data on Windows via PowerShell CIM queries.
// Individual subsystem failures are logged but do not fail the overall collection.
func CollectMetrics(ctx context.Context) (*LiveMetrics, error) {
	m := &LiveMetrics{CollectedAt: time.Now()}

	fillCPUWindows(ctx, m)
	fillMemoryWindows(ctx, m)
	fillDiskWindows(ctx, m)
	fillUptimeWindows(ctx, m)
	fillGPUWindows(ctx, m)

	return m, nil
}

// fillGPUWindows populates GPUUsagePct using the GPU Engine performance counter.
func fillGPUWindows(ctx context.Context, m *LiveMetrics) {
	const psCmd = `try { $s=(Get-Counter '\GPU Engine(*engtype_3D*)\Utilization Percentage' -ErrorAction Stop).CounterSamples | Where-Object {$_.CookedValue -gt 0}; if($s){[math]::Min(100,[math]::Round(($s|Measure-Object CookedValue -Sum).Sum))}else{0} } catch { 0 }`
	out, err := runPS(ctx, psCmd)
	if err != nil {
		slog.DebugContext(ctx, "windows metrics: gpu query failed", "err", err)
		return
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	if err != nil {
		return
	}
	m.GPUUsagePct = val
}

// runPS runs a PowerShell command and returns its trimmed stdout output.
func runPS(ctx context.Context, cmd string) (string, error) {
	out, err := exec.CommandContext(ctx,
		"powershell", "-NoProfile", "-NonInteractive", "-Command", cmd,
	).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes.TrimSpace(out))), nil
}

// fillCPUWindows populates CPUUsagePct from Win32_Processor.LoadPercentage.
func fillCPUWindows(ctx context.Context, m *LiveMetrics) {
	const psCmd = `Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average | Select-Object -ExpandProperty Average`
	out, err := runPS(ctx, psCmd)
	if err != nil {
		slog.WarnContext(ctx, "windows metrics: cpu query failed", "err", err)
		return
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	if err != nil {
		slog.WarnContext(ctx, "windows metrics: cpu parse failed", "out", out, "err", err)
		return
	}
	m.CPUUsagePct = val
}

// winMemInfo mirrors the JSON returned by the PowerShell memory query.
type winMemInfo struct {
	TotalVisibleMemorySize uint64 `json:"TotalVisibleMemorySize"`
	FreePhysicalMemory     uint64 `json:"FreePhysicalMemory"`
	TotalVirtualMemorySize uint64 `json:"TotalVirtualMemorySize"`
	FreeVirtualMemory      uint64 `json:"FreeVirtualMemory"`
}

// fillMemoryWindows populates memory fields from Win32_OperatingSystem.
func fillMemoryWindows(ctx context.Context, m *LiveMetrics) {
	const psCmd = `Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize,FreePhysicalMemory,TotalVirtualMemorySize,FreeVirtualMemory | ConvertTo-Json -Compress`
	out, err := runPS(ctx, psCmd)
	if err != nil {
		slog.WarnContext(ctx, "windows metrics: memory query failed", "err", err)
		return
	}
	var info winMemInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		slog.WarnContext(ctx, "windows metrics: memory parse failed", "err", err)
		return
	}
	// CIM returns KB; convert to bytes.
	m.MemoryTotalBytes = info.TotalVisibleMemorySize * 1024
	m.MemoryAvailableBytes = info.FreePhysicalMemory * 1024
	m.MemoryUsedBytes = m.MemoryTotalBytes - m.MemoryAvailableBytes
	if m.MemoryTotalBytes > 0 {
		m.MemoryUsedPct = float64(m.MemoryUsedBytes) / float64(m.MemoryTotalBytes) * 100.0
	}
	// Swap = virtual memory minus physical.
	totalVirtBytes := info.TotalVirtualMemorySize * 1024
	freeVirtBytes := info.FreeVirtualMemory * 1024
	if totalVirtBytes > m.MemoryTotalBytes {
		m.SwapTotalBytes = totalVirtBytes - m.MemoryTotalBytes
		swapFreeBytes := uint64(0)
		if freeVirtBytes > m.MemoryAvailableBytes {
			swapFreeBytes = freeVirtBytes - m.MemoryAvailableBytes
		}
		if m.SwapTotalBytes > swapFreeBytes {
			m.SwapUsedBytes = m.SwapTotalBytes - swapFreeBytes
		}
	}
}

// winDiskInfo mirrors one element of the JSON array returned by the disk query.
type winDiskInfo struct {
	DeviceID  string `json:"DeviceID"`
	Size      uint64 `json:"Size"`
	FreeSpace uint64 `json:"FreeSpace"`
}

// fillDiskWindows populates Filesystems from Win32_LogicalDisk (DriveType=3 = local fixed).
func fillDiskWindows(ctx context.Context, m *LiveMetrics) {
	const psCmd = `Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | Select-Object DeviceID,Size,FreeSpace | ConvertTo-Json -Compress`
	out, err := runPS(ctx, psCmd)
	if err != nil {
		slog.WarnContext(ctx, "windows metrics: disk query failed", "err", err)
		return
	}

	disks, err := parseWinDiskJSON(out)
	if err != nil {
		slog.WarnContext(ctx, "windows metrics: disk parse failed", "err", err)
		return
	}

	for _, d := range disks {
		if d.Size == 0 {
			continue
		}
		used := d.Size - d.FreeSpace
		var usePct float64
		usePct = float64(used) / float64(d.Size) * 100.0
		m.Filesystems = append(m.Filesystems, FSMetric{
			Mount:      d.DeviceID,
			Device:     d.DeviceID,
			FSType:     "ntfs",
			TotalBytes: d.Size,
			UsedBytes:  used,
			AvailBytes: d.FreeSpace,
			UsePct:     usePct,
		})
	}
}

// parseWinDiskJSON handles both a single-object and array response from PowerShell.
func parseWinDiskJSON(out string) ([]winDiskInfo, error) {
	out = strings.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}
	if strings.HasPrefix(out, "[") {
		var disks []winDiskInfo
		if err := json.Unmarshal([]byte(out), &disks); err != nil {
			return nil, err
		}
		return disks, nil
	}
	var disk winDiskInfo
	if err := json.Unmarshal([]byte(out), &disk); err != nil {
		return nil, err
	}
	return []winDiskInfo{disk}, nil
}

// fillUptimeWindows populates UptimeSeconds via [Environment]::TickCount64.
func fillUptimeWindows(ctx context.Context, m *LiveMetrics) {
	const psCmd = `[math]::Floor([Environment]::TickCount64 / 1000)`
	out, err := runPS(ctx, psCmd)
	if err != nil {
		slog.WarnContext(ctx, "windows metrics: uptime query failed", "err", err)
		return
	}
	val, err := strconv.ParseUint(strings.TrimSpace(out), 10, 64)
	if err != nil {
		slog.WarnContext(ctx, "windows metrics: uptime parse failed", "out", out, "err", err)
		return
	}
	m.UptimeSeconds = val
}
