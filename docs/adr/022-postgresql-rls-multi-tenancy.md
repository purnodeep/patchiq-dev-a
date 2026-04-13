# ADR-022: PostgreSQL Row-Level Security for Multi-Tenant Data Isolation

## Status

Accepted

## Context

PatchIQ Patch Manager and Hub Manager are multi-tenant: a single database instance serves multiple customer organizations. Every tenant's data (endpoints, patches, policies, deployments, audit events) must be strictly isolated. A bug in application code must never expose one tenant's data to another. The isolation mechanism must work transparently with sqlc-generated queries, require no per-query WHERE clause discipline, and be enforceable at the database level independent of application correctness.

## Decision

### Row-Level Security with `app.current_tenant_id`

Use PostgreSQL Row-Level Security (RLS) as the primary tenant isolation mechanism. Every tenant-scoped table has a `tenant_id UUID NOT NULL` column as the first column after the primary key, an RLS policy, and `FORCE ROW LEVEL SECURITY` enabled.

The RLS policy (`internal/server/store/migrations/002_rls_policies.sql`) uses a PostgreSQL session variable:

```sql
CREATE POLICY tenant_isolation ON <table>
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

The `USING` clause filters SELECT/UPDATE/DELETE. The `WITH CHECK` clause validates INSERT/UPDATE. Both reference the same transaction-scoped variable, ensuring reads and writes are constrained to the active tenant.

### Tenant Context Flow

1. **HTTP layer**: `tenant.Middleware` (`internal/shared/tenant/middleware.go`) extracts the tenant ID from the `X-Tenant-ID` header, validates it as a UUID, and injects it into `context.Context` via `tenant.WithTenantID()`.
2. **Database layer**: `store.BeginTx()` reads `tenant.TenantIDFromContext(ctx)` and executes `SET LOCAL app.current_tenant_id = $tenant_id` as the first statement in the transaction. `SET LOCAL` scopes the variable to the current transaction only — it is automatically cleared on commit/rollback.
3. **Query layer**: sqlc-generated queries execute within the transaction. RLS policies filter rows transparently. No application-level `WHERE tenant_id = ?` is needed (though queries include `tenant_id` in indexes for performance).

### FORCE ROW LEVEL SECURITY

All tenant-scoped tables use `ALTER TABLE <table> FORCE ROW LEVEL SECURITY` (`002_rls_policies.sql:125-138`). Without `FORCE`, the table owner role bypasses RLS entirely. `FORCE` ensures that even if the migration superuser role is accidentally used at runtime, RLS still applies.

### Application Role with Restricted Privileges

A dedicated `patchiq_app` role (`002_rls_policies.sql:9-12`) receives `SELECT, INSERT, UPDATE, DELETE` on all tables, with `UPDATE` and `DELETE` revoked on `audit_events` and all its monthly partitions (enforcing append-only audit). Default privileges ensure future tables automatically grant to `patchiq_app`.

### Global vs Tenant-Scoped Tables

Global tables (no `tenant_id`, no RLS): `tenants`. Tenant-scoped tables (14 total): `endpoints`, `endpoint_groups`, `endpoint_group_members`, `patches`, `cves`, `patch_cves`, `policies`, `policy_groups`, `deployments`, `deployment_targets`, `deployment_waves`, `agent_registrations`, `config_overrides`, `audit_events`.

### Context Safety

`tenant.WithTenantID()` panics on empty string (`internal/shared/tenant/context.go:10`). `tenant.MustTenantID()` panics if no tenant ID is in context. These fail-fast guards ensure programming errors surface immediately rather than silently querying without RLS.

## Consequences

- **Positive**: Tenant isolation is enforced at the database level — application bugs cannot leak data across tenants; sqlc queries work transparently without manual WHERE clauses; `SET LOCAL` scoping prevents tenant context leaking between requests; `FORCE` protects against owner-role bypass; fail-fast panics catch missing tenant context during development
- **Negative**: Every transaction must call `SET LOCAL` before any query — forgetting this causes `current_setting` to error (fail-safe, not fail-open); RLS adds ~2-5% query overhead per PostgreSQL benchmarks; schema-per-tenant would provide stronger isolation but at prohibitive operational cost; debugging RLS-filtered queries requires awareness of the active tenant context

## Alternatives Considered

- **Application-level filtering** (`WHERE tenant_id = ?` in every query): Simplest — rejected because a single missed WHERE clause leaks all tenants' data; not enforceable at the database layer; error-prone with sqlc code generation
- **Schema-per-tenant**: Strongest isolation — rejected because it creates operational complexity (thousands of schemas for SaaS), breaks connection pooling, complicates migrations (must run against every schema), and does not scale for Hub Manager
- **Separate database per tenant**: Maximum isolation — rejected for same reasons as schema-per-tenant, amplified; connection pool exhaustion with hundreds of tenants
- **Citus/partitioning by tenant**: Distributed multi-tenant — rejected as premature; adds significant operational complexity; PatchIQ's current scale does not warrant distributed PostgreSQL
