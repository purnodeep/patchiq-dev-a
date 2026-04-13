# Implementation Plan — Organization-Scoped RBAC / MSP

**Design:** [2026-04-11-org-scoped-rbac-msp.md](2026-04-11-org-scoped-rbac-msp.md) · **ADR:** [025](../adr/025-organization-scoped-rbac-msp.md)

TDD per CLAUDE.md. Every task = failing test first, then minimal production code. File paths are relative to repo root. Next free migration numbers: **server 059**, **hub 019**.

---

## Phase 1 — Server schema + backfill

### 1.1 Migration `059_organizations.sql`
- **Write:** `internal/server/store/migrations/059_organizations.sql`
- **Content:**
  - `CREATE TABLE organizations (id UUID PK, name TEXT, slug TEXT UNIQUE, type TEXT CHECK IN ('direct','msp','reseller'), parent_org_id UUID NULL FK organizations(id), zitadel_org_id TEXT NULL UNIQUE, license_id UUID NULL, created_at, updated_at)`
  - `ALTER TABLE tenants ADD COLUMN organization_id UUID REFERENCES organizations(id)`
  - Backfill: `INSERT INTO organizations (id, name, slug, type) SELECT gen_random_uuid(), name, slug || '-org', 'direct' FROM tenants; UPDATE tenants t SET organization_id = o.id FROM organizations o WHERE o.slug = t.slug || '-org';`
  - `ALTER TABLE tenants ALTER COLUMN organization_id SET NOT NULL`
  - `CREATE TABLE org_user_roles (organization_id, user_id, role_id, assigned_at, PK (organization_id, user_id, role_id))` — NO RLS (org-scoped grants)
  - FK `org_user_roles.role_id → roles(id) ON DELETE CASCADE`
  - FK `org_user_roles.organization_id → organizations(id) ON DELETE CASCADE`
  - Trigger `organizations_set_updated_at` mirroring tenants trigger pattern.
- **Test:** `internal/server/store/migrations_test.go` — add subtest `059_organizations`. Assert: seeds N tenants → after migration, exactly N orgs exist, each tenant has organization_id set, org_user_roles exists and is empty.
- **Down migration:** not authored (goose up-only convention); rollback is `DROP TABLE org_user_roles; ALTER TABLE tenants DROP COLUMN organization_id; DROP TABLE organizations;`.

### 1.2 Seed data
- **Edit:** `scripts/seed/*.sql` — verify seed loads still pass after migration. Seeds set `tenant_id`; migration adds org; no change needed in SQL. Add explicit `organizations` inserts for demo MSP if `seed-demo`.
- **Test:** `make seed-demo` succeeds on a fresh database.

## Phase 2 — Server store + sqlc queries

### 2.1 Queries file
- **Write:** `internal/server/store/queries/organizations.sql`
  - `CreateOrganization :one`
  - `GetOrganizationByID :one`
  - `GetOrganizationBySlug :one`
  - `GetOrganizationByZitadelOrgID :one`
  - `ListOrganizations :many` (paginated)
  - `UpdateOrganization :one`
  - `DeleteOrganization :exec` (soft delete? — for v1, hard delete with CASCADE; TODO(PIQ-TBD) soft delete)
  - `ListTenantsByOrganization :many`
  - `CreateOrgUserRole :exec`
  - `DeleteOrgUserRole :exec`
  - `ListOrgUserRoles :many`
  - `GetUserOrgPermissions :many` (joins org_user_roles → roles → role_permissions)
- **Run:** `make sqlc` → regenerates `internal/server/store/sqlcgen/organizations.sql.go`
- **Test:** `internal/server/store/organizations_test.go` — CRUD tests using testcontainers Postgres.

### 2.2 Store wrapper methods
- **Edit:** `internal/server/store/store.go` — add `ForEachTenant(ctx, orgID, userID string, fn func(ctx context.Context, tenantID string) error) error`. Implementation: query `ListTenantsByOrganization` + filter by user's accessible tenants (join org_user_roles ∪ user_roles), loop invoking `fn` in a per-tenant `BeginTx`.
- **Write:** `internal/server/store/org_scope.go` (new file for the helper + its unit tests).
- **Test:** `internal/server/store/org_scope_test.go` — seed 3 tenants under one org, verify fn called 3 times; verify per-tenant RLS context by having fn read a tenant-scoped table and assert counts match that tenant only.

