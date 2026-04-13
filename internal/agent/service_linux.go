//go:build linux

package agent

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// LinuxService manages the PatchIQ agent systemd service.
type LinuxService struct {
	ServiceName string
	BinaryPath  string
	DataDir     string
	LogFile     string
}

const unitTemplate = `[Unit]
Description=PatchIQ Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=5
User=root
Environment=PATCHIQ_AGENT_DATA_DIR=%s
Environment=PATCHIQ_AGENT_LOG_FILE=%s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=patchiq-agent

[Install]
WantedBy=multi-user.target
`

// UnitFilePath returns the systemd unit file path.
func (s *LinuxService) UnitFilePath() string {
	return "/etc/systemd/system/" + s.ServiceName + ".service"
}

// GenerateUnitFile returns the systemd unit file content.
func (s *LinuxService) GenerateUnitFile() string {
	return fmt.Sprintf(unitTemplate, s.BinaryPath, s.DataDir, s.LogFile)
}

// Install writes the unit file, reloads systemd, and enables the service.
func (s *LinuxService) Install() error {
	unit := s.GenerateUnitFile()
	if err := os.WriteFile(s.UnitFilePath(), []byte(unit), 0644); err != nil {
		return fmt.Errorf("write unit file %s: %w", s.UnitFilePath(), err)
	}
	slog.Info("wrote systemd unit file", "path", s.UnitFilePath())

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %s: %w", string(out), err)
	}

	if out, err := exec.Command("systemctl", "enable", s.ServiceName).CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable %s: %s: %w", s.ServiceName, string(out), err)
	}
	slog.Info("enabled systemd service", "service", s.ServiceName)

	return nil
}

// Uninstall disables and stops the service, removes the unit file, and reloads systemd.
func (s *LinuxService) Uninstall() error {
	if out, err := exec.Command("systemctl", "disable", s.ServiceName).CombinedOutput(); err != nil {
		slog.Warn("systemctl disable failed", "service", s.ServiceName, "output", string(out), "error", err)
	}

	if out, err := exec.Command("systemctl", "stop", s.ServiceName).CombinedOutput(); err != nil {
		slog.Warn("systemctl stop failed", "service", s.ServiceName, "output", string(out), "error", err)
	}

	if err := os.Remove(s.UnitFilePath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file %s: %w", s.UnitFilePath(), err)
	}
	slog.Info("removed systemd unit file", "path", s.UnitFilePath())

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %s: %w", string(out), err)
	}

	return nil
}

// Start starts the systemd service.
func (s *LinuxService) Start() error {
	out, err := exec.Command("systemctl", "start", s.ServiceName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl start %s: %s: %w", s.ServiceName, string(out), err)
	}
	return nil
}

// Stop stops the systemd service.
func (s *LinuxService) Stop() error {
	out, err := exec.Command("systemctl", "stop", s.ServiceName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl stop %s: %s: %w", s.ServiceName, string(out), err)
	}
	return nil
}

// Status returns the active state of the systemd service.
func (s *LinuxService) Status() (string, error) {
	out, err := exec.Command("systemctl", "is-active", s.ServiceName).CombinedOutput()
	// systemctl is-active returns exit code 3 for inactive/dead, which is not an error for us.
	result := strings.TrimSpace(string(out))
	if err != nil && result == "" {
		return "", fmt.Errorf("systemctl is-active %s: %w", s.ServiceName, err)
	}
	return result, nil
}
