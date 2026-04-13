# PatchIQ Full Project Audit — Executive Summary

**Date**: 2026-04-09
**Branch**: dev-a
**Audited by**: 12 parallel audit agents covering all code, config, and docs

---

## Verdict

The product is **architecturally solid but has significant gaps between "built" and "works E2E"**. The Go backend is well-structured with good test coverage (245 test files, 1175+ functions). The frontend has the right stack and patterns but suffers from dead code accumulation, inconsistent API client usage, and non-functional UI elements that would embarrass a client POC.

**The project is ~80% done but the last 20% is scattered across every layer.** It's not one big missing feature — it's dozens of small things: buttons that don't work, events that silently fail, pages that are stubs, queries that are never called, components duplicated instead of shared.

---

## Critical Findings (Fix Before Client POC)

These are bugs or security issues that would directly impact a client deployment.

| # | Finding | Area | Impact |
|---|---------|------|--------|
| 1 | **JWT claim injection** — Hub `mintJWT` uses `fmt.Sprintf` with user-controlled values | Hub Auth | Security: attacker can inject arbitrary JWT claims |
| 2 | **PolicyAutoDeployed event silently fails** — not in AllTopics(), so Emit() errors out | Server Events | Auto-deploy policies create no audit trail |
| 3 | **2 write operations emit no domain events** — UpdatePreferences, HubSync UpdateConfig | Server API | Audit gaps, violates core invariant |
| 4 | **Duplicate UserID context keys** — otel vs user package use different keys | Shared | User IDs missing from ALL structured logs |
| 5 | **No 2027 audit partitions** — data falls into default partition in ~8 months | Database | Partition pruning breaks, audit queries degrade |
| 6 | **deployment_targets missing indexes** on endpoint_id and patch_id | Database | Hot-path queries will degrade at scale |
| 7 | **Workflow CRUD routes missing RBAC** — any authenticated user can modify workflows | Server API | Authorization bypass |
| 8 | **Postgres password mismatch** between docker-compose and config files | Infrastructure | Dev environment breaks without env overlay |
| 9 | **Hub route path mismatch** — router vs handler/tests reference different paths | Hub API | Client registration status check broken |
| 10 | **Scan endpoint is a no-op** — returns 202 but never triggers scan | Agent | Core feature non-functional |

---

## The "Clutter" Problem (Your Intuition Is Right)

**~170+ files that should be deleted or consolidated:**

| Category | Estimated Files | Lines |
|----------|----------------|-------|
| Dead frontend components/pages | ~25 files | ~4,000 lines |
| Unused sqlc queries (62 total) | 62 query definitions | ~1,500 lines |
| Root-level snapshot artifacts | 4 files | ~200 lines |
| `prototype-ui/` (stale mockups) | entire directory | 5.6 MB |
| `.superpowers/` tracked despite .gitignore | 31 files | — |
| `.omniprod/` + `.omniprod-plugin/` tracked | 75 files | — |
| Unused shared UI exports | ~40 exports | — |
| Duplicate install scripts (deploy/ vs scripts/) | 2+ files | — |
| Dead Go code (unused types, functions) | ~15 items | ~500 lines |
| **Total estimated removable** | **~250+ files/items** | **~6,000+ lines + 5.6 MB** |

---

## Non-Functional Buttons & Stub Pages (Client-Visible)

These are UI elements a client would click and nothing happens:

| UI Element | App | Status |
|-----------|-----|--------|
| Install/Skip buttons on Pending Patches | web-agent | No onClick handler |
| Add to Group dialog | web | TODO — confirms but doesn't call API |
| Patch Recall button | web | TODO — visible but non-functional |
| Deployments page | web-hub | "Coming Soon" stub |
| Add Feed form | web-hub | Shows "not yet supported" toast |
| IAM Test Connection | web-hub | Fakes success with setTimeout |
| Search button (header) | web-hub | Non-functional |
| Bell/notifications button | web-hub | Non-functional |
| Rotate API Key button | web-hub | Non-functional |
| Add Role Mapping button | web-hub | Non-functional |
| Save All Feeds button | web-hub | Non-functional |
| Compliance dashboard widget | web-agent | Static placeholder |

---

## What To Do: Prioritized Action Plan

### Phase 1: Security & Correctness (YOU — this week)

These require core dev judgment. Don't delegate to interns.

1. **Fix JWT claim injection** in hub `mintJWT` — use proper JWT library claims builder
2. **Add PolicyAutoDeployed to AllTopics()** — one-line fix, huge impact
3. **Add domain events** to UpdatePreferences and HubSync UpdateConfig
4. **Fix UserID context key duplication** — align otel and user packages
5. **Add RBAC middleware** to workflow CRUD routes
6. **Fix hub route path** mismatch (register/status)
7. **Wire scan endpoint** to actually trigger CollectionRunner in agent
8. **Create 2027 audit partitions** (server + hub)
9. **Add indexes** on deployment_targets.endpoint_id, deployment_targets.patch_id, deployments.policy_id
10. **Fix Postgres password** consistency across configs

