# Fix Plan: 20 Issues to Production-Grade Platform

**Date**: 2026-04-10
**Source**: Audit reports 15, 16, 17 (~207 discrete findings)
**Verified**: 8 parallel agents verified every finding against actual source code (2026-04-10)
**Developers**: Rishab (Dev 3), Danish (Dev 4) — both working with Claude Code
**Reviewer**: Heramb (core dev, all PRs)
**Target**: 4 weeks to client-deployment-ready platform
**Branch**: all work branches off `dev-a`, merges back to `dev-a`

## GitHub Issues

| # | Issue | GitHub |
|---|-------|--------|
| 1 | Tenant Context Security | [#316](https://github.com/herambskanda/patchiq/issues/316) |
| 2 | Auth Security | [#317](https://github.com/herambskanda/patchiq/issues/317) |
| 3 | Agent Security & Patcher | [#318](https://github.com/herambskanda/patchiq/issues/318) |
| 4 | RBAC & Permissions | [#319](https://github.com/herambskanda/patchiq/issues/319) |
| 5 | Crash Prevention | [#320](https://github.com/herambskanda/patchiq/issues/320) |
| 6 | Worker Instantiation | [#321](https://github.com/herambskanda/patchiq/issues/321) |
| 7 | Agent Data Integrity | [#322](https://github.com/herambskanda/patchiq/issues/322) |
| 8 | Event Bus & Audit Trail | [#323](https://github.com/herambskanda/patchiq/issues/323) |
| 9 | DB Performance | [#324](https://github.com/herambskanda/patchiq/issues/324) |
| 10 | Query Optimization | [#325](https://github.com/herambskanda/patchiq/issues/325) |
| 11 | Runtime Scalability | [#326](https://github.com/herambskanda/patchiq/issues/326) |
| 12 | Deployment & Policy Engine | [#327](https://github.com/herambskanda/patchiq/issues/327) |
| 13 | CVE Data Flow | [#328](https://github.com/herambskanda/patchiq/issues/328) |
| 14 | Hub Feed Pipeline | [#329](https://github.com/herambskanda/patchiq/issues/329) |
| 15 | Hub API Hardening | [#330](https://github.com/herambskanda/patchiq/issues/330) |
| 16 | Integration Hardening | [#331](https://github.com/herambskanda/patchiq/issues/331) |
| 17 | Frontend Critical | [#332](https://github.com/herambskanda/patchiq/issues/332) |
| 18 | Frontend Cleanup | [#333](https://github.com/herambskanda/patchiq/issues/333) |
| 19 | Production Readiness | [#334](https://github.com/herambskanda/patchiq/issues/334) |
| 20 | Testing & Dead Code | [#335](https://github.com/herambskanda/patchiq/issues/335) |

## Verification Results Summary (8 agents, 2026-04-10)

**10 FALSE POSITIVES removed from original audit:**
1. E4 (notification pagination panic) — guard is correct, not a bug
2. NC8 (invite cookie raw user ID) — code correctly uses Zitadel JWT
3. NC2 on wave_dispatcher.go — correctly delegates to txFactory with `true`
4. NM21 (agent settings no validation) — comprehensive validation already exists
5. H-F4 (wave rollback missing) — rollback IS implemented at wave_dispatcher.go:396-412
6. H-E5 (cron at exec time only) — validated at creation/update time
7. NM13 (CVE correlator "policy not found") — error message doesn't exist
8. S4 (forgeable license keys) — RSA signing infrastructure exists, just not wired
9. NM6 (hub_sync_state filter) — not confirmed
10. M-DC1 partial — SeverityPills and StatsStrip files don't exist (3 orphaned, not 5)

**Key count corrections:**
- Unbounded queries: 20+ → **90+** (70+ genuinely dangerous)
- Correlated subqueries in ListPatchesFiltered: 5 → **9**
- `emitEvent` "system" actor calls: 15+ → **44 across 15 files**
- `as any` casts: 39 → **43**
- Raw fetch() calls: 26+ → **45** (web-hub/web-agent have ZERO openapi-fetch)
- Unused SQL functions: 90 → **73**
- Workflow routes missing RBAC: 6 → **8**

**Key NEW findings added to issues:**
- SQL injection in `custom_compliance.go:283,543` (2 new locations)
- 7 additional transactions missing tenant context (policies ×5, roles ×2)
- Server `login.go:45` has same JWT injection as hub
- Dev auth bypass via X-Tenant-ID/X-User-ID headers
- Hub has ZERO RBAC anywhere (entire router unprotected)
- Cross-tenant queries without tenant filter in deployments.sql:155,158
- HubSyncAPIHandler struct lacks eventBus field entirely
- web-hub and web-agent have NO openapi-fetch integration at all

---

## How to Use This Plan

Each issue below is a self-contained work package. The interns' job is:
1. Read the issue carefully before starting
2. Let Claude Code implement the fix
3. **Verify everything yourself** — Claude can hallucinate success
4. While fixing, hunt for related issues in the same files
5. Create PR, get Heramb's review

**The intern is the quality gate, not Claude.**

---

## Standard Workflow (copy into every issue)

### Before Starting
```bash
git checkout dev-a && git pull origin dev-a
git worktree add .worktrees/fix/issue-short-name dev-a
cd .worktrees/fix/issue-short-name
```

### Implementation (with Claude Code)
1. Open Claude Code in the worktree directory
2. Paste the full issue description to Claude
3. Tell Claude: **"Follow the Fix Track workflow: write a failing test first, then fix, then verify. Use `make test` and `make lint` after every change."**
4. Watch Claude work. Ask questions if something looks wrong.

### Verification (YOU do this, not Claude)
5. **Read every changed file** — open the diff in your editor, read line by line
6. **Run tests yourself**: `make test` — read the FULL output, don't just check exit code
7. **Run linter yourself**: `make lint` — zero warnings
8. **Check each item** in the issue's Verification Checklist — these are specific things to manually confirm
9. **Run the grep commands** in "Related Patterns to Check" — fix anything you find
10. **If the issue involves SQL**: connect to the database and run the queries manually to verify they work

### Common Claude Code Mistakes — Watch For These
- **Claims tests pass when they don't** — ALWAYS run `make test` yourself and read output
- **Fixes the symptom but not root cause** — verify against the "Problem" description
- **Adds `//nolint` or `_ = err`** to silence errors instead of fixing them
- **Changes test expectations** to match buggy behavior instead of fixing the bug
- **Creates new files** when editing existing ones is correct
- **Adds TODO/FIXME without issue reference** — violates CLAUDE.md anti-slop rules
- **Uses `fmt.Println` or `log.Println`** — must use `slog`
- **Introduces unused imports** — check `make lint`
- **Skips the failing test step** — insist on TDD

### Submitting
11. Run `/review-pr` in Claude Code — fix any Critical/Important findings
12. Run `/commit-push-pr` to create PR targeting `dev-a`
13. In PR description, list which audit findings (e.g., NC1, NH3) are addressed
14. Tag @heramb for review

---

## Dependency Graph

```
Issues with NO dependencies (can start immediately):
  1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 14, 15, 17

Dependencies:
  12 (Deployment Engine) ← soft depends on 6 (Workers wired up)
  13 (CVE Data Flow)     ← depends on 1 (Tenant context safe for catalog_sync.go)
  16 (Integration)       ← depends on 1 + 6 (Tenant context + workers for enrollment)
  18 (Frontend Cleanup)  ← depends on 17 (Frontend Critical must land first)
  20 (Testing & Cleanup) ← depends on all feature issues (1-19)

Critical path: 1 → 13 → 20
              6 → 12, 16 → 20
```

## 4-Week Schedule (2 Interns)

```
WEEK 1 — SECURITY & STABILITY
┌─────────────────────────────────────────────────────────────────┐
│ Rishab                        │ Danish                          │
│ Issue  1: Tenant Context  4-6h│ Issue  2: Auth Security    6-8h │
│ Issue  6: Worker Wiring   6-8h│ Issue  4: RBAC & Perms    4-6h │
│ Issue  5: Crash Prevention 6-8h│ Issue  3: Agent Security  8-10h│
│ Issue  7: Agent Data      6-8h│ Issue  9: DB Performance  4-6h │
├─────────────────────────────────────────────────────────────────┤
│ ★ CHECKPOINT 1: Merge all Wave 1 PRs to dev-a, both rebase     │
│   Gate: `make test && make lint` passes on merged dev-a         │
│   Heramb reviews all 8 PRs before moving to Week 2              │
└─────────────────────────────────────────────────────────────────┘

WEEK 2 — SCALABILITY & DATA INTEGRITY
┌─────────────────────────────────────────────────────────────────┐
│ Rishab                        │ Danish                          │
│ Issue 12: Deploy Engine  8-10h│ Issue  8: Event Bus      10-14h │
│ Issue 11: Runtime Scale   6-8h│ Issue 15: Hub API         6-8h │
│ Issue 10: Query Opt (start)   │ Issue 14: Hub Feeds (start)     │
├─────────────────────────────────────────────────────────────────┤
│ ★ CHECKPOINT 2: Merge all Wave 2 PRs to dev-a, both rebase     │
│   Gate: `make test && make lint` passes on merged dev-a         │
└─────────────────────────────────────────────────────────────────┘

WEEK 3 — INTEGRATION & HUB
┌─────────────────────────────────────────────────────────────────┐
│ Rishab                        │ Danish                          │
│ Issue 10: Query Opt (finish)  │ Issue 14: Hub Feeds (finish)    │
│ Issue 17: Frontend Critical   │ Issue 13: CVE Data Flow  12-16h │
│           8-10h               │                                 │
├─────────────────────────────────────────────────────────────────┤
│ ★ CHECKPOINT 3: Merge all Wave 3 PRs to dev-a, both rebase     │
│   Gate: `make test && make lint` passes on merged dev-a         │
│   Gate: Manual E2E test of enrollment → deploy → compliance     │
└─────────────────────────────────────────────────────────────────┘

WEEK 4 — FRONTEND, PRODUCTION, TESTING
┌─────────────────────────────────────────────────────────────────┐
│ Rishab                        │ Danish                          │
│ Issue 18: Frontend Cleanup    │ Issue 16: Integration   14-18h  │
│           8-12h               │                                 │
│ Issue 20: Testing (split)     │ Issue 19: Production    12-16h  │
│                               │ Issue 20: Testing (split)       │
├─────────────────────────────────────────────────────────────────┤
│ ★ FINAL CHECKPOINT: Merge everything, full regression           │
│   Gate: `make ci` passes                                        │
│   Gate: Full manual E2E with 3+ endpoints                       │
│   Gate: Heramb signs off                                        │
└─────────────────────────────────────────────────────────────────┘
```

**Total estimate: ~180-220 hours (~90-110h per intern over 4 weeks)**

---

## WAVE 1: SECURITY & STABILITY (Issues 1-7, Week 1)

No dependencies between these — fully parallel.

---

### Issue 1: Tenant Context Security — SQL Injection & Session-Level Leaks

**Priority**: P0-CRITICAL | **Est**: 4-6 hours | **Assign**: Rishab
**Dependencies**: None (but do this FIRST — Issue 13 depends on it)
**Findings**: NC1, NC2, NH13

#### Problem

Three files use unsafe patterns to set the PostgreSQL tenant context, creating SQL injection risk and cross-tenant data leaks:

1. **SQL Injection** — `fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantIDStr)` allows injection if tenantIDStr contains a single quote. Two locations:
   - `internal/server/workers/catalog_sync.go:267`
   - `internal/server/workflow/worker.go:83`

2. **Session-level tenant leak** — `set_config('app.current_tenant_id', $1, false)` sets at SESSION level instead of transaction-local. Tenant context persists on the pooled connection after the job completes, leaking to subsequent requests from other tenants. Two locations:
   - `internal/server/deployment/wave_dispatcher.go:83`
   - `internal/server/workflow/worker.go:83`

3. **Missing tenant context** — `internal/server/api/v1/groups.go:341` starts a transaction without `SET LOCAL app.current_tenant_id`, bypassing RLS.

#### Files to Change
- `internal/server/workers/catalog_sync.go` — line ~267
- `internal/server/workflow/worker.go` — line ~83
- `internal/server/deployment/wave_dispatcher.go` — line ~83
- `internal/server/api/v1/groups.go` — line ~341

#### Fix Approach
1. Replace all `fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", ...)` with parameterized `SELECT set_config('app.current_tenant_id', $1, true)` — the `true` parameter makes it transaction-local.
2. Change all `set_config(..., false)` to `set_config(..., true)`.
3. Add `SET LOCAL app.current_tenant_id` to the groups.go transaction.
4. Search the entire codebase for any other instances of this pattern.

#### Verification Checklist
- [ ] `grep -rn "fmt.Sprintf.*SET LOCAL" internal/` returns zero results
- [ ] `grep -rn "set_config.*false" internal/` returns zero results (or only non-tenant contexts)
- [ ] `grep -rn "app.current_tenant_id" internal/` — every instance uses `set_config($1, true)` or `SET LOCAL`
- [ ] The groups.go transaction includes tenant context setting
- [ ] `make test` passes
- [ ] `make lint` passes

#### Related Patterns to Check
While in these files, also check:
```bash
# Any other unsafe SQL string formatting
grep -rn 'fmt.Sprintf.*SET\|fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE' internal/
# Any other transactions without tenant context
grep -rn 'BeginTx\|Begin(' internal/server/ | head -30
```

#### Context for Verification
- The **correct pattern** is already used elsewhere in the codebase — look at `internal/server/store/` for examples of `set_config($1, true)`
- `true` = transaction-local (safe). `false` = session-level (dangerous with connection pooling)
- With pgx connection pooling, session-level settings persist after the connection returns to the pool

---

### Issue 2: Auth Security — JWT Race Condition, Invite Cookie, Hub JWT Injection

**Priority**: P0-CRITICAL | **Est**: 6-8 hours | **Assign**: Danish
**Dependencies**: None
**Findings**: NC3, NC8, S1

#### Problem

Three auth-related security bugs across Server and Hub:

1. **JWT Signing Key Race Condition** — `internal/server/auth/jwt.go:100-101` has `var LocalSigningKey []byte` as mutable global state. `InitSigningKey()` in `login.go:143` is called on every login request and checks `len(c.SigningKey) == 0` without synchronization. Two concurrent logins can both see length 0, both generate keys, one overwrites the other — invalidating all JWTs signed with the first key.

2. **Invite Registration Invalid Cookie** — `internal/server/auth/invite.go:254-259` stores a raw user ID string (not a JWT) as the session cookie after Zitadel auth. JWT middleware rejects this on subsequent requests. Users who register via invite link cannot authenticate.

3. **Hub JWT Claim Injection** — `internal/hub/auth/session.go:39` uses `fmt.Sprintf` with user-controlled email/name to build JWT claims JSON. Attacker can inject arbitrary claims via crafted fields (e.g., `"email": "x\", \"role\": \"admin"}`).

#### Files to Change
- `internal/server/auth/jwt.go` — lines ~100-101 (signing key)
- `internal/server/auth/login.go` — line ~143 (InitSigningKey call)
- `internal/server/auth/invite.go` — lines ~254-259 (session cookie)
- `internal/hub/auth/session.go` — line ~39 (JWT claims)

#### Fix Approach
1. **JWT Race**: Use `sync.Once` for signing key initialization, or initialize at startup (not per-request). Remove the global mutable variable — inject via config or singleton.
2. **Invite Cookie**: After Zitadel auth, generate a proper JWT (same as login flow) and set that as the session cookie, not the raw user ID.
3. **Hub JWT**: Replace `fmt.Sprintf` JSON construction with `json.Marshal` on a typed claims struct.

#### Verification Checklist
- [ ] `jwt.go` no longer has `var LocalSigningKey []byte` as a bare global (should be behind sync.Once or initialized once at startup)
- [ ] `login.go` does NOT call `InitSigningKey()` per-request
- [ ] `invite.go` sets a JWT (not raw user ID) as the session cookie — verify by reading the cookie-setting code
- [ ] `session.go` uses `json.Marshal` (not `fmt.Sprintf`) for JWT claims
- [ ] Write a test that calls `InitSigningKey()` concurrently (use goroutines + sync.WaitGroup) — key should be stable
- [ ] `make test` passes
- [ ] `make lint` passes

#### Related Patterns to Check
```bash
# Any other fmt.Sprintf used to build JSON
grep -rn 'fmt.Sprintf.*{.*"' internal/ --include='*.go'
# Any other global mutable state in auth
grep -rn 'var.*\[\]byte\|var.*string' internal/server/auth/ internal/hub/auth/
# Any other places setting session cookies
grep -rn 'SetCookie\|Set-Cookie\|cookie' internal/server/auth/ internal/hub/auth/
```

#### Context for Verification
- For the race condition: run `go test -race ./internal/server/auth/...` — the race detector should NOT flag anything after the fix
- For the invite cookie: the login flow already generates JWTs correctly — the invite flow should use the same mechanism
- For the hub JWT: `json.Marshal` is already used throughout the codebase for JSON — search for examples

---

### Issue 3: Agent Security — Command Injection, Auth, Dry-Run, Process Kill, Temp Cleanup

**Priority**: P0-CRITICAL | **Est**: 8-10 hours | **Assign**: Danish
**Dependencies**: None
**Findings**: NC4, NC9, S3, NH18, NH19

#### Problem

Five security and correctness issues in the agent:

1. **MSIX Command Injection** — `internal/agent/patcher/msix.go:16` uses `fmt.Sprintf("Add-AppxPackage -Path '%s'", pkg.Name)`. A single quote in `pkg.Name` allows PowerShell command injection.

2. **Temp File Leak** — `internal/agent/patcher/patcher.go:176-195` downloads patch binaries to temp files but never cleans them up. No `defer os.Remove()`, no cleanup goroutine, no retention policy. Disk fills over time.

3. **Agent API Zero Auth** — `internal/agent/api/handler.go:26-62` has no authentication middleware on any endpoint. Any network client can read/modify agent settings, trigger scans, view patch history.

4. **Dry-Run Silently Installs** — `internal/agent/patcher/msi.go:15`, `msix.go:15`, `exe_windows.go:24` accept a `dryRun` parameter but assign it to blank identifier `_`. Requesting dry-run actually installs the package.

5. **Process Group Kill Missing** — `internal/agent/executor/executor.go:122-125` uses `Process.Kill()` which only kills the direct child on Linux/macOS. Script child processes become orphans on timeout.

#### Files to Change
- `internal/agent/patcher/msix.go` — line ~16 (command injection + dry-run)
- `internal/agent/patcher/msi.go` — line ~15 (dry-run)
- `internal/agent/patcher/exe_windows.go` — line ~24 (dry-run)
- `internal/agent/patcher/patcher.go` — lines ~176-195 (temp cleanup)
- `internal/agent/api/handler.go` — lines ~26-62 (auth)
- `internal/agent/executor/executor.go` — lines ~122-125 (process group)

#### Fix Approach
1. **Command Injection**: Never interpolate user input into shell commands. Use PowerShell argument arrays or properly escape. Same pattern for all patcher files.
2. **Temp Cleanup**: Add `defer os.Remove(tempFile)` after download, or implement a cleanup function that runs on agent startup to remove old temp files.
3. **Agent Auth**: Add a shared-secret middleware. The enrollment token or a derived key can serve as the auth token. Agent config already has a `server_url` field — add an `api_key` field.
4. **Dry-Run**: Replace `_ = dryRun` with actual conditional logic: if dryRun, log what would be done and return without executing.
5. **Process Group**: Use `syscall.Setpgid` when starting the process and `syscall.Kill(-pid, syscall.SIGKILL)` to kill the process group. This is Linux/macOS specific — use build tags.

#### Verification Checklist
- [ ] `grep -rn 'fmt.Sprintf.*AppxPackage\|fmt.Sprintf.*msiexec' internal/agent/` returns zero results (no shell interpolation)
- [ ] `grep -rn '_ = dryRun\|_ = dry' internal/agent/` returns zero results
- [ ] Patcher download code has `defer os.Remove` or equivalent cleanup
- [ ] Agent API has auth middleware — `grep -rn 'middleware\|Auth\|auth' internal/agent/api/handler.go`
- [ ] Executor uses process group kill — look for `Setpgid` or `SysProcAttr` in executor.go
- [ ] All new code has tests (especially dry-run behavior)
- [ ] `make test` passes
- [ ] `make lint` passes

#### Related Patterns to Check
```bash
# Any other shell command construction in agent
grep -rn 'fmt.Sprintf.*exec\|fmt.Sprintf.*cmd\|fmt.Sprintf.*powershell' internal/agent/
# Any other blank identifier assignments hiding bugs
grep -rn '_ = ' internal/agent/ | grep -v '_test.go'
# Any other temp file creation without cleanup
grep -rn 'os.CreateTemp\|ioutil.TempFile\|os.MkdirTemp' internal/agent/
```

#### Context for Verification
- For command injection: the safest approach is `exec.Command("powershell", "-Command", "Add-AppxPackage", "-Path", pkg.Name)` — PowerShell handles escaping when arguments are passed as separate strings
- For dry-run: verify by reading the patcher code that when dryRun=true, NO actual installation commands execute
- Agent auth: this is the local agent API (typically on localhost:8090), so even basic shared-secret auth is sufficient

---

### Issue 4: RBAC & Permission Gaps

**Priority**: P0-CRITICAL | **Est**: 4-6 hours | **Assign**: Danish
**Dependencies**: None
**Findings**: S2, NC6, NH5

#### Problem

Three access control gaps across all three platforms:

1. **Workflow Routes Missing RBAC** — `internal/server/api/router.go:330-345` — all workflow CRUD routes (List, Create, Get, Update, Delete, Publish) have no `.With(rp(...))` RBAC middleware. Any authenticated user can modify workflows.

2. **compliance_scores Missing GRANT UPDATE** — `internal/server/store/migrations/019_compliance.sql:111-113` — GRANT only gives SELECT, INSERT, DELETE to `patchiq_app` role. But queries `UpdateEndpointScoresForRun` and `UpdateEndpointScoreByID` use UPDATE. These queries fail at runtime when connected as `patchiq_app`.

3. **Hub Login Always Returns admin** — `internal/hub/auth/login.go:133-139` and `Me` handler (~line 167) hardcode `Role: "admin"` regardless of actual user role. No RBAC integration on hub side.

#### Files to Change
- `internal/server/api/router.go` — lines ~330-345 (add RBAC middleware)
- `internal/server/store/migrations/` — new migration to add GRANT UPDATE on compliance_scores
- `internal/hub/auth/login.go` — lines ~133-139, ~167 (role from DB/Zitadel, not hardcoded)

#### Fix Approach
1. **Workflow RBAC**: Look at how other resource routes (e.g., policies, deployments) apply RBAC middleware. Copy that pattern for workflow routes.
2. **GRANT UPDATE**: Create a new migration file (next sequential number) that runs `GRANT UPDATE ON compliance_scores TO patchiq_app;`
3. **Hub Role**: If Zitadel integration exists on hub side, fetch role from there. If not, at minimum don't hardcode "admin" — use the actual role from the user record.

#### Verification Checklist
- [ ] `router.go` workflow routes all have `.With(rp(...))` middleware — compare visually with other resource routes
- [ ] New migration exists and runs: `make migrate` succeeds
- [ ] Connect to DB as `patchiq_app` and run `UPDATE compliance_scores SET ... WHERE ...` — should NOT fail with permission denied
- [ ] Hub `/api/v1/auth/me` returns actual role, not hardcoded "admin" — `curl` the endpoint
- [ ] `make test` passes
- [ ] `make lint` passes

#### Related Patterns to Check
```bash
# Any other routes missing RBAC middleware
grep -n 'r.Route\|r.Get\|r.Post\|r.Put\|r.Delete' internal/server/api/router.go | grep -v 'rp('
# Any other GRANT statements missing permissions
grep -rn 'GRANT.*TO.*patchiq_app' internal/server/store/migrations/ | grep -v UPDATE
# Any other hardcoded roles
grep -rn '"admin"\|"viewer"\|"editor"' internal/hub/auth/
```

#### Context for Verification
- The RBAC middleware pattern is `r.With(rp("resource:action"))` — look at `policies` or `deployments` routes for examples
- For the GRANT: migrations are sequential. Find the highest-numbered migration and create the next one
- **IMPORTANT**: Do NOT modify existing migrations — always create new ones. Existing migrations may have already run in production.

---

### Issue 5: Crash Prevention — Panics, Nil Dereferences, Channel Races

**Priority**: P0-CRITICAL | **Est**: 6-8 hours | **Assign**: Rishab
**Dependencies**: None
**Findings**: E1, E2, E3, E4, NH20

#### Problem

Five paths that cause the server to crash (panic):

1. **MustTenantID() Panics** — `internal/shared/tenant/context.go:24-29` — panics if tenant ID not in context. Used in 28+ handlers. If middleware is misconfigured on any route, entire server crashes.

2. **Workflow Worker Nil Dereference** — `internal/server/workflow/worker.go:404` — `ne, _ := q.GetNodeExecutionByNodeID(...)` discards error. Line 409 accesses `ne.ID` without nil check → panic if query fails.

3. **Workflow Worker Ignores DB Updates** — `internal/server/workflow/worker.go:394, 409` — `_, _ = q.UpdateNodeExecution(...)` silently ignores errors. Execution state not recorded.

4. **Notification Pagination Empty Slice** — `internal/server/api/v1/notifications.go:753` — `history[len(history)-1].ID` panics if `history` is empty when condition passes.

5. **errCh Close Race** — `cmd/server/main.go:754-757` — `close(errCh)` called while 4 goroutines may still write to it. Write to closed channel = panic.

#### Files to Change
- `internal/shared/tenant/context.go` — lines ~24-29
- `internal/server/workflow/worker.go` — lines ~394, 404, 409
- `internal/server/api/v1/notifications.go` — line ~753
- `cmd/server/main.go` — lines ~754-757

#### Fix Approach
1. **MustTenantID**: Create a `TenantIDFromContext(ctx) (string, error)` that returns an error instead of panicking. Update callers gradually — or keep `MustTenantID` but wrap it in a recovery middleware that converts panics to 500 responses with proper error messages.
2. **Workflow nil deref**: Check the error from `GetNodeExecutionByNodeID`. If error, log and return. Don't access `ne.ID` without nil check.
3. **Workflow DB updates**: Check errors from `UpdateNodeExecution`. If error, log with `slog.Error`.
4. **Notification pagination**: Add `len(history) > 0` check before accessing last element.
5. **errCh race**: Use `sync.WaitGroup` to wait for all goroutines to finish before closing the channel, OR don't close the channel at all (let GC handle it), OR use a select with a done channel.

#### Verification Checklist
- [ ] `tenant/context.go` either has a non-panicking alternative or `MustTenantID` is wrapped in recovery
- [ ] `workflow/worker.go` — `grep -n '_ =' internal/server/workflow/worker.go` shows NO silently discarded errors
- [ ] `notifications.go` has a `len > 0` check before accessing the last element
- [ ] `cmd/server/main.go` does NOT have `close(errCh)` without waiting for goroutines
- [ ] All error paths log with `slog` (not fmt.Println)
- [ ] `go test -race ./internal/server/...` passes (race detector)
- [ ] `make test` passes

#### Related Patterns to Check
```bash
# Other MustXxx functions that panic
grep -rn 'func Must' internal/
# Other discarded errors in workflow
grep -rn '_ =' internal/server/workflow/ | grep -v '_test.go'
# Other channel close patterns
grep -rn 'close(' cmd/ | grep -v '_test.go'
# Other empty-slice access without length check
grep -rn '\[len(' internal/server/api/
```

#### Context for Verification
- `MustTenantID` is in `internal/shared/` which is a **protected file** — changes here need extra scrutiny from Heramb
- The errCh fix in main.go: look at how other Go programs handle graceful shutdown with multiple goroutines — `errgroup` from `golang.org/x/sync` is a good pattern
- Run `go test -race` specifically — this catches the channel race and any goroutine data races

---

### Issue 6: Worker Instantiation & Job Registration

**Priority**: P0-CRITICAL | **Est**: 6-8 hours | **Assign**: Rishab
**Dependencies**: None (but do this early — Issues 12, 16 soft-depend on it)
**Findings**: I3, I4, F3, NH17, NM14

#### Problem

Critical background workers are fully implemented but never instantiated, making major features dead code:

1. **Nil Workers Passed to River** — `cmd/server/main.go:412` — `RegisterWorkers()` receives TWO nil pointers: `userSyncWorker` and `policySchedulerWorker` are never created. `workers/registry.go:33-47` calls `river.AddWorker(workers, nil)` which will panic.

2. **Policy Scheduler Dead** — `internal/server/policy/scheduler.go` is fully implemented but never instantiated in `main.go`. No periodic River job registered. Automatic policy-based deployments never execute.

3. **User Sync Dead** — `internal/server/workers/user_sync.go` is fully implemented but never instantiated. No periodic job registered. Users created in Zitadel never appear in PatchIQ.

4. **User Sync Reports False Success** — `internal/server/workers/user_sync.go:47-66` — even when implemented, if every user fails `EnsureUser`, the function returns nil (success).

5. **Policy Scheduler Nil Checks** — `internal/server/policy/scheduler.go` constructor has no nil checks on dependencies — will panic at runtime if any dependency is nil.

#### Files to Change
- `cmd/server/main.go` — instantiate workers, register periodic jobs
- `internal/server/workers/registry.go` — handle nil workers safely
- `internal/server/workers/user_sync.go` — fix false success reporting
- `internal/server/policy/scheduler.go` — add nil checks in constructor

#### Fix Approach
1. In `main.go`, find where other workers are created (e.g., `catalogSyncWorker`). Follow the same pattern to create `userSyncWorker` and `policySchedulerWorker` with their required dependencies.
2. Register periodic River jobs for both workers (user sync: e.g., every 15 minutes; policy scheduler: e.g., every 5 minutes).
3. In `registry.go`, add nil checks before `river.AddWorker` — skip nil workers with a warning log.
4. In `user_sync.go`, track individual failures and return an error if ALL users fail.
5. In `policy/scheduler.go`, validate constructor arguments.

#### Verification Checklist
- [ ] `main.go` creates both `userSyncWorker` and `policySchedulerWorker` with non-nil dependencies
- [ ] Both workers are registered in `RegisterWorkers()` — read the code to confirm
- [ ] Periodic jobs registered — search for `river.PeriodicJob` or equivalent for both workers
- [ ] `registry.go` handles nil workers gracefully (log warning, skip)
- [ ] `user_sync.go` returns error when all users fail — read the error-tracking logic
- [ ] `policy/scheduler.go` constructor validates inputs
- [ ] `make test` passes — especially test the worker registration
- [ ] Start the server and check logs: workers should log their initialization

#### Related Patterns to Check
```bash
# Any other nil workers or unregistered workers
grep -rn 'nil.*Worker\|Worker.*nil' cmd/server/main.go
# Any other "return nil" masking all-failures
grep -rn 'return nil' internal/server/workers/ | grep -v '_test.go'
# Check what periodic jobs ARE registered
grep -rn 'PeriodicJob\|river.Add\|AddWorker' cmd/server/main.go internal/server/workers/
```

#### Context for Verification
- Look at how `catalogSyncWorker` is created and registered in `main.go` — use the exact same pattern
- After fixing, start the dev server (`make dev`) and watch logs for worker initialization messages
- The policy scheduler is critical for auto-deploy policies — without it, policies with auto-deploy enabled do nothing
- User sync is critical for Zitadel integration — without it, users added in Zitadel never appear in PatchIQ

---

### Issue 7: Agent Data Integrity — Log Deletion, Heartbeat Blocking, Schema Duplication

**Priority**: P0-CRITICAL | **Est**: 6-8 hours | **Assign**: Rishab
**Dependencies**: None
**Findings**: NC5, NM18, NM19, NM20, NM21, NM22, NM23, NM24

#### Problem

Multiple data integrity and correctness issues in the agent:

1. **All Logs Deleted on Restart** — `cmd/agent/main.go:163-168` — `DELETE FROM agent_logs WHERE id LIKE 'l%'` is intended to clean seed data (IDs like `l001`), but `generateLogID()` in `sqlite_log.go:113` produces IDs starting with `"log-"` which also match `LIKE 'l%'`. Every restart deletes ALL operational logs.

2. **Heartbeat Blocks 200ms** — `NM18` — CPU measurement blocks 200ms on every heartbeat (Linux). This adds latency to every heartbeat cycle.

3. **macOS CPU Wrong Context** — `NM22` — macOS CPU measurement uses `context.Background()` instead of the passed context, so cancellation doesn't work.

4. **Windows IsRoot Placeholder** — `NM23` — `IsRoot()` always returns false (placeholder), affecting privilege checks.

5. **Agent Schema in Two Places** — `NM24` — Schema defined in both `schema.sql` and `db.go` migrations. They can drift.

6. **Outbox Loads Full Payloads to Count** — `NM20` — `outbox.Pending` loads full payloads just to count items.

7. **Settings Written Without Validation** — `NM21` — Agent settings update writes raw values without validation.

#### Files to Change
- `cmd/agent/main.go` — lines ~163-168 (fix log deletion pattern)
- `internal/agent/store/sqlite_log.go` — verify ID generation pattern
- `internal/agent/` — heartbeat CPU measurement
- `internal/agent/` — macOS CPU context, Windows IsRoot
- `internal/agent/store/` — schema consolidation
- `internal/agent/comms/` — outbox counting

#### Fix Approach
1. **Log deletion**: Change `LIKE 'l%'` to a more specific pattern that only matches seed IDs. E.g., `WHERE id IN ('l001', 'l002', ...)` or `WHERE id LIKE 'l___'` (exactly 4 chars). Better yet: use a different prefix for seed data, e.g., `seed-`.
2. **Heartbeat CPU**: Make CPU measurement async or cache the result (measure in background goroutine, report cached value in heartbeat).
3. **macOS context**: Pass the actual context parameter instead of `context.Background()`.
4. **Windows IsRoot**: Implement using `windows.GetCurrentProcessToken()` or `golang.org/x/sys/windows`.
5. **Schema**: Consolidate to one source of truth (preferably `schema.sql`), have `db.go` read from it.
6. **Outbox**: Use `SELECT COUNT(*) FROM outbox WHERE status = 'pending'` instead of loading all rows.
7. **Settings**: Add validation before writing (type checks, range checks for known settings).

#### Verification Checklist
- [ ] `grep -n "LIKE 'l%" cmd/agent/main.go` — pattern should NOT match production `log-*` IDs
- [ ] Start agent, create some logs, restart agent — logs should STILL BE THERE
- [ ] Heartbeat CPU measurement no longer blocks the calling goroutine
- [ ] `make test` passes
- [ ] `make lint` passes

#### Related Patterns to Check
```bash
# Any other seed cleanup that could hit production data
grep -rn "DELETE.*LIKE\|DELETE.*WHERE id" cmd/agent/ internal/agent/
# Any other context.Background() where a real context should be used
grep -rn 'context.Background()' internal/agent/ | grep -v '_test.go'
# Any other placeholder implementations
grep -rn 'return false\|return true\|return nil' internal/agent/ | grep -v '_test.go' | head -20
```

#### Context for Verification
- **CRITICAL**: The log deletion bug is the most dangerous. To verify: read the `generateLogID()` function, confirm the format, then confirm the DELETE pattern does NOT match it.
- For the schema duplication: compare `schema.sql` with `db.go` — they should be identical. After consolidation, only one should be the source of truth.

---

## WAVE 2: SCALABILITY & DATA INTEGRITY (Issues 8-12, Week 2)

### CHECKPOINT GATE: All Wave 1 PRs must be merged before starting Wave 2

---

### Issue 8: Event Bus & Audit Trail Integrity

**Priority**: HIGH | **Est**: 10-14 hours | **Assign**: Danish
**Dependencies**: None (but benefits from Wave 1 being merged)
**Findings**: D1, D2, D3, D4, NH11, NH7, NM7, NM11, NM12, NM15, NM16

#### Problem

The event bus and audit trail have multiple gaps that undermine data integrity and observability:

1. **PolicyAutoDeployed Missing from AllTopics** — `internal/server/events/topics.go:38` — topic defined but not in `AllTopics()`. Emit() silently fails.

2. **4+ Write Ops Emit No Events** — `notifications.go:584-634` (UpdatePreferences), `hub_sync.go:161-204` (UpdateConfig), `patches.go:594` (QuickDeploy), `patches.go:790` (DeployCritical) — all write to DB without emitting domain events.

3. **Duplicate UserID Context Keys** — `internal/shared/user/context.go` uses `userCtxKey struct{}`, `internal/shared/otel/context.go` uses `userIDKey struct{}`. Different keys → user IDs missing from all traces and structured logs.

4. **emitEvent Always Uses "system" Actor** — `internal/server/api/v1/helpers.go:68` — 15+ handlers (Groups, Tags, TagRules, Schedules, Policies, NotificationChannels, Settings, HubSync) all produce audit entries that say "system" performed the action instead of the actual user.

5. **Sync Events with Empty TenantID** — `internal/hub/api/v1/sync.go:226, 282, 307` — passes empty string for tenant ID.

6. **SyncStarted Event Never Emitted** — `NM7` — defined but never used.

7. **NewAuditEvent Accepts Empty TenantID** — `NM11` — no validation.

8. **Default Tenant LIMIT 1 Without ORDER BY** — `NM12` — non-deterministic.

9. **AllTopics() Linear Scan** — `NM16` — O(n) with 90+ topics on every Emit().

10. **Alert Subscriber Loads ALL Tenant Rules** — `NM15` — bypasses RLS for cache.

#### Files to Change
- `internal/server/events/topics.go` — add missing topic, optimize AllTopics()
- `internal/server/api/v1/helpers.go` — fix emitEvent to use actual user ID
- `internal/server/api/v1/notifications.go` — add event emission
- `internal/server/api/v1/hub_sync.go` — add event emission
- `internal/server/api/v1/patches.go` — add event emission for QuickDeploy, DeployCritical
- `internal/shared/user/context.go` — consolidate user context key
- `internal/shared/otel/context.go` — use shared user context key
- `internal/hub/api/v1/sync.go` — fix empty tenant ID
- `internal/shared/domain/` — validate tenant ID in NewAuditEvent

#### Fix Approach
1. Add `PolicyAutoDeployed` to `AllTopics()`. Check if any other topics are missing by comparing the topic constants list with `AllTopics()`.
2. Add `emitEventWithActor` calls (using actual user ID from context) to all write handlers that currently use `emitEvent`.
3. Consolidate user ID context to a single key — use the one from `shared/user/` package, update otel package to import from there.
4. Fix `emitEvent` in `helpers.go` to extract user ID from context (using the user middleware's context key).
5. Fix sync.go to pass actual tenant ID.
6. Add `ORDER BY id` to default tenant query.
7. Convert AllTopics() to a map for O(1) lookup.

#### Verification Checklist
- [ ] `AllTopics()` returns ALL topic constants — write a test that compares the list
- [ ] `grep -rn 'emitEvent(' internal/server/api/v1/ | grep -v 'emitEventWithActor'` — should be zero (all should use the actor-aware version)
- [ ] `grep -n 'userCtxKey\|userIDKey' internal/shared/` — should be ONE canonical key
- [ ] QuickDeploy and DeployCritical in patches.go emit events — read the code
- [ ] `sync.go` passes non-empty tenant ID
- [ ] `make test` passes
- [ ] `make lint` passes

#### Related Patterns to Check
```bash
# Any other write operations missing events (look for DB writes without Emit)
grep -rn 'tx.Commit\|Exec(ctx' internal/server/api/v1/ | head -20
# Compare that each has a corresponding Emit call nearby
```

#### Context for Verification
- The domain event invariant is: **every write operation MUST emit a domain event** (from CLAUDE.md)
- `emitEventWithActor` already exists and is used by Roles, UserRoles, Registrations, Alerts — look at those for the correct pattern
- `helpers.go` line 68 is the root cause for 15+ handlers — fixing `emitEvent` to extract user from context fixes all of them at once

---

### Issue 9: Database Performance — Connection Pool, Indexes, Partitions

**Priority**: P0-CRITICAL | **Est**: 4-6 hours | **Assign**: Danish
**Dependencies**: None
**Findings**: P1, P4, P5, H-P4

#### Problem

1. **Connection Pool Too Small** — `configs/server.yaml:25-26` — `max_conns: 25, min_conns: 5`. 1000 agents × heartbeat transactions = guaranteed pool exhaustion.

2. **No gRPC MaxConcurrentStreams** — `internal/server/grpc/server.go:24-38`. 1000 agents each with heartbeat + sync streams = unlimited connections → memory exhaustion.

3. **No 2027 Audit Partitions** — `internal/server/store/migrations/001_init_schema.sql` — partitions through 2026-12 only.

4. **Missing Database Indexes** — Hot-path tables (`deployment_targets`, `endpoint_cves`, `cves`, `patches`, `workflows`) missing indexes for common query patterns.

#### Files to Change
- `configs/server.yaml` — increase `max_conns`
- `internal/server/grpc/server.go` — add `MaxConcurrentStreams`
- `internal/server/store/migrations/` — new migration for 2027 partitions
- `internal/server/store/migrations/` — new migration for indexes

#### Fix Approach
1. Change `max_conns: 200, min_conns: 50` in server.yaml.
2. Add `grpc.MaxConcurrentStreams(500)` to gRPC server options.
3. New migration: CREATE TABLE partitions for audit_events 2027-01 through 2027-12 (follow pattern in 001_init_schema.sql).
4. New migration: CREATE INDEX for frequently queried columns. Analyze the query files to determine which columns need indexes.

#### Verification Checklist
- [ ] `configs/server.yaml` shows `max_conns: 200` (or similar high value)
- [ ] `grep -n 'MaxConcurrentStreams' internal/server/grpc/server.go` returns a match
- [ ] New migration creates 2027 monthly partitions — read the SQL
- [ ] New migration creates indexes — verify columns match query WHERE clauses
- [ ] `make migrate` succeeds
- [ ] `make test` passes

#### Related Patterns to Check
```bash
# Check hub config too
grep -n 'max_conns' configs/hub.yaml
# Check for any other missing gRPC options
grep -rn 'grpc.NewServer\|grpc.ServerOption' internal/
```

#### Context for Verification
- Look at `001_init_schema.sql` for the partition pattern — 2027 should follow the exact same format
- For indexes: look at the SQL queries in `internal/server/store/queries/` — any `WHERE` clause column that's not already indexed is a candidate
- **Do NOT modify existing migrations** — always create new sequential migrations

---

### Issue 10: Query Optimization — Unbounded Queries & Correlated Subqueries

**Priority**: P0-CRITICAL | **Est**: 12-16 hours | **Assign**: Rishab
**Dependencies**: None
**Findings**: P2, P3, H-P1, H-P3, NH14, NM17

#### Problem

Multiple queries will cause severe performance issues at 1000 endpoints:

1. **ListEndpointsByTenant Has No LIMIT** — `internal/server/store/queries/endpoints.sql:55-56` — loads ALL endpoints into memory. Used in QuickDeploy (`patches.go:659`).

2. **ListPatchesFiltered: 5 Correlated Subqueries Per Row** — `internal/server/store/queries/patches.sql:93-123` — 50 patches × 5 subqueries = 250 subqueries per page load.

3. **20+ Unbounded Queries** — Various query files have SELECT without LIMIT.

4. **Deployment Target Creation: 1 INSERT Per Endpoint** — `internal/server/api/v1/patches.go:729-743` — loop with individual INSERTs instead of batch.

5. **Dashboard INNER JOIN Excludes Ad-Hoc Deploys** — `internal/server/store/queries/dashboard.sql:21-35` — since migration 034 made `policy_id` nullable, INNER JOIN excludes ad-hoc deployments.

6. **ListAvailablePatches Ignores endpointID** — `NM17` — blank identifier discards the parameter.

#### Files to Change
- `internal/server/store/queries/endpoints.sql` — add LIMIT, pagination
- `internal/server/store/queries/patches.sql` — refactor to CTEs
- `internal/server/store/queries/dashboard.sql` — change INNER JOIN to LEFT JOIN
- `internal/server/api/v1/patches.go` — batch INSERT for deployment targets
- Multiple query files — add LIMIT to unbounded queries
- Regenerate: `make sqlc` after SQL changes

#### Fix Approach
1. Add `LIMIT @limit OFFSET @offset` parameters to all unbounded queries.
2. Refactor `ListPatchesFiltered` to use CTEs with pre-aggregated stats instead of correlated subqueries.
3. Change dashboard INNER JOINs to LEFT JOINs for nullable policy_id.
4. Replace the deployment target INSERT loop with a multi-row INSERT.
5. Fix `ListAvailablePatches` to use the endpointID parameter.
6. Run `make sqlc` to regenerate Go code after ALL SQL changes.

#### Verification Checklist
- [ ] `grep -rn 'SELECT.*FROM' internal/server/store/queries/ | grep -v 'LIMIT\|COUNT\|EXISTS\|JOIN\|WHERE.*=' | head -20` — most should have LIMIT
- [ ] `patches.sql` no longer has correlated subqueries in `ListPatchesFiltered` — read the SQL
- [ ] `dashboard.sql` uses LEFT JOIN for policy tables
- [ ] `patches.go` deployment creation uses batch INSERT — read the code
- [ ] `make sqlc` succeeds (generated code matches SQL)
- [ ] `make test` passes
- [ ] MANUALLY test: load the patches page and dashboard — they should still work correctly

#### Related Patterns to Check
```bash
# Find ALL unbounded queries
grep -rn 'SELECT.*FROM' internal/server/store/queries/ | grep -v 'LIMIT\|COUNT\|EXISTS\|WHERE.*id = '
# Find all correlated subqueries
grep -rn 'SELECT.*SELECT' internal/server/store/queries/
# Find all single-row INSERT loops
grep -rn 'for.*range.*Insert\|for.*range.*Create' internal/server/api/
```

#### Context for Verification
- **CRITICAL**: After changing SQL, ALWAYS run `make sqlc` — this regenerates the Go code in `store/sqlcgen/`. If you skip this, the Go code won't match the SQL and compilation will fail.
- For the CTE refactor: CTEs pre-compute the aggregated data once, then join — much faster than per-row subqueries.
- **Test manually**: Load patches page with test data. Compare query execution time before and after (use PostgreSQL `EXPLAIN ANALYZE`).

---

### Issue 11: Runtime Scalability — River Queues, Payload Limits, Wave Frequency

**Priority**: HIGH | **Est**: 6-8 hours | **Assign**: Rishab
**Dependencies**: None
**Findings**: H-P2, H-E2, M-P1

#### Problem

1. **Single River Queue** — `H-P2` — all job types (notifications, deployments, compliance, discovery, CVE sync) compete in one queue with 100 max workers. Notifications can starve deployments.

2. **No JSON Payload Size Limits** — `H-E2` — `json.NewDecoder(r.Body).Decode()` accepts unbounded input across all API handlers. A single malicious request can OOM the server.

3. **Wave Dispatcher Too Frequent** — `M-P1` — runs every 30 seconds. At 1000 endpoints with many deployments, this creates unnecessary DB load.

#### Files to Change
- River queue configuration in `cmd/server/main.go` — split into priority queues
- HTTP middleware or handler level — add `http.MaxBytesReader`
- Wave dispatcher configuration — adjust frequency

#### Fix Approach
1. **River queues**: Configure River with multiple queues (e.g., `critical` for deployments, `default` for notifications, `background` for compliance/discovery). Assign workers proportionally.
2. **Payload limits**: Add `r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)` (10MB) as middleware or per-handler. Chi middleware is ideal.
3. **Wave frequency**: Change from 30s to 60s or make configurable via config.

#### Verification Checklist
- [ ] River has multiple queues — read the configuration in main.go
- [ ] `grep -rn 'MaxBytesReader' internal/` returns middleware implementation
- [ ] Wave dispatcher frequency is >=60s or configurable
- [ ] `make test` passes
- [ ] Test payload limit: `curl -X POST -d @/dev/urandom -H 'Content-Type: application/json' http://localhost:8080/api/v1/endpoints` should return 413

#### Related Patterns to Check
```bash
# Any other unbounded readers
grep -rn 'json.NewDecoder(r.Body)' internal/server/ internal/hub/ | wc -l
# Other periodic job frequencies that might be too aggressive
grep -rn 'Periodic\|Schedule\|Interval\|30\|60' cmd/server/main.go
```

---

### Issue 12: Deployment & Policy Engine Fixes

**Priority**: HIGH | **Est**: 8-10 hours | **Assign**: Rishab
**Dependencies**: Soft depends on Issue 6 (workers instantiated)
**Findings**: NH1, NH12, NH16, H-F4, NM8, NM26, H-E5

#### Problem

1. **Overnight Maintenance Windows Broken** — `internal/server/deployment/maintenance.go:56-62` — if start > end (e.g., 22:00-06:00), check `minuteOfDay >= start && minuteOfDay < end` always returns false. Overnight windows silently non-functional.

2. **Policy Update Not Transactional** — `internal/server/api/v1/policies.go:834-916` — policy update, severity_filter update, and group replacement each run in separate transactions. If group update fails after policy update succeeds, data is inconsistent.

3. **Duplicate Severity Filter Functions** — `internal/server/deployment/evaluator.go:106` vs line 156 — `buildSeverityFilter` (private) and `BuildSeverityFilter` (exported) have different severity rankings (critical=0 vs critical=4, "none" included vs excluded).

4. **No Wave Rollback Trigger** — `H-F4` — wave dispatcher has no mechanism to trigger rollback when failure threshold is exceeded.

5. **Compliance Subqueries Lack framework_id Filter** — `NM8` — cross-framework results returned.

6. **Compliance Check Can't Set Value to 0** — `NM26` — treated as unset.

7. **Cron Validation at Wrong Time** — `H-E5` — bad cron expressions validated at execution time, not creation time.

#### Files to Change
- `internal/server/deployment/maintenance.go` — fix overnight window logic
- `internal/server/api/v1/policies.go` — wrap in single transaction
- `internal/server/deployment/evaluator.go` — consolidate severity functions
- `internal/server/deployment/wave_dispatcher.go` — add rollback trigger
- Compliance query files — add framework_id filter
- Schedule creation handler — validate cron at creation time

#### Fix Approach
1. **Overnight windows**: If start > end, check should be `minuteOfDay >= start || minuteOfDay < end` (OR, not AND).
2. **Policy update**: Wrap all three operations in a single transaction.
3. **Severity filter**: Remove one, keep the correct one. Ensure all callers use the consolidated version.
4. **Wave rollback**: Add failure count tracking. When failures exceed threshold (configurable), emit a rollback event.
5. **Compliance**: Add `WHERE framework_id = $1` to subqueries.
6. **Cron validation**: Validate cron expression in the API handler before storing. Return 400 if invalid.

#### Verification Checklist
- [ ] Maintenance window test: create window 22:00-06:00, check at 23:00 → should be IN window
- [ ] Policy update uses single transaction — `grep -A5 'BeginTx\|Begin(' internal/server/api/v1/policies.go`
- [ ] Only ONE severity filter function exists — `grep -n 'buildSeverityFilter\|BuildSeverityFilter' internal/server/deployment/evaluator.go`
- [ ] Bad cron expression returns 400 at creation time — test with `curl`
- [ ] `make test` passes

#### Related Patterns to Check
```bash
# Other time window checks that might have the same overnight bug
grep -rn 'minuteOfDay\|maintenance\|window' internal/server/deployment/
# Other multi-transaction operations that should be atomic
grep -rn 'BeginTx.*BeginTx\|Begin.*Begin' internal/server/api/v1/ | head -10
```

---

## WAVE 3: INTEGRATION & HUB (Issues 13-16, Week 3)

### CHECKPOINT GATE: All Wave 2 PRs merged. Both interns rebase.

---

### Issue 13: CVE Data Flow — Hub→Server→Agent

**Priority**: P0-CRITICAL | **Est**: 12-16 hours | **Assign**: Danish
**Dependencies**: Issue 1 must be merged (tenant context fixed in catalog_sync.go)
**Findings**: I1, H-I7, H-I8, NM13, H-E4

#### Problem

The CVE data pipeline between Hub, Server, and Agent is fundamentally broken:

1. **Agent CVE Detection Dropped** — `internal/server/grpc/sync_outbox.go:141-254` — Agent sends `detected_cves` in inventory proto, server's `processInventory` only processes `installed_packages`. Agent-side CVE data silently discarded.

2. **No Hub→Server CVE Sync** — Hub curates CVEs from 6 feeds (NVD, CISA KEV, MSRC, RedHat, Ubuntu, Apple). Server ignores Hub entirely and independently fetches only NVD. Hub's enriched CVE data never reaches Server.

3. **Catalog Sync Loses CVE Relationships** — `H-I7` — Hub's `patch_catalog_cves` table (patch-to-CVE mappings) not transmitted during catalog sync.

4. **Deleted Patches Not Propagated** — `H-I8` — Patches deleted from Hub are not soft-deleted on Server. `TODO(PIQ-118)`.

5. **CVE Correlator Exits on First Error** — `H-E4` — single patch lookup failure stops all correlation.

#### Files to Change
- `internal/server/grpc/sync_outbox.go` — process agent CVE data
- `internal/server/workers/catalog_sync.go` — extend to sync CVE relationships
- CVE correlator — continue on error instead of exiting
- Server-side CVE store — handle patch deletions

#### Fix Approach
1. In `sync_outbox.go`, add processing of `detected_cves` field from inventory reports. Either store them directly or use them to supplement server-side CVE matching.
2. Extend catalog sync to include `patch_catalog_cves` data from Hub. This requires Hub to include CVE mappings in its sync response and Server to store them.
3. Add soft-delete handling: when catalog sync detects a patch no longer exists on Hub, mark it as deleted on Server.
4. Fix CVE correlator to log errors and continue processing remaining patches.

#### Verification Checklist
- [ ] Agent `detected_cves` are processed — read `sync_outbox.go` processing logic
- [ ] Catalog sync includes CVE relationships — check both Hub response and Server storage
- [ ] CVE correlator doesn't stop on first error — read the error handling
- [ ] `make test` passes
- [ ] **Manual test**: Trigger a catalog sync and verify CVE data appears on Server

#### Context for Verification
- This is a large integration issue. Break it into sub-tasks if needed.
- The proto definitions in `proto/patchiq/v1/` define the contract between Hub and Server — read them to understand what data fields exist.
- `catalog_sync.go` is the file that was fixed in Issue 1 for SQL injection — make sure that fix is preserved.

---

### Issue 14: Hub Feed & Catalog Pipeline

**Priority**: HIGH | **Est**: 12-16 hours | **Assign**: Rishab
**Dependencies**: None
**Findings**: NH2, NH3, NH4, NH6, NH8, NH9, NH10, NM1, NM2, NM3

#### Problem

Hub's feed processing and catalog pipeline have multiple data corruption issues:

1. **MSRC Cursor Broken** — `internal/hub/feeds/msrc.go:66-69` — `updateID <= cursor` uses lexicographic comparison on strings like "2025-Jan", "2025-Aug". "Aug" < "Feb" lexicographically → August entries skipped.

2. **Binary Fetch Wrong Path** — `internal/hub/workers/binary_fetch.go:109` — same value passed as both osFamily and osVersion. Files stored as `patches/ubuntu/ubuntu/curl.deb` instead of `patches/ubuntu/22.04/curl.deb`.

3. **Ubuntu Severity Always "medium"** — `internal/hub/feeds/ubuntu.go:179` — hardcoded regardless of actual severity.

4. **Package Alias Update Ignores URL ID** — `internal/hub/api/v1/package_aliases.go:180-228` — PUT to `/package-aliases/123` may update wrong record.

5. **UUID Formatting Non-Standard** — `internal/hub/workers/binary_fetch.go:167-172` — `uuidToStr` uses `%x` without zero-padding. Also in `internal/server/workers/catalog_sync.go:404-409`.

6. **Dead Fetchers** — `internal/hub/catalog/` has `fetcher_yum.go`, `fetcher_msu.go`, `fetcher_apple.go` that are never instantiated.

7. **License Generator Never Wired** — `internal/hub/license/generate.go` is complete but never imported or used.

8. **RedHat Only RHEL 9** — `NM1` — RHEL 7/8 not covered.

9. **Ubuntu Only 3 Versions** — `NM2` — Noble, Jammy, Focal. Missing LTS versions.

10. **APT Only amd64** — `NM3` — arm64 packages never resolved.

#### Files to Change
- `internal/hub/feeds/msrc.go` — fix cursor comparison
- `internal/hub/workers/binary_fetch.go` — fix path + UUID formatting
- `internal/hub/feeds/ubuntu.go` — parse actual severity
- `internal/hub/api/v1/package_aliases.go` — use URL ID parameter
- `internal/server/workers/catalog_sync.go` — fix UUID formatting (same bug)
- `internal/hub/feeds/redhat.go` — add RHEL 7/8 support
- `internal/hub/feeds/ubuntu.go` — add more versions + arm64

#### Fix Approach
1. **MSRC cursor**: Parse the updateID into a date and compare dates, not strings. E.g., parse "2025-Aug" to time.Time and compare chronologically.
2. **Binary path**: Pass the correct osVersion parameter instead of osFamily twice.
3. **Ubuntu severity**: Parse the actual severity from the USN feed data (it should be in the feed response).
4. **Package alias**: Use the URL `{id}` parameter for the UPDATE WHERE clause.
5. **UUID formatting**: Use `%08x-%04x-%04x-%04x-%012x` with zero-padding, or use a UUID library.
6. **Dead fetchers**: Wire them into the feed registry in `cmd/hub/main.go`, or remove if not needed yet.
7. **License generator**: Wire into `licenses.go` API handler, or mark as TODO with issue reference.

#### Verification Checklist
- [ ] MSRC cursor comparison works: "2025-Aug" should be AFTER "2025-Feb" — write a test
- [ ] Binary fetch path includes correct osVersion — read the `FetchAndStore` call
- [ ] Ubuntu feed extracts actual severity from feed data — read the parsing code
- [ ] Package alias PUT uses URL ID — `grep -n '{id}' internal/hub/api/v1/package_aliases.go`
- [ ] UUID format is 36 characters (8-4-4-4-12) — write a test
- [ ] `make test` passes

#### Related Patterns to Check
```bash
# Same UUID formatting bug in server
grep -rn 'uuidToStr\|%x-%x-%x-%x' internal/server/
# Any other lexicographic comparisons on non-alphabetical data
grep -rn '<= cursor\|>= cursor\|< cursor\|> cursor' internal/hub/
```

---

### Issue 15: Hub API & Auth Hardening

**Priority**: HIGH | **Est**: 6-8 hours | **Assign**: Rishab (or Danish if ahead)
**Dependencies**: None
**Findings**: NH22, H-S2, NM4, NM5, NM6, NM7

#### Problem

1. **Rotate API Key Button Does Nothing** — `web-hub/src/pages/settings/APIWebhookSettings.tsx:217-233` — renders clickable red button with no onClick handler.

2. **Client Registration Uses Hardcoded Tenant** — `H-S2` — `internal/hub/api/v1/clients.go:85` uses default tenant ID for all registrations. Breaks multi-tenancy.

3. **Bootstrap Token Stored in Plaintext** — `NM4` — not hashed like API keys.

4. **Dashboard References 'degraded' Status** — `NM5` — not in CHECK constraint, always shows 0.

5. **hub_sync_state Filter on 'disabled'** — `NM6` — not in CHECK constraint, no-op filter.

6. **SyncStarted Event Never Emitted** — `NM7` — defined but never used.

#### Files to Change
- `web-hub/src/pages/settings/APIWebhookSettings.tsx` — add onClick handler or remove button
- `internal/hub/api/v1/clients.go` — use actual tenant ID
- Hub bootstrap token storage — hash before storing
- Dashboard/sync queries — fix status values

#### Verification Checklist
- [ ] Rotate button either works (calls API) or is hidden/disabled with a TODO
- [ ] Client registration uses tenant from context, not hardcoded
- [ ] `make test` passes

---

### Issue 16: Integration Hardening — Enrollment, mTLS, Commands, Config Push, License

**Priority**: HIGH | **Est**: 14-18 hours | **Assign**: Danish
**Dependencies**: Issues 1 and 6 should be merged first
**Findings**: H-I1, H-I2, H-I3, H-I4, H-I5, H-I6, I2, S4

#### Problem

Multiple integration flows between Hub/Server/Agent are incomplete:

1. **Enrollment Token Never Expires** — `H-I1` — no `expires_at` column. Proto defines TOKEN_EXPIRED error but it's never checked.

2. **mTLS Certificate Always Empty** — `H-I2` — `EnrollResponse` has cert field but no cert generation code.

3. **Command Timeout Not Enforced** — `H-I3` — agent reboot mid-deployment hangs forever.

4. **No Cross-Platform Audit Correlation** — `H-I4` — no correlation ID between Hub/Server/Agent events.

5. **Config Push Not Implemented** — `H-I5` — heartbeat `config_update` field always nil.

6. **Duplicate agent_binaries Table** — `H-I6` — exists in both Hub and Server DBs with no sync.

7. **License Validation No-Op** — `I2` — Proto defines `ValidateLicense` RPC but Server has no client.

8. **License Keys Forgeable** — `S4` — just JSON, not RSA-signed.

#### Files to Change
- Enrollment: add `expires_at` column + migration, check in enrollment handler
- Command handling: add timeout tracking + enforcement
- Heartbeat: implement config push mechanism
- License: implement validation client or stub with clear TODO

#### Fix Approach

**Prioritize for client demo**: Focus on enrollment expiry (1), command timeout (3), and config push (5). The others (mTLS, correlation ID, license) are important but less likely to surface in a 1-month test.

1. **Enrollment expiry**: Add migration for `expires_at` column on enrollment tokens. Check expiry during enrollment. Set default TTL (e.g., 7 days).
2. **Command timeout**: Add a `timeout_at` field to command tracking. Background job checks for timed-out commands and marks them failed.
3. **Config push**: Populate the `config_update` field in heartbeat response when endpoint config has changed since last heartbeat.
4. **License**: If RSA signing is too complex for this sprint, at minimum add a TODO with issue reference and ensure the forgeable keys can't be exploited (e.g., validate server-side).

#### Verification Checklist
- [ ] Enrollment token has `expires_at` — check migration and DB schema
- [ ] Enrollment handler rejects expired tokens — read the code
- [ ] Commands have timeout enforcement — check the timeout job
- [ ] Heartbeat response includes config updates when config changed
- [ ] `make test` passes
- [ ] **Manual test**: Create an enrollment token, wait for expiry, try to enroll — should fail

#### Context for Verification
- This is the largest issue. Break it into sub-PRs if needed (e.g., one for enrollment, one for commands, one for config push).
- Read the proto definitions in `proto/patchiq/v1/` — they define what fields exist for enrollment, heartbeat, etc.
- The enrollment flow: Agent calls `Enroll()` with a token → Server validates → returns config + cert. The token validation is where expiry should be checked.

---

## WAVE 4: FRONTEND & PRODUCTION (Issues 17-20, Week 4)

### CHECKPOINT GATE: All Wave 3 PRs merged. Full E2E test.

---

### Issue 17: Frontend Critical — SLA Fake Data, Error Boundaries, TypeScript

**Priority**: P0-CRITICAL | **Est**: 8-10 hours | **Assign**: Rishab
**Dependencies**: None
**Findings**: NC7, H-FE1, NH21

#### Problem

1. **SLA Dashboard Shows Fabricated Data** — `web/src/pages/dashboard/SLACountdown.tsx:130-155` constructs fake SLA deadlines from hardcoded arrays. `SLAStatus.tsx:87-100` calculates progress from array position. Client will see "22d 12h remaining" regardless of actual state.

2. **No Error Boundaries in web-hub and web-agent** — `H-FE1` — component crash = full app crash.

3. **39 `as any` TypeScript Casts** — `NH21` — concentrated in useCompliance.ts (8), useIAMSettings.ts (4), useSettings.ts (2), useChannelByType.ts (3), useRoles.ts (2), useDashboard.ts (3), plus compliance/audit/deployment pages.

#### Files to Change
- `web/src/pages/dashboard/SLACountdown.tsx` — use real API data
- `web/src/pages/dashboard/SLAStatus.tsx` — use real API data
- `web-hub/src/` — add error boundary component
- `web-agent/src/` — add error boundary component
- Multiple hook files — fix `as any` casts

#### Fix Approach
1. **SLA**: If a real SLA API exists, use it. If not, either hide the SLA widgets or show "SLA data not configured" instead of fake numbers.
2. **Error boundaries**: Create a reusable `ErrorBoundary` component (React class component with `componentDidCatch`). Wrap route-level components.
3. **TypeScript casts**: Fix each `as any` by using proper types from the generated API types or creating local interfaces.

#### Verification Checklist
- [ ] SLA widgets show real data OR clearly indicate "not configured" — look at the component, no hardcoded dates/arrays
- [ ] Error boundaries exist in web-hub and web-agent — check route wrapping
- [ ] `grep -rn 'as any' web/src/ web-hub/src/ web-agent/src/ | wc -l` — should be significantly reduced (ideally zero without justification)
- [ ] `pnpm run build` passes for all 3 frontends (type check)
- [ ] `pnpm run lint` passes

#### Related Patterns to Check
```bash
# Other hardcoded/fake data in dashboards
grep -rn 'hardcode\|mock\|fake\|dummy\|placeholder' web/src/pages/dashboard/
# Other missing error boundaries
grep -rn 'ErrorBoundary\|componentDidCatch' web/src/ web-hub/src/ web-agent/src/
```

#### Context for Verification
- **The SLA fake data is the #1 most embarrassing finding.** Client will see countdown timers showing completely fictional deadlines. This MUST be fixed.
- For `as any`: the generated types in `web/src/api/types.ts` (5831 lines) should have correct types for most API responses. The casts are usually because the hook doesn't match the generated type — fix the type, not add a cast.

---

### Issue 18: Frontend Cleanup — Raw Fetch, Dead Components, Code Splitting

**Priority**: HIGH | **Est**: 8-12 hours | **Assign**: Rishab
**Dependencies**: Issue 17 should be merged first (frontend critical)
**Findings**: NH23, H-FE2, H-FE3, M-DC1, M-FE2, M-DC3

#### Problem

1. **26+ Raw fetch() Calls** — `NH23` — Tags (7), endpoints hooks (7), login hooks (4), auth hooks (2), command palette (3), audit export (1), endpoint export (1), health (1), agent binaries (1). All should use `openapi-fetch`.

2. **KEV Column Placeholder** — `H-FE3` — vulnerability status shows "—" dash.

3. **5 Orphaned Components** — `M-DC1` — SegmentedProgressBar, CVSSVectorBreakdown, SlidePanel, SeverityPills, StatsStrip.

4. **No Route-Level Code Splitting** — `M-FE2` — 12MB+ @xyflow loaded upfront even on non-workflow pages.

5. **console.error in Production** — `M-DC3` — SoftwareTab, CreateTagDialog.

#### Files to Change
- Multiple hook files — convert raw fetch to openapi-fetch
- Orphaned components — delete them
- Route config — add React.lazy() for heavy pages
- Components with console.error — replace with proper error handling

#### Fix Approach
1. Convert raw `fetch()` calls to use the existing `openapi-fetch` client from `api/client.ts`. Follow patterns in hooks that already do this correctly.
2. Delete orphaned components (verify they're truly unused first with grep).
3. Use `React.lazy()` and `Suspense` for the workflow builder route (imports @xyflow).
4. Replace `console.error` with proper error handling (toast notification or error state).

#### Verification Checklist
- [ ] `grep -rn 'fetch(' web/src/ web-hub/src/ web-agent/src/ | grep -v node_modules | grep -v 'openapi-fetch\|client.ts' | wc -l` — significantly reduced
- [ ] Orphaned components deleted — `grep -rn 'SegmentedProgressBar\|CVSSVectorBreakdown\|SlidePanel\|SeverityPills\|StatsStrip' web/ web-hub/` returns nothing
- [ ] Workflow page uses lazy loading — check routes.tsx
- [ ] `grep -rn 'console.error\|console.log' web/src/ web-hub/src/ web-agent/src/ | grep -v node_modules | grep -v '_test\|.test\|.spec'` — clean
- [ ] `pnpm run build` succeeds
- [ ] `pnpm run lint` passes

---

### Issue 19: Production Readiness — Health, Config, Secrets, Rate Limiting

**Priority**: HIGH | **Est**: 12-16 hours | **Assign**: Danish
**Dependencies**: None
**Findings**: H-PR1, H-PR2, H-PR3, H-PR4, S5, M-PR1, M-PR2, M-PR3, M-S1, M-S2, M-S3, H-S1, NM9, NM10, M-C1, M-C2, M-C3, H-CI1, H-CI3

#### Problem

Multiple production-readiness gaps:

1. **Health Checks Only Verify DB** — `H-PR1` — missing gRPC, Valkey, event bus, River queue checks.
2. **Idempotency Fallback Loses Guarantees** — `H-PR2` — falls back to in-memory when Valkey unavailable.
3. **Notification Encryption Key Ephemeral** — `H-PR3` — changes on restart, breaking stored credentials.
4. **No Config Validation at Startup** — `H-PR4` — services start with incomplete config.
5. **Hardcoded Dev Credentials** — `S5` — DB passwords, MinIO creds in committed configs.
6. **CORS Falls Back to Localhost** — `M-PR1` — dev origins used if not configured.
7. **Rate Limit Store Unbounded** — `M-PR2` — no expired bucket cleanup.
8. **No API-Wide Rate Limiting** — `H-S1` — only auth endpoints rate-limited.
9. **SSL Disabled** — `H-CI3` — `sslmode=disable` in all configs.
10. **ServerConfig Missing Timeout Validation** — `NM9`.
11. **gRPC Ignores TLS Config** — `NM10`.
12. **Sensitive Data Logged** — `M-S1` — Valkey URL with credentials.

#### Files to Change
- Health check handlers — add dependency checks
- Config validation — add fail-fast validation at startup
- Config files — move secrets to env vars
- Rate limiting middleware — add to all API routes
- Various production hardening

#### Fix Approach

**Prioritize for client demo**: Focus on health checks (1), config validation (4), secrets (5), rate limiting (8). Others are important but lower priority.

1. **Health checks**: Extend health endpoint to ping Valkey, check River queue status, verify event bus connectivity.
2. **Config validation**: Add a `Validate()` method to ServerConfig that checks all required fields. Call at startup before anything else.
3. **Secrets**: Create `.env.example` with placeholder values. Update configs to use `${ENV_VAR}` references. Add `*.key`, `*.pem` to `.gitignore`.
4. **Rate limiting**: Add rate limiting middleware to the main router (not just auth). Use `M-PR2` fix for cleanup.
5. **Encryption key**: Require `notification.encryption_key` in config. Fail startup if missing in production mode.

#### Verification Checklist
- [ ] Health endpoint checks more than just DB — read the handler
- [ ] Server fails to start with missing required config fields
- [ ] No passwords in committed config files (only env var references)
- [ ] `.gitignore` includes `*.key`, `*.pem`, `.env`
- [ ] Rate limiting on all API routes — check middleware chain
- [ ] `make test` passes

---

### Issue 20: Testing & Dead Code Cleanup

**Priority**: HIGH | **Est**: 14-18 hours | **Assign**: Both interns (split work)
**Dependencies**: All feature issues (1-19) should be merged first
**Findings**: H-T1, H-T2, H-T3, H-T4, H-DC1, H-DC2, M-T1, M-T2, M-T3, NM27, 90 unused SQL queries

#### Problem

1. **No Tenant Isolation Test** — `H-T1` — if a query forgets `WHERE tenant_id`, all tenants see all data. No integration test catches this.
2. **Integration Tests Not in CI** — `H-T2` — build tag `integration` never passed.
3. **4 Tests for 34 DB Store Files** — `H-T3` — almost no query correctness testing.
4. **CVE Version Range Untested** — `H-T4`.
5. **90 Unused SQL Queries** — `H-DC1` — generating dead Go code.
6. **MockSender in Production Code** — `NM27` — `notify/sender.go` instead of test file.

#### Work Split
- **Rishab**: Testing (H-T1, H-T2, H-T3, H-T4, M-T1, M-T2, M-T3)
- **Danish**: Dead code cleanup (H-DC1, H-DC2, NM27, orphaned components already in Issue 18)

#### Fix Approach
1. **Tenant isolation test**: Create integration test that inserts data for tenant A, queries as tenant B, verifies zero results.
2. **CI integration tests**: Add `-tags integration` to CI test command. May need testcontainers config.
3. **Store tests**: Add tests for the most critical query files (endpoints, patches, deployments, compliance).
4. **Unused queries**: Remove from `.sql` files, run `make sqlc` to regenerate. Verify no compile errors.
5. **MockSender**: Move to a `_test.go` file or a `testutil` package.

#### Verification Checklist
- [ ] Tenant isolation test exists and passes — `go test -run TestTenantIsolation`
- [ ] `make sqlc` succeeds after removing unused queries
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make build` succeeds (no compilation errors from removed queries)

#### Context for Verification
- **CRITICAL for dead code removal**: After removing SQL queries, run `make sqlc` AND `make build`. If any Go code references the removed queries, compilation will fail and tell you exactly which files need updating.
- For unused queries: the audit report lists them. Verify each one is truly unused by grep-ing for the generated Go function name.

---

## Deferred Items (Post-Client-Test)

These medium/low issues are unlikely to surface during the 1-month client test and can be addressed after:

- M-DB1 through M-DB7: Database schema denormalization (works, just not ideal)
- M-FE1, M-FE3, M-FE4, M-FE5: Frontend polish (responsive, accessibility, staleTime)
- M-I2, M-I3, M-I4: Minor integration edge cases
- M-E1: Deferred rollback errcheck (11 instances)
- H-CI2: Grafana dashboards not committed
- NM19: WUA COM apartment models (Windows-specific edge case)
- General TODO/FIXME cleanup (29 comments — address as encountered)

---

## Summary

| Wave | Issues | Intern A (Rishab) | Intern B (Danish) | Gate |
|------|--------|-------------------|-------------------|------|
| 1: Security & Stability | 1-7 | 1, 6, 5, 7 | 2, 4, 3, 9 | `make test && make lint` |
| 2: Scalability & Data | 8-12 | 12, 11, 10(start) | 8, 15, 14(start) | `make test && make lint` |
| 3: Integration & Hub | 13-16 | 10(finish), 14(finish), 17 | 13, 15 | Manual E2E test |
| 4: Frontend & Production | 17-20 | 17, 18, 20(split) | 16, 19, 20(split) | `make ci` + full E2E |

**20 issues. ~180-220 hours. 4 weeks. 2 interns.**

Every issue follows the same workflow: worktree → Claude Code implements → intern verifies → `/review-pr` → `/commit-push-pr` → Heramb reviews.

**The intern is the quality gate, not Claude.**
