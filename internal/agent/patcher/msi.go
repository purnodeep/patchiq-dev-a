package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// msiInstaller installs MSI packages via msiexec.
type msiInstaller struct {
	executor CommandExecutor
}

func (m *msiInstaller) Name() string { return "msi" }

func (m *msiInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	if dryRun {
		slog.InfoContext(ctx, "dry-run: would install MSI package", "package", pkg.Name)
		return InstallResult{
			Stdout: []byte(fmt.Sprintf("dry-run: would install MSI package %s", pkg.Name)),
		}, nil
	}

	logPath := filepath.Join(os.TempDir(), fmt.Sprintf("patchiq-msi-%d.log", time.Now().UnixMilli()))
	args := []string{"/i", pkg.Name, "/quiet", "/norestart", "/l*v", logPath}

	slog.InfoContext(ctx, "msi: installing package", "package", pkg.Name, "log", logPath)

	execResult, err := m.executor.Execute(ctx, "msiexec", args...)
	if err != nil {
		return InstallResult{}, fmt.Errorf("msi install %s: %w", pkg.Name, err)
	}

	return InstallResult{
		Stdout:         execResult.Stdout,
		Stderr:         execResult.Stderr,
		ExitCode:       execResult.ExitCode,
		RebootRequired: msiRebootRequired(execResult.ExitCode),
	}, nil
}

// msiRebootRequired returns true for MSI exit codes that indicate a reboot is needed.
func msiRebootRequired(exitCode int) bool {
	return exitCode == 1641 || exitCode == 3010
}
