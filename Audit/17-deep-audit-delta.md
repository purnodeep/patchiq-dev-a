# PatchIQ Deep Audit — Delta Findings (Round 3)

**Date**: 2026-04-09
**Scope**: Line-by-line reading of every file across all 3 platforms
**Purpose**: New issues NOT in the v2 report (Audit/16-full-platform-audit-v2.md)

---

## NEW CRITICAL ISSUES

### NC1. SQL Injection via String Formatting in Tenant Context Setting
- `internal/server/workers/catalog_sync.go:267` — `fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantIDStr)`
- `internal/server/workflow/worker.go:83` — same pattern
- If tenantIDStr ever contains a single quote, SQL injection occurs. The rest of the codebase uses parameterized `set_config($1, true)`.

### NC2. Tenant Context Leak via set_config(false)
- `internal/server/deployment/wave_dispatcher.go:83` — `set_config('app.current_tenant_id', $1, false)` sets at SESSION level, not transaction-local
- `internal/server/workflow/worker.go:83` — same issue
- Tenant context persists on the pooled connection after the job completes, leaking to subsequent requests from other tenants.

### NC3. Race Condition on Global JWT Signing Key
- `internal/server/auth/jwt.go:100-101` — `var LocalSigningKey []byte` is mutable global state
- `internal/server/auth/login.go:143` — `InitSigningKey()` called on every login request, checks `len(c.SigningKey) == 0` without synchronization
- Two concurrent login requests can both see length 0, both generate keys, one overwrites the other, invalidating all JWTs signed with the overwritten key.

### NC4. MSIX Installer Command Injection
- `internal/agent/patcher/msix.go:16` — `fmt.Sprintf("Add-AppxPackage -Path '%s'", pkg.Name)`
- Single quote in pkg.Name allows PowerShell command injection.

### NC5. ALL Agent Logs Deleted on Every Restart
- `cmd/agent/main.go:163-168` — `DELETE FROM agent_logs WHERE id LIKE 'l%'`
- Intended to clean seed data (IDs like `l001`), but `generateLogID()` in `sqlite_log.go:113` produces IDs starting with `"log-"` which also matches `LIKE 'l%'`.
- Every agent restart deletes all operational logs.

### NC6. compliance_scores Missing GRANT UPDATE — Queries Will Fail at Runtime
- `internal/server/store/migrations/019_compliance.sql:111-113` — GRANT only gives SELECT, INSERT, DELETE
- Queries `UpdateEndpointScoresForRun` and `UpdateEndpointScoreByID` use UPDATE
- These queries fail when connected as `patchiq_app` role. Compliance scoring is broken.

### NC7. SLA Dashboard Widgets Show Entirely Fabricated Data
- `web/src/pages/dashboard/SLACountdown.tsx:130-155` — constructs fake SLA deadlines from hardcoded arrays, NOT from API data
- `web/src/pages/dashboard/SLAStatus.tsx:87-100` — progress percentages are calculated from array position, not server data
- Client will see "22d 12h remaining" on SLA regardless of actual vulnerability state.

### NC8. Invite Registration Sets Invalid Session Cookie
- `internal/server/auth/invite.go:254-259` — after Zitadel auth, stores raw user ID string (not a JWT) as session cookie
- JWT middleware rejects this on subsequent requests. Users who register via invite cannot authenticate.

### NC9. Downloaded Patch Binaries Never Cleaned Up on Agent
- `internal/agent/patcher/patcher.go:176-195` — `Downloader.Download` creates temp files, but `handleInstallPatch` never removes them
- No cleanup goroutine, no retention policy, no `defer os.Remove()`. Disk fills up over time.

---

## NEW HIGH ISSUES

### NH1. Overnight Maintenance Windows Broken
- `internal/server/deployment/maintenance.go:56-62` — if start > end (e.g., 22:00-06:00), check `minuteOfDay >= start && minuteOfDay < end` always returns false.
- Overnight maintenance windows are silently non-functional.

### NH2. MSRC Feed Cursor Comparison Fundamentally Broken
- `internal/hub/feeds/msrc.go:66-69` — `updateID <= cursor` uses lexicographic comparison on strings like "2025-Jan", "2025-Aug"
- "2025-Aug" < "2025-Feb" lexicographically, so August entries get skipped. MSRC delta sync is broken within a year.

### NH3. Binary Fetch Stores Files Under Wrong Path
- `internal/hub/workers/binary_fetch.go:109` — `FetchAndStore(ctx, fetchURL, fetch.OsDistribution, fetch.OsDistribution, filename)`
- Same value passed as both osFamily and osVersion. Files stored as `patches/ubuntu/ubuntu/curl.deb` instead of `patches/ubuntu/22.04/curl.deb`.

