//go:build windows

package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// defenderExclude adds a path to Windows Defender's exclusion list.
// This prevents Defender from quarantining downloaded patch binaries.
// The function variable allows tests to override the implementation.
var defenderExclude = defenderAddExclusion
var defenderRemoveExclude = defenderRemoveExclusion

func defenderAddExclusion(ctx context.Context, executor CommandExecutor, path string) error {
	escapedPath := strings.ReplaceAll(path, "'", "''")
	psCmd := fmt.Sprintf("Add-MpPreference -ExclusionPath '%s'", escapedPath)
	result, err := executor.Execute(ctx, "powershell.exe", "-NoProfile", "-Command", psCmd)
	if err != nil {
		slog.WarnContext(ctx, "defender: failed to add exclusion", "path", path, "error", err)
		return fmt.Errorf("defender add exclusion %s: %w", path, err)
	}
	if result.ExitCode != 0 {
		slog.WarnContext(ctx, "defender: add exclusion returned non-zero",
			"path", path, "exit_code", result.ExitCode, "stderr", string(result.Stderr))
		// Non-fatal: proceed even if exclusion fails (may not have Defender admin rights).
		return nil
	}
	slog.InfoContext(ctx, "defender: added exclusion", "path", path)
	return nil
}

func defenderRemoveExclusion(ctx context.Context, executor CommandExecutor, path string) error {
	escapedPath := strings.ReplaceAll(path, "'", "''")
	psCmd := fmt.Sprintf("Remove-MpPreference -ExclusionPath '%s'", escapedPath)
	result, err := executor.Execute(ctx, "powershell.exe", "-NoProfile", "-Command", psCmd)
	if err != nil {
		slog.WarnContext(ctx, "defender: failed to remove exclusion", "path", path, "error", err)
		return nil // Non-fatal.
	}
	if result.ExitCode != 0 {
		slog.DebugContext(ctx, "defender: remove exclusion returned non-zero",
			"path", path, "exit_code", result.ExitCode)
		return nil
	}
	slog.DebugContext(ctx, "defender: removed exclusion", "path", path)
	return nil
}
