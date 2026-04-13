//go:build darwin

package agent

import (
	"os"
	"os/exec"
	"strings"
)

// Hostname returns the machine's hostname on macOS. It prefers
// scutil --get ComputerName (user-facing name), falls back to
// scutil --get LocalHostName (Bonjour name), and finally os.Hostname().
// On macOS, os.Hostname() can return an IP address when DNS reverse
// lookup fails, so the scutil commands are tried first.
func Hostname() (string, error) {
	// Try ComputerName first (e.g. "Sandy's MacBook Pro").
	if out, err := exec.Command("scutil", "--get", "ComputerName").Output(); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name, nil
		}
	}

	// Try LocalHostName (e.g. "Sandys-MacBook-Pro").
	if out, err := exec.Command("scutil", "--get", "LocalHostName").Output(); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name, nil
		}
	}

	// Fallback to os.Hostname() (may return IP on some networks).
	return os.Hostname()
}

// OSVersion returns a human-readable OS version string on macOS,
// e.g. "macOS 15.4". Uses sw_vers to get the product name and version.
func OSVersion() string {
	name, err := exec.Command("sw_vers", "-productName").Output()
	if err != nil {
		return "macOS"
	}
	version, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return strings.TrimSpace(string(name))
	}
	return strings.TrimSpace(string(name)) + " " + strings.TrimSpace(string(version))
}

// OSVersionDetail returns a detailed OS version string on macOS,
// e.g. "macOS 15.4 (24E248)". Includes the build number.
func OSVersionDetail() string {
	base := OSVersion()
	if base == "" {
		return ""
	}
	if build, err := exec.Command("sw_vers", "-buildVersion").Output(); err == nil {
		if b := strings.TrimSpace(string(build)); b != "" {
			return base + " (" + b + ")"
		}
	}
	return base
}
