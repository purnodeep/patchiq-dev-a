//go:build linux

package inventory

import (
	"os"
	"strings"
)

// collectKernelVersion returns the Linux kernel version from /proc/version,
// e.g. "5.15.0-100-generic".
func collectKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	parts := strings.Fields(string(data))
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
