# Alerts/Events Page — Design Spec

> **POC-PLAN ref:** Section 4.1 (B1)
> **Branch:** dev-b
> **Date:** 2026-03-30

---

## 1. Purpose

A unified operational view of system activity — "what needs attention now." Distinct from the audit log (forensic/historical), the alerts page is operational: real-time feed of classified events that surface deployment failures, agent disconnects, compliance breaches, and other actionable items.

Clients deploying PatchIQ on their infrastructure need a single place to monitor system health without digging through audit logs.

## 2. Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Alert storage | Separate `alerts` table materialized from domain events | Clean separation from audit, own lifecycle (read/ack/dismiss), faster filtered queries |
| Alert rules | Database-driven `alert_rules` table per tenant | Client POC needs fully functional, configurable system — not hardcoded maps |
| Severity levels | critical / warning / info | Maps to existing signal colors (red/amber/green) |
| Alert lifecycle | unread → read → acknowledged → dismissed | Four states cover operational workflow: triage → investigate → resolve → archive |
| Row layout | Card rows with severity icon block | Option B from visual brainstorm — colored icon with tinted background, no left border stripes |
| Sidebar placement | Compliance group (after Audit) | Operational monitoring alongside compliance and audit |
| Real-time | Polling with configurable interval (default 30s) | SSE not needed for POC; polling is simpler and sufficient |
| Sidebar badge | Unread critical + warning count | Shows product maturity; lightweight count endpoint |

## 3. Data Model

### 3.1 `alert_rules` table

Tenant-scoped. Each row defines which event type triggers an alert and how it's classified.

```sql
CREATE TABLE alert_rules (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    event_type            TEXT NOT NULL,
    severity              TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    category              TEXT NOT NULL CHECK (category IN ('deployments', 'agents', 'cves', 'compliance', 'system')),
    title_template        TEXT NOT NULL,
    description_template  TEXT NOT NULL,
    enabled               BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, event_type)
);
```

**RLS:** Standard tenant isolation policy. GRANT SELECT, INSERT, UPDATE, DELETE to `patchiq_app`.

**Indexes:**
- `idx_alert_rules_tenant_enabled` on `(tenant_id, enabled)` — for subscriber cache loading

### 3.2 `alerts` table

Tenant-scoped, partitioned monthly (same strategy as `audit_events`).

```sql
CREATE TABLE alerts (
    id              TEXT NOT NULL,
    tenant_id       UUID NOT NULL,
    rule_id         UUID,  -- no FK: PG 16 does not support FK from partitioned to non-partitioned tables; enforced at app level
    event_id        TEXT NOT NULL,
    severity        TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    category        TEXT NOT NULL CHECK (category IN ('deployments', 'agents', 'cves', 'compliance', 'system')),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    resource        TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'acknowledged', 'dismissed')),
    payload         JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL,
    read_at         TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    dismissed_at    TIMESTAMPTZ,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
```

**Partitions:** Monthly for 2026 (Jan–Dec) + default partition.

**RLS:** Standard tenant isolation. No UPDATE restriction (unlike audit_events which is append-only) — alerts need status updates.

**Indexes:**
- `idx_alerts_tenant_status_created` on `(tenant_id, status, created_at DESC)` — main list query
- `idx_alerts_tenant_unread_severity` on `(tenant_id, severity)` WHERE `status = 'unread'` — badge count (partial index, unread only)
- `UNIQUE INDEX idx_alerts_event_id` on `(event_id, created_at)` — dedup: `ON CONFLICT (event_id, created_at) DO NOTHING`

### 3.3 Default alert rules (seed data)

Seeded per tenant using `ON CONFLICT DO NOTHING`. 15 default rules:

