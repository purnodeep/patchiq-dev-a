# PatchIQ Full Platform Audit v2 — Client Deployment Readiness

**Date**: 2026-04-09
**Branch**: dev-a
**Scope**: 14 parallel audit agents, exhaustive line-by-line analysis
**Goal**: Complete inventory of every issue before client's 1-month stress test at 1000+ endpoints

---

## Executive Summary

**Total issues found: 200+** across every layer of the platform.

The platform has solid architecture and good patterns in many areas, but is riddled with **incomplete implementations, silent failures, dead code, and scalability blockers**. The "last 20%" problem from the prior audit is confirmed — but it's worse than 20%. Key workers are never instantiated, compliance frameworks are stubs, 90 SQL queries are dead code, critical integration flows between Hub/Server/Agent are broken, and the system will hit hard scalability walls at ~200 concurrent agents.

| Category | Critical | High | Medium | Low | Total |
|----------|----------|------|--------|-----|-------|
| Security | 5 | 3 | 3 | 1 | 12 |
| Integration (Hub/Server/Agent) | 4 | 8 | 6 | 0 | 18 |
| Scalability & Performance | 5 | 6 | 5 | 0 | 16 |
| Broken/Incomplete Features | 5 | 5 | 8 | 4 | 22 |
| Data Integrity & Events | 4 | 3 | 3 | 0 | 10 |
| Error Handling & Stability | 4 | 5 | 5 | 0 | 14 |
| Dead Code & Clutter | 0 | 2 | 3 | 2 | 7 |
| Database Schema | 1 | 4 | 7 | 1 | 13 |
| Testing & CI | 0 | 4 | 3 | 0 | 7 |
| Frontend | 1 | 3 | 5 | 3 | 12 |
| Config & Infrastructure | 2 | 3 | 3 | 1 | 9 |
| Production Readiness | 1 | 4 | 3 | 0 | 8 |
| **TOTAL** | **32** | **50** | **54** | **12** | **148 discrete issues** |

Plus: 90 unused SQL queries, 5 orphaned frontend components, 29 TODO/FIXME comments, 448 generic error messages.

---

## PART 1: CRITICAL ISSUES (32 total — fix before any deployment)

### Security (5)

**S1. JWT Claim Injection — Hub Auth** [STILL OPEN FROM PRIOR AUDIT]
- `internal/hub/auth/session.go:39`
- `mintJWT` uses `fmt.Sprintf` with user-controlled email/name to build JSON claims. Attacker injects arbitrary claims.
- Fix: Use `json.Marshal` with typed struct.

**S2. Workflow CRUD Routes Missing RBAC** [STILL OPEN]
- `internal/server/api/router.go:330-345`
- All workflow routes (List, Create, Get, Update, Delete, Publish) have no `.With(rp(...))` middleware.
- Fix: Add RBAC middleware matching other resource routes.

**S3. Agent HTTP API Has Zero Authentication**
- `internal/agent/api/handler.go:26-62`
- No auth middleware on any agent endpoint. Any network client can read/modify agent settings, trigger scans, view patch history.
- Fix: Add shared-secret or certificate-based auth middleware.

**S4. Hub License Keys Are Forgeable Placeholders**
- `internal/hub/api/v1/licenses.go:131-137`
- License keys are just JSON-encoded data, not RSA-signed. Comment: "M2 will use RSA-signed keys."
- Fix: Implement cryptographic license signing.

**S5. Hardcoded Dev Credentials in Committed Config Files**
- `configs/server.yaml`, `configs/hub.yaml`, `docker-compose.dev.yml`
- DB passwords (`patchiq:patchiq`), MinIO creds (`minioadmin:minioadmin`), Zitadel master key all in git.
- Fix: Move to env vars, add `.env.example` template, remove from configs.

### Integration — Hub/Server/Agent (4)

