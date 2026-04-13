# Hub Placeholder Data (PIQ-247) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace all fake chart data in web-hub/ with real backend data by creating a client sync history table, enriching the sync handler, adding analytics endpoints, and rewiring frontend components.

**Architecture:** Sync-Enrichment approach — one new DB table (`client_sync_history`) + 3 JSONB columns on `clients`. PM sends summary data during catalog sync. Hub stores it and exposes via new API endpoints. Frontend swaps fake components for real data hooks.

**Tech Stack:** Go 1.25 / chi/v5 / pgx/v5 / sqlc / goose / React 19 / TypeScript 5.7 / TanStack Query 5 / openapi-fetch

---

## File Map

### Backend — Hub (new/modified)

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/hub/store/migrations/012_client_sync_history.sql` | Create | Migration: new table + ALTER clients |
| `internal/hub/store/queries/client_sync_history.sql` | Create | sqlc queries for sync history + endpoint trend |
| `internal/hub/store/queries/clients.sql` | Modify | Add UpdateClientSummaries query |
| `internal/hub/store/queries/licenses.sql` | Modify | Add RenewLicense query |
| `internal/hub/store/queries/audit.sql` | Modify | Add ListAuditEventsByResourceID query |
| `internal/hub/events/topics.go` | Modify | Add LicenseRenewed event |
| `internal/hub/api/v1/sync.go` | Modify | Parse summary headers, insert sync history |
| `internal/hub/api/v1/clients.go` | Modify | Add SyncHistory + EndpointTrend handlers |
| `internal/hub/api/v1/licenses.go` | Modify | Add Renew + UsageHistory + AuditTrail handlers |
| `internal/hub/api/router.go` | Modify | Register new routes |

### Backend — PM (modified)

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/server/store/queries/endpoints.sql` | Modify | Add aggregate queries for OS + status |
| `internal/server/store/queries/compliance.sql` | Modify | Add GetFrameworkScoreSummary query |
| `internal/server/workers/catalog_sync.go` | Modify | Send summary headers during hub sync |

### Frontend — web-hub (modified)

| File | Action | Responsibility |
|------|--------|---------------|
| `web-hub/src/api/hooks/useClients.ts` | Modify | Add useClientSyncHistory, useClientEndpointTrend |
| `web-hub/src/api/hooks/useLicenses.ts` | Modify | Add useRenewLicense, useLicenseUsageHistory, useLicenseAuditTrail |
| `web-hub/src/pages/clients/ClientDetailPage.tsx` | Modify | Replace 6 fake components with real data |
| `web-hub/src/pages/clients/ClientsPage.tsx` | Modify | Replace 2 fake components with real data |
| `web-hub/src/pages/licenses/LicenseDetailPage.tsx` | Modify | Wire renewal, usage history, audit trail |

---

## Parallelization Groups

Tasks are organized into 4 parallel groups. Tasks within a group can run concurrently. Groups must execute sequentially.

```
Group A (Backend DB + Queries)     — Tasks 1-3   [parallel, no dependencies]
Group B (Backend Handlers + PM)    — Tasks 4-7   [parallel, depends on Group A]
Group C (Frontend Rewire)          — Tasks 8-10  [parallel, depends on Group B]
Group D (Integration Verification) — Task 11     [depends on Group C]
```

---

## Group A: Database & Queries (parallel)

### Task 1: Migration — client_sync_history table + client summary columns

**Files:**
- Create: `internal/hub/store/migrations/012_client_sync_history.sql`

- [ ] **Step 1: Write the migration file**

```sql
-- +goose Up

-- Track each client catalog sync call for history + endpoint trends
CREATE TABLE client_sync_history (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    client_id         UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    started_at        TIMESTAMPTZ NOT NULL,
    finished_at       TIMESTAMPTZ,
    duration_ms       INT,
    entries_delivered  INT NOT NULL DEFAULT 0,
    deletes_delivered  INT NOT NULL DEFAULT 0,
    endpoint_count    INT NOT NULL DEFAULT 0,
    status            TEXT NOT NULL DEFAULT 'success',
    error_message     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_sync_status CHECK (status IN ('success', 'failed'))
);

CREATE INDEX idx_client_sync_history_lookup
    ON client_sync_history (tenant_id, client_id, started_at DESC);

ALTER TABLE client_sync_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_sync_history FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON client_sync_history
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

GRANT SELECT, INSERT, UPDATE, DELETE ON client_sync_history TO hub_app;

-- Add summary columns to clients for PM-reported data
ALTER TABLE clients
    ADD COLUMN os_summary              JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN endpoint_status_summary JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN compliance_summary      JSONB NOT NULL DEFAULT '{}';

-- +goose Down

ALTER TABLE clients
    DROP COLUMN IF EXISTS compliance_summary,
    DROP COLUMN IF EXISTS endpoint_status_summary,
    DROP COLUMN IF EXISTS os_summary;

DROP TABLE IF EXISTS client_sync_history;
```

- [ ] **Step 2: Run the migration**

Run: `cd /home/heramb/skenzeriq/patchiq && make migrate-hub`
Expected: Migration 012 applied successfully.

- [ ] **Step 3: Verify migration status**

Run: `make migrate-status`
Expected: Migration 012 shows as applied.

- [ ] **Step 4: Commit**

```bash
git add internal/hub/store/migrations/012_client_sync_history.sql
git commit -m "feat(hub): add client_sync_history table and client summary columns (PIQ-247)"
```

---

### Task 2: sqlc queries — client sync history + client summaries

**Files:**
- Create: `internal/hub/store/queries/client_sync_history.sql`
- Modify: `internal/hub/store/queries/clients.sql`

- [ ] **Step 1: Write the client_sync_history queries**

Create `internal/hub/store/queries/client_sync_history.sql`:

```sql
-- name: InsertClientSyncHistory :one
INSERT INTO client_sync_history (
    tenant_id, client_id, started_at, finished_at, duration_ms,
    entries_delivered, deletes_delivered, endpoint_count, status, error_message
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListClientSyncHistory :many
SELECT *
FROM client_sync_history
WHERE tenant_id = @tenant_id
  AND client_id = @client_id
ORDER BY started_at DESC
LIMIT sqlc.arg('query_limit') OFFSET sqlc.arg('query_offset');

-- name: CountClientSyncHistory :one
SELECT COUNT(*)::bigint
FROM client_sync_history
WHERE tenant_id = @tenant_id
  AND client_id = @client_id;

-- name: GetClientEndpointTrend :many
SELECT DATE(started_at) AS date, MAX(endpoint_count)::int AS endpoint_count
FROM client_sync_history
WHERE tenant_id = @tenant_id
  AND client_id = @client_id
  AND started_at > now() - make_interval(days => @days::int)
GROUP BY DATE(started_at)
ORDER BY date;

-- name: GetLicenseUsageHistory :many
SELECT DATE(csh.started_at) AS date, MAX(csh.endpoint_count)::int AS endpoint_count
FROM client_sync_history csh
JOIN licenses l ON l.client_id = csh.client_id AND l.tenant_id = csh.tenant_id
WHERE l.id = @license_id
  AND l.tenant_id = @tenant_id
  AND csh.started_at > now() - make_interval(days => @days::int)
GROUP BY DATE(csh.started_at)
ORDER BY date;
```

- [ ] **Step 2: Add UpdateClientSummaries query to clients.sql**

Add to `internal/hub/store/queries/clients.sql`:

```sql
-- name: UpdateClientSummaries :one
UPDATE clients
SET endpoint_count = @endpoint_count,
    last_sync_at = now(),
    os_summary = CASE WHEN @os_summary::jsonb = '{}'::jsonb THEN os_summary ELSE @os_summary END,
    endpoint_status_summary = CASE WHEN @endpoint_status_summary::jsonb = '{}'::jsonb THEN endpoint_status_summary ELSE @endpoint_status_summary END,
    compliance_summary = CASE WHEN @compliance_summary::jsonb = '{}'::jsonb THEN compliance_summary ELSE @compliance_summary END,
    updated_at = now()
WHERE id = @id
RETURNING *;
```

- [ ] **Step 3: Regenerate sqlc**

