//go:build windows

package patcher

import (
	"os"
	"strings"
)

// detectInstallerType reads the first portion of an EXE file and attempts to
// identify the installer framework. Returns silent-install arguments for the
// detected type, or empty string if unrecognized.
func detectInstallerType(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Read first 64KB — enough to find most installer framework signatures.
	buf := make([]byte, 64*1024)
	n, _ := f.Read(buf)
	if n == 0 {
		return ""
	}
	content := string(buf[:n])

	// Inno Setup: look for "Inno Setup" string in the binary.
	if strings.Contains(content, "Inno Setup") {
		return "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
	}

	// NSIS (Nullsoft): look for "Nullsoft" or "NSIS" markers.
	if strings.Contains(content, "Nullsoft") || strings.Contains(content, "NSIS") {
		return "/S"
	}

	// InstallShield: look for "InstallShield" string.
	if strings.Contains(content, "InstallShield") {
		return "/s /v\"/qn /norestart\""
	}

	return ""
}
