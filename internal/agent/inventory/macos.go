package inventory

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// macosCollector collects available updates via softwareupdate --list.
type macosCollector struct {
	runner commandRunner

	mu               sync.RWMutex
	extendedPackages []ExtendedPackageInfo
}

func (c *macosCollector) Name() string {
	return "softwareupdate"
}

// ExtendedPackages returns the most recently collected extended package info.
func (c *macosCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.extendedPackages))
	copy(out, c.extendedPackages)
	return out
}

func (c *macosCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	// softwareupdate --list contacts Apple servers and can take minutes; cap at 120s.
	collectCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	out, err := c.runner.Run(collectCtx, "softwareupdate", "--list")
	if err != nil {
		return nil, fmt.Errorf("softwareupdate collector: %w", err)
	}
	pkgs := parseSoftwareUpdate(out)

	// Build extended info from the same parsed data, including size from detail lines.
	sizeMap := parseSoftwareUpdateSizes(out)
	var extended []ExtendedPackageInfo
	for _, p := range pkgs {
		ext := ExtendedPackageInfo{
			Name:         p.Name,
			Version:      p.Version,
			Architecture: p.Architecture,
			Source:       "softwareupdate",
			Status:       p.Status,
			Category:     "System",
		}
		if sizeKB, ok := sizeMap[p.Name]; ok {
			ext.InstalledSize = sizeKB
		}
		extended = append(extended, ext)
	}
	c.mu.Lock()
	c.extendedPackages = extended
	c.mu.Unlock()

	return pkgs, nil
}

// parseSoftwareUpdate parses the output of `softwareupdate --list` into PackageInfo slices.
//
// The expected format has pairs of lines for each update:
//
//   - Label: macOS Ventura 13.6.1-13.6.1
//     Title: macOS Ventura 13.6.1, Version: 13.6.1, Size: 11283480KiB, Recommended: YES, Action: restart,
func parseSoftwareUpdate(data []byte) []*pb.PackageInfo {
	var pkgs []*pb.PackageInfo

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		// Look for label lines: "* Label: <label>"
		if !strings.HasPrefix(line, "* Label: ") {
			continue
		}
		label := strings.TrimPrefix(line, "* Label: ")

		// Next line should be the tab-indented detail line.
		if !scanner.Scan() {
			break
		}
		detailLine := strings.TrimSpace(scanner.Text())

		title, version := parseDetailLine(detailLine)
		if title == "" {
			continue
		}

		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         title,
			Version:      version,
			Source:       "softwareupdate",
			Status:       "available",
			Release:      label,
			Architecture: runtime.GOARCH,
		})
	}

	return pkgs
}

// parseDetailLine parses a softwareupdate detail line into title and version.
// Example input: "Title: macOS Ventura 13.6.1, Version: 13.6.1, Size: 11283480KiB, Recommended: YES, Action: restart,"
func parseDetailLine(line string) (title, version string) {
	fields := parseDetailFields(line)
	return fields["Title"], fields["Version"]
}

// parseDetailFields splits a softwareupdate detail line into key-value pairs.
func parseDetailFields(line string) map[string]string {
	fields := make(map[string]string)
	for _, part := range strings.Split(line, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) == 2 {
			fields[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return fields
}

// parseSoftwareUpdateSizes extracts Size (in KB) per title from softwareupdate --list output.
func parseSoftwareUpdateSizes(data []byte) map[string]int {
	sizes := make(map[string]int)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "* Label: ") {
			continue
		}
		if !scanner.Scan() {
			break
		}
		detailLine := strings.TrimSpace(scanner.Text())
		fields := parseDetailFields(detailLine)
		title := fields["Title"]
		sizeStr := fields["Size"]
		if title == "" || sizeStr == "" {
			continue
		}
		// Size format: "11283480KiB" or "11283480K"
		sizeStr = strings.TrimSuffix(sizeStr, "iB")
		sizeStr = strings.TrimSuffix(sizeStr, "K")
		if v, err := strconv.Atoi(sizeStr); err == nil {
			sizes[title] = v
		}
	}
	return sizes
}
