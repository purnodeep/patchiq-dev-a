//go:build windows

package sysinfo

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"runtime"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// EnrichEndpointInfo populates EndpointInfo with Windows-specific system details
// via a single combined PowerShell CIM query plus Go stdlib for IPs and CPU cores.
func EnrichEndpointInfo(info *pb.EndpointInfo, logger *slog.Logger) {
	const psCmd = `$os = Get-CimInstance Win32_OperatingSystem | Select-Object Caption, Version, BuildNumber, TotalVisibleMemorySize
$cpu = (Get-CimInstance Win32_Processor | Select-Object -First 1).Name
$disk = (Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | Measure-Object -Property Size -Sum).Sum
@{
  os_caption = $os.Caption
  os_version = $os.Version
  os_build = $os.BuildNumber
  cpu_name = $cpu
  mem_total_kb = $os.TotalVisibleMemorySize
  disk_total_bytes = $disk
} | ConvertTo-Json -Compress`

	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
	if err != nil {
		logger.Warn("enrich endpoint info: powershell CIM query failed, enrollment will have incomplete system info", "error", err)
	} else {
		ei := parseWinEnrollmentJSON(string(bytes.TrimSpace(out)))
		if ei.osCaption != "" {
			info.OsVersion = ei.osCaption
			info.OsVersionDetail = ei.osCaption + " Build " + ei.osBuild
		}
		if ei.osVersion != "" {
			info.HardwareModel = ei.osVersion
		}
		if ei.cpuName != "" {
			info.CpuType = ei.cpuName
		}
		if ei.memTotalKB > 0 {
			info.MemoryBytes = ei.memTotalKB * 1024
		}
		if ei.diskTotalBytes > 0 {
			if info.Tags == nil {
				info.Tags = make(map[string]string)
			}
			info.Tags["disk_total_gb"] = fmt.Sprintf("%d", ei.diskTotalBytes/(1024*1024*1024))
		}
	}

	addrs := localIPs()
	if len(addrs) > 0 {
		info.IpAddresses = addrs
	}

	if info.Tags == nil {
		info.Tags = make(map[string]string)
	}
	info.Tags["cpu_cores"] = fmt.Sprintf("%d", runtime.NumCPU())
	info.Tags["arch"] = runtime.GOARCH
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
