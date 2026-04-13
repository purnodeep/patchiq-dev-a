package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// msixInstaller installs MSIX/AppX packages via PowerShell Add-AppxPackage.
type msixInstaller struct {
	executor CommandExecutor
}

func (m *msixInstaller) Name() string { return "msix" }

func (m *msixInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	if dryRun {
		slog.InfoContext(ctx, "dry-run: would install MSIX package", "package", pkg.Name)
		return InstallResult{
			Stdout: []byte(fmt.Sprintf("dry-run: would install MSIX package %s", pkg.Name)),
		}, nil
	}

	// MSIX/AppX installation requires administrator privileges.
	if err := checkAdmin(); err != nil {
		return InstallResult{}, fmt.Errorf("msix install %s: %w", pkg.Name, err)
	}

	// Escape single quotes in the package path (PowerShell escapes ' as '') and
	// wrap in single quotes so the path is treated as a string literal. This
	// prevents injection even if pkg.Name contains ; | $() or other metacharacters,
	// because PowerShell does not interpret content inside single-quoted strings.
	escapedName := strings.ReplaceAll(pkg.Name, "'", "''")
	psCommand := fmt.Sprintf("Add-AppxPackage -Path '%s'", escapedName)
	args := []string{"-NoProfile", "-Command", psCommand}

	execResult, err := m.executor.Execute(ctx, "powershell.exe", args...)
	if err != nil {
		return InstallResult{}, fmt.Errorf("msix install %s: %w", pkg.Name, err)
	}

	return InstallResult{
		Stdout:   execResult.Stdout,
		Stderr:   execResult.Stderr,
		ExitCode: execResult.ExitCode,
	}, nil
}
