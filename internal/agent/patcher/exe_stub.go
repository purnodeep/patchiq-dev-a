//go:build !windows

package patcher

import (
	"context"
	"log/slog"
)

// exeInstaller is a stub on non-Windows platforms. It is never instantiated at
// runtime because detectEXEInstaller is only registered on Windows, but the type
// must exist so that the silent_args dispatch logic in patcher.go compiles on all
// platforms.
type exeInstaller struct {
	executor   CommandExecutor
	logger     *slog.Logger
	silentArgs string
}

func (e *exeInstaller) Name() string { return "exe" }

func (e *exeInstaller) Install(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
	return InstallResult{}, nil
}