### NH4. Ubuntu Feed Severity Always "medium"
- `internal/hub/feeds/ubuntu.go:179` — `Severity: "medium"` hardcoded for all Ubuntu USN entries regardless of actual severity.

### NH5. Hub Login Always Returns role: "admin"
- `internal/hub/auth/login.go:133-139` and `Me` handler line 167 — hardcoded `Role: "admin"` regardless of actual user role. No RBAC integration.

### NH6. PackageAlias Update Ignores URL {id} Parameter
- `internal/hub/api/v1/package_aliases.go:180-228` — parses `{id}` from URL but calls `UpsertPackageAlias` using body fields, not the URL id. PUT to `/package-aliases/123` may create a new record or update a different one.

### NH7. Sync Events Emitted with Empty TenantID
- `internal/hub/api/v1/sync.go:226, 282, 307` — `domain.NewSystemEvent(events.SyncCompleted, "", ...)` passes empty string for tenant ID. Audit subscriber may fail or bypass RLS.

### NH8. Hub Binary UUID Formatting Non-Standard
- `internal/hub/workers/binary_fetch.go:167-172` — `uuidToStr` uses `%x-%x-%x-%x-%x` (no zero-padding). Same issue in `internal/server/workers/catalog_sync.go:404-409`. UUIDs may be shorter than 36 chars, breaking downstream matching.

### NH9. RPM/MSU/Apple Binary Fetchers Are Dead Code
- `internal/hub/catalog/` has `fetcher_yum.go`, `fetcher_msu.go`, `fetcher_apple.go` but they are NEVER instantiated. Only the generic `BinaryFetcher` is created in main.go. These fetcher implementations are fully written but unused.

### NH10. License Generator Fully Built But Never Wired
- `internal/hub/license/generate.go` — complete RSA license signing with tests
- `internal/hub/api/v1/licenses.go` — does NOT import or use it, uses JSON placeholders instead
- `cmd/hub/main.go` — does NOT initialize Generator or load RSA keys

### NH11. emitEvent Always Uses "system" Actor — 15+ Handlers Affected
- `internal/server/api/v1/helpers.go:68` — `emitEvent` always sets `ActorID: "system"`
- Affected: Groups, Tags, TagRules, DeploymentSchedules, Policies, NotificationChannels, Settings (general, IAM, role mapping), HubSync
- Only Roles, UserRoles, Registrations, and Alerts use `emitEventWithActor` with actual user ID
- Audit trail cannot identify WHO performed operations.

### NH12. Policy Update Not Transactional — 3 Separate Transactions
- `internal/server/api/v1/policies.go:834-916` — policy update, severity_filter update, and group replacement each in separate transactions
- If group update fails after policy update succeeds, data is inconsistent.

### NH13. GroupHandler.SetMembers Does Not Set Tenant Context in Transaction
- `internal/server/api/v1/groups.go:341` — transaction started without `SET LOCAL app.current_tenant_id`
- Bypasses RLS if PostgreSQL RLS policies are enforced via set_config.

### NH14. Dashboard INNER JOINs Exclude Ad-Hoc Deployments
- `internal/server/store/queries/dashboard.sql:21-35` and lines 91-106 — `JOIN policies p ON p.id = d.policy_id`
- Since migration 034 made `policy_id` nullable (for quick/ad-hoc deploys), INNER JOIN excludes them. Dashboard undercounts active deployments.

### NH15. Notification History Category Filter Breaks Pagination
- `internal/server/api/v1/notifications.go:729-743` — category filter applied in Go AFTER database query
- Total count doesn't reflect filter. Page may return fewer items than limit even when more exist.

### NH16. Duplicate Inconsistent Severity Filter Functions
- `internal/server/deployment/evaluator.go:106` vs line 156 — `buildSeverityFilter` (private) and `BuildSeverityFilter` (exported) have different severity rankings (critical=0 vs critical=4, "none" included vs excluded). Callers get different behavior.

### NH17. User Sync Worker Reports Success When All Users Fail
- `internal/server/workers/user_sync.go:47-66` — if every user fails `EnsureUser`, function returns nil (success). Periodic job reports success.

### NH18. MSI/MSIX/EXE Dry-Run Silently Does Real Install
- `internal/agent/patcher/msi.go:15`, `msix.go:15`, `exe_windows.go:24` — `dryRun` parameter silently ignored (blank identifier `_`). Requesting dry-run actually installs the package.

### NH19. Process Group Kill Missing in Executor
- `internal/agent/executor/executor.go:122-125` — `Process.Kill()` only kills direct child on Linux/macOS, not descendants. Script child processes become orphans on timeout.

