//go:build darwin

package inventory

import (
	"bufio"
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// appleChipFreqs maps Apple Silicon chip names to [maxMHz, minMHz(efficiency)].
// Used as a fallback when sysctl perflevel keys are unavailable and the agent
// is not running as root (no powermetrics access).
var appleChipFreqs = map[string][2]int{ // [maxMHz, minMHz(efficiency)]
	"Apple M1":       {3200, 2064},
	"Apple M1 Pro":   {3228, 2064},
	"Apple M1 Max":   {3228, 2064},
	"Apple M1 Ultra": {3228, 2064},
	"Apple M2":       {3490, 2420},
	"Apple M2 Pro":   {3490, 2420},
	"Apple M2 Max":   {3490, 2420},
	"Apple M2 Ultra": {3490, 2420},
	"Apple M3":       {4050, 2750},
	"Apple M3 Pro":   {4050, 2750},
	"Apple M3 Max":   {4050, 2750},
	"Apple M3 Ultra": {4050, 2750},
	"Apple M4":       {4400, 2850},
	"Apple M4 Pro":   {4500, 2850},
	"Apple M4 Max":   {4500, 2850},
}

// lookupAppleChipFreq checks whether modelName matches a known Apple Silicon
// chip and returns (maxMHz, minMHz, true) if found, or (0, 0, false) otherwise.
func lookupAppleChipFreq(modelName string) (maxMHz, minMHz float64, ok bool) {
	// Try exact match first.
	if freq, found := appleChipFreqs[modelName]; found {
		return float64(freq[0]), float64(freq[1]), true
	}
	// Try prefix match — model strings from sysctl can include extra suffixes
	// (e.g. bin count). Match the longest known key that is a prefix.
	var bestKey string
	for key := range appleChipFreqs {
		if strings.HasPrefix(modelName, key) && len(key) > len(bestKey) {
			bestKey = key
		}
	}
	if bestKey != "" {
		freq := appleChipFreqs[bestKey]
		return float64(freq[0]), float64(freq[1]), true
	}
	return 0, 0, false
}

// powermetricsFreq holds parsed CPU cluster frequencies from powermetrics output.
type powermetricsFreq struct {
	PClusterMHz float64 // Performance cluster active frequency.
	EClusterMHz float64 // Efficiency cluster active frequency.
	// PerCoreMHz maps core index to its active frequency in MHz.
	// Only populated when per-core lines are present in the output.
	PerCoreMHz map[int]float64
}

// Regex patterns for powermetrics cpu_power output.
var (
	// Matches: "P-Cluster HW active frequency: 4408 MHz" (or P0-Cluster, P1-Cluster, etc.)
	rePClusterFreq = regexp.MustCompile(`P\d*-Cluster HW active frequency:\s*(\d+)\s*MHz`)
	// Matches: "E-Cluster HW active frequency: 2856 MHz" (or E0-Cluster, E1-Cluster, etc.)
	reEClusterFreq = regexp.MustCompile(`E\d*-Cluster HW active frequency:\s*(\d+)\s*MHz`)
	// Matches: "CPU 0 active frequency: 4408 MHz" or "Core 0 active frequency: ..."
	rePerCoreFreq = regexp.MustCompile(`(?:CPU|Core)\s+(\d+)\s+.*?active frequency:\s*(\d+)\s*MHz`)
)

// readPowermetricsFreq runs powermetrics as root and parses CPU cluster frequencies.
// Returns nil if not running as root, if powermetrics is unavailable, or on parse failure.
func readPowermetricsFreq(ctx context.Context) *powermetricsFreq {
	if os.Geteuid() != 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(timeoutCtx, "powermetrics",
		"--samplers", "cpu_power", "-n", "1", "--sample-rate", "1").Output()
	if err != nil {
		slog.Debug("cpufreq darwin: powermetrics failed", "error", err)
		return nil
	}

	return parsePowermetricsFreq(out)
}

// parsePowermetricsFreq extracts cluster and per-core frequencies from
// powermetrics cpu_power output. Exported for testing.
func parsePowermetricsFreq(data []byte) *powermetricsFreq {
	result := &powermetricsFreq{
		PerCoreMHz: make(map[int]float64),
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		if m := rePClusterFreq.FindStringSubmatch(line); m != nil {
			if mhz, err := strconv.ParseFloat(m[1], 64); err == nil && mhz > result.PClusterMHz {
				result.PClusterMHz = mhz
			}
			continue
		}

		if m := reEClusterFreq.FindStringSubmatch(line); m != nil {
			if mhz, err := strconv.ParseFloat(m[1], 64); err == nil {
				// Keep the lowest E-cluster frequency (or first seen).
				if result.EClusterMHz == 0 || mhz < result.EClusterMHz {
					result.EClusterMHz = mhz
				}
			}
			continue
		}

		if m := rePerCoreFreq.FindStringSubmatch(line); m != nil {
			coreID, err1 := strconv.Atoi(m[1])
			mhz, err2 := strconv.ParseFloat(m[2], 64)
			if err1 == nil && err2 == nil {
				result.PerCoreMHz[coreID] = mhz
			}
			continue
		}
	}

	if result.PClusterMHz == 0 && result.EClusterMHz == 0 && len(result.PerCoreMHz) == 0 {
		return nil
	}
	return result
}

// fillAppleSiliconFreq populates MaxMHz and MinMHz on a CPUInfo using the
// priority chain: powermetrics (root) > lookup table (non-root) > leave as-is.
// It is called after sysctl-based frequency collection when values are still zero.
func fillAppleSiliconFreq(ctx context.Context, info *CPUInfo) {
	if info.MaxMHz > 0 && info.MinMHz > 0 {
		return // Already populated by sysctl.
	}

	// Priority 1: powermetrics (requires root).
	if pm := readPowermetricsFreq(ctx); pm != nil {
		if info.MaxMHz == 0 && pm.PClusterMHz > 0 {
			info.MaxMHz = pm.PClusterMHz
			slog.Debug("cpufreq darwin: max_mhz from powermetrics P-Cluster", "mhz", pm.PClusterMHz)
		}
		if info.MinMHz == 0 && pm.EClusterMHz > 0 {
			info.MinMHz = pm.EClusterMHz
			slog.Debug("cpufreq darwin: min_mhz from powermetrics E-Cluster", "mhz", pm.EClusterMHz)
		}
		if info.MaxMHz > 0 && info.MinMHz > 0 {
			return
		}
	}

	// Priority 2: hardcoded lookup table.
	if maxMHz, minMHz, ok := lookupAppleChipFreq(info.ModelName); ok {
		if info.MaxMHz == 0 {
			info.MaxMHz = maxMHz
			slog.Debug("cpufreq darwin: max_mhz from lookup table", "model", info.ModelName, "mhz", maxMHz)
		}
		if info.MinMHz == 0 {
			info.MinMHz = minMHz
			slog.Debug("cpufreq darwin: min_mhz from lookup table", "model", info.ModelName, "mhz", minMHz)
		}
	} else if info.MaxMHz == 0 || info.MinMHz == 0 {
		slog.Debug("cpufreq darwin: unknown Apple Silicon chip, frequency unavailable",
			"model", info.ModelName)
	}
}

// fillDarwinPerCoreFreq populates FreqMHz on per-core metrics using
// powermetrics (root) or the lookup table (non-root).
func fillDarwinPerCoreFreq(ctx context.Context, cores []CoreMetric, modelName string) {
	if len(cores) == 0 {
		return
	}

	// Priority 1: powermetrics per-core data (requires root).
	if pm := readPowermetricsFreq(ctx); pm != nil {
		if len(pm.PerCoreMHz) > 0 {
			for i := range cores {
				if freq, ok := pm.PerCoreMHz[cores[i].CoreID]; ok {
					cores[i].FreqMHz = freq
				}
			}
			slog.Debug("cpufreq darwin: per-core freq from powermetrics",
				"cores_populated", len(pm.PerCoreMHz))
			return
		}
		// No per-core data, but we have cluster-level — use P-cluster for all.
		if pm.PClusterMHz > 0 {
			for i := range cores {
				cores[i].FreqMHz = pm.PClusterMHz
			}
			slog.Debug("cpufreq darwin: per-core freq from powermetrics P-Cluster",
				"mhz", pm.PClusterMHz)
			return
		}
	}

	// Priority 2: lookup table — use max (P-core) freq for all cores.
	if maxMHz, _, ok := lookupAppleChipFreq(modelName); ok && maxMHz > 0 {
		for i := range cores {
			cores[i].FreqMHz = maxMHz
		}
		slog.Debug("cpufreq darwin: per-core freq from lookup table",
			"model", modelName, "mhz", maxMHz)
	}
}

// cpuModelName reads the CPU model name via sysctl for use by the metrics collector.
func cpuModelName(ctx context.Context) string {
	name, err := sysctlString(ctx, "machdep.cpu.brand_string")
	if err != nil {
		return ""
	}
	return name
}
