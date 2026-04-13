//go:build darwin

package inventory

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent"
)

// CollectMetrics gathers real-time system performance data on macOS using CLI tools.
// Individual subsystem failures are logged but do not fail the overall collection.
func CollectMetrics(ctx context.Context) (*LiveMetrics, error) {
	m := &LiveMetrics{}

	// First sample for rate-based metrics (CPU per-core + network I/O).
	cpuSample1, cpuErr1 := readDarwinCPUTicks()
	netSample1 := readDarwinNetStats(ctx)

	// Single 200ms delay for rate-based metrics (same as Linux).
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("collect metrics: %w", ctx.Err())
	case <-time.After(200 * time.Millisecond):
	}

	// Second sample.
	cpuSample2, cpuErr2 := readDarwinCPUTicks()
	netSample2 := readDarwinNetStats(ctx)

	m.CollectedAt = time.Now()

	// Per-core CPU from Mach host_processor_info.
	if cpuErr1 == nil && cpuErr2 == nil && len(cpuSample1) > 0 && len(cpuSample2) > 0 {
		m.CPUPerCore = calcDarwinPerCoreCPU(cpuSample1, cpuSample2)
		// Compute aggregate from per-core data.
		var totalUsage float64
		for _, c := range m.CPUPerCore {
			totalUsage += c.UsagePct
		}
		if len(m.CPUPerCore) > 0 {
			m.CPUUsagePct = totalUsage / float64(len(m.CPUPerCore))
		}
	} else {
		// Fallback to top for aggregate CPU if Mach API fails.
		fillDarwinCPU(ctx, m)
	}
	fillDarwinLoadAvg(ctx, m)
	fillDarwinMemory(ctx, m)
	fillDarwinSwap(ctx, m)
	m.UptimeSeconds = readDarwinUptime(ctx)
	m.ProcessCount = readDarwinProcessCount(ctx)
	m.DiskIO = readDarwinDiskIO(ctx)
	m.NetworkIO = calcDarwinNetIO(netSample1, netSample2)
	m.Filesystems = readDarwinFilesystems(ctx)
	m.CPUTempCelsius = readDarwinCPUTemp(ctx)
	m.GPUUsagePct = readDarwinGPUUsage(ctx)

	// Per-core CPU frequency (Apple Silicon: powermetrics or lookup table).
	if len(m.CPUPerCore) > 0 {
		modelName := cpuModelName(ctx)
		fillDarwinPerCoreFreq(ctx, m.CPUPerCore, modelName)
	}

	return m, nil
}

// readDarwinGPUUsage reads GPU utilization from IOKit via ioreg IOAccelerator.
func readDarwinGPUUsage(ctx context.Context) float64 {
	out, err := exec.CommandContext(ctx, "ioreg", "-c", "IOAccelerator", "-s0", "-w0").Output()
	if err != nil {
		return 0
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "Device Utilization %") {
			continue
		}
		idx := strings.LastIndex(line, "=")
		if idx < 0 {
			continue
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(line[idx+1:]), 64); err == nil {
			return v
		}
	}
	return 0
}

// fillDarwinCPU runs `top -l 2 -n 0 -s 0` and parses the second CPU usage line.
// Per-core usage requires powermetrics (root access) — known limitation.
// CPU temperature requires IOKit — known limitation.
func fillDarwinCPU(ctx context.Context, m *LiveMetrics) {
	out, err := exec.CommandContext(ctx, "top", "-l", "2", "-n", "0", "-s", "0").Output()
	if err != nil {
		slog.Warn("collect darwin cpu: top command failed", "error", err)
		return
	}
	user, sys, _ := parseDarwinTopCPU(out)
	m.CPUUsagePct = user + sys
}

// fillDarwinLoadAvg runs `sysctl -n vm.loadavg` and parses the output.
func fillDarwinLoadAvg(ctx context.Context, m *LiveMetrics) {
	out, err := exec.CommandContext(ctx, "sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		slog.Warn("collect darwin load avg: sysctl failed", "error", err)
		return
	}
	m.LoadAvg1, m.LoadAvg5, m.LoadAvg15 = parseSysctlLoadAvg(out)
}

// fillDarwinMemory runs `sysctl -n hw.memsize` and `vm_stat` to populate memory metrics.
func fillDarwinMemory(ctx context.Context, m *LiveMetrics) {
	// Total memory from sysctl.
	totalOut, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		slog.Warn("collect darwin memory: sysctl hw.memsize failed", "error", err)
		return
	}
	m.MemoryTotalBytes, _ = strconv.ParseUint(strings.TrimSpace(string(totalOut)), 10, 64)

	// Page statistics from vm_stat.
	vmOut, err := exec.CommandContext(ctx, "vm_stat").Output()
	if err != nil {
		slog.Warn("collect darwin memory: vm_stat failed", "error", err)
		return
	}
	pageSize, pages := parseVmStat(vmOut)
	if pageSize == 0 {
		return
	}

	// parseVmStat (vmstat_darwin.go) keys are lowercase full labels, e.g. "pages free".
	active := pages["pages active"] * pageSize
	inactive := pages["pages inactive"] * pageSize
	wired := pages["pages wired down"] * pageSize
	purgeable := pages["pages purgeable"] * pageSize
	compressor := pages["pages occupied by compressor"] * pageSize

	// macOS memory model (matches Activity Monitor):
	//   App Memory  = active - purgeable (purgeable is reclaimable, part of active)
	//   Wired       = wired (kernel, can't be paged or compressed)
	//   Compressed  = compressor (physical pages holding compressed data)
	//   Used        = app + wired + compressed
	//   Cached      = inactive + purgeable (can be reclaimed on demand)
	//   Available   = total - used
	appMemory := active
	if active > purgeable {
		appMemory = active - purgeable
	}
	m.MemoryUsedBytes = appMemory + wired + compressor
	if m.MemoryTotalBytes > m.MemoryUsedBytes {
		m.MemoryAvailableBytes = m.MemoryTotalBytes - m.MemoryUsedBytes
	}
	m.MemoryCachedBytes = inactive + purgeable
	// On macOS, repurpose MemoryBuffersBytes to carry compressed memory size
	// so the UI can show the macOS-style breakdown.
	m.MemoryBuffersBytes = compressor

	if m.MemoryTotalBytes > 0 {
		m.MemoryUsedPct = float64(m.MemoryUsedBytes) / float64(m.MemoryTotalBytes) * 100.0
	}
}

