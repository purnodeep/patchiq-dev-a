# Refactoring Assessment

**Question**: Does PatchIQ need structural refactoring, or is cleanup sufficient?

**Verdict**: Cleanup is sufficient for the POC. There are two areas that warrant targeted refactoring (not rewrites) -- the server `NewRouter` signature and the largest frontend pages -- but neither is blocking. The architecture is fundamentally sound.

---

## 1. Backend Architecture

**Needs refactoring? NO**

The three entry points (`cmd/server/main.go`, `cmd/hub/main.go`, `cmd/agent/main.go`) all follow the same disciplined init sequence: config, logger, otel, signal context, database, event bus, services, servers. Shutdown is ordered and clean with proper error aggregation.

- No god objects. Each domain (deployment, cve, discovery, compliance, workflow) is its own package with focused types.
- The deployment package is well-factored: state machine, evaluator, wave dispatcher, timeout checker, schedule checker, result handler -- all separate files with matching test files (5838 lines total, ~50% tests).
- Handler structs use interface-based querier dependencies (e.g., `DeploymentQuerier`, `EndpointQuerier`), making them testable without database access.
- No circular dependencies. The import hierarchy (shared <-- server/hub/agent, no cross-imports between server/hub/agent) is enforced and clean -- verified with grep, zero violations found.

**One concern**: `cmd/server/main.go` is 808 lines. The `run()` function does a lot of wiring. This is typical for Go service composition and not worth refactoring before POC -- it reads linearly and changing it risks introducing bugs for no user-facing benefit. If it grows past ~1000 lines, extract the wiring into a `wire.go` or `bootstrap.go` file.

## 2. Frontend Architecture

**Needs refactoring? PARTIALLY -- targeted splits for the largest pages**

The three frontends (web, web-hub, web-agent) share identical directory structures:
```
src/{App.tsx, api/, app/, components/, pages/, types/, lib/, test/}
```

Each has properly scoped API hooks (`api/hooks/`), consistent routing, and shared UI from `@patchiq/ui`. The pattern is sound.

**Problem**: Several `web/` page components are too large:
- `EndpointsPage.tsx` -- 2603 lines
- `PatchesPage.tsx` -- 2490 lines
- `PatchDetailPage.tsx` -- 2413 lines
- `CVEsPage.tsx` -- 2311 lines
- `AlertsPage.tsx` -- 2032 lines

Pages above ~800 lines should be split into sub-components (table columns, filter bars, detail panels, dialogs). This is not an architecture problem -- the pattern is right, the files are just too big. Extract components into the same `pages/<resource>/` directory. This can be done incrementally, one page at a time, without touching anything else.

web-hub and web-agent pages are appropriately sized. No action needed there.

## 3. API Layer (Handlers)

**Needs refactoring? NO (with one exception)**

Server handlers: 65 files, 25K total lines across `internal/server/api/v1/`. Files are organized one-per-resource with matching test files. The largest handlers:
- `compliance.go` (1473 lines) -- complex domain, size is justified
- `policies.go` (1206 lines) -- many operations (CRUD + bulk + toggle + evaluation)
- `deployments.go` (1103 lines) -- complex create with waves/targets
- `endpoints.go` (1014 lines) -- many sub-resources (CVEs, packages, patches, deployments)

These are at the upper end but not alarming. Each handler struct has focused dependencies through interfaces. Shared utilities (pagination, response helpers, UUID conversion) are properly extracted into `helpers.go`, `pagination.go`, `response.go`.

Hub handlers: 7K total lines, well-proportioned. No issues.

**One exception -- the `NewRouter` signature**:
```go
func NewRouter(st *store.Store, eventBus domain.EventBus, hubURL, hubAPIKey string,
    startTime time.Time, idempotencyStore idempotency.Store, version string,
    discoveryHandler, deploymentHandler, scheduleHandler, scanScheduler,
    licenseSvc, corsOrigins, notificationHandler, complianceHandler,
    jwtMiddleware, ssoHandler, iamHandler, roleMappingHandler,
    hubSyncAPIHandler, cveMatchInserter, notifByTypeHandler,
    generalSettingsHandler, loginHandler, inviteHandler, alertBackfiller)
```

This function takes 25+ parameters. It should be converted to accept a `RouterDeps` struct. This is a mechanical change (30 minutes) with no risk. Do it.

## 4. Store Layer (sqlc Queries)

**Needs refactoring? NO**

Server: 30 query files, 3172 total lines. Organized one-per-resource, matching the handler structure. Largest files:
- `compliance.sql` (495 lines) -- complex framework evaluation queries
- `endpoints.sql` (408 lines) -- many joins for CVE/patch counts
- `deployments.sql` (301 lines) -- wave/target management

Hub: 16 query files, clean separation. No issues.

The queries are well-scoped. The generated code (17K lines in `sqlcgen/`) is committed and matches the query files. No restructuring needed.

## 5. Shared Packages

**Needs refactoring? NO**

10 packages in `internal/shared/`: config, crypto, domain, idempotency, license, models, otel, protocol, tenant, user. Total: 4658 lines (including tests).

Each package has a single responsibility:
- `tenant/` -- context injection + HTTP middleware
- `domain/` -- event envelope + EventBus interface
- `idempotency/` -- middleware + cache (Valkey + memory)
- `config/` -- YAML loader + config hierarchy
- `crypto/` -- AES + RSA utilities
- `otel/` -- OTLP setup + slog handler

No misplaced code found. The boundaries are clean. All three platforms import from `shared/` without any reverse dependencies.

## 6. Dependency Graph

**Needs refactoring? NO**

Verified with grep across the entire codebase:
- Zero imports from `server` into `hub` or `agent`
- Zero imports from `hub` into `server` or `agent`
- Zero imports from `agent` into `server` or `hub`
- All three import only from `shared/`

The coupling is minimal and correct. Handler structs depend on interfaces (querier interfaces defined per-handler), not concrete implementations. The event bus is the only cross-cutting dependency, and it flows through the `domain.EventBus` interface.

---

## Summary: What To Do

| Area | Action | Effort | Priority |
|------|--------|--------|----------|
| Server `NewRouter` | Convert 25+ params to `RouterDeps` struct | 30 min | Medium |
| Large frontend pages | Split 5 biggest pages into sub-components | 2-3 hours each | Low (cosmetic) |
| Everything else | No structural changes needed | -- | -- |

**Bottom line**: The architecture is solid for a POC-stage product. The codebase follows consistent patterns across all three platforms. Spend time fixing bugs and polishing E2E flows, not refactoring. The two items above are nice-to-haves that reduce cognitive load but don't affect functionality or reliability.
