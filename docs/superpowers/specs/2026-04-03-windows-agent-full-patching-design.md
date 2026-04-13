# Windows Agent Full Patching Support — E2E Pipeline

**Date:** 2026-04-03
**Author:** Rishab + Claude
**Status:** Approved
**Scope:** Full E2E — Proto + Hub + Server + Agent changes

---

## Problem

The Linux agent handles all software (OS, system, third-party) through unified package managers (APT/YUM). On Windows, the ecosystem is fragmented: OS updates use WUA, system software uses MSI/WUA, and third-party apps use MSI/MSIX/EXE. The current Windows agent has partial support — MSI and MSIX installers work, hotfix detection works, but WUA inventory/patching and third-party software inventory/EXE patching are missing.

Beyond the agent, the full pipeline is broken for Windows:
- Hub's MSRC feed doesn't set `installer_type` on Windows patches
- Hub→Server sync doesn't include `installer_type` or `silent_args`
- Server's `patches` table lacks `installer_type` and `silent_args` columns
- Server's wave dispatcher hardcodes `source: "msi"` for all Windows patches
- Proto lacks `silent_args` field in `InstallPatchPayload`

## Goal

Make Windows endpoints fully functional end-to-end: register on Patch Manager, collect complete inventory (OS updates + system software + third-party apps), and deploy patches using the correct installer mechanism — all flowing through the Hub → Server → Agent pipeline.

---

## Layer 1: Proto Changes

One field addition to `InstallPatchPayload` in `proto/patchiq/v1/common.proto`:

```protobuf
message InstallPatchPayload {
    repeated PatchTarget packages = 1;
    bool dry_run = 2;
    string pre_script = 3;
    string post_script = 4;
    string download_url = 5;
    string checksum_sha256 = 6;
    string silent_args = 7;        // NEW — e.g., "/S", "/quiet /norestart"
}
```

`PatchTarget.source` already exists as a string field — used for installer dispatch with values: `"wua"`, `"msi"`, `"msix"`, `"exe"` for Windows (alongside existing `"apt"`, `"yum"` for Linux).

No new messages, no new enums. The `silent_args` is at payload level, which is correct because the wave dispatcher creates one command per patch per endpoint (single-package payloads).

After proto change: `make proto` to regenerate Go code.

---

## Layer 2: Hub Changes

### 2a. New Migration — `silent_args` Column

```sql
ALTER TABLE patch_catalog ADD COLUMN silent_args TEXT NOT NULL DEFAULT '';
```

The `installer_type` column already exists (migration 005) but is never populated for Windows.

### 2b. MSRC Feed — Populate `installer_type`

Currently `internal/hub/feeds/msrc.go` leaves `InstallerType` empty for all entries. Fix: set `InstallerType = "wua"` for all MSRC entries.

Rationale: MSRC patches are fundamentally Windows Updates — they are always installable via the WUA COM API. The distinction between `"msi"`, `"exe"`, etc. matters for third-party catalog entries (added manually or from future feeds), not MSRC security updates.

### 2c. Hub Sync Endpoint — Include New Fields

Update the sync API response (`/api/v1/sync`) to include `installer_type` and `silent_args` in the JSON payload sent to servers. These fields already exist in the `patch_catalog` table (or will after migration) — they just need to be included in the serialized response.

---

## Layer 3: Server Changes

### 3a. New Migration — Add Columns to `patches` Table

```sql
ALTER TABLE patches ADD COLUMN installer_type TEXT NOT NULL DEFAULT '';
ALTER TABLE patches ADD COLUMN silent_args TEXT NOT NULL DEFAULT '';
```

### 3b. Catalog Sync Worker — Receive and Store New Fields

Update `catalogEntry` struct in `internal/server/workers/catalog_sync.go` to include `InstallerType` and `SilentArgs`. Update `UpsertDiscoveredPatch` sqlc query to write them.

### 3c. Wave Dispatcher — Use Per-Patch Installer Type

Replace hardcoded `osFamilyToSource()` in `internal/server/deployment/wave_dispatcher.go` (~line 282):

**Before:**
```go
Source: osFamilyToSource(patch.OsFamily)  // returns "msi" for all Windows
```