### NH20. close(errCh) Can Race With Goroutine Writes — Server Panic
- `cmd/server/main.go:754-757` — `close(errCh)` called while 4 goroutines may still be running. If a goroutine writes to a closed channel, the program panics.

### NH21. 39 `as any` Casts in Frontend
- Concentrated in: useCompliance.ts (8), useIAMSettings.ts (4), useSettings.ts (2), useChannelByType.ts (3), useRoles.ts (2), useDashboard.ts (3), plus compliance pages, audit pages, deployment pages

### NH22. Hub "Rotate API Key" Button Has No onClick Handler
- `web-hub/src/pages/settings/APIWebhookSettings.tsx:217-233` — renders a clickable red button that does absolutely nothing

### NH23. 26+ Raw fetch() Calls Violating openapi-fetch Convention
- Tags (7), endpoints hooks (7), login hooks (4), auth hooks (2), command palette (3), audit export (1), endpoint export (1), health (1), agent binaries (1)

---

## NEW MEDIUM ISSUES

### NM1. RedHat Feed Only Fetches RHEL 9 — RHEL 7/8 not covered
### NM2. APT Resolver Only Covers 3 Ubuntu Versions (Noble, Jammy, Focal)
### NM3. APT Resolver Only Indexes amd64 — arm64 packages never resolved
### NM4. Hub Bootstrap Token Stored in Plaintext (not hashed like API keys)
### NM5. Dashboard references 'degraded' endpoint status not in CHECK constraint — always shows 0
### NM6. hub_sync_state query filters by 'disabled' status not in CHECK — no-op filter
### NM7. SyncStarted event defined but never emitted anywhere
### NM8. Compliance evaluations subqueries lack framework_id filter — cross-framework results
### NM9. ServerConfig.validate() doesn't validate HTTP timeouts (unlike HubConfig which does)
### NM10. gRPC NewGRPCServer silently ignores TLS config fields
### NM11. NewAuditEvent accepts empty tenantID without validation
### NM12. Default tenant lookup `LIMIT 1` without `ORDER BY` — non-deterministic
### NM13. CVE correlator discards original error — DB errors reported as "policy not found"
### NM14. PolicyScheduler constructor has no nil checks on dependencies — runtime panic risk
### NM15. Alert subscriber loads ALL tenant rules bypassing RLS for cache
### NM16. AllTopics() does linear scan on every Emit() — O(n) with 90+ topics
### NM17. ListAvailablePatches ignores endpointID parameter — blank identifier
### NM18. Heartbeat CPU measurement blocks 200ms on every heartbeat (Linux)
### NM19. WUA COM operations use inconsistent apartment models
### NM20. Agent outbox.Pending loads full payloads just to count items
### NM21. Agent settings update writes raw values without validation
### NM22. macOS CPU measurement uses context.Background() instead of passed context
### NM23. Windows IsRoot() always returns false (placeholder)
### NM24. Agent schema defined in TWO places (schema.sql + db.go migrations)
### NM25. Catalog Get handler makes 7 sequential DB queries (could parallelize)
### NM26. Compliance check conditions can't set value to 0 (treated as unset)
### NM27. MockSender defined in production code (notify/sender.go), not test file

---

## SUMMARY: ROUND 3 vs ROUND 2

| | Round 2 (v2 report) | Round 3 (this delta) | Combined Total |
|--|---------------------|---------------------|----------------|
| Critical | 32 | +9 | **41** |
| High | 50 | +23 | **73** |
| Medium | 54 | +27 | **81** |
| Total discrete issues | ~148 | +59 | **~207** |

Plus: 90 unused SQL queries, 39 `as any` casts (up from 13), 26+ raw fetch violations, fabricated SLA dashboard data.

---

## TOP 10 MOST DANGEROUS FINDINGS (New in Round 3)

1. **SQL injection** via string-formatted SET LOCAL in catalog_sync and workflow worker
2. **Tenant context leak** via session-level set_config in wave dispatcher and workflow worker
3. **Agent logs deleted on every restart** due to seed cleanup matching production log IDs
4. **compliance_scores GRANT missing UPDATE** — compliance scoring queries fail at runtime
5. **JWT signing key race condition** — concurrent logins can invalidate all active sessions
6. **SLA dashboard fabricated data** — client sees fake countdown timers
7. **Invite registration broken** — session cookie contains user ID not JWT
8. **MSIX command injection** — PowerShell injection via patch name
9. **MSRC cursor broken** — monthly updates within same year skipped
10. **Patch binaries never cleaned up** — agent disk fills over time
