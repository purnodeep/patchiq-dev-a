# Inventory Pipeline Reliability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the Windows agent inventory collection pipeline so that data reliably flows from endpoint collectors through gRPC sync to the server database and is accurately displayed in both the agent and server UIs.

**Architecture:** Six layers of fixes progressing bottom-up: (1) collector detection logging, (2) Windows collectors implement `extendedCollector` for local API, (3) proto `PackageInfo` gains Windows-specific fields, (4) server accepts partial inventory reports, (5) agent persists inventory locally in SQLite, (6) agent status API exposes collection health. Each layer is independently testable and deployable.

**Tech Stack:** Go 1.25, protobuf/buf, SQLite (agent), PostgreSQL (server), React/TypeScript (web-agent)

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/agent/inventory/detect_windows.go` | Add logging to all detector functions |
| Modify | `internal/agent/inventory/registry_windows.go` | Implement `extendedCollector`, add arch/publisher/install_date to PackageInfo |
| Modify | `internal/agent/inventory/wua_parse.go` | Map fields to new proto fields instead of overloading generic ones |
| Modify | `internal/agent/inventory/hotfix_parse.go` | Map install_date to new proto field |
| Modify | `internal/agent/inventory/pending_reboot_windows.go` | Include which registry key triggered reboot |
| Modify | `proto/patchiq/v1/common.proto` | Add `kb_article`, `severity`, `install_date`, `category`, `publisher` to `PackageInfo` |
| Modify | `internal/server/grpc/sync_outbox.go` | Accept partial reports with `collection_errors`; store `collection_errors` |
| Modify | `internal/server/store/inventory.go` | Store new `PackageInfo` fields in bulk insert |
| Modify | `internal/server/store/queries/inventory.sql` | Add new columns to insert/select |
| Create | `internal/server/store/migrations/046_package_windows_fields.sql` | Add `kb_article`, `severity`, `install_date`, `category`, `publisher` to `endpoint_packages`; add `collection_errors` JSONB to `endpoint_inventories` |
| Modify | `internal/agent/store/schema.sql` | Add `inventory_cache` table |
| Modify | `internal/agent/store/db.go` | Add `ApplyMigrations` entries for inventory_cache |
| Create | `internal/agent/store/sqlite_inventory.go` | SQLite inventory cache read/write |
| Create | `internal/agent/store/sqlite_inventory_test.go` | Tests for inventory cache |
| Modify | `internal/agent/inventory/collector.go` | Track last collection time and collector results |
| Modify | `internal/agent/api/inventory_provider.go` | Fall back to SQLite cache when in-memory is empty |
| Modify | `internal/agent/api/status.go` | Add collector health to status response |
| Modify | `internal/agent/runner.go` | Persist inventory to local SQLite after collection |
| Test | `internal/agent/inventory/detect_windows_test.go` | (existing) extend for logging verification |
| Test | `internal/agent/inventory/wua_parse_test.go` | (existing) update for new field mapping |
| Test | `internal/agent/inventory/collector_test.go` | (existing) extend for collection metadata |
| Test | `internal/server/grpc/sync_outbox_test.go` | (existing) extend for partial report acceptance |
| Test | `internal/server/store/inventory_test.go` | (existing) extend for new columns |

---

### Task 1: Add Logging to Windows Collector Detection

**Files:**
- Modify: `internal/agent/inventory/detect_windows.go`

Currently all detector functions silently return `nil` when dependencies are missing. This makes it impossible to diagnose why collectors aren't running.

- [ ] **Step 1: Add slog import and logging to each detector**

The detector functions don't have access to a logger (they're `func() packageCollector`). We need to log using `slog.Default()` since the detection happens during `init()` registration and the functions are called in `detectCollectors` before module logger is available.

```go
//go:build windows

package inventory

import (
	"log/slog"
	"os/exec"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func init() {
	platformCollectorDetectors = []collectorDetectorFunc{
		detectHotFixCollector,
		detectWUACollector,
		detectWUAInstalledCollector,
		detectRegistryCollector,
		detectPendingRebootCollector,
		detectWindowsFeaturesCollector,
	}
}

func detectHotFixCollector() packageCollector {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		slog.Warn("hotfix collector unavailable: powershell.exe not found", "error", err)
		return nil
	}
	return &hotfixCollector{runner: &execRunner{}}
}

func detectWUACollector() packageCollector {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		slog.Warn("wua collector unavailable: COM init failed", "error", err)
		return nil
	}
	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		ole.CoUninitialize()
		slog.Warn("wua collector unavailable: cannot create Update.Session", "error", err)
		return nil
	}
	unknown.Release()
	ole.CoUninitialize()

	return &wuaCollector{
		searcher: &comSearcher{logger: slog.Default()},
		logger:   slog.Default(),
	}
}

func detectWUAInstalledCollector() packageCollector {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		slog.Warn("wua_installed collector unavailable: COM init failed", "error", err)
		return nil
	}
	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		ole.CoUninitialize()
		slog.Warn("wua_installed collector unavailable: cannot create Update.Session", "error", err)
		return nil
	}
	unknown.Release()
	ole.CoUninitialize()

	return &wuaInstalledCollector{
		searcher: &comSearcher{logger: slog.Default()},
		logger:   slog.Default(),
	}
}

func detectPendingRebootCollector() packageCollector {
	return &pendingRebootCollector{
		checker: &winRebootChecker{logger: slog.Default()},
		logger:  slog.Default(),
	}
}

func detectRegistryCollector() packageCollector {
	return &registryCollector{
		reader: &winRegistryReader{logger: slog.Default()},
		logger: slog.Default(),
	}
}

func detectWindowsFeaturesCollector() packageCollector {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		slog.Warn("windows_features collector unavailable: powershell.exe not found", "error", err)
		return nil
	}
	return &windowsFeaturesCollector{
		runner: &execRunner{},
		logger: slog.Default(),
	}
}
```

- [ ] **Step 2: Verify build compiles on Windows**

Run: `GOOS=windows go build ./internal/agent/inventory/...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/agent/inventory/detect_windows.go
git commit -m "fix(agent): log reasons when Windows inventory collectors are unavailable

