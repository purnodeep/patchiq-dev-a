package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

type homebrewCollector struct {
	runner   commandRunner
	brewPath string // absolute path to brew binary

	mu               sync.RWMutex
	extendedPackages []ExtendedPackageInfo
}

func (c *homebrewCollector) Name() string { return "homebrew" }

// ExtendedPackages returns the most recently collected extended package info.
func (c *homebrewCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.extendedPackages))
	copy(out, c.extendedPackages)
	return out
}

func (c *homebrewCollector) brew() string {
	if c.brewPath != "" {
		return c.brewPath
	}
	return "brew"
}

// brewOwnerUser is implemented in homebrew_unix.go (darwin/linux) and
// homebrew_windows.go (stub). It returns the username that owns the brew
// binary so we can drop privileges when running as root.

// runBrew executes a brew command, dropping to the brew owner if running as root.
// Uses sudo -H and cd ~ to avoid getcwd errors when PWD is root-only.
func (c *homebrewCollector) runBrew(ctx context.Context, args ...string) ([]byte, error) {
	if owner := c.brewOwnerUser(); owner != "" {
		brewCmd := c.brew() + " " + strings.Join(args, " ")
		sudoArgs := []string{"-u", owner, "-H", "--", "sh", "-c", "cd ~ && " + brewCmd}
		return c.runner.Run(ctx, "sudo", sudoArgs...)
	}
	return c.runner.Run(ctx, c.brew(), args...)
}

func (c *homebrewCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	listOut, err := c.runBrew(ctx, "list", "--versions")
	if err != nil {
		return nil, fmt.Errorf("homebrew list: %w", err)
	}

	pkgs := parseBrewList(listOut)

	outdatedOut, err := c.runBrew(ctx, "outdated")
	if err != nil {
		return pkgs, nil // Non-fatal
	}

	pkgs = append(pkgs, parseBrewOutdated(outdatedOut)...)

	// Collect extended info via brew info --json=v2 (best-effort).
	if infoOut, infoErr := c.runBrew(ctx, "info", "--json=v2", "--installed"); infoErr == nil {
		extended := parseBrewInfoJSON(infoOut)
		c.mu.Lock()
		c.extendedPackages = extended
		c.mu.Unlock()
	}

	return pkgs, nil
}

// parseBrewList parses `brew list --versions` output.
// Each line: "<name> <version> [<version>...]" — take first version.
func parseBrewList(data []byte) []*pb.PackageInfo {
	var pkgs []*pb.PackageInfo
	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         fields[0],
			Version:      fields[1],
			Architecture: runtime.GOARCH,
			Source:       "homebrew",
			Status:       "installed",
		})
	}
	return pkgs
}

// parseBrewOutdated parses `brew outdated` output.
// Each line: "<name> (<current>) < <latest>" or "<name> (<current>)"
func parseBrewOutdated(data []byte) []*pb.PackageInfo {
	var pkgs []*pb.PackageInfo
	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		version := strings.Trim(fields[1], "()")
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         name,
			Version:      version,
			Architecture: runtime.GOARCH,
			Source:       "homebrew",
			Status:       "outdated",
		})
	}
	return pkgs
}

// brewInfoV2 is the top-level JSON structure from `brew info --json=v2 --installed`.
type brewInfoV2 struct {
	Formulae []brewFormula `json:"formulae"`
	Casks    []brewCask    `json:"casks"`
}

type brewFormula struct {
	Name       string          `json:"name"`
	FullName   string          `json:"full_name"`
	Desc       string          `json:"desc"`
	Homepage   string          `json:"homepage"`
	License    string          `json:"license"`
	Versions   brewVersions    `json:"versions"`
	Installed  []brewInstalled `json:"installed"`
	Outdated   bool            `json:"outdated"`
	Deprecated bool            `json:"deprecated"`
}

type brewVersions struct {
	Stable string `json:"stable"`
}

type brewInstalled struct {
	Version               string `json:"version"`
	InstalledOnRequest    bool   `json:"installed_on_request"`
	InstalledAsDependency bool   `json:"installed_as_dependency"`
	InstalledTime         int64  `json:"time"`
}

type brewCask struct {
	Token     string   `json:"token"`
	FullToken string   `json:"full_token"`
	Name      []string `json:"name"`
	Desc      string   `json:"desc"`
	Homepage  string   `json:"homepage"`
	Version   string   `json:"version"`
	Installed string   `json:"installed"`
}

