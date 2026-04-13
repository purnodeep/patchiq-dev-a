# PatchIQ Agent Backend Audit

**Scope**: `internal/agent/`, `cmd/agent/`
**Date**: 2026-04-09
**Auditor**: Claude Opus 4.6

---

## Executive Summary

The agent backend is well-architected with a clean module registry pattern, proper
gRPC lifecycle management (enrollment, heartbeat, outbox sync, inbox fetch), and
real implementations for all target platforms (Linux, macOS, Windows). There are no
critical blocking issues. The main findings are: one functional gap in the HTTP scan
endpoint (does not actually trigger a scan), several dead/unused code items, a few
incomplete implementations on Windows, and minor hardening opportunities.

---

## 1. Dead/Unused Code

### 1.1 `parseRPMOutput` function — only used in tests (Minor)
- **File**: `internal/agent/inventory/yum.go:55`
- **Detail**: `parseRPMOutput` (4-field parser) is only called from `yum_test.go`.
  Production code calls `parseRPMOutputExtended` (6-field parser) via `rpmCollector.Collect()`.
  The old function is dead code in production.
- **Fix**: Remove `parseRPMOutput` and update tests to use `parseRPMOutputExtended`.

### 1.2 `detectInstaller` (singular) wrapper — only used in tests (Minor)
- **File**: `internal/agent/patcher/installer.go:83`
- **Detail**: `detectInstaller` is a convenience wrapper around `detectInstallers` that
  returns only the first installer. Production code (`patcher.go:84`) calls `detectInstallers`
  (plural) directly. `detectInstaller` is only used in `installer_test.go`.
- **Fix**: Inline or remove; tests can call `detectInstallers` and check the first element.

### 1.3 `local_inventory` table — never written to (Minor)
- **File**: `internal/agent/comms/schema.sql:29`
- **Detail**: The `local_inventory` table is created in the schema but never inserted
  into or queried by any production code. Only referenced in `comms/db_test.go` to
  verify schema creation. Inventory data is serialized to protobuf and sent via outbox,
  not stored locally.
- **Fix**: Remove the table from the schema or implement local inventory caching if needed.

### 1.4 `NegotiateProtocolVersion` var — never called in production (Minor)
- **File**: `internal/agent/comms/enroll.go:114`
- **Detail**: `NegotiateProtocolVersion` is assigned from the shared protocol package but
  is never called in agent code. Only used in `enroll_test.go`. The server handles
  protocol negotiation; the agent just stores the result.
- **Fix**: Remove if not needed; the test can import the shared package directly.

### 1.5 `Downloader` type — never used in production agent code (Minor)
- **File**: `internal/agent/patcher/download.go:15`
- **Detail**: `Downloader` and `NewDownloader` are only used in `download_test.go`. The
  agent does not download patch binaries itself (the server pushes commands; installers
  use OS package managers). This was likely scaffolded for future EXE/MSI download support.
- **Fix**: Keep if planned for M3+ binary download; otherwise remove.

### 1.6 `SaveServerCert` — never called in production (Minor)
- **File**: `internal/agent/comms/certgen.go:119`
- **Detail**: Forward-compatibility function for M2 mTLS. Documented as such. Only tested.
- **Fix**: Acceptable as forward-compat stub. No action needed.

---

## 2. Incomplete Implementations

### 2.1 HTTP `POST /api/v1/scan` does not trigger an actual scan (Important)
- **File**: `internal/agent/api/scan.go:26-36`
- **Detail**: The scan endpoint only writes a log entry and returns `{"status": "scan_triggered"}`.
  It does NOT actually trigger the `CollectionRunner` to perform an inventory scan.
  The `CollectionRunner` has no `Trigger()` method, and the `ScanHandler` has no reference
  to it. The frontend agent UI and CLI both call this endpoint expecting a scan to happen.
- **Fix**: Add a `Trigger()` method to `CollectionRunner` (similar to `CommandProcessor.Trigger()`),
  wire it into `ScanHandler` via `HandlerDeps`, and call it from the handler.

### 2.2 Windows `IsRoot()` always returns false (Important)
- **File**: `internal/agent/privilege_windows.go:7`
- **Detail**: `IsRoot()` on Windows is hardcoded to return `false`. It is called by
  `inventory/metrics_darwin.go:257` (only on macOS), so it does not cause a runtime
  issue today. However, any future Windows code that checks privilege will get wrong results.
  The TODO references `PIQ-0` which is not a valid issue number.
