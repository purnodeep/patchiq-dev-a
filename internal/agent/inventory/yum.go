package inventory

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// rpmCollector collects installed packages via rpm -qa.
type rpmCollector struct {
	runner commandRunner

	mu               sync.RWMutex
	extendedPackages []ExtendedPackageInfo
}

func (c *rpmCollector) Name() string {
	return "rpm"
}

// ExtendedPackages returns the most recently collected extended package info.
func (c *rpmCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.extendedPackages))
	copy(out, c.extendedPackages)
	return out
}

func (c *rpmCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	// Use extended queryformat that includes install time and size.
	out, err := c.runner.Run(ctx, "rpm", "-qa", "--queryformat",
		`%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\t%{INSTALLTIME}\t%{SIZE}\n`)
	if err != nil {
		return nil, fmt.Errorf("rpm collector: %w", err)
	}
	pkgs, extended := parseRPMOutputExtended(out)

	c.mu.Lock()
	c.extendedPackages = extended
	c.mu.Unlock()

	return pkgs, nil
}

// parseRPMOutput parses tab-separated rpm -qa output into PackageInfo slices.
// Each line must have exactly 4 tab-separated fields: NAME, VERSION, RELEASE, ARCH.
// Empty lines and lines with an incorrect number of fields are skipped.
func parseRPMOutput(data []byte) []*pb.PackageInfo {
	var pkgs []*pb.PackageInfo

	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}

		fields := strings.Split(trimmed, "\t")
		if len(fields) != 4 {
			continue
		}

		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         fields[0],
			Version:      fields[1],
			Release:      fields[2],
			Architecture: fields[3],
			Source:       "rpm",
		})
	}

	return pkgs
}

// parseRPMOutputExtended parses the extended rpm -qa output that includes
// INSTALLTIME and SIZE fields (6 tab-separated fields per line).
// It returns both proto PackageInfo and ExtendedPackageInfo slices.
func parseRPMOutputExtended(data []byte) ([]*pb.PackageInfo, []ExtendedPackageInfo) {
	var pkgs []*pb.PackageInfo
	var extended []ExtendedPackageInfo

	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}

		fields := strings.Split(trimmed, "\t")
		// Accept 4 fields (legacy) or 6 fields (extended).
		if len(fields) != 4 && len(fields) != 6 {
			continue
		}

		pkg := &pb.PackageInfo{
			Name:         fields[0],
			Version:      fields[1],
			Release:      fields[2],
			Architecture: fields[3],
			Source:       "rpm",
		}
		pkgs = append(pkgs, pkg)

		ext := ExtendedPackageInfo{
			Name:         fields[0],
			Version:      fields[1],
			Architecture: fields[3],
			Source:       "rpm",
			Status:       "installed",
		}

		if len(fields) == 6 {
			if installEpoch, err := strconv.ParseInt(fields[4], 10, 64); err == nil && installEpoch > 0 {
				ext.InstallDate = time.Unix(installEpoch, 0).UTC().Format(time.RFC3339)
			}
			if sizeBytes, err := strconv.ParseInt(fields[5], 10, 64); err == nil && sizeBytes > 0 {
				ext.InstalledSize = int(sizeBytes / 1024) // convert bytes to KB
			}
		}

		extended = append(extended, ext)
	}

	return pkgs, extended
}
