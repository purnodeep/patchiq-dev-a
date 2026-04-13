//go:build linux

package sysinfo

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

func EnrichEndpointInfo(info *pb.EndpointInfo, _ *slog.Logger) {
	// Architecture
	info.OsVersion = readFileField("/etc/os-release", "PRETTY_NAME", runtime.GOOS+"/"+runtime.GOARCH)

	// CPU
	info.CpuType = readCPUModel()

	// Memory (total bytes)
	info.MemoryBytes = readMemTotal()

	// Kernel version
	if data, err := os.ReadFile("/proc/version"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			info.HardwareModel = parts[2] // kernel version string
		}
	}

	// IP addresses
	addrs := localIPs()
	if len(addrs) > 0 {
		info.IpAddresses = addrs
	}

	// Extra hardware info via tags (fields not in proto schema).
	if info.Tags == nil {
		info.Tags = make(map[string]string)
	}
	if cores := countCPUCores(); cores > 0 {
		info.Tags["cpu_cores"] = fmt.Sprintf("%d", cores)
	}
	if diskGB := totalDiskGB(); diskGB > 0 {
		info.Tags["disk_total_gb"] = fmt.Sprintf("%d", diskGB)
	}
}

func readFileField(path, key, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), "\"")
		}
	}
	return fallback
}

func readCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok {
			if strings.TrimSpace(k) == "model name" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func readMemTotal() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				var kb uint64
				for _, c := range parts[1] {
					if c >= '0' && c <= '9' {
						kb = kb*10 + uint64(c-'0')
					}
				}
				return kb * 1024 // convert kB to bytes
			}
		}
	}
	return 0
}

// countCPUCores counts the number of "processor" lines in /proc/cpuinfo.
func countCPUCores() int {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "processor") {
			count++
		}
	}
	return count
}

// totalDiskGB returns the total size of the root filesystem in GB.
func totalDiskGB() int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0
	}
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	return int64(totalBytes / (1024 * 1024 * 1024))
}

func localIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}
