//go:build darwin

package inventory

import (
	"os/exec"
	"strings"
)

// collectOSVersionDetail returns the OS version detail string on macOS,
// e.g. "macOS 15.2" or "macOS 14.2.1 (23C71)".
func collectOSVersionDetail() string {
	// sw_vers -productName gives "macOS", -productVersion gives "15.2".
	name, err := exec.Command("sw_vers", "-productName").Output()
	if err != nil {
		return ""
	}
	version, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return ""
	}

	detail := strings.TrimSpace(string(name)) + " " + strings.TrimSpace(string(version))

	// Optionally append build version (e.g. "24C101").
	if build, err := exec.Command("sw_vers", "-buildVersion").Output(); err == nil {
		if b := strings.TrimSpace(string(build)); b != "" {
			detail += " (" + b + ")"
		}
	}

	return detail
}

// collectOSVersion returns the actual OS version string on macOS,
// e.g. "15.2" or "14.2.1".
func collectOSVersion() string {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
