//go:build linux

package inventory

import (
	"os"
	"strings"
)

// collectOSVersionDetail returns the OS version detail string on Linux,
// e.g. "Ubuntu 22.04.3 LTS" or "Red Hat Enterprise Linux 9.3".
func collectOSVersionDetail() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == "PRETTY_NAME" {
			return strings.Trim(strings.TrimSpace(v), "\"")
		}
	}
	return ""
}

// collectOSVersion returns the actual OS version string on Linux,
// e.g. "22.04.3" or "9.3".
func collectOSVersion() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == "VERSION_ID" {
			return strings.Trim(strings.TrimSpace(v), "\"")
		}
	}
	return ""
}
