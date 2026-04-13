# PatchIQ Comprehensive Platform Audit — Client Deployment Readiness

**Date**: 2026-04-09
**Branch**: dev-a
**Scope**: Full platform audit across 8 parallel agents — database, features, scalability, production readiness, frontend, testing, cross-platform integration, delta from prior audit
**Goal**: Identify every issue that could surface during client's 1-month stress test at 1000+ endpoints

---

## Executive Summary

The platform is **architecturally sound but has critical integration gaps between the three tiers**. Individual components (server API, agent enrollment, hub feeds) work in isolation, but the end-to-end data flows that connect them have broken links, silent data drops, and missing implementation. A client stress-testing at 1000 endpoints will hit scalability walls within the first week.

**Verdict: NOT READY for client deployment without the Critical and High items below.**

### Issue Counts by Severity

| Severity | Count | Category Breakdown |
|----------|-------|--------------------|
| **CRITICAL** | 14 | 4 security, 4 scalability, 3 integration, 2 data integrity, 1 feature |
| **HIGH** | 21 | 6 scalability, 5 integration, 4 production-readiness, 3 database, 2 testing, 1 frontend |
| **MEDIUM** | 25 | Mixed across all categories |
| **LOW** | 12 | Polish, naming, minor UX |

---

## PART 1: CRITICAL ISSUES (Fix Before Deployment)

### C1. JWT Claim Injection — Hub Auth [SECURITY] [STILL OPEN FROM PRIOR AUDIT]
**File**: `internal/hub/auth/session.go:39`
**Issue**: `mintJWT` uses `fmt.Sprintf` with user-controlled email/name values to construct JWT claims JSON. Attacker can inject arbitrary claims via crafted email field.
**Impact**: Authentication bypass, privilege escalation.
**Fix**: Use `json.Marshal` with a proper claims struct instead of string interpolation.

### C2. Workflow CRUD Routes Missing RBAC [SECURITY] [STILL OPEN]
**File**: `internal/server/api/router.go:330-345`
**Issue**: All workflow routes (List, Create, Get, Update, Delete) have NO `.With(rp(...))` RBAC middleware. Any authenticated user can read/modify/delete all workflows.
**Impact**: Authorization bypass — intern-level users can modify production workflows.
**Fix**: Add RBAC middleware matching other resource routes.

### C3. PolicyAutoDeployed Event Not In AllTopics [DATA INTEGRITY] [STILL OPEN]
**File**: `internal/server/events/topics.go:38`
**Issue**: `PolicyAutoDeployed` topic is defined but missing from `AllTopics()` function (lines 171-288). `Emit()` calls silently fail.
**Impact**: Auto-deploy policies create no audit trail. Violates core domain event invariant.
**Fix**: Add to `AllTopics()`.

### C4. Agent CVE Detection Silently Dropped [INTEGRATION]
**File**: `internal/server/grpc/sync_outbox.go:141-254`
**Issue**: Agent sends `InventoryReport.detected_cves` via proto, but server's `processInventory` only processes `installed_packages`. CVE detection data is transmitted but completely discarded.
**Impact**: Agent-side vulnerability detection is a dead feature. Server re-does CVE matching independently.
**Fix**: Either process `detected_cves` in `processInventory` or remove the field from proto to avoid confusion.

### C5. No Hub→Server CVE Sync Protocol [INTEGRATION]
**File**: Proto defines no mechanism; `internal/server/cve/` independently fetches NVD
**Issue**: Hub curates CVEs from 6 feeds (NVD, CISA KEV, MSRC, RedHat, Ubuntu, Apple). Server ignores all of this and independently fetches only NVD. Hub's enriched CVE data never reaches the server.
**Impact**: Server has incomplete CVE coverage. Hub's multi-feed curation is wasted effort.
**Fix**: Extend catalog sync to include CVE data, or create dedicated CVE sync flow.

