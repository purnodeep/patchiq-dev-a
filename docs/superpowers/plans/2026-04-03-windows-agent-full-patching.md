# Windows Agent Full Patching E2E Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable Windows endpoints to fully inventory and patch all software (OS updates, system software, third-party apps) through the complete Hub → Server → Agent pipeline.

**Architecture:** Four-layer change: Proto adds `silent_args` field → Hub populates `installer_type` for MSRC feed and syncs it → Server stores and uses per-patch installer metadata in wave dispatcher → Agent adds WUA/Registry collectors and WUA/EXE installers. Each layer builds on the previous.

**Tech Stack:** Go 1.25, protobuf/gRPC, PostgreSQL (goose migrations, sqlc), COM interop via go-ole (Windows), golang.org/x/sys/windows/registry

**Spec:** `docs/superpowers/specs/2026-04-03-windows-agent-full-patching-design.md`

---

## File Map

### Proto
| File | Action | Responsibility |
|------|--------|---------------|
| `proto/patchiq/v1/common.proto` | Modify | Add `silent_args` field to `InstallPatchPayload` |

### Hub
| File | Action | Responsibility |
|------|--------|---------------|
| `internal/hub/store/migrations/017_add_silent_args.sql` | Create | Add `silent_args` column to `patch_catalog` |
| `internal/hub/feeds/msrc.go` | Modify | Set `InstallerType = "wua"` for MSRC entries |
| `internal/hub/store/queries/patch_catalog.sql` | Modify | Add `silent_args` to `UpsertCatalogEntryFromFeed` |
| `internal/hub/api/v1/sync.go` | Modify | Include `installer_type` + `silent_args` in sync response |

### Server
| File | Action | Responsibility |
|------|--------|---------------|
| `internal/server/store/migrations/048_add_installer_metadata.sql` | Create | Add `installer_type` + `silent_args` to `patches` + backfill |
| `internal/server/store/queries/patches.sql` | Modify | Add new columns to `UpsertDiscoveredPatch` |
| `internal/server/workers/catalog_sync.go` | Modify | Parse + store `installer_type` + `silent_args` from hub |
| `internal/server/deployment/wave_dispatcher.go` | Modify | Use per-patch `installer_type` instead of hardcoded source |

### Agent
| File | Action | Responsibility |
|------|--------|---------------|
| `internal/agent/inventory/detect_windows.go` | Modify | Add WUA + Registry collector detectors |
| `internal/agent/inventory/registry_windows.go` | Create | Windows registry scanner collector |
| `internal/agent/patcher/detect_windows.go` | Modify | Add WUA + EXE installer detectors |
| `internal/agent/patcher/wua_windows.go` | Create | WUA COM installer (download + install) |
| `internal/agent/patcher/exe_windows.go` | Create | EXE silent installer |

### Tests
| File | Action | Responsibility |
|------|--------|---------------|
| `internal/hub/feeds/msrc_test.go` | Modify | Verify `InstallerType = "wua"` |
| `internal/server/workers/catalog_sync_test.go` | Modify | Verify new fields parsed + stored |
| `internal/server/deployment/wave_dispatcher_test.go` | Modify | Verify per-patch source dispatch |
| `internal/agent/inventory/registry_windows_test.go` | Create | Registry collector unit tests |
| `internal/agent/inventory/wua_test.go` | Create/Modify | WUA collector detection tests |
| `internal/agent/patcher/wua_windows_test.go` | Create | WUA installer unit tests |
| `internal/agent/patcher/exe_windows_test.go` | Create | EXE installer unit tests |
| `internal/agent/patcher/detect_windows_test.go` | Create | Installer detection tests |

---

## Task 1: Proto — Add `silent_args` to `InstallPatchPayload`

**Files:**
- Modify: `proto/patchiq/v1/common.proto:149-163`

- [ ] **Step 1: Add `silent_args` field**

In `proto/patchiq/v1/common.proto`, add field 7 to `InstallPatchPayload`:

```protobuf
message InstallPatchPayload {
  repeated PatchTarget packages = 1;
  bool dry_run = 2;
  string pre_script = 3;
  string post_script = 4;
  string download_url = 5;
  string checksum_sha256 = 6;
  string silent_args = 7;
}
```

- [ ] **Step 2: Regenerate protobuf Go code**

Run: `make proto`
Expected: Clean regeneration, no errors.

- [ ] **Step 3: Verify generated code has the new field**

Run: `grep -n "SilentArgs" gen/patchiq/v1/common.pb.go | head -5`
Expected: `SilentArgs string` field appears in the generated struct.

- [ ] **Step 4: Commit**

```bash
git add proto/patchiq/v1/common.proto gen/patchiq/v1/
git commit -m "feat(proto): add silent_args field to InstallPatchPayload"
```

---

## Task 2: Hub Migration — Add `silent_args` Column

**Files:**
- Create: `internal/hub/store/migrations/017_add_silent_args.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
ALTER TABLE patch_catalog ADD COLUMN silent_args TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE patch_catalog DROP COLUMN IF EXISTS silent_args;
```

- [ ] **Step 2: Run hub migration**

Run: `make migrate-hub`
Expected: Migration 017 applied successfully.

- [ ] **Step 3: Verify column exists**

Run: `PGPASSWORD=$PATCHIQ_HUB_DB_PASSWORD psql -h localhost -p $PATCHIQ_HUB_DB_PORT -U $PATCHIQ_HUB_DB_USER -d $PATCHIQ_HUB_DB_NAME -c "\d patch_catalog" | grep silent_args`
Expected: `silent_args | text | not null | ''`

- [ ] **Step 4: Commit migration**

```bash
git add internal/hub/store/migrations/017_add_silent_args.sql
git commit -m "feat(hub): add silent_args column to patch_catalog"
```

- [ ] **Step 5: Update sqlc query to include `silent_args`**

In `internal/hub/store/queries/patch_catalog.sql`, modify `UpsertCatalogEntryFromFeed` to add the 15th parameter:

```sql
-- name: UpsertCatalogEntryFromFeed :one
INSERT INTO patch_catalog (name, vendor, os_family, version, severity, release_date, description, feed_source_id, source_url, installer_type, binary_ref, checksum_sha256, product, os_package_name, silent_args)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (feed_source_id, vendor, name, version)
    WHERE feed_source_id IS NOT NULL AND deleted_at IS NULL
DO UPDATE SET
    severity = EXCLUDED.severity,
    release_date = EXCLUDED.release_date,
    description = EXCLUDED.description,
    os_family = EXCLUDED.os_family,
    source_url = EXCLUDED.source_url,
    installer_type = EXCLUDED.installer_type,
    binary_ref = EXCLUDED.binary_ref,
    checksum_sha256 = EXCLUDED.checksum_sha256,
    product = EXCLUDED.product,
    os_package_name = EXCLUDED.os_package_name,
    silent_args = EXCLUDED.silent_args,
    updated_at = now()
RETURNING *;
```