Previously all detector functions silently returned nil on failure,
making it impossible to diagnose missing collectors. Now each logs
a warning with the specific reason (COM init failed, PowerShell
not found, etc.)."
```

---

### Task 2: Add Windows-Specific Fields to PackageInfo Proto

**Files:**
- Modify: `proto/patchiq/v1/common.proto`

The `PackageInfo` proto currently only has generic fields. Windows collectors shove KB article IDs into `name`, severity into `status`, and categories into `release` — losing semantic meaning. Add proper fields.

- [ ] **Step 1: Add new fields to PackageInfo message**

In `proto/patchiq/v1/common.proto`, add fields after `release` (field 6):

```protobuf
// PackageInfo describes an installed OS package collected by the agent.
message PackageInfo {
  string name = 1;
  string version = 2;
  string architecture = 3;
  // Package manager source: "apt", "rpm", "wua", "wua_installed", "hotfix", "registry", "windows_feature", "softwareupdate", or "homebrew".
  string source = 4;
  // Package manager status (e.g. "install ok installed" for dpkg).
  string status = 5;
  // RPM release field (e.g. "1.el9"). Empty for APT packages.
  string release = 6;
  // Windows KB article ID (e.g. "KB5034441"). Set by WUA and hotfix collectors.
  string kb_article = 7;
  // MSRC severity rating (e.g. "Critical", "Important"). Set by WUA collector.
  string severity = 8;
  // Install date as ISO 8601 string. Set by registry and hotfix collectors.
  string install_date = 9;
  // Comma-separated categories (e.g. "Security Updates, Windows 11"). Set by WUA collector.
  string category = 10;
  // Software publisher (e.g. "Microsoft Corporation"). Set by registry collector.
  string publisher = 11;
}
```

- [ ] **Step 2: Regenerate protobuf code**

Run: `make proto`
Expected: `gen/patchiq/v1/common.pb.go` regenerated with new fields.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: No errors (new fields are additive, backward compatible).

- [ ] **Step 4: Commit**

```bash
git add proto/patchiq/v1/common.proto gen/patchiq/v1/
git commit -m "feat(proto): add Windows-specific fields to PackageInfo

