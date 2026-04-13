//go:build windows

package agent

import (
	"os"
	"runtime"
)

// Hostname returns the machine's hostname on Windows via os.Hostname().
func Hostname() (string, error) {
	return os.Hostname()
}

// OSVersion returns a human-readable OS version string on Windows.
// Falls back to "windows/amd64" until Windows-specific collection
// is implemented.
func OSVersion() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// OSVersionDetail returns the same as OSVersion on Windows.
func OSVersionDetail() string {
	return OSVersion()
}
