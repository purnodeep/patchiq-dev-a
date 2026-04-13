# Production Hardening Plan — Everest Deployment

**Context**: Deploying PatchIQ to Everest's infrastructure for a 1-month testing period.
**Constraint**: Everest previously used Zirozen; their automation expects Zirozen's API format.
**Date**: 2026-04-09
**Source**: Full 12-report audit (see `Audit/`)

---

## Phase 0: Security & Correctness (Heramb — Week 1)

Core dev only. Non-delegable. These would cause security incidents or data loss.

| # | Fix | File(s) | Effort | Audit Ref |
|---|-----|---------|--------|-----------|
| 1 | JWT claim injection — use `json.Marshal` for claims | `hub/auth/session.go:39` | 1h | Hub H-C1 |
| 2 | Add `PolicyAutoDeployed` to `AllTopics()` | `server/events/topics.go` | 5min | Server 7.1 |
| 3 | Add domain events to UpdatePreferences + HubSync UpdateConfig | `server/api/v1/notifications.go`, `hub_sync.go` | 1h | Server 3.1, 3.2 |
| 4 | Add RBAC middleware to workflow CRUD routes | `server/api/router.go:325-331` | 30min | Server 4.4 |
| 5 | Fix UserID context key duplication | `shared/otel/context.go` → use `user.UserIDFromContext` | 30min | Shared O-1 |
| 6 | Fix hub route path mismatch | `hub/api/router.go:93` | 15min | Hub H-C2 |
| 7 | Wire agent scan endpoint to CollectionRunner | `agent/api/scan.go` | 2h | Agent 2.1 |
| 8 | Fix catalog_entry_syncs RLS (add FORCE + WITH CHECK) | Hub migration | 15min | DB T01 |
| 9 | Add missing indexes (deployment_targets, deployments) | Server migration | 30min | DB I01-I03 |
| 10 | Create 2027 audit partitions | Server + hub migrations | 30min | DB M01 |
| 11 | Fix Postgres password consistency | docker-compose, configs | 15min | Infra 2A |
| 12 | Fix web-hub auth guard (remove dev user fallback) | `web-hub/AuthContext.tsx` | 1h | Web-Hub 7.2 |
| 13 | Fix UUID formatting in hub binary_fetch worker | `hub/workers/binary_fetch.go:167` | 15min | Hub H-C3 |
| | **Total** | | **~8h** | |

---

## Phase 0.5: Zirozen Compatibility Layer (Heramb + Sandy — Week 1-2)

Build the adapter layer so Everest's existing automation works.

| # | Component | Effort | Details |
|---|-----------|--------|---------|
| 1 | Foundation: types, transforms, qualification parser | 3h | `internal/server/compat/zirozen/` |
| 2 | ID mapping table + migration + sqlc queries | 2h | `compat_id_map` table |
| 3 | Auth: compat token endpoint + middleware | 3h | Password grant → compat JWT |
| 4 | Asset search by UUID | 2h | Simplest handler, validates pattern |
| 5 | Patch search with qualification filters | 4h | Most-used endpoint |
| 6 | Asset-patch relation (installed/missing) | 4h | Complex joins |
| 7 | Scan trigger | 1h | Depends on Phase 0 #7 |
| 8 | Deployment create | 4h | Policy model bridging |
| 9 | Deployment search | 3h | Status mapping |
| | **Total** | **~26h** | |

See `compat-layer-plan.md` for full implementation details.

---

## Phase 1: Client-Visible Broken UX (Heramb + Sandy — Week 2)

Hide or implement every non-functional UI element.

**Hide (backend not ready):**
- Add to Group dialog (web) — hide confirm button
- Patch Recall button (web) — hide
- Deployments page (web-hub) — remove from nav
- Add Feed form (web-hub) — hide button
- Search, Bell, Rotate API Key, Add Role Mapping, Save All Feeds (web-hub) — remove all

**Implement (core features):**
- Install/Skip buttons on Pending Patches (web-agent) — wire to patch install/skip API
- IAM Test Connection (web) — remove fake setTimeout, either wire to real endpoint or remove

**Fix:**
- Add 404 catch-all routes to all 3 frontends
- Add `EscapeLikePattern` to all List handlers (not just workflows)
- Wire RSA license generator into hub API (replace JSON placeholder)

---

## Phase 2: Dead Code Cleanup (Interns — Week 2-3, parallel)

**Intern Task A: Frontend dead files (~25 files)**
- Delete all files from `Audit/13-dead-code-verification.md` marked "YES"
- Clean up re-exports in `web/src/components/data-table/index.ts`
- Delete `web/src/pages/groups/` directory + tests

**Intern Task B: Root & config cleanup**
- Delete root snapshots (4 files)
- Delete `prototype-ui/` directory
- `git rm -r --cached .superpowers/ .omniprod/ .omniprod-plugin/`
- Fix `scripts/dev-cleanup.sh` paths
- Consolidate install scripts (keep `scripts/`, remove `deploy/scripts/` duplicates)

**Intern Task C: Unused sqlc queries**
- Remove 62 confirmed-unused queries from `.sql` files
- Run `make sqlc` to regenerate

**Intern Task D: web-hub API client migration**
- Replace `apiFetch` copies with openapi-fetch typed client
- Consolidate hardcoded tenant IDs to single constant
- Remove 3 duplicate `apiFetch` implementations

---

## Phase 3: Frontend Consistency (Sandy + Interns — Week 3)

- Update OpenAPI specs to cover all endpoints
- Regenerate types, remove 50+ `as any` casts
- Upgrade shared StatCard, migrate 14+ inline copies
- Move FilterBar (3 copies) and DataTable (3 copies) to `packages/ui/`
- Delete local EmptyState/ErrorState duplicates
- Bundle Geist fonts (client infra may be air-gapped)

---

## Phase 4: Testing & Hardening (Heramb + Sandy — Week 3-4)

**Must-have:**
1. Multi-tenant isolation integration test
2. Smoke E2E tests for top 5 user flows
3. Compat layer round-trip tests (using exact Zirozen PDF examples)
4. Add `go test -coverprofile` to CI (60% floor)

**Nice-to-have:**
- Basic load test (100 agents, 50 API users)
- Hub-to-server sync integration test
- Frontend API hook tests (MSW)

---

## What NOT to Do

- No new features until all existing features work E2E
- No RSA-2048 → 4096 upgrade (works fine for POC)
- No NewRouter refactor (cosmetic)
- No config hierarchy wiring (dead but harmless)
- No load testing before correctness is locked down
- No README (focus on product)

---

## Assignment Summary

| Person | Week 1 | Week 2 | Week 3 | Week 4 |
|--------|--------|--------|--------|--------|
| **Heramb** | Phase 0 (security) | Phase 0.5 (compat) + Phase 1 | Phase 4 (testing) | QA + buffer |
| **Sandy** | Phase 0.5 (compat) | Phase 1 (UX fixes) | Phase 3 (frontend) | QA + buffer |
| **Danish** | Phase 2A (dead files) | Phase 2B (cleanup) | Phase 2D (API client) | Support |
| **Rishab** | Phase 2A (dead files) | Phase 2C (unused queries) | Phase 3 (help Sandy) | Support |
