# Audit 10: Test Coverage

**Date**: 2026-04-09
**Branch**: dev-a
**Auditor**: Claude Opus 4.6

---

## Executive Summary

PatchIQ has **surprisingly strong test coverage** for a project at this maturity stage. The backend has 245 test files containing ~1,175+ test functions across all four platforms (server, hub, agent, shared). The frontend has 89 test files with ~419 test cases. Two solid integration tests exercise the full core loop and offline agent behavior using testcontainers. The main gaps are in frontend page coverage (settings, compliance, alerts, admin pages), some shared packages, and the absence of E2E and load tests.

**Overall severity: Important** -- the foundation is excellent but several critical code paths lack tests.

---

## 1. Go Backend Test Coverage

### 1.1 Server (`internal/server/`) -- 151 production files, 137 test files

| Package | Test Files | Test Functions | Quality | Notes |
|---------|-----------|---------------|---------|-------|
| `api/v1` | 30 | 177 | High | Table-driven, edge cases, error paths, exports |
| `auth` | 13 | 51 | High | Middleware, JWT, RBAC, SSO, sessions, rate limiting, role mapping |
| `compliance` | 4 | 24 | Good | Evaluator, frameworks, scorer, service |
| `cve` | 11 | 41 | High | Bulk, client, correlator, hub client, NVD, KEV, matcher, scorer, integration |
| `deployment` | 14 | 79 | High | State machine (17 tests), waves, timeout, scheduler, evaluator, integration |
| `discovery` | 8 | 24 | Good | APT, YUM, HTTP, service, job, integration |
| `events` | 5 | 16 | Good | Alert rules, alert subscriber, notification handler, topics |
| `grpc` | 11 | 33 | High | Enroll, heartbeat, sync inbox/outbox, interceptors, integration |
| `license` | 3 | 15 | High | Validator (valid/expired/grace/clock-drift/tampered), middleware, service |
| `notify` | 4 | 13 | Good | Sender, triggers, worker |
| `policy` | 3 | 23 | Good | Evaluator, scheduler, strategy |
| `repo` | 2 | 8 | Good | Cache, handler |
| `store` | 4 | 14 | Good | DB setup, inventory, agent registrations, vulnerability (testcontainers) |
| `tags` | 1 | 13 | Good | Evaluator |
| `workers` | 3 | 12 | Good | Audit retention, catalog sync, user sync |
| `workflow` | 7 | 31 | Good | Executor, graph, model, templates, validate, worker, integration |
| `workflow/handlers` | 14 | 42 | High | All 14 handler types tested individually |
| `mcp` | 0 | 0 | N/A | Empty package (`.gitkeep` only) |

**Untested server packages**:
- `internal/server/` root (1 file) -- Minor, likely just package doc
- `internal/server/api/` (1 file) -- Minor, likely router setup
- `internal/server/store/sqlcgen/` (32 files) -- N/A, generated code

### 1.2 Hub (`internal/hub/`) -- 40 production files, 31 test files

| Package | Test Files | Test Functions | Quality | Notes |
|---------|-----------|---------------|---------|-------|
| `api/v1` | 11 | 72 | High | Catalog, clients, feeds, dashboard, licenses, sync, settings, analytics |
| `auth` | 4 | 19 | Good | JWT, login, session, Zitadel |
| `catalog` | 4 | 16 | Good | Fetcher, APT fetcher, MinIO, pipeline |
| `events` | 1 | 4 | Minimal | Basic event tests only |
| `feeds` | 7 | 22 | Good | All 6 feeds tested + general feed tests |
| `license` | 1 | 6 | Good | Generate |
| `store` | 2 | 11 | Good | DB setup (testcontainers), store ops |
| `workers` | 1 | 3 | Minimal | Feed sync job only |

**Untested hub packages**:
- `internal/hub/api/` (1 file) -- Minor, router setup
- `internal/hub/store/sqlcgen/` (18 files) -- N/A, generated code

