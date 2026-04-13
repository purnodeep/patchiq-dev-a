package patcher

import (
	"context"
	"fmt"
	"log/slog"
)

// yumInstaller installs packages via yum or dnf.
type yumInstaller struct {
	executor CommandExecutor
	logger   *slog.Logger
	binary   string // "yum" or "dnf"
}

func (y *yumInstaller) Name() string { return y.binary }

func (y *yumInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	args := y.buildArgs(pkg, dryRun)

	execResult, err := y.executor.Execute(ctx, y.binary, args...)
	if err != nil {
		return InstallResult{}, fmt.Errorf("%s install %s: %w", y.binary, pkg.Name, err)
	}

	reboot := false
	if execResult.ExitCode == 0 {
		reboot = y.checkRebootRequired(ctx)
	}

	return InstallResult{
		Stdout:         execResult.Stdout,
		Stderr:         execResult.Stderr,
		ExitCode:       execResult.ExitCode,
		RebootRequired: reboot,
	}, nil
}

func (y *yumInstaller) buildArgs(pkg PatchTarget, dryRun bool) []string {
	pkgSpec := pkg.Name
	if pkg.Version != "" {
		pkgSpec = pkg.Name + "-" + pkg.Version
	}

	if dryRun {
		return []string{"install", "--assumeno", pkgSpec}
	}
	return []string{"install", "-y", pkgSpec}
}

// checkRebootRequired runs needs-restarting -r. Exit code 1 means reboot needed.
// Returns false if the check itself fails (e.g., needs-restarting not installed).
func (y *yumInstaller) checkRebootRequired(ctx context.Context) bool {
	result, err := y.executor.Execute(ctx, "needs-restarting", "-r")
	if err != nil {
		y.logger.WarnContext(ctx, "patcher: needs-restarting check failed", "error", err)
		return false
	}
	return result.ExitCode == 1
}
