package inventory

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// packageCollector is the internal interface for OS-specific package collectors.
type packageCollector interface {
	Name() string
	Collect(ctx context.Context) ([]*pb.PackageInfo, error)
}

// commandRunner abstracts command execution for testability.
type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// execRunner implements commandRunner using os/exec.
type execRunner struct{}

func (r *execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run %s: %s: %w", name, stderr.String(), err)
	}
	return stdout.Bytes(), nil
}

// collectorDetectorFunc probes for a specific collector and returns it, or nil if unavailable.
type collectorDetectorFunc func() packageCollector

// platformCollectorDetectors is populated by platform-specific init() functions
// (e.g., detect_windows.go). On platforms with no additional detectors, this remains nil.
var platformCollectorDetectors []collectorDetectorFunc

// detectorDeps holds injectable dependencies for OS detection.
type detectorDeps struct {
	dpkgStatusPath    string
	rpmLookPath       func() (string, error)
	brewLookPath      func() (string, error)
	snapLookPath      func() (string, error)
	logger            *slog.Logger
	goos              string
	platformDetectors []collectorDetectorFunc
}

// defaultDetectorDeps returns production defaults for OS detection.
func defaultDetectorDeps() detectorDeps {
	return detectorDeps{
		dpkgStatusPath: "/var/lib/dpkg/status",
		rpmLookPath:    func() (string, error) { return exec.LookPath("rpm") },
		brewLookPath: func() (string, error) {
			if path, err := exec.LookPath("brew"); err == nil {
				return path, nil
			}
			// Fallback for LaunchDaemon/root where /opt/homebrew isn't in PATH.
			for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
				if _, err := os.Stat(p); err == nil {
					return p, nil
				}
			}
			return "", fmt.Errorf("brew not found")
		},
		snapLookPath:      func() (string, error) { return exec.LookPath("snap") },
		goos:              runtime.GOOS,
		platformDetectors: platformCollectorDetectors,
	}
}

// detectCollectors probes the system and returns available package collectors.
func detectCollectors(deps detectorDeps) []packageCollector {
	var collectors []packageCollector

	if _, err := os.Stat(deps.dpkgStatusPath); err == nil {
		collectors = append(collectors, &aptCollector{
			statusPath: deps.dpkgStatusPath,
			runner:     &execRunner{},
			logger:     deps.logger,
		})
	}

	if _, err := deps.rpmLookPath(); err == nil {
		collectors = append(collectors, &rpmCollector{runner: &execRunner{}})
	}

	if deps.goos == "darwin" {
		collectors = append(collectors, &macosCollector{runner: &execRunner{}})
	}

	if brewPath, err := deps.brewLookPath(); err == nil {
		collectors = append(collectors, &homebrewCollector{runner: &execRunner{}, brewPath: brewPath})
	}

	if deps.snapLookPath != nil {
		if _, err := deps.snapLookPath(); err == nil {
			collectors = append(collectors, &snapCollector{
				runner: &execRunner{},
				logger: deps.logger,
			})
		}
	}

	for _, detect := range deps.platformDetectors {
		if c := detect(); c != nil {
			collectors = append(collectors, c)
		}
	}

	return collectors
}