**After:**
```go
Source: installerTypeOrFallback(patch.InstallerType, patch.OsFamily)
```

Where `installerTypeOrFallback` returns `patch.InstallerType` if non-empty, otherwise falls back to `osFamilyToSource()` for backward compatibility with existing patches pre-migration.

Pass `silent_args` into the payload:
```go
installPayload.SilentArgs = patch.SilentArgs
```

### 3d. Update sqlc Queries

- `UpsertDiscoveredPatch` — accept `installer_type` and `silent_args` params
- Regenerate with `make sqlc`

---

## Layer 4: Agent Changes

### New Components

| # | Component | Type | File | Purpose |
|---|-----------|------|------|---------|
| 1 | WUA collector registration | Collector | `inventory/detect_windows.go` | Wire existing `wuaCollector` into platform detection |
| 2 | Registry collector | Collector | `inventory/registry_windows.go` | Scan Uninstall registry keys for all installed software |
| 3 | WUA installer | Installer | `patcher/wua_windows.go` | Download + install Windows Updates via COM API |
| 4 | EXE installer | Installer | `patcher/exe_windows.go` | Silent-install third-party `.exe` packages |

All files use `//go:build windows` tags. No changes to Linux/macOS code. No new dependencies beyond what's already in `go.mod` (`go-ole`, `golang.org/x/sys/windows`).

### Inventory: Three Collectors Working Together

Each collector has a distinct purpose with no overlap:

| Collector | What it reports | Source field |
|-----------|----------------|-------------|
| **Registry** (new) | All installed software — name, version, publisher | `"registry"` |
| **Hotfix** (existing) | Installed KB patches | `"hotfix"` |
| **WUA** (existing code, newly registered) | Available OS updates not yet installed | `"wua"` |

All three run during each inventory collection cycle and their results are sent to the server via the outbox.

### Registry Collector

Scans two registry paths to cover both 64-bit and 32-bit applications:
- `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*`
- `HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*`

For each subkey, reads: `DisplayName`, `DisplayVersion`, `Publisher`, `InstallDate`, `InstallLocation`, `UninstallString`.

Behavior:
- Skips entries with no `DisplayName` (system components, orphaned entries)
- Deduplicates across 64-bit and 32-bit paths (same `DisplayName` + `DisplayVersion` = one entry)
- Maps each entry to `*pb.PackageInfo` with source set to `"registry"`
- Does NOT determine if updates are available — that's the server/hub's job

Uses `golang.org/x/sys/windows/registry` package (already in go.mod via transitive dependency).

### WUA Collector Registration

The `wuaCollector` implementation already exists in `inventory/wua.go`. It uses COM interop via `go-ole` to search for available updates (`IsInstalled=0`). The only missing piece is registration in `detect_windows.go`:

```
platformCollectorDetectors = []collectorDetectorFunc{
    detectHotFixCollector,   // existing
    detectWUACollector,      // new — returns existing wuaCollector
    detectRegistryCollector, // new
}
```

Detection: attempt `ole.CoInitializeEx()` and create `Microsoft.Update.Session`. If either fails (e.g., WUA service disabled, COM unavailable), return nil — collector is simply unavailable. COM is uninitialized in the detector cleanup path.

### WUA Installer

Downloads and installs Windows Updates via COM API, following the same pattern as the existing WUA collector.

COM API sequence:
```
IUpdateSession
  → IUpdateSearcher.Search("UpdateID='xxx' or KBArticleID='KB5034441'")
  → IUpdateDownloader.Download(matchedUpdates)
  → IUpdateInstaller.Install(downloadedUpdates)
  → IInstallationResult (ResultCode, RebootRequired)
```

Server sends `install_patch` command with KB article ID or update title. Installer searches WUA for the matching update, downloads it, then installs it.

Reboot detection:
- `IInstallationResult.RebootRequired` → sets `reboot_required` in `InstallResult`
- Result codes: 2 = succeeded, 3 = succeeded with errors, 4 = failed, 5 = aborted

Dry-run mode: searches + downloads but skips install.

Testability: COM calls wrapped behind an interface (same `updateSearcher` pattern from `wua.go`), so tests can mock the COM layer.

### EXE Installer

