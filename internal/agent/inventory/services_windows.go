//go:build windows

package inventory

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
)

// CollectServices collects Windows service information via Get-Service.
func CollectServices(ctx context.Context, logger *slog.Logger) ([]ServiceInfo, error) {
	const psCmd = `Get-Service | Select-Object Name, DisplayName, Status, StartType | ConvertTo-Json -Compress`

	out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
	if err != nil {
		return nil, fmt.Errorf("collect services: Get-Service: %w", err)
	}

	services := parseWinServices(string(bytes.TrimSpace(out)))
	if logger != nil {
		logger.Info("windows services collected", "count", len(services))
	}
	return services, nil
}