## Phase 3 — Organization context package

### 3.1 Context helpers
- **Write:** `internal/shared/organization/context.go` — mirrors `internal/shared/tenant/context.go`:
  - `type ctxKey struct{}`
  - `WithOrgID(ctx, id) context.Context`
  - `OrgIDFromContext(ctx) (string, bool)`
  - `RequireOrgID(ctx) (string, error)` — returns `ErrMissingOrgID`
  - `MustOrgID(ctx) string`
- **Test:** `internal/shared/organization/context_test.go` — table-driven, mirrors tenant tests.

### 3.2 HTTP middleware
- **Write:** `internal/shared/organization/middleware.go` — extracts `X-Organization-ID` header when present (optional — org is normally derived from JWT, but explicit header is used for MSP dashboard calls that target the org, not a specific tenant). Validates UUID, injects into context.
- **Test:** `internal/shared/organization/middleware_test.go` — missing header (pass through), invalid UUID (400), valid header (context populated).

## Phase 4 — RBAC extension

### 4.1 Evaluator org-scoped grants
- **Edit:** `internal/server/auth/evaluator.go`:
  - Add method `GetUserOrgPermissions(ctx, orgID, userID) ([]Permission, error)` on the `PermissionStore` interface.
  - In `HasPermission`: extract `orgID` from context, check org-scoped grants first, then fall back to tenant-scoped grants.
- **Edit:** `internal/server/auth/permission_store.go` (or wherever `PermissionStore` is implemented) — add sqlcgen wiring for `GetUserOrgPermissions`.
- **Test:** `internal/server/auth/evaluator_test.go` — add cases:
  - User has org-scoped `*:*:*` via MSP Admin → permission granted across any tenant in the org.
  - User has tenant-scoped `endpoints:read:*` only → granted in that tenant, denied in sibling tenant under same org.
  - User has no grants → denied.
  - Missing orgID in context but valid tenantID → existing behavior (tenant-only check).

### 4.2 MSP preset roles
- **Edit:** `internal/server/auth/roles.go`:
  - Remove the `// NOTE: MSP Admin role removed ...` comment.
  - Append three org-scoped presets: `MSP Admin` (`*:*:*`), `MSP Technician` (endpoints read/update, deployments create/execute, patches read, reports read — no RBAC, no billing), `MSP Auditor` (read-only on everything including audit).
  - Add a `Scope` field to `RoleTemplate` (values `"tenant"` or `"org"`) to mark which presets belong in the platform tenant vs a regular tenant.
- **Test:** `internal/server/auth/roles_test.go` — assert `PresetRoles()` returns the three new org-scoped roles with correct scope and permissions.

### 4.3 Platform tenant seeding
- **Write:** `internal/server/auth/platform_tenant.go` — function `EnsurePlatformTenant(ctx, store, orgID) (tenantID string, err error)` that creates (if missing) the hidden platform tenant for an org and seeds the org-scoped preset roles into it. Idempotent.
- **Test:** `internal/server/auth/platform_tenant_test.go` — call twice, assert single tenant created, roles seeded once.

## Phase 5 — Auth / JWT wiring

### 5.1 Org resolver
- **Write:** `internal/server/auth/org_resolver.go` — `type OrgResolver interface { ByZitadelOrgID(ctx, zitadelOrgID) (*Organization, error); UserAccessibleTenants(ctx, orgID, userID) ([]Tenant, error) }`. sqlcgen-backed impl.
- **Test:** `internal/server/auth/org_resolver_test.go`.

### 5.2 JWT middleware rewrite
- **Edit:** `internal/server/auth/jwt.go` — around lines 345-355 (the `DefaultTenantID` fallback):
  - Replace the fallback logic: extract `zitadelOrgID` from claims → `OrgResolver.ByZitadelOrgID` → load accessible tenants → select active tenant via `X-Tenant-ID` header (must be in accessible list) or default to first accessible → inject both org and tenant into context.
  - Keep `DefaultTenantID` config field for single-tenant dev mode but rename semantics: now `DefaultOrgSlug` — resolved to an org at startup.