Add kb_article, severity, install_date, category, publisher fields.
These allow Windows collectors to send structured data instead of
overloading generic fields (name, status, release) with unrelated
semantics."
```

---

### Task 3: Update WUA Collectors to Use New Proto Fields

**Files:**
- Modify: `internal/agent/inventory/wua_parse.go`
- Modify: `internal/agent/inventory/wua_parse_test.go`

The `mapWindowsUpdates` function currently maps `KBID` → `Name`, `Title` → `Version`, `Severity` → `Status`, `Categories` → `Release`. Fix to use proper fields.

- [ ] **Step 1: Write the failing test**

In `internal/agent/inventory/wua_parse_test.go`, add a test that asserts the new field mapping:

```go
func TestMapWindowsUpdates_NewFields(t *testing.T) {
	updates := []windowsUpdate{
		{
			KBID:       "KB5034441",
			Title:      "2024-01 Cumulative Update for Windows 11",
			Severity:   "Critical",
			Categories: []string{"Security Updates", "Windows 11"},
		},
	}

	pkgs := mapWindowsUpdates(updates)
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	p := pkgs[0]

	// Name should be the title (human-readable), not the KB ID.
	if p.Name != "2024-01 Cumulative Update for Windows 11" {
		t.Errorf("Name = %q, want title", p.Name)
	}
	if p.KbArticle != "KB5034441" {
		t.Errorf("KbArticle = %q, want KB5034441", p.KbArticle)
	}
	if p.Severity != "Critical" {
		t.Errorf("Severity = %q, want Critical", p.Severity)
	}
	if p.Category != "Security Updates, Windows 11" {
		t.Errorf("Category = %q, want joined categories", p.Category)
	}
	// Source should remain "wua".
	if p.Source != "wua" {
		t.Errorf("Source = %q, want wua", p.Source)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/inventory/ -run TestMapWindowsUpdates_NewFields -v`
Expected: FAIL — `Name` is currently set to KBID, `KbArticle` field is empty.

- [ ] **Step 3: Update mapWindowsUpdates implementation**

In `internal/agent/inventory/wua_parse.go`:

```go
package inventory

import (
	"runtime"
	"strings"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// windowsUpdate represents a single Windows Update result from IUpdateSearcher.
type windowsUpdate struct {
	KBID       string
	Title      string
	Severity   string
	Categories []string
}

// mapWindowsUpdates converts WUA search results to PackageInfo protos.
// Entries with empty KBID are skipped.
func mapWindowsUpdates(updates []windowsUpdate) []*pb.PackageInfo {
	var pkgs []*pb.PackageInfo
	for _, u := range updates {
		if u.KBID == "" {
			continue
		}
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         u.Title,
			Version:      u.KBID,
			Architecture: runtime.GOARCH,
			Source:       "wua",
			KbArticle:   u.KBID,
			Severity:    u.Severity,
			Category:    strings.Join(u.Categories, ", "),
		})
	}
	return pkgs
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/agent/inventory/ -run TestMapWindowsUpdates -v`
Expected: PASS

- [ ] **Step 5: Update existing WUA parse tests if any fail**

Run: `go test ./internal/agent/inventory/ -v`
Check for regressions in existing tests that assert on the old field mapping (e.g., `Name == KBID`). Update those assertions to match the new mapping.

- [ ] **Step 6: Commit**

```bash
git add internal/agent/inventory/wua_parse.go internal/agent/inventory/wua_parse_test.go
git commit -m "fix(agent): map WUA fields to proper PackageInfo proto fields

KBID now goes to kb_article (was overloaded into name).
Title now goes to name (was overloaded into version).
Severity now goes to severity (was overloaded into status).
Categories now goes to category (was overloaded into release)."
```

---

### Task 4: Update Hotfix Collector to Use New Proto Fields

**Files:**
- Modify: `internal/agent/inventory/hotfix_parse.go`
- Modify: `internal/agent/inventory/hotfix_parse_test.go`

The hotfix collector maps `HotFixID` → `Name`, `InstalledOn` → `Version`, `Description` → `Status`. Fix to use proper fields.

- [ ] **Step 1: Write the failing test**

In the existing hotfix parse test file, add:

```go
func TestParseHotFixOutput_NewFields(t *testing.T) {
	input := `[{"HotFixID":"KB5034441","Description":"Security Update","InstalledOn":"1/15/2024"}]`
	pkgs, err := parseHotFixOutput([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	p := pkgs[0]

	if p.Name != "KB5034441" {
		t.Errorf("Name = %q, want KB5034441", p.Name)
	}
	if p.KbArticle != "KB5034441" {
		t.Errorf("KbArticle = %q, want KB5034441", p.KbArticle)
	}
	if p.InstallDate != "1/15/2024" {
		t.Errorf("InstallDate = %q, want 1/15/2024", p.InstallDate)
	}
	if p.Status != "Security Update" {
		t.Errorf("Status = %q, want Security Update", p.Status)
	}
	if p.Source != "hotfix" {
		t.Errorf("Source = %q, want hotfix", p.Source)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/inventory/ -run TestParseHotFixOutput_NewFields -v`
Expected: FAIL — `KbArticle` is empty, `InstallDate` is empty.

- [ ] **Step 3: Update parseHotFixOutput implementation**

In `internal/agent/inventory/hotfix_parse.go`, update the mapping in the `parseHotFixOutput` function:

```go
	var pkgs []*pb.PackageInfo
	for _, e := range entries {
		if e.HotFixID == "" {
			continue
		}
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:        e.HotFixID,
			Version:     installedOnString(e.InstalledOn),
			Source:      "hotfix",
			Status:      e.Description,
			KbArticle:   e.HotFixID,
			InstallDate: installedOnString(e.InstalledOn),
		})
	}
	return pkgs, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/agent/inventory/ -run TestParseHotFixOutput -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/inventory/hotfix_parse.go internal/agent/inventory/hotfix_parse_test.go
git commit -m "fix(agent): populate kb_article and install_date in hotfix collector

HotFixID is now also set in kb_article field.
InstalledOn is now also set in install_date field."
```

---

### Task 5: Update Registry Collector to Use New Proto Fields and Implement extendedCollector

**Files:**
- Modify: `internal/agent/inventory/registry_windows.go`

The registry collector collects `Publisher`, `InstallDate`, and bitness info but discards them. It also doesn't implement `extendedCollector`, so the agent's local `/api/v1/software` endpoint returns nothing for Windows.

- [ ] **Step 1: Write the failing test**

Create or update `internal/agent/inventory/registry_windows_test.go`:

```go
//go:build windows

package inventory

import (
	"context"
	"testing"
)

type fakeRegistryReader struct {
	entries []registryEntry
	err     error
}

func (f *fakeRegistryReader) ReadUninstallKeys() ([]registryEntry, error) {
	return f.entries, f.err
}

func TestRegistryCollector_NewFields(t *testing.T) {
	reader := &fakeRegistryReader{
		entries: []registryEntry{
			{
				DisplayName:    "Visual Studio Code",
				DisplayVersion: "1.85.0",
				Publisher:      "Microsoft Corporation",
				InstallDate:    "20240115",
				Is64Bit:        true,
			},
		},
	}
	c := &registryCollector{reader: reader, logger: testLogger(t)}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	p := pkgs[0]

	if p.Publisher != "Microsoft Corporation" {
		t.Errorf("Publisher = %q, want Microsoft Corporation", p.Publisher)
	}
	if p.InstallDate != "20240115" {
		t.Errorf("InstallDate = %q, want 20240115", p.InstallDate)
	}
	if p.Architecture != "x64" {
		t.Errorf("Architecture = %q, want x64", p.Architecture)
	}
}

func TestRegistryCollector_ExtendedPackages(t *testing.T) {
	reader := &fakeRegistryReader{
		entries: []registryEntry{
			{
				DisplayName:    "Visual Studio Code",
				DisplayVersion: "1.85.0",
				Publisher:      "Microsoft Corporation",
				InstallDate:    "20240115",
				Is64Bit:        true,
			},
		},
	}
	c := &registryCollector{reader: reader, logger: testLogger(t)}
	// Must collect first to populate cache.
	_, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	ext := c.ExtendedPackages()
	if len(ext) != 1 {
		t.Fatalf("expected 1 extended package, got %d", len(ext))
	}
	if ext[0].Publisher != "Microsoft Corporation" {
		t.Errorf("Publisher = %q", ext[0].Publisher)
	}
	if ext[0].Category != "Application" {
		t.Errorf("Category = %q, want Application", ext[0].Category)
	}
}
```

Note: `testLogger` is a helper that returns a `*slog.Logger` for tests. If it doesn't exist yet, add it:

```go
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.Default()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOOS=windows go test ./internal/agent/inventory/ -run TestRegistryCollector_NewFields -v`
Expected: FAIL — `Publisher` is empty, `InstallDate` is empty.

- [ ] **Step 3: Update registryCollector implementation**

In `internal/agent/inventory/registry_windows.go`:

```go
type registryCollector struct {
	reader   registryReaderIface
	logger   *slog.Logger
	lastPkgs []ExtendedPackageInfo // cache for extendedCollector interface
}

func (c *registryCollector) Name() string { return "registry" }

func (c *registryCollector) Collect(_ context.Context) ([]*pb.PackageInfo, error) {
	entries, err := c.reader.ReadUninstallKeys()
	if err != nil {
		return nil, fmt.Errorf("registry collector: %w", err)
	}

	seen := make(map[string]struct{})
	var pkgs []*pb.PackageInfo
	var extended []ExtendedPackageInfo

	for _, e := range entries {
		if e.DisplayName == "" {
			continue
		}
		dedupKey := e.DisplayName + "|" + e.DisplayVersion
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}

		arch := ""
		if e.Is64Bit {
			arch = "x64"
		} else {
			arch = "x86"
		}

		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         e.DisplayName,
			Version:      e.DisplayVersion,
			Architecture: arch,
			Source:       "registry",
			Publisher:    e.Publisher,
			InstallDate: e.InstallDate,
		})
		extended = append(extended, ExtendedPackageInfo{
			Name:         e.DisplayName,
			Version:      e.DisplayVersion,
			Architecture: arch,
			Source:       "registry",
			Category:    "Application",
			InstallDate: e.InstallDate,
			License:     "",
			Description: "",
		})
	}

	c.lastPkgs = extended
	return pkgs, nil
}

// ExtendedPackages implements the extendedCollector interface for the local agent API.
func (c *registryCollector) ExtendedPackages() []ExtendedPackageInfo {
	return c.lastPkgs
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOOS=windows go test ./internal/agent/inventory/ -run TestRegistryCollector -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/inventory/registry_windows.go internal/agent/inventory/registry_windows_test.go
git commit -m "feat(agent): registry collector populates new proto fields and implements extendedCollector

Publisher, InstallDate, Architecture now set in PackageInfo.
Implements extendedCollector so /api/v1/software returns data on Windows."
```

---

### Task 6: Implement extendedCollector on WUA and Hotfix Collectors

**Files:**
- Modify: `internal/agent/inventory/wua.go` (add lastPkgs cache + ExtendedPackages)
- Modify: `internal/agent/inventory/wua_installed_windows.go` (same)
- Modify: `internal/agent/inventory/hotfix.go` (same, build tag windows)

These collectors need to cache their last collection results as `[]ExtendedPackageInfo` so the agent's local `/api/v1/software` endpoint returns Windows data.

- [ ] **Step 1: Add ExtendedPackages to wuaCollector**

In `internal/agent/inventory/wua.go`, add a `lastPkgs` field and `ExtendedPackages()` method:

```go
type wuaCollector struct {
	searcher updateSearcher
	logger   *slog.Logger
	lastPkgs []ExtendedPackageInfo
}

func (c *wuaCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	updates, err := c.searcher.Search(ctx, "IsInstalled=0")
	if err != nil {
		return nil, fmt.Errorf("wua collector: %w", err)
	}
	pkgs := mapWindowsUpdates(updates)

	// Cache extended info for local API.
	c.lastPkgs = make([]ExtendedPackageInfo, 0, len(pkgs))
	for _, p := range pkgs {
		c.lastPkgs = append(c.lastPkgs, ExtendedPackageInfo{
			Name:        p.Name,
			Version:     p.Version,
			Source:      p.Source,
			Category:   p.Category,
			Status:     "available",
			Description: p.KbArticle,
		})
	}

	return pkgs, nil
}

func (c *wuaCollector) ExtendedPackages() []ExtendedPackageInfo {
	return c.lastPkgs
}
```

- [ ] **Step 2: Add ExtendedPackages to wuaInstalledCollector**

In `internal/agent/inventory/wua_installed_windows.go`:

```go
type wuaInstalledCollector struct {
	searcher updateSearcher
	logger   *slog.Logger
	lastPkgs []ExtendedPackageInfo
}

func (c *wuaInstalledCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	updates, err := c.searcher.Search(ctx, "IsInstalled=1")
	if err != nil {
		return nil, fmt.Errorf("wua installed collector: %w", err)
	}
	pkgs := mapWindowsUpdates(updates)
	for _, p := range pkgs {
		p.Source = "wua_installed"
	}

	c.lastPkgs = make([]ExtendedPackageInfo, 0, len(pkgs))
	for _, p := range pkgs {
		c.lastPkgs = append(c.lastPkgs, ExtendedPackageInfo{
			Name:        p.Name,
			Version:     p.Version,
			Source:      p.Source,
			Category:   p.Category,
			Status:     "installed",
			Description: p.KbArticle,
		})
	}

	return pkgs, nil
}

func (c *wuaInstalledCollector) ExtendedPackages() []ExtendedPackageInfo {
	return c.lastPkgs
}
```

- [ ] **Step 3: Add ExtendedPackages to hotfixCollector**

In `internal/agent/inventory/hotfix.go`:

```go
type hotfixCollector struct {
	runner   commandRunner
	lastPkgs []ExtendedPackageInfo
}

func (c *hotfixCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	out, err := c.runner.Run(ctx, "powershell.exe", "-NoProfile", "-Command",
		"Get-HotFix | ConvertTo-Json")
	if err != nil {
		return nil, fmt.Errorf("hotfix collector: %w", err)
	}
	pkgs, err := parseHotFixOutput(out)
	if err != nil {
		return nil, fmt.Errorf("hotfix collector: %w", err)
	}

	c.lastPkgs = make([]ExtendedPackageInfo, 0, len(pkgs))
	for _, p := range pkgs {
		c.lastPkgs = append(c.lastPkgs, ExtendedPackageInfo{
			Name:        p.Name,
			Version:     p.Version,
			Source:      "hotfix",
			Status:      p.Status,
			InstallDate: p.InstallDate,
			Category:   "System",
		})
	}

	return pkgs, nil
}

func (c *hotfixCollector) ExtendedPackages() []ExtendedPackageInfo {
	return c.lastPkgs
}
```

- [ ] **Step 4: Verify build**

Run: `GOOS=windows go build ./internal/agent/inventory/...`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add internal/agent/inventory/wua.go internal/agent/inventory/wua_installed_windows.go internal/agent/inventory/hotfix.go
git commit -m "feat(agent): WUA and hotfix collectors implement extendedCollector

All Windows collectors now cache extended package metadata so the
local /api/v1/software endpoint returns data on Windows systems."
```

---

### Task 7: Update Pending Reboot Collector to Include Source Key

**Files:**
- Modify: `internal/agent/inventory/pending_reboot_windows.go`
- Modify: `internal/agent/inventory/pending_reboot_windows_test.go`

Currently returns a single `REBOOT_PENDING` entry without indicating which registry key triggered it. This makes it impossible to know the reboot reason.

- [ ] **Step 1: Write the failing test**

In `internal/agent/inventory/pending_reboot_windows_test.go`, add:

```go
func TestPendingRebootCollector_IncludesSourceKey(t *testing.T) {
	checker := &fakeRebootChecker{
		existing: map[string]bool{
			`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`: true,
		},
	}
	c := &pendingRebootCollector{checker: checker, logger: slog.Default()}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("expected reboot pending result")
	}
	if pkgs[0].Category != "WindowsUpdate" {
		t.Errorf("Category = %q, want WindowsUpdate", pkgs[0].Category)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOOS=windows go test ./internal/agent/inventory/ -run TestPendingRebootCollector_IncludesSourceKey -v`
Expected: FAIL — `Category` is empty.

- [ ] **Step 3: Update implementation to include source identification**

```go
// pendingRebootPaths maps registry paths to human-readable reboot reason categories.
var pendingRebootPaths = []struct {
	path     string
	category string
}{
	{`SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending`, "CBS"},
	{`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`, "WindowsUpdate"},
	{`SYSTEM\CurrentControlSet\Control\Session Manager\PendingFileRenameOperations`, "FileRename"},
}

func (c *pendingRebootCollector) Collect(_ context.Context) ([]*pb.PackageInfo, error) {
	var pkgs []*pb.PackageInfo
	for _, entry := range pendingRebootPaths {
		if c.checker.KeyExists(entry.path) {
			c.logger.Info("pending reboot detected", "registry_key", entry.path, "category", entry.category)
			pkgs = append(pkgs, &pb.PackageInfo{
				Name:     "REBOOT_PENDING",
				Source:   "system",
				Status:   "pending",
				Category: entry.category,
			})
		}
	}
	return pkgs, nil
}
```

Note: This now returns ALL matching reboot reasons instead of stopping at the first one. This gives the dashboard full visibility.

- [ ] **Step 4: Update existing test assertions**

Existing tests that assert `len(pkgs) == 1` when multiple keys exist will need updating. Check `pendingRebootPaths` usage in existing tests and update to match the struct change.

- [ ] **Step 5: Run all pending reboot tests**

Run: `GOOS=windows go test ./internal/agent/inventory/ -run TestPendingReboot -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/agent/inventory/pending_reboot_windows.go internal/agent/inventory/pending_reboot_windows_test.go
git commit -m "fix(agent): pending reboot collector reports all sources with category

Returns all matching reboot registry keys with category labels
(CBS, WindowsUpdate, FileRename) instead of stopping at first match."
```

---

### Task 8: Server-Side Migration — Add Windows Fields to endpoint_packages

**Files:**
- Create: `internal/server/store/migrations/046_package_windows_fields.sql`
- Modify: `internal/server/store/queries/inventory.sql`

Add new columns to `endpoint_packages` and `collection_errors` JSONB to `endpoint_inventories`.

- [ ] **Step 1: Check the latest migration number**

Run: `ls internal/server/store/migrations/ | tail -5`
If the latest is `045_*.sql`, use `046`. Adjust the number accordingly.

- [ ] **Step 2: Create the migration file**

```sql
-- +goose Up
-- ============================================================
-- Windows-specific package fields and collection error tracking
-- ============================================================

-- Add structured Windows fields to endpoint_packages.
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS kb_article TEXT;
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS severity TEXT;
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS install_date TEXT;
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS category TEXT;
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS publisher TEXT;

-- Track collection errors on inventory snapshots so partial reports
-- are distinguishable from clean endpoints with no software.
ALTER TABLE endpoint_inventories ADD COLUMN IF NOT EXISTS collection_errors JSONB DEFAULT '[]';

-- Index for severity-based queries on endpoint packages.
CREATE INDEX IF NOT EXISTS idx_endpoint_packages_severity
    ON endpoint_packages(tenant_id, severity) WHERE severity IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_endpoint_packages_severity;
ALTER TABLE endpoint_inventories DROP COLUMN IF EXISTS collection_errors;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS publisher;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS category;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS install_date;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS severity;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS kb_article;
```

- [ ] **Step 3: Update inventory.sql queries to include new columns**

In `internal/server/store/queries/inventory.sql`, update `CreateEndpointPackage`:

```sql
-- name: CreateEndpointPackage :one
INSERT INTO endpoint_packages (tenant_id, endpoint_id, inventory_id, package_name, version, arch, source, release, kb_article, severity, install_date, category, publisher)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;
```

Update `CreateEndpointInventory` to accept `collection_errors`:

```sql
-- name: CreateEndpointInventory :one
INSERT INTO endpoint_inventories (tenant_id, endpoint_id, scanned_at, package_count, collection_errors)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
```

- [ ] **Step 4: Regenerate sqlc**

Run: `make sqlc`
Expected: `internal/server/store/sqlcgen/inventory.sql.go` regenerated with new params.

- [ ] **Step 5: Verify build**

Run: `go build ./internal/server/...`
Expected: Compilation errors in `sync_outbox.go` and `inventory.go` where the old param structs are used. These will be fixed in Tasks 9 and 10.

- [ ] **Step 6: Commit**

```bash
git add internal/server/store/migrations/046_package_windows_fields.sql internal/server/store/queries/inventory.sql internal/server/store/sqlcgen/
git commit -m "feat(server): add Windows package fields and collection_errors to inventory schema

New columns: kb_article, severity, install_date, category, publisher
on endpoint_packages. collection_errors JSONB on endpoint_inventories
for tracking partial collection failures."
```

---

### Task 9: Update BulkInsertEndpointPackages to Store New Fields

**Files:**
- Modify: `internal/server/store/inventory.go`
- Modify: `internal/server/store/inventory_test.go`

The `buildMultiRowInsert` function currently inserts 10 columns. Update to include the 5 new fields.

- [ ] **Step 1: Write the failing test**

In `internal/server/store/inventory_test.go`, add a test that passes `PackageInfo` with the new fields and asserts they are included in the generated SQL:

```go
func TestBuildMultiRowInsert_IncludesWindowsFields(t *testing.T) {
	pkgs := []*pb.PackageInfo{
		{
			Name:         "2024-01 Cumulative Update",
			Version:      "KB5034441",
			Architecture: "amd64",
			Source:       "wua",
			KbArticle:   "KB5034441",
			Severity:    "Critical",
			InstallDate: "2024-01-15",
			Category:    "Security Updates",
			Publisher:   "Microsoft Corporation",
		},
	}
	tenantID := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	endpointID := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	inventoryID := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	query, args := buildMultiRowInsert(pkgs, tenantID, endpointID, inventoryID, now)

	// Should have 15 columns per row now.
	if len(args) != 15 {
		t.Errorf("expected 15 args, got %d", len(args))
	}
	if !strings.Contains(query, "kb_article") {
		t.Error("query should contain kb_article column")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/store/ -run TestBuildMultiRowInsert_IncludesWindowsFields -v`
Expected: FAIL — only 10 args, no `kb_article` in query.

- [ ] **Step 3: Update buildMultiRowInsert**

In `internal/server/store/inventory.go`:

```go
func buildMultiRowInsert(
	packages []*pb.PackageInfo,
	tenantID, endpointID, inventoryID pgtype.UUID,
	now pgtype.Timestamptz,
) (string, []any) {
	const colsPerRow = 15
	args := make([]any, 0, len(packages)*colsPerRow)

	query := make([]byte, 0, 300+len(packages)*180)
	query = append(query, "INSERT INTO endpoint_packages (id, tenant_id, endpoint_id, inventory_id, package_name, version, arch, source, release, created_at, kb_article, severity, install_date, category, publisher) VALUES "...)

	for i, pkg := range packages {
		if i > 0 {
			query = append(query, ", "...)
		}
		base := i * colsPerRow
		query = append(query, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5,
			base+6, base+7, base+8, base+9, base+10,
			base+11, base+12, base+13, base+14, base+15,
		)...)

		args = append(args,
			pgtype.UUID{Bytes: uuid.New(), Valid: true},
			tenantID, endpointID, inventoryID,
			pkg.GetName(), pkg.GetVersion(),
			nullableText(pkg.GetArchitecture()),
			nullableText(pkg.GetSource()),
			nullableText(pkg.GetRelease()),
			now,
			nullableText(pkg.GetKbArticle()),
			nullableText(pkg.GetSeverity()),
			nullableText(pkg.GetInstallDate()),
			nullableText(pkg.GetCategory()),
			nullableText(pkg.GetPublisher()),
		)
	}

	return string(query), args
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server/store/ -run TestBuildMultiRowInsert -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/store/inventory.go internal/server/store/inventory_test.go
git commit -m "feat(server): store Windows-specific package fields in bulk insert

BulkInsertEndpointPackages now writes kb_article, severity,
install_date, category, publisher for each package row."
```

---

### Task 10: Server Accepts Partial Inventory Reports

**Files:**
- Modify: `internal/server/grpc/sync_outbox.go`

This is the **critical fix**. Currently the server rejects inventory reports with zero packages (`OUTBOX_REJECTION_CODE_PAYLOAD_INVALID`). This silently drops all data when Windows collectors fail. The proto comment on `collection_errors` already says "the server must not interpret missing data as a clean endpoint" — but the code doesn't follow this.

- [ ] **Step 1: Write the failing test**

In `internal/server/grpc/sync_outbox_test.go`, add a test that sends an inventory report with zero packages but non-empty `collection_errors`:

```go
func TestProcessInventory_AcceptsPartialReport(t *testing.T) {
	// Build an inventory report with zero packages but collection errors.
	report := &pb.InventoryReport{
		ProtocolVersion: 1,
		EndpointInfo: &pb.EndpointInfo{
			Hostname: "test-host",
			OsFamily: pb.OsFamily_OS_FAMILY_WINDOWS,
		},
		CollectedAt: timestamppb.Now(),
		CollectionErrors: []*pb.InventoryCollectionError{
			{Collector: "wua", ErrorMessage: "COM init failed"},
			{Collector: "hotfix", ErrorMessage: "powershell not found"},
		},
		InstalledPackages: nil, // zero packages
	}

	// The server should accept this as a valid (partial) report,
	// not reject it as PAYLOAD_INVALID.
	// ... (test setup depends on existing test infrastructure — see existing tests
	// in the file for how to create mock store, tenantID, endpointID, etc.)
}
```

Adapt to whatever test infrastructure exists in `sync_outbox_test.go`. The key assertion: the ack should be `ACCEPTED`, not `REJECTED`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/grpc/ -run TestProcessInventory_AcceptsPartialReport -v`
Expected: FAIL — returns `OUTBOX_REJECTION_CODE_PAYLOAD_INVALID`.

- [ ] **Step 3: Update processInventory validation logic**

In `internal/server/grpc/sync_outbox.go`, replace the existing validation at lines 159-164:

**Old code:**
```go
	// 2. Validate: at least one package required.
	if len(report.GetInstalledPackages()) == 0 {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID,
			"inventory report contains no packages",
		), fmt.Errorf("process inventory: empty packages")
	}
```

**New code:**
```go
	// 2. Validate: reject only if zero packages AND zero collection errors.
	// A report with collection_errors but no packages is a valid partial report —
	// the agent tried but some collectors failed. Rejecting it would silently
	// discard error diagnostics and prevent the server from knowing collection
	// is broken on this endpoint.
	if len(report.GetInstalledPackages()) == 0 && len(report.GetCollectionErrors()) == 0 {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID,
			"inventory report contains no packages and no collection errors",
		), fmt.Errorf("process inventory: empty report")
	}
