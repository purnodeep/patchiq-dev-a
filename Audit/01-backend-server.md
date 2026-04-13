# Backend Server Audit Report

**Scope**: `/home/patchiq/patchiq-dev-a/internal/server/`
**Date**: 2026-04-09
**Files examined**: 320 Go files (non-test: ~120 production files)

---

## 1. Dead/Unused Code

### 1.1 Server Module Interface — Never Implemented
- **File**: `internal/server/module.go` (all lines)
- **Severity**: Minor
- **Detail**: The `Module` interface (with `Init`, `Start`, `Stop`, `RegisterRoutes`, `RegisterGRPCServices`, `MigrationSource`, `EventSubscriptions`) and `ModuleDeps` struct are defined but never implemented by any type in the server package. No code references `server.Module`. The agent has its own separate `Module` interface that IS used. This server interface is dead code.

### 1.2 Empty Placeholder Directories
- **Files**: `internal/server/mcp/.gitkeep`, `internal/server/engine/.gitkeep`, `internal/server/apm/.gitkeep`
- **Severity**: Minor
- **Detail**: Three empty directories with only `.gitkeep` files. No Go code, no implementation. These are placeholders for future features (MCP = Model Context Protocol, APM = Application Performance Monitoring). Not harmful but contribute to the impression of incomplete architecture.

### 1.3 BinaryCache Never Instantiated in Server
- **File**: `internal/server/repo/cache.go` (lines 15-110)
- **Severity**: Minor
- **Detail**: `BinaryCache` struct with `Download()`, `Dir()`, and `NewBinaryCache()` is defined and tested (`cache_test.go`), but never instantiated in `cmd/server/main.go` or anywhere in the server init sequence. The `repo.MountFileServer` is used (for serving files), but the `BinaryCache` download/caching functionality is not wired up. The actual binary caching happens elsewhere or is not yet integrated.

### 1.4 ListEnabledTagRules — Querier Interface Method Never Called
- **File**: `internal/server/api/v1/tag_rules.go` (line 22)
- **Severity**: Minor
- **Detail**: `ListEnabledTagRules` is declared in the `TagRuleQuerier` interface but never called by any `TagRuleHandler` method. Only `ListTagRules` is used in the `List()` handler. The query exists in sqlcgen (generated from SQL), but the handler never invokes it. This method likely exists for the tag evaluator's use (`internal/server/tags/evaluator.go`) and is in the wrong interface.

### 1.5 EndpointCounter Interface — nil in Production
- **File**: `internal/server/api/v1/license.go` (lines 11-13, 46)
- **Severity**: Minor
- **Detail**: `EndpointCounter` interface is defined and used in `LicenseHandler`, but `NewLicenseHandler` is called with `nil` counter in `router.go` (line 346). The `Status()` handler has a nil check, so it works, but endpoint usage is never populated in the license status response.

---

## 2. Incomplete Implementations / TODOs

### 2.1 Hub Sync Delete — Soft-delete Patches Not Implemented
- **File**: `internal/server/api/v1/sync.go` (line 164)
- **Severity**: Important
- **Detail**: `TODO(PIQ-118): soft-delete patches by hub catalog ID when delete-sync is implemented.` — When Hub sends deleted catalog IDs, the server logs them and skips. Deleted patches from the Hub are never removed from the PM database.

### 2.2 CVE Event Adapter — Incomplete Implementation
- **File**: `internal/server/cve/event_adapter.go` (line 13)
- **Severity**: Important
- **Detail**: `TODO(PIQ-107): Task 10 will flesh out this implementation.` — The CVE event adapter has basic structure but the TODO indicates it is not fully implemented.

### 2.3 Notification Recipient — Hardcoded Channel Type
- **File**: `internal/server/notify/resolver.go` (line 79)
- **Severity**: Important
- **Detail**: `TODO(PIQ-244): decode recipient from config when crypto layer supports it` — The notification resolver sets `Recipient` to the channel type string instead of the actual recipient address decoded from encrypted config. This means notification history does not record WHO was notified, only the channel type.

### 2.4 Outbox Pattern for Event Delivery Not Implemented
- **File**: `internal/server/api/v1/helpers.go` (line 57)
- **Severity**: Minor
- **Detail**: `TODO(#177): Consider an outbox pattern to guarantee event delivery for audit compliance.` — Currently events are fire-and-forget; if the event bus fails, the event is lost (logged but not retried). For audit compliance, this is a gap.

### 2.5 Event Emission Failure Counter Not Implemented
- **Files**: `internal/server/deployment/emit.go` (line 18), `internal/server/grpc/sync_outbox.go` (line 240)
- **Severity**: Minor
- **Detail**: `TODO(PIQ-145): add event emission failure counter to surface silent drops.` — Failed event emissions are logged but not counted. There is no OTel/Prometheus metric to alert operators about dropped events.

### 2.6 Workflow Publish TOCTOU Race
- **File**: `internal/server/api/v1/workflows.go` (line 552)
- **Severity**: Minor
- **Detail**: `TODO(#177): Move reads inside the transaction to eliminate TOCTOU race.` — The publish handler reads the draft version and validates it outside the transaction, then publishes inside. A concurrent update could cause a stale publish.