**I1. No Hub→Server CVE Sync Protocol — CVE Flow is BROKEN**
- Hub curates CVEs from 6 feeds. Server ignores Hub entirely and independently fetches only NVD.
- Agent sends `detected_cves` in inventory proto — Server silently drops it (`sync_outbox.go:141-254`).
- Fix: Extend catalog sync to include CVE data, or create dedicated CVE sync job.

**I2. License Validation is a Complete No-Op**
- Proto defines `ValidateLicense` RPC but Server has no gRPC client to call it.
- License expiry never checked. Endpoint count limits never enforced. Enrollment accepts unlimited agents.
- Fix: Implement license validation client with periodic revalidation.

**I3. Nil Workers Passed to River — Crash Risk**
- `cmd/server/main.go:412` — `RegisterWorkers()` receives TWO nil pointers: `userSyncWorker` and `policySchedulerWorker` are never instantiated.
- `workers/registry.go:33-47` calls `river.AddWorker(workers, nil)` which will panic.
- Fix: Either instantiate workers or remove from registration.

**I4. Policy Scheduler Worker Never Created — Auto-Deploy Policies Are Dead**
- `internal/server/policy/scheduler.go` — fully implemented but never instantiated in `main.go`.
- No periodic River job registered. Automatic policy-based deployments never execute.
- Fix: Instantiate worker, register periodic job.

### Scalability (5)

**P1. DB Connection Pool: 25 Max for 1000 Agents**
- `configs/server.yaml:25-26` — `max_conns: 25, min_conns: 5`
- 1000 agents × heartbeat transactions = guaranteed pool exhaustion.
- Fix: Increase to `max_conns: 200, min_conns: 50`.

**P2. ListEndpointsByTenant Has No LIMIT — Loads All 1000 Rows**
- `internal/server/store/queries/endpoints.sql:55-56`
- Used in QuickDeploy (`patches.go:659`) — loads ALL endpoints into memory, filters in Go.
- Fix: Add WHERE clause filtering + LIMIT, or use paginated query.

**P3. ListPatchesFiltered Runs 5 Correlated Subqueries Per Row**
- `internal/server/store/queries/patches.sql:93-123`
- 50 patches × 5 subqueries = 250 subqueries per page load. Dashboard unusable at scale.
- Fix: Refactor to CTEs with pre-aggregated stats.

**P4. gRPC Server Has No MaxConcurrentStreams**
- `internal/server/grpc/server.go:24-38`
- 1000 agents each with heartbeat + sync streams = unlimited connections → memory exhaustion.
- Fix: Set `MaxConcurrentStreams` proportional to pool size.

**P5. No 2027 Audit Partitions** [STILL OPEN]
- `internal/server/store/migrations/001_init_schema.sql`
- Partitions defined through 2026-12 only. Default partition catches everything after.
- Fix: Add migration creating 2027 monthly partitions.

### Broken/Incomplete Features (5)

**F1. Compliance Check Workflow Handler Always Returns Pass**
- `internal/server/workflow/handlers/compliance_check.go:12-39`
- Comment: "M2 stub that always returns pass." Any workflow using compliance check nodes will always pass regardless of actual compliance state.
- Fix: Implement real compliance evaluation or disable the node type.

**F2. Compliance Frameworks Are Partially Stubs**
- `internal/server/compliance/frameworks.go:83-511`
- CIS Controls: 6 controls have NO evaluation logic ("available in a future release").
- HIPAA: 2 of 3 controls are stubs. Only HIPAA-164.308a1 has real logic.
- Fix: Implement control evaluation or mark as "not evaluated" in UI.

**F3. User Sync Worker Never Created — Zitadel Users Don't Sync**
- `internal/server/workers/user_sync.go` — fully implemented but never instantiated.
- No periodic job registered. Users created in Zitadel never appear in PatchIQ.
- Fix: Instantiate worker, register periodic job.

**F4. No Endpoint Offline Detection**
- `internal/server/grpc/heartbeat.go`
- Endpoints marked "online" on heartbeat but NO background job marks them "offline" after missed heartbeats.
- Fix: Add periodic job scanning for `last_heartbeat_at < threshold`.

