//go:build linux

package cli

import (
	"fmt"

	"github.com/skenzeriq/patchiq/internal/agent"
)

// installAndStartService registers the agent as a systemd service and starts it.
// Called by the install TUI after successful enrollment. Uses the same defaults
// as the `patchiq-agent service install` subcommand (see service_linux.go).
func installAndStartService() error {
	svc := &agent.LinuxService{
		ServiceName: "patchiq-agent",
		BinaryPath:  "/usr/local/bin/patchiq-agent",
		DataDir:     "/var/lib/patchiq",
		LogFile:     "/var/log/patchiq-agent.log",
	}
	if err := svc.Install(); err != nil {
		return fmt.Errorf("install systemd service: %w", err)
	}
	if err := svc.Start(); err != nil {
		return fmt.Errorf("start systemd service: %w", err)
	}
	return nil
}
