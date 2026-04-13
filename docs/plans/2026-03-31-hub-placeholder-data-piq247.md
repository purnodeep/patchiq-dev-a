# Hub Placeholder Data — Real Data Implementation (PIQ-247)

> **Goal**: Replace all 8+ fake/placeholder chart TODOs in web-hub/ with real backend data.
>
> **Approach**: Sync-Enrichment — one new table, three new JSONB columns, reuse existing infrastructure.
>
> **Created**: 2026-03-31 | **Branch**: dev-b

---

## 1. Problem

The web-hub frontend has 8+ TODOs (all tagged PIQ-247) across `ClientDetailPage`, `ClientsPage`, and `LicenseDetailPage` that use fake generated data for charts and visualizations. The hub backend already has real data flowing through it (feed sync history, catalog entry syncs, audit events, client metadata) but nobody wired it to the frontend.

Additionally, the license renewal button is non-functional and the license usage/audit tabs show empty placeholders.

## 2. Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Per-client sync tracking | New `client_sync_history` table | Precise sync logs with duration, entry counts, endpoint_count for trends |
| OS/status/compliance data | 3 JSONB columns on `clients` | Hub doesn't own endpoint-level data; PM sends summaries during sync |
| Endpoint trend chart | Reuse `client_sync_history.endpoint_count` | Natural time-series from sync rows, no extra table |
| License usage history | Join license → client → sync history | Reuse endpoint_count from sync history, no extra table |
| License audit trail | Query existing `audit_events` | Already captured by audit subscriber on every license event |
| License renewal | New `PUT /licenses/{id}/renew` endpoint | Edit terms + extend expiry, same key, RBAC-gated |
| Data freshness | Sync cadence (default 15 min) | Acceptable for POC; charts show "as of last sync" |

## 3. Database Changes

### 3.1 New Table: `client_sync_history`

Migration: `internal/hub/store/migrations/00012_client_sync_history.sql`

```sql
CREATE TABLE client_sync_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    client_id       UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    started_at      TIMESTAMPTZ NOT NULL,
    finished_at     TIMESTAMPTZ,
    duration_ms     INT,
    entries_delivered INT NOT NULL DEFAULT 0,
    deletes_delivered INT NOT NULL DEFAULT 0,
    endpoint_count  INT NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'success',
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_client_sync_history_lookup
    ON client_sync_history (tenant_id, client_id, started_at DESC);

ALTER TABLE client_sync_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON client_sync_history
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

### 3.2 Alter `clients` Table: 3 New JSONB Columns

Added to migration `00012_client_sync_history.sql`:

```sql
ALTER TABLE clients
    ADD COLUMN os_summary              JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN endpoint_status_summary JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN compliance_summary      JSONB NOT NULL DEFAULT '{}';
