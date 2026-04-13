//go:build windows

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
		return exec.CommandContext(ctx, "shutdown", "/r", "/t", "0").Run()
	case pb.RebootMode_REBOOT_MODE_GRACEFUL:
		args := []string{"/r", "/t", fmt.Sprintf("%d", gracePeriod)}
		if msg != "" {
			args = append(args, "/c", msg)
		}
		return exec.CommandContext(ctx, "shutdown", args...).Run()
	case pb.RebootMode_REBOOT_MODE_DEFERRED:
		return nil
	default:
		return exec.CommandContext(ctx, "shutdown", "/r", "/t", "0").Run()
	}
}
