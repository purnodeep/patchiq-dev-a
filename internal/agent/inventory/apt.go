package inventory

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// aptCollector collects installed packages from dpkg-query (preferred) or
// dpkg/status file (fallback). It also gathers extended metadata (install
// dates, licenses) that the gRPC proto does not carry.
type aptCollector struct {
	statusPath string
	runner     commandRunner
	logger     *slog.Logger

	mu               sync.RWMutex
	extendedPackages []ExtendedPackageInfo
}

// ExtendedPackages returns the most recently collected extended package info.
func (c *aptCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.extendedPackages))
	copy(out, c.extendedPackages)
	return out
}

func (c *aptCollector) Name() string {
	return "apt"
}

func (c *aptCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	// Try dpkg-query first for richer data.
	if c.runner != nil {
		pkgs, extended, err := c.collectViaDpkgQuery(ctx)
		if err == nil {
			c.mu.Lock()
			c.extendedPackages = extended
			c.mu.Unlock()
			return pkgs, nil
		}
		if c.logger != nil {
			c.logger.Warn("dpkg-query failed, falling back to status file", "error", err)
		}
	}

	// Fallback: parse dpkg/status file directly (original method).
	return c.collectViaStatusFile()
}

// collectViaStatusFile is the original collection method that reads the dpkg
// status file directly. Used as fallback when dpkg-query is unavailable.
func (c *aptCollector) collectViaStatusFile() ([]*pb.PackageInfo, error) {
	f, err := os.Open(c.statusPath)
	if err != nil {
		return nil, fmt.Errorf("apt collect: open %s: %w", c.statusPath, err)
	}
	defer f.Close()

	pkgs, err := parseAPTStatus(f)
	if err != nil {
		return nil, fmt.Errorf("apt collect: parse %s: %w", c.statusPath, err)
	}
	return pkgs, nil
}

// collectViaDpkgQuery runs dpkg-query to get richer package metadata.
func (c *aptCollector) collectViaDpkgQuery(ctx context.Context) ([]*pb.PackageInfo, []ExtendedPackageInfo, error) {
	out, err := c.runner.Run(ctx, "dpkg-query", "--show", "-f",
		"${Package}\t${Version}\t${Architecture}\t${Installed-Size}\t${Maintainer}\t${Section}\t${Homepage}\t${db:Status-Abbrev}\n")
	if err != nil {
		return nil, nil, fmt.Errorf("apt collect dpkg-query: %w", err)
	}

	var pkgs []*pb.PackageInfo
	var extended []ExtendedPackageInfo

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 8 {
			continue
		}

		name := fields[0]
		version := fields[1]
		arch := fields[2]
		installedSizeStr := fields[3]
		maintainer := fields[4]
		section := fields[5]
		homepage := fields[6]
		statusAbbrev := fields[7]

		// db:Status-Abbrev format: "ii " means installed, "rc " means removed/config-files.
		// Only include packages where first two chars are "ii" (desired/installed).
		if len(statusAbbrev) < 2 || statusAbbrev[:2] != "ii" {
			continue
		}

		installedSize, _ := strconv.Atoi(strings.TrimSpace(installedSizeStr))

		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         name,
			Version:      version,
			Architecture: arch,
			Source:       "apt",
			Status:       "install ok installed",
		})

		ext := ExtendedPackageInfo{
			Name:          name,
			Version:       version,
			Architecture:  arch,
			Source:        "apt",
			Status:        "install ok installed",
			InstalledSize: installedSize,
			Maintainer:    maintainer,
			Section:       section,
			Homepage:      homepage,
		}

		ext.InstallDate = readInstallDate(name)
		ext.License = readLicense(name)
		ext.Category = ClassifyPackage(section, name)

		extended = append(extended, ext)
	}

	return pkgs, extended, nil
}

// readInstallDate returns the modification time of the dpkg list file for the
// given package, formatted as RFC 3339. Returns empty string on any error.
func readInstallDate(pkg string) string {
	listPath := filepath.Join("/var/lib/dpkg/info", pkg+".list")
	info, err := os.Stat(listPath)
	if err != nil {
		return ""
	}
	return info.ModTime().UTC().Format(time.RFC3339)
}

// readLicense attempts to extract a license identifier from the package's
// copyright file. Returns the first line containing "License:", or empty
// string if not found. This is best-effort and does not fail.
func readLicense(pkg string) string {
	copyrightPath := filepath.Join("/usr/share/doc", pkg, "copyright")
	f, err := os.Open(copyrightPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "License:"); idx >= 0 {
			return strings.TrimSpace(line[idx+len("License:"):])
		}
	}
	return ""
}

// parseAPTStatus parses RFC 822-style paragraphs from a dpkg/status file.
// It extracts Package, Version, Architecture, and Status fields.
// Only packages where the Status third word is "installed" are included.
func parseAPTStatus(r io.Reader) ([]*pb.PackageInfo, error) {
	var pkgs []*pb.PackageInfo

	scanner := bufio.NewScanner(r)

	var (
		name    string
		version string
		arch    string
		status  string
	)

	flush := func() {
		if name == "" {
			return
		}
		if isInstalled(status) {
			pkgs = append(pkgs, &pb.PackageInfo{
				Name:         name,
				Version:      version,
				Architecture: arch,
				Source:       "apt",
				Status:       status,
			})
		}
		name, version, arch, status = "", "", "", ""
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Blank line separates paragraphs.
		if line == "" {
			flush()
			continue
		}

		// Skip continuation lines (start with space or tab).
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}

		key, val, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}

		switch key {
		case "Package":
			name = val
		case "Version":
			version = val
		case "Architecture":
			arch = val
		case "Status":
			status = val
		}
	}

	// Flush the last paragraph.
	flush()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan dpkg status: %w", err)
	}

	return pkgs, nil
}

// isInstalled returns true if the dpkg status string's third word is "installed".
// The status format is "want flag status" (e.g., "install ok installed").
func isInstalled(status string) bool {
	fields := strings.Fields(status)
	return len(fields) >= 3 && fields[2] == "installed"
}