Handles third-party software shipped as `.exe` installers (Chrome, Firefox, Adobe, 7-Zip, etc.).

The agent does not know per-app silent install flags. The Hub catalog stores per-package metadata including `silent_args`. Server sends `install_patch` command with:
- `download_url` or local file path to the `.exe`
- `silent_args` — the silent install flags (e.g., `/S`, `/silent`, `/quiet /norestart`)
- `sha256` — expected checksum

Agent flow:
1. Downloads `.exe` to `os.TempDir()\patchiq-install\` with restricted permissions
2. Verifies SHA256 checksum — rejects if mismatch
3. Executes with provided `silent_args`
4. Captures stdout/stderr, exit code
5. Cleans up temp file after install

Reboot detection:
- Exit code 3010 = reboot required (standard Windows convention)
- Exit code 1641 = reboot initiated by installer
- Exit code 0 = success, anything else = failure

Execution timeout: 30 minutes (same as other installers, configurable via settings watcher).

Security:
- SHA256 verification mandatory — if checksum doesn't match, install is rejected
- Temp directory: `os.TempDir()\patchiq-install\` with restricted permissions
- Execution as agent service user (SYSTEM)

### Installer Dispatch

The patcher module dispatches `install_patch` commands to the correct installer based on the `source` field set by the server:

| `source` field | Installer | Handles |
|---|---|---|
| `"wua"` | WUA installer | OS updates, cumulative updates, security patches |
| `"msi"` | MSI installer | `.msi` packages |
| `"msix"` | MSIX installer | `.msix` / `.appx` packages |
| `"exe"` | EXE installer | Third-party `.exe` installers |

No structural changes to the patcher module — it already supports multiple installers via `map[string]Installer`. New installers register via `detect_windows.go`:

```
platformInstallerDetectors = []installerDetectorFunc{
    detectMSIInstaller,    // existing
    detectMSIXInstaller,   // existing
    detectWUAInstaller,    // new
    detectEXEInstaller,    // new
}
```

If a source installer isn't available, the command fails with: `"installer '<source>' not available on this endpoint"`.

---

## E2E Data Flow

### Full Pipeline: Hub → Server → Agent → Server

```
Hub:
  MSRC feed fetches Microsoft security updates
    → installer_type = "wua", severity, KB IDs
    → Upsert to patch_catalog
    → Sync endpoint serves to Server

Server:
  CatalogSyncWorker pulls from Hub
    → Stores patches with installer_type + silent_args
    → Deployment engine creates deployment
    → Wave dispatcher builds InstallPatchPayload:
        source = patch.installer_type (e.g., "wua")
        silent_args = patch.silent_args
        download_url, checksum_sha256
    → Command queued in endpoint's inbox

Agent:
  Inventory (periodic):
    Registry collector → all installed software
    Hotfix collector   → installed KB patches
    WUA collector      → available OS updates
      → Outbox → SyncOutbox → Server

  Patching (on command):
    SyncInbox → Inbox → Command Processor
      → Patcher dispatches by source:
          "wua" → WUA installer (COM API)
          "msi" → MSI installer (msiexec)
          "msix" → MSIX installer (Add-AppxPackage)
          "exe" → EXE installer (silent exec)
      → InstallResult → Outbox → SyncOutbox → Server
