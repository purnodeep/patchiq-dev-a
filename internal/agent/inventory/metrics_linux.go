//go:build linux

package inventory

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// sectorSize is the standard Linux sector size used in /proc/diskstats.
const sectorSize = 512

// CollectMetrics gathers real-time system performance data from /proc and /sys.
// Individual subsystem failures are logged but do not fail the overall collection.
func CollectMetrics(ctx context.Context) (*LiveMetrics, error) {
	m := &LiveMetrics{}

	// --- First sample: CPU, disk, network ---
	cpuSample1 := readProcStat()
	diskSample1 := readDiskStats()
	netSample1 := readNetDev()

	// Single 200ms delay for all rate-based metrics.
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("collect metrics: %w", ctx.Err())
	case <-time.After(200 * time.Millisecond):
	}

	// --- Second sample ---
	cpuSample2 := readProcStat()
	diskSample2 := readDiskStats()
	netSample2 := readNetDev()

	m.CollectedAt = time.Now()

	// CPU usage from deltas.
	calcCPUUsage(m, cpuSample1, cpuSample2)

	// CPU frequency per core.
	fillCPUFrequency(m)

	// CPU temperature.
	m.CPUTempCelsius = readCPUTemp()

	// Load average.
	fillLoadAvg(m)

	// Memory.
	fillMemory(m)

	// Uptime.
	m.UptimeSeconds = readUptime()

	// Process count from /proc/loadavg.
	m.ProcessCount = readProcessCount()

	// Disk I/O rates.
	m.DiskIO = calcDiskIO(diskSample1, diskSample2)

	// Network I/O rates.
	m.NetworkIO = calcNetIO(netSample1, netSample2)

	// Filesystem usage.
	m.Filesystems = readFilesystems()

	// GPU utilization.
	m.GPUUsagePct = readLinuxGPUUsage()

	return m, nil
}

// readLinuxGPUUsage returns GPU utilization percent.
// Tries nvidia-smi first, then falls back to AMD sysfs gpu_busy_percent.
func readLinuxGPUUsage() float64 {
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		out, err := exec.Command("nvidia-smi",
			"--query-gpu=utilization.gpu",
			"--format=csv,noheader,nounits").Output()
		if err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(out))
			if scanner.Scan() {
				if v, err := strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64); err == nil {
					return v
				}
			}
		}
	}

	// AMD: /sys/class/drm/card*/device/gpu_busy_percent
	entries, err := os.ReadDir("/sys/class/drm")
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) < 4 || name[:4] != "card" || strings.ContainsRune(name[4:], '-') {
			continue
		}
		data, err := os.ReadFile("/sys/class/drm/" + name + "/device/gpu_busy_percent")
		if err != nil {
			continue
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			return v
		}
	}
	return 0
}

// cpuTicks holds per-CPU tick counts from /proc/stat.
type cpuTicks struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func (t cpuTicks) total() uint64 {
	return t.user + t.nice + t.system + t.idle + t.iowait + t.irq + t.softirq + t.steal
}

func (t cpuTicks) busy() uint64 {
	return t.total() - t.idle - t.iowait
}

// readProcStat reads /proc/stat and returns CPU tick data.
// Index 0 is the aggregate "cpu" line; index 1+ are per-core "cpuN" lines.
func readProcStat() []cpuTicks {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil
	}

	var ticks []cpuTicks
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		// "cpu" (aggregate) or "cpu0", "cpu1", etc.
		t := cpuTicks{}
		t.user, _ = strconv.ParseUint(fields[1], 10, 64)
		t.nice, _ = strconv.ParseUint(fields[2], 10, 64)
		t.system, _ = strconv.ParseUint(fields[3], 10, 64)
		t.idle, _ = strconv.ParseUint(fields[4], 10, 64)
		if len(fields) > 5 {
			t.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
		}
		if len(fields) > 6 {
			t.irq, _ = strconv.ParseUint(fields[6], 10, 64)
		}
		if len(fields) > 7 {
			t.softirq, _ = strconv.ParseUint(fields[7], 10, 64)
		}
		if len(fields) > 8 {
			t.steal, _ = strconv.ParseUint(fields[8], 10, 64)
		}
		ticks = append(ticks, t)
	}
	return ticks
}

