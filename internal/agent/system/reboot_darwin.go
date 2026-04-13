//go:build darwin

package system

import (
	"context"
	"fmt"
	"os/exec"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

func platformReboot(ctx context.Context, mode pb.RebootMode, gracePeriod int32, msg string) error {
	switch mode {
	case pb.RebootMode_REBOOT_MODE_IMMEDIATE:
		return exec.CommandContext(ctx, "shutdown", "-r", "now").Run()
	case pb.RebootMode_REBOOT_MODE_GRACEFUL:
		minutes := gracePeriod / 60
		if minutes < 1 {
			minutes = 1
		}
		return exec.CommandContext(ctx, "shutdown", "-r", fmt.Sprintf("+%d", minutes)).Run()
	case pb.RebootMode_REBOOT_MODE_DEFERRED:
		return nil
	default:
		return exec.CommandContext(ctx, "shutdown", "-r", "now").Run()
	}
}