```

---

## Files Changed — Complete List

### Proto (1 file)
| File | Change |
|------|--------|
| `proto/patchiq/v1/common.proto` | Add `silent_args` field to `InstallPatchPayload` |

### Hub (4 files + 1 migration)
| File | Change |
|------|--------|
| `internal/hub/store/migrations/NNN_add_silent_args.sql` | Add `silent_args` column to `patch_catalog` |
| `internal/hub/feeds/msrc.go` | Set `InstallerType = "wua"` for MSRC entries |
| `internal/hub/store/queries/patch_catalog.sql` | Include `silent_args` in upsert |
| `internal/hub/api/v1/sync.go` (or equivalent) | Include `installer_type` + `silent_args` in sync response |

### Server (5 files + 1 migration)
| File | Change |
|------|--------|
| `internal/server/store/migrations/NNN_add_installer_metadata.sql` | Add `installer_type` + `silent_args` to `patches` + backfill existing Windows patches |
| `internal/server/workers/catalog_sync.go` | Parse + store `installer_type` + `silent_args` from hub |
| `internal/server/store/queries/patches.sql` | Update `UpsertDiscoveredPatch` with new columns |
| `internal/server/deployment/wave_dispatcher.go` | Use `patch.InstallerType` instead of hardcoded source; pass `silent_args` |

### Agent (3 files modified + 2 new)
| File | Change |
|------|--------|
| `internal/agent/inventory/detect_windows.go` | Add `detectWUACollector` + `detectRegistryCollector` to init |
| `internal/agent/inventory/registry_windows.go` | **New** — registry scanner implementation |
| `internal/agent/patcher/detect_windows.go` | Add `detectWUAInstaller` + `detectEXEInstaller` to init |
| `internal/agent/patcher/wua_windows.go` | **New** — WUA download + install via COM |
| `internal/agent/patcher/exe_windows.go` | **New** — EXE silent installer |

### Generated (regenerate, don't hand-edit)
| File | How |
|------|-----|
| `gen/patchiq/v1/*.go` | `make proto` |
| `internal/server/store/sqlcgen/` | `make sqlc` |
| `internal/hub/store/sqlcgen/` | `make sqlc` |

### Files NOT Changed
- No Linux/macOS agent code
- No frontend changes
- No gRPC handler structural changes (payload is opaque bytes)

---

## Edge Cases and Backward Compatibility

### Pre-migration patches in server DB
Existing Windows patches have `installer_type = ""`. The server migration includes a backfill step that classifies existing Windows patches using heuristics:
- Name contains "KB" or "Cumulative Update" or "Security Update" → `installer_type = "wua"`
- `package_url` ends with `.msi` → `installer_type = "msi"`
- `package_url` ends with `.msix` or `.appx` → `installer_type = "msix"`
- `package_url` ends with `.exe` → `installer_type = "exe"`
- All other Windows patches → `installer_type = "wua"` (safe default — most Windows patches are WUA-deliverable)

This prevents stale patches from deploying with the wrong installer. After the next Hub sync, `installer_type` values are overwritten with authoritative data from the catalog.

### WUA service disabled on endpoint
WUA collector detector probes COM initialization. If WUA service is disabled, detector returns nil, collector is skipped. Hotfix + Registry collectors still work. Agent logs a warning.

### EXE without silent_args
If Hub catalog entry has empty `silent_args`, agent executes EXE with no flags. Most installers will show GUI. Server should only dispatch EXE installs when `silent_args` is populated — this is a catalog data quality concern, not an agent concern.

### COM thread safety
WUA collector and WUA installer both use COM. COM is initialized per-goroutine via `ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)`. Each collector/installer initializes and uninitializes COM in its own execution scope — no shared COM state.

---

## Testing Strategy

All new code uses interface abstractions for external dependencies (COM, registry, command execution). Tests mock these interfaces.

| Layer | Component | Test approach |
|-------|-----------|--------------|
| Proto | silent_args field | Verify marshal/unmarshal roundtrip |
| Hub | MSRC feed installer_type | Verify MSRC entries get `installer_type = "wua"` |
| Hub | Sync endpoint | Verify response includes `installer_type` + `silent_args` |
| Server | Catalog sync | Verify new fields stored in patches table |
| Server | Wave dispatcher | Verify `source` uses `installer_type`; verify fallback for empty |
| Agent | Registry collector | Mock registry reader, verify parsing + deduplication |
| Agent | WUA collector registration | Verify detector returns collector when COM succeeds, nil when fails |
| Agent | WUA installer | Mock COM interfaces, verify search → download → install sequence |
| Agent | EXE installer | Mock command executor, verify checksum validation + arg passing + cleanup |
| Agent | Dispatch | Verify `source` field routes to correct installer |

Integration testing requires a Windows environment — covered by manual QA on Windows test endpoint.

---

## Dependency Order

Changes must be implemented in this order (each layer depends on the previous):

1. **Proto** — add `silent_args` field, regenerate
2. **Hub** — migration + MSRC feed fix + sync endpoint update
3. **Server** — migration + catalog sync + wave dispatcher fix
4. **Agent** — collectors + installers (can parallelize all 4 agent changes)