### 2.7 gRPC Enroll — Missing Architecture Field
- **File**: `internal/server/grpc/enroll.go` (lines 125, 267)
- **Severity**: Minor
- **Detail**: `TODO(PIQ-ARCH): add an explicit arch field to EndpointInfo proto` — Architecture is currently inferred from OS info rather than explicitly sent by the agent.

---

## 3. Error Handling Issues

### 3.1 Notification Preferences Update — No Event Emitted
- **File**: `internal/server/api/v1/notifications.go` (lines 583-634)
- **Severity**: Critical (per project rules: every write MUST emit an event)
- **Detail**: `UpdatePreferences()` upserts notification preferences to the database but does NOT emit a domain event. Per CLAUDE.md: "Every write operation MUST emit a domain event." There is no `NotificationPreferencesUpdated` event type defined in topics.go either.

### 3.2 Hub Sync API UpdateConfig — No Event Emitted
- **File**: `internal/server/api/v1/hub_sync.go` (lines 161-204)
- **Severity**: Critical (per project rules)
- **Detail**: `HubSyncAPIHandler.UpdateConfig()` upserts hub sync configuration to the database but does NOT emit a domain event. This write operation has no audit trail.

### 3.3 Hub Sync API Trigger — No Event Emitted
- **File**: `internal/server/api/v1/hub_sync.go` (lines 117-150)
- **Severity**: Important
- **Detail**: `HubSyncAPIHandler.Trigger()` enqueues a River job but does NOT emit a domain event for the manual trigger action. The CatalogSyncStarted event type exists but is not used here.

### 3.4 Inconsistent Error Code Casing in Hub Sync API
- **File**: `internal/server/api/v1/hub_sync.go` (lines 97, 98, 107, 108, etc.)
- **Severity**: Minor
- **Detail**: The `HubSyncAPIHandler` uses lowercase error codes (`"invalid_tenant_id"`, `"not_found"`, `"internal_error"`, `"validation_error"`) while every other handler uses uppercase (`"INVALID_ID"`, `"NOT_FOUND"`, `"INTERNAL_ERROR"`, `"VALIDATION_ERROR"`). This inconsistency affects frontend error handling.

---

## 4. Code Quality

### 4.1 Duplicate `nullableText` Function — Different Signatures
- **Files**: `internal/server/store/inventory.go` (line 106), `internal/server/api/v1/helpers.go` (line 117)
- **Severity**: Minor
- **Detail**: Two functions named `nullableText` in different packages with DIFFERENT signatures:
  - `store.nullableText(s string) pgtype.Text` — converts string to pgtype.Text
  - `v1.nullableText(t pgtype.Text) *string` — converts pgtype.Text to *string pointer
  The names are confusing. They do inverse operations. The `v1` version should be named `textToStringPtr` or similar for clarity.