- **Fix**: Implement using `golang.org/x/sys/windows` token check, or at minimum
  update the TODO to reference a real issue.

### 2.3 MSI installer ignores `dryRun` parameter (Minor)
- **File**: `internal/agent/patcher/msi.go:15`
- **Detail**: The `Install` method signature accepts `dryRun bool` but the parameter
  name is `_`, meaning it is always ignored. A dry-run install of an MSI will actually
  install it.
- **Fix**: Check `dryRun` and either skip execution or log a simulated result.

### 2.4 MSIX installer ignores `dryRun` parameter (Minor)
- **File**: `internal/agent/patcher/msix.go:15`
- **Detail**: Same issue as MSI. The `dryRun` parameter is discarded (`_ bool`).
- **Fix**: Same as MSI.

### 2.5 EXE installer ignores `dryRun` parameter (Minor)
- **File**: `internal/agent/patcher/exe_windows.go:24`
- **Detail**: Same issue. The `dryRun` parameter is discarded.
- **Fix**: Same as MSI/MSIX.

### 2.6 macOS service not wired into CLI `service` subcommand (Minor)
- **File**: `cmd/agent/cli/service_stub.go:11`
- **Detail**: `DarwinService` is fully implemented in `internal/agent/service_darwin.go`
  (launchd plist generation, install, start, stop, uninstall, status). But the CLI
  `service` subcommand on darwin falls through to the stub (`service_stub.go`) which
  prints "service management is not available on this platform". The build tag is
  `!linux && !windows`, which includes darwin.
- **Fix**: Create `cmd/agent/cli/service_darwin.go` with build tag `darwin` that wires
  `DarwinService` into the CLI, similar to `service_linux.go`.

### 2.7 Deferred reboot mode is a no-op (Minor)
- **File**: `internal/agent/system/reboot_linux.go:27`, `reboot_darwin.go:23`, `reboot_windows.go:23`
- **Detail**: All three platforms handle `REBOOT_MODE_DEFERRED` by returning `nil` immediately.
  The comment says "store request for maintenance window execution" but no storage happens.
  The settings watcher does not check for deferred reboots.
- **Fix**: Either persist the deferred reboot request to `agent_state` and add a watcher
  check during the maintenance window, or document this as not-yet-implemented.

---

## 3. Error Handling Issues

### 3.1 Heartbeat outbox pending count reads all items into memory (Minor)
- **File**: `internal/agent/comms/heartbeat.go:160-166`
- **Detail**: To get the outbox queue depth, `outbox.Pending(ctx, 1000)` fetches up to
  1000 rows into memory just to count them. With large outbox backlogs this wastes memory.
- **Fix**: Add a `Count(ctx)` method to `Outbox` using `SELECT COUNT(*) FROM outbox WHERE status = 'pending'`.

### 3.2 Status provider swallows query errors for patch counts (Minor)
- **File**: `internal/agent/store/status.go:79-81`
- **Detail**: Three `QueryRow` calls discard errors with `_`. If the table is missing or
  corrupted, the counts silently default to 0 with no logging.
- **Fix**: Log errors at warn level (matching the pattern used elsewhere in the agent).

### 3.3 `slog.InfoContext`/`slog.ErrorContext` used in `download.go` without context (Minor)
- **File**: `internal/agent/patcher/download.go:30,84,89`
- **Detail**: Uses the bare `slog` package-level functions instead of a logger from deps.
  This bypasses any configured log handler or trace context.
- **Fix**: Accept a `*slog.Logger` in `Downloader` and use it consistently.

---

## 4. Cross-Platform Completeness

### 4.1 Inventory Collectors

| Collector | Linux | macOS | Windows | Status |
|-----------|-------|-------|---------|--------|
| APT (dpkg) | Real | N/A | N/A | Complete |
| RPM (yum/dnf) | Real | N/A | N/A | Complete |
| Homebrew | Real | Real | Stub (no-op) | Complete |
| Snap | Real (linux) | Stub | Stub | Complete |
| macOS softwareupdate | N/A | Real | N/A | Complete |
| WUA (Windows Update) | N/A | N/A | Real (COM) | Complete |
| Hotfix | N/A | N/A | Real (PowerShell) | Complete |
| Registry | N/A | N/A | Real (Win registry) | Complete |
| Hardware | Real | Real | Real | Complete |
| Metrics (CPU/Mem/Disk) | Real | Real | Real | Complete |
| Services | Real (systemd) | Real (launchctl) | Real (WMI) | Complete |