### 1.3 Agent (`internal/agent/`) -- 118 production files, 62 test files

| Package | Test Files | Test Functions | Quality | Notes |
|---------|-----------|---------------|---------|-------|
| `api` | 7 | 14 | Good | Handler, history, logs, patches, response, settings, status |
| `comms` | 13 | 58 | High | Certgen, client, DB, enroll, heartbeat, inbox, outbox, sync, throttle, integration |
| `executor` | 1 | 8 | Good | Executor tests |
| `hooks` | 1 | 4 | Good | PowerShell hooks |
| `inventory` | 18 | 126 | High | All collectors: APT, YUM, WUA, Hotfix, Homebrew, macOS, hardware, services, metrics, classify |
| `patcher` | 12 | 75 | High | All patchers: APT, YUM, WUA, MSI, MSIX, Homebrew, macOS, download, executor, installer |
| `store` | 6 | 25 | Good | DB, patches, history, logs, rollback, status |
| `system` | 1 | 8 | Good | System info |
| root | 4 | varies | Good | Export, registry, runner, service (linux/darwin), settings watcher |

### 1.4 Shared (`internal/shared/`) -- 26 production files, 20 test files

| Package | Test Files | Test Functions | Quality | Notes |
|---------|-----------|---------------|---------|-------|
| `config` | 2 | 5 | Good | Hierarchy, loader |
| `crypto` | 2 | 7 | Good | AES, RSA |
| `domain` | 1 | 4 | Good | Events |
| `idempotency` | 5 | 20 | High | Cache, middleware, command, optimistic, integration (Valkey) |
| `license` | 1 | 4 | Good | Tiers |
| `otel` | 5 | 14 | Good | Context, gRPC, init, middleware, slog |
| `protocol` | 1 | 1 | Minimal | Version only |
| `tenant` | 2 | 7 | Good | Context, middleware (valid/missing/invalid UUID) |
| `user` | 1 | 4 | Good | Context |
| `models` | 0 | 0 | N/A | Type definitions only, no logic to test |

---

## 2. Frontend Test Coverage

### 2.1 Web (Patch Manager) -- 89 test files total across all apps

**Tested pages/components** (71 test files):
- Dashboard: 18 test files (page, widgets, charts, command palette, alerts, timeline)
- Audit: 5 tests (page, activity stream, timeline view, event helpers, CSV export)
- CVEs: 2 tests (list page, detail page)
- Deployments: 2 tests (list page, detail page)
- Endpoints: 3 tests (list page, detail page, deriveStatus logic)
- Groups: 3 tests (page, create dialog, edit dialog)
- Notifications: 3 tests (page, history tab, preferences tab)
- Patches: 3 tests (page, detail page, deployment modal)
- Policies: 3 tests (page, create page, detail page)
- Workflows: 2 tests (page, editor)
- Workflow builder: 10 tests (canvas, DAG validation, edge validation, execution, hooks, nodes, palette, panels)
- Components: 10 tests (CVSSScore, DeploymentStatusBadge, KEVBadge, MonospaceOutput, PolicyModeBadge, ProgressBar, SeverityBadge, StatusBadge, TagInput)
- DataTable: 3 tests (DataTable, pagination, search)
- Auth: 1 test (AuthLayout)
- Login: 3 tests (login, register, forgot password)
- Lib: 2 tests (format, time utilities)

