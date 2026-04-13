//go:build linux

package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// snapCollector collects installed snap packages via `snap list`.
type snapCollector struct {
	runner commandRunner
	logger *slog.Logger

	mu               sync.RWMutex
	extendedPackages []ExtendedPackageInfo
}

// ExtendedPackages returns the most recently collected extended snap package info.
func (c *snapCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.extendedPackages))
	copy(out, c.extendedPackages)
	return out
}

func (c *snapCollector) Name() string { return "snap" }

func (c *snapCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	out, err := c.runner.Run(ctx, "snap", "list")
	if err != nil {
		return nil, fmt.Errorf("collect snap packages: %w", err)
	}

	pkgs, extended := parseSnapList(out)

	c.mu.Lock()
	c.extendedPackages = extended
	c.mu.Unlock()

	return pkgs, nil
}

// parseSnapList parses the tabular output of `snap list`.
//
// Example output:
//
//	Name               Version           Rev    Tracking         Publisher   Notes
//	firefox            136.0.4-1         6650   latest/stable    mozilla✓    -
//	core22             20240111          1122   latest/stable    canonical✓  base
func parseSnapList(data []byte) ([]*pb.PackageInfo, []ExtendedPackageInfo) {
	var pkgs []*pb.PackageInfo
	var extended []ExtendedPackageInfo

	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil, nil
	}

	// Parse header to find column positions.
	header := lines[0]
	colStarts := findColumnStarts(header)
	if len(colStarts) < 6 {
		return nil, nil
	}

	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		cols := splitByColumns(line, colStarts)
		if len(cols) < 6 {
			continue
		}

		name := cols[0]
		version := cols[1]
		rev := cols[2]
		tracking := cols[3]
		publisher := cols[4]
		notes := cols[5]

		if name == "" || version == "" {
			continue
		}

		// Clean publisher (remove verification mark).
		publisher = strings.TrimRight(publisher, "\u2713\u2714\u2611\u2705")
		publisher = strings.TrimRight(publisher, "✓")

		status := "installed"
		if strings.Contains(notes, "disabled") {
			status = "disabled"
		}

		pkgs = append(pkgs, &pb.PackageInfo{
			Name:    name,
			Version: version,
			Source:  "snap",
			Status:  status,
			Release: rev,
		})

		ext := ExtendedPackageInfo{
			Name:          name,
			Version:       version,
			Source:        "snap",
			Status:        status,
			SourcePackage: tracking,
			Description:   fmt.Sprintf("publisher: %s, notes: %s", publisher, notes),
			Category:      ClassifyPackage("", name),
		}

		// Best-effort: get install date and size from snap directory.
		snapDir := filepath.Join("/snap", name, "current")
		if info, statErr := os.Lstat(snapDir); statErr == nil {
			ext.InstallDate = info.ModTime().UTC().Format(time.RFC3339)
		}
		if size := dirSizeBytes(filepath.Join("/snap", name, rev)); size > 0 {
			ext.InstalledSize = int(size / 1024)
		}

		extended = append(extended, ext)
	}

	return pkgs, extended
}

// findColumnStarts returns the starting index of each column header word.
func findColumnStarts(header string) []int {
	var starts []int
	inWord := false
	for i, ch := range header {
		if ch != ' ' && !inWord {
			starts = append(starts, i)
			inWord = true
		} else if ch == ' ' {
			inWord = false
		}
	}
	return starts
}

// splitByColumns extracts column values from a line using column start positions.
func splitByColumns(line string, colStarts []int) []string {
	cols := make([]string, len(colStarts))
	for i, start := range colStarts {
		if start >= len(line) {
			break
		}
		end := len(line)
		if i+1 < len(colStarts) && colStarts[i+1] < len(line) {
			end = colStarts[i+1]
		}
		cols[i] = strings.TrimSpace(line[start:end])
	}
	return cols
}
