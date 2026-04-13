package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const defaultRebootRequiredPath = "/var/run/reboot-required"

// aptInstaller implements Installer for Debian/Ubuntu systems.
type aptInstaller struct {
	executor           CommandExecutor
	logger             *slog.Logger
	rebootRequiredPath string
}

func (a *aptInstaller) Name() string { return "apt" }

func (a *aptInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	// Refresh the package cache before installing so that versioned installs
	// (pkg=version) do not fail against a stale index.
	updateCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if _, err := a.executor.Execute(updateCtx, "apt-get", "update"); err != nil {
		a.logger.WarnContext(ctx, "apt patcher: apt-get update failed, proceeding with install anyway", "error", err)
	}

	args := a.buildArgs(pkg, dryRun)
	execResult, err := a.executor.Execute(ctx, "apt-get", args...)
	if err != nil {
		return InstallResult{}, fmt.Errorf("apt install %s: %w", pkg.Name, err)
	}

	// If a versioned install failed, retry without the version pin.
	// Catalog versions often don't match exact apt repo versions.
	if execResult.ExitCode != 0 && pkg.Version != "" && !dryRun {
		a.logger.WarnContext(ctx, "apt patcher: versioned install failed, retrying without version pin",
			"package", pkg.Name, "version", pkg.Version, "exit_code", execResult.ExitCode,
			"stderr", string(execResult.Stderr))
		fallbackTarget := PatchTarget{Name: pkg.Name}
		fallbackArgs := a.buildArgs(fallbackTarget, false)
		fallbackResult, fallbackErr := a.executor.Execute(ctx, "apt-get", fallbackArgs...)
		if fallbackErr != nil {
			return InstallResult{}, fmt.Errorf("apt install %s (fallback): %w", pkg.Name, fallbackErr)
		}
		execResult = fallbackResult
	}

	rebootPath := a.rebootRequiredPath
	if rebootPath == "" {
		rebootPath = defaultRebootRequiredPath
	}

	reboot := false
	if execResult.ExitCode == 0 {
		reboot = fileExists(rebootPath)
	}

	return InstallResult{
		Stdout:         execResult.Stdout,
		Stderr:         execResult.Stderr,
		ExitCode:       execResult.ExitCode,
		RebootRequired: reboot,
	}, nil
}

func (a *aptInstaller) buildArgs(pkg PatchTarget, dryRun bool) []string {
	pkgSpec := pkg.Name
	if pkg.Version != "" {
		pkgSpec = pkg.Name + "=" + pkg.Version
	}

	if dryRun {
		return []string{"install", "--dry-run", pkgSpec}
	}
	return []string{"install", "-y", pkgSpec}
}

// GetCurrentVersion queries dpkg for the currently installed version of a package.
// Returns errNotFound if the package is not installed.
func (a *aptInstaller) GetCurrentVersion(ctx context.Context, packageName string) (string, error) {
	result, err := a.executor.Execute(ctx, "dpkg-query", "--show", "-f", "${Version}", packageName)
	if err != nil {
		return "", fmt.Errorf("get current version of %s: %w", packageName, err)
	}
	if result.ExitCode != 0 {
		return "", errNotFound
	}
	version := strings.TrimSpace(string(result.Stdout))
	if version == "" {
		return "", errNotFound
	}
	return version, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