**Untested web pages** (Important):
| Page/Directory | Files | Severity |
|---------------|-------|----------|
| `pages/settings/` | 10 .tsx | **Critical** -- IAM, general settings, role mapping |
| `pages/compliance/` + `compliance/components/` | 20 .tsx | **Critical** -- CIS, PCI-DSS, HIPAA, NIST, ISO 27001, SOC 2 |
| `pages/alerts/` | 4 .tsx | Important |
| `pages/admin/roles/` + `admin/roles/components/` | 4 .tsx | Important -- RBAC management |
| `pages/admin/users/` | 1 .tsx | Important |
| `pages/endpoints/tabs/` | 8 .tsx | Important -- endpoint detail tabs |
| `pages/policies/tabs/` + `policies/components/` | 7 .tsx | Minor |
| `pages/deployments/components/` | 5 .tsx | Minor |
| `pages/tags/` | 1 .tsx | Minor |
| `pages/agent-downloads/` | 1 .tsx | Minor |
| `pages/preview/` | 1 .tsx | Minor |
| `api/hooks/` | 24 files, 0 tests | **Critical** -- all data fetching hooks untested |
| `app/auth/AuthContext.tsx` | 1 file | **Critical** -- auth context untested |

### 2.2 Web-Hub (Hub Manager) -- 10 test files

**Tested**: App, computeStatus, useClient hook, useLicense hook, ClientsPage, DashboardPage, FeedsPage, LicensesPage, AuthContext, LoginPage

**Untested** (Important):
| Page/Directory | Files | Severity |
|---------------|-------|----------|
| `pages/settings/` | 10 .tsx | Important |
| `pages/catalog/` | 3 .tsx | Important |
| `pages/deployments/` | 1 .tsx | Minor |
| `pages/feeds/` | 3 .tsx | Minor (list page tested, detail/components not) |
| `pages/dashboard/` | 7 .tsx | Minor (page tested, widgets not) |
| `pages/clients/` | 2 .tsx | Minor (page tested, detail not) |
| `pages/licenses/` | 3 .tsx | Minor (page tested, detail not) |

### 2.3 Web-Agent -- 6 test files

**Tested**: App, HistoryPage, LogsPage, PendingPatchesPage, SettingsPage, StatusPage

**Untested**:
| Page | Severity |
|------|----------|
| `pages/hardware/` | Minor |
| `pages/software/` | Minor |
| `pages/services/` | Minor |

### 2.4 Shared UI (`packages/ui/`) -- 15 test files

**Tested**: 10 custom components (DotMosaic, EmptyState, ErrorState, MonoTag, PageHeader, RingGauge, SeverityText, SkeletonCard, StatCard, ThemeConfigurator), theme provider, tokens, Switch, render utility.

**Untested**: 22 shadcn/ui primitives -- acceptable since these are upstream vendor components.

---

## 3. Integration Tests

### 3.1 Full E2E Integration (`test/integration/`)

| Test | Lines | What It Covers | Quality |
|------|-------|---------------|---------|
| `core_loop_test.go` | 404 | 10-step core loop: PostgreSQL setup, gRPC server + agent container, enrollment, inventory sync, patch discovery, CVE ingestion, policy creation, deployment dispatch, command delivery, audit verification | **Excellent** |
| `offline_test.go` | 136 | Offline-first pattern: agent starts without server, queues to outbox, server comes online, agent reconnects and drains | **Excellent** |

### 3.2 Test Infrastructure (`test/integration/testutil/`)

| File | Purpose |
|------|---------|
| `postgres.go` | Testcontainers PostgreSQL 16, migrations, app role pool |
| `agent_container.go` | Cross-compiles agent binary, runs in Docker container |
| `grpc_server.go` | Insecure gRPC server for integration tests |
| `assertions.go` | DB assertion helpers (package exists, CVE exists, patch exists) |
| `certs.go` | TLS certificate generation |
| `fixtures.go` | NVD fixture writing |
| `Dockerfile.agent` | Agent container image |

### 3.3 Package-Level Integration Tests (build tag: `integration`)