- **Edit:** `internal/server/auth/config.go` (or wherever `JWTConfig` is defined) — add `DefaultOrgSlug string`, deprecate `DefaultTenantID` with a comment.
- **Test:** `internal/server/auth/jwt_test.go`:
  - JWT with known zitadel_org_id → org + default-active-tenant injected.
  - JWT with `X-Tenant-ID` header matching an accessible tenant → that tenant active.
  - JWT with `X-Tenant-ID` header NOT in accessible list → 403.
  - JWT with unknown zitadel_org_id and no default → 401.
  - JWT with empty org claim + DefaultOrgSlug set → falls back to default org.

### 5.3 Session active-tenant endpoint
- **Write:** `internal/server/api/v1/session.go` — `POST /api/v1/session/active-tenant` body `{tenant_id}`. Validates tenant is in user's accessible list, updates session cookie (re-signs JWT with new active tenant claim or stores in server-side session state).
- **Test:** handler test — happy path + forbidden path.

## Phase 6 — Organizations REST API

### 6.1 Handlers
- **Write:** `internal/server/api/v1/organizations.go` — pattern matches `roles.go`:
  - `GET /api/v1/organizations` — list (platform admins only)
  - `GET /api/v1/organizations/{id}` — get by ID (org members only)
  - `GET /api/v1/organizations/{id}/tenants` — list child tenants
  - `POST /api/v1/organizations/{id}/tenants` — provision new child tenant (MSP Admin perm)
  - `GET /api/v1/organizations/{id}/dashboard` — cross-tenant aggregations via `ForEachTenant`
  - `POST /api/v1/organizations/{id}/users/{user_id}/roles` — assign org-scoped role
  - `DELETE /api/v1/organizations/{id}/users/{user_id}/roles/{role_id}`
- **Edit:** `internal/server/api/router.go` — register handler + middleware chain (JWT → org → RBAC permission check).
- **Edit:** `api/openapi/openapi.yaml` — add schemas and paths. Regenerate frontend types: `make api-client` + `cd web && pnpm run gen:api`.
- **Test:** `internal/server/api/v1/organizations_test.go` — table-driven handler tests for each endpoint.

### 6.2 Domain events
- **Edit:** `internal/shared/domain/events.go` (or wherever event types live) — add: `OrganizationCreated`, `OrganizationUpdated`, `OrganizationDeleted`, `OrgUserRoleAssigned`, `OrgUserRoleRevoked`, `TenantProvisionedUnderOrg`.
- **Edit:** handlers emit these via `eventBus.Publish` (matches existing tenant-write pattern).
- **Test:** handler tests assert event published.

## Phase 7 — Hub umbrella licensing

### 7.1 Migration `019_organizations_and_umbrella_license.sql`
- **Write:** `internal/hub/store/migrations/019_organizations_and_umbrella_license.sql`
  - `CREATE TABLE organizations (...)` — same schema as server side
  - `ALTER TABLE tenants ADD COLUMN organization_id UUID REFERENCES organizations(id)` + backfill (mirror server 059)
  - `ALTER TABLE licenses ADD COLUMN organization_id UUID NULL REFERENCES organizations(id)`
- **Test:** hub migration test mirrors server 059 test.

### 7.2 License validator
- **Edit:** `internal/hub/license/validator.go` — add org path:
  - `Validate(ctx, orgID)` → sums `clients.endpoint_count` across all tenants in the org → compares to the umbrella license's `max_endpoints`.
  - Existing per-tenant validation path unchanged.
- **Test:** `internal/hub/license/validator_test.go` — cases: umbrella license under-quota, at-quota, over-quota.

### 7.3 Async enforcement
- **Write:** `internal/hub/workers/org_license_check_job.go` — River job triggered by `ClientEndpointCountUpdated` event. Recomputes org-wide usage, emits `OrgLicenseOverQuota` event on breach.
- **Test:** worker test.

## Phase 8 — Frontend

### 8.1 Regenerate API types
- **Run:** `cd web && pnpm run gen:api` (or equivalent). New `organizations.*` types land in `web/src/api/types.ts`.

### 8.2 AuthContext
- **Edit:** `web/src/app/auth/AuthContext.tsx`:
  - Add fields: `organization`, `active_tenant_id`, `accessible_tenants`, `org_permissions`, `tenant_permissions`.
  - `can(resource, action)` considers both org and tenant permissions.
- **Edit:** `web/src/app/auth/useAuth.ts` — load org/tenant metadata from `GET /api/v1/session/me` (new endpoint) on mount.
- **Write:** `internal/server/api/v1/session.go` — `GET /api/v1/session/me` returns the AuthUser shape.
- **Test:** `web/src/app/auth/__tests__/AuthContext.test.tsx`.