### C6. License Validation is a No-Op [INTEGRATION]
**File**: `internal/server/api/v1/license.go`, `proto/patchiq/v1/hub.proto`
**Issue**: Proto defines `ValidateLicense` RPC but server has no gRPC client to call it. License expiry is never checked. Endpoint count limits are never enforced.
**Impact**: Client can exceed licensed endpoint count. Expired licenses continue working.
**Fix**: Implement license validation client, check on enrollment + periodic background check.

### C7. DB Connection Pool Sized for Dev, Not Production [SCALABILITY]
**File**: `configs/server.yaml:25-26`
**Issue**: `max_conns: 25, min_conns: 5`. With 1000 agents each sending heartbeats requiring DB transactions, pool exhaustion is guaranteed.
**Impact**: Connection queue buildup → request timeouts → cascading failures.
**Fix**: Increase to `max_conns: 200, min_conns: 50` for 1000-endpoint deployments.

### C8. ListEndpointsByTenant Returns All Rows — No LIMIT [SCALABILITY]
**File**: `internal/server/store/queries/endpoints.sql:55-56`
**Issue**: `SELECT * FROM endpoints WHERE tenant_id = $1 ORDER BY created_at` — no LIMIT clause. Used in QuickDeploy which loads ALL endpoints into memory then filters in Go.
**Impact**: With 1000 endpoints: every QuickDeploy loads 1000 rows into memory. OOM risk at scale.
**Fix**: Add LIMIT/OFFSET or cursor pagination. Move filtering to SQL WHERE clause.

### C9. ListPatchesFiltered Has 5 Correlated Subqueries Per Row [SCALABILITY]
**File**: `internal/server/store/queries/patches.sql:93-123`
**Issue**: Each patch row executes 5 scalar subqueries (cve_count, highest_cvss, remediation_pct, endpoints_deployed, affected_endpoints). For a page of 50 patches = 250 subqueries.
**Impact**: Query time grows quadratically with data volume. Dashboard becomes unusable at scale.
**Fix**: Refactor to use CTEs or window functions with pre-aggregated stats.

### C10. gRPC Server Has No MaxConcurrentStreams [SCALABILITY]
**File**: `internal/server/grpc/server.go:24-38`
**Issue**: No `MaxConcurrentStreams` configured. With 1000 agents each maintaining heartbeat + sync streams, the gRPC server accepts unlimited connections until resources are exhausted.
**Impact**: Memory exhaustion, DB pool starvation.
**Fix**: Set `MaxConcurrentStreams` proportional to DB pool size.

### C11. Duplicate UserID Context Keys [DATA INTEGRITY] [STILL OPEN]
**File**: `internal/shared/user/context.go` vs `internal/shared/otel/context.go`
**Issue**: Two different struct types used as context keys for user ID. OTel tracing never sees the user ID set by the user middleware.
**Impact**: User IDs missing from ALL structured logs and traces. Debugging production issues impossible.
**Fix**: Use single shared context key.

### C12. No Endpoint Stale/Offline Detection [FEATURE]
**File**: `internal/server/grpc/heartbeat.go`
**Issue**: Endpoints are marked "online" on heartbeat but there is NO background job to mark them "offline" after missed heartbeats. Once online, always online in the dashboard.
**Impact**: Dashboard shows stale endpoint status. Client will immediately notice "online" endpoints that are actually down.
**Fix**: Add periodic job that marks endpoints offline if last_heartbeat > threshold.

### C13. 2 Write Operations Emit No Domain Events [DATA INTEGRITY] [STILL OPEN]
**Files**: `internal/server/api/v1/notifications.go:584-634`, `internal/server/api/v1/hub_sync.go:161-204`
**Issue**: `UpdatePreferences` and `UpdateConfig` (hub sync) modify data but never emit domain events. Violates the "every write emits an event" invariant.
**Impact**: Audit trail gaps. Event-driven subscribers never notified.
**Fix**: Add event emission to both handlers.

### C14. No 2027 Audit Partitions [SCALABILITY] [STILL OPEN]
**File**: `internal/server/store/migrations/001_init_schema.sql`
**Issue**: Audit event partitions only defined through 2026-12. After that, all events fall into default partition.
**Impact**: Partition pruning breaks, audit queries degrade, retention jobs may fail.
**Fix**: Add migration creating 2027 monthly partitions.