| File | Uses Testcontainers | Purpose |
|------|-------------------|---------|
| `server/grpc/integration_test.go` | Yes | Full gRPC flow |
| `server/grpc/inventory_integration_test.go` | Yes | Inventory sync |
| `server/deployment/integration_test.go` | Likely | Deployment lifecycle |
| `server/discovery/integration_test.go` | Likely | Discovery flow |
| `server/cve/integration_test.go` | Likely | CVE pipeline |
| `server/store/db_test.go` | Yes | Store operations |
| `server/events/events_test.go` | Yes | Event bus |
| `hub/store/db_test.go` | Yes | Hub store |
| `hub/store/store_test.go` | Yes | Hub store queries |
| `hub/events/events_test.go` | Yes | Hub event bus |
| `shared/idempotency/cache_integration_test.go` | Yes (Valkey) | Idempotency cache |

### 3.4 Missing Integration Tests

| Gap | Severity |
|-----|----------|
| `test/e2e/` | **Critical** -- empty `.gitkeep`, no browser/API E2E tests |
| `test/load/` | **Critical** -- empty `.gitkeep`, no load/performance tests |
| Hub-to-server sync integration | Important |
| Multi-tenant isolation integration | **Critical** |
| RBAC permission matrix integration | Important |
| License enforcement E2E | Important |

---

## 4. Test Quality Assessment

### Strengths

1. **Table-driven tests throughout** -- the codebase consistently uses Go's `t.Run` + table-driven pattern. Example: `auth/middleware_test.go` tests valid, max-length, missing, too-long, and control-character user IDs.

2. **Fake/stub pattern over mocks** -- tests use hand-written fakes (e.g., `fakeEventBus`, `fakeStartQuerier`) rather than mock frameworks. This produces more readable, maintainable tests.

3. **Edge case coverage is good** -- license validator tests cover valid, expired, grace period, clock drift, invalid signature, and tampered payload. State machine tests cover all transitions including error paths.

4. **Integration tests are production-quality** -- the core loop test exercises enrollment through deployment in 10 steps with real PostgreSQL and a real agent binary in a container. The offline test validates the store-and-forward pattern.

5. **Test helpers are well-organized** -- `test/integration/testutil/` provides reusable PostgreSQL, gRPC, assertion, and fixture helpers.

6. **Frontend tests are meaningful** -- not just snapshot tests. They test user interactions, loading states, error states, and data rendering.

7. **Auth test code is proportional** -- 2,380 lines of tests for 2,194 lines of production auth code (~1.08:1 ratio).

### Weaknesses

1. **No test coverage measurement** -- no `go test -coverprofile` in CI, no coverage thresholds enforced.

2. **Frontend API hooks completely untested** -- 24 hook files with 0 dedicated tests. These are the data-fetching layer for the entire UI.

3. **AuthContext untested** -- the auth provider (Zitadel OIDC integration) in the frontend has no tests.

4. **Some test files have helper-only content** -- files exist but the grep for `func Test` found 0 functions initially (though re-checking showed they do have tests, the count was a grep methodology issue).

5. **No mutation testing** -- no evidence of mutation testing tools (e.g., `go-mutesting`) to validate test effectiveness.

---

## 5. Coverage Matrix Summary

### Go Backend (by priority to add tests)

| Package | Has Tests | Test Count | Quality | Priority |
|---------|-----------|-----------|---------|----------|
| `server/mcp` | No | 0 | N/A | Minor (empty package) |
| All `sqlcgen/` | No | 0 | N/A | None (generated) |
| `server/api` (router) | No | 0 | N/A | Minor |
| **Everything else** | **Yes** | **1,175+** | **Good-High** | **Covered** |

### Frontend (by priority to add tests)

