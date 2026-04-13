//go:build windows

package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"golang.org/x/sys/windows/registry"
)

type registryEntry struct {
	DisplayName    string
	DisplayVersion string
	Publisher      string
	InstallDate    string
	Is64Bit        bool
}

type registryReaderIface interface {
	ReadUninstallKeys() ([]registryEntry, error)
}

type winRegistryReader struct {
	logger *slog.Logger
}

func (r *winRegistryReader) ReadUninstallKeys() ([]registryEntry, error) {
	paths := []struct {
		key     registry.Key
		path    string
		is64Bit bool
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, true},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, false},
	}

	var entries []registryEntry
	for _, p := range paths {
		key, err := registry.OpenKey(p.key, p.path, registry.READ)
		if err != nil {
			r.logger.Warn("skip registry path", "path", p.path, "error", err)
			continue
		}

		subkeys, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			r.logger.Warn("read subkeys failed", "path", p.path, "error", err)
			continue
		}

		for _, subkeyName := range subkeys {
			subkey, err := registry.OpenKey(p.key, p.path+`\`+subkeyName, registry.READ)
			if err != nil {
				continue
			}

			entry := registryEntry{Is64Bit: p.is64Bit}
			entry.DisplayName, _, _ = subkey.GetStringValue("DisplayName")
			entry.DisplayVersion, _, _ = subkey.GetStringValue("DisplayVersion")
			entry.Publisher, _, _ = subkey.GetStringValue("Publisher")
			entry.InstallDate, _, _ = subkey.GetStringValue("InstallDate")
			subkey.Close()

			entries = append(entries, entry)
		}
	}

	return entries, nil
}

type registryCollector struct {
	reader   registryReaderIface
	logger   *slog.Logger
	mu       sync.RWMutex
	lastPkgs []ExtendedPackageInfo
}

// ExtendedPackages implements the extendedCollector interface for the local agent API.
func (c *registryCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.lastPkgs))
	copy(out, c.lastPkgs)
	return out
}

func (c *registryCollector) Name() string { return "registry" }

func (c *registryCollector) Collect(_ context.Context) ([]*pb.PackageInfo, error) {
	entries, err := c.reader.ReadUninstallKeys()
	if err != nil {
		return nil, fmt.Errorf("registry collector: %w", err)
	}

	seen := make(map[string]struct{})
	var pkgs []*pb.PackageInfo
	extended := make([]ExtendedPackageInfo, 0, len(entries))

	for _, e := range entries {
		if e.DisplayName == "" {
			continue
		}
		dedupKey := e.DisplayName + "|" + e.DisplayVersion
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}

		arch := "x86"
		if e.Is64Bit {
			arch = "x64"
		}
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         e.DisplayName,
			Version:      e.DisplayVersion,
			Architecture: arch,
			Source:       "registry",
			Publisher:    e.Publisher,
			InstallDate:  e.InstallDate,
		})
		extended = append(extended, ExtendedPackageInfo{
			Name:         e.DisplayName,
			Version:      e.DisplayVersion,
			Architecture: arch,
			Source:       "registry",
			Category:     "Application",
			InstallDate:  e.InstallDate,
		})
	}

	c.mu.Lock()
	c.lastPkgs = extended
	c.mu.Unlock()
	return pkgs, nil
}
