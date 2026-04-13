//go:build linux

package cli

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

const systemdUnitName = "patchiq-agent.service"
const systemdUnitPath = "/etc/systemd/system/" + systemdUnitName

// installSystemdService writes a systemd unit file, reloads the daemon,
// enables and starts the service as a system-level service (requires root).
func installSystemdService(binaryPath, configPath string) error {
	unit := fmt.Sprintf(`[Unit]
Description=PatchIQ Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s --config %s
Restart=on-failure
RestartSec=10
KillMode=mixed

[Install]
WantedBy=multi-user.target
`, binaryPath, configPath)

	if err := os.WriteFile(systemdUnitPath, []byte(unit), 0644); err != nil {
		return fmt.Errorf("write systemd unit %s: %w", systemdUnitPath, err)
	}

	for _, args := range [][]string{
		{"daemon-reload"},
		{"enable", systemdUnitName},
		{"start", systemdUnitName},
	} {
		if out, err := exec.Command("systemctl", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl %v: %w: %s", args, err, string(out))
		}
	}

	slog.Info("systemd service installed and started", "unit", systemdUnitName)
	return nil
}

// copyBinaryToInstallDir copies the current executable to /usr/local/bin/patchiq-agent
// so the systemd unit has a stable path. Returns the installed path.
func copyBinaryToInstallDir() (string, error) {
	src, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}
	src, err = filepath.EvalSymlinks(src)
	if err != nil {
		return "", fmt.Errorf("resolve executable symlinks: %w", err)
	}

	dst := "/usr/local/bin/patchiq-agent"
	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("read binary %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0755); err != nil {
		return "", fmt.Errorf("write binary %s: %w", dst, err)
	}

	slog.Info("agent binary installed", "path", dst)
	return dst, nil
}

// RunService manages the systemd service.
func RunService(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: patchiq-agent service [status|stop|restart]")
		return ExitError
	}

	switch args[0] {
	case "status":
		out, err := exec.Command("systemctl", "status", systemdUnitName).CombinedOutput()
		fmt.Print(string(out))
		if err != nil {
			return ExitError
		}
		return 0
	case "stop":
		if out, err := exec.Command("systemctl", "stop", systemdUnitName).CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "stop failed: %s\n", string(out))
			return ExitError
		}
		fmt.Println("Agent service stopped.")
		return 0
	case "restart":
		if out, err := exec.Command("systemctl", "restart", systemdUnitName).CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "restart failed: %s\n", string(out))
			return ExitError
		}
		fmt.Println("Agent service restarted.")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown service command: %s\n", args[0])
		return ExitError
	}
}