---

## PART 2: HIGH ISSUES (Fix Before Scale Testing)

### H1. Missing Database Indexes on Hot-Path Tables [DATABASE]
Multiple migration files. Missing indexes:
- `deployment_targets(tenant_id, endpoint_id)` — needed for per-endpoint deployment queries
- `endpoint_cves(tenant_id, cve_id)` — needed for "which endpoints affected by CVE X"
- `cves(tenant_id, severity)` — dashboard filters by severity
- `patches(tenant_id, severity, os_family)` — ListPatchesFiltered WHERE clauses
- `workflow_executions(tenant_id, workflow_id, status)` — workflow listing
- `approval_requests(tenant_id, status, timeout_at)` — pending approval queries

### H2. 20+ Unbounded Queries Without LIMIT [SCALABILITY]
Files: Multiple `.sql` files in `internal/server/store/queries/`
Affected: `ListPatchesByTenant`, `ListDeploymentsByTenant`, `ListAlertRules`, `ListRegistrationsByTenant`, `ListConfigOverridesByScope`, `ListCustomFrameworks`, and ~15 more.
**Fix**: Add LIMIT clauses or convert to cursor-based pagination.

### H3. River Job Queue — Single Queue, Low Workers [SCALABILITY]
**File**: `configs/server.yaml:31-32`, `cmd/server/main.go:265-280`
**Issue**: All jobs share `queue.default` with 100 max workers. High-volume notification events can starve deployment execution jobs. With 1000 endpoints × 5 jobs per deployment = 5000 queued jobs.
**Fix**: Split into priority queues (deployments: 150, discovery: 50, notifications: 50, default: 100).

### H4. Deployment Target Creation — No Batch INSERT [SCALABILITY]
**File**: `internal/server/api/v1/patches.go:729-743`
**Issue**: QuickDeploy creates deployment targets in a loop — 1 INSERT per endpoint. With 1000 endpoints = 1000 individual INSERTs.
**Fix**: Use multi-row INSERT.

### H5. Hub Connection UI Missing — Backend Exists, Frontend Doesn't [FEATURE]
**File**: `web/src/pages/settings/PatchSourcesSettingsPage.tsx:203-258`
**Issue**: "Hub Not Connected" state tells users to manually make a PUT request. `useUpdateSyncConfig()` hook exists in `web/src/api/hooks/useHubSync.ts:53-67` but is **never imported anywhere**.
**Impact**: Users cannot configure hub connection through the UI.
**Fix**: Build form dialog that uses the existing hook.

### H6. Agent Server URL Not Reconfigurable [FEATURE]
**File**: `web-agent/src/pages/settings/SettingsPage.tsx:850`
**Issue**: Server URL is displayed read-only. Agent API's `SettingsUpdateRequest` struct doesn't include `server_url`. Once enrolled with wrong URL, agent cannot be reconfigured via UI.
**Fix**: Add `server_url` to settings update API and frontend form.

### H7. Rate Limiting Only on Auth Endpoints [SECURITY]
**File**: `internal/server/api/router.go:77-80`
**Issue**: Rate limiting (10 req/min) only applied to login routes. No rate limiting on data endpoints.
**Impact**: Resource exhaustion attacks, data scraping without throttling.
**Fix**: Implement global API rate limiting with per-endpoint policies.

### H8. Health Checks Only Verify Database [PRODUCTION]
**Files**: `internal/server/api/v1/health.go:43-62`, `internal/hub/api/v1/health.go:40-70`
**Issue**: `/ready` endpoint only checks DB connectivity. Missing: gRPC, Valkey/cache, event bus, River queue.
**Impact**: Service reports "ready" while dependent systems are down.
**Fix**: Extend health checks to cover all critical dependencies.

### H9. Idempotency Store Falls Back to In-Memory [PRODUCTION]
**Files**: `cmd/server/main.go:545-567`, `cmd/hub/main.go:300-302`
**Issue**: When Valkey is unavailable, silently falls back to in-memory store. Idempotency guarantees lost across restarts.
**Impact**: Duplicate deployment executions possible after server restart.
**Fix**: Fail fast if Valkey unavailable in production mode, or log explicit warning.