```

- [ ] **Step 4: Store collection_errors in the inventory record**

In the same function, after `CreateEndpointInventory`, add collection_errors storage. Update the `CreateEndpointInventoryParams` to include the new `collection_errors` field:

```go
	// Marshal collection errors to JSONB.
	var collectionErrorsJSON []byte
	if len(report.GetCollectionErrors()) > 0 {
		type collError struct {
			Collector string `json:"collector"`
			Error     string `json:"error"`
		}
		errs := make([]collError, len(report.GetCollectionErrors()))
		for i, ce := range report.GetCollectionErrors() {
			errs[i] = collError{Collector: ce.GetCollector(), Error: ce.GetErrorMessage()}
		}
		collectionErrorsJSON, _ = json.Marshal(errs)
	}
	if collectionErrorsJSON == nil {
		collectionErrorsJSON = []byte("[]")
	}

	inv, err := qtx.CreateEndpointInventory(ctx, sqlcgen.CreateEndpointInventoryParams{
		TenantID:         tenantPgUUID,
		EndpointID:       endpointID,
		ScannedAt:        pgtype.Timestamptz{Time: scannedAt, Valid: true},
		PackageCount:     int32(len(report.GetInstalledPackages())),
		CollectionErrors: collectionErrorsJSON,
	})
