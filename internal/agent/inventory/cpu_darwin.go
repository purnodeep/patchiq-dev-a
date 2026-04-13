//go:build darwin

package inventory

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

// darwinCPUTicks holds per-CPU tick counts parsed from sysctl kern.cp_time.
type darwinCPUTicks struct {
	user, system, idle, nice uint32
}

// readDarwinCPUTicks reads per-CPU tick counters via sysctl.
// This is a pure-Go replacement for the Mach host_processor_info CGO approach.
// It reads aggregate CPU ticks from `top -l 1 -n 0 -s 0` output.
// For per-core data, macOS does not expose per-core ticks via sysctl without
// root access, so we return a single aggregate entry which the caller will use.
func readDarwinCPUTicks() ([]darwinCPUTicks, error) {
	out, err := exec.Command("top", "-l", "1", "-n", "0", "-s", "0").Output()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "CPU usage:") {
			continue
		}
		// Format: "CPU usage: 5.26% user, 10.52% sys, 84.21% idle"
		user, sys, idle := parseDarwinTopCPU(out)
		return []darwinCPUTicks{{
			user:   uint32(user * 100),
			system: uint32(sys * 100),
			idle:   uint32(idle * 100),
			nice:   0,
		}}, nil
	}

	return nil, nil
}

// calcDarwinPerCoreCPU computes per-core usage from two tick snapshots.
func calcDarwinPerCoreCPU(s1, s2 []darwinCPUTicks) []CoreMetric {
	n := len(s1)
	if len(s2) < n {
		n = len(s2)
	}

	cores := make([]CoreMetric, 0, n)
	for i := 0; i < n; i++ {
		dUser := s2[i].user - s1[i].user
		dSys := s2[i].system - s1[i].system
		dIdle := s2[i].idle - s1[i].idle
		dNice := s2[i].nice - s1[i].nice
		total := float64(dUser + dSys + dIdle + dNice)
		busy := float64(dUser + dSys + dNice)

		var pct float64
		if total > 0 {
			pct = busy / total * 100
		}

		cores = append(cores, CoreMetric{
			CoreID:   i,
			UsagePct: pct,
		})
	}

	return cores
}

// parseCPUFlags splits a space-separated CPU flags string into a slice.
// Used for Intel Macs where sysctl machdep.cpu.features returns flags.
func parseCPUFlags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	fields := strings.Fields(raw)
	flags := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			flags = append(flags, strings.ToLower(f))
		}
	}
	return flags
}