### H10. Notification Encryption Key Generated Ephemerally [PRODUCTION]
**File**: `cmd/server/main.go:276-287`
**Issue**: If `PATCHIQ_NOTIFICATION_KEY` not configured, generates random key. Key changes on every restart, breaking encrypted notification credentials.
**Fix**: Require key in production configuration.

### H11. Enrollment Token Never Expires [INTEGRATION]
**File**: `internal/server/grpc/enroll.go`
**Issue**: `agent_registrations` table has no `expires_at` column. Proto defines `TOKEN_EXPIRED` error code but server never checks token age. Tokens valid forever.
**Fix**: Add expiry column and validation.

### H12. mTLS Certificate Never Generated [INTEGRATION]
**File**: Proto `EnrollResponse.mtls_certificate` (line 92)
**Issue**: Proto field defined but server always sends empty bytes. No certificate generation code exists.
**Fix**: Either implement mTLS or remove from proto.

### H13. Command Timeout Not Enforced [INTEGRATION]
**File**: `internal/server/deployment/wave_dispatcher.go:56`
**Issue**: `commandTimeout` parameter exists but never used to actually timeout commands. Agent reboot mid-deployment hangs forever.
**Fix**: Implement deadline checking for long-running commands.

### H14. No Cross-Platform Audit Correlation [INTEGRATION]
**Issue**: Hub and Server each have separate `audit_events` tables. No trace ID or correlation ID connects events across platforms.
**Impact**: Cannot reconstruct full operation timeline for a deployment (Hub patch → Server deploy → Agent execute).
**Fix**: Add correlation_id field to all cross-platform messages.

### H15. Config Propagation to Agents Not Implemented [INTEGRATION]
**File**: Proto `HeartbeatResponse.config_update` (never populated)
**Issue**: Server has `config_overrides` table with hierarchy (tenant→group→endpoint) but no mechanism pushes config changes to agents. Heartbeat response field exists but is always nil.
**Fix**: Implement config push via heartbeat response or UPDATE_CONFIG commands.

### H16. Duplicate agent_binaries Table [DATABASE]
**Issue**: `agent_binaries` exists in both Hub DB (migration 001) and Server DB (migration 049) with identical schema. No sync mechanism between them.
**Fix**: Server should query Hub via API, not maintain separate copy.

### H17. Hardware Details Stored in Two Formats [DATABASE]
**Issue**: Structured columns (`cpu_model`, `memory_total_mb`) AND JSONB `hardware_details` column both exist on endpoints table. No clear canonical source.
**Fix**: Pick one representation, deprecate the other.

### H18. web-hub and web-agent Have No Error Boundaries [FRONTEND]
**Issue**: Any component crash in these apps takes down the entire application. Only web/ has a partial error boundary (dashboard widgets only).
**Fix**: Add global React error boundary at root level in all three apps.

### H19. Tenant Data Isolation Has No Integration Test [TESTING]
**Issue**: Platform relies on tenant_id filtering but NO test verifies one tenant can't access another's data. If a query forgets `WHERE tenant_id = $1`, all tenants see all data.
**Fix**: Add cross-tenant access integration test.

### H20. Integration Tests Not Run in CI [TESTING]
**File**: `.github/workflows/test-unit.yml`
**Issue**: Integration tests exist (8 total) but are tagged `//go:build integration` and CI only runs `go test ./...` (no `-tags=integration`).
**Fix**: Add integration test step to CI pipeline.

### H21. Database Store Layer Has 4 Tests for 34 Files [TESTING]
**Issue**: Server store package has 34 Go files but only `db_test.go`, `agent_registrations_test.go`, `inventory_test.go`, `vulnerability_test.go`. All query correctness is untested.
**Fix**: Add tests for critical query paths, especially tenant-filtered queries.

---

## PART 3: MEDIUM ISSUES

