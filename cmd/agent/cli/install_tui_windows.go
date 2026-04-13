//go:build windows

package cli

import (
	"fmt"
	"time"
)

// installAndStartService installs, starts, and health-checks the Windows service.
// Called from the TUI wizard after enrollment succeeds.
func installAndStartService() error {
	// Step 1: Install service.
	if code := serviceInstall(); code != ExitOK {
		return fmt.Errorf("service install failed (exit code %d)", code)
	}

	// Step 2: Start service.
	if code := serviceStart(); code != ExitOK {
		return fmt.Errorf("service start failed (exit code %d)", code)
	}

	// Step 3: Wait for service to reach running state.
	if err := waitForServiceRunning(serviceName, 30*time.Second); err != nil {
		return fmt.Errorf("service health check: %w", err)
	}

	return nil
}
