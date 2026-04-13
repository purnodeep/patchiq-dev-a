package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// softwareupdateFailureStrings are stderr patterns that indicate a silent failure
// from macOS softwareupdate (exit code 0 but the update was not actually installed).
var softwareupdateFailureStrings = []string{
	"No such update",
	"No updates are available",
	"not eligible",
	"requires restart first",
}

type macosInstaller struct {
	executor CommandExecutor
	logger   *slog.Logger
}

func (m *macosInstaller) Name() string { return "softwareupdate" }

func (m *macosInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	if dryRun {
		execResult, err := m.executor.Execute(ctx, "softwareupdate", "--list")
		if err != nil {
			return InstallResult{}, fmt.Errorf("softwareupdate dry-run: %w", err)
		}
		return InstallResult{
			Stdout:   execResult.Stdout,
			Stderr:   execResult.Stderr,
			ExitCode: execResult.ExitCode,
		}, nil
	}

	execResult, err := m.executor.Execute(ctx, "softwareupdate", "--install", pkg.Name)
	if err != nil {
		return InstallResult{}, fmt.Errorf("softwareupdate install %s: %w", pkg.Name, err)
	}

	reboot := strings.Contains(string(execResult.Stdout), "restart") ||
		strings.Contains(string(execResult.Stderr), "restart")

	// Detect false-positive success: softwareupdate exits 0 but stderr indicates
	// the update was not found or not eligible for installation.
	if execResult.ExitCode == 0 {
		if reason := detectSoftwareupdateFailure(string(execResult.Stderr)); reason != "" {
			m.logger.WarnContext(ctx, "softwareupdate: exit code 0 but stderr indicates failure",
				"package", pkg.Name, "reason", reason)
			execResult.ExitCode = 1
			execResult.Stderr = append(execResult.Stderr, []byte("\nsoftwareupdate: "+reason)...)
		}
	}

	return InstallResult{
		Stdout:         execResult.Stdout,
		Stderr:         execResult.Stderr,
		ExitCode:       execResult.ExitCode,
		RebootRequired: reboot,
	}, nil
}

// detectSoftwareupdateFailure checks stderr for known softwareupdate false-positive
// patterns. Returns the matched reason string, or empty if no failure detected.
func detectSoftwareupdateFailure(stderr string) string {
	lower := strings.ToLower(stderr)
	for _, pattern := range softwareupdateFailureStrings {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return pattern
		}
	}
	return ""
}
