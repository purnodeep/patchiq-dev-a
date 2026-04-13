//go:build windows

package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// wuaInstalledCollector collects currently installed Windows updates via COM IUpdateSearcher.
type wuaInstalledCollector struct {
	searcher updateSearcher
	logger   *slog.Logger
	mu       sync.RWMutex
	lastPkgs []ExtendedPackageInfo
}

func (c *wuaInstalledCollector) Name() string { return "wua_installed" }

func (c *wuaInstalledCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	updates, err := c.searcher.Search(ctx, "IsInstalled=1")
	if err != nil {
		return nil, fmt.Errorf("wua installed collector: %w", err)
	}
	pkgs := mapWindowsUpdates(updates)
	// Override source to distinguish from available updates.
	for _, p := range pkgs {
		p.Source = "wua_installed"
	}

	next := make([]ExtendedPackageInfo, 0, len(pkgs))
	for _, p := range pkgs {
		next = append(next, ExtendedPackageInfo{
			Name:        p.Name,
			Version:     p.Version,
			Source:      p.Source,
			Category:    p.Category,
			Status:      "installed",
			Description: p.KbArticle,
		})
	}
	c.mu.Lock()
	c.lastPkgs = next
	c.mu.Unlock()
	return pkgs, nil
}

func (c *wuaInstalledCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.lastPkgs))
	copy(out, c.lastPkgs)
	return out
}