// fillDarwinSwap runs `sysctl vm.swapusage` and parses swap usage.
func fillDarwinSwap(ctx context.Context, m *LiveMetrics) {
	out, err := exec.CommandContext(ctx, "sysctl", "vm.swapusage").Output()
	if err != nil {
		slog.Warn("collect darwin swap: sysctl vm.swapusage failed", "error", err)
		return
	}
	m.SwapTotalBytes, m.SwapUsedBytes = parseSysctlSwap(out)
}

// readDarwinUptime runs `sysctl -n kern.boottime` and computes uptime.
func readDarwinUptime(ctx context.Context) uint64 {
	out, err := exec.CommandContext(ctx, "sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		slog.Warn("collect darwin uptime: sysctl kern.boottime failed", "error", err)
		return 0
	}
	bootEpoch := parseSysctlBoottime(out)
	if bootEpoch <= 0 {
		return 0
	}
	uptime := time.Now().Unix() - bootEpoch
	if uptime < 0 {
		return 0
	}
	return uint64(uptime)
}

// readDarwinProcessCount runs `ps -ax -o pid=` and counts processes.
func readDarwinProcessCount(ctx context.Context) int {
	out, err := exec.CommandContext(ctx, "ps", "-ax", "-o", "pid=").Output()
	if err != nil {
		slog.Warn("collect darwin process count: ps failed", "error", err)
		return 0
	}
	count := 0
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	return count
}

// readDarwinDiskIO runs `iostat -d -c 2 -w 1` and parses per-disk throughput.
func readDarwinDiskIO(ctx context.Context) []DiskIOMetric {
	out, err := exec.CommandContext(ctx, "iostat", "-d", "-c", "2", "-w", "1").Output()
	if err != nil {
		slog.Warn("collect darwin disk io: iostat failed", "error", err)
		return nil
	}
	return parseIostat(out)
}

// readDarwinFilesystems runs `df -Pk` and `mount` to collect filesystem usage.
// df -Pk uses POSIX format with 1024-byte blocks; mount provides fs type info.
func readDarwinFilesystems(ctx context.Context) []FSMetric {
	dfOut, err := exec.CommandContext(ctx, "df", "-Pk").Output()
	if err != nil {
		slog.Warn("collect darwin filesystems: df -Pk failed", "error", err)
		return nil
	}

	mountOut, err := exec.CommandContext(ctx, "mount").Output()
	if err != nil {
		slog.Warn("collect darwin filesystems: mount failed", "error", err)
		// Proceed without fs type info — df data is still useful.
		mountOut = nil
	}

	mountTypes := parseMountOutput(mountOut)
	return parseDfPk(dfOut, mountTypes)
}

// cpuDieTempRe matches lines like "CPU die temperature: 45.32 C".
var cpuDieTempRe = regexp.MustCompile(`(?i)CPU die temperature:\s+([\d.]+)\s*C`)

// readDarwinCPUTemp reads CPU die temperature via powermetrics when running as root.
// Returns 0 when not root or if the command fails.
func readDarwinCPUTemp(ctx context.Context) float64 {
	if !agent.IsRoot() {
		return 0
	}

	tempCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(tempCtx, "powermetrics", "--samplers", "smc", "-n", "1", "--sample-rate", "1").Output()
	if err != nil {
		slog.Warn("collect darwin cpu temp: powermetrics failed", "error", err)
		return 0
	}

	return parsePowermetricsCPUTemp(out)
}

// parsePowermetricsCPUTemp extracts the CPU die temperature from powermetrics output.
func parsePowermetricsCPUTemp(data []byte) float64 {
	m := cpuDieTempRe.FindSubmatch(data)
	if m == nil {
		return 0
	}
	temp, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		slog.Warn("collect darwin cpu temp: parse temperature value", "raw", string(m[1]), "error", err)
		return 0
	}
	return temp
}

// readDarwinNetStats runs `netstat -ib` and parses interface statistics.
func readDarwinNetStats(ctx context.Context) []darwinNetSample {
	out, err := exec.CommandContext(ctx, "netstat", "-ib").Output()
	if err != nil {
		slog.Warn("collect darwin net stats: netstat failed", "error", err)
		return nil
	}
	return parseNetstatIb(out)
}
