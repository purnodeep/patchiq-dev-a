//go:build windows

package inventory

import (
	"context"
	"fmt"
	"log/slog"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// windowsFeaturesCollector collects Windows optional features via PowerShell.
type windowsFeaturesCollector struct {
	runner commandRunner
	logger *slog.Logger
}

func (c *windowsFeaturesCollector) Name() string { return "windows_features" }

func (c *windowsFeaturesCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	// Try Get-WindowsOptionalFeature first (works on Windows 10/11 client).
	out, err := c.runner.Run(ctx, "powershell.exe", "-NoProfile", "-Command",
		"Get-WindowsOptionalFeature -Online | Select-Object FeatureName, State | ConvertTo-Json -Compress")
	if err != nil {
		c.logger.Debug("windows features: Get-WindowsOptionalFeature failed, trying Get-WindowsFeature", "error", err)
		// Fallback: Get-WindowsFeature (Server SKU only).
		out, err = c.runner.Run(ctx, "powershell.exe", "-NoProfile", "-Command",
			"Get-WindowsFeature | Select-Object Name, InstallState | ConvertTo-Json -Compress")
		if err != nil {
			return nil, fmt.Errorf("windows features collector: %w", err)
		}
		return parseWindowsServerFeatures(out)
	}
	return parseWindowsOptionalFeatures(out)
}