### Database
- **M1.** Policy evaluation denormalized columns on `policies` table have no consistency trigger with `policy_evaluations` table
- **M2.** `compliance_control_results.passing_endpoints` denormalized from `compliance_evaluations` with no sync mechanism
- **M3.** Hub `clients` table denormalized summaries (`os_summary`, `endpoint_status_summary`) from `client_sync_history` with no trigger
- **M4.** `notification_preferences` boolean columns (`email_enabled`, `slack_enabled`) duplicate data derivable from `notification_channels`
- **M5.** CVE data stored redundantly: Hub `cve_feeds`, Server `cves`, Server `endpoint_cves` — no consistency guarantees
- **M6.** Missing FK constraints: `alerts.rule_id`, `compliance_evaluations.endpoint_id`, `compliance_evaluations.cve_id`

### Scalability
- **M7.** Watermill event bus uses PostgreSQL polling — will bottleneck at 1000 endpoints generating 16+ events/sec
- **M8.** No data caching layer — CVE metadata, endpoint details, deployment status re-fetched from DB every request
- **M9.** Wave dispatcher runs every 30 seconds — too frequent for large deployments, should be 2 minutes
- **M10.** Heartbeat processing creates DB transaction per message — no batching for concurrent heartbeats

### Production Readiness
- **M11.** gRPC reflection enabled in production — exposes full API surface to unauthenticated clients
- **M12.** Sensitive data logged: Valkey URL (may contain credentials), Hub credentials on MinIO failure
- **M13.** No circuit breaker for external calls (NVD API, Hub, MinIO)
- **M14.** CORS falls back to localhost dev origins if not configured — should fail loudly
- **M15.** MemoryRateLimitStore has no cleanup of expired buckets — unbounded memory growth under attack

### Frontend
- **M16.** 13 files with `as any` TypeScript casts — type safety lost on policy creation, deployment detail, compliance pages
- **M17.** Fixed-width wizards (PolicyWizard: 680px, DeploymentWizard: similar) — overflow on smaller screens
- **M18.** No route-level code splitting — @xyflow/react (12MB), recharts, codemirror loaded upfront
- **M19.** Dashboard grid uses `repeat(4, 1fr)` with no responsive breakpoints

### Integration
- **M20.** Catalog sync loses patch-to-CVE relationships — Hub `patch_catalog_cves` junction table data not transmitted
- **M21.** Command result parsing is fragile — unknown output types return empty strings, losing all result data
- **M22.** Inventory insertion post-commit operations are best-effort — hardware details, software summary can silently fail
- **M23.** Hub client scope filtering shows "available in a future release" placeholder
- **M24.** Agent enrollment stores architecture via tags side-channel instead of first-class proto field
- **M25.** Hub sync settings `UpdatedBy` uses tenant ID instead of actual user ID (TODO PIQ-245)

---

## PART 4: PRIOR AUDIT STATUS (6 of 10 Still Open)

| # | Finding | Status |
|---|---------|--------|
| 1 | JWT claim injection | **STILL OPEN** → C1 above |
| 2 | PolicyAutoDeployed event | **STILL OPEN** → C3 above |
| 3 | Missing domain events | **STILL OPEN** → C13 above |
| 4 | Duplicate UserID keys | **STILL OPEN** → C11 above |
| 5 | No 2027 audit partitions | **STILL OPEN** → C14 above |
| 6 | deployment_targets indexes | **STILL OPEN** → H1 above |
| 7 | Workflow RBAC missing | **STILL OPEN** → C2 above |
| 8 | Postgres password mismatch | **FIXED** — passwords consistent |
| 9 | Hub route path mismatch | **RESOLVED** — path is `/register/status`, correct |
| 10 | Scan endpoint no-op | **FIXED** — properly triggers scan |

---

## PART 5: CROSS-PLATFORM INTEGRATION FLOW STATUS

