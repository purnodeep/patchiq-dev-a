//go:build linux

package agent

import (
	"os"
	"runtime"
	"strings"
)

// Hostname returns the machine's hostname on Linux via os.Hostname().
func Hostname() (string, error) {
	return os.Hostname()
}

// OSVersion returns a human-readable OS version string on Linux,
// e.g. "Ubuntu 22.04.3 LTS". Falls back to "linux/amd64".
func OSVersion() string {
	if detail := readOSReleaseField("PRETTY_NAME"); detail != "" {
		return detail
	}
	return runtime.GOOS + "/" + runtime.GOARCH
}

// OSVersionDetail returns the same as OSVersion on Linux (PRETTY_NAME
// already includes the full detail).
func OSVersionDetail() string {
	return OSVersion()
}

func readOSReleaseField(key string) string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), "\"")
		}
	}
	return ""
}
