# ADR-025: Organization-Scoped RBAC for MSP Model

## Status

Accepted — 2026-04-11

## Context

PatchIQ's current multi-tenancy model (ADR-022) enforces tenant isolation via PostgreSQL Row-Level Security. Every tenant-scoped table carries `tenant_id` and a `tenant_isolation` RLS policy that filters rows by `current_setting('app.current_tenant_id')`. This is a strong isolation boundary — rows from tenant A can never leak into a tenant-B query.

But tenants are flat. There is no parent entity that owns multiple tenants. This rules out the Managed Service Provider (MSP) go-to-market motion, where one operator manages many client organizations from a single login. Specifically:

- A single user cannot hold permissions across multiple tenants without per-tenant provisioning.
- There is no cross-tenant dashboard for MSPs to monitor client fleets.
- Licensing is per-tenant; no umbrella license spans many clients.
- Zitadel's native organizations claim (`urn:zitadel:iam:org:id`) is extracted in `internal/server/auth/jwt.go` but maps to a hardcoded `DefaultTenantID` — the mapping infrastructure exists only in stub form.
- The "MSP Admin" preset role was explicitly removed from `internal/server/auth/roles.go` pending cross-tenant RBAC enforcement.

The M3/M4 roadmap (`docs/blueprint/m3/11-msp-portal-foundations.md`, `docs/blueprint/m4/03-msp-portal-full.md`) describes MSP Portal as future work but specifies a UI on top of a data model that does not exist.

MSPs are a strategic market: one MSP contract can subsume 10-100 client relationships, with ARPU 2-4× direct customers. Without organization-scoped RBAC, every MSP RFP is an immediate loss.

## Decision

We introduce an `organizations` entity as the parent of `tenants`:

```
organizations 1 ─── N tenants ─── N endpoints/patches/...
```

Key choices:

1. **Organizations are global** — no `tenant_id`, no RLS. They live in a single global table on each of server and hub. Authorization to an org is carried in application context (`internal/shared/organization/`), parallel to tenant context.

2. **Tenants remain the RLS boundary.** We do NOT modify RLS policies to be org-scoped. `app.current_tenant_id` still filters every tenant-scoped query. This is an explicit trade-off — we keep the strong isolation invariant that ADR-022 established, at the cost of needing application-level fan-out for cross-tenant reads. We evaluated widening RLS to an `app.current_org_id` session variable and rejected it: modifying 40+ RLS policies during an active client testing window is too risky.

3. **Cross-tenant reads happen via application fan-out**, not SQL. A new helper `store.ForEachTenant(ctx, orgID, userID, fn)` iterates tenants the user has access to in the org and runs `fn` within a per-tenant transaction. MSPs with <100 tenants run aggregations in milliseconds; larger scales get a materialized snapshot table in a later phase.

4. **Org-scoped roles use a dedicated "platform tenant" per organization.** The `roles` table is tenant-scoped (RLS). Rather than weaken that invariant, each `msp`-type org gets a hidden platform tenant (`slug = org:<slug>:platform`) whose sole purpose is to host org-scoped role definitions (MSP Admin, MSP Technician, MSP Auditor). A new `org_user_roles` table (not RLS-protected) maps `(organization_id, user_id, role_id)` to those roles. The evaluator checks org-scoped grants first, then falls back to tenant-scoped grants.

