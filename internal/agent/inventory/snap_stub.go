//go:build !linux

package inventory

import (
	"context"
	"fmt"
	"log/slog"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// snapCollector is a no-op stub on non-Linux platforms. The real implementation
// lives in snap.go (linux-only).
type snapCollector struct {
	runner commandRunner
	logger *slog.Logger
}

func (c *snapCollector) Name() string { return "snap" }

func (c *snapCollector) Collect(_ context.Context) ([]*pb.PackageInfo, error) {
	return nil, fmt.Errorf("snap collector: not supported on this platform")
}

func (c *snapCollector) ExtendedPackages() []ExtendedPackageInfo {
	return nil
}