- [ ] **Step 6: Regenerate sqlc**

Run: `make sqlc`
Expected: Clean regeneration. `UpsertCatalogEntryFromFeedParams` struct now has `SilentArgs` field.

- [ ] **Step 7: Commit query + generated code**

```bash
git add internal/hub/store/queries/patch_catalog.sql internal/hub/store/sqlcgen/
git commit -m "feat(hub): add silent_args to UpsertCatalogEntryFromFeed query"
```

- [ ] **Step 8: Update pipeline.go to pass `silent_args`**

In `internal/hub/catalog/pipeline.go`, at the `UpsertCatalogEntryFromFeed` call (~line 95-108), add `SilentArgs`:

```go
catalogEntry, uErr := p.store.UpsertCatalogEntryFromFeed(ctx, sqlcgen.UpsertCatalogEntryFromFeedParams{
    Name:          entry.Name,
    Vendor:        entry.Vendor,
    OsFamily:      entry.OSFamily,
    Version:       entry.Version,
    Severity:      entry.Severity,
    ReleaseDate:   pgtype.Timestamptz{Time: entry.ReleaseDate, Valid: !entry.ReleaseDate.IsZero()},
    Description:   pgtype.Text{String: entry.Summary, Valid: entry.Summary != ""},
    FeedSourceID:  source.ID,
    SourceUrl:     entry.SourceURL,
    InstallerType: entry.InstallerType,
    Product:       entry.Product,
    OsPackageName: osPackageName,
    SilentArgs:    entry.SilentArgs,
})
```

Note: `RawEntry` in the feeds package needs `SilentArgs` field — add it if not present.

- [ ] **Step 9: Add `SilentArgs` to `RawEntry` struct if missing**

Check `internal/hub/feeds/types.go` (or wherever `RawEntry` is defined). If `SilentArgs string` is not already a field, add it.

- [ ] **Step 10: Verify hub builds**

Run: `go build ./internal/hub/...`
Expected: Clean build, no errors.

- [ ] **Step 11: Commit pipeline + RawEntry wiring**

```bash
git add internal/hub/catalog/pipeline.go internal/hub/feeds/
git commit -m "feat(hub): wire silent_args through catalog pipeline and RawEntry"
```

---

## Task 3: Hub — MSRC Feed Sets `installer_type`

**Files:**
- Modify: `internal/hub/feeds/msrc.go:130-140`
- Test: `internal/hub/feeds/msrc_test.go`

- [ ] **Step 1: Write failing test**

In `internal/hub/feeds/msrc_test.go`, add a test verifying MSRC entries get `InstallerType = "wua"`:

```go
func TestMSRCFeed_InstallerType(t *testing.T) {
	// Use existing test fixture or create minimal MSRC JSON response
	feed := &MSRCFeed{logger: slog.Default()}

	// Minimal valid MSRC response with one update + one vulnerability
	data := []byte(`{
		"value": [{
			"ID": "2024-Feb",
			"InitialReleaseDate": "2024-02-13T08:00:00Z",
			"Vulnerabilities": [{
				"CVE": "CVE-2024-1234",
				"Title": "Windows Kernel Elevation of Privilege Vulnerability",
				"Severity": "Important",
				"AffectedProducts": ["Windows 11"],
				"KBArticles": [{"ID": "5034765", "URL": "https://support.microsoft.com/kb/5034765"}]
			}]
		}]
	}`)

	entries, err := feed.parse(data)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "wua", entries[0].InstallerType)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hub/feeds/ -run TestMSRCFeed_InstallerType -v`
Expected: FAIL — `InstallerType` is empty string, not `"wua"`.

- [ ] **Step 3: Set `InstallerType` in MSRC parse function**

In `internal/hub/feeds/msrc.go`, in the `parse` function (~line 130), add `InstallerType: "wua"` to the `RawEntry` literal:

```go
entry := RawEntry{
    CVEs:          []string{vuln.CVE},
    Name:          vuln.Title,
    Vendor:        "microsoft",
    OSFamily:      "windows",
    Severity:      strings.ToLower(vuln.Severity),
    ReleaseDate:   releaseDate,
    InstallerType: "wua",
    Metadata: map[string]string{
        "update_id": update.ID,
    },
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hub/feeds/ -run TestMSRCFeed_InstallerType -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/hub/feeds/msrc.go internal/hub/feeds/msrc_test.go
git commit -m "feat(hub): set installer_type=wua for MSRC feed entries"
```

---

## Task 4: Hub Sync — Include `installer_type` and `silent_args` in Response

**Files:**
- Modify: `internal/hub/api/v1/sync.go:51-56`

The sync handler returns `[]sqlcgen.PatchCatalog` directly as JSON (`syncResponse.Entries`). The `PatchCatalog` struct is generated by sqlc and already includes `InstallerType` and `SilentArgs` columns (after migration + sqlc regen). So the sync response **automatically includes these fields** — they're part of the generated model that gets serialized.

- [ ] **Step 1: Verify sync response includes new fields**

After hub migration + sqlc regen from Task 2, check that `sqlcgen.PatchCatalog` has the fields:

Run: `grep -n "InstallerType\|SilentArgs" internal/hub/store/sqlcgen/models.go`
Expected: Both fields appear in the `PatchCatalog` struct.

- [ ] **Step 2: Verify JSON serialization includes the fields**

The `syncResponse.Entries` field is `[]sqlcgen.PatchCatalog` which serializes all fields. No code change needed — confirm by checking the struct tags:

Run: `grep -A2 "InstallerType\|SilentArgs" internal/hub/store/sqlcgen/models.go`
Expected: Fields have JSON tags (sqlc generates them automatically).

- [ ] **Step 3: Commit (if any changes were needed)**

If the fields are already included via sqlc generation, no commit needed for this task. Mark as done.

---

## Task 5: Server Migration — Add `installer_type` + `silent_args` + Backfill

