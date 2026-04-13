package patcher

import (
	"context"
	"fmt"
	"strings"
)

type homebrewInstaller struct {
	executor CommandExecutor
	brewPath string // absolute path to brew binary
}

func (h *homebrewInstaller) Name() string { return "homebrew" }

func (h *homebrewInstaller) brew() string {
	if h.brewPath != "" {
		return h.brewPath
	}
	return "brew"
}

// brewOwnerUser is implemented in homebrew_unix.go (darwin/linux) and
// homebrew_windows.go (stub).

// executeBrew runs a brew command, dropping to the brew owner if running as root.
// Uses sudo -H and cd ~ to avoid getcwd errors when PWD is root-only.
func (h *homebrewInstaller) executeBrew(ctx context.Context, args ...string) (ExecResult, error) {
	if owner := h.brewOwnerUser(); owner != "" {
		brewCmd := h.brew() + " " + strings.Join(args, " ")
		sudoArgs := []string{"-u", owner, "-H", "--", "sh", "-c", "cd ~ && " + brewCmd}
		return h.executor.Execute(ctx, "sudo", sudoArgs...)
	}
	return h.executor.Execute(ctx, h.brew(), args...)
}

func (h *homebrewInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	args := h.buildArgs(pkg, dryRun)

	execResult, err := h.executeBrew(ctx, args...)
	if err != nil {
		return InstallResult{}, fmt.Errorf("brew upgrade %s: %w", pkg.Name, err)
	}

	return InstallResult{
		Stdout:         execResult.Stdout,
		Stderr:         execResult.Stderr,
		ExitCode:       execResult.ExitCode,
		RebootRequired: false,
	}, nil
}

func (h *homebrewInstaller) buildArgs(pkg PatchTarget, dryRun bool) []string {
	if dryRun {
		return []string{"upgrade", "--dry-run", pkg.Name}
	}
	// Always use "brew upgrade" — versioned formula syntax (pkg@version) only
	// exists for a handful of packages (node@18, python@3.11, etc.) and not for
	// the vast majority like gh, git, curl. Version in PatchTarget is the target
	// version we want, not a formula name suffix.
	return []string{"upgrade", pkg.Name}
}

// GetCurrentVersion returns the installed version of a package via `brew list --versions`.
// Returns ("", nil) if the package is not installed.
func (h *homebrewInstaller) GetCurrentVersion(ctx context.Context, packageName string) (string, error) {
	result, err := h.executeBrew(ctx, "list", "--versions", packageName)
	if err != nil {
		return "", fmt.Errorf("brew list --versions %s: %w", packageName, err)
	}
	output := strings.TrimSpace(string(result.Stdout))
	if output == "" {
		return "", nil
	}
	// Output format: "curl 8.1.0" or "curl 8.1.0 8.0.1" (multiple versions).
	// Return the first (latest) version after the package name.
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return "", nil
	}
	return parts[1], nil
}
