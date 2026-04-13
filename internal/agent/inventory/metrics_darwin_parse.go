//go:build darwin

package inventory

import (
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// parseDarwinTopCPU extracts CPU usage from `top -l 2 -n 0 -s 0` output.
// It returns the LAST "CPU usage:" line values (first is since-boot average).
// Format: "CPU usage: 12.50% user, 8.33% sys, 79.16% idle"
func parseDarwinTopCPU(data []byte) (user, sys, idle float64) {
	var cpuLineCount int
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "CPU usage:") {
			continue
		}
		cpuLineCount++
		user, sys, idle = parseSingleTopCPULine(line)
	}
	if cpuLineCount == 0 {
		return 0, 0, 0
	}
	return user, sys, idle
}

// parseSingleTopCPULine parses "CPU usage: 12.50% user, 8.33% sys, 79.16% idle".
func parseSingleTopCPULine(line string) (user, sys, idle float64) {
	line = strings.TrimPrefix(line, "CPU usage: ")
	parts := strings.Split(line, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		fields := strings.Fields(p)
		if len(fields) < 2 {
			continue
		}
		valStr := strings.TrimSuffix(fields[0], "%")
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		switch fields[1] {
		case "user":
			user = val
		case "sys":
			sys = val
		case "idle":
			idle = val
		}
	}
	return user, sys, idle
}

// parseSysctlLoadAvg parses `sysctl -n vm.loadavg` output.
// Format: "{ 1.23 0.89 0.67 }"
func parseSysctlLoadAvg(data []byte) (l1, l5, l15 float64) {
	s := strings.TrimSpace(string(data))
	s = strings.Trim(s, "{ }")
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	if len(fields) < 3 {
		return 0, 0, 0
	}
	l1, _ = strconv.ParseFloat(fields[0], 64)
	l5, _ = strconv.ParseFloat(fields[1], 64)
	l15, _ = strconv.ParseFloat(fields[2], 64)
	return l1, l5, l15
}

// parseSysctlSwap parses `sysctl vm.swapusage` output.
// Format: "vm.swapusage: total = 2048.00M  used = 512.00M  free = 1536.00M  (encrypted)"
func parseSysctlSwap(data []byte) (totalBytes, usedBytes uint64) {
	s := strings.TrimSpace(string(data))
	if _, after, ok := strings.Cut(s, "vm.swapusage:"); ok {
		s = after
	}

	parts := strings.Fields(s)
	for i := 0; i < len(parts)-2; i++ {
		if parts[i] == "total" && parts[i+1] == "=" {
			totalBytes = parseSwapValue(parts[i+2])
		}
		if parts[i] == "used" && parts[i+1] == "=" {
			usedBytes = parseSwapValue(parts[i+2])
		}
	}
	return totalBytes, usedBytes
}

// parseSwapValue converts a swap size string like "2048.00M" or "4.00G" to bytes.
func parseSwapValue(s string) uint64 {
	s = strings.TrimSpace(s)
	var multiplier float64 = 1
	if strings.HasSuffix(s, "G") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "G")
	} else if strings.HasSuffix(s, "M") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "M")
	} else if strings.HasSuffix(s, "K") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "K")
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return uint64(val * multiplier)
}

