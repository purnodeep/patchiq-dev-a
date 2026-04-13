package patcher

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
)

var errNotFound = errors.New("not found")

// InstallResult captures the outcome of a single package installation.
type InstallResult struct {
	Stdout         []byte
	Stderr         []byte
	ExitCode       int
	RebootRequired bool
}

// Installer executes OS-specific package installations.
// Install returns a non-nil error only for infrastructure failures (binary not found,
// context cancelled). A non-zero exit code from the package manager is reported via
// InstallResult.ExitCode with a nil error.
type Installer interface {
	Name() string
	Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error)
}

// VersionQuerier is optionally implemented by installers that can query the
// currently installed version of a package. Used for rollback tracking.
type VersionQuerier interface {
	GetCurrentVersion(ctx context.Context, packageName string) (string, error)
}

// PatchTarget identifies a package and version to install.
type PatchTarget struct {
	Name    string
	Version string
}

// installerDetectorFunc probes for a specific installer and returns it, or nil if unavailable.
type installerDetectorFunc func(executor CommandExecutor) Installer

// platformInstallerDetectors is populated by platform-specific init() functions.
var platformInstallerDetectors []installerDetectorFunc

// installerDetectorDeps holds injectable lookups for testing.
type installerDetectorDeps struct {
	aptGetLookPath    func() (string, error)
	dnfLookPath       func() (string, error)
	yumLookPath       func() (string, error)
	brewLookPath      func() (string, error)
	goos              string
	platformDetectors []installerDetectorFunc
}

// defaultInstallerDetectorDeps returns production PATH lookups.
func defaultInstallerDetectorDeps() installerDetectorDeps {
	return installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return exec.LookPath("apt-get") },
		dnfLookPath:    func() (string, error) { return exec.LookPath("dnf") },
		yumLookPath:    func() (string, error) { return exec.LookPath("yum") },
		brewLookPath: func() (string, error) {
			if path, err := exec.LookPath("brew"); err == nil {
				return path, nil
			}
			for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
				if _, serr := os.Stat(p); serr == nil {
					return p, nil
				}
			}
			return "", errors.New("brew not found")
		},
		goos:              runtime.GOOS,
		platformDetectors: platformInstallerDetectors,
	}
}

// detectInstaller probes the system and returns the first appropriate installer.
// Kept as a convenience wrapper around detectInstallers for backward compatibility.
func detectInstaller(deps installerDetectorDeps, executor CommandExecutor, logger *slog.Logger) Installer {
	installers := detectInstallers(deps, executor, logger)
	if len(installers) > 0 {
		return installers[0]
	}
	for _, detect := range deps.platformDetectors {
		if inst := detect(executor); inst != nil {
			return inst
		}
	}
	return nil
}

// detectInstallers probes the system and returns all available installers.
// Priority order: apt-get, dnf/yum, macOS softwareupdate, Homebrew.
func detectInstallers(deps installerDetectorDeps, executor CommandExecutor, logger *slog.Logger) []Installer {
	var installers []Installer

	if _, err := deps.aptGetLookPath(); err == nil {
		installers = append(installers, &aptInstaller{executor: executor, logger: logger, rebootRequiredPath: defaultRebootRequiredPath})
	}
	if _, err := deps.dnfLookPath(); err == nil {
		installers = append(installers, &yumInstaller{executor: executor, logger: logger, binary: "dnf"})
	} else if _, err := deps.yumLookPath(); err == nil {
		installers = append(installers, &yumInstaller{executor: executor, logger: logger, binary: "yum"})
	}

	if deps.goos == "darwin" {
		installers = append(installers, &macosInstaller{executor: executor, logger: logger})
	}

	if brewPath, err := deps.brewLookPath(); err == nil {
		installers = append(installers, &homebrewInstaller{executor: executor, brewPath: brewPath})
	}

	for _, detect := range deps.platformDetectors {
		if inst := detect(executor); inst != nil {
			installers = append(installers, inst)
		}
	}

	return installers
}
