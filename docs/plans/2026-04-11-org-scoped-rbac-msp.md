# Organization-Scoped RBAC for MSP Model

**Status:** Draft · **Date:** 2026-04-11 · **Owner:** Heramb · **ADR:** [025-organization-scoped-rbac-msp](../adr/025-organization-scoped-rbac-msp.md)

## Problem

PatchIQ currently has flat tenant isolation. One login = one tenant. There is no concept of an organization that owns multiple tenants.

For a Managed Service Provider (MSP) go-to-market motion this is disqualifying. An MSP operator needs:

- **One pane of glass** across all client tenants
- **Technician roles** that span clients (a 5-tech MSP should not need 50 × 5 user provisioning operations)
- **Umbrella licensing** (one contract, N clients, endpoint totals enforced across the fleet)
- **Aggregated SLA / compliance reports** per MSP contract
- **Bulk client provisioning** APIs

The existing code explicitly acknowledges this gap: `internal/server/auth/roles.go:90-91` documents that the MSP Admin role was removed "requires tenant-scope enforcement not yet implemented."

`iam_settings.zitadel_org_id` exists but is never populated. Zitadel's native organizations claim (`urn:zitadel:iam:org:id`) is extracted in `internal/server/auth/jwt.go:96-99` but falls back to a hardcoded `DefaultTenantID`. The hooks are there; the hierarchy is not.

## Goals

1. Add an `organizations` layer above `tenants` without breaking the existing single-tenant deployments.
2. Keep PostgreSQL RLS as the hard isolation boundary (RLS is tenant-scoped, unchanged).
3. Support MSP workflows: cross-tenant roles, tenant switcher UI, aggregated dashboards, umbrella licensing, bulk provisioning.
4. Bind Zitadel organizations to PatchIQ organizations (not tenants).
5. Zero-risk backfill: every existing tenant lands in its own default organization with identical behavior.

## Non-Goals

- Changing RLS to be org-scoped. Tenant stays the isolation boundary. Cross-tenant reads happen in application code by iterating tenants the user has access to.
- Multi-level reseller hierarchies (MSP → sub-MSP → client). Deferred to a future phase; the schema supports one level (`parent_org_id` is added but initially always NULL).
- Migrating away from `X-Tenant-ID` header. It remains the active-tenant selector; we add validation that the user actually has access to it.
- Cross-tenant write operations. MSPs can read across tenants; any write still targets a specific tenant.

## Architecture

### Entity model

```
organizations (NEW)
  id              UUID PK
  name            TEXT
  slug            TEXT UNIQUE
  type            TEXT  ('direct' | 'msp' | 'reseller')
  parent_org_id   UUID NULL FK organizations(id)   -- reserved, NULL in v1
  zitadel_org_id  TEXT NULL UNIQUE                 -- maps JWT urn:zitadel:iam:org:id
  license_id      UUID NULL FK hub.licenses(id)    -- umbrella license (hub-side FK, logical only on server)
  created_at, updated_at

tenants (MODIFIED — add FK)
  id              UUID PK                          -- unchanged
  organization_id UUID NOT NULL FK organizations(id)   -- NEW
  name, slug, license_id, created_at, updated_at

org_user_roles (NEW)                               -- parallel to user_roles, but org-scoped
  organization_id UUID FK organizations(id)
  user_id         TEXT
  role_id         UUID FK roles(id)                -- role lives in a "platform org" tenant (see below)
  assigned_at     TIMESTAMPTZ
  PRIMARY KEY (organization_id, user_id, role_id)
```

**Why a new `org_user_roles` table instead of adding `organization_id` to `user_roles`:** `user_roles` has RLS on `tenant_id`. Mixing org-scoped grants into that table would require weakening RLS or adding a sentinel tenant_id, both of which are worse than a parallel table.

**Where do org-scoped roles live?** `roles` is tenant-scoped (RLS). We introduce a **platform organization tenant** per organization: a dedicated tenant with `slug` = `org:<org_slug>:platform`, used solely to host org-scoped role definitions (MSP Admin, MSP Technician, MSP Auditor). The `roles` table stays unchanged; the RLS policy still works; org-scoped grants reference roles in this dedicated platform tenant. The platform tenant is hidden from the tenant switcher UI.

### Org types

- `direct` — a standard customer org with one child tenant. Single-tenant deployments backfill to this.
- `msp` — MSP operator with N child tenants. Supports tenant switcher, cross-tenant dashboards, org-scoped RBAC, umbrella licensing.
- `reseller` — reserved for future multi-level hierarchies.

### Zitadel binding

Zitadel's native `organizations` concept becomes the authoritative IdP mapping. JWT claim `urn:zitadel:iam:org:id` → lookup `organizations.zitadel_org_id` → resolve `organization_id` → load user's accessible tenants for that org → active tenant selected via `X-Tenant-ID` header (must be in the accessible list).

**Flow rewrite in `internal/server/auth/jwt.go`:**