**Verdict**: All inventory collectors have real implementations on their target platforms.
Auto-detection via `detectCollectors` correctly probes for available tools.

### 4.2 Patcher Installers

| Installer | Linux | macOS | Windows | Status |
|-----------|-------|-------|---------|--------|
| APT | Real | N/A | N/A | Complete, with version fallback |
| YUM/DNF | Real | N/A | N/A | Complete |
| Homebrew | Real | Real | Stub | Complete |
| macOS softwareupdate | N/A | Real | N/A | Complete, with false-positive detection |
| MSI | N/A | N/A | Real | Missing dry-run (see 2.3) |
| MSIX | N/A | N/A | Real | Missing dry-run (see 2.4) |
| WUA | N/A | N/A | Real (COM) | Complete |
| EXE | N/A | N/A | Real | Missing dry-run (see 2.5) |

**Verdict**: All patcher installers have real implementations. Windows installers
(MSI, MSIX, EXE) lack dry-run support.

### 4.3 System Commands

| Command | Linux | macOS | Windows | Status |
|---------|-------|-------|---------|--------|
| Reboot (immediate) | Real | Real | Real | Complete |
| Reboot (graceful) | Real | Real | Real | Complete |
| Reboot (deferred) | No-op | No-op | No-op | Stub (see 2.7) |
| update_config | Real | Real | Real | Complete |

### 4.4 Service Management

| Platform | Implementation | CLI wired? |
|----------|---------------|------------|
| Linux (systemd) | Real | Yes |
| macOS (launchd) | Real | **No** (see 2.6) |
| Windows (SCM) | Real | Yes |

---

## 5. gRPC Client Assessment

### 5.1 Enrollment — Fully implemented
- **File**: `internal/agent/comms/enroll.go`
- Builds `EnrollRequest` with validation, calls gRPC `Enroll`, persists `agent_id`
  and `negotiated_protocol_version` to `agent_state`. Idempotent (skips if already enrolled).
  Retry with backoff in `Client.Run`.

### 5.2 Heartbeat — Fully implemented
- **File**: `internal/agent/comms/heartbeat.go`
- Bidirectional streaming. Sender transmits resource usage, uptime, queue depth at
  configurable interval. Receiver handles directives: re-enroll, shutdown, update required,
  protocol unsupported. Commands-pending notification triggers inbox fetch and command
  processing. Dynamic interval via settings watcher.

### 5.3 Outbox Sync (SyncOutbox) — Fully implemented
- **File**: `internal/agent/comms/sync.go`
- `SyncRunner` drains outbox over bidi stream with bandwidth throttling, dead-letter
  handling (max attempts), transient/permanent rejection handling. Stream-per-sync-cycle
  when using opener. Trigger mechanism for immediate sync.

### 5.4 Inbox Fetch (SyncInbox) — Fully implemented
- **File**: `internal/agent/comms/inbox_sync.go`
- `FetchInbox` opens server-streaming `SyncInbox`, reads all commands to EOF, stores
  idempotently in local inbox. Triggered by heartbeat's `OnCommandsPending` callback.

### 5.5 Connection Lifecycle — Fully implemented
- **File**: `internal/agent/comms/client.go:241`
- `Client.Run`: cert gen -> connect with retry -> enroll with retry -> heartbeat loop
  with reconnection. Re-enrollment on server request. Sync runner runs alongside heartbeat
  and stops when heartbeat fails.

**Verdict**: The gRPC client is production-ready. No stubs or missing implementations.

---

## 6. Local HTTP API Assessment

| Endpoint | Method | Handler | Status |
|----------|--------|---------|--------|
| `/health` | GET | Inline | Working |
| `/api/v1/status` | GET | `StatusHandler` | Working |
| `/api/v1/patches/pending` | GET | `PatchesHandler` | Working (cursor pagination) |
| `/api/v1/history` | GET | `HistoryHandler` | Working |
| `/api/v1/logs` | GET | `LogsHandler` | Working |
| `/api/v1/settings` | GET | `SettingsHandler` | Working |
| `/api/v1/settings` | PUT | `SettingsUpdateHandler` | Working |
| `/api/v1/hardware` | GET | `HardwareHandler` | Working |
| `/api/v1/software` | GET | `SoftwareHandler` | Working |
| `/api/v1/services` | GET | `ServicesHandler` | Working |
| `/api/v1/metrics` | GET | `MetricsHandler` | Working |
| `/api/v1/scan` | POST | `ScanHandler` | **Broken** (see 2.1 -- logs only, does not trigger scan) |