```

- [ ] **Step 5: Log partial reports distinctly**

Add logging after validation:

```go
	if len(report.GetCollectionErrors()) > 0 {
		s.logger.WarnContext(ctx, "process inventory: partial report received",
			"agent_id", agentIDStr,
			"package_count", len(report.GetInstalledPackages()),
			"collection_errors", len(report.GetCollectionErrors()),
		)
	}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/server/grpc/ -run TestProcessInventory -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/server/grpc/sync_outbox.go internal/server/grpc/sync_outbox_test.go
git commit -m "fix(server): accept partial inventory reports with collection errors

Previously the server rejected any inventory report with zero packages
as PAYLOAD_INVALID. This silently dropped reports when Windows
collectors failed (COM errors, WMI unavailable). Now the server
accepts reports that have collection_errors, stores the errors in
endpoint_inventories.collection_errors JSONB, and logs a warning."
```

---

### Task 11: Agent SQLite Inventory Cache

**Files:**
- Modify: `internal/agent/store/schema.sql`
- Modify: `internal/agent/store/db.go`
- Create: `internal/agent/store/sqlite_inventory.go`
- Create: `internal/agent/store/sqlite_inventory_test.go`

Currently inventory only lives in-memory. On agent restart, `/api/v1/software` returns empty until the next collection (up to 24h). Add a local cache table.

- [ ] **Step 1: Write the failing test**

Create `internal/agent/store/sqlite_inventory_test.go`:

```go
package store

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInventoryCache_SaveAndLoad(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Save inventory snapshot.
	pkgsJSON := []byte(`[{"name":"vscode","version":"1.85"}]`)
	if err := SaveInventoryCache(db, ctx, pkgsJSON); err != nil {
		t.Fatalf("SaveInventoryCache: %v", err)
	}

	// Load it back.
	loaded, collectedAt, err := LoadInventoryCache(db, ctx)
	if err != nil {
		t.Fatalf("LoadInventoryCache: %v", err)
	}
	if string(loaded) != string(pkgsJSON) {
		t.Errorf("loaded = %s, want %s", loaded, pkgsJSON)
	}
	if collectedAt.IsZero() {
		t.Error("collected_at should not be zero")
	}
}

