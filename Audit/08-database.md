# Database Audit

Audited: Server migrations (001-053), Hub migrations (001-017), Server queries (30 files), Hub queries (16 files), sqlc config.

---

## 1. Migration Issues

### Critical

**[DB-M01] No 2027 audit partitions -- data will land in default partition in ~8 months**
- Files: `internal/server/store/migrations/001_init_schema.sql:208-226`, `internal/hub/store/migrations/001_init_schema.sql` (same range), `internal/server/store/migrations/046_alerts.sql:57-68`
- Both server and hub `audit_events` tables plus the server `alerts` table only have 2026 monthly partitions. The default partition catches out-of-range timestamps, but once 2027 data starts landing there, partition pruning stops working and queries degrade. A migration creating 2027 partitions (or adopting pg_partman) should be added before Q4 2026.

### Important

**[DB-M02] Migration 039 drops and recreates tags/endpoint_tags in Up section -- destructive with data loss**
- File: `internal/server/store/migrations/039_tags.sql:9-12`
- The Up section runs `DROP TABLE IF EXISTS endpoint_tags; DROP TABLE IF EXISTS tags;` before recreating them with different columns. If any data existed from migration 037, it would be permanently lost. While this was likely intentional during early development, it sets a dangerous precedent and should never be repeated for tables with production data.

**[DB-M03] Migration 047 renames column that may not exist if 039 already ran with different schema**
- File: `internal/server/store/migrations/047_tags_rename_key_to_name.sql:3-5`
- Migration 039 creates tags with `key`/`value`/`color` columns, then 047 renames `key` to `name` and drops `value`/`color`. This two-step approach works but means the intermediate schema (039 applied, 047 not yet applied) has a `key` column while all queries reference `name`. Any failure between 039 and 047 leaves the DB in an inconsistent state with queries.

**[DB-M04] Migration 009 drops policy columns without safety check**
- File: `internal/server/store/migrations/009_policy_engine.sql:17-18`
- The Up section runs `DROP COLUMN schedule, DROP COLUMN maintenance_window` without `IF EXISTS`. If the migration is re-run or the columns were already removed, it will fail. Other migrations correctly use `IF EXISTS` (e.g., 013, 035).

### Minor

**[DB-M05] audit_events_default partition in server migration 002 does not get REVOKE like named partitions**
- File: `internal/server/store/migrations/002_rls_policies.sql:22-34`
- The `audit_events_default` partition is not included in the REVOKE UPDATE/DELETE list for `patchiq_app`. Hub migration 002 correctly includes it (line ~33). This means the patchiq_app role can UPDATE/DELETE audit rows that land in the default partition on the server side.

**[DB-M06] Inconsistent migration naming convention**
- Some migrations use descriptive names (`004_m1_core_tables`), others use action-oriented names (`012_add_cancelled_status`, `047_tags_rename_key_to_name`). Minor consistency issue.

---

## 2. Tenant Isolation Gaps

### Important

**[DB-T01] Hub `catalog_entry_syncs` missing FORCE ROW LEVEL SECURITY and WITH CHECK**
- File: `internal/hub/store/migrations/009_catalog_syncs_tenant_and_checks.sql:12-15`
- The migration enables RLS and creates a USING policy but omits `FORCE ROW LEVEL SECURITY` (present on all other hub tables) and `WITH CHECK` clause (only has USING). Without FORCE, the table owner bypasses RLS. Without WITH CHECK, inserts/updates are not validated against the tenant context.

### Minor

**[DB-T02] Hub `catalog_entry_syncs` Down migration does not use `WITH CHECK` on restore either**
- File: `internal/hub/store/migrations/009_catalog_syncs_tenant_and_checks.sql:23-27`
- Consistent with the Up section -- both omit WITH CHECK.

---

## 3. Index Coverage Gaps

### Critical

**[DB-I01] `deployment_targets.endpoint_id` has no index**
- File: `internal/server/store/migrations/001_init_schema.sql:140`
- Only index is `(tenant_id, deployment_id)`. Multiple queries filter/join on `endpoint_id`: `ListDeploymentTargetsByEndpoint`, `ListPatchesForEndpoint`, `ListAvailablePatchesForEndpointByOS` (NOT IN subquery on endpoint_id). This is a hot path for endpoint detail pages and will cause sequential scans on the deployment_targets table.
- Suggested: `CREATE INDEX idx_deployment_targets_endpoint ON deployment_targets(tenant_id, endpoint_id);`

**[DB-I02] `deployment_targets.patch_id` has no index**
- File: `internal/server/store/migrations/001_init_schema.sql:141`
- Queries `CountAffectedEndpointsForPatch`, `ListAffectedEndpointsForPatch`, `ListDeploymentHistoryForPatch` all filter on `patch_id`. Without an index, these perform sequential scans.
- Suggested: `CREATE INDEX idx_deployment_targets_patch ON deployment_targets(tenant_id, patch_id);`