// calcCPUUsage computes overall and per-core CPU usage from two samples.
func calcCPUUsage(m *LiveMetrics, s1, s2 []cpuTicks) {
	if len(s1) == 0 || len(s2) == 0 {
		return
	}

	// Index 0 is the aggregate line.
	totalDelta := s2[0].total() - s1[0].total()
	if totalDelta > 0 {
		busyDelta := s2[0].busy() - s1[0].busy()
		m.CPUUsagePct = float64(busyDelta) / float64(totalDelta) * 100.0
	}

	// Per-core: indices 1..N.
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}
	for i := 1; i < minLen; i++ {
		td := s2[i].total() - s1[i].total()
		var pct float64
		if td > 0 {
			bd := s2[i].busy() - s1[i].busy()
			pct = float64(bd) / float64(td) * 100.0
		}
		m.CPUPerCore = append(m.CPUPerCore, CoreMetric{
			CoreID:   i - 1,
			UsagePct: pct,
		})
	}
}

// fillCPUFrequency reads per-core frequency from sysfs.
func fillCPUFrequency(m *LiveMetrics) {
	matches, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/cpufreq/scaling_cur_freq")
	if err != nil {
		return
	}

	// Build a map from core ID to frequency.
	freqMap := make(map[int]float64, len(matches))
	for _, path := range matches {
		// Extract CPU number from path: .../cpu0/cpufreq/...
		parts := strings.Split(path, "/")
		for _, p := range parts {
			if strings.HasPrefix(p, "cpu") && len(p) > 3 {
				numStr := p[3:]
				id, err := strconv.Atoi(numStr)
				if err != nil {
					continue
				}
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					continue
				}
				khz, parseErr := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
				if parseErr != nil {
					continue
				}
				freqMap[id] = khz / 1000.0 // KHz → MHz
				break
			}
		}
	}

	for i := range m.CPUPerCore {
		if freq, ok := freqMap[m.CPUPerCore[i].CoreID]; ok {
			m.CPUPerCore[i].FreqMHz = freq
		}
	}
}

// readCPUTemp reads CPU package temperature from /sys/class/thermal/.
func readCPUTemp() float64 {
	matches, err := filepath.Glob("/sys/class/thermal/thermal_zone*")
	if err != nil || len(matches) == 0 {
		return 0
	}

	// Prefer x86_pkg_temp; fallback to first zone.
	var fallbackTemp float64
	var fallbackSet bool

	for _, zone := range matches {
		typeData, err := os.ReadFile(filepath.Join(zone, "type"))
		if err != nil {
			continue
		}
		typeName := strings.TrimSpace(string(typeData))

		tempData, err := os.ReadFile(filepath.Join(zone, "temp"))
		if err != nil {
			continue
		}
		millideg, err := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64)
		if err != nil {
			continue
		}
		temp := millideg / 1000.0

		if typeName == "x86_pkg_temp" {
			return temp
		}
		if !fallbackSet {
			fallbackTemp = temp
			fallbackSet = true
		}
	}

	return fallbackTemp
}

// fillLoadAvg reads /proc/loadavg.
func fillLoadAvg(m *LiveMetrics) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		m.LoadAvg1, _ = strconv.ParseFloat(fields[0], 64)
		m.LoadAvg5, _ = strconv.ParseFloat(fields[1], 64)
		m.LoadAvg15, _ = strconv.ParseFloat(fields[2], 64)
	}
}

// fillMemory parses /proc/meminfo.
func fillMemory(m *LiveMetrics) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}

	vals := make(map[string]uint64)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		key, val, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		val = strings.TrimSuffix(val, " kB")
		val = strings.TrimSpace(val)
		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			continue
		}
		vals[key] = n * 1024 // kB → bytes
	}

	m.MemoryTotalBytes = vals["MemTotal"]
	m.MemoryAvailableBytes = vals["MemAvailable"]
	m.MemoryCachedBytes = vals["Cached"]
	m.MemoryBuffersBytes = vals["Buffers"]

	memFree := vals["MemFree"]
	m.MemoryUsedBytes = m.MemoryTotalBytes - memFree - m.MemoryBuffersBytes - m.MemoryCachedBytes
	if m.MemoryTotalBytes > 0 {
		m.MemoryUsedPct = float64(m.MemoryUsedBytes) / float64(m.MemoryTotalBytes) * 100.0
	}

	m.SwapTotalBytes = vals["SwapTotal"]
	swapFree := vals["SwapFree"]
	if m.SwapTotalBytes > swapFree {
		m.SwapUsedBytes = m.SwapTotalBytes - swapFree
	}
}