### 8.3 API client tenant middleware
- **Edit:** `web/src/api/client.ts` — add a middleware that reads the active tenant ID from a module-level store (updated by AuthContext) and injects `X-Tenant-ID` on every request.
- **Write:** `web/src/api/activeTenantStore.ts` — tiny module-level store with subscribe/get/set (avoids a Zustand dep for a single value).
- **Test:** vitest — middleware injects correct header.

### 8.4 TenantSwitcher
- **Write:** `web/src/app/layout/TenantSwitcher.tsx` — shadcn DropdownMenu, shows `accessible_tenants`, selects via `POST /api/v1/session/active-tenant` → updates AuthContext → invalidates all TanStack Query caches → user sees new tenant's data.
- **Edit:** `web/src/app/layout/TopBar.tsx` — render `<TenantSwitcher />` when `accessible_tenants.length > 1`.
- **Test:** `web/src/app/layout/__tests__/TenantSwitcher.test.tsx`.

### 8.5 MSP Dashboard page
- **Write:** `web/src/pages/msp/MspDashboardPage.tsx` — aggregate cards (total endpoints, compliance %, deployment success, license utilization) + per-tenant breakdown table (TanStack Table). Data from `useMspDashboard()` hook.
- **Write:** `web/src/api/hooks/useMspDashboard.ts` — TanStack Query hook against `GET /api/v1/organizations/{id}/dashboard`.
- **Edit:** `web/src/app/routes.tsx` — add `/msp` route gated on `organization.type === 'msp'`.
- **Edit:** `web/src/app/layout/AppSidebar.tsx` — add MSP link (conditional).
- **Test:** `web/src/pages/msp/__tests__/MspDashboardPage.test.tsx`.

### 8.6 Organizations settings page
- **Write:** `web/src/pages/settings/OrganizationSettingsPage.tsx` — child tenants list, invite MSP users, umbrella license usage, convert direct→msp button.
- **Edit:** `web/src/app/routes.tsx` — add `/settings/organization`.
- **Test:** `web/src/pages/settings/__tests__/OrganizationSettingsPage.test.tsx`.

## Phase 9 — E2E verification

- **Write:** `internal/server/integration/msp_flow_test.go`:
  1. Spin up testcontainers Postgres + migrate.
  2. Create MSP org with 3 child tenants.
  3. Create user, assign MSP Admin at org scope.
  4. Assert `Evaluator.HasPermission` grants `endpoints:read:*` across all 3 tenants.
  5. Assert `ForEachTenant` returns expected aggregate.
  6. Assert `X-Tenant-ID` switching enforced.
  7. Assert `X-Tenant-ID` with non-accessible tenant → 403.
- **Write:** `internal/server/integration/rls_regression_test.go`:
  1. Set `app.current_tenant_id = A`.
  2. Query a tenant-scoped table joined across tenants.
  3. Assert tenant B rows NOT returned, even with an active org context.
- **Run:** `make lint && make test && make test-integration && make lint-frontend`.
- **Manual:** `make dev` → open web UI, create demo MSP, verify tenant switcher + MSP dashboard render correctly.

## Phase 10 — Review & Ship

- **Run:** `/review-pr all parallel` — fix all Critical/Important findings.
- **Run:** `/commit-push-pr` — base `dev-a`, title `feat(rbac,msp): add organization-scoped RBAC and MSP model`.
- **Update:** CLAUDE.md — note new `organization` context package and `ForEachTenant` helper under "Shared Packages".

---

## Risks & Watch-list

- **Migration on a large tenants table** — backfill is O(N) inserts. Wrap in a single transaction; test on a fixture with 10k tenants.
- **RLS regression** — the rls_regression_test.go is non-negotiable; must fail CI if cross-tenant reads leak.
- **Frontend cache staleness on tenant switch** — `queryClient.clear()` on switch to avoid stale data; verify in e2e.
- **Zitadel org ID duplication** between `organizations.zitadel_org_id` and `iam_settings.zitadel_org_id` — document deprecation in code comments and schedule removal ADR.
- **Protected files touched**: migrations, `internal/shared/*`, `internal/server/auth/*`, RBAC tables. PR must get core dev review before merge.