### Important

**[DB-I03] `deployments.policy_id` has no index**
- File: `internal/server/store/migrations/001_init_schema.sql:124`
- `ListDeploymentsForPolicy`, `CountDeploymentsForPolicy` queries filter on `policy_id`. FK without index also means CASCADE operations on policies will sequentially scan deployments.
- Suggested: `CREATE INDEX idx_deployments_policy ON deployments(tenant_id, policy_id);`

**[DB-I04] `commands` table missing `tenant_id` index**
- File: `internal/server/store/migrations/011_deployment_engine.sql:4-20`
- Existing indexes are `(agent_id, status)`, `(deployment_id)`, `(deadline)`. Queries like `ListActiveEndpointsByTenant` and `ListPatchesForPolicyFilters` filter by `tenant_id` alone but there is no index on tenant_id.
- Suggested: `CREATE INDEX idx_commands_tenant ON commands(tenant_id);`

**[DB-I05] `notification_history.channel_id` has no index (FK column)**
- File: `internal/server/store/migrations/016_notifications.sql:50`
- FK to `notification_channels(id)` without an index. If a notification channel is deleted, PostgreSQL will sequentially scan notification_history. Existing indexes cover `tenant_id`, `(tenant_id, created_at)`, `(tenant_id, trigger_type)` but not channel_id.

**[DB-I06] `hub.patch_catalog.feed_source_id` has no dedicated index**
- File: `internal/hub/store/migrations/005_feed_aggregation.sql:41`
- The dedup index only covers rows `WHERE feed_source_id IS NOT NULL AND deleted_at IS NULL`. A plain FK lookup (e.g., cascading delete of a feed_source) would sequentially scan.

### Minor

**[DB-I07] `workflow_edges.source_node_id` and `target_node_id` have no indexes**
- File: `internal/server/store/migrations/018_workflow_tables.sql:58-59`
- FKs to workflow_nodes without indexes. Low volume currently but will matter if workflows grow complex.

**[DB-I08] `endpoint_inventories` only indexed on `(endpoint_id, scanned_at)` -- no tenant_id leading column**
- File: `internal/server/store/migrations/004_m1_core_tables.sql:44-45`
- RLS handles filtering, but the index doesn't have tenant_id as a leading column, which means RLS filter checks cannot use this index efficiently.

---

## 4. Query Coverage Gaps (CRUD)

### Important

**[DB-Q01] No Delete query for `patches` or `cves`**
- Files: `internal/server/store/queries/patches.sql`
- Patches have Create, Get, List, Update but no Delete/SoftDelete. If a patch is ingested incorrectly or needs removal, there is no query for it. CVEs similarly have no delete.

**[DB-Q02] No Delete or SoftDelete for `deployments`**
- File: `internal/server/store/queries/deployments.sql`
- Deployments can be cancelled but never deleted. Old deployments accumulate indefinitely with no cleanup mechanism.

### Minor

**[DB-Q03] Hub `cve_feeds` has no delete query**
- File: `internal/hub/store/queries/cve_feeds.sql`
- Only Create, Upsert, Get, List, Update. If a CVE feed entry needs removal, there is no query.

**[DB-Q04] No Update query for `endpoint_inventories` or `endpoint_packages`**
- File: `internal/server/store/queries/inventory.sql`
- These are append-only by design (new inventory = new snapshot), so this is likely intentional. However, there is no cleanup/delete for old inventories either, which could lead to unbounded growth.

---

## 5. Query/Migration Mismatches

### Minor

**[DB-QM01] `ListEndpointsByTenant` returns all columns but does not filter out decommissioned**
- File: `internal/server/store/queries/endpoints.sql:51`
- Simple `SELECT * FROM endpoints WHERE tenant_id = $1` without decommissioned filter. The richer `ListEndpoints` correctly filters. This simpler query may return ghost endpoints.

---

## 6. Unused Queries

### Important (dead code -- 44 server, 18 hub)