func TestInventoryCache_EmptyOnFreshDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	loaded, _, err := LoadInventoryCache(db, ctx)
	if err != nil {
		t.Fatalf("LoadInventoryCache: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil on fresh db, got %s", loaded)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/store/ -run TestInventoryCache -v`
Expected: FAIL — `SaveInventoryCache` and `LoadInventoryCache` don't exist.

- [ ] **Step 3: Add inventory_cache table to schema**

In `internal/agent/store/schema.sql`, add:

```sql
CREATE TABLE IF NOT EXISTS inventory_cache (
    id           INTEGER PRIMARY KEY CHECK (id = 1),
    packages_json TEXT NOT NULL,
    collected_at  TEXT NOT NULL
);
```

The `CHECK (id = 1)` constraint ensures only one row exists (latest snapshot).

- [ ] **Step 4: Add migration for existing databases**

In `internal/agent/store/db.go`, add to `ApplyMigrations`:

```go
	// inventory_cache table (idempotent).
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS inventory_cache (
		id            INTEGER PRIMARY KEY CHECK (id = 1),
		packages_json TEXT NOT NULL,
		collected_at  TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create inventory_cache: %w", err)
	}
```

- [ ] **Step 5: Create sqlite_inventory.go**

```go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SaveInventoryCache upserts the latest inventory snapshot into the local cache.
func SaveInventoryCache(db *sql.DB, ctx context.Context, packagesJSON []byte) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO inventory_cache (id, packages_json, collected_at) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET packages_json = excluded.packages_json, collected_at = excluded.collected_at`,
		string(packagesJSON), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("save inventory cache: %w", err)
	}
	return nil
}

// LoadInventoryCache reads the latest cached inventory snapshot.
// Returns nil, zero time, nil if no cache exists.
func LoadInventoryCache(db *sql.DB, ctx context.Context) ([]byte, time.Time, error) {
	var pkgsJSON string
	var collectedAtStr string
	err := db.QueryRowContext(ctx, `SELECT packages_json, collected_at FROM inventory_cache WHERE id = 1`).
		Scan(&pkgsJSON, &collectedAtStr)
	if err == sql.ErrNoRows {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("load inventory cache: %w", err)
	}

	collectedAt, _ := time.Parse(time.RFC3339, collectedAtStr)
	return []byte(pkgsJSON), collectedAt, nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/agent/store/ -run TestInventoryCache -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/agent/store/schema.sql internal/agent/store/db.go internal/agent/store/sqlite_inventory.go internal/agent/store/sqlite_inventory_test.go
git commit -m "feat(agent): add SQLite inventory cache for persistence across restarts

Adds inventory_cache table (single row, upsert pattern). Software
data survives agent restarts instead of going blank for up to 24h."
```

---

### Task 12: Wire Inventory Cache into Collection Runner and API

**Files:**
- Modify: `internal/agent/runner.go`
- Modify: `internal/agent/api/inventory_provider.go`
- Modify: `internal/agent/inventory/collector.go`

After collection, persist to SQLite. When API serves software, fall back to SQLite if in-memory is empty.

- [ ] **Step 1: Add LocalDB to CollectionRunner**

In `internal/agent/runner.go`, add a `localDB` field and save inventory after collection:

```go
type CollectionRunner struct {
	modules      []Module
	outbox       OutboxWriter
	logger       *slog.Logger
	intervalFunc func() time.Duration
	logWriter    OperationalLogWriter
	localDB      *sql.DB
}

func NewCollectionRunner(modules []Module, outbox OutboxWriter, logger *slog.Logger) *CollectionRunner {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &CollectionRunner{modules: modules, outbox: outbox, logger: logger}
}

// SetLocalDB sets the local SQLite database for inventory caching.
func (r *CollectionRunner) SetLocalDB(db *sql.DB) {
	r.localDB = db
}
```

In `collectOnce`, after outbox write, persist to cache:

```go
func (r *CollectionRunner) collectOnce(ctx context.Context, mod Module, logger *slog.Logger) {
	items, err := mod.Collect(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "collection failed", "error", err)
		r.writeLog(ctx, "error", fmt.Sprintf("Collection failed for module %s: %v", mod.Name(), err), "collector")
		return
	}

	for _, item := range items {
		if _, err := r.outbox.Add(ctx, item.MessageType, item.Payload); err != nil {
			logger.ErrorContext(ctx, "outbox write failed", "error", err, "message_type", item.MessageType)
		}
	}

	// Persist extended packages to local cache for the API.
	if r.localDB != nil {
		if invMod, ok := mod.(interface{ ExtendedPackages() []inventory.ExtendedPackageInfo }); ok {
			pkgs := invMod.ExtendedPackages()
			if len(pkgs) > 0 {
				pkgsJSON, err := json.Marshal(pkgs)
				if err != nil {
					logger.WarnContext(ctx, "marshal inventory cache", "error", err)
				} else if err := store.SaveInventoryCache(r.localDB, ctx, pkgsJSON); err != nil {
					logger.WarnContext(ctx, "save inventory cache", "error", err)
				}
			}
		}
	}

	r.writeLog(ctx, "info", fmt.Sprintf("Inventory scan completed for module %s, %d items collected", mod.Name(), len(items)), "collector")
}
```

Add imports for `encoding/json`, `database/sql`, the inventory and store packages.

Note: The `inventory.ExtendedPackageInfo` import means `runner.go` in `internal/agent` needs to import `internal/agent/inventory` — check that this doesn't create a cycle. The `Module` interface is in `internal/agent`, and `inventory.Module` implements it, so `agent` → `inventory` would be a cycle. Instead, use a type assertion on a local interface:

```go
// extendedPackageProvider is satisfied by inventory.Module.
type extendedPackageProvider interface {
	ExtendedPackages() []any
}
```

Actually, the cleaner approach: have the runner accept raw JSON from the module. Add a method to the inventory module that returns JSON bytes directly. But to keep this simple and avoid changing the Module interface, use the store package directly with raw bytes:

The runner already imports `internal/agent` for the `Module` type. To avoid cycles, use `json.RawMessage` and have the collect step also return cached bytes. The simplest approach:

```go
// In collectOnce, after successful collection:
// Cache the raw protobuf payload for the API to decode if needed.
if r.localDB != nil && len(items) > 0 {
	for _, item := range items {
		if item.MessageType == "inventory" {
			if err := store.SaveInventoryPayload(r.localDB, ctx, item.Payload); err != nil {
				logger.WarnContext(ctx, "save inventory payload cache", "error", err)
			}
		}
	}
}
```

This avoids importing the inventory package. The store saves the raw protobuf, and the API can decode it. But this changes the API flow. Let me reconsider.

**Simpler approach**: The `softwareAdapter` already holds a `*inventory.Module`. We can add a `SetLocalDB` method to the adapter and have it check the cache on empty results.

- [ ] **Step 2: Update softwareAdapter to fall back to cache**

In `internal/agent/api/inventory_provider.go`:

```go
type softwareAdapter struct {
	module  *inventory.Module
	localDB *sql.DB
}

func NewSoftwareAdapter(module *inventory.Module, localDB *sql.DB) SoftwareProvider {
	return &softwareAdapter{module: module, localDB: localDB}
}

func (a *softwareAdapter) ExtendedPackages(ctx context.Context) ([]inventory.ExtendedPackageInfo, error) {
	pkgs := a.module.ExtendedPackages()
	if len(pkgs) > 0 {
		return pkgs, nil
	}

	// Fall back to SQLite cache if in-memory is empty (e.g., after restart).
	if a.localDB != nil {
		cached, _, err := store.LoadInventoryCache(a.localDB, ctx)
		if err != nil {
			return nil, fmt.Errorf("load inventory cache: %w", err)
		}
		if cached != nil {
			var fromCache []inventory.ExtendedPackageInfo
			if err := json.Unmarshal(cached, &fromCache); err != nil {
				return nil, fmt.Errorf("unmarshal inventory cache: %w", err)
			}
			return fromCache, nil
		}
	}

	return nil, nil
}
```

- [ ] **Step 3: Update call sites that create NewSoftwareAdapter**

Search for `NewSoftwareAdapter(` in `cmd/agent/main.go` and add the `localDB` parameter.

- [ ] **Step 4: Wire localDB into CollectionRunner**

In `cmd/agent/main.go`, after creating the runner, add:

```go
runner.SetLocalDB(localDB)
```

- [ ] **Step 5: Verify build**

Run: `go build ./cmd/agent/...`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add internal/agent/runner.go internal/agent/api/inventory_provider.go cmd/agent/main.go
git commit -m "feat(agent): persist inventory to SQLite cache, API falls back on empty

CollectionRunner saves extended packages to inventory_cache after
each collection. SoftwareAdapter reads from cache when in-memory
data is empty (e.g., after agent restart)."
```

---

### Task 13: Add Collection Health to Agent Status API

**Files:**
- Modify: `internal/agent/inventory/collector.go`
- Modify: `internal/agent/api/status.go`

The status endpoint currently has no visibility into collection health. Add `last_collection_at`, `collector_count`, and `collection_errors` fields.

- [ ] **Step 1: Add collection metadata to inventory Module**

In `internal/agent/inventory/collector.go`, add tracking fields:

```go
type Module struct {
	logger          *slog.Logger
	collectors      []packageCollector
	lastCollectedAt time.Time
	lastErrors      []string
	mu              sync.Mutex
}
```

Update `buildReport` to record timing:

```go
func (m *Module) buildReport(ctx context.Context) (*pb.InventoryReport, error) {
	// ... existing code ...

	// After all collectors run, record metadata.
	m.mu.Lock()
	m.lastCollectedAt = time.Now()
	m.lastErrors = nil
	for _, ce := range report.CollectionErrors {
		m.lastErrors = append(m.lastErrors, ce.Collector+": "+ce.ErrorMessage)
	}
	m.mu.Unlock()

	return report, nil
}
```

Add public accessors:

```go
// LastCollectedAt returns when the last inventory collection completed.
func (m *Module) LastCollectedAt() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastCollectedAt
}