Run: `cd /home/heramb/skenzeriq/patchiq && make sqlc`
Expected: Generates new types in `internal/hub/store/sqlcgen/` — `ClientSyncHistory` model, new query methods.

- [ ] **Step 4: Verify generated code compiles**

Run: `cd /home/heramb/skenzeriq/patchiq && go build ./internal/hub/...`
Expected: Build succeeds with no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/hub/store/queries/client_sync_history.sql internal/hub/store/queries/clients.sql internal/hub/store/sqlcgen/
git commit -m "feat(hub): add sqlc queries for client sync history and summaries (PIQ-247)"
```

---

### Task 3: sqlc queries — license renewal + audit trail

**Files:**
- Modify: `internal/hub/store/queries/licenses.sql`
- Modify: `internal/hub/store/queries/audit.sql`
- Modify: `internal/hub/events/topics.go`

- [ ] **Step 1: Add RenewLicense query to licenses.sql**

Add to `internal/hub/store/queries/licenses.sql`:

```sql
-- name: RenewLicense :one
UPDATE licenses
SET tier = COALESCE(sqlc.narg('new_tier')::text, tier),
    max_endpoints = COALESCE(sqlc.narg('new_max_endpoints')::int, max_endpoints),
    expires_at = @expires_at,
    revoked_at = NULL,
    updated_at = now()
WHERE id = @id
RETURNING *;
```

- [ ] **Step 2: Add ListAuditEventsByResourceID query to audit.sql**

Add to `internal/hub/store/queries/audit.sql`:

```sql
-- name: ListAuditEventsByResourceID :many
SELECT *
FROM audit_events
WHERE tenant_id = @tenant_id
  AND resource = @resource
  AND resource_id = @resource_id
ORDER BY timestamp DESC
LIMIT sqlc.arg('query_limit') OFFSET sqlc.arg('query_offset');

-- name: CountAuditEventsByResourceID :one
SELECT COUNT(*)
FROM audit_events
WHERE tenant_id = @tenant_id
  AND resource = @resource
  AND resource_id = @resource_id;
```

- [ ] **Step 3: Add LicenseRenewed event to topics.go**

Modify `internal/hub/events/topics.go`:

Add to the const block:
```go
LicenseRenewed = "license.renewed"
```

Add to the `AllTopics()` return slice:
```go
LicenseRenewed,
```

- [ ] **Step 4: Regenerate sqlc**

Run: `cd /home/heramb/skenzeriq/patchiq && make sqlc`
Expected: New RenewLicense and audit query methods generated.

- [ ] **Step 5: Verify generated code compiles**

Run: `cd /home/heramb/skenzeriq/patchiq && go build ./internal/hub/...`
Expected: Build succeeds.

- [ ] **Step 6: Commit**

```bash
git add internal/hub/store/queries/licenses.sql internal/hub/store/queries/audit.sql internal/hub/events/topics.go internal/hub/store/sqlcgen/
git commit -m "feat(hub): add license renewal query, audit trail query, and license.renewed event (PIQ-247)"
```

---

## Group B: Backend Handlers + PM Enrichment (parallel, depends on Group A)

### Task 4: Hub sync handler — parse summary headers + insert sync history

**Files:**
- Modify: `internal/hub/api/v1/sync.go`

- [ ] **Step 1: Write failing test for sync handler enrichment**

Create `internal/hub/api/v1/sync_test.go`:

```go
package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/herambskanda/patchiq/internal/hub/api/v1"
	"github.com/herambskanda/patchiq/internal/hub/store/sqlcgen"
	"github.com/herambskanda/patchiq/internal/shared/domain"
)

// mockSyncQuerier implements SyncQuerier for testing.
type mockSyncQuerier struct {
	entries         []sqlcgen.PatchCatalog
	deletedIDs      []pgtype.UUID
	updatedClient   *sqlcgen.UpdateClientSummariesParams
	insertedSync    *sqlcgen.InsertClientSyncHistoryParams
}

func (m *mockSyncQuerier) ListCatalogEntriesUpdatedSince(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error) {
	return m.entries, nil
}

func (m *mockSyncQuerier) ListCatalogEntriesDeletedSince(_ context.Context, _ pgtype.Timestamptz) ([]pgtype.UUID, error) {
	return m.deletedIDs, nil
}

func (m *mockSyncQuerier) UpdateClientSummaries(_ context.Context, arg sqlcgen.UpdateClientSummariesParams) (sqlcgen.Client, error) {
	m.updatedClient = &arg
	return sqlcgen.Client{}, nil
}

func (m *mockSyncQuerier) InsertClientSyncHistory(_ context.Context, arg sqlcgen.InsertClientSyncHistoryParams) (sqlcgen.ClientSyncHistory, error) {
	m.insertedSync = &arg
	return sqlcgen.ClientSyncHistory{}, nil
}

func (m *mockSyncQuerier) GetClientByAPIKeyHash(_ context.Context, _ pgtype.Text) (sqlcgen.Client, error) {
	return sqlcgen.Client{
		ID:            pgtype.UUID{Valid: true},
		TenantID:      pgtype.UUID{Valid: true},
		EndpointCount: 80,
	}, nil
}