**F5. Patch Recall Button is a No-Op**
- `web/src/pages/patches/PatchDetailPage.tsx:1985`
- Button appears functional but only hides a notice. `TODO(#306)` — API not called.
- Fix: Implement API call or hide the button.

### Data Integrity & Events (4)

**D1. PolicyAutoDeployed Event Not in AllTopics** [STILL OPEN]
- `internal/server/events/topics.go:38` — Topic defined but missing from `AllTopics()` function.
- `Emit()` calls silently fail. Auto-deploy policies create no audit trail.

**D2. 2+ Write Operations Emit No Domain Events** [STILL OPEN]
- `notifications.go:584-634` (UpdatePreferences) — no event emission.
- `hub_sync.go:161-204` (UpdateConfig) — no event emission.
- `patches.go:594` (QuickDeploy) — creates deployments with no event.
- `patches.go:790` (DeployCritical) — creates deployments with no event.

**D3. Duplicate UserID Context Keys** [STILL OPEN]
- `internal/shared/user/context.go` uses `userCtxKey struct{}`.
- `internal/shared/otel/context.go` uses `userIDKey struct{}`.
- Different keys → user IDs missing from ALL traces and structured logs.

**D4. Event Emission After Commit — State Inconsistency Risk**
- Pattern across all handlers: DB commit succeeds, then event emitted, then response sent.
- If event emission fails, data is committed but no audit trail. TODO(#177) acknowledges this.

### Error Handling (4)

**E1. tenant.MustTenantID() Panics — Used in 28 Handlers**
- `internal/shared/tenant/context.go:24-29`
- If middleware misconfigured, entire API server panics instead of returning 401/500.
- Fix: Return error instead of panic, or ensure middleware is bulletproof.

**E2. Workflow Worker Ignores DB Errors, Risks Nil Dereference**
- `internal/server/workflow/worker.go:404` — `ne, _ := q.GetNodeExecutionByNodeID(...)` discards error.
- Line 409: `ne.ID` accessed without nil check → panic if query fails.
- Fix: Check error, handle nil.

**E3. Workflow Worker Ignores DB Update Results**
- `internal/server/workflow/worker.go:394, 409` — `_, _ = q.UpdateNodeExecution(...)` completely ignores errors.
- Execution state silently not recorded.

**E4. Notification History Pagination — Empty Slice Panic Risk**
- `internal/server/api/v1/notifications.go:753`
- `nextCursor = history[len(history)-1].ID` — panics if `history` is empty and condition passes.

---

## PART 2: HIGH ISSUES (50 total)

### Security (3)
- **H-S1.** Rate limiting only on auth endpoints (`router.go:77-80`). No API-wide rate limiting.
- **H-S2.** Hub client registration uses hardcoded default tenant ID (`clients.go:85`). Breaks multi-tenancy.
- **H-S3.** gRPC reflection configurable but no warning when enabled in production.

### Integration (8)
- **H-I1.** Enrollment token never expires — no `expires_at` column. Proto defines TOKEN_EXPIRED error but never checked.
- **H-I2.** mTLS certificate field in EnrollResponse always empty — no cert generation code exists.
- **H-I3.** Command timeout not enforced — agent reboot mid-deployment hangs forever.
- **H-I4.** No cross-platform audit correlation ID between Hub/Server/Agent events.
- **H-I5.** Config propagation to agents not implemented — heartbeat `config_update` field always nil.
- **H-I6.** Duplicate `agent_binaries` table in both Hub and Server DBs with no sync mechanism.
- **H-I7.** Catalog sync loses patch-to-CVE relationships (Hub `patch_catalog_cves` not transmitted).
- **H-I8.** Deleted patches from Hub not soft-deleted on Server (`TODO(PIQ-118)`).

### Scalability (6)
- **H-P1.** 20+ unbounded queries without LIMIT across server store queries.
- **H-P2.** River job queue: single queue, 100 max workers — all job types compete. Notifications can starve deployments.
- **H-P3.** Deployment target creation: 1 INSERT per endpoint in a loop (`patches.go:729-743`).
- **H-P4.** Missing database indexes on hot-path tables (deployment_targets, endpoint_cves, cves, patches, workflows).
- **H-P5.** Watermill event bus uses PostgreSQL polling — bottleneck at 1000+ endpoints.
- **H-P6.** No data caching layer — CVE metadata, endpoint details, deployment status re-fetched every request.

### Features (5)
- **H-F1.** Hub connection UI missing — backend exists, frontend form never built. `useUpdateSyncConfig()` hook unused.
- **H-F2.** Agent server URL not reconfigurable after enrollment.
- **H-F3.** Patch "Add to Group" dialog completes but does nothing (`TODO(#306)`).
- **H-F4.** Wave dispatcher has no rollback trigger on failure threshold exceeded.
- **H-F5.** Fresh install: no default tenant → all periodic jobs fail silently (`main.go:127-131`).

### Data Integrity (3)
- **H-D1.** RLS bypass if `txFactory` is nil in TimeoutChecker/WaveDispatcher — tenant data leaks.
- **H-D2.** RLS context SET error ignored during bootstrap (`main.go:334-335`).
- **H-D3.** Post-commit operations in inventory sync are best-effort — hardware, software, network data silently lost.

### Error Handling (5)
- **H-E1.** 448 instances of generic "INTERNAL_ERROR" with no resource context across all handlers.
- **H-E2.** No JSON payload size limits — `json.NewDecoder(r.Body).Decode()` accepts unbounded input.
- **H-E3.** CSV audit export writes silently fail — client receives partial/corrupted data (`audit.go:232,277,304`).
- **H-E4.** CVE correlator exits on first error — single patch lookup failure stops all correlation.
- **H-E5.** Schedule checker validates cron at execution time, not creation time — bad expressions fail every 30s.

### Database (4)
- **H-DB1.** CVE data stored in 3 places (Hub `cve_feeds`, Server `cves`, Server `endpoint_cves`) with no consistency mechanism.
- **H-DB2.** Hardware details stored in both structured columns AND JSONB — no canonical source.
- **H-DB3.** Policy evaluation denormalized columns have no consistency trigger/check.
- **H-DB4.** Missing FK constraints: `alerts.rule_id`, `compliance_evaluations.endpoint_id/cve_id`.

### Testing (4)
- **H-T1.** Tenant data isolation has NO integration test — if a query forgets `WHERE tenant_id`, all tenants see all data.
- **H-T2.** Integration tests not run in CI (build tag `integration` never passed).
- **H-T3.** Database store layer: 4 tests for 34 files — all query correctness untested.
- **H-T4.** CVE correlation version range logic completely untested.

### Frontend (3)
- **H-FE1.** web-hub and web-agent have NO React error boundaries — component crash = full app crash.
- **H-FE2.** 13 files with `as any` TypeScript casts — type safety lost on policy creation, deployment detail, compliance.
- **H-FE3.** KEV vulnerability status column is placeholder dash ("—") across endpoint CVE tabs (`TODO PIQ-243`).

### Production Readiness (4)
- **H-PR1.** Health checks only verify database — missing gRPC, Valkey, event bus, River queue.
- **H-PR2.** Idempotency store falls back to in-memory when Valkey unavailable — guarantees lost on restart.
- **H-PR3.** Notification encryption key generated ephemerally — changes on restart, breaking stored credentials.
- **H-PR4.** Missing config validation at startup — services start with incomplete config and fail later.

### Config & Infrastructure (3)
- **H-CI1.** OpenAPI types not auto-generated — `api-client` Makefile target is deferred. TypeScript types drift from API.
- **H-CI2.** No Grafana dashboards committed — lost on container recreation.
- **H-CI3.** SSL disabled in all config files (`sslmode=disable`).

### Dead Code (2)
- **H-DC1.** 90 unused SQL queries generating dead Go code (full list in dead code inventory).
- **H-DC2.** Compliance Evidence tab intentionally removed — feature gap visible to users.

---

## PART 3: MEDIUM ISSUES (54 total — condensed)

### Database Schema (7)
- M-DB1. `notification_preferences` boolean columns duplicate `notification_channels` data
- M-DB2. `compliance_control_results` denormalized counters with no sync mechanism
- M-DB3. Hub `clients` table denormalized summaries with no trigger
- M-DB4. `binary_fetch_state` duplicates `patch_catalog` columns
- M-DB5. `workflow_executions` missing error history (only last error stored)
- M-DB6. Audit FTS index missing tenant_id composite
- M-DB7. Inconsistent index naming conventions

### Scalability (5)
- M-P1. Wave dispatcher runs every 30 seconds — too frequent for large deployments
- M-P2. Heartbeat processing: 1 DB transaction per message, no batching
- M-P3. Compliance evaluation lacks SLA deadline checking and threshold alerts
- M-P4. Discovery engine: partial failures are silent (3/5 repos fail → job succeeds)
- M-P5. CVE sync has no pagination fallback for NVD API

### Features (8)
- M-F1. Failed Eval stat card on policies page has empty onClick handler
- M-F2. Compliance export report button permanently disabled ("Soon")
- M-F3. Hub client scope filtering shows "available in a future release" placeholder
- M-F4. Agent enrollment wizard missing — users given token + curl command only
- M-F5. Hub settings `UpdatedBy` uses tenant ID not user ID (`TODO PIQ-245`)
- M-F6. Agent enrollment stores architecture via tags side-channel, not proto field
- M-F7. Heartbeat response never sends config_update or update_available
- M-F8. Alert subscriber is synchronous — will block event processing at scale

### Integration (6)
- M-I1. Command result parsing fragile — unknown output types return empty strings
- M-I2. Inventory deduplication on retry causes duplicate CVE match jobs
- M-I3. Package uniqueness constraint may reject legitimate multi-source packages
- M-I4. No acknowledgment of command results — agent may re-send on crash
- M-I5. Hub sync bootstrap silently fails — wrong Hub credentials not detected until sync runs
- M-I6. Catalog sync: post-sync CVE sync is fire-and-forget

### Error Handling (5)
- M-E1. 11 instances of `defer tx.Rollback(ctx) //nolint:errcheck` in CVE store adapter
- M-E2. `cmd/server/main.go:502` — type assertion error discarded, proceeds with empty string
- M-E3. Silent JSON unmarshal failures in deployment wave_config
- M-E4. Maintenance window time parsing errors ignored (`_, _ = fmt.Sscanf`)
- M-E5. Response.go: partial JSON already sent when encode error occurs

### Frontend (5)
- M-FE1. Fixed-width wizards (680px) overflow on smaller screens
- M-FE2. No route-level code splitting — 12MB+ @xyflow loaded upfront
- M-FE3. Dashboard grid `repeat(4, 1fr)` with no responsive breakpoints
- M-FE4. Accessibility gaps — missing aria labels on action buttons and tables
- M-FE5. TanStack Query staleTime inconsistencies across hooks

### Production (3)
- M-PR1. CORS falls back to localhost dev origins if not configured
- M-PR2. MemoryRateLimitStore has no expired bucket cleanup — unbounded growth
- M-PR3. No circuit breaker for external calls (NVD, Hub, MinIO)

### Security (3)
- M-S1. Sensitive data logged: Valkey URL with potential credentials, Hub credentials on error
- M-S2. World-readable/group-writable script permissions on shared dev server
- M-S3. No `.gitignore` for `*.key`, `*.pem`, `*.pfx` certificate files

### Config (3)
- M-C1. Zitadel `service_account_key: ""` silently skipped if missing
- M-C2. Agent config `data_dir: "~/.patchiq"` tilde expansion not documented
- M-C3. Down migrations exist but never tested in CI

### Dead Code (3)
- M-DC1. 5 orphaned frontend components (SegmentedProgressBar, CVSSVectorBreakdown, SlidePanel, SeverityPills, StatsStrip)
- M-DC2. 29 TODO/FIXME comments, many referencing old issue numbers
- M-DC3. `console.error()` left in production code (SoftwareTab, CreateTagDialog)

### Testing (3)
- M-T1. Deployment wave-level state transitions untested
- M-T2. Compliance multi-tenant evaluation untested
- M-T3. Hub-Server sync end-to-end untested

---

## PART 4: PRIOR AUDIT STATUS

| # | Finding | Status |
|---|---------|--------|
| 1 | JWT claim injection | **STILL OPEN** → S1 |
| 2 | PolicyAutoDeployed event | **STILL OPEN** → D1 |
| 3 | Missing domain events | **STILL OPEN** → D2 |
| 4 | Duplicate UserID keys | **STILL OPEN** → D3 |
| 5 | No 2027 audit partitions | **STILL OPEN** → P5 |
| 6 | deployment_targets indexes | **STILL OPEN** → H-P4 |
| 7 | Workflow RBAC missing | **STILL OPEN** → S2 |
| 8 | Postgres password mismatch | **FIXED** |
| 9 | Hub route path mismatch | **RESOLVED** |
| 10 | Scan endpoint no-op | **FIXED** |

**6 of 10 prior critical findings remain open.**

---

## PART 5: CROSS-PLATFORM INTEGRATION STATUS

| Flow | Hub→Server→Agent | Status | Key Blocker |
|------|-------------------|--------|-------------|
| Patch Catalog Sync | Hub → Server | **PARTIAL** | CVE relationships lost; deletes not propagated |
| CVE Data Flow | Hub → Server → Agent | **BROKEN** | No Hub→Server CVE sync; agent CVEs dropped |
| Agent Enrollment | Agent → Server | **COMPLETE** | Token expiry unchecked; no endpoint limit |
| Heartbeat | Agent ↔ Server | **COMPLETE** | No offline detection; no config push |
| Inventory Sync | Agent → Server | **COMPLETE** | CVE data ignored; hardware inconsistent |
| Deployment Execution | Server → Agent → Server | **PARTIAL** | No command timeout; no wave rollback trigger |
| License Validation | Server → Hub | **BROKEN** | No implementation exists |
| Binary Distribution | Hub → Server → Agent | **INCOMPLETE** | No Hub→Server sync; no update notification |
| Config Propagation | Server → Agent | **INCOMPLETE** | No push mechanism |
| Audit Trail | All platforms | **PARTIAL** | No cross-platform correlation ID |

---

## PART 6: DEAD CODE INVENTORY

### 90 Unused SQL Queries (top offenders)
| File | Unused Count | Examples |
|------|-------------|----------|
| deployments.sql | 11 | CreateDeployment, CreateDeploymentTarget, ListDeploymentsByTenant, UpdateDeploymentStatus |
| patches.sql | 8 | CreatePatch, UpdatePatch, ListPatchesByTenant, CreateCVE |
| compliance.sql | 6 | CountEvaluationsByState, DeleteOldControlResults, ListSLADeadlinesApproaching |
| audit.sql | 5 | CountAuditEventsByEndpoint, ListAuditEventsByActor/Endpoint/Resource/Type |
| config.sql | 5 | All config override CRUD operations |
| roles.sql | 4 | CountRoleUsers, GetRoleByName, ListRoleUsers, ListRoles |
| cve_feeds.sql (hub) | 5 | Full CRUD for cve_feeds |
| agent_binaries.sql (hub) | 5 | Full CRUD for agent_binaries |

### 5 Orphaned Frontend Components
- `web/src/components/SegmentedProgressBar.tsx`
- `web/src/components/CVSSVectorBreakdown.tsx`
- `web/src/components/SlidePanel.tsx`
- `web-hub/src/components/SeverityPills.tsx`
- `web-hub/src/components/StatsStrip.tsx`

### 29 TODO/FIXME Comments (critical ones)
- `PIQ-116`: Replace insecure agent credentials with mTLS
- `PIQ-118`: Soft-delete patches when Hub delete-sync implemented
- `PIQ-145`: Add event emission failure counter (2 locations)
- `PIQ-243`: Show KEV flag when API exposes kev_due_date
- `PIQ-244`: Decode notification recipient when crypto layer supports it
- `PIQ-245`: Use actual user ID in Hub settings audit
- `#177`: Outbox pattern for event delivery guarantees (2 locations)
- `#177`: TOCTOU race in workflow publish
- `#306`: Patch recall API call + patch group association (2 locations)

---

## PART 7: PHASED FIX PLAN

### Phase 1: Stop the Bleeding (Week 1) — ~40 hours
**Security + crash prevention. Non-negotiable before any deployment.**

| # | Fix | Est. Hours | Issues Addressed |
|---|-----|-----------|-----------------|
| 1 | Fix JWT claim injection (use json.Marshal) | 1h | S1 |
| 2 | Add RBAC to workflow routes | 1h | S2 |
| 3 | Add auth middleware to agent API | 4h | S3 |
| 4 | Fix nil workers in RegisterWorkers (instantiate or remove) | 2h | I3, I4, F3 |
| 5 | Add PolicyAutoDeployed to AllTopics() | 0.5h | D1 |
| 6 | Fix duplicate UserID context keys | 2h | D3 |
| 7 | Add domain events to QuickDeploy, DeployCritical, UpdatePreferences, UpdateConfig | 3h | D2 |
| 8 | Fix MustTenantID() to return error instead of panic | 4h | E1 |
| 9 | Fix nil dereference in workflow worker | 1h | E2, E3 |
| 10 | Fix notification history empty slice panic | 0.5h | E4 |
| 11 | Increase DB pool to max_conns: 200 | 0.5h | P1 |
| 12 | Set gRPC MaxConcurrentStreams: 500 | 0.5h | P4 |
| 13 | Add 2027 audit partitions migration | 1h | P5 |
| 14 | Add missing database indexes | 2h | H-P4 |
| 15 | Add LIMIT to all unbounded queries | 4h | P2, H-P1 |
| 16 | Create default tenant on fresh install | 2h | H-F5 |
| 17 | Disable compliance check workflow node (returns misleading "pass") | 1h | F1 |
| 18 | Add endpoint offline detection job | 4h | F4 |
| 19 | Extend health checks to all dependencies | 4h | H-PR1 |
| 20 | Require notification encryption key in production | 1h | H-PR3 |

### Phase 2: Scalability (Week 2) — ~30 hours
**Make the system survive 1000 endpoints.**

| # | Fix | Est. Hours | Issues Addressed |
|---|-----|-----------|-----------------|
| 21 | Refactor ListPatchesFiltered to use CTEs | 4h | P3 |
| 22 | Batch deployment target creation (multi-row INSERT) | 2h | H-P3 |
| 23 | Split River into priority queues | 4h | H-P2 |
| 24 | Add JSON payload size limits | 2h | H-E2 |
| 25 | Move QuickDeploy endpoint filtering to SQL WHERE | 2h | P2 |
| 26 | Add rate limiting to all API endpoints | 4h | H-S1 |
| 27 | Add config validation at startup (fail fast) | 4h | H-PR4 |
| 28 | Fix cron expression validation at schedule creation | 2h | H-E5 |
| 29 | Fix CVE correlator to continue on single-error | 1h | H-E4 |
| 30 | Add error context to critical API responses | 4h | H-E1 (partial) |

### Phase 3: Integration Flows (Week 2-3) — ~50 hours
**Make the three-tier system work as one.**

| # | Fix | Est. Hours | Issues Addressed |
|---|-----|-----------|-----------------|
| 31 | Process agent CVE detection in inventory sync (or remove proto field) | 4h | I1 |
| 32 | Implement Hub→Server CVE sync in catalog sync job | 8h | I1 |
| 33 | Implement license validation client | 8h | I2 |
| 34 | Add enrollment token expiry | 4h | H-I1 |
| 35 | Add command timeout enforcement | 4h | H-I3 |
| 36 | Implement config push via heartbeat | 6h | H-I5 |
| 37 | Add cross-platform correlation ID | 6h | H-I4 |
| 38 | Propagate Hub patch deletions to Server | 2h | H-I8 |
| 39 | Include CVE relationships in catalog sync | 4h | H-I7 |
| 40 | Implement wave failure threshold rollback | 4h | H-F4 |

### Phase 4: Feature Completeness (Week 3) — ~20 hours

| # | Fix | Est. Hours | Issues Addressed |
|---|-----|-----------|-----------------|
| 41 | Build hub connection UI form | 4h | H-F1 |
| 42 | Make agent server URL reconfigurable | 4h | H-F2 |
| 43 | Implement or hide patch recall + group association | 3h | F5, H-F3 |
| 44 | Add error boundaries to all 3 frontends | 2h | H-FE1 |
| 45 | Fix `as any` TypeScript casts | 3h | H-FE2 |
| 46 | Implement compliance framework controls (CIS, HIPAA stubs) | 4h | F2 |

### Phase 5: Testing & Cleanup (Week 3-4) — ~30 hours

| # | Fix | Est. Hours | Issues Addressed |
|---|-----|-----------|-----------------|
| 47 | Add tenant isolation integration test | 6h | H-T1 |
| 48 | Enable integration tests in CI | 1h | H-T2 |
| 49 | Add database store layer tests for critical queries | 8h | H-T3 |
| 50 | Remove 90 unused SQL queries | 4h | H-DC1 |
| 51 | Remove orphaned components + dead code | 2h | M-DC1 |
| 52 | Implement RSA-signed license keys | 8h | S4 |
| 53 | Move credentials to env vars, add .env.example | 2h | S5 |

**Total estimated effort: ~170 hours (4-5 developer-weeks)**

---

## APPENDIX A: Files Most Frequently Cited

| File | Issue Count |
|------|------------|
| `cmd/server/main.go` | 9 |
| `internal/server/api/router.go` | 5 |
| `internal/server/store/queries/patches.sql` | 4 |
| `internal/server/store/queries/endpoints.sql` | 4 |
| `internal/server/grpc/sync_outbox.go` | 4 |
| `internal/server/workflow/worker.go` | 4 |
| `internal/server/events/topics.go` | 3 |
| `internal/server/api/v1/patches.go` | 3 |
| `internal/server/api/v1/notifications.go` | 3 |
| `internal/server/deployment/wave_dispatcher.go` | 3 |
| `configs/server.yaml` | 3 |
| `internal/server/compliance/frameworks.go` | 3 |

## APPENDIX B: What's Actually Good

To be fair, significant parts of the platform are solid:

- **Architecture**: Clean separation between Hub/Server/Agent with shared packages
- **RLS**: Row-level security properly enforced on all tenant tables
- **Deployment state machine**: Well-tested with proper optimistic locking
- **Auth middleware chain**: OTel → RequestID → CORS → JWT → Tenant → User → RBAC
- **gRPC agent communication**: Enrollment, heartbeat, sync protocols well-designed
- **Frontend patterns**: TanStack Query hooks, Zod validation, proper loading/error states
- **CI/CD**: Codegen drift detection, multi-platform builds, disk space monitoring
- **Multi-user dev isolation**: Per-developer ports and databases
- **Audit system**: Append-only, partitioned, ULID-based — strong foundation
- **Dashboard**: All widgets functional with proper skeleton loading