5. **Zitadel organization is the authoritative IdP mapping.** The `organizations` table gains a `zitadel_org_id TEXT UNIQUE` column. JWT middleware resolves `urn:zitadel:iam:org:id` → organization → accessible tenants → active tenant (via `X-Tenant-ID` header, validated against the user's accessible list).

6. **Umbrella licensing on Hub.** `hub.licenses` gains `organization_id UUID NULL`. When set, `max_endpoints` is enforced as the sum of `endpoint_count` across all clients belonging to tenants in that org, validated asynchronously via a River job after enrollment.

7. **Backward compatibility via backfill.** A migration wraps every existing tenant in its own `direct`-type organization (1:1). Single-tenant deployments see zero behavior change. MSP features only activate when an org is explicitly converted to `msp` type.

## Consequences

### Easier
- MSPs can be onboarded with bulk tenant provisioning, cross-tenant roles, and aggregated dashboards.
- Umbrella licensing unlocks MSP pricing tiers.
- Future reseller hierarchies (MSP → sub-MSP → client) are a straightforward extension — the `parent_org_id` column is already reserved in v1.
- Zitadel organizations finally have a real binding instead of being silently ignored.
- The removed `MSP Admin` preset role comes back, this time backed by actual enforcement.

### Harder
- **Cross-tenant reads are now O(N) in application code.** Developers must use `ForEachTenant` rather than writing naive SQL across tenants. Lint rule or PR review should catch violations.
- **Two grant tables to check on every permission evaluation** (`user_roles` and `org_user_roles`). Small constant-factor cost on the hot auth path; negligible with Valkey caching.
- **Platform tenants add schema surface area.** Each MSP org owns a hidden tenant whose only purpose is role storage. We accept this to preserve RLS purity on `roles`.
- **Two sources of truth for Zitadel org ID** during the deprecation window (`organizations.zitadel_org_id` authoritative, `iam_settings.zitadel_org_id` legacy). Removal scheduled in a later migration.
- **Frontend complexity**: tenant switcher, accessible-tenants state, MSP dashboard, organizations settings page. TanStack Query cache must be invalidated on tenant switch.
- **Agent enrollment is unchanged** — agents still enroll into a specific tenant via registration token. MSPs provision agents per-client-tenant; there is no "org-wide agent enrollment."

### Risk mitigation
- **Additive migrations**: all changes can be rolled back by dropping the new columns/tables.
- **Lazy platform-tenant creation**: we only create platform tenants when an org is converted to `msp`, avoiding schema bloat.
- **RLS regression test**: assert that `app.current_tenant_id=X` still blocks reads of tenant Y even with an active org context.
- **Feature gate**: the tenant switcher UI only renders when `accessible_tenants.length > 1`, so direct customers see no change.

## Alternatives Considered

### A. Widen RLS to org scope via `app.current_org_id`

Add an org session variable alongside the tenant variable, and amend every RLS policy to accept `tenant_id IN (SELECT id FROM tenants WHERE organization_id = current_setting('app.current_org_id'))`. Rejected because:
- Requires rewriting 40+ RLS policies across server and hub.
- Every sqlc-generated query implicitly broadens — developers lose the mental model of "one query, one tenant."
- The change is irreversible without a second migration pass.
- Client testing is in progress; RLS regressions would be catastrophic for the beta.

### B. `SECURITY DEFINER` views for cross-tenant aggregation

Create Postgres views owned by a privileged role that bypass RLS and filter by `organization_id` in SQL. Rejected because:
- Authorization logic hidden in SQL is hard to audit and test.
- Every new aggregation query requires a new view.
- Ownership/privilege model conflicts with the principle that the application is the only authz boundary.

### C. Collapse "tenant" into "organization" — one entity, rename

Rename `tenants` → `organizations` and make it the only level. Rejected because:
- Migration blast radius is enormous — every tenant_id column, RLS policy, sqlcgen file, handler, and frontend hook.
- Loses the distinction between "isolated workspace" (tenant) and "billing/identity parent" (org) — which is actually useful for MSP workflows where one org spans many workspaces.
- No existing code would compile after the rename without a multi-week refactor.

### D. External service (separate org management microservice)

Push organization/tenant routing to an API gateway or sidecar. Rejected because:
- Adds operational complexity to a product that needs to deploy on client infrastructure.
- The platform is a monolith by design (ADR-019); introducing a new service violates that.
- Authorization data lives in the same database anyway — a separate service just adds a network hop.

### E. Defer until after client testing window

Wait 1 month, then implement. Rejected by product direction (Heramb, 2026-04-11): MSP story is needed now for sales conversations, even if the feature ships alongside the end of the client POC.

## References

- ADR-004: Custom RBAC Model
- ADR-012: Zitadel for IAM
- ADR-022: PostgreSQL RLS Multi-Tenancy
- Design doc: `docs/plans/2026-04-11-org-scoped-rbac-msp.md`
- Removed role marker: `internal/server/auth/roles.go:90-91`