// LastErrors returns collector errors from the last collection.
func (m *Module) LastErrors() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.lastErrors...)
}
```

- [ ] **Step 2: Update StatusInfo struct**

In `internal/agent/api/status.go`, add fields:

```go
type StatusInfo struct {
	AgentID              string     `json:"agent_id"`
	Hostname             string     `json:"hostname"`
	OSFamily             string     `json:"os_family"`
	OSVersion            string     `json:"os_version"`
	AgentVersion         string     `json:"agent_version"`
	EnrollmentStatus     string     `json:"enrollment_status"`
	ServerURL            string     `json:"server_url"`
	LastHeartbeat        *time.Time `json:"last_heartbeat"`
	UptimeSeconds        int64      `json:"uptime_seconds"`
	PendingPatchCount    int64      `json:"pending_patch_count"`
	InstalledCount       int64      `json:"installed_count"`
	FailedCount          int64      `json:"failed_count"`
	LastCollectionAt     *time.Time `json:"last_collection_at,omitempty"`
	CollectorCount       int        `json:"collector_count"`
	CollectionErrors     []string   `json:"collection_errors,omitempty"`
}
```

- [ ] **Step 3: Wire collection metadata into status provider**

This depends on how the status provider is constructed in `cmd/agent/main.go`. Find the status provider setup and add the inventory module reference so it can read `LastCollectedAt()` and `LastErrors()`.

- [ ] **Step 4: Verify build and run tests**

Run: `go build ./cmd/agent/... && go test ./internal/agent/api/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/inventory/collector.go internal/agent/api/status.go cmd/agent/main.go
git commit -m "feat(agent): expose collection health in status API

