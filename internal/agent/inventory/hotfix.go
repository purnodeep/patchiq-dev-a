//go:build windows

package inventory

import (
	"context"
	"fmt"
	"sync"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// hotfixCollector collects installed hotfixes via Get-HotFix.
type hotfixCollector struct {
	runner   commandRunner
	mu       sync.RWMutex
	lastPkgs []ExtendedPackageInfo
}

func (c *hotfixCollector) Name() string { return "hotfix" }

func (c *hotfixCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	out, err := c.runner.Run(ctx, "powershell.exe", "-NoProfile", "-Command",
		"Get-HotFix | ConvertTo-Json")
	if err != nil {
		return nil, fmt.Errorf("hotfix collector: %w", err)
	}
	pkgs, err := parseHotFixOutput(out)
	if err != nil {
		return nil, fmt.Errorf("hotfix collector: %w", err)
	}

	next := make([]ExtendedPackageInfo, 0, len(pkgs))
	for _, p := range pkgs {
		next = append(next, ExtendedPackageInfo{
			Name:        p.Name,
			Version:     p.Version,
			Source:      "hotfix",
			Status:      p.Status,
			InstallDate: p.InstallDate,
			Category:    "System",
		})
	}
	c.mu.Lock()
	c.lastPkgs = next
	c.mu.Unlock()
	return pkgs, nil
}

func (c *hotfixCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.lastPkgs))
	copy(out, c.lastPkgs)
	return out
}
