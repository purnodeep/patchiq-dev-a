//go:build linux

package agent

import (
	"strings"
	"testing"
)

func TestLinuxService_GenerateUnitFile(t *testing.T) {
	svc := &LinuxService{
		ServiceName: "patchiq-agent",
		BinaryPath:  "/usr/local/bin/patchiq-agent",
		DataDir:     "/var/lib/patchiq",
		LogFile:     "/var/log/patchiq-agent.log",
	}

	unit := svc.GenerateUnitFile()

	checks := []string{
		"Description=PatchIQ Agent",
		"After=network-online.target",
		"Wants=network-online.target",
		"Type=simple",
		"ExecStart=/usr/local/bin/patchiq-agent",
		"Restart=always",
		"RestartSec=5",
		"User=root",
		"Environment=PATCHIQ_AGENT_DATA_DIR=/var/lib/patchiq",
		"Environment=PATCHIQ_AGENT_LOG_FILE=/var/log/patchiq-agent.log",
		"StandardOutput=journal",
		"StandardError=journal",
		"SyslogIdentifier=patchiq-agent",
		"WantedBy=multi-user.target",
	}

	for _, check := range checks {
		if !strings.Contains(unit, check) {
			t.Errorf("unit file missing %q", check)
		}
	}
}

func TestLinuxService_UnitFilePath(t *testing.T) {
	svc := &LinuxService{ServiceName: "patchiq-agent"}
	want := "/etc/systemd/system/patchiq-agent.service"
	if got := svc.UnitFilePath(); got != want {
		t.Errorf("UnitFilePath() = %q, want %q", got, want)
	}
}
