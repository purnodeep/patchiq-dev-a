//go:build darwin

package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// CollectServices collects launchd service information on macOS.
// It combines user-level services from `launchctl list` with system daemons
// discovered from plist files in LaunchDaemons and LaunchAgents directories.
func CollectServices(ctx context.Context, logger *slog.Logger) ([]ServiceInfo, error) {
	runner := &execRunner{}

	// User-level services from launchctl list.
	out, err := runner.Run(ctx, "launchctl", "list")
	if err != nil {
		return nil, fmt.Errorf("collect services: launchctl list: %w", err)
	}

	services := parseLaunchctlList(string(out))
	loadedSet := make(map[string]bool, len(services))
	for _, s := range services {
		loadedSet[s.Name] = true
	}

	// System daemons from plist directories (doesn't require root).
	plistDirs := []string{
		"/System/Library/LaunchDaemons",
		"/Library/LaunchDaemons",
		"/Library/LaunchAgents",
	}
	// User-level LaunchAgents.
	if home, err := os.UserHomeDir(); err == nil {
		plistDirs = append(plistDirs, filepath.Join(home, "Library", "LaunchAgents"))
	}

	for _, dir := range plistDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".plist") {
				continue
			}
			label := strings.TrimSuffix(e.Name(), ".plist")
			if loadedSet[label] {
				continue // Already captured by launchctl list.
			}
			loadedSet[label] = true
			services = append(services, ServiceInfo{
				Name:        label,
				Description: "",
				LoadState:   "installed",
				ActiveState: "inactive",
				SubState:    "not loaded",
				Enabled:     true,
			})
		}
	}

	for i := range services {
		services[i].Category = categorizeDarwinService(services[i].Name)
	}

	if logger != nil {
		logger.Info("collected launchd services", "count", len(services))
	}

	return services, nil
}
