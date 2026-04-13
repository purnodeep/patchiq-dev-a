//go:build linux

package cli

import "fmt"

// installService copies the binary to a stable path, installs a systemd unit,
// and starts the agent daemon.
func installService(configPath string, logStatus func(string)) error {
	logStatus("Copying agent binary to /usr/local/bin...")
	binaryPath, err := copyBinaryToInstallDir()
	if err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}

	logStatus("Installing systemd service...")
	if err := installSystemdService(binaryPath, configPath); err != nil {
		return fmt.Errorf("systemd service: %w", err)
	}

	logStatus("Agent service started.")
	return nil
}