```go
// Current (lines 345-355):
orgID := customClaims.OrgID
if orgID == "" && cfg.DefaultTenantID != "" {
    orgID = cfg.DefaultTenantID  // treat DefaultTenantID as tenant UUID — WRONG in MSP world
}

// New:
zitadelOrgID := customClaims.OrgID
org, err := orgResolver.ByZitadelOrgID(ctx, zitadelOrgID)  // NEW: resolver interface
if err != nil {
    if errors.Is(err, ErrOrgNotFound) && cfg.DefaultOrgID != "" {
        org = fallbackDefaultOrg  // single-tenant deployments
    } else {
        return unauthorized(w, "unknown organization")
    }
}
accessibleTenants := orgResolver.UserTenants(ctx, org.ID, userID)  // from org_user_roles ∪ user_roles
activeTenant := selectActiveTenant(r.Header.Get("X-Tenant-ID"), accessibleTenants)
ctx = tenant.WithTenantID(ctx, activeTenant)
ctx = org_ctx.WithOrgID(ctx, org.ID)
```

New package: `internal/shared/organization/` with:
- `context.go` — `WithOrgID`, `OrgIDFromContext`, `RequireOrgID` (mirrors `internal/shared/tenant/context.go`)
- `middleware.go` — extracts `X-Organization-ID` when present for explicit org selection (MSP dashboard calls that target the org, not a specific tenant)

### RBAC extension

The existing `Evaluator.HasPermission` (`internal/server/auth/evaluator.go:38-61`) becomes:

```go
func (e *Evaluator) HasPermission(ctx context.Context, required Permission) (bool, error) {
    tenantID, _ := tenant.TenantIDFromContext(ctx)
    orgID, _    := organization.OrgIDFromContext(ctx)
    userID, _   := user.UserIDFromContext(ctx)

    // 1. Org-scoped grants (MSP Admin etc.) — apply across ALL tenants in the org.
    orgPerms, err := e.store.GetUserOrgPermissions(ctx, orgID, userID)
    if err != nil { return false, err }
    for _, p := range orgPerms { if p.Covers(required) { return true, nil } }

    // 2. Tenant-scoped grants (existing behavior).
    tenantPerms, err := e.store.GetUserPermissions(ctx, tenantID, userID)
    if err != nil { return false, err }
    for _, p := range tenantPerms { if p.Covers(required) { return true, nil } }

    return false, nil
}
```

**New preset roles** in `internal/server/auth/roles.go`:
- **MSP Admin** — `*:*:*` across all child tenants of an org (re-introduce the removed role, this time backed by real enforcement)
- **MSP Technician** — read all, deploy/remediate, no RBAC or billing
- **MSP Auditor** — read-only across the fleet including audit logs

These are seeded into each org's platform tenant at org creation.

### Cross-tenant queries

For MSP dashboards we need aggregations across child tenants of an org.

**Option A (chosen):** Application-level fan-out. A new helper `store.ForEachTenant(ctx, orgID, fn)` iterates tenants the user has access to, runs `fn` within a per-tenant transaction (respecting RLS), collects and sums results. Cost: N round trips per query, where N = number of child tenants. Acceptable for MSPs with <100 tenants; for larger deployments we add a materialized `org_dashboard_snapshots` table updated periodically.

**Option B (rejected):** Session variable `app.current_org_id` that RLS policies would additionally honor. Requires modifying every RLS policy in the codebase — 40+ policies, high regression risk in a client-testing window.

**Option C (rejected):** `SECURITY DEFINER` views that bypass RLS. Hides authorization logic in SQL, hard to audit.

The helper lives in `internal/server/store/org_scope.go`:

```go
// ForEachTenant runs fn within a per-tenant transaction for each tenant the
// given user has access to in the given organization. Errors short-circuit.
// Use for org-scoped aggregations that must respect RLS.
func (s *Store) ForEachTenant(ctx context.Context, orgID, userID string,
    fn func(ctx context.Context, tenantID string) error) error { ... }
```

### Licensing

Hub-side `licenses` table gets `organization_id UUID NULL FK organizations(id)`. When set, the license is an **umbrella license** whose `max_endpoints` is enforced as the sum of `endpoint_count` across all `clients` belonging to tenants in that org. License validation (currently client-scoped) gains an org path:

```go
func (v *Validator) Validate(ctx context.Context, orgID string) (ValidationResult, error)
```

### Frontend

1. **AuthContext** (`web/src/app/auth/AuthContext.tsx`) carries:
   ```ts
   interface AuthUser {
     user_id: string;
     organization: { id: string; name: string; type: 'direct'|'msp'|'reseller' };
     active_tenant_id: string;
     accessible_tenants: Array<{ id: string; name: string; slug: string }>;
     org_permissions: Permission[];  // from org_user_roles
     tenant_permissions: Permission[]; // from user_roles
     // ...
   }
   ```

2. **TenantSwitcher** component in `web/src/app/layout/TopBar.tsx` — shadcn dropdown, shown only when `accessible_tenants.length > 1`. Selecting a tenant calls `POST /api/v1/session/active-tenant` which rewrites the session cookie, then the client flips `X-Tenant-ID` for all subsequent requests.

