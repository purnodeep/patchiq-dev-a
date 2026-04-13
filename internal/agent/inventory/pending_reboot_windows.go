//go:build windows

package inventory

import (
	"context"
	"log/slog"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"golang.org/x/sys/windows/registry"
)

// rebootChecker abstracts registry key existence checks for testability.
type rebootChecker interface {
	KeyExists(path string) bool
}

// winRebootChecker implements rebootChecker using the Windows registry.
type winRebootChecker struct {
	logger *slog.Logger
}

func (c *winRebootChecker) KeyExists(path string) bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// pendingRebootPaths are the registry keys that indicate a reboot is pending,
// along with a human-readable category for each source.
var pendingRebootPaths = []struct {
	path     string
	category string
}{
	{`SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending`, "CBS"},
	{`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`, "WindowsUpdate"},
	{`SYSTEM\CurrentControlSet\Control\Session Manager\PendingFileRenameOperations`, "FileRename"},
}

// pendingRebootCollector detects if a Windows reboot is pending.
type pendingRebootCollector struct {
	checker rebootChecker
	logger  *slog.Logger
}

func (c *pendingRebootCollector) Name() string { return "pending_reboot" }

func (c *pendingRebootCollector) Collect(_ context.Context) ([]*pb.PackageInfo, error) {
	var pkgs []*pb.PackageInfo
	for _, entry := range pendingRebootPaths {
		if c.checker.KeyExists(entry.path) {
			c.logger.Info("pending reboot detected", "registry_key", entry.path, "category", entry.category)
			pkgs = append(pkgs, &pb.PackageInfo{
				Name:     "REBOOT_PENDING",
				Source:   "system",
				Status:   "pending",
				Category: entry.category,
			})
		}
	}
	return pkgs, nil
}
