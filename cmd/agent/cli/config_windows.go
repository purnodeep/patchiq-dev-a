//go:build windows

package cli

import (
	"os"
	"path/filepath"
)

const defaultConfigPath = `C:\ProgramData\PatchIQ\agent.yaml`

// DefaultDataDir returns C:\ProgramData\PatchIQ, or a fallback under the user's
// home directory if ProgramData is not available.
func DefaultDataDir() string {
	if pd := os.Getenv("ProgramData"); pd != "" {
		return filepath.Join(pd, "PatchIQ")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return `C:\PatchIQ`
	}
	return filepath.Join(home, ".patchiq")
}
