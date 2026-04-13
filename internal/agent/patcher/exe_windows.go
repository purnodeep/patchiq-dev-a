//go:build windows

package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// exeInstaller installs third-party Windows software distributed as .exe files.
type exeInstaller struct {
	executor   CommandExecutor
	logger     *slog.Logger
	silentArgs string
}

func (e *exeInstaller) Name() string { return "exe" }

// Install runs the .exe installer with optional silent args.
// A non-zero exit code is reported via InstallResult.ExitCode with a nil error
// unless the execution itself fails (binary not found, context cancelled, etc.).
func (e *exeInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	if dryRun {
		slog.InfoContext(ctx, "dry-run: would install EXE package", "package", pkg.Name)
		return InstallResult{
			Stdout: []byte(fmt.Sprintf("dry-run: would install EXE package %s", pkg.Name)),
		}, nil
	}

	// Validate that the executable exists before attempting to run it.
	if _, err := os.Stat(pkg.Name); err != nil {
		return InstallResult{}, fmt.Errorf("exe install %s: binary not found: %w", pkg.Name, err)
	}

	var args []string
	if e.silentArgs != "" {
		args = splitArgs(e.silentArgs)
	} else if detected := detectInstallerType(pkg.Name); detected != "" {
		args = splitArgs(detected)
		e.logger.InfoContext(ctx, "exe: auto-detected installer type", "package", pkg.Name, "args", detected)
	}

	execResult, err := e.executor.Execute(ctx, pkg.Name, args...)
	if err != nil {
		return InstallResult{}, fmt.Errorf("exe install %s: %w", pkg.Name, err)
	}

	return InstallResult{
		Stdout:         execResult.Stdout,
		Stderr:         execResult.Stderr,
		ExitCode:       execResult.ExitCode,
		RebootRequired: exeRebootRequired(execResult.ExitCode),
	}, nil
}

// exeRebootRequired returns true for exit codes that indicate a reboot is needed.
func exeRebootRequired(exitCode int) bool {
	return exitCode == 3010 || exitCode == 1641
}

// splitArgs splits a space-separated argument string, respecting double-quoted tokens.
func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false

	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