### 4.2 SQL LIKE Injection — Inconsistent Search Escaping
- **Files**: All handler List methods that accept `search` parameter
- **Severity**: Important
- **Detail**: Only `WorkflowHandler.List()` (`workflows.go:112`) uses `EscapeLikePattern()` to sanitize the search parameter before passing it to SQL queries. All other handlers pass the raw search string directly:
  - `endpoints.go:172` — raw search
  - `groups.go:89` — raw search
  - `policies.go:599` — raw search
  - `cves.go:89` — raw search
  - `patches.go:136` — raw search
  - `roles.go:126` — raw search
  If the SQL queries use LIKE/ILIKE with the search parameter, characters like `%`, `_`, and `\` in user input could cause unexpected matching behavior. Whether this is exploitable depends on the sqlc query implementations (parameterized queries prevent SQL injection, but LIKE wildcards in user input affect result correctness).

### 4.3 `math/rand` Used Instead of `crypto/rand` for Deployment Shuffle
- **File**: `internal/server/api/v1/deployments.go` (line 10, 537)
- **Severity**: Minor
- **Detail**: `math/rand.Shuffle` is used to randomize endpoint assignment to deployment waves. Since Go 1.20, `math/rand` uses a random seed by default, so this is not predictable. However, for security-sensitive operations (deployment targeting), `crypto/rand` would be more appropriate. Low risk since this is just wave assignment ordering.

### 4.4 Workflow Routes Missing RBAC Middleware
- **File**: `internal/server/api/router.go` (lines 325-331)
- **Severity**: Important
- **Detail**: The workflow CRUD routes (List, Create, Get, Update, Delete, Publish, ListVersions) do NOT use the `rp()` RBAC permission middleware, while workflow execution routes (lines 333-338) DO use `rp("workflows", "execute")` and `rp("workflows", "read")`. This means any authenticated user can create, update, delete, and publish workflows regardless of their RBAC permissions. The `ListTemplates` endpoint (line 340) also lacks RBAC.

### 4.5 Large Handler Functions
- **File**: `internal/server/api/v1/deployments.go` — `Create()` method (lines 266-614)
- **Severity**: Minor
- **Detail**: The deployment `Create()` handler is ~350 lines. It handles validation, transaction management, wave creation, target assignment, and job enqueuing all in one function. Consider extracting wave creation and target assignment into helper functions.

### 4.6 `severityFilterFromPolicy` Duplicated Logic
- **File**: `internal/server/api/v1/policies.go` (lines 400-423)
- **Severity**: Minor
- **Detail**: The function comment explicitly states: "This duplicates the logic in deployment.BuildSeverityFilter intentionally." While the comment explains why, maintaining two copies of severity ranking logic is a maintenance risk. The comment warns not to consolidate, which is fine, but the duplication should be tracked.

---

## 5. API Handler Completeness

### 5.1 All CRUD Operations Implemented
- **Severity**: N/A (no issues)
- **Detail**: All registered routes in `router.go` have corresponding handler implementations. No stub bodies were found. Every handler has proper error handling, tenant scoping, and response formatting.

### 5.2 Endpoints — No Create Handler (By Design)
- **File**: `internal/server/api/router.go` (lines 154-167)
- **Severity**: N/A (informational)
- **Detail**: There is no POST /endpoints route. Endpoints are created via gRPC enrollment, not REST API. This is correct by design.

### 5.3 License — Read-Only (No Create/Update/Delete)
- **File**: `internal/server/api/v1/license.go`
- **Severity**: N/A (informational)
- **Detail**: License handler only has a `Status()` method. License management is done via Hub, not the PM server. This is by design.

---

## 6. River Job Issues

### 6.1 All Workers Registered
- **File**: `internal/server/workers/registry.go`
- **Severity**: N/A (no issues)
- **Detail**: The `RegisterWorkers` function registers all 15 job workers. Cross-referencing with the worker definitions confirms all defined workers are registered: DiscoveryWorker, NVDSyncWorker, EndpointMatchWorker, ExecutorWorker, TimeoutWorker, ScanWorker, SendWorker, WaveDispatcherWorker, ScheduleCheckerWorker, AuditRetentionWorker, ComplianceEvalWorker, UserSyncWorker, CatalogSyncWorker, WorkflowExecuteWorker, PolicySchedulerWorker.

### 6.2 No Job Failure Metrics
- **Severity**: Minor
- **Detail**: River workers log errors on failure but do not emit OTel metrics for job failure rates. Combined with TODO(PIQ-145) about missing event emission counters, there is no operational visibility into job health beyond log analysis.

---

## 7. Domain Event Gaps

### 7.1 PolicyAutoDeployed Missing from AllTopics()
- **File**: `internal/server/events/topics.go`
- **Severity**: Critical
- **Detail**: The `PolicyAutoDeployed` constant is defined (line 38) and used by `policy/scheduler.go` and `policy/worker.go` to emit events. However, it is **NOT included in the `AllTopics()` return list** (lines 171-288). The `Emit()` method on the Watermill event bus (publisher.go:53) checks `isRegisteredTopic()` and returns an error if the topic is not in `AllTopics()`. This means **every `PolicyAutoDeployed` event emission silently fails** with the error: `"emit event: topic \"policy.auto_deployed\" is not registered in AllTopics()"`. Auto-deploy policy events are never persisted to the audit table.

### 7.2 Notification Preferences Update — No Event (See 3.1)
- **Severity**: Critical

### 7.3 Hub Sync Config Update — No Event (See 3.2)
- **Severity**: Critical

### 7.4 No Event for TestChannel Results
- **File**: `internal/server/api/v1/notifications.go` — `TestChannel()` handler
- **Severity**: Minor
- **Detail**: The `ChannelTested` event type is defined in topics.go but the TestChannel handler does not emit it. The test result is persisted via `UpdateNotificationChannelTestResult` but there is no audit event for the test action itself. Note: this is a read-like operation (testing connectivity), so the severity is low.

---

## 8. Import Violations

### 8.1 No Forbidden Imports Found
- **Severity**: N/A (clean)
- **Detail**: Searched for `internal/hub/` and `internal/agent/` imports across all server `.go` files. The only match is a comment in `store/db.go:3`: `"IMPORTANT: This file mirrors internal/hub/store/db.go."` — this is a documentation comment, not an import. No actual import violations exist.

---

## Summary by Severity

| Severity | Count | Key Issues |
|----------|-------|------------|
| Critical | 3 | PolicyAutoDeployed missing from AllTopics (silent event loss), NotifPreferences no event, HubSync config no event |
| Important | 5 | Workflow routes missing RBAC, SQL LIKE injection inconsistency, Hub sync delete not implemented, CVE event adapter incomplete, Notification recipient hardcoded |
| Minor | 14 | Dead Module interface, empty dirs, BinaryCache unused, duplicate nullableText, math/rand, large handler functions, various TODOs |

### Top 3 Priorities

1. **Fix AllTopics() to include PolicyAutoDeployed** — This is a silent data loss bug. Every auto-deploy event is dropped because the topic is not registered. One-line fix in `topics.go`.

2. **Add RBAC middleware to workflow CRUD routes** — Any authenticated user can create/delete/modify workflows without permission checks. Security gap.

3. **Add domain events to NotificationPreferences.Update and HubSyncAPI.UpdateConfig** — These write operations violate the project's foundational rule that every write must emit an event.