```

Both changes go in a single migration file `00012_client_sync_history.sql` to keep it atomic.

## 4. Backend API Changes

### 4.1 Modified: Sync Handler (`GET /api/v1/sync`)

The existing sync handler is enriched to:

1. Parse 3 new optional request headers from the Patch Manager:
   - `X-Os-Summary` — URL-encoded JSON (e.g. `{"windows":45,"linux":30,"macos":5}`)
   - `X-Endpoint-Status-Summary` — URL-encoded JSON (e.g. `{"connected":60,"disconnected":15,"stale":5}`)
   - `X-Compliance-Summary` — URL-encoded JSON (e.g. `{"NIST 800-53":87,"PCI-DSS":92}`)
2. Update `clients` row with `os_summary`, `endpoint_status_summary`, `compliance_summary` alongside existing `endpoint_count` and `last_sync_at` update.
3. Insert a `client_sync_history` row capturing: `client_id`, `started_at`, `finished_at`, `duration_ms`, `entries_delivered`, `deletes_delivered`, `endpoint_count`, `status`.

Headers are optional — if absent, JSONB columns retain their current values.

### 4.2 New Endpoints: Client Analytics

**`GET /api/v1/clients/{id}/sync-history`**

Query params: `limit` (default 20), `offset` (default 0).

Returns paginated sync history rows:
```json
{
  "items": [
    {
      "id": "uuid",
      "started_at": "2026-03-31T10:00:00Z",
      "finished_at": "2026-03-31T10:00:02Z",
      "duration_ms": 2100,
      "entries_delivered": 47,
      "deletes_delivered": 3,
      "endpoint_count": 80,
      "status": "success"
    }
  ],
  "total": 156
}
```

**`GET /api/v1/clients/{id}/endpoint-trend`**

Query params: `days` (default 90).

Returns aggregated endpoint count over time (one point per sync, or daily max if many syncs per day):
```json
{
  "points": [
    {"date": "2026-01-15", "endpoint_count": 52},
    {"date": "2026-01-16", "endpoint_count": 55}
  ]
}
```

Implementation: `SELECT DATE(started_at) as date, MAX(endpoint_count) as endpoint_count FROM client_sync_history WHERE client_id = $1 AND started_at > now() - interval '$2 days' GROUP BY DATE(started_at) ORDER BY date`.

### 4.3 New Endpoint: License Renewal

**`PUT /api/v1/licenses/{id}/renew`**

Request body:
```json
{
  "tier": "enterprise",
  "max_endpoints": 500,
  "expires_at": "2027-03-31T00:00:00Z"
}
```

`expires_at` is required (must be in the future). `tier` and `max_endpoints` are optional — only provided fields are updated. If the license was revoked, clears `revoked_at`.

Emits `license.renewed` domain event. RBAC-gated.

Response: updated license object (same format as `GET /licenses/{id}`).

### 4.4 New Endpoints: License Analytics

**`GET /api/v1/licenses/{id}/usage-history`**

Query params: `days` (default 90).

Joins: license → client → client_sync_history to get endpoint_count over time.
```json
{
  "max_endpoints": 500,
  "points": [
    {"date": "2026-01-15", "endpoint_count": 52},
    {"date": "2026-01-16", "endpoint_count": 55}
  ]
}
```

**`GET /api/v1/licenses/{id}/audit-trail`**

Query params: `limit` (default 50), `offset` (default 0).

Queries `audit_events WHERE resource = 'license' AND resource_id = $id`.
```json
{
  "items": [
    {
      "id": "ulid",
      "type": "license.issued",
      "actor_id": "uuid",
      "action": "issued",
      "payload": {},
      "timestamp": "2026-03-30T14:00:00Z"
    }
  ],
  "total": 12
}
```

## 5. Patch Manager Changes

### 5.1 Catalog Sync Worker Enrichment

File: `internal/server/workers/catalog_sync.go` (or equivalent hub-sync caller).

Before making the `GET hub/api/v1/sync?since=...` request, the PM runs 3 lightweight aggregate queries against its own database:

| Data | Query | Header |
|------|-------|--------|
| OS breakdown | `SELECT os_family, COUNT(*) FROM endpoints WHERE tenant_id = $1 GROUP BY os_family` | `X-Os-Summary` |
| Status breakdown | `SELECT status, COUNT(*) FROM endpoints WHERE tenant_id = $1 GROUP BY status` | `X-Endpoint-Status-Summary` |
| Compliance scores | `SELECT name, score FROM compliance_framework_scores WHERE tenant_id = $1 AND active = true` | `X-Compliance-Summary` |

Serialized as JSON, set as request headers. These are cheap queries that run once per sync cycle.

No new PM API endpoints. No new PM tables. No PM frontend changes.

## 6. Frontend Changes

### 6.1 New API Hooks (`web-hub/src/api/hooks/`)

| Hook | Endpoint | Type |
|------|----------|------|
| `useClientSyncHistory(clientId, limit)` | `GET /clients/{id}/sync-history` | Query |
| `useClientEndpointTrend(clientId, days)` | `GET /clients/{id}/endpoint-trend` | Query |
| `useLicenseUsageHistory(licenseId, days)` | `GET /licenses/{id}/usage-history` | Query |
| `useLicenseAuditTrail(licenseId, limit)` | `GET /licenses/{id}/audit-trail` | Query |
| `useRenewLicense()` | `PUT /licenses/{id}/renew` | Mutation |

### 6.2 ClientDetailPage.tsx Replacements

| Component | Fake → Real |
|-----------|-------------|
| `EndpointTrendChart` | `useClientEndpointTrend(id, 90)` → real line/bar chart |
| `SyncSuccessGrid` | `useClientSyncHistory(id, 20)` → real success/failure squares from `status` field |
| `SyncHistoryTab` | `useClientSyncHistory(id, 50)` → real table: timestamp, entries, duration, status |
| `OsDonutChart` | `client.os_summary` from existing GET response → real donut |
| `EndpointStatusBars` | `client.endpoint_status_summary` from existing GET response → real bars |
| `ComplianceBars` | `client.compliance_summary` from existing GET response → real bars |

### 6.3 ClientsPage.tsx Replacements

| Component | Fake → Real |
|-----------|-------------|
| `generateSyncEvents()` | `useClientSyncHistory(id, 5)` per expanded row |
| `OsBreakdownPie` | `client.os_summary` from list response |

### 6.4 LicenseDetailPage.tsx Replacements

| Component | Fake → Real |
|-----------|-------------|
| Renew button | Opens dialog with tier/max_endpoints/expires_at fields → `useRenewLicense()` mutation |
| Usage history tab | `useLicenseUsageHistory(id, 90)` → real line chart showing endpoint_count vs max_endpoints |
| Audit trail | `useLicenseAuditTrail(id, 50)` → real event timeline |

### 6.5 Empty States

Every chart/table shows:
- **Skeleton loader** while fetching
- **Empty state** if no data: "No sync data yet. Data appears after the first catalog sync."
- **Error state** if API fails

No fake data. No placeholders. Honest UI.

## 7. Data Flow

```
PM Catalog Sync Worker (every 15 min)
  │
  ├─ Queries own DB: OS breakdown, status breakdown, compliance scores
  │
  ├─ GET hub/api/v1/sync?since=...
  │   Headers: X-Endpoint-Count, X-Os-Summary, X-Endpoint-Status-Summary, X-Compliance-Summary
  │
  └─ Hub Sync Handler
      ├─ UPDATE clients SET os_summary, endpoint_status_summary, compliance_summary, endpoint_count, last_sync_at
      ├─ INSERT INTO client_sync_history (entries, deletes, endpoint_count, duration, status)
      ├─ Emit sync.completed event → audit_events
      └─ Return delta patches + deleted IDs (unchanged)

Frontend:
  ClientDetail  → client.{os,status,compliance}_summary (GET /clients/{id})
                → sync-history, endpoint-trend (new endpoints)
  ClientsList   → client.os_summary (GET /clients)
                → sync-history per expanded row
  LicenseDetail → PUT /licenses/{id}/renew (renewal dialog)
                → usage-history (join license→client→sync_history)
                → audit-trail (filter audit_events)
```

## 8. Scope Boundary

**In scope:**
- 1 new migration (table + columns)
- 6 new sqlc queries
- 5 new API endpoints + 1 modified sync handler
- 5 new TanStack Query hooks + 1 mutation
- 3 page rewires (ClientDetail, Clients, LicenseDetail)
- PM sync worker enrichment (3 headers)
- Register `license.renewed` in hub domain event topics

**Out of scope:**
- Feed sync history (already real)
- Dashboard queries (already real)
- web-agent (already complete)
- OpenAPI spec regeneration (separate PIQ-239)
- Hub auth (separate PIQ-12)