// readUptime reads /proc/uptime and returns seconds.
func readUptime() uint64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return uint64(secs)
}

// readProcessCount extracts the running/total process count from /proc/loadavg.
func readProcessCount() int {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 4 {
		return 0
	}
	// Field 3 is "running/total", e.g. "1/423".
	_, total, ok := strings.Cut(fields[3], "/")
	if !ok {
		return 0
	}
	n, _ := strconv.Atoi(total)
	return n
}

// diskSample holds raw sector counts from /proc/diskstats for one device.
type diskSample struct {
	device       string
	readSectors  uint64
	writeSectors uint64
	ioMs         uint64
}

// readDiskStats reads /proc/diskstats and returns samples for physical disks.
func readDiskStats() []diskSample {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return nil
	}

	var samples []diskSample
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}
		dev := fields[2]

		// Skip loop, dm, partition (ends with digit after alpha), ram, sr devices.
		if !isPhysicalDisk(dev) {
			continue
		}

		s := diskSample{device: dev}
		s.readSectors, _ = strconv.ParseUint(fields[5], 10, 64)
		s.writeSectors, _ = strconv.ParseUint(fields[9], 10, 64)
		s.ioMs, _ = strconv.ParseUint(fields[12], 10, 64)
		samples = append(samples, s)
	}
	return samples
}

// isPhysicalDisk returns true if the device name looks like a physical disk
// (not a partition, loopback, device-mapper, or ram device).
func isPhysicalDisk(dev string) bool {
	if strings.HasPrefix(dev, "loop") {
		return false
	}
	if strings.HasPrefix(dev, "dm-") {
		return false
	}
	if strings.HasPrefix(dev, "ram") {
		return false
	}
	if strings.HasPrefix(dev, "sr") {
		return false
	}
	// Partitions: e.g. sda1, nvme0n1p1. Skip names ending with digits after
	// a letter (sda1) or containing "p" followed by digits at the end (nvme0n1p1).
	if len(dev) == 0 {
		return false
	}
	// sd* partitions: sda1, sdb2, etc.
	if strings.HasPrefix(dev, "sd") && len(dev) > 3 {
		last := dev[len(dev)-1]
		if last >= '0' && last <= '9' {
			// Check if the char before the trailing digits is a letter (partition).
			for i := len(dev) - 1; i >= 2; i-- {
				if dev[i] < '0' || dev[i] > '9' {
					if dev[i] >= 'a' && dev[i] <= 'z' {
						return false
					}
					break
				}
			}
		}
	}
	// nvme partitions: nvme0n1p1, nvme0n1p2, etc.
	if strings.HasPrefix(dev, "nvme") && strings.Contains(dev, "p") {
		// nvme0n1 is a disk, nvme0n1p1 is a partition.
		parts := strings.SplitN(dev, "n", 2)
		if len(parts) == 2 && strings.Contains(parts[1], "p") {
			return false
		}
	}
	// vd* partitions: vda1, vdb2, etc.
	if strings.HasPrefix(dev, "vd") && len(dev) > 3 {
		last := dev[len(dev)-1]
		if last >= '0' && last <= '9' {
			return false
		}
	}
	return true
}

