//go:build linux

package comms

import (
	"context"
	"os"
	"strings"
	"sync"
	"syscall"
)

// systemResourceUsage returns OS-level CPU percent, memory used (bytes) and disk used (bytes) for the root filesystem.
func systemResourceUsage(_ context.Context) (cpuPct float64, memUsed uint64, diskUsed uint64) {
	cpuPct = readCPUPercent()
	memUsed = readMemUsed()
	diskUsed = readDiskUsed()
	return
}

// cpuState holds the previous /proc/stat sample so readCPUPercent can compute
// a delta without sleeping. On the first call it returns 0 (no prior sample).
var cpuState struct {
	mu         sync.Mutex
	prevIdle   uint64
	prevTotal  uint64
	hasReading bool
}

// readCPUPercent computes CPU utilization by diffing against the previous
// /proc/stat sample. Returns 0 on the first call (no baseline yet).
// This avoids the 200ms sleep that previously blocked the heartbeat goroutine.
func readCPUPercent() float64 {
	idle, total := readCPUStat()
	if total == 0 {
		return 0
	}

	cpuState.mu.Lock()
	defer cpuState.mu.Unlock()

	if !cpuState.hasReading {
		cpuState.prevIdle = idle
		cpuState.prevTotal = total
		cpuState.hasReading = true
		return 0
	}

	// Guard against counter wraparound or /proc/stat reset after suspend.
	if total < cpuState.prevTotal {
		cpuState.prevIdle = idle
		cpuState.prevTotal = total
		return 0
	}

	totalDelta := total - cpuState.prevTotal
	idleDelta := idle - cpuState.prevIdle
	cpuState.prevIdle = idle
	cpuState.prevTotal = total

	if totalDelta == 0 {
		return 0
	}
	return float64(totalDelta-idleDelta) / float64(totalDelta) * 100
}

// readCPUStat reads the aggregate cpu line from /proc/stat and returns idle and total ticks.
func readCPUStat() (idle, total uint64) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0, 0
			}
			// fields: cpu user nice system idle iowait irq softirq steal ...
			for i := 1; i < len(fields); i++ {
				v := parseUint(fields[i])
				total += v
				if i == 4 { // idle is the 4th value (index 4 in fields)
					idle = v
				}
			}
			return idle, total
		}
	}
	return 0, 0
}

func parseUint(s string) uint64 {
	var v uint64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + uint64(c-'0')
		}
	}
	return v
}

func readMemUsed() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	var total, available uint64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseMemInfoKB(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			available = parseMemInfoKB(line)
		}
	}
	if total > available {
		return (total - available) * 1024 // kB to bytes
	}
	return 0
}

func parseMemInfoKB(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	var val uint64
	for _, c := range parts[1] {
		if c >= '0' && c <= '9' {
			val = val*10 + uint64(c-'0')
		}
	}
	return val
}

func readDiskUsed() uint64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0
	}
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bfree * uint64(stat.Bsize)
	if totalBytes > freeBytes {
		return totalBytes - freeBytes
	}
	return 0
}
