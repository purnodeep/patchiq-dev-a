package sysinfo

import (
	"log/slog"
	"os"
	"runtime"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// BuildEndpointInfo constructs an EndpointInfo populated with hostname, OS family,
// OS version, and platform-specific hardware details via EnrichEndpointInfo.
func BuildEndpointInfo(logger *slog.Logger) *pb.EndpointInfo {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Warn("failed to get hostname for endpoint info", "error", err)
		hostname = "unknown"
	}
	var osFamily pb.OsFamily
	switch runtime.GOOS {
	case "linux":
		osFamily = pb.OsFamily_OS_FAMILY_LINUX
	case "windows":
		osFamily = pb.OsFamily_OS_FAMILY_WINDOWS
	case "darwin":
		osFamily = pb.OsFamily_OS_FAMILY_MACOS
	default:
		logger.Warn("unrecognized OS for endpoint info, reporting as unspecified", "os", runtime.GOOS)
	}
	info := &pb.EndpointInfo{
		Hostname:  hostname,
		OsFamily:  osFamily,
		OsVersion: runtime.GOOS + "/" + runtime.GOARCH,
	}
	EnrichEndpointInfo(info, logger)
	return info
}