// calcDiskIO computes per-device I/O rates from two samples taken ~200ms apart.
func calcDiskIO(s1, s2 []diskSample) []DiskIOMetric {
	m1 := make(map[string]diskSample, len(s1))
	for _, s := range s1 {
		m1[s.device] = s
	}

	const intervalSec = 0.2
	var metrics []DiskIOMetric
	for _, cur := range s2 {
		prev, ok := m1[cur.device]
		if !ok {
			continue
		}
		readDelta := cur.readSectors - prev.readSectors
		writeDelta := cur.writeSectors - prev.writeSectors
		ioDelta := cur.ioMs - prev.ioMs

		dm := DiskIOMetric{
			Device:       cur.device,
			ReadBytesPS:  float64(readDelta*sectorSize) / intervalSec,
			WriteBytesPS: float64(writeDelta*sectorSize) / intervalSec,
		}
		// I/O utilization: ms spent doing I/O / total ms in interval.
		totalMs := intervalSec * 1000.0
		if totalMs > 0 {
			dm.IOUtilPct = float64(ioDelta) / totalMs * 100.0
			if dm.IOUtilPct > 100 {
				dm.IOUtilPct = 100
			}
		}
		metrics = append(metrics, dm)
	}
	return metrics
}

// netSample holds raw byte/packet counts from /proc/net/dev for one interface.
type netSample struct {
	iface     string
	rxBytes   uint64
	txBytes   uint64
	rxPackets uint64
	txPackets uint64
}

// readNetDev reads /proc/net/dev and returns per-interface byte/packet counts.
func readNetDev() []netSample {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil
	}

	var samples []netSample
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue // Skip header lines.
		}
		line := scanner.Text()
		iface, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		iface = strings.TrimSpace(iface)
		if iface == "lo" {
			continue
		}

		fields := strings.Fields(rest)
		if len(fields) < 10 {
			continue
		}

		s := netSample{iface: iface}
		s.rxBytes, _ = strconv.ParseUint(fields[0], 10, 64)
		s.rxPackets, _ = strconv.ParseUint(fields[1], 10, 64)
		s.txBytes, _ = strconv.ParseUint(fields[8], 10, 64)
		s.txPackets, _ = strconv.ParseUint(fields[9], 10, 64)
		samples = append(samples, s)
	}
	return samples
}

// readFilesystems reads /proc/mounts and uses statfs to get usage for real filesystems.
func readFilesystems() []FSMetric {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil
	}

	// Only report real filesystem types, skip virtual/pseudo-fs.
	realFS := map[string]bool{
		"ext4": true, "ext3": true, "ext2": true,
		"xfs": true, "btrfs": true, "zfs": true,
		"ntfs": true, "vfat": true, "fat32": true,
		"tmpfs": true, "nfs": true, "nfs4": true,
	}

	seen := make(map[string]bool)
	var metrics []FSMetric

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		device, mount, fsType := fields[0], fields[1], fields[2]
		if !realFS[fsType] {
			continue
		}
		if seen[mount] {
			continue
		}
		seen[mount] = true

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mount, &stat); err != nil {
			continue
		}

		total := stat.Blocks * uint64(stat.Bsize)
		avail := stat.Bavail * uint64(stat.Bsize)
		used := total - (stat.Bfree * uint64(stat.Bsize))
		var usePct float64
		if total > 0 {
			usePct = float64(used) / float64(total) * 100.0
		}

		metrics = append(metrics, FSMetric{
			Mount:      mount,
			Device:     device,
			FSType:     fsType,
			TotalBytes: total,
			UsedBytes:  used,
			AvailBytes: avail,
			UsePct:     usePct,
		})
	}
	return metrics
}

// calcNetIO computes per-interface network I/O rates from two samples.
func calcNetIO(s1, s2 []netSample) []NetIOMetric {
	m1 := make(map[string]netSample, len(s1))
	for _, s := range s1 {
		m1[s.iface] = s
	}

	const intervalSec = 0.2
	var metrics []NetIOMetric
	for _, cur := range s2 {
		prev, ok := m1[cur.iface]
		if !ok {
			continue
		}
		metrics = append(metrics, NetIOMetric{
			Interface:   cur.iface,
			RxBytesPS:   float64(cur.rxBytes-prev.rxBytes) / intervalSec,
			TxBytesPS:   float64(cur.txBytes-prev.txBytes) / intervalSec,
			RxPacketsPS: float64(cur.rxPackets-prev.rxPackets) / intervalSec,
			TxPacketsPS: float64(cur.txPackets-prev.txPackets) / intervalSec,
		})
	}
	return metrics
}
