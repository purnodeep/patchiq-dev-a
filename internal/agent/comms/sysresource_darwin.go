//go:build darwin

package comms

import (
	"context"
	"encoding/binary"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// systemResourceUsage returns OS-level CPU percent, memory used (bytes) and disk used (bytes).
func systemResourceUsage(ctx context.Context) (cpuPct float64, memUsed uint64, diskUsed uint64) {
	cpuPct = readCPUPercent(ctx)
	memUsed = readMemUsed()
	diskUsed = readDiskUsed()
	return
}

// readCPUPercent parses macOS top output to compute CPU usage from idle percentage.
func readCPUPercent(ctx context.Context) float64 {
	out, err := exec.CommandContext(ctx, "top", "-l", "1", "-n", "0", "-s", "0").Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "CPU usage:") {
			// Format: "CPU usage: 12.34% user, 5.67% sys, 81.99% idle"
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "idle" && i > 0 {
					idle, err := strconv.ParseFloat(strings.TrimSuffix(parts[i-1], "%"), 64)
					if err == nil {
						return 100 - idle
					}
				}
			}
		}
	}
	return 0
}

func readMemUsed() uint64 {
	total := syctlUint64("hw.memsize")
	if total == 0 {
		return 0
	}

	free := freeMemFromVMStat()
	if total > free {
		return total - free
	}
	return 0
}

func syctlUint64(name string) uint64 {
	raw, err := syscall.Sysctl(name)
	if err != nil || len(raw) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64([]byte(raw)[:8])
}

// freeMemFromVMStat parses vm_stat output to get free + inactive pages in bytes.
func freeMemFromVMStat() uint64 {
	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0
	}

	var freePages, inactivePages uint64
	pageSize := uint64(16384) // default on Apple Silicon; overridden below

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Mach Virtual Memory Statistics") {
			if idx := strings.Index(line, "page size of "); idx != -1 {
				s := strings.TrimSuffix(strings.TrimSpace(line[idx+len("page size of "):]), " bytes)")
				if v, err := strconv.ParseUint(s, 10, 64); err == nil {
					pageSize = v
				}
			}
			continue
		}
		if strings.HasPrefix(line, "Pages free:") {
			freePages = parseVMStatValue(line)
		} else if strings.HasPrefix(line, "Pages inactive:") {
			inactivePages = parseVMStatValue(line)
		}
	}

	return (freePages + inactivePages) * pageSize
}

func parseVMStatValue(line string) uint64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return 0
	}
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "."))
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
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
