//go:build darwin

package sysinfo

import (
	"bytes"
	"log/slog"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

func EnrichEndpointInfo(info *pb.EndpointInfo, logger *slog.Logger) {
	out, err := exec.Command("system_profiler", "SPHardwareDataType", "SPSoftwareDataType").Output()
	if err != nil {
		logger.Warn("enrich endpoint info: system_profiler failed", "error", err)
		return
	}
	ParseSystemProfiler(out, info)
}

// ParseSystemProfiler extracts hardware and software info from system_profiler output.
func ParseSystemProfiler(data []byte, info *pb.EndpointInfo) {
	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if k, v, ok := strings.Cut(trimmed, ": "); ok {
			switch strings.TrimSpace(k) {
			case "Model Name", "Model Identifier":
				if info.HardwareModel == "" {
					info.HardwareModel = strings.TrimSpace(v)
				}
			case "Chip", "Processor Name":
				if info.CpuType == "" {
					info.CpuType = strings.TrimSpace(v)
				}
			case "Memory":
				info.MemoryBytes = ParseMemoryString(strings.TrimSpace(v))
			case "System Version":
				info.OsVersionDetail = strings.TrimSpace(v)
				info.OsVersion = runtime.GOOS + "/" + runtime.GOARCH
			}
		}
	}
}

// ParseMemoryString converts "16 GB" or "32 GB" to bytes.
func ParseMemoryString(s string) uint64 {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return 0
	}
	val, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(fields[1]) {
	case "GB":
		return val * 1024 * 1024 * 1024
	case "MB":
		return val * 1024 * 1024
	case "TB":
		return val * 1024 * 1024 * 1024 * 1024
	}
	return 0
}