// parseSysctlBoottime parses `sysctl -n kern.boottime` output.
// Format: "{ sec = 1234567890, usec = 0 }"
func parseSysctlBoottime(data []byte) int64 {
	s := strings.TrimSpace(string(data))
	re := regexp.MustCompile(`sec\s*=\s*(\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0
	}
	sec, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0
	}
	return sec
}

// parseIostat parses `iostat -d -c 2 -w 1` output.
// Uses the second data row (rates over the 1s interval).
// macOS iostat reports combined throughput only (no read/write split).
func parseIostat(data []byte) []DiskIOMetric {
	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil
	}

	// Find the disk name line.
	var diskNames []string
	var headerLine int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "disk") && !strings.Contains(trimmed, "KB/t") {
			diskNames = strings.Fields(trimmed)
			headerLine = i
			break
		}
	}

	if len(diskNames) == 0 {
		return nil
	}

	// Find the second data row (skip KB/t header rows).
	var secondDataFields []string
	dataLineCount := 0
	for i := headerLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.Contains(trimmed, "KB/t") {
			continue
		}
		dataLineCount++
		if dataLineCount == 2 {
			secondDataFields = strings.Fields(trimmed)
			break
		}
	}

	// Fallback to first data line if only one exists.
	if secondDataFields == nil {
		for i := headerLine + 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" || strings.Contains(trimmed, "KB/t") {
				continue
			}
			secondDataFields = strings.Fields(trimmed)
			break
		}
	}

	if secondDataFields == nil {
		return nil
	}

	// Each disk has 3 fields: KB/t, tps, MB/s.
	var metrics []DiskIOMetric
	for i, name := range diskNames {
		fieldIdx := i * 3
		if fieldIdx+2 >= len(secondDataFields) {
			break
		}
		mbps, err := strconv.ParseFloat(secondDataFields[fieldIdx+2], 64)
		if err != nil {
			continue
		}
		metrics = append(metrics, DiskIOMetric{
			Device:       name,
			ReadBytesPS:  mbps * 1048576, // MB/s to bytes/s (combined throughput).
			WriteBytesPS: 0,              // macOS iostat doesn't split read/write.
			IOUtilPct:    0,              // Not available on macOS.
		})
	}
	return metrics
}

// darwinNetSample holds raw byte/packet counts from netstat -ib.
type darwinNetSample struct {
	iface     string
	rxBytes   uint64
	txBytes   uint64
	rxPackets uint64
	txPackets uint64
}

// parseNetstatIb parses `netstat -ib` output.
// Only rows with "<Link#>" in the Network column have byte counts.
// Loopback (lo0) is skipped.
func parseNetstatIb(data []byte) []darwinNetSample {
	var samples []darwinNetSample
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// Skip header line.
	if !scanner.Scan() {
		return nil
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}

		name := fields[0]
		network := fields[2]

		// Only use Link rows (these have actual byte counts).
		if !strings.HasPrefix(network, "<Link#") {
			continue
		}

		// Skip loopback.
		if name == "lo0" {
			continue
		}

		// Fields vary based on whether Address is present.
		// With address (11 fields): Name Mtu Network Address Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll
		// Without address (10 fields): Name Mtu Network Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll
		var ipktsIdx int
		if len(fields) >= 11 {
			ipktsIdx = 4
		} else {
			ipktsIdx = 3
		}

		if ipktsIdx+5 >= len(fields) {
			continue
		}

		rxPkts, _ := strconv.ParseUint(fields[ipktsIdx], 10, 64)
		rxBytes, _ := strconv.ParseUint(fields[ipktsIdx+2], 10, 64)
		txPkts, _ := strconv.ParseUint(fields[ipktsIdx+3], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[ipktsIdx+5], 10, 64)

		samples = append(samples, darwinNetSample{
			iface:     name,
			rxBytes:   rxBytes,
			txBytes:   txBytes,
			rxPackets: rxPkts,
			txPackets: txPkts,
		})
	}
	return samples
}

// parseDfPk parses `df -Pk` output (POSIX format with 1024-byte blocks).
// Columns: Filesystem, 1024-blocks, Used, Available, Capacity%, Mounted on
// It filters out pseudo-filesystems (devfs, autofs, nullfs, map, etc.).
func parseDfPk(dfData []byte, mountTypes map[string]string) []FSMetric {
	// Pseudo-filesystem devices/types to skip.
	pseudoDevices := map[string]bool{
		"devfs":     true,
		"autofs":    true,
		"nullfs":    true,
		"map":       true,
		"none":      true,
		"tmpfs":     true,
		"fdescfs":   true,
		"procfs":    true,
		"linprocfs": true,
	}

	scanner := bufio.NewScanner(bytes.NewReader(dfData))

	// Skip header line.
	if !scanner.Scan() {
		return nil
	}

	seen := make(map[string]bool)
	var metrics []FSMetric

	for scanner.Scan() {
		line := scanner.Text()
		// "Mounted on" can contain spaces, so we parse carefully.
		// The format has 6 columns, but the last (mount point) can have spaces.
		// Strategy: split into at most 6 fields.
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		device := fields[0]

		// Skip pseudo devices by prefix.
		skipPrefix := false
		for pseudo := range pseudoDevices {
			if strings.HasPrefix(device, pseudo) {
				skipPrefix = true
				break
			}
		}
		if skipPrefix {
			continue
		}

		// Only keep real block devices (start with /).
		if !strings.HasPrefix(device, "/") {
			continue
		}

		totalBlocks, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		usedBlocks, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			continue
		}
		availBlocks, err := strconv.ParseUint(fields[3], 10, 64)
		if err != nil {
			continue
		}

		capStr := strings.TrimSuffix(fields[4], "%")
		usePct, _ := strconv.ParseFloat(capStr, 64)

		// Mount point is everything from field 5 onward (may contain spaces).
		mount := strings.Join(fields[5:], " ")

		if seen[mount] {
			continue
		}
		seen[mount] = true

		// Look up filesystem type from mount output.
		var fsType string
		if mountTypes != nil {
			fsType = mountTypes[mount]
		}

		metrics = append(metrics, FSMetric{
			Mount:      mount,
			Device:     device,
			FSType:     fsType,
			TotalBytes: totalBlocks * 1024,
			UsedBytes:  usedBlocks * 1024,
			AvailBytes: availBlocks * 1024,
			UsePct:     usePct,
		})
	}
	return metrics
}

// parseMountOutput parses macOS `mount` output and returns a map of mount point to fs type.
// Format: "/dev/disk3s1s1 on / (apfs, sealed, local, read-only, journaled)"
func parseMountOutput(data []byte) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		// Parse: <device> on <mount> (<fstype>, <options>...)
		onIdx := strings.Index(line, " on ")
		if onIdx < 0 {
			continue
		}
		rest := line[onIdx+4:]
		// Find the opening parenthesis for options.
		parenIdx := strings.LastIndex(rest, " (")
		if parenIdx < 0 {
			continue
		}
		mount := rest[:parenIdx]
		optStr := rest[parenIdx+2:]
		optStr = strings.TrimSuffix(optStr, ")")

		// First option is the filesystem type.
		parts := strings.SplitN(optStr, ",", 2)
		if len(parts) == 0 {
			continue
		}
		fsType := strings.TrimSpace(parts[0])
		result[mount] = fsType
	}
	return result
}

// calcDarwinNetIO computes per-interface network I/O rates from two samples
// taken ~200ms apart.
func calcDarwinNetIO(s1, s2 []darwinNetSample) []NetIOMetric {
	m1 := make(map[string]darwinNetSample, len(s1))
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