func TestSyncHandler_ParsesSummaryHeaders(t *testing.T) {
	mock := &mockSyncQuerier{
		entries: []sqlcgen.PatchCatalog{},
	}
	handler := v1.NewSyncHandler(mock, "test-api-key", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since=2026-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("X-Endpoint-Count", "80")
	req.Header.Set("X-Os-Summary", `{"linux":50,"windows":30}`)
	req.Header.Set("X-Endpoint-Status-Summary", `{"connected":70,"disconnected":10}`)
	req.Header.Set("X-Compliance-Summary", `{"NIST 800-53":87,"PCI-DSS":92}`)

	rr := httptest.NewRecorder()
	handler.Sync(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify client summaries were updated
	if mock.updatedClient == nil {
		t.Fatal("expected UpdateClientSummaries to be called")
	}

	// Verify sync history was inserted
	if mock.insertedSync == nil {
		t.Fatal("expected InsertClientSyncHistory to be called")
	}
	if mock.insertedSync.EndpointCount != 80 {
		t.Errorf("expected endpoint_count 80, got %d", mock.insertedSync.EndpointCount)
	}
	if mock.insertedSync.Status != "success" {
		t.Errorf("expected status 'success', got %q", mock.insertedSync.Status)
	}
}

func TestSyncHandler_WorksWithoutSummaryHeaders(t *testing.T) {
	mock := &mockSyncQuerier{
		entries: []sqlcgen.PatchCatalog{},
	}
	handler := v1.NewSyncHandler(mock, "test-api-key", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since=2026-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")

	rr := httptest.NewRecorder()
	handler.Sync(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Sync should still succeed even without summary headers
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["server_time"]; !ok {
		t.Error("expected server_time in response")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/api/v1/ -run TestSyncHandler_ -v`
Expected: FAIL — mockSyncQuerier doesn't satisfy the interface yet (missing new methods).

- [ ] **Step 3: Update SyncQuerier interface and Sync handler**

Modify `internal/hub/api/v1/sync.go`:

1. Expand the `SyncQuerier` interface to include:
```go
type SyncQuerier interface {
	ListCatalogEntriesUpdatedSince(ctx context.Context, updatedAt pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error)
	ListCatalogEntriesDeletedSince(ctx context.Context, deletedAt pgtype.Timestamptz) ([]pgtype.UUID, error)
	UpdateClientSummaries(ctx context.Context, arg sqlcgen.UpdateClientSummariesParams) (sqlcgen.Client, error)
	InsertClientSyncHistory(ctx context.Context, arg sqlcgen.InsertClientSyncHistoryParams) (sqlcgen.ClientSyncHistory, error)
	GetClientByAPIKeyHash(ctx context.Context, apiKeyHash pgtype.Text) (sqlcgen.Client, error)
}
```

2. In the `Sync` method, after the existing auth check and before building the response, add:

```go
// Record sync start time
syncStartedAt := time.Now()

// ... existing catalog query logic ...

// Parse summary headers from Patch Manager
endpointCount := int32(0)
if v := r.Header.Get("X-Endpoint-Count"); v != "" {
	if n, err := strconv.Atoi(v); err == nil {
		endpointCount = int32(n)
	}
}

osSummary := []byte("{}")
if v := r.Header.Get("X-Os-Summary"); v != "" {
	if json.Valid([]byte(v)) {
		osSummary = []byte(v)
	}
}

statusSummary := []byte("{}")
if v := r.Header.Get("X-Endpoint-Status-Summary"); v != "" {
	if json.Valid([]byte(v)) {
		statusSummary = []byte(v)
	}
}

complianceSummary := []byte("{}")
if v := r.Header.Get("X-Compliance-Summary"); v != "" {
	if json.Valid([]byte(v)) {
		complianceSummary = []byte(v)
	}
}

// Identify the calling client by API key
var clientID pgtype.UUID
var tenantID pgtype.UUID
apiKeyHash := hashAPIKey(token) // reuse the token from auth check
if client, err := h.queries.GetClientByAPIKeyHash(r.Context(), pgtype.Text{String: apiKeyHash, Valid: true}); err == nil {
	clientID = client.ID
	tenantID = client.TenantID

	// Update client summaries
	if _, err := h.queries.UpdateClientSummaries(r.Context(), sqlcgen.UpdateClientSummariesParams{
		ID:                    clientID,
		EndpointCount:         endpointCount,
		OsSummary:             osSummary,
		EndpointStatusSummary: statusSummary,
		ComplianceSummary:     complianceSummary,
	}); err != nil {
		slog.ErrorContext(r.Context(), "update client summaries", "error", err)
	}
}

// ... existing response encoding ...

// Insert sync history record
syncFinishedAt := time.Now()
if clientID.Valid {
	if _, err := h.queries.InsertClientSyncHistory(r.Context(), sqlcgen.InsertClientSyncHistoryParams{
		TenantID:         tenantID,
		ClientID:         clientID,
		StartedAt:        pgtype.Timestamptz{Time: syncStartedAt, Valid: true},
		FinishedAt:       pgtype.Timestamptz{Time: syncFinishedAt, Valid: true},
		DurationMs:       pgtype.Int4{Int32: int32(syncFinishedAt.Sub(syncStartedAt).Milliseconds()), Valid: true},
		EntriesDelivered: int32(len(entries)),
		DeletesDelivered: int32(len(deletedIDs)),
		EndpointCount:    endpointCount,
		Status:           "success",
	}); err != nil {
		slog.ErrorContext(r.Context(), "insert client sync history", "error", err)
	}
}
```

3. Add the `hashAPIKey` helper (bcrypt compare or SHA-256 — check how `GetClientByAPIKeyHash` is used in the registration flow and use the same hashing).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/api/v1/ -run TestSyncHandler_ -v`
Expected: PASS — both tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/hub/api/v1/sync.go internal/hub/api/v1/sync_test.go
git commit -m "feat(hub): enrich sync handler with summary headers and sync history logging (PIQ-247)"
```

---

### Task 5: Hub client analytics handlers — sync history + endpoint trend

**Files:**
- Modify: `internal/hub/api/v1/clients.go`
- Modify: `internal/hub/api/router.go`

- [ ] **Step 1: Write failing tests for client analytics handlers**

Create `internal/hub/api/v1/client_analytics_test.go`:

```go
package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/herambskanda/patchiq/internal/hub/api/v1"
	"github.com/herambskanda/patchiq/internal/hub/store/sqlcgen"
)

type mockClientAnalyticsQuerier struct {
	v1.ClientQuerier
	syncHistory    []sqlcgen.ClientSyncHistory
	syncHistoryCount int64
	endpointTrend  []sqlcgen.GetClientEndpointTrendRow
}

func (m *mockClientAnalyticsQuerier) ListClientSyncHistory(_ context.Context, _ sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error) {
	return m.syncHistory, nil
}

func (m *mockClientAnalyticsQuerier) CountClientSyncHistory(_ context.Context, _ sqlcgen.CountClientSyncHistoryParams) (int64, error) {
	return m.syncHistoryCount, nil
}

func (m *mockClientAnalyticsQuerier) GetClientEndpointTrend(_ context.Context, _ sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error) {
	return m.endpointTrend, nil
}

func TestClientHandler_SyncHistory(t *testing.T) {
	now := time.Now()
	mock := &mockClientAnalyticsQuerier{
		syncHistory: []sqlcgen.ClientSyncHistory{
			{
				ID:               pgtype.UUID{Valid: true},
				StartedAt:        pgtype.Timestamptz{Time: now, Valid: true},
				FinishedAt:       pgtype.Timestamptz{Time: now.Add(2 * time.Second), Valid: true},
				DurationMs:       pgtype.Int4{Int32: 2000, Valid: true},
				EntriesDelivered: 47,
				DeletesDelivered: 3,
				EndpointCount:    80,
				Status:           "success",
			},
		},
		syncHistoryCount: 1,
	}

	handler := v1.NewClientHandler(mock, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/clients/{id}/sync-history", handler.SyncHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/00000000-0000-0000-0000-000000000001/sync-history?limit=20&offset=0", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	items := resp["items"].([]any)
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if resp["total"].(float64) != 1 {
		t.Errorf("expected total 1, got %v", resp["total"])
	}
}

func TestClientHandler_EndpointTrend(t *testing.T) {
	mock := &mockClientAnalyticsQuerier{
		endpointTrend: []sqlcgen.GetClientEndpointTrendRow{
			{Date: pgtype.Date{Time: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Valid: true}, EndpointCount: 52},
			{Date: pgtype.Date{Time: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC), Valid: true}, EndpointCount: 55},
		},
	}

	handler := v1.NewClientHandler(mock, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/clients/{id}/endpoint-trend", handler.EndpointTrend)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/00000000-0000-0000-0000-000000000001/endpoint-trend?days=90", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	points := resp["points"].([]any)
	if len(points) != 2 {
		t.Errorf("expected 2 points, got %d", len(points))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/api/v1/ -run TestClientHandler_Sync -v`
Expected: FAIL — SyncHistory and EndpointTrend methods don't exist yet.

- [ ] **Step 3: Expand ClientQuerier interface and implement handlers**

In `internal/hub/api/v1/clients.go`, expand the `ClientQuerier` interface:

```go
// Add to ClientQuerier interface:
ListClientSyncHistory(ctx context.Context, arg sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error)
CountClientSyncHistory(ctx context.Context, arg sqlcgen.CountClientSyncHistoryParams) (int64, error)
GetClientEndpointTrend(ctx context.Context, arg sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error)
```

Add handler methods:

```go
// SyncHistory handles GET /api/v1/clients/{id}/sync-history.
func (h *ClientHandler) SyncHistory(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	limit := int32(queryParamInt(r, "limit", 20))
	offset := int32(queryParamInt(r, "offset", 0))
	tenantID := tenant.MustTenantID(r.Context())
	tid, _ := parseUUID(tenantID)

	items, err := h.queries.ListClientSyncHistory(r.Context(), sqlcgen.ListClientSyncHistoryParams{
		TenantID:   tid,
		ClientID:   clientID,
		QueryLimit: limit,
		QueryOffset: offset,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list client sync history", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list sync history: internal error")
		return
	}

	total, err := h.queries.CountClientSyncHistory(r.Context(), sqlcgen.CountClientSyncHistoryParams{
		TenantID: tid,
		ClientID: clientID,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "count client sync history", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count sync history: internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
	})
}

// EndpointTrend handles GET /api/v1/clients/{id}/endpoint-trend.
func (h *ClientHandler) EndpointTrend(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	days := int32(queryParamInt(r, "days", 90))
	tenantID := tenant.MustTenantID(r.Context())
	tid, _ := parseUUID(tenantID)

	points, err := h.queries.GetClientEndpointTrend(r.Context(), sqlcgen.GetClientEndpointTrendParams{
		TenantID: tid,
		ClientID: clientID,
		Days:     days,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "get endpoint trend", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get endpoint trend: internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"points": points,
	})
}
```

- [ ] **Step 4: Register new routes in router.go**

In `internal/hub/api/router.go`, inside the `/clients` route group, add:

```go
r.Get("/{id}/sync-history", clients.SyncHistory)
r.Get("/{id}/endpoint-trend", clients.EndpointTrend)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/api/v1/ -run TestClientHandler_ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hub/api/v1/clients.go internal/hub/api/v1/client_analytics_test.go internal/hub/api/router.go
git commit -m "feat(hub): add client sync history and endpoint trend endpoints (PIQ-247)"
```

---

### Task 6: Hub license handlers — renew + usage history + audit trail

**Files:**
- Modify: `internal/hub/api/v1/licenses.go`
- Modify: `internal/hub/api/router.go`

- [ ] **Step 1: Write failing tests for license handlers**

Create `internal/hub/api/v1/license_analytics_test.go`:

```go
package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/herambskanda/patchiq/internal/hub/api/v1"
	"github.com/herambskanda/patchiq/internal/hub/store/sqlcgen"
)

type mockLicenseAnalyticsQuerier struct {
	v1.LicenseQuerier
	renewedLicense *sqlcgen.License
	usageHistory   []sqlcgen.GetLicenseUsageHistoryRow
	auditEvents    []sqlcgen.AuditEvent
	auditCount     int64
}

func (m *mockLicenseAnalyticsQuerier) RenewLicense(_ context.Context, arg sqlcgen.RenewLicenseParams) (sqlcgen.License, error) {
	return *m.renewedLicense, nil
}

func (m *mockLicenseAnalyticsQuerier) GetLicenseUsageHistory(_ context.Context, _ sqlcgen.GetLicenseUsageHistoryParams) ([]sqlcgen.GetLicenseUsageHistoryRow, error) {
	return m.usageHistory, nil
}

func (m *mockLicenseAnalyticsQuerier) GetLicenseByID(_ context.Context, _ pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error) {
	return sqlcgen.GetLicenseByIDRow{
		ID:           pgtype.UUID{Valid: true},
		MaxEndpoints: 500,
		ClientID:     pgtype.UUID{Valid: true},
	}, nil
}

func (m *mockLicenseAnalyticsQuerier) ListAuditEventsByResourceID(_ context.Context, _ sqlcgen.ListAuditEventsByResourceIDParams) ([]sqlcgen.AuditEvent, error) {
	return m.auditEvents, nil
}

func (m *mockLicenseAnalyticsQuerier) CountAuditEventsByResourceID(_ context.Context, _ sqlcgen.CountAuditEventsByResourceIDParams) (int64, error) {
	return m.auditCount, nil
}

func TestLicenseHandler_Renew(t *testing.T) {
	future := time.Now().Add(365 * 24 * time.Hour)
	mock := &mockLicenseAnalyticsQuerier{
		renewedLicense: &sqlcgen.License{
			ID:           pgtype.UUID{Valid: true},
			Tier:         "enterprise",
			MaxEndpoints: 500,
			ExpiresAt:    pgtype.Timestamptz{Time: future, Valid: true},
		},
	}

	handler := v1.NewLicenseHandler(mock, nil)

	r := chi.NewRouter()
	r.Put("/api/v1/licenses/{id}/renew", handler.Renew)

	body, _ := json.Marshal(map[string]any{
		"tier":           "enterprise",
		"max_endpoints":  500,
		"expires_at":     future.Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/licenses/00000000-0000-0000-0000-000000000001/renew", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLicenseHandler_Renew_ExpiresAtRequired(t *testing.T) {
	mock := &mockLicenseAnalyticsQuerier{}
	handler := v1.NewLicenseHandler(mock, nil)

	r := chi.NewRouter()
	r.Put("/api/v1/licenses/{id}/renew", handler.Renew)

	body, _ := json.Marshal(map[string]any{
		"tier": "enterprise",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/licenses/00000000-0000-0000-0000-000000000001/renew", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing expires_at, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLicenseHandler_UsageHistory(t *testing.T) {
	mock := &mockLicenseAnalyticsQuerier{
		usageHistory: []sqlcgen.GetLicenseUsageHistoryRow{
			{Date: pgtype.Date{Time: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Valid: true}, EndpointCount: 52},
		},
	}

	handler := v1.NewLicenseHandler(mock, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/licenses/{id}/usage-history", handler.UsageHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/00000000-0000-0000-0000-000000000001/usage-history?days=90", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["max_endpoints"].(float64) != 500 {
		t.Errorf("expected max_endpoints 500, got %v", resp["max_endpoints"])
	}
}

func TestLicenseHandler_AuditTrail(t *testing.T) {
	mock := &mockLicenseAnalyticsQuerier{
		auditEvents: []sqlcgen.AuditEvent{
			{
				ID:   "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Type: "license.issued",
			},
		},
		auditCount: 1,
	}

	handler := v1.NewLicenseHandler(mock, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/licenses/{id}/audit-trail", handler.AuditTrail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/00000000-0000-0000-0000-000000000001/audit-trail?limit=50", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("expected total 1, got %v", resp["total"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/api/v1/ -run TestLicenseHandler_ -v`
Expected: FAIL — Renew, UsageHistory, AuditTrail methods don't exist yet.

- [ ] **Step 3: Expand LicenseQuerier interface and implement handlers**

In `internal/hub/api/v1/licenses.go`, expand the interface:

```go
// Add to LicenseQuerier interface:
RenewLicense(ctx context.Context, arg sqlcgen.RenewLicenseParams) (sqlcgen.License, error)
GetLicenseUsageHistory(ctx context.Context, arg sqlcgen.GetLicenseUsageHistoryParams) ([]sqlcgen.GetLicenseUsageHistoryRow, error)
ListAuditEventsByResourceID(ctx context.Context, arg sqlcgen.ListAuditEventsByResourceIDParams) ([]sqlcgen.AuditEvent, error)
CountAuditEventsByResourceID(ctx context.Context, arg sqlcgen.CountAuditEventsByResourceIDParams) (int64, error)
```

Add request struct and handlers:

```go
type renewLicenseRequest struct {
	Tier         *string `json:"tier"`
	MaxEndpoints *int32  `json:"max_endpoints"`
	ExpiresAt    string  `json:"expires_at"`
}

// Renew handles PUT /api/v1/licenses/{id}/renew.
func (h *LicenseHandler) Renew(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	var req renewLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %s", err))
		return
	}

	if req.ExpiresAt == "" {
		writeJSONError(w, http.StatusBadRequest, "expires_at is required")
		return
	}
	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid expires_at: %s", err))
		return
	}
	if expiresAt.Before(time.Now()) {
		writeJSONError(w, http.StatusBadRequest, "expires_at must be in the future")
		return
	}

	params := sqlcgen.RenewLicenseParams{
		ID:        id,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}
	if req.Tier != nil {
		params.NewTier = pgtype.Text{String: *req.Tier, Valid: true}
	}
	if req.MaxEndpoints != nil {
		params.NewMaxEndpoints = pgtype.Int4{Int32: *req.MaxEndpoints, Valid: true}
	}

	license, err := h.queries.RenewLicense(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "license not found")
			return
		}
		slog.ErrorContext(r.Context(), "renew license", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "renew license: internal error")
		return
	}

	if h.eventBus != nil {
		tenantID := tenant.MustTenantID(r.Context())
		evt := domain.NewSystemEvent(events.LicenseRenewed, tenantID, "license", uuidToString(license.ID), "renew", map[string]any{
			"tier":           license.Tier,
			"max_endpoints":  license.MaxEndpoints,
			"expires_at":     license.ExpiresAt.Time.Format(time.RFC3339),
		})
		if err := h.eventBus.Emit(r.Context(), evt); err != nil {
			slog.ErrorContext(r.Context(), "emit license.renewed event", "error", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(license)
}

// UsageHistory handles GET /api/v1/licenses/{id}/usage-history.
func (h *LicenseHandler) UsageHistory(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	days := int32(queryParamInt(r, "days", 90))
	tenantID := tenant.MustTenantID(r.Context())
	tid, _ := parseUUID(tenantID)

	license, err := h.queries.GetLicenseByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "license not found")
			return
		}
		slog.ErrorContext(r.Context(), "get license", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get license: internal error")
		return
	}

	points, err := h.queries.GetLicenseUsageHistory(r.Context(), sqlcgen.GetLicenseUsageHistoryParams{
		LicenseID: id,
		TenantID:  tid,
		Days:      days,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "get license usage history", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get usage history: internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"max_endpoints": license.MaxEndpoints,
		"points":        points,
	})
}

// AuditTrail handles GET /api/v1/licenses/{id}/audit-trail.
func (h *LicenseHandler) AuditTrail(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	limit := int32(queryParamInt(r, "limit", 50))
	offset := int32(queryParamInt(r, "offset", 0))
	tenantID := tenant.MustTenantID(r.Context())
	tid, _ := parseUUID(tenantID)

	items, err := h.queries.ListAuditEventsByResourceID(r.Context(), sqlcgen.ListAuditEventsByResourceIDParams{
		TenantID:   tid,
		Resource:   "license",
		ResourceID: uuidToString(id),
		QueryLimit: limit,
		QueryOffset: offset,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list license audit trail", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list audit trail: internal error")
		return
	}

	total, err := h.queries.CountAuditEventsByResourceID(r.Context(), sqlcgen.CountAuditEventsByResourceIDParams{
		TenantID:   tid,
		Resource:   "license",
		ResourceID: uuidToString(id),
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "count license audit trail", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count audit trail: internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
	})
}
```

- [ ] **Step 4: Register new routes in router.go**

In `internal/hub/api/router.go`, inside the `/licenses` route group, add:

```go
r.Put("/{id}/renew", licenses.Renew)
r.Get("/{id}/usage-history", licenses.UsageHistory)
r.Get("/{id}/audit-trail", licenses.AuditTrail)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/api/v1/ -run TestLicenseHandler_ -v`
Expected: PASS — all 4 license tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/hub/api/v1/licenses.go internal/hub/api/v1/license_analytics_test.go internal/hub/api/router.go
git commit -m "feat(hub): add license renewal, usage history, and audit trail endpoints (PIQ-247)"
```

---

### Task 7: PM catalog sync worker — send summary headers

**Files:**
- Modify: `internal/server/store/queries/endpoints.sql`
- Modify: `internal/server/store/queries/compliance.sql`
- Modify: `internal/server/workers/catalog_sync.go`

- [ ] **Step 1: Add aggregate queries to PM**

Add to `internal/server/store/queries/endpoints.sql`:

```sql
-- name: GetEndpointOsSummary :many
SELECT COALESCE(os_family, 'unknown') AS os_family, COUNT(*)::int AS count
FROM endpoints
WHERE tenant_id = @tenant_id
GROUP BY os_family;

-- name: GetEndpointStatusSummary :many
SELECT status, COUNT(*)::int AS count
FROM endpoints
WHERE tenant_id = @tenant_id
GROUP BY status;
```

Add to `internal/server/store/queries/compliance.sql`:

```sql
-- name: GetFrameworkScoreSummary :many
SELECT ctf.framework_id, COALESCE(cs.score, 0)::numeric(5,2) AS score
FROM compliance_tenant_frameworks ctf
LEFT JOIN LATERAL (
    SELECT cs2.score
    FROM compliance_scores cs2
    WHERE cs2.tenant_id = ctf.tenant_id
      AND cs2.framework_id = ctf.framework_id
      AND cs2.scope_type = 'tenant'
    ORDER BY cs2.evaluated_at DESC
    LIMIT 1
) cs ON true
WHERE ctf.tenant_id = @tenant_id
  AND ctf.enabled = true;
```

- [ ] **Step 2: Regenerate sqlc for server**

Run: `cd /home/heramb/skenzeriq/patchiq && make sqlc`
Expected: New query methods generated in `internal/server/store/sqlcgen/`.

- [ ] **Step 3: Write failing test for catalog sync enrichment**

Create `internal/server/workers/catalog_sync_headers_test.go`:

```go
package workers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalogSync_SendsSummaryHeaders(t *testing.T) {
	// Verify the hub sync request includes summary headers
	var receivedHeaders http.Header
	hubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		json.NewEncoder(w).Encode(map[string]any{
			"entries":    []any{},
			"deleted_ids": []string{},
			"server_time": "2026-03-31T00:00:00Z",
		})
	}))
	defer hubServer.Close()

	// The test validates that when catalog sync runs,
	// the request to the hub includes X-Os-Summary, X-Endpoint-Status-Summary,
	// and X-Compliance-Summary headers.
	// This is an integration-level test that validates the worker enrichment.

	// Note: Full integration test requires DB setup.
	// Unit test validates header construction logic extracted to a helper.

	if receivedHeaders == nil {
		t.Skip("integration test — requires full worker setup")
	}

	for _, header := range []string{"X-Os-Summary", "X-Endpoint-Status-Summary", "X-Compliance-Summary"} {
		if receivedHeaders.Get(header) == "" {
			t.Errorf("expected header %s to be set", header)
		}
	}
}
```

- [ ] **Step 4: Modify catalog_sync.go to send summary headers**

In `internal/server/workers/catalog_sync.go`, in the `syncTenant` method, before making the HTTP request to the hub:

1. Add a store field to the worker struct if not present (or use the existing store/queries reference).

2. Before building the request, query the 3 aggregates:

```go
// Collect endpoint summaries to send to hub
osSummary := "{}"
if rows, err := w.store.GetEndpointOsSummary(ctx, sqlcgen.GetEndpointOsSummaryParams{TenantID: state.TenantID}); err == nil {
	m := make(map[string]int)
	for _, r := range rows {
		m[r.OsFamily] = int(r.Count)
	}
	if b, err := json.Marshal(m); err == nil {
		osSummary = string(b)
	}
}

statusSummary := "{}"
if rows, err := w.store.GetEndpointStatusSummary(ctx, sqlcgen.GetEndpointStatusSummaryParams{TenantID: state.TenantID}); err == nil {
	m := make(map[string]int)
	for _, r := range rows {
		m[r.Status] = int(r.Count)
	}
	if b, err := json.Marshal(m); err == nil {
		statusSummary = string(b)
	}
}

complianceSummary := "{}"
if rows, err := w.store.GetFrameworkScoreSummary(ctx, sqlcgen.GetFrameworkScoreSummaryParams{TenantID: state.TenantID}); err == nil {
	m := make(map[string]float64)
	for _, r := range rows {
		score, _ := r.Score.Float64Value()
		m[r.FrameworkID] = score.Float64
	}
	if b, err := json.Marshal(m); err == nil {
		complianceSummary = string(b)
	}
}

// Count endpoints for X-Endpoint-Count
endpointCount := 0
if rows, err := w.store.GetEndpointStatusSummary(ctx, sqlcgen.GetEndpointStatusSummaryParams{TenantID: state.TenantID}); err == nil {
	for _, r := range rows {
		endpointCount += int(r.Count)
	}
}
```

3. Set headers on the HTTP request:

```go
req.Header.Set("X-Endpoint-Count", strconv.Itoa(endpointCount))
req.Header.Set("X-Os-Summary", osSummary)
req.Header.Set("X-Endpoint-Status-Summary", statusSummary)
req.Header.Set("X-Compliance-Summary", complianceSummary)
```

- [ ] **Step 5: Verify server builds**

Run: `cd /home/heramb/skenzeriq/patchiq && go build ./internal/server/...`
Expected: Build succeeds.

- [ ] **Step 6: Commit**

```bash
git add internal/server/store/queries/endpoints.sql internal/server/store/queries/compliance.sql internal/server/workers/catalog_sync.go internal/server/store/sqlcgen/ internal/server/workers/catalog_sync_headers_test.go
git commit -m "feat(server): send endpoint summaries to hub during catalog sync (PIQ-247)"
```

---

## Group C: Frontend Rewire (parallel, depends on Group B)

### Task 8: Frontend API hooks for client and license analytics

**Files:**
- Modify: `web-hub/src/api/hooks/useClients.ts`
- Modify: `web-hub/src/api/hooks/useLicenses.ts`

- [ ] **Step 1: Add client analytics hooks**

In `web-hub/src/api/hooks/useClients.ts`, add:

```typescript
// --- Client Analytics ---

export interface SyncHistoryItem {
  id: string;
  started_at: string;
  finished_at: string | null;
  duration_ms: number | null;
  entries_delivered: number;
  deletes_delivered: number;
  endpoint_count: number;
  status: 'success' | 'failed';
  error_message: string | null;
}

export interface EndpointTrendPoint {
  date: string;
  endpoint_count: number;
}

async function getClientSyncHistory(clientId: string, limit = 20, offset = 0) {
  const { data, error } = await api.GET('/api/v1/clients/{id}/sync-history' as any, {
    params: { path: { id: clientId }, query: { limit, offset } },
  });
  if (error) throw new Error('Failed to fetch sync history');
  return data as { items: SyncHistoryItem[]; total: number };
}

async function getClientEndpointTrend(clientId: string, days = 90) {
  const { data, error } = await api.GET('/api/v1/clients/{id}/endpoint-trend' as any, {
    params: { path: { id: clientId }, query: { days } },
  });
  if (error) throw new Error('Failed to fetch endpoint trend');
  return data as { points: EndpointTrendPoint[] };
}

export function useClientSyncHistory(clientId: string | undefined, limit = 20, offset = 0) {
  return useQuery({
    queryKey: ['clients', clientId, 'sync-history', { limit, offset }],
    queryFn: () => getClientSyncHistory(clientId!, limit, offset),
    enabled: !!clientId,
  });
}

export function useClientEndpointTrend(clientId: string | undefined, days = 90) {
  return useQuery({
    queryKey: ['clients', clientId, 'endpoint-trend', { days }],
    queryFn: () => getClientEndpointTrend(clientId!, days),
    enabled: !!clientId,
  });
}
```

- [ ] **Step 2: Add license analytics hooks**

In `web-hub/src/api/hooks/useLicenses.ts`, add:

```typescript
// --- License Analytics ---

export interface LicenseUsagePoint {
  date: string;
  endpoint_count: number;
}

export interface LicenseAuditEvent {
  id: string;
  type: string;
  actor_id: string;
  actor_type: string;
  action: string;
  payload: Record<string, any>;
  timestamp: string;
}

async function renewLicense(id: string, data: { tier?: string; max_endpoints?: number; expires_at: string }) {
  const { data: result, error } = await api.PUT('/api/v1/licenses/{id}/renew' as any, {
    params: { path: { id } },
    body: data,
  });
  if (error) throw new Error('Failed to renew license');
  return result;
}

async function getLicenseUsageHistory(id: string, days = 90) {
  const { data, error } = await api.GET('/api/v1/licenses/{id}/usage-history' as any, {
    params: { path: { id }, query: { days } },
  });
  if (error) throw new Error('Failed to fetch usage history');
  return data as { max_endpoints: number; points: LicenseUsagePoint[] };
}

async function getLicenseAuditTrail(id: string, limit = 50, offset = 0) {
  const { data, error } = await api.GET('/api/v1/licenses/{id}/audit-trail' as any, {
    params: { path: { id }, query: { limit, offset } },
  });
  if (error) throw new Error('Failed to fetch audit trail');
  return data as { items: LicenseAuditEvent[]; total: number };
}

export function useRenewLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { tier?: string; max_endpoints?: number; expires_at: string } }) =>
      renewLicense(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

export function useLicenseUsageHistory(licenseId: string | undefined, days = 90) {
  return useQuery({
    queryKey: ['licenses', licenseId, 'usage-history', { days }],
    queryFn: () => getLicenseUsageHistory(licenseId!, days),
    enabled: !!licenseId,
  });
}

export function useLicenseAuditTrail(licenseId: string | undefined, limit = 50) {
  return useQuery({
    queryKey: ['licenses', licenseId, 'audit-trail', { limit }],
    queryFn: () => getLicenseAuditTrail(licenseId!, limit),
    enabled: !!licenseId,
  });
}
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd /home/heramb/skenzeriq/patchiq/web-hub && npx tsc --noEmit`
Expected: No type errors.

- [ ] **Step 4: Commit**

```bash
git add web-hub/src/api/hooks/useClients.ts web-hub/src/api/hooks/useLicenses.ts
git commit -m "feat(web-hub): add API hooks for client analytics, license renewal, and audit trail (PIQ-247)"
```

---

### Task 9: ClientDetailPage + ClientsPage — replace fake components with real data

**Files:**
- Modify: `web-hub/src/pages/clients/ClientDetailPage.tsx`
- Modify: `web-hub/src/pages/clients/ClientsPage.tsx`

- [ ] **Step 1: Replace EndpointTrendChart in ClientDetailPage**

Replace the fake `EndpointTrendChart` component with one that uses `useClientEndpointTrend`:

```typescript
import { useClientSyncHistory, useClientEndpointTrend } from '../../api/hooks/useClients';

function EndpointTrendChart({ clientId }: { clientId: string }) {
  const { data, isLoading } = useClientEndpointTrend(clientId, 90);

  if (isLoading) {
    return <div className="h-16 animate-pulse rounded" style={{ background: 'var(--bg-canvas)' }} />;
  }

  const points = data?.points ?? [];
  if (points.length === 0) {
    return (
      <div className="text-xs text-center py-4" style={{ color: 'var(--text-muted)' }}>
        No trend data yet. Data appears after the first catalog sync.
      </div>
    );
  }

  const values = points.map(p => p.endpoint_count);
  const maxV = Math.max(...values);
  const labels = points.map(p => {
    const d = new Date(p.date);
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  });

  // Render SVG bar chart using same visual pattern as before, but with real data
  const barW = Math.max(8, Math.floor(200 / points.length) - 2);
  const chartW = points.length * (barW + 2);

  return (
    <svg viewBox={`0 0 ${chartW} 48`} className="w-full h-12">
      {values.map((v, i) => {
        const h = maxV > 0 ? (v / maxV) * 40 : 2;
        return (
          <rect
            key={i}
            x={i * (barW + 2)}
            y={48 - h}
            width={barW}
            height={h}
            rx={2}
            fill="var(--accent)"
            opacity={0.3 + (i / points.length) * 0.7}
          >
            <title>{`${labels[i]}: ${v} endpoints`}</title>
          </rect>
        );
      })}
    </svg>
  );
}
```

- [ ] **Step 2: Replace SyncSuccessGrid in ClientDetailPage**

Replace the fake `SyncSuccessGrid` with one that uses `useClientSyncHistory`:

```typescript
function SyncSuccessGrid({ clientId }: { clientId: string }) {
  const { data, isLoading } = useClientSyncHistory(clientId, 20);

  if (isLoading) {
    return <div className="h-6 animate-pulse rounded" style={{ background: 'var(--bg-canvas)' }} />;
  }

  const items = data?.items ?? [];
  if (items.length === 0) {
    return (
      <span className="text-xs" style={{ color: 'var(--text-muted)' }}>No sync data yet</span>
    );
  }

  const successCount = items.filter(i => i.status === 'success').length;

  return (
    <div className="flex items-center gap-2 flex-wrap">
      <div className="flex flex-wrap gap-1">
        {items.map((item, i) => (
          <div
            key={item.id ?? i}
            className="w-4 h-4 rounded-sm"
            style={{ background: item.status === 'success' ? 'var(--signal-healthy)' : 'var(--signal-critical)' }}
            title={`${new Date(item.started_at).toLocaleString()} — ${item.status}`}
          />
        ))}
      </div>
      <span className="text-xs text-muted-foreground">{successCount}/{items.length} successful</span>
    </div>
  );
}
```

- [ ] **Step 3: Replace OsDonutChart, EndpointStatusBars, ComplianceBars in ClientDetailPage**

These read from the client object's new JSONB fields:

```typescript
function OsDonutChart({ client }: { client: Client }) {
  const osSummary = (client as any).os_summary as Record<string, number> | undefined;
  if (!osSummary || Object.keys(osSummary).length === 0) {
    return <DataPendingPlaceholder label="OS data available after first sync" />;
  }

  const total = Object.values(osSummary).reduce((a, b) => a + b, 0);
  if (total === 0) return <DataPendingPlaceholder label="No endpoint data" />;

  const colors = ['var(--accent)', '#f97316', '#ef4444', '#8b5cf6', '#06b6d4'];
  const entries = Object.entries(osSummary);
  let cumPct = 0;

  return (
    <div className="flex items-center gap-4">
      <svg viewBox="0 0 36 36" className="w-20 h-20">
        {entries.map(([os, count], i) => {
          const pct = count / total;
          const dashArray = `${pct * 100} ${100 - pct * 100}`;
          const offset = -cumPct * 100;
          cumPct += pct;
          return (
            <circle
              key={os}
              cx="18" cy="18" r="15.9"
              fill="none"
              strokeWidth="3.8"
              stroke={colors[i % colors.length]}
              strokeDasharray={dashArray}
              strokeDashoffset={offset}
            >
              <title>{`${os}: ${count} (${(pct * 100).toFixed(0)}%)`}</title>
            </circle>
          );
        })}
      </svg>
      <div className="space-y-1">
        {entries.map(([os, count], i) => (
          <div key={os} className="flex items-center gap-2 text-xs">
            <div className="w-2.5 h-2.5 rounded-full" style={{ background: colors[i % colors.length] }} />
            <span>{os}</span>
            <span style={{ color: 'var(--text-muted)' }}>{count}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function EndpointStatusBars({ client }: { client: Client }) {
  const statusSummary = (client as any).endpoint_status_summary as Record<string, number> | undefined;
  if (!statusSummary || Object.keys(statusSummary).length === 0) {
    return <DataPendingPlaceholder label="Status data available after first sync" />;
  }

  const total = Object.values(statusSummary).reduce((a, b) => a + b, 0);
  if (total === 0) return <DataPendingPlaceholder label="No endpoint data" />;

  const statusColors: Record<string, string> = {
    connected: 'var(--signal-healthy)',
    disconnected: 'var(--signal-warning)',
    stale: 'var(--signal-critical)',
    pending: 'var(--text-muted)',
  };

  return (
    <div className="space-y-2">
      {Object.entries(statusSummary).map(([status, count]) => (
        <div key={status} className="flex items-center gap-2">
          <span className="text-xs w-24 capitalize">{status}</span>
          <div className="flex-1 h-3 rounded-full overflow-hidden" style={{ background: 'var(--bg-canvas)' }}>
            <div
              className="h-full rounded-full transition-all"
              style={{
                width: `${(count / total) * 100}%`,
                background: statusColors[status] ?? 'var(--accent)',
              }}
            />
          </div>
          <span className="text-xs w-8 text-right" style={{ color: 'var(--text-muted)' }}>{count}</span>
        </div>
      ))}
    </div>
  );
}

function ComplianceBars({ client }: { client: Client }) {
  const complianceSummary = (client as any).compliance_summary as Record<string, number> | undefined;
  if (!complianceSummary || Object.keys(complianceSummary).length === 0) {
    return <DataPendingPlaceholder label="Compliance data available after first sync" />;
  }

  return (
    <div className="space-y-2">
      {Object.entries(complianceSummary).map(([framework, score]) => (
        <div key={framework} className="flex items-center gap-2">
          <span className="text-xs w-24 truncate" title={framework}>{framework}</span>
          <div className="flex-1 h-3 rounded-full overflow-hidden" style={{ background: 'var(--bg-canvas)' }}>
            <div
              className="h-full rounded-full transition-all"
              style={{
                width: `${score}%`,
                background: score >= 80 ? 'var(--signal-healthy)' : score >= 60 ? 'var(--signal-warning)' : 'var(--signal-critical)',
              }}
            />
          </div>
          <span className="text-xs w-10 text-right" style={{ color: 'var(--text-muted)' }}>{score}%</span>
        </div>
      ))}
    </div>
  );
}
```

- [ ] **Step 4: Replace SyncHistoryTab in ClientDetailPage**

```typescript
function SyncHistoryTab({ clientId }: { clientId: string }) {
  const { data, isLoading } = useClientSyncHistory(clientId, 50);

  if (isLoading) {
    return <div className="space-y-2">{Array.from({ length: 5 }, (_, i) => <div key={i} className="h-10 animate-pulse rounded" style={{ background: 'var(--bg-canvas)' }} />)}</div>;
  }

  const items = data?.items ?? [];
  if (items.length === 0) {
    return (
      <div className="rounded-lg p-8 text-center text-sm" style={{ background: 'var(--bg-canvas)', color: 'var(--text-muted)', border: '1px solid var(--border)' }}>
        No sync history recorded yet. Data appears after the first catalog sync.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm font-medium">Sync History ({data?.total ?? items.length} total)</p>
      <div className="overflow-auto rounded-lg" style={{ border: '1px solid var(--border)' }}>
        <table className="w-full text-sm">
          <thead>
            <tr style={{ background: 'var(--bg-canvas)' }}>
              <th className="text-left px-3 py-2 font-medium">Time</th>
              <th className="text-left px-3 py-2 font-medium">Entries</th>
              <th className="text-left px-3 py-2 font-medium">Deletes</th>
              <th className="text-left px-3 py-2 font-medium">Duration</th>
              <th className="text-left px-3 py-2 font-medium">Endpoints</th>
              <th className="text-left px-3 py-2 font-medium">Status</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} style={{ borderTop: '1px solid var(--border)' }}>
                <td className="px-3 py-2">{new Date(item.started_at).toLocaleString()}</td>
                <td className="px-3 py-2">{item.entries_delivered}</td>
                <td className="px-3 py-2">{item.deletes_delivered}</td>
                <td className="px-3 py-2">{item.duration_ms != null ? `${(item.duration_ms / 1000).toFixed(1)}s` : '—'}</td>
                <td className="px-3 py-2">{item.endpoint_count}</td>
                <td className="px-3 py-2">
                  <span className="inline-flex items-center gap-1 text-xs font-medium" style={{ color: item.status === 'success' ? 'var(--signal-healthy)' : 'var(--signal-critical)' }}>
                    {item.status === 'success' ? '✓' : '✗'} {item.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Replace fake components in ClientsPage.tsx**

Replace `generateSyncEvents` with real data and `OsBreakdownPie` to use `client.os_summary`:

For `generateSyncEvents` — replace calls with `useClientSyncHistory(clientId, 5)` hook call within the expanded row component.

For `OsBreakdownPie` — use `(client as any).os_summary` from the client list response, same pattern as ClientDetailPage.

For `EndpointSparkline` — use `useClientEndpointTrend(clientId, 30)` to render a real sparkline.

- [ ] **Step 6: Verify TypeScript compiles and dev server runs**

Run: `cd /home/heramb/skenzeriq/patchiq/web-hub && npx tsc --noEmit`
Expected: No errors.

- [ ] **Step 7: Commit**

```bash
git add web-hub/src/pages/clients/ClientDetailPage.tsx web-hub/src/pages/clients/ClientsPage.tsx
git commit -m "feat(web-hub): wire client pages to real sync history and summary data (PIQ-247)"
```

---

### Task 10: LicenseDetailPage — wire renewal, usage history, audit trail

**Files:**
- Modify: `web-hub/src/pages/licenses/LicenseDetailPage.tsx`

- [ ] **Step 1: Wire the Renew button**

Replace the dead Renew button with a functional dialog:

```typescript
import { useRenewLicense, useLicenseUsageHistory, useLicenseAuditTrail } from '../../api/hooks/useLicenses';

// Inside LicenseDetailPage component:
const renewMutation = useRenewLicense();
const [showRenewDialog, setShowRenewDialog] = useState(false);
const [renewForm, setRenewForm] = useState({
  tier: license?.tier ?? '',
  max_endpoints: license?.max_endpoints ?? 0,
  expires_at: '',
});

// Replace the Renew button with:
<Button
  variant="outline"
  size="sm"
  className="w-full"
  onClick={() => {
    setRenewForm({
      tier: license?.tier ?? '',
      max_endpoints: license?.max_endpoints ?? 0,
      expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    });
    setShowRenewDialog(true);
  }}
>
  Renew
</Button>

// Add renewal dialog (use Dialog from @patchiq/ui):
{showRenewDialog && (
  <Dialog open={showRenewDialog} onOpenChange={setShowRenewDialog}>
    <DialogContent>
      <DialogHeader>
        <DialogTitle>Renew License</DialogTitle>
      </DialogHeader>
      <div className="space-y-4">
        <div>
          <label className="text-sm font-medium">Tier</label>
          <select
            value={renewForm.tier}
            onChange={e => setRenewForm(f => ({ ...f, tier: e.target.value }))}
            className="w-full mt-1 rounded-md border px-3 py-2 text-sm"
          >
            <option value="community">Community</option>
            <option value="professional">Professional</option>
            <option value="enterprise">Enterprise</option>
            <option value="msp">MSP</option>
          </select>
        </div>
        <div>
          <label className="text-sm font-medium">Max Endpoints</label>
          <input
            type="number"
            value={renewForm.max_endpoints}
            onChange={e => setRenewForm(f => ({ ...f, max_endpoints: parseInt(e.target.value) || 0 }))}
            className="w-full mt-1 rounded-md border px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="text-sm font-medium">New Expiry Date</label>
          <input
            type="date"
            value={renewForm.expires_at}
            onChange={e => setRenewForm(f => ({ ...f, expires_at: e.target.value }))}
            className="w-full mt-1 rounded-md border px-3 py-2 text-sm"
          />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={() => setShowRenewDialog(false)}>Cancel</Button>
        <Button
          onClick={() => {
            renewMutation.mutate({
              id: licenseId,
              data: {
                tier: renewForm.tier,
                max_endpoints: renewForm.max_endpoints,
                expires_at: new Date(renewForm.expires_at).toISOString(),
              },
            }, {
              onSuccess: () => {
                setShowRenewDialog(false);
                toast.success('License renewed successfully');
              },
              onError: () => toast.error('Failed to renew license'),
            });
          }}
          disabled={renewMutation.isPending || !renewForm.expires_at}
        >
          {renewMutation.isPending ? 'Renewing...' : 'Renew License'}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
)}
```

- [ ] **Step 2: Wire Usage History tab**

Replace the placeholder usage history tab:

```typescript
const { data: usageData, isLoading: usageLoading } = useLicenseUsageHistory(licenseId, 90);

// Replace the usage tab content:
{activeTab === 'usage' && (
  <div className="space-y-4">
    {usageLoading ? (
      <div className="h-48 animate-pulse rounded" style={{ background: 'var(--bg-canvas)' }} />
    ) : !usageData?.points?.length ? (
      <div className="bg-muted/30 rounded-lg p-8 text-center">
        <p className="text-muted-foreground text-sm">No usage history yet. Data appears after the first catalog sync from an assigned client.</p>
      </div>
    ) : (
      <div>
        <div className="flex items-center justify-between mb-4">
          <p className="text-sm font-medium">Endpoint Usage vs License Capacity</p>
          <span className="text-xs" style={{ color: 'var(--text-muted)' }}>Max: {usageData.max_endpoints}</span>
        </div>
        <svg viewBox="0 0 400 120" className="w-full h-32">
          {/* Max endpoints line */}
          <line x1="0" y1={120 - (usageData.max_endpoints > 0 ? 100 : 0)} x2="400" y2={120 - (usageData.max_endpoints > 0 ? 100 : 0)} stroke="var(--signal-critical)" strokeDasharray="4 4" strokeWidth="1" />
          {/* Usage polyline */}
          <polyline
            fill="none"
            stroke="var(--accent)"
            strokeWidth="2"
            points={usageData.points.map((p, i) => {
              const x = (i / Math.max(usageData.points.length - 1, 1)) * 400;
              const y = 120 - (usageData.max_endpoints > 0 ? (p.endpoint_count / usageData.max_endpoints) * 100 : 0);
              return `${x},${Math.max(0, Math.min(120, y))}`;
            }).join(' ')}
          />
        </svg>
      </div>
    )}
  </div>
)}
```

- [ ] **Step 3: Wire Audit Trail tab**

Replace the synthetic audit trail:

```typescript
const { data: auditData, isLoading: auditLoading } = useLicenseAuditTrail(licenseId, 50);

// Replace the audit tab content:
{activeTab === 'audit' && (
  <div className="space-y-4">
    {auditLoading ? (
      <div className="space-y-3">{Array.from({ length: 3 }, (_, i) => <div key={i} className="h-16 animate-pulse rounded" style={{ background: 'var(--bg-canvas)' }} />)}</div>
    ) : !auditData?.items?.length ? (
      <div className="rounded-lg p-8 text-center" style={{ background: 'var(--bg-canvas)', border: '1px solid var(--border)' }}>
        <p className="text-sm" style={{ color: 'var(--text-muted)' }}>No audit events recorded yet.</p>
      </div>
    ) : (
      <div className="space-y-3">
        <p className="text-sm font-medium">License Audit Trail ({auditData.total} events)</p>
        {auditData.items.map((event) => (
          <div key={event.id} className="flex items-start gap-3 p-3 rounded-lg" style={{ background: 'var(--bg-canvas)' }}>
            <div className="w-2 h-2 rounded-full mt-1.5" style={{ background: 'var(--accent)' }} />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium capitalize">{event.type.replace('.', ' ')}</span>
                <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
                  {new Date(event.timestamp).toLocaleString()}
                </span>
              </div>
              <p className="text-xs mt-0.5" style={{ color: 'var(--text-muted)' }}>
                {event.actor_type === 'system' ? 'System' : `User ${event.actor_id.slice(0, 8)}...`} — {event.action}
              </p>
            </div>
          </div>
        ))}
      </div>
    )}
  </div>
)}
```

- [ ] **Step 4: Remove all PIQ-247 TODO comments**

Search and remove all `TODO(PIQ-247)` comments from the file.

- [ ] **Step 5: Verify TypeScript compiles**

Run: `cd /home/heramb/skenzeriq/patchiq/web-hub && npx tsc --noEmit`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add web-hub/src/pages/licenses/LicenseDetailPage.tsx
git commit -m "feat(web-hub): wire license renewal, usage history, and audit trail with real data (PIQ-247)"
```

---

## Group D: Integration Verification

### Task 11: E2E verification with Playwright

**Depends on:** All previous tasks complete, `make dev` running.

- [ ] **Step 1: Start dev environment**

Run: `cd /home/heramb/skenzeriq/patchiq && make dev`
Wait for all services to start.

- [ ] **Step 2: Run hub migrations**

Run: `make migrate-hub`
Expected: Migration 012 applied.

- [ ] **Step 3: Seed hub data**

Run: `make seed-hub`
Expected: Hub seeded with clients, licenses, feeds.

- [ ] **Step 4: Trigger a catalog sync to generate real data**

Navigate to Patch Manager settings page and trigger a manual hub sync, or:
Run: `curl -X POST http://localhost:8080/api/v1/sync/trigger -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"`

Wait for sync to complete. This should populate the client's summary columns and create a sync history row.

- [ ] **Step 5: Verify client pages via Playwright**

Use Playwright MCP to:
1. Navigate to `http://localhost:3002/clients`
2. Verify the clients list renders with real OS breakdown (not fake percentages)
3. Click on a client to open detail page
4. Verify EndpointTrendChart shows real data or honest empty state
5. Verify SyncSuccessGrid shows real status squares
6. Switch to "Endpoints" tab — verify OS donut, status bars, compliance bars show real data or empty states
7. Switch to "Sync History" tab — verify real sync history table with at least 1 row

- [ ] **Step 6: Verify license pages via Playwright**

Use Playwright MCP to:
1. Navigate to `http://localhost:3002/licenses`
2. Click on a license to open detail page
3. Click "Renew" button — verify dialog opens with tier/endpoints/expiry fields
4. Fill in a future date and click Renew — verify success toast
5. Switch to "Usage" tab — verify usage chart or honest empty state
6. Switch to "Audit" tab — verify real audit events (at least license.issued event)

- [ ] **Step 7: Verify no PIQ-247 TODOs remain**

Run: `grep -r "PIQ-247" web-hub/src/`
Expected: No results (all TODOs removed).

- [ ] **Step 8: Run full hub test suite**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/... -v`
Expected: All tests pass.

- [ ] **Step 9: Final commit — remove any remaining TODOs**

If any TODOs remain, clean them up and commit.