**Critical:**
| event_type | category | title_template | description_template |
|-----------|----------|----------------|---------------------|
| `deployment.failed` | deployments | `Deployment failed: {{.name}}` | `{{.failed_count}} targets failed in wave {{.wave}}.` |
| `deployment.rollback_triggered` | deployments | `Rollback triggered: {{.name}}` | `Deployment rolled back due to failure threshold.` |
| `agent.disconnected` | agents | `Agent disconnected: {{.hostname}}` | `Endpoint {{.hostname}} lost connection.` |
| `compliance.threshold_breach` | compliance | `Compliance breach: {{.framework}}` | `Score dropped below threshold ({{.score}}%).` |
| `license.expired` | system | `License expired` | `PatchIQ license has expired. Features may be restricted.` |
| `catalog.sync_failed` | system | `Catalog sync failed` | `Hub catalog synchronization failed: {{.error}}.` |

**Warning:**
| event_type | category | title_template | description_template |
|-----------|----------|----------------|---------------------|
| `command.timed_out` | deployments | `Command timed out: {{.command_id}}` | `Deployment command timed out on target.` |
| `license.expiring` | system | `License expiring soon` | `License expires in {{.days_remaining}} days.` |
| `cve.discovered` | cves | `New CVE: {{.cve_id}}` | `CVSS {{.cvss_score}} — affects {{.package}}.` |
| `notification.failed` | system | `Notification delivery failed` | `Failed to send via {{.channel}}: {{.error}}.` |
| `deployment.wave_failed` | deployments | `Wave failed: {{.name}}` | `Wave {{.wave}} failed — deployment may need attention.` |

**Info:**
| event_type | category | title_template | description_template |
|-----------|----------|----------------|---------------------|
| `deployment.completed` | deployments | `Deployment completed: {{.name}}` | `All targets patched successfully.` |
| `deployment.started` | deployments | `Deployment started: {{.name}}` | `Deployment began execution.` |
| `endpoint.enrolled` | agents | `New endpoint enrolled: {{.hostname}}` | `Agent registered from {{.ip_address}}.` |
| `cve.remediation_available` | cves | `Remediation available: {{.cve_id}}` | `Patch available for {{.cve_id}}.` |

## 4. Backend

### 4.1 AlertSubscriber (`internal/server/events/alert_subscriber.go`)

Watermill subscriber on `"*"` (wildcard). On each event:

1. **Guard:** Skip any event whose type starts with `alert.` or `alert_rule.` (prevents infinite loops from the subscriber's own output)
2. Check in-memory rule cache for matching `event_type` + `tenant_id` where `enabled = true`
3. If no match → skip (most events won't match — this is the fast path)
4. If match → render title and description using Go `text/template` with event payload as data
5. Insert into `alerts` table with dedup on `event_id` (`ON CONFLICT (event_id, created_at) DO NOTHING`)

**Rule cache:** `sync.Map` keyed by `tenant_id:event_type`. Refreshed every 30 seconds by a background goroutine. Cache miss falls through to DB lookup (and populates cache on hit).

**Template rendering:** `text/template` with `{{.field_name}}` syntax. Template errors are logged and fall back to raw event type as title. Payload fields are accessed by JSON key name from the event's `Payload` map.

**Registration in `cmd/server/main.go`:**
```go
alertSub := events.NewAlertSubscriber(pool, logger)
if err := eventBus.Subscribe("*", alertSub.Handle); err != nil {
    return fmt.Errorf("subscribe alert handler: %w", err)
}
```

### 4.2 AlertHandler (`internal/server/api/v1/alerts.go`)

**Querier interface:**
```go
type AlertQuerier interface {
    // Alerts
    ListAlertsFiltered(ctx context.Context, arg sqlcgen.ListAlertsFilteredParams) ([]sqlcgen.Alert, error)
    CountAlertsFiltered(ctx context.Context, arg sqlcgen.CountAlertsFilteredParams) (int64, error)
    CountUnreadAlerts(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.CountUnreadAlertsRow, error)
    UpdateAlertStatus(ctx context.Context, arg sqlcgen.UpdateAlertStatusParams) (sqlcgen.Alert, error)
    BulkUpdateAlertStatus(ctx context.Context, arg sqlcgen.BulkUpdateAlertStatusParams) (int64, error)

    // Alert Rules
    ListAlertRules(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.AlertRule, error)
    GetAlertRule(ctx context.Context, arg sqlcgen.GetAlertRuleParams) (sqlcgen.AlertRule, error)
    CreateAlertRule(ctx context.Context, arg sqlcgen.CreateAlertRuleParams) (sqlcgen.AlertRule, error)
    UpdateAlertRule(ctx context.Context, arg sqlcgen.UpdateAlertRuleParams) (sqlcgen.AlertRule, error)
    DeleteAlertRule(ctx context.Context, arg sqlcgen.DeleteAlertRuleParams) (int64, error)
}
```

**Constructor:** `NewAlertHandler(q AlertQuerier, pool TxBeginner, eventBus domain.EventBus)` — validates all deps with panics per codebase convention. Pool required for transactional status updates with RLS tenant context. Constructed inside `NewRouter` using the shared querier, pool, and eventBus already in scope (same pattern as `auditH`), NOT passed as a new parameter.

### 4.3 API Endpoints

| Method | Path | RBAC | Handler | Description |
|--------|------|------|---------|-------------|
| GET | `/api/v1/alerts` | `alerts.read` | `List` | Paginated alert list with filters |
| GET | `/api/v1/alerts/count` | `alerts.read` | `Count` | Unread counts by severity (for badge) |
| PATCH | `/api/v1/alerts/{id}/status` | `alerts.update` | `UpdateStatus` | Set status on single alert |
| PATCH | `/api/v1/alerts/bulk-status` | `alerts.update` | `BulkUpdateStatus` | Set status on multiple alerts |
| GET | `/api/v1/alert-rules` | `alerts.manage` | `ListRules` | List tenant's alert rules |
| POST | `/api/v1/alert-rules` | `alerts.manage` | `CreateRule` | Create new alert rule |
| PUT | `/api/v1/alert-rules/{id}` | `alerts.manage` | `UpdateRule` | Update alert rule |
| DELETE | `/api/v1/alert-rules/{id}` | `alerts.manage` | `DeleteRule` | Delete alert rule |

**List alerts query parameters:**
- `severity` — filter by severity (critical, warning, info)
- `category` — filter by category (deployments, agents, cves, compliance, system)
- `status` — filter by status (unread, read, acknowledged, dismissed). Default: `unread,read` (active alerts)
- `search` — full-text search on title and description
- `from_date` / `to_date` — RFC3339 date range
- `cursor` / `limit` — keyset pagination on `(created_at, id)`

**Count response** (counts only `status = 'unread'`, not `read`):
```json
{
  "critical_unread": 3,
  "warning_unread": 7,
  "info_unread": 2,
  "total_unread": 12
}
```

**Update status request:**
```json
{
  "status": "acknowledged"
}
```

**Bulk update request:**
```json
{
  "ids": ["01HXK...", "01HXL..."],
  "status": "dismissed"
}
```

### 4.4 sqlc Queries (`internal/server/store/queries/alerts.sql`)

Key queries:

- `InsertAlert :exec` — ON CONFLICT (event_id, created_at) DO NOTHING
- `ListAlertsFiltered :many` — keyset pagination with severity/category/status/search/date filters using named params
- `CountAlertsFiltered :one` — matching count for pagination metadata
- `CountUnreadAlerts :one` — returns `critical_unread`, `warning_unread`, `info_unread` counts (uses conditional aggregation)
- `UpdateAlertStatus :one` — sets status + corresponding timestamp (read_at, acknowledged_at, dismissed_at) RETURNING *
- `BulkUpdateAlertStatus :execrows` — updates multiple alerts by ID list
- `ListAlertRules :many` — all rules for tenant
- `GetAlertRule :one` — single rule by ID + tenant
- `CreateAlertRule :one` — INSERT RETURNING *
- `UpdateAlertRule :one` — UPDATE RETURNING *
- `DeleteAlertRule :execrows` — DELETE returning row count

### 4.5 OpenAPI Spec Updates (`api/server.yaml`)

New schemas: `Alert`, `AlertRule`, `AlertCountResponse`, `CreateAlertRuleRequest`, `UpdateAlertRuleRequest`, `UpdateAlertStatusRequest`, `BulkUpdateAlertStatusRequest`.

New paths: All 8 endpoints above with full parameter definitions, following the existing audit/notification spec patterns.

New tag: `Alerts`.

### 4.6 Domain Events

New event types in `internal/server/events/topics.go`:
```go
AlertCreated        = "alert.created"
AlertStatusUpdated  = "alert.status_updated"
AlertRuleCreated    = "alert_rule.created"
AlertRuleUpdated    = "alert_rule.updated"
AlertRuleDeleted    = "alert_rule.deleted"
```

The AlertSubscriber itself does NOT emit `alert.created` events (to avoid infinite loops — the subscriber listens on `"*"`). Instead, the `InsertAlert` query is fire-and-forget. The handler endpoints for status updates and rule CRUD emit events normally.

### 4.7 Migration (`046_alerts.sql`)

> **Note:** Migration number is provisional; verify no conflict exists at implementation time.

Single migration file containing:
1. `alert_rules` table + RLS + indexes + grants
2. `alerts` table + monthly partitions (2026) + default partition + RLS + indexes + grants
3. Seed default alert rules for the default tenant (`00000000-0000-0000-0000-000000000001`)

## 5. Frontend

### 5.1 Route & Navigation

**Route:** `/alerts` — added to `web/src/app/routes.tsx` under protected routes.

**Sidebar:** New entry in the Compliance group in `web/src/app/layout/AppSidebar.tsx`:
```
Compliance group:
  - Compliance (ShieldCheck)
  - Alerts (BellRing + badge)  ← NEW (before Audit — more operationally urgent)
  - Audit (FileSearch)
```

Icon: `BellRing` from lucide-react (distinct from `Bell` used for Notifications in System group).

**Badge:** Small count indicator on the Alerts nav item. Shows `critical_unread + warning_unread`. Hidden when count is 0. Styled with `--signal-critical` background when critical > 0, `--signal-warning` otherwise. Polled via `useAlertCount` from sidebar level.

### 5.2 AlertsPage (`web/src/pages/alerts/AlertsPage.tsx`)

**Layout** (follows AuditPage pattern — inline styles with CSS variables):

1. **Header row:** `PageHeader` with title "Alerts", total unread count badge, actions:
   - Refresh interval selector (dropdown: 10s, 30s, 60s, off)
   - "Manage Rules" button → opens rules sheet

2. **Filter bar:**
   - Severity pills: All | Critical (red) | Warning (amber) | Info (green) — each showing count
   - Status pills: Active | Acknowledged | Dismissed | All — "Active" = unread + read (default)
   - Category pills: All | Deployments | Agents | CVEs | Compliance | System
   - Search input + date range picker (preset: 24h, 7d, 30d, custom)

3. **Alert list:** Scrollable list of alert rows (not DataTable — custom list for richer layout)
   - Checkbox for bulk selection
   - Unread indicator (green dot, 6px)
   - Severity icon block (28x28, tinted background per severity)
   - Title (13px, weight 500, `--text-emphasis`)
   - Description (12px, `--text-secondary`)
   - Meta row: severity badge + category + entity link (accent color, monospace) + relative timestamp
   - Click row → navigates to source entity (e.g., `/deployments/d-2847a`)
   - Right side: individual action menu (Mark Read, Acknowledge, Dismiss)

4. **Bulk action bar:** Appears when 1+ alerts selected. Actions: Mark Read, Acknowledge, Dismiss. Follows existing `BulkActionBar` pattern from endpoints page.

5. **Pagination:** Cursor-based with `DataTablePagination` component.

6. **States:**
   - Loading: Skeleton rows (8 rows)
   - Empty: `EmptyState` — "No alerts" with icon, description varies by active filter
   - Error: `ErrorState` with retry button

### 5.3 Alert Rules Sheet (`web/src/pages/alerts/AlertRulesSheet.tsx`)

Slide-out panel (560px, uses `Sheet` from `@patchiq/ui`). Contains:

- Table of alert rules with columns: Event Type, Severity, Category, Enabled toggle
- Click row to expand inline edit (severity dropdown, title/description template fields)
- "Add Rule" button at top
- Delete action per row (with confirmation)

### 5.4 API Hooks (`web/src/api/hooks/useAlerts.ts`)

```typescript
// Alert list with auto-refresh
useAlerts(filters: AlertFilters, refetchInterval?: number)

// Lightweight count for sidebar badge
useAlertCount(refetchInterval?: number)

// Mutations
useUpdateAlertStatus()
useBulkUpdateAlertStatus()

// Alert rules CRUD
useAlertRules()
useCreateAlertRule()
useUpdateAlertRule()
useDeleteAlertRule()
```

All hooks follow existing patterns: TanStack Query with `queryKey` arrays, `useMutation` with `onSuccess` cache invalidation via `queryClient.invalidateQueries`.

### 5.5 Sidebar Badge Component

Inline in `AppSidebar.tsx` — a small `<span>` positioned to the right of the "Alerts" nav label. Styled:

- Background: `var(--signal-critical)` if any critical unread, else `var(--signal-warning)` if any warning unread
- Text: white, 10px, font-weight 600, monospace
- Border-radius: `var(--radius-full)` (pill shape)
- Min-width: 18px, padding: 0 5px
- Hidden when total count is 0

Data from `useAlertCount()` called at sidebar level with 30s refetch interval.

## 6. Files Changed

### New files:
| File | Purpose |
|------|---------|
| `internal/server/store/migrations/046_alerts.sql` | Migration: alert_rules + alerts tables |
| `internal/server/store/queries/alerts.sql` | sqlc queries for alerts and alert rules |
| `internal/server/events/alert_subscriber.go` | Watermill subscriber: event → alert materialization |
| `internal/server/events/alert_subscriber_test.go` | Subscriber tests |
| `internal/server/events/alert_rules.go` | Severity map, template rendering, rule cache |
| `internal/server/events/alert_rules_test.go` | Rule cache and template tests |
| `internal/server/api/v1/alerts.go` | REST handler for alerts + alert rules |
| `internal/server/api/v1/alerts_test.go` | Handler tests |
| `web/src/pages/alerts/AlertsPage.tsx` | Main alerts page |
| `web/src/pages/alerts/AlertRulesSheet.tsx` | Alert rules management sheet |
| `web/src/pages/alerts/AlertRow.tsx` | Single alert row component |
| `web/src/pages/alerts/AlertFilters.tsx` | Filter bar component |
| `web/src/api/hooks/useAlerts.ts` | TanStack Query hooks |

### Modified files:
| File | Change |
|------|--------|
| `internal/server/events/topics.go` | Add 5 new alert event types |
| `cmd/server/main.go` | Wire AlertSubscriber + AlertHandler |
| `internal/server/api/router.go` | Register alert routes with RBAC |
| `api/server.yaml` | Add Alert schemas + 8 endpoint definitions |
| `web/src/app/routes.tsx` | Add `/alerts` route |
| `web/src/app/layout/AppSidebar.tsx` | Add Alerts nav item with badge |
| `scripts/seed.sql` | Seed default alert rules |
| `internal/server/store/sqlcgen/` | Regenerated (sqlc) |
| `web/src/api/types.ts` | Regenerated (openapi-typescript) |

## 7. Testing Strategy

**Backend:**
- Table-driven unit tests for AlertSubscriber (event matching, template rendering, dedup)
- Table-driven unit tests for rule cache (load, refresh, invalidation)
- Table-driven unit tests for AlertHandler (all 8 endpoints, error cases, RBAC)
- Mock `AlertQuerier` interface for handler tests

**Frontend:**
- Vitest unit tests for AlertsPage (rendering, filter state, loading/error/empty states)
- Vitest unit tests for AlertRow (severity icon, status display, click behavior)
- Vitest unit tests for useAlerts hooks (query key construction, cache invalidation)

## 8. Out of Scope

- SSE/WebSocket push (polling is sufficient for POC)
- Alert grouping/deduplication beyond event_id uniqueness
- Alert escalation chains (M3 alert pipelines)
- Email/Slack forwarding of alerts (use existing notification channels)
- Alert rule templates marketplace
- Historical alert analytics/charts