**[DB-U01] Server: 44 unused sqlc queries**
- Full list:
  - `agent_binaries.sql`: GetLatestAgentBinary
  - `alerts.sql`: ListEnabledAlertRules
  - `audit.sql`: ListAuditEventsByTenant, ListAuditEventsByResource, ListAuditEventsByActor, ListAuditEventsByType
  - `compliance.sql`: ListEvaluationsByFramework, CountEvaluationsByState, ListSLADeadlinesApproaching, GetLatestScoresByFramework, DeleteOldControlResults
  - `config.sql`: UpsertConfigOverride, GetConfigOverride, ListConfigOverridesByTenant, ListConfigOverridesByScope, DeleteConfigOverride (entire config.sql is unused)
  - `deployments.sql`: UpdateDeploymentStatus, UpdateDeploymentWaveStatus, ListScheduledDeploymentsDue
  - `endpoints.sql`: UpdateEndpointStatus
  - `groups.sql`: ListEndpointGroupsByTenant, RemoveEndpointFromGroup
  - `iam.sql`: UpsertUserIdentity, GetUserIdentityByExternalID, ListUserIdentities, DisableUserIdentity
  - `notifications.sql`: RetryNotificationHistory
  - `patches.sql`: CreatePatch, ListPatchesByTenant, UpdatePatch, CreateCVE, GetCVEByCVEID, ListCVEsByTenant
  - `policies.sql`: RemoveGroupFromPolicy
  - `roles.sql`: GetRoleByName, ListRoleUsers, CountRoleUsers
  - `tenant_settings.sql`: GetTenantSettings, UpsertTenantSettings
  - `tenants.sql`: GetTenantBySlug
  - `workflow_executions.sql`: GetRunningExecutionsForWorkflow, CreateApprovalRequest, GetApprovalRequest
  - `workflows.sql`: GetVersionByID

**[DB-U02] Hub: 18 unused sqlc queries**
- Full list:
  - `agent_binaries.sql`: CreateAgentBinary, GetAgentBinaryByID, ListAgentBinaries, GetLatestBinary (entire file unused)
  - `audit.sql`: ListAuditEventsByActor, ListAuditEventsByType
  - `binary_fetch.sql`: GetBinaryFetchState
  - `cve_feeds.sql`: GetCVEFeedByID, UpdateCVEFeed
  - `feed_sources.sql`: ListEnabledFeedSources, UpdateFeedSourceEnabled
  - `feed_sync_history.sql`: CreateFeedSyncHistory
  - `hub_config.sql`: DeleteHubConfig
  - `tenants.sql`: CreateTenant, GetTenantByID, GetTenantBySlug, ListTenants, UpdateTenant (entire file unused)

**[DB-U03] Notable: entire `config.sql` (server) and `tenants.sql` (hub) query files are unused**
- `internal/server/store/queries/config.sql` -- all 5 queries unused. The config_overrides table has no callers.
- `internal/hub/store/queries/tenants.sql` -- all 5 queries unused. Hub tenant management is likely done through seed data only.

---

## 7. N+1 Query Patterns

### Important

**[DB-N01] `GetPolicyByID` called in loop for deployment list**
- File: `internal/server/api/v1/deployments.go:712-722`
- When listing deployments, the handler iterates over results and calls `GetPolicyByID` per unique policy_id to build a name cache. While it deduplicates (`!seen` check), this is still N individual queries where N = number of distinct policies in the page. Should be a single batch query or JOIN in the list query itself.

**[DB-N02] `RemoveTagFromEndpoint` called in loop**
- File: `internal/server/api/v1/tags.go:325-340`
- When bulk-unassigning a tag from multiple endpoints, each removal is a separate query in a loop. Should use a single `DELETE ... WHERE tag_id = $1 AND endpoint_id = ANY($2)` pattern.

**[DB-N03] `UpsertNotificationPreference` called in loop**
- File: `internal/server/api/v1/notifications.go:618-630`
- When updating multiple notification preferences, each trigger type is upserted individually. Bounded by the number of trigger types (~10-15), so impact is limited, but a bulk upsert would be cleaner.

### Minor

**[DB-N04] `ListEndpoints` query has 4 CTEs computing aggregates across entire tenant**
- File: `internal/server/store/queries/endpoints.sql:72-148`
- The `cve_counts`, `patch_counts`, `compliance_avgs`, and `tag_info` CTEs each scan their respective tables for the entire tenant before joining to the paginated endpoint result. On tenants with many endpoints, this means computing aggregates for ALL endpoints even though only a page of ~25 is returned. Consider computing aggregates only for the endpoints in the page (subquery on the paginated set first, then join).

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 3 |
| Important | 14 |
| Minor | 10 |

**Top priorities for the POC deployment:**
1. **[DB-I01] + [DB-I02]**: Add indexes on `deployment_targets(endpoint_id)` and `deployment_targets(patch_id)` -- these are hot-path queries that will degrade with real data volumes.
2. **[DB-M01]**: Create 2027 partitions before they are needed (audit_events + alerts on both server and hub).
3. **[DB-T01]**: Fix catalog_entry_syncs RLS (add FORCE and WITH CHECK) on the hub.
4. **[DB-N01]**: Resolve GetPolicyByID N+1 in the deployment list handler.
5. **[DB-U01/U02]**: Clean up 62 unused queries to reduce generated code size and maintenance burden.