// parseBrewInfoJSON parses `brew info --json=v2 --installed` output into
// ExtendedPackageInfo entries.
func parseBrewInfoJSON(data []byte) []ExtendedPackageInfo {
	var info brewInfoV2
	if err := json.Unmarshal(data, &info); err != nil {
		return nil
	}

	var pkgs []ExtendedPackageInfo

	for _, f := range info.Formulae {
		version := f.Versions.Stable
		status := "installed"
		if len(f.Installed) > 0 {
			version = f.Installed[0].Version
		}
		if f.Outdated {
			status = "outdated"
		}

		category := "Application"
		if len(f.Installed) > 0 && f.Installed[0].InstalledAsDependency {
			category = "Library"
		}

		var installDate string
		if len(f.Installed) > 0 && f.Installed[0].InstalledTime > 0 {
			installDate = time.Unix(f.Installed[0].InstalledTime, 0).UTC().Format(time.RFC3339)
		}

		sizeKB := brewCellarSizeKB(f.Name, version)

		pkgs = append(pkgs, ExtendedPackageInfo{
			Name:          f.Name,
			Version:       version,
			Architecture:  runtime.GOARCH,
			Source:        "homebrew",
			Status:        status,
			Description:   f.Desc,
			Homepage:      f.Homepage,
			License:       f.License,
			Category:      category,
			InstallDate:   installDate,
			InstalledSize: sizeKB,
		})
	}

	for _, c := range info.Casks {
		name := c.Token
		desc := c.Desc
		if len(c.Name) > 0 {
			desc = c.Name[0]
			if c.Desc != "" {
				desc = c.Name[0] + " — " + c.Desc
			}
		}

		sizeKB := brewCaskSizeKB(c.Token)

		pkgs = append(pkgs, ExtendedPackageInfo{
			Name:          name,
			Version:       c.Version,
			Architecture:  runtime.GOARCH,
			Source:        "homebrew-cask",
			Status:        "installed",
			Description:   desc,
			Homepage:      c.Homepage,
			Category:      "Application",
			InstalledSize: sizeKB,
		})
	}

	return pkgs
}

// brewCellarSizeKB returns the installed size in KB for a Homebrew formula by
// walking its Cellar directory. Returns 0 if the directory cannot be read.
func brewCellarSizeKB(name, version string) int {
	// Homebrew installs to /opt/homebrew/Cellar (Apple Silicon) or
	// /usr/local/Cellar (Intel).
	for _, prefix := range []string{"/opt/homebrew/Cellar", "/usr/local/Cellar"} {
		dir := filepath.Join(prefix, name, version)
		if size := dirSizeBytes(dir); size > 0 {
			return int(size / 1024)
		}
	}
	return 0
}

// brewCaskSizeKB returns the installed size in KB for a Homebrew cask by
// checking /Applications/<token>.app or the Caskroom directory.
func brewCaskSizeKB(token string) int {
	// Many casks install as /Applications/<Token>.app (title-cased token).
	// Try common patterns.
	// Cask tokens are lowercase-kebab (e.g. "visual-studio-code"). The .app
	// name often uses title case with spaces (e.g. "Visual Studio Code.app").
	titleName := strings.ReplaceAll(token, "-", " ")
	if len(titleName) > 0 {
		titleName = strings.ToUpper(titleName[:1]) + titleName[1:]
	}
	for _, pattern := range []string{
		filepath.Join("/Applications", token+".app"),
		filepath.Join("/Applications", titleName+".app"),
	} {
		if size := dirSizeBytes(pattern); size > 0 {
			return int(size / 1024)
		}
	}

	// Fallback: check Caskroom.
	for _, prefix := range []string{"/opt/homebrew/Caskroom", "/usr/local/Caskroom"} {
		dir := filepath.Join(prefix, token)
		if size := dirSizeBytes(dir); size > 0 {
			return int(size / 1024)
		}
	}
	return 0
}

// dirSizeBytes returns the total size of all files under dir. Returns 0 on error.
func dirSizeBytes(dir string) int64 {
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if info, infoErr := d.Info(); infoErr == nil {
				total += info.Size()
			}
		}
		return nil
	})
	if err != nil {
		return 0
	}
	return total
}