**Files:**
- Create: `internal/server/store/migrations/048_add_installer_metadata.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
ALTER TABLE patches ADD COLUMN installer_type TEXT NOT NULL DEFAULT '';
ALTER TABLE patches ADD COLUMN silent_args TEXT NOT NULL DEFAULT '';

-- Backfill existing Windows patches with heuristic installer_type
UPDATE patches SET installer_type = CASE
    WHEN os_family != 'windows' THEN installer_type
    WHEN name ~* '(^KB|Cumulative Update|Security Update|Servicing Stack|\.NET Framework)' THEN 'wua'
    WHEN package_url IS NOT NULL AND package_url LIKE '%.msi' THEN 'msi'
    WHEN package_url IS NOT NULL AND package_url LIKE '%.msix' THEN 'msix'
    WHEN package_url IS NOT NULL AND package_url LIKE '%.appx' THEN 'msix'
    WHEN package_url IS NOT NULL AND package_url LIKE '%.exe' THEN 'exe'
    ELSE 'wua'
END
WHERE os_family = 'windows' AND installer_type = '';

-- +goose Down
ALTER TABLE patches DROP COLUMN IF EXISTS installer_type;
ALTER TABLE patches DROP COLUMN IF EXISTS silent_args;
```

- [ ] **Step 2: Run server migration**

Run: `make migrate`
Expected: Migration 048 applied successfully.

- [ ] **Step 3: Verify columns and backfill**

Run: `PGPASSWORD=$PATCHIQ_DB_PASSWORD psql -h localhost -p $PATCHIQ_DB_PORT -U $PATCHIQ_DB_USER -d $PATCHIQ_DB_NAME -c "SELECT installer_type, count(*) FROM patches WHERE os_family='windows' GROUP BY installer_type;"`
Expected: Rows grouped by installer_type (mostly 'wua', some 'msi'/'exe' if applicable).

- [ ] **Step 4: Commit**

```bash
git add internal/server/store/migrations/048_add_installer_metadata.sql
git commit -m "feat(server): add installer_type + silent_args columns with Windows backfill"
```

---

## Task 6: Server — Update `UpsertDiscoveredPatch` Query + Catalog Sync

**Files:**
- Modify: `internal/server/store/queries/patches.sql:51-65`
- Modify: `internal/server/workers/catalog_sync.go:49-62, 271-296`
- Test: `internal/server/workers/catalog_sync_test.go`

- [ ] **Step 1: Write failing test**

In `internal/server/workers/catalog_sync_test.go`, add a test that verifies `installer_type` and `silent_args` are parsed from hub sync response and passed to `UpsertDiscoveredPatch`:

```go
func TestCatalogSync_ParsesInstallerMetadata(t *testing.T) {
	entry := `{
		"name": "KB5034765",
		"vendor": "microsoft",
		"os_family": "windows",
		"version": "10.0.22621.3155",
		"severity": "critical",
		"description": "Cumulative Update",
		"installer_type": "wua",
		"silent_args": "",
		"checksum_sha256": "",
		"binary_ref": "",
		"source_url": "https://support.microsoft.com/kb/5034765",
		"product": "Windows 11",
		"os_package_name": ""
	}`

	var ce catalogEntry
	err := json.Unmarshal([]byte(entry), &ce)
	require.NoError(t, err)
	assert.Equal(t, "wua", ce.InstallerType)
	assert.Equal(t, "", ce.SilentArgs)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/workers/ -run TestCatalogSync_ParsesInstallerMetadata -v`
Expected: FAIL — `catalogEntry` struct doesn't have `InstallerType` or `SilentArgs` fields.

- [ ] **Step 3: Add fields to `catalogEntry` struct**

In `internal/server/workers/catalog_sync.go` (~line 49-62), add the two new fields:

```go
type catalogEntry struct {
	Name           string `json:"name"`
	Vendor         string `json:"vendor"`
	OsFamily       string `json:"os_family"`
	Version        string `json:"version"`
	Severity       string `json:"severity"`
	Description    string `json:"description"`
	BinaryRef      string `json:"binary_ref"`
	ChecksumSha256 string `json:"checksum_sha256"`
	SourceUrl      string `json:"source_url"`
	Product        string `json:"product"`
	OsPackageName  string `json:"os_package_name"`
	InstallerType  string `json:"installer_type"`
	SilentArgs     string `json:"silent_args"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server/workers/ -run TestCatalogSync_ParsesInstallerMetadata -v`
Expected: PASS

- [ ] **Step 5: Commit struct update + test**

```bash
git add internal/server/workers/catalog_sync.go internal/server/workers/catalog_sync_test.go
git commit -m "feat(server): add installer_type + silent_args to catalogEntry struct"
```

- [ ] **Step 6: Update SQL query to include new columns**

In `internal/server/store/queries/patches.sql`, modify `UpsertDiscoveredPatch` (~line 51-65):

```sql
-- name: UpsertDiscoveredPatch :one
INSERT INTO patches (tenant_id, name, version, severity, os_family, status,
    os_distribution, package_url, checksum_sha256, source_repo, description, package_name, installer_type, silent_args)
