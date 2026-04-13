//go:build windows

package inventory

import (
	"log/slog"
	"os/exec"
	"strings"
)

// collectOSVersionDetail returns the full OS caption on Windows,
// e.g. "Microsoft Windows 11 Pro" or "Microsoft Windows Server 2022 Standard".
func collectOSVersionDetail() string {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`(Get-CimInstance Win32_OperatingSystem).Caption`).Output()
	if err != nil {
		slog.Warn("inventory: failed to collect OS version detail", "error", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

// collectOSVersion returns the Windows build version string,
// e.g. "10.0.22631" (Windows 11 23H2).
func collectOSVersion() string {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`(Get-CimInstance Win32_OperatingSystem).Version`).Output()
	if err != nil {
		slog.Warn("inventory: failed to collect OS version", "error", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}