StatusInfo now includes last_collection_at, collector_count, and
collection_errors. Frontend can display collection freshness and
diagnose when collectors are failing."
```

---

### Task 14: Update buildReportWithProgress to Match buildReport Changes

**Files:**
- Modify: `internal/agent/inventory/collector.go`

The `buildReportWithProgress` method is a copy of `buildReport` with progress callbacks. It also needs the collection metadata tracking added in Task 13.

- [ ] **Step 1: Add metadata tracking to buildReportWithProgress**

At the end of `buildReportWithProgress`, before `return report, nil`:

```go
	m.mu.Lock()
	m.lastCollectedAt = time.Now()
	m.lastErrors = nil
	for _, ce := range report.CollectionErrors {
		m.lastErrors = append(m.lastErrors, ce.Collector+": "+ce.ErrorMessage)
	}
	m.mu.Unlock()
```

- [ ] **Step 2: Run all collector tests**

Run: `go test ./internal/agent/inventory/ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/agent/inventory/collector.go
git commit -m "fix(agent): track collection metadata in progress-aware collection path"
```

---

### Task 15: Integration Smoke Test

**Files:**
- None created, just verification.

End-to-end verification that the full pipeline works.

- [ ] **Step 1: Run all agent tests**

Run: `go test -race ./internal/agent/... -v`
Expected: PASS, no race conditions.

- [ ] **Step 2: Run all server tests**

Run: `go test -race ./internal/server/... -v`
Expected: PASS

- [ ] **Step 3: Build all binaries**

Run: `make build`
Expected: No errors.

- [ ] **Step 4: Run proto linting**

Run: `make proto`
Expected: No errors.

- [ ] **Step 5: Run full lint**

Run: `make lint`
Expected: No lint errors in changed files.

- [ ] **Step 6: Final commit if any fixups needed**

Fix any issues found in steps 1-5, then commit.

---

## Summary of Data Flow After Fixes

```
Windows Endpoint
├── detect_windows.go         — logs why collectors are unavailable
├── registryCollector          — populates publisher, install_date, architecture, implements extendedCollector
├── wuaCollector               — populates kb_article, severity, category, implements extendedCollector
├── wuaInstalledCollector      — same as wuaCollector
├── hotfixCollector            — populates kb_article, install_date, implements extendedCollector
├── pendingRebootCollector     — returns all sources with category labels
└── windowsFeaturesCollector   — unchanged

Collection → buildReport()
├── Collects from all active collectors
├── Records collection_errors for failed collectors
├── Tracks lastCollectedAt, lastErrors
└── Marshals to InventoryReport proto

Runner → collectOnce()
├── Writes to outbox
└── Saves ExtendedPackages to SQLite inventory_cache

Agent API → /api/v1/software
├── Returns in-memory ExtendedPackages
└── Falls back to SQLite cache if empty

Agent API → /api/v1/status
└── Includes last_collection_at, collector_count, collection_errors

gRPC Sync → Server processInventory()
├── Accepts partial reports (packages=0, errors>0)
├── Stores collection_errors JSONB
├── BulkInserts with kb_article, severity, install_date, category, publisher
└── Existing best-effort hardware/software/CVE pipeline
```
