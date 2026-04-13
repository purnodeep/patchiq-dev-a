//go:build !windows

package cli

import (
	"os"
	"path/filepath"
)

const defaultConfigPath = "/etc/patchiq/agent.yaml"

// DefaultDataDir returns /var/lib/patchiq if running as root, otherwise ~/.patchiq.
func DefaultDataDir() string {
	if os.Geteuid() == 0 {
		return "/var/lib/patchiq"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".patchiq"
	}
	return filepath.Join(home, ".patchiq")
}