VALUES ($1, $2, $3, $4, $5, 'available', $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (tenant_id, name, version, os_family)
DO UPDATE SET
    severity = EXCLUDED.severity,
    os_distribution = EXCLUDED.os_distribution,
    package_url = EXCLUDED.package_url,
    checksum_sha256 = EXCLUDED.checksum_sha256,
    source_repo = EXCLUDED.source_repo,
    description = EXCLUDED.description,
    package_name = EXCLUDED.package_name,
    installer_type = EXCLUDED.installer_type,
    silent_args = EXCLUDED.silent_args,
    updated_at = now()
RETURNING *;
```

- [ ] **Step 7: Regenerate sqlc**

Run: `make sqlc`
Expected: `UpsertDiscoveredPatchParams` struct now has `InstallerType` and `SilentArgs` fields.

- [ ] **Step 8: Commit query + generated code**

```bash
git add internal/server/store/queries/patches.sql internal/server/store/sqlcgen/
git commit -m "feat(server): add installer_type + silent_args to UpsertDiscoveredPatch query"
```

- [ ] **Step 9: Update `upsertEntries` to pass new fields**

In `internal/server/workers/catalog_sync.go` (~line 271-296), add the new params to the `UpsertDiscoveredPatch` call:

```go
_, err := qtx.UpsertDiscoveredPatch(ctx, sqlcgen.UpsertDiscoveredPatchParams{
    TenantID:       tenantID,
    Name:           entry.Name,
    Version:        entry.Version,
    Severity:       entry.Severity,
    OsFamily:       entry.OsFamily,
    Description:    pgtype.Text{String: entry.Description, Valid: entry.Description != ""},
    SourceRepo:     pgtype.Text{String: entry.Vendor, Valid: entry.Vendor != ""},
    PackageUrl:     pgtype.Text{String: entry.BinaryRef, Valid: entry.BinaryRef != ""},
    ChecksumSha256: pgtype.Text{String: entry.ChecksumSha256, Valid: entry.ChecksumSha256 != ""},
    PackageName:    resolvePackageName(entry),
    InstallerType:  entry.InstallerType,
    SilentArgs:     entry.SilentArgs,
})
```

- [ ] **Step 10: Verify server builds**

Run: `go build ./internal/server/...`
Expected: Clean build.

- [ ] **Step 11: Commit upsertEntries wiring**

```bash
git add internal/server/workers/catalog_sync.go
git commit -m "feat(server): wire installer_type + silent_args through catalog sync upsert"
```

---

## Task 7: Server — Wave Dispatcher Uses Per-Patch Installer Type

**Files:**
- Modify: `internal/server/deployment/wave_dispatcher.go:278-295, 507-520`
- Test: `internal/server/deployment/wave_dispatcher_test.go`

- [ ] **Step 1: Write failing test**

In `internal/server/deployment/wave_dispatcher_test.go`, add a test verifying that a Windows patch with `installer_type = "wua"` produces a payload with `Source: "wua"` (not `"msi"`):

```go
func TestInstallerTypeOrFallback(t *testing.T) {
	tests := []struct {
		name          string
		installerType string
		osFamily      string
		wantSource    string
	}{
		{"wua from installer_type", "wua", "windows", "wua"},
		{"exe from installer_type", "exe", "windows", "exe"},
		{"fallback for empty windows", "", "windows", "msi"},
		{"fallback for linux", "", "linux-ubuntu", "apt"},
		{"apt from installer_type", "apt", "linux-debian", "apt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := installerTypeOrFallback(tt.installerType, tt.osFamily)
			assert.Equal(t, tt.wantSource, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/deployment/ -run TestInstallerTypeOrFallback -v`
Expected: FAIL — `installerTypeOrFallback` does not exist.

- [ ] **Step 3: Implement `installerTypeOrFallback`**

In `internal/server/deployment/wave_dispatcher.go`, add the function near `osFamilyToSource` (~line 520):

```go
// installerTypeOrFallback returns the patch's installer_type if set,
// otherwise falls back to the legacy OS-family-based mapping.
func installerTypeOrFallback(installerType, osFamily string) string {
	if installerType != "" {
		return installerType
	}
	return osFamilyToSource(osFamily)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server/deployment/ -run TestInstallerTypeOrFallback -v`
Expected: PASS

- [ ] **Step 5: Update payload construction to use new function**

In `internal/server/deployment/wave_dispatcher.go` (~line 278-284), replace `osFamilyToSource`:

```go
installPayload := &pb.InstallPatchPayload{
    Packages: []*pb.PatchTarget{{
        Name:    pkgName,
        Version: patch.Version,
        Source:  installerTypeOrFallback(patch.InstallerType, patch.OsFamily),
    }},
}
```

And after the download_url/checksum block (~line 295), add:

```go
if patch.SilentArgs != "" {
    installPayload.SilentArgs = patch.SilentArgs
}
```

- [ ] **Step 6: Verify server builds and tests pass**

Run: `go test ./internal/server/deployment/ -v -count=1`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/server/deployment/wave_dispatcher.go internal/server/deployment/wave_dispatcher_test.go
git commit -m "feat(server): use per-patch installer_type in wave dispatcher"
```

---

## Task 8: Agent — Register WUA Collector

**Files:**
- Modify: `internal/agent/inventory/detect_windows.go`
- Test: `internal/agent/inventory/wua_test.go` (create if not exists)

- [ ] **Step 1: Write failing test for WUA detection**

Create `internal/agent/inventory/wua_test.go` (with `//go:build windows` or use a portable mock test):

```go
package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// mockSearcher implements updateSearcher for testing.
type mockSearcher struct {
	updates []windowsUpdate
	err     error
}

func (m *mockSearcher) Search(_ context.Context, _ string) ([]windowsUpdate, error) {
	return m.updates, m.err
}

func TestWUACollector_Name(t *testing.T) {
	c := &wuaCollector{}
	assert.Equal(t, "wua", c.Name())
}

func TestWUACollector_Collect(t *testing.T) {
	c := &wuaCollector{
		searcher: &mockSearcher{
			updates: []windowsUpdate{
				{Title: "2024-02 Cumulative Update", KBID: "KB5034765", Severity: "Critical"},
			},
		},
	}

	pkgs, err := c.Collect(context.Background())
	assert.NoError(t, err)
	assert.Len(t, pkgs, 1)
	assert.Equal(t, "KB5034765", pkgs[0].Name)
	assert.Equal(t, "wua", pkgs[0].Source)
}
```

Note: This test file may need `//go:build windows` since `wuaCollector` is in a windows-only file. If the project has a pattern for cross-platform testing of windows types, follow it. Otherwise, create this as a windows-only test.

- [ ] **Step 2: Add WUA detector to `detect_windows.go`**

In `internal/agent/inventory/detect_windows.go`, add `detectWUACollector` and register it:

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
	}
}

func detectHotFixCollector() packageCollector {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return nil
	}
	return &hotfixCollector{runner: &execRunner{}}
}

func detectWUACollector() packageCollector {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return nil
	}
	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		ole.CoUninitialize()
		return nil
	}
	unknown.Release()
	ole.CoUninitialize()

	return &wuaCollector{
		searcher: &comSearcher{logger: slog.Default()},
		logger:   slog.Default(),
	}
}
```

- [ ] **Step 3: Verify build compiles (cross-compile check)**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./internal/agent/...`
Expected: Clean build (or expected errors only from cgo-dependent darwin files).

- [ ] **Step 4: Commit**

```bash
git add internal/agent/inventory/detect_windows.go internal/agent/inventory/wua_test.go
git commit -m "feat(agent): register WUA collector in Windows platform detection"
```

---

## Task 9: Agent — Registry Collector

**Files:**
- Create: `internal/agent/inventory/registry_windows.go`
- Create: `internal/agent/inventory/registry_windows_test.go`
- Modify: `internal/agent/inventory/detect_windows.go`

- [ ] **Step 1: Write failing test**

Create `internal/agent/inventory/registry_windows_test.go`:

```go
//go:build windows

package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryReader abstracts Windows registry access for testability.
type mockRegistryReader struct {
	entries []registryEntry
}

func (m *mockRegistryReader) ReadUninstallKeys() ([]registryEntry, error) {
	return m.entries, nil
}

func TestRegistryCollector_Name(t *testing.T) {
	c := &registryCollector{}
	assert.Equal(t, "registry", c.Name())
}

func TestRegistryCollector_Collect(t *testing.T) {
	c := &registryCollector{
		reader: &mockRegistryReader{
			entries: []registryEntry{
				{DisplayName: "Google Chrome", DisplayVersion: "122.0.6261.94", Publisher: "Google LLC"},
				{DisplayName: "7-Zip", DisplayVersion: "23.01", Publisher: "Igor Pavlov"},
				{DisplayName: "", DisplayVersion: "1.0", Publisher: "System"},
			},
		},
	}

	pkgs, err := c.Collect(context.Background())
	require.NoError(t, err)
	assert.Len(t, pkgs, 2, "should skip entry with empty DisplayName")
	assert.Equal(t, "Google Chrome", pkgs[0].Name)
	assert.Equal(t, "122.0.6261.94", pkgs[0].Version)
	assert.Equal(t, "registry", pkgs[0].Source)
}

func TestRegistryCollector_Dedup(t *testing.T) {
	c := &registryCollector{
		reader: &mockRegistryReader{
			entries: []registryEntry{
				{DisplayName: "App", DisplayVersion: "1.0", Publisher: "Vendor", Is64Bit: true},
				{DisplayName: "App", DisplayVersion: "1.0", Publisher: "Vendor", Is64Bit: false},
			},
		},
	}

	pkgs, err := c.Collect(context.Background())
	require.NoError(t, err)
	assert.Len(t, pkgs, 1, "should deduplicate same app across 64/32-bit paths")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOOS=windows go test ./internal/agent/inventory/ -run TestRegistryCollector -v` (will fail because types don't exist)

- [ ] **Step 3: Implement registry collector**

Create `internal/agent/inventory/registry_windows.go`:

```go
//go:build windows

package inventory

import (
	"context"
	"fmt"
	"log/slog"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"golang.org/x/sys/windows/registry"
)

// registryEntry represents a single installed program from the Uninstall registry.
type registryEntry struct {
	DisplayName    string
	DisplayVersion string
	Publisher      string
	InstallDate    string
	Is64Bit        bool
}

// registryReaderIface abstracts registry access for testability.
type registryReaderIface interface {
	ReadUninstallKeys() ([]registryEntry, error)
}

// winRegistryReader reads from the actual Windows registry.
type winRegistryReader struct {
	logger *slog.Logger
}

func (r *winRegistryReader) ReadUninstallKeys() ([]registryEntry, error) {
	paths := []struct {
		key     registry.Key
		path    string
		is64Bit bool
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, true},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, false},
	}

	var entries []registryEntry
	for _, p := range paths {
		key, err := registry.OpenKey(p.key, p.path, registry.READ)
		if err != nil {
			r.logger.Warn("skip registry path", "path", p.path, "error", err)
			continue
		}

		subkeys, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			r.logger.Warn("read subkeys failed", "path", p.path, "error", err)
			continue
		}

		for _, subkeyName := range subkeys {
			subkey, err := registry.OpenKey(p.key, p.path+`\`+subkeyName, registry.READ)
			if err != nil {
				continue
			}

			entry := registryEntry{Is64Bit: p.is64Bit}
			entry.DisplayName, _, _ = subkey.GetStringValue("DisplayName")
			entry.DisplayVersion, _, _ = subkey.GetStringValue("DisplayVersion")
			entry.Publisher, _, _ = subkey.GetStringValue("Publisher")
			entry.InstallDate, _, _ = subkey.GetStringValue("InstallDate")
			subkey.Close()

			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// registryCollector implements packageCollector using the Windows registry.
type registryCollector struct {
	reader registryReaderIface
	logger *slog.Logger
}

func (c *registryCollector) Name() string { return "registry" }

func (c *registryCollector) Collect(_ context.Context) ([]*pb.PackageInfo, error) {
	entries, err := c.reader.ReadUninstallKeys()
	if err != nil {
		return nil, fmt.Errorf("registry collector: %w", err)
	}

	seen := make(map[string]struct{})
	var pkgs []*pb.PackageInfo

	for _, e := range entries {
		if e.DisplayName == "" {
			continue
		}
		dedupKey := e.DisplayName + "|" + e.DisplayVersion
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}

		pkgs = append(pkgs, &pb.PackageInfo{
			Name:    e.DisplayName,
			Version: e.DisplayVersion,
			Source:  "registry",
		})
	}

	return pkgs, nil
}
```

- [ ] **Step 4: Register registry collector in `detect_windows.go`**

Add to the init function in `internal/agent/inventory/detect_windows.go`:

```go
func init() {
	platformCollectorDetectors = []collectorDetectorFunc{
		detectHotFixCollector,
		detectWUACollector,
		detectRegistryCollector,
	}
}

func detectRegistryCollector() packageCollector {
	return &registryCollector{
		reader: &winRegistryReader{logger: slog.Default()},
		logger: slog.Default(),
	}
}
```

- [ ] **Step 5: Verify cross-compile**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./internal/agent/...`
Expected: Clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/agent/inventory/registry_windows.go internal/agent/inventory/registry_windows_test.go internal/agent/inventory/detect_windows.go
git commit -m "feat(agent): add Windows registry collector for installed software inventory"
```

---

## Task 10: Agent — WUA Installer

**Files:**
- Create: `internal/agent/patcher/wua_windows.go`
- Create: `internal/agent/patcher/wua_windows_test.go`
- Modify: `internal/agent/patcher/detect_windows.go`

- [ ] **Step 1: Write failing test**

Create `internal/agent/patcher/wua_windows_test.go`:

```go
//go:build windows

package patcher

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWUAClient mocks the WUA COM interface for testing.
type mockWUAClient struct {
	searchResults []wuaUpdate
	searchErr     error
	downloadErr   error
	installErr    error
	installResult wuaInstallResult
}

func (m *mockWUAClient) SearchUpdates(_ context.Context, criteria string) ([]wuaUpdate, error) {
	return m.searchResults, m.searchErr
}

func (m *mockWUAClient) DownloadUpdates(_ context.Context, updates []wuaUpdate) error {
	return m.downloadErr
}

func (m *mockWUAClient) InstallUpdates(_ context.Context, updates []wuaUpdate) (wuaInstallResult, error) {
	return m.installResult, m.installErr
}

func TestWUAInstaller_Name(t *testing.T) {
	inst := &wuaInstaller{}
	assert.Equal(t, "wua", inst.Name())
}

func TestWUAInstaller_Install_Success(t *testing.T) {
	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{{Title: "KB5034765", UpdateID: "abc-123"}},
			installResult: wuaInstallResult{ResultCode: 2, RebootRequired: false},
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: "KB5034765"}, false)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.False(t, result.RebootRequired)
}

func TestWUAInstaller_Install_RebootRequired(t *testing.T) {
	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{{Title: "KB5034765", UpdateID: "abc-123"}},
			installResult: wuaInstallResult{ResultCode: 2, RebootRequired: true},
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: "KB5034765"}, false)
	require.NoError(t, err)
	assert.True(t, result.RebootRequired)
}

func TestWUAInstaller_Install_NotFound(t *testing.T) {
	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{},
		},
	}

	_, err := inst.Install(context.Background(), PatchTarget{Name: "KB9999999"}, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching update found")
}

func TestWUAInstaller_Install_DryRun(t *testing.T) {
	installed := false
	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{{Title: "KB5034765", UpdateID: "abc-123"}},
			installResult: wuaInstallResult{ResultCode: 2},
			installErr:    fmt.Errorf("should not be called"),
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: "KB5034765"}, true)
	require.NoError(t, err)
	assert.False(t, installed)
	assert.Equal(t, 0, result.ExitCode)
}
```

- [ ] **Step 2: Implement WUA installer**

Create `internal/agent/patcher/wua_windows.go`:

```go
//go:build windows

package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// wuaUpdate represents a Windows Update found by search.
type wuaUpdate struct {
	Title    string
	UpdateID string
}

// wuaInstallResult captures the COM IInstallationResult.
type wuaInstallResult struct {
	ResultCode     int // 2=succeeded, 3=succeeded with errors, 4=failed, 5=aborted
	RebootRequired bool
}

// wuaClientIface abstracts WUA COM operations for testability.
type wuaClientIface interface {
	SearchUpdates(ctx context.Context, criteria string) ([]wuaUpdate, error)
	DownloadUpdates(ctx context.Context, updates []wuaUpdate) error
	InstallUpdates(ctx context.Context, updates []wuaUpdate) (wuaInstallResult, error)
}

// wuaInstaller installs Windows Updates via the WUA COM API.
type wuaInstaller struct {
	client wuaClientIface
	logger *slog.Logger
}

func (w *wuaInstaller) Name() string { return "wua" }

func (w *wuaInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	// Search for the update by name (KB article ID or title)
	criteria := fmt.Sprintf("IsInstalled=0 and Type='Software'")
	updates, err := w.client.SearchUpdates(ctx, criteria)
	if err != nil {
		return InstallResult{}, fmt.Errorf("wua search: %w", err)
	}

	// Find matching update
	var matched []wuaUpdate
	for _, u := range updates {
		if strings.Contains(u.Title, pkg.Name) || strings.Contains(u.UpdateID, pkg.Name) {
			matched = append(matched, u)
		}
	}
	if len(matched) == 0 {
		return InstallResult{}, fmt.Errorf("wua install %s: no matching update found", pkg.Name)
	}

	// Download
	if err := w.client.DownloadUpdates(ctx, matched); err != nil {
		return InstallResult{}, fmt.Errorf("wua download %s: %w", pkg.Name, err)
	}

	// Dry-run: stop after download
	if dryRun {
		return InstallResult{
			Stdout: []byte(fmt.Sprintf("dry-run: downloaded %d update(s) for %s", len(matched), pkg.Name)),
		}, nil
	}

	// Install
	result, err := w.client.InstallUpdates(ctx, matched)
	if err != nil {
		return InstallResult{}, fmt.Errorf("wua install %s: %w", pkg.Name, err)
	}

	exitCode := 0
	if result.ResultCode == 4 || result.ResultCode == 5 {
		exitCode = result.ResultCode
	}

	return InstallResult{
		Stdout:         []byte(fmt.Sprintf("WUA result code: %d", result.ResultCode)),
		ExitCode:       exitCode,
		RebootRequired: result.RebootRequired,
	}, nil
}

// comWUAClient implements wuaClientIface using real COM interop.
type comWUAClient struct {
	logger *slog.Logger
}

func (c *comWUAClient) SearchUpdates(_ context.Context, criteria string) ([]wuaUpdate, error) {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return nil, fmt.Errorf("COM init: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return nil, fmt.Errorf("create update session: %w", err)
	}
	session, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}
	defer session.Release()

	searcherDisp, err := oleutil.CallMethod(session, "CreateUpdateSearcher")
	if err != nil {
		return nil, fmt.Errorf("create searcher: %w", err)
	}
	searcher := searcherDisp.ToIDispatch()
	defer searcher.Release()

	resultDisp, err := oleutil.CallMethod(searcher, "Search", criteria)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	result := resultDisp.ToIDispatch()
	defer result.Release()

	updatesDisp, err := oleutil.GetProperty(result, "Updates")
	if err != nil {
		return nil, fmt.Errorf("get updates: %w", err)
	}
	updatesCol := updatesDisp.ToIDispatch()
	defer updatesCol.Release()

	countVar, err := oleutil.GetProperty(updatesCol, "Count")
	if err != nil {
		return nil, fmt.Errorf("get count: %w", err)
	}
	count := int(countVar.Val)

	var updates []wuaUpdate
	for i := range count {
		itemDisp, err := oleutil.GetProperty(updatesCol, "Item", i)
		if err != nil {
			continue
		}
		item := itemDisp.ToIDispatch()
		wu := wuaUpdate{}
		if title, err := oleutil.GetProperty(item, "Title"); err == nil {
			wu.Title = title.ToString()
		}
		if id, err := oleutil.CallMethod(item, "Identity"); err == nil {
			idDisp := id.ToIDispatch()
			if uid, err := oleutil.GetProperty(idDisp, "UpdateID"); err == nil {
				wu.UpdateID = uid.ToString()
			}
			idDisp.Release()
		}
		item.Release()
		updates = append(updates, wu)
	}

	return updates, nil
}

func (c *comWUAClient) DownloadUpdates(_ context.Context, updates []wuaUpdate) error {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return fmt.Errorf("COM init: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	session, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("query session: %w", err)
	}
	defer session.Release()

	downloaderDisp, err := oleutil.CallMethod(session, "CreateUpdateDownloader")
	if err != nil {
		return fmt.Errorf("create downloader: %w", err)
	}
	downloader := downloaderDisp.ToIDispatch()
	defer downloader.Release()

	// Create update collection and add matched updates
	collDisp, err := oleutil.CreateObject("Microsoft.Update.UpdateColl")
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	coll, err := collDisp.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("query collection: %w", err)
	}
	defer coll.Release()

	// Re-search and add matching updates to the collection
	searcherDisp, err := oleutil.CallMethod(session, "CreateUpdateSearcher")
	if err != nil {
		return fmt.Errorf("create searcher for download: %w", err)
	}
	searcher := searcherDisp.ToIDispatch()
	defer searcher.Release()

	resultDisp, err := oleutil.CallMethod(searcher, "Search", "IsInstalled=0 and Type='Software'")
	if err != nil {
		return fmt.Errorf("search for download: %w", err)
	}
	result := resultDisp.ToIDispatch()
	defer result.Release()

	updatesDisp, err := oleutil.GetProperty(result, "Updates")
	if err != nil {
		return fmt.Errorf("get updates for download: %w", err)
	}
	updatesCol := updatesDisp.ToIDispatch()
	defer updatesCol.Release()

	countVar, _ := oleutil.GetProperty(updatesCol, "Count")
	count := int(countVar.Val)

	matchIDs := make(map[string]bool, len(updates))
	for _, u := range updates {
		matchIDs[u.UpdateID] = true
	}

	for i := range count {
		itemDisp, err := oleutil.GetProperty(updatesCol, "Item", i)
		if err != nil {
			continue
		}
		item := itemDisp.ToIDispatch()
		if id, err := oleutil.CallMethod(item, "Identity"); err == nil {
			idDisp := id.ToIDispatch()
			if uid, err := oleutil.GetProperty(idDisp, "UpdateID"); err == nil {
				if matchIDs[uid.ToString()] {
					oleutil.CallMethod(coll, "Add", item)
				}
			}
			idDisp.Release()
		}
		item.Release()
	}

	oleutil.PutProperty(downloader, "Updates", coll)
	_, err = oleutil.CallMethod(downloader, "Download")
	return err
}

func (c *comWUAClient) InstallUpdates(_ context.Context, updates []wuaUpdate) (wuaInstallResult, error) {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return wuaInstallResult{}, fmt.Errorf("COM init: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("create session: %w", err)
	}
	session, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("query session: %w", err)
	}
	defer session.Release()

	installerDisp, err := oleutil.CallMethod(session, "CreateUpdateInstaller")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("create installer: %w", err)
	}
	installer := installerDisp.ToIDispatch()
	defer installer.Release()

	// Build collection of downloaded updates (same search + filter as download)
	collDisp, err := oleutil.CreateObject("Microsoft.Update.UpdateColl")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("create collection: %w", err)
	}
	coll, err := collDisp.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("query collection: %w", err)
	}
	defer coll.Release()

	searcherDisp, err := oleutil.CallMethod(session, "CreateUpdateSearcher")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("create searcher: %w", err)
	}
	searcher := searcherDisp.ToIDispatch()
	defer searcher.Release()

	resultDisp, err := oleutil.CallMethod(searcher, "Search", "IsInstalled=0 and Type='Software'")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("search for install: %w", err)
	}
	result := resultDisp.ToIDispatch()
	defer result.Release()

	updatesDisp, err := oleutil.GetProperty(result, "Updates")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("get updates: %w", err)
	}
	updatesCol := updatesDisp.ToIDispatch()
	defer updatesCol.Release()

	countVar, _ := oleutil.GetProperty(updatesCol, "Count")
	count := int(countVar.Val)

	matchIDs := make(map[string]bool, len(updates))
	for _, u := range updates {
		matchIDs[u.UpdateID] = true
	}

	for i := range count {
		itemDisp, err := oleutil.GetProperty(updatesCol, "Item", i)
		if err != nil {
			continue
		}
		item := itemDisp.ToIDispatch()
		if id, err := oleutil.CallMethod(item, "Identity"); err == nil {
			idDisp := id.ToIDispatch()
			if uid, err := oleutil.GetProperty(idDisp, "UpdateID"); err == nil {
				if matchIDs[uid.ToString()] {
					oleutil.CallMethod(coll, "Add", item)
				}
			}
			idDisp.Release()
		}
		item.Release()
	}

	oleutil.PutProperty(installer, "Updates", coll)

	installResultDisp, err := oleutil.CallMethod(installer, "Install")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("install: %w", err)
	}
	installResult := installResultDisp.ToIDispatch()
	defer installResult.Release()

	rcVar, _ := oleutil.GetProperty(installResult, "ResultCode")
	rebootVar, _ := oleutil.GetProperty(installResult, "RebootRequired")

	return wuaInstallResult{
		ResultCode:     int(rcVar.Val),
		RebootRequired: rebootVar.Val != 0,
	}, nil
}
```

- [ ] **Step 3: Register WUA installer in `detect_windows.go`**

In `internal/agent/patcher/detect_windows.go`, add the detector:

```go
//go:build windows

package patcher

import (
	"log/slog"
	"os/exec"
)

func init() {
	platformInstallerDetectors = []installerDetectorFunc{
		detectMSIInstaller,
		detectMSIXInstaller,
		detectWUAInstaller,
	}
}

func detectMSIInstaller(executor CommandExecutor) Installer {
	if _, err := exec.LookPath("msiexec"); err != nil {
		return nil
	}
	return &msiInstaller{executor: executor}
}

func detectMSIXInstaller(executor CommandExecutor) Installer {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return nil
	}
	return &msixInstaller{executor: executor}
}

func detectWUAInstaller(_ CommandExecutor) Installer {
	return &wuaInstaller{
		client: &comWUAClient{logger: slog.Default()},
		logger: slog.Default(),
	}
}
```

- [ ] **Step 4: Verify cross-compile**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./internal/agent/...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/agent/patcher/wua_windows.go internal/agent/patcher/wua_windows_test.go internal/agent/patcher/detect_windows.go
git commit -m "feat(agent): add WUA installer for Windows Update deployment"
```

---

## Task 11: Agent — EXE Installer

**Files:**
- Create: `internal/agent/patcher/exe_windows.go`
- Create: `internal/agent/patcher/exe_windows_test.go`
- Modify: `internal/agent/patcher/detect_windows.go`

- [ ] **Step 1: Write failing test**

Create `internal/agent/patcher/exe_windows_test.go`:

```go
//go:build windows

package patcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEXEInstaller_Name(t *testing.T) {
	inst := &exeInstaller{}
	assert.Equal(t, "exe", inst.Name())
}

func TestEXEInstaller_Install_Success(t *testing.T) {
	inst := &exeInstaller{
		executor: &mockExecutor{
			result: ExecResult{ExitCode: 0, Stdout: []byte("ok")},
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{
		Name:    "/tmp/patchiq-install/chrome-setup.exe",
		Version: "122.0",
	}, false)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.False(t, result.RebootRequired)
}

func TestEXEInstaller_Install_RebootRequired(t *testing.T) {
	inst := &exeInstaller{
		executor: &mockExecutor{
			result: ExecResult{ExitCode: 3010},
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: "/tmp/setup.exe"}, false)
	require.NoError(t, err)
	assert.True(t, result.RebootRequired)
	assert.Equal(t, 3010, result.ExitCode)
}

func TestEXEInstaller_Install_WithSilentArgs(t *testing.T) {
	executor := &recordingExecutor{}
	inst := &exeInstaller{
		executor:   executor,
		silentArgs: "/S /D=C:\\Program Files\\App",
	}

	_, err := inst.Install(context.Background(), PatchTarget{Name: "/tmp/setup.exe"}, false)
	require.NoError(t, err)
	// Verify silent args were passed to executor
	assert.Contains(t, executor.lastArgs, "/S")
	assert.Contains(t, executor.lastArgs, "/D=C:\\Program Files\\App")
}

// recordingExecutor captures the last command execution for assertion.
type recordingExecutor struct {
	lastBinary string
	lastArgs   []string
}

func (r *recordingExecutor) Execute(_ context.Context, binary string, args ...string) (ExecResult, error) {
	r.lastBinary = binary
	r.lastArgs = args
	return ExecResult{ExitCode: 0}, nil
}

// mockExecutor returns a preset result.
type mockExecutor struct {
	result ExecResult
	err    error
}

func (m *mockExecutor) Execute(_ context.Context, _ string, _ ...string) (ExecResult, error) {
	return m.result, m.err
}
```

- [ ] **Step 2: Implement EXE installer**

Create `internal/agent/patcher/exe_windows.go`:

```go
//go:build windows

package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// exeInstaller installs third-party software via .exe silent execution.
type exeInstaller struct {
	executor   CommandExecutor
	logger     *slog.Logger
	silentArgs string // populated from InstallPatchPayload.SilentArgs at dispatch time
}

func (e *exeInstaller) Name() string { return "exe" }

func (e *exeInstaller) Install(ctx context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
	var args []string
	if e.silentArgs != "" {
		args = splitArgs(e.silentArgs)
	}

	execResult, err := e.executor.Execute(ctx, pkg.Name, args...)
	if err != nil {
		return InstallResult{}, fmt.Errorf("exe install %s: %w", pkg.Name, err)
	}

	return InstallResult{
		Stdout:         execResult.Stdout,
		Stderr:         execResult.Stderr,
		ExitCode:       execResult.ExitCode,
		RebootRequired: exeRebootRequired(execResult.ExitCode),
	}, nil
}

// exeRebootRequired returns true for standard Windows installer reboot exit codes.
func exeRebootRequired(exitCode int) bool {
	return exitCode == 3010 || exitCode == 1641
}

// splitArgs splits a silent_args string into individual arguments.
// Handles quoted strings with spaces.
func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false

	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
```

- [ ] **Step 3: Register EXE installer in `detect_windows.go`**

Update `internal/agent/patcher/detect_windows.go` to add the EXE detector:

```go
func init() {
	platformInstallerDetectors = []installerDetectorFunc{
		detectMSIInstaller,
		detectMSIXInstaller,
		detectWUAInstaller,
		detectEXEInstaller,
	}
}

func detectEXEInstaller(executor CommandExecutor) Installer {
	return &exeInstaller{
		executor: executor,
		logger:   slog.Default(),
	}
}
```

- [ ] **Step 4: Wire `silent_args` from payload to EXE installer at dispatch time**

In `internal/agent/patcher/patcher.go`, in `handleInstallPatch` (~line 191-225), after resolving the installer, if it's an EXE installer, set `silentArgs` from the payload:

```go
inst := m.resolveInstaller(pkg.Source)
if inst == nil {
    // ... existing error handling ...
    continue
}

// Pass silent_args to EXE installer
if exeInst, ok := inst.(*exeInstaller); ok && payload.SilentArgs != "" {
    exeInst = &exeInstaller{
        executor:   exeInst.executor,
        logger:     exeInst.logger,
        silentArgs: payload.SilentArgs,
    }
    inst = exeInst
}
```

- [ ] **Step 5: Verify cross-compile**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./internal/agent/...`
Expected: Clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/agent/patcher/exe_windows.go internal/agent/patcher/exe_windows_test.go internal/agent/patcher/detect_windows.go internal/agent/patcher/patcher.go
git commit -m "feat(agent): add EXE silent installer for third-party Windows software"
```

---

## Task 12: Integration Verification

**Files:** No new files — this task verifies the full pipeline.

- [ ] **Step 1: Run all Go tests**

Run: `make test`
Expected: All tests pass with race detector.

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: No new lint errors.

- [ ] **Step 3: Cross-compile Windows agent**

Run: `make build-agent-windows`
Expected: `patchiq-agent-windows-amd64.exe` binary produced.

- [ ] **Step 4: Build all platforms**

Run: `make build`
Expected: All 3 Go binaries (server, hub, agent) build successfully.

- [ ] **Step 5: Run CI-quick**

Run: `make ci-quick`
Expected: All checks pass.

- [ ] **Step 6: Final commit if any fixups needed**

```bash
git add -A
git commit -m "fix: address integration issues from Windows patching pipeline"
```

---

## Dependency Order

```
Task 1 (Proto) → Task 2 (Hub migration + sqlc) → Task 3 (MSRC feed)
                                                 → Task 4 (Hub sync — verify only)
               → Task 5 (Server migration) → Task 6 (Server catalog sync) → Task 7 (Wave dispatcher)
               → Task 8 (Agent WUA collector) → Task 9 (Agent registry collector)
               → Task 10 (Agent WUA installer) → Task 11 (Agent EXE installer)
               → Task 12 (Integration verification)
```

Tasks 3, 4, 8, 9, 10, 11 can be parallelized after their dependencies complete. Tasks 8-11 (agent) can all run in parallel after Task 1 (proto regen).