**Verdict**: 11 of 12 endpoints are functional. The scan trigger endpoint is a no-op.

---

## 7. SQLite Store Assessment

### 7.1 Schema (comms/schema.sql)
Tables: `outbox`, `inbox`, `local_inventory`, `agent_state`
- All have `CREATE TABLE IF NOT EXISTS` (idempotent).
- Proper indexes on status/timestamp columns.
- `local_inventory` is unused (see 1.3).

### 7.2 Schema (store/schema.sql)
Tables: `pending_patches`, `patch_history`, `agent_logs`, `rollback_records`
- All idempotent.
- Proper indexes for query patterns.
- `rollback_records` duplicated in both schema.sql and `ApplyMigrations` (harmless, both
  use `IF NOT EXISTS`).

### 7.3 Migrations (store/db.go:26-74)
- `ApplyMigrations` adds columns to `pending_patches` and `patch_history` via
  `ALTER TABLE ADD COLUMN` with duplicate-name error suppression (correct for SQLite).
- Creates `rollback_records` table.
- Idempotent and safe to run multiple times.

### 7.4 WAL mode and connection settings
- **File**: `internal/agent/comms/db.go:23`
- WAL mode enabled, busy_timeout=5000ms, MaxOpenConns=1.
- Correct for single-writer SQLite.

### 7.5 Retention
- **File**: `internal/agent/store/retention.go`
- Hourly retention job deletes logs, sent outbox items, and history older than
  configurable days. Clean implementation.

**Verdict**: Schema is complete and well-maintained. Migration approach is appropriate
for embedded SQLite (no goose, just idempotent ALTER TABLE).

---

## 8. CLI Subcommands Assessment

### 8.1 `install` — Fully implemented
- **File**: `cmd/agent/cli/install.go`
- Interactive TUI (Bubble Tea) and headless (`--non-interactive`) modes.
- Validates server connectivity, performs enrollment, writes config YAML.
- Token can be passed via env var to avoid process list exposure.

### 8.2 `status` — Fully implemented
- **File**: `cmd/agent/cli/status.go`
- Reads agent_state from SQLite. Supports `--json` and `--watch` (live TUI) modes.
- Shows agent ID, connection state, last heartbeat, last scan, queue depth.

### 8.3 `scan` — Fully implemented
- **File**: `cmd/agent/cli/scan.go`
- Interactive TUI with spinner and results table.
- `--dry-run` mode skips outbox queuing.
- Initializes inventory module, runs collection, displays results, queues to outbox.

### 8.4 `service` — Fully implemented (Linux, Windows); missing macOS wiring
- **Linux**: `cmd/agent/cli/service_linux.go` — install/uninstall/start/stop/status via systemd.
- **Windows**: `cmd/agent/cli/service_windows.go` — install/uninstall/start/stop/status via SCM.
- **macOS**: Falls to stub (see 2.6).

**Verdict**: All CLI subcommands are implemented and functional. macOS `service`
subcommand needs wiring.

---

## 9. Architecture Observations (Informational)

### 9.1 Insecure gRPC transport
- **File**: `internal/agent/comms/client.go:84`
- Connection uses `insecure.NewCredentials()`. Comment in `install.go:182` references
  `TODO(PIQ-116)` for mTLS. Cert generation exists but is not used for transport.
- **Risk**: Acceptable for M1/M2; must be addressed before production deployment.

### 9.2 Settings watcher design is solid
- Centralized `SettingsWatcher` reads SQLite every 30s and caches in memory.
- Consumers (heartbeat, scan, bandwidth, concurrency, log level) use getter functions
  that read from the cache under RLock.
- Runtime log level changes propagate via `slog.LevelVar`.

### 9.3 Module registry pattern is clean
- `Module` interface is well-defined with lifecycle methods.
- `Registry` handles init, start, stop (with rollback on start failure).
- Command dispatch via `SupportedCommands()` mapping is extensible.

### 9.4 Dynamic semaphore for concurrent installs
- `patcher/dynsem.go` uses `sync.Cond` to allow runtime limit changes.
- Correctly re-evaluates `maxFunc()` on every acquire.

---

## Summary by Severity

| Severity | Count | Key Items |
|----------|-------|-----------|
| Critical | 0 | -- |
| Important | 2 | Scan endpoint no-op (2.1), Windows IsRoot stub (2.2) |
| Minor | 14 | Dead code (1.1-1.5), dry-run gaps (2.3-2.5), macOS service CLI (2.6), deferred reboot no-op (2.7), error handling (3.1-3.3) |