### Phase 2: Dead Code Cleanup (INTERNS — next 1-2 weeks)

Low-risk, high-impact cleanup. Perfect intern work with clear checklists.

**Intern Task A: Frontend Dead Code Removal**
- Delete all dead files identified in reports 05, 06, 07 (~25 files)
- Remove duplicate EmptyState/ErrorState (use @patchiq/ui versions)
- Remove 3 unused data-table subcomponents from web/
- Delete PlaceholderPage.tsx from all 3 frontends
- Remove dead types/exports

**Intern Task B: Root & Config Cleanup**
- Delete root snapshot files (agent-overview-snapshot.md, hardware-snapshot.md, compliance-page.yml, policy-detail-snap.md)
- Delete `prototype-ui/` directory
- `git rm -r --cached .superpowers/` and ensure .gitignore works
- `git rm -r --cached .omniprod/ .omniprod-plugin/`
- Consolidate duplicate install scripts (pick deploy/scripts/ as canonical)
- Fix `scripts/dev-cleanup.sh` paths

**Intern Task C: Unused Query Cleanup**
- Audit the 44 unused server queries + 18 unused hub queries
- Remove confirmed-unused ones (keep any that are "planned for next milestone")
- Run `make sqlc` after cleanup

### Phase 3: Frontend Consistency (INTERNS with Sandy reviewing)

**Intern Task D: web-hub API Client Migration**
- Replace all `apiFetch` calls with the existing `openapi-fetch` typed client
- Remove the 3 duplicate `apiFetch` copies
- Remove hardcoded tenant IDs (read from context)

**Intern Task E: web-agent API Client Migration**
- Replace 6 raw `fetch()` hooks with `openapi-fetch` client
- Extend OpenAPI spec to cover the 6 missing agent endpoints first

**Intern Task F: Non-Functional Button Audit**
- For each non-functional button: either implement it or remove it
- "Coming Soon" stubs should be hidden behind feature flags, not shown to clients
- Fake "Test Connection" must either work or be removed

### Phase 4: Shared Component Consolidation (Sandy)

- Upgrade shared `StatCard` to support `active`, `onClick`, `valueColor` props
- Migrate all 14+ inline StatCard reimplementations to shared version
- Build shared `DataTable` with TanStack Table integration (replaces 3 copies)
- Build shared `FilterBar` (replaces 3 copies)
- Add `Textarea` to shared components
- Unify Toaster usage (use shared, not direct sonner import)

### Phase 5: Testing & Hardening (After cleanup)

- **Add multi-tenant isolation integration test** (Critical — #1 security risk has no E2E test)
- **Add E2E smoke tests** for the client POC happy path
- **Escape SQL LIKE patterns** consistently (only workflows.go does it today)
- **Fix hub `catalog_entry_syncs` RLS** (missing FORCE ROW LEVEL SECURITY)
- **Add auth guard** to web-hub (currently falls back to hardcoded dev user)
- **Add 404 catch-all routes** to all 3 frontends

---

## What NOT To Do

- Don't add new features. Every existing feature needs to work first.
- Don't refactor the shared config hierarchy — it's unused but harmless. Delete it later.
- Don't upgrade RSA-2048 to 4096 now — it works, it's not a client POC blocker.
- Don't build load tests yet — correctness before performance.
- Don't write a README yet — focus on the product, not onboarding docs.

---

## Detailed Reports

| Report | File |
|--------|------|
| Backend Server | [01-backend-server.md](01-backend-server.md) |
| Backend Hub | [02-backend-hub.md](02-backend-hub.md) |
| Backend Agent | [03-backend-agent.md](03-backend-agent.md) |
| Shared Packages | [04-shared-packages.md](04-shared-packages.md) |
| Frontend Web | [05-frontend-web.md](05-frontend-web.md) |
| Frontend Web-Hub | [06-frontend-web-hub.md](06-frontend-web-hub.md) |
| Frontend Web-Agent | [07-frontend-web-agent.md](07-frontend-web-agent.md) |
| Database | [08-database.md](08-database.md) |
| Infrastructure | [09-infrastructure.md](09-infrastructure.md) |
| Test Coverage | [10-test-coverage.md](10-test-coverage.md) |
| Shared UI | [11-shared-ui.md](11-shared-ui.md) |
| Docs & Clutter | [12-docs-and-clutter.md](12-docs-and-clutter.md) |