| Area | Has Tests | Test Count | Quality | Priority |
|------|-----------|-----------|---------|----------|
| `web/api/hooks/` (24 files) | No | 0 | N/A | **Critical** |
| `web/app/auth/AuthContext.tsx` | No | 0 | N/A | **Critical** |
| `web/pages/settings/` (10 files) | No | 0 | N/A | **Critical** |
| `web/pages/compliance/` (20 files) | No | 0 | N/A | **Critical** |
| `web/pages/alerts/` (4 files) | No | 0 | N/A | Important |
| `web/pages/admin/` (5 files) | No | 0 | N/A | Important |
| `web/pages/endpoints/tabs/` (8 files) | No | 0 | N/A | Important |
| `web-hub/pages/settings/` (10 files) | No | 0 | N/A | Important |
| `web-hub/pages/catalog/` (3 files) | No | 0 | N/A | Important |
| Dashboard, audit, CVEs, etc. | Yes | 71 files | Good | Covered |
| Shared UI (`packages/ui/`) | Yes | 15 files | Good | Covered |

### Integration / E2E / Load

| Area | Has Tests | Quality | Priority |
|------|-----------|---------|----------|
| Core loop integration | Yes | Excellent | Covered |
| Offline behavior integration | Yes | Excellent | Covered |
| Package-level integration (11 files) | Yes | Good | Covered |
| Multi-tenant isolation E2E | No | N/A | **Critical** |
| E2E browser tests | No | N/A | **Critical** |
| Load/performance tests | No | N/A | **Critical** |
| Hub-server sync integration | No | N/A | Important |
| RBAC permission matrix | No | N/A | Important |

---

## 6. Findings by Severity

### Critical

| # | Finding | Impact |
|---|---------|--------|
| C1 | **No E2E tests** (`test/e2e/` is empty) | Cannot validate user-facing flows before client deployment |
| C2 | **No load tests** (`test/load/` is empty) | No performance baseline; unknown breaking point for POC |
| C3 | **No multi-tenant isolation integration test** | Tenant data leakage is the #1 security risk; only unit-level middleware tests exist |
| C4 | **Frontend API hooks untested** (24 files, 0 tests) | Data fetching layer has no safety net; refactors could silently break all pages |
| C5 | **Frontend AuthContext untested** | Auth flow (OIDC, session, token refresh) has no automated validation |
| C6 | **No coverage measurement in CI** | Cannot track coverage trends or enforce minimums |

### Important

| # | Finding | Impact |
|---|---------|--------|
| I1 | Settings pages untested (web: 10 files, web-hub: 10 files) | IAM configuration changes could break silently |
| I2 | Compliance pages untested (20 files) | 6 framework views with no test coverage |
| I3 | Alerts pages untested (4 files) | Alert management UI unvalidated |
| I4 | Admin/RBAC pages untested (5 files) | Role and user management UI unvalidated |
| I5 | Hub-to-server catalog sync not integration-tested | Sync failures would go undetected |
| I6 | RBAC permission matrix not integration-tested | Permission misconfigurations only caught in production |

### Minor

| # | Finding | Impact |
|---|---------|--------|
| M1 | Tags, agent-downloads, preview pages untested | Low-complexity pages |
| M2 | Some deployment/policy sub-components untested | Parent pages are tested |
| M3 | Web-agent hardware/software/services pages untested | Simple read-only views |
| M4 | `shared/protocol` has only 1 test function | Small package, low risk |

---

## 7. Recommendations (Prioritized)

1. **Add `go test -coverprofile` to CI** and set a floor (suggest 60% initially, raise to 75% over time). This is the single highest-leverage improvement.

2. **Write multi-tenant isolation integration test** -- create two tenants, insert data for each, verify queries with tenant A context never return tenant B data. This directly de-risks the POC deployment.

3. **Add frontend API hook tests** -- use MSW (Mock Service Worker) to test the 24 hooks. This protects the entire data layer.

4. **Add AuthContext test** -- mock Zitadel OIDC responses, test login flow, token refresh, and session expiry.

5. **Add at least smoke-level E2E tests** using Playwright or Cypress for the top 5 user flows: login, view dashboard, create deployment, view compliance, manage settings.

6. **Add basic load test** using k6 or vegeta: 100 concurrent agents heartbeating, 50 concurrent API users. Establishes a performance baseline before POC.

7. **Test settings and compliance pages** -- these are client-facing configuration surfaces.