| Flow | Direction | Status | Blocking Issues |
|------|-----------|--------|-----------------|
| Patch Catalog Sync | Hub → Server | **PARTIAL** | CVE relationships lost in transit; proto/HTTP mismatch |
| CVE Data Flow | Hub → Server → Agent | **BROKEN** | No Hub→Server CVE sync; agent CVE detection dropped |
| Agent Enrollment | Agent → Server | **COMPLETE** | Token expiry not checked; mTLS not generated; no endpoint limit |
| Heartbeat | Agent ↔ Server | **COMPLETE** | Config/update not pushed; no offline detection |
| Inventory Sync | Agent → Server | **COMPLETE** | Agent CVE data ignored; hardware details inconsistent |
| Deployment Execution | Server → Agent → Server | **PARTIAL** | No command timeout; reboot not handled; fragile result parsing |
| License Validation | Server → Hub | **BROKEN** | No implementation exists |
| Binary Distribution | Hub → Server → Agent | **INCOMPLETE** | No Hub→Server sync; no update notification |
| Config Propagation | Server → Agent | **INCOMPLETE** | No push mechanism; hierarchy resolution missing |
| Audit Trail | All platforms | **PARTIAL** | No cross-platform correlation ID |

---

## PART 6: RECOMMENDED FIX ORDER

### Phase 1: Security + Data Integrity (Week 1)
1. Fix JWT claim injection (C1) — 2 hours
2. Add RBAC to workflow routes (C2) — 1 hour
3. Add PolicyAutoDeployed to AllTopics (C3) — 30 minutes
4. Fix duplicate UserID context keys (C11) — 2 hours
5. Add missing domain event emissions (C13) — 2 hours
6. Add rate limiting to all API endpoints (H7) — 4 hours

### Phase 2: Scalability (Week 1-2)
7. Increase DB pool size (C7) — 30 minutes (config change)
8. Add LIMIT to all unbounded queries (C8, H2) — 4 hours
9. Refactor ListPatchesFiltered subqueries (C9) — 4 hours
10. Set gRPC MaxConcurrentStreams (C10) — 1 hour
11. Add missing database indexes (H1) — 2 hours
12. Create 2027 audit partitions (C14) — 1 hour
13. Split River into priority queues (H3) — 4 hours
14. Batch deployment target creation (H4) — 2 hours

### Phase 3: Integration Flows (Week 2-3)
15. Implement endpoint offline detection (C12) — 4 hours
16. Process agent CVE detection OR remove proto field (C4) — 4 hours
17. Implement license validation client (C6) — 8 hours
18. Implement config push via heartbeat (H15) — 8 hours
19. Add command timeout enforcement (H13) — 4 hours
20. Add enrollment token expiry (H11) — 4 hours
21. Add cross-platform correlation ID (H14) — 8 hours

### Phase 4: Feature Completeness (Week 3)
22. Build hub connection UI form (H5) — 4 hours
23. Make agent server URL reconfigurable (H6) — 4 hours
24. Add error boundaries to all frontends (H18) — 2 hours
25. Extend health checks (H8) — 4 hours
26. Fail fast on missing production config (H10, H9) — 4 hours

### Phase 5: Testing (Week 3-4)
27. Add tenant isolation integration test (H19) — 6 hours
28. Enable integration tests in CI (H20) — 1 hour
29. Add database store layer tests (H21) — 12 hours
30. Add CVE correlation version range tests — 3 hours

**Total estimated effort: ~110 hours (3-4 developer-weeks)**

---

## APPENDIX: Files Most Frequently Cited

| File | Issues |
|------|--------|
| `internal/server/store/queries/patches.sql` | C9, H2 |
| `internal/server/store/queries/endpoints.sql` | C8, H2 |
| `internal/server/api/router.go` | C2, H7, M14 |
| `internal/server/grpc/sync_outbox.go` | C4, M22 |
| `internal/server/events/topics.go` | C3 |
| `internal/hub/auth/session.go` | C1 |
| `configs/server.yaml` | C7, H3 |
| `internal/server/grpc/server.go` | C10 |
| `internal/server/grpc/heartbeat.go` | C12, H15 |
| `cmd/server/main.go` | H9, H10, M11 |
| `internal/server/api/v1/hub_sync.go` | C13, H5 |
| `web/src/pages/settings/PatchSourcesSettingsPage.tsx` | H5 |