3. **API client middleware** (`web/src/api/client.ts`) gains a tenant-id injector that reads the active tenant from AuthContext and sets `X-Tenant-ID` on every request. Today this is implicit (header set somewhere; see openapi-fetch params); we make it explicit.

4. **MSP Dashboard** at `/msp` — only visible when `organization.type === 'msp'`. Shows: aggregated endpoint count, critical patch compliance %, deployment success rate, license utilization, per-tenant breakdown table. Data comes from new endpoint `GET /api/v1/organizations/{id}/dashboard`.

5. **Organizations settings page** at `/settings/organization` — lets MSP Admins list child tenants, create new tenants, invite MSP technicians, view umbrella license usage.

### gRPC / agent enrollment

`proto/patchiq/v1/agent.proto` — no changes. Enrollment tokens already encode `tenant_id`; an MSP provisioning a new client just issues tokens under the right tenant. Org-awareness is server-side only.

## Migration & Backfill

### Server migration `059_organizations.sql`
1. `CREATE TABLE organizations (...)` — global (no RLS).
2. `ALTER TABLE tenants ADD COLUMN organization_id UUID REFERENCES organizations(id)` — nullable first.
3. Backfill: for each existing tenant, create a default `direct`-type org (`name = tenant.name`, `slug = tenant.slug + '-org'`) and set `tenant.organization_id`.
4. `ALTER TABLE tenants ALTER COLUMN organization_id SET NOT NULL`.
5. Create `org_user_roles` table (no RLS — org-scoped, not tenant-scoped; FK to `roles` via CASCADE).
6. Seed MSP preset roles into a newly-created platform tenant for each org (or on-demand at first MSP conversion — see below).

**Lazy platform-tenant creation:** We do NOT seed platform tenants for `direct` orgs during backfill. They're only created when an org is converted to `msp` type. This avoids creating thousands of empty platform tenants in large deployments and keeps the backfill O(orgs).

### Hub migration `019_organizations.sql`
1. `CREATE TABLE organizations (...)` — global.
2. `ALTER TABLE tenants ADD COLUMN organization_id UUID REFERENCES organizations(id)` + backfill.
3. `ALTER TABLE licenses ADD COLUMN organization_id UUID NULL REFERENCES organizations(id)` — umbrella licenses.

### Deprecation of `iam_settings.zitadel_org_id`
The new `organizations.zitadel_org_id` is the authoritative mapping. `iam_settings.zitadel_org_id` is preserved for backward compat but set to empty and marked deprecated in code comments. Removal in a future migration after full rollout.

### Rollback
All migrations are additive. Rollback = drop the new columns/tables; tenant behavior unchanged. The backfill preserves tenant IDs and all tenant-scoped data.

## Rollout Phases

Each phase is independently shippable.

1. **Phase 1 — Schema + backfill.** Migrations ship, `organizations` table populated, `tenants.organization_id` backfilled. Zero behavior change.
2. **Phase 2 — Store layer.** sqlc queries for organizations CRUD, org-to-tenants resolution, GetUserOrgPermissions. No handlers yet.
3. **Phase 3 — RBAC extension.** `Evaluator` reads org-scoped grants. MSP preset roles defined. Removed `// NOTE: MSP Admin role removed` comment.
4. **Phase 4 — Auth/JWT.** Zitadel org mapping wired into JWT middleware. `X-Tenant-ID` header validated against user's accessible tenants. Tenant switcher API endpoint.
5. **Phase 5 — Organizations REST API + Hub umbrella licensing.**
6. **Phase 6 — Frontend.** AuthContext, TenantSwitcher, MSP Dashboard, Organizations settings page.
7. **Phase 7 — E2E tests + verification.** Integration tests for single-tenant (no regression), direct-org, msp-org flows.

## Testing Strategy

Strict TDD per CLAUDE.md anti-slop rule #2.

- **Unit tests** per Go package: store (sqlc), auth (Evaluator with org grants), organization context helpers, JWT middleware with org resolution.
- **Integration tests** (`test-integration`): full flow — provision MSP org, create 3 child tenants, assign MSP Admin to a user, verify cross-tenant read, verify X-Tenant-ID switching.
- **RLS regression test**: assert that app.current_tenant_id=X still prevents reads of tenant Y, regardless of org membership.
- **Backfill test**: seed migration fixture with N tenants, run migration, assert 1:1 org creation and FK integrity.
- **Frontend**: vitest for TenantSwitcher state, AuthContext shape, MSP dashboard rendering with mock data.

## Open Questions

1. **Cross-tenant audit log**: should MSP admin actions be logged in each child tenant's audit stream, the org's audit stream, or both? **Proposed:** both. Action logged in the target tenant's audit table (for tenant owner visibility) and duplicated into an org-level audit view materialized from the child streams.
2. **License validation timing**: umbrella enforcement on each endpoint enrollment requires summing across tenants — potential lock contention. **Proposed:** enforce asynchronously via a River job triggered after enrollment, with a soft in-memory counter for fast-path decisions.
3. **Tenant switcher session persistence**: should the active tenant persist across browser reloads? **Proposed:** yes, store in JWT session claim via `POST /api/v1/session/active-tenant`.

These will be resolved during implementation; none block starting Phase 1.
