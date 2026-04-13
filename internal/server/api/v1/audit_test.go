package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuditQuerier mocks AuditQuerier.
type fakeAuditQuerier struct {
	listResult  []sqlcgen.AuditEvent
	listErr     error
	countResult int64
	countErr    error
	listCalls   int  // tracks number of ListAuditEventsFiltered calls
	onceOnly    bool // if true, return empty slice after first call
}

func (f *fakeAuditQuerier) ListAuditEventsFiltered(_ context.Context, _ sqlcgen.ListAuditEventsFilteredParams) ([]sqlcgen.AuditEvent, error) {
	f.listCalls++
	if f.onceOnly && f.listCalls > 1 {
		return []sqlcgen.AuditEvent{}, f.listErr
	}
	return f.listResult, f.listErr
}

func (f *fakeAuditQuerier) CountAuditEventsFiltered(_ context.Context, _ sqlcgen.CountAuditEventsFilteredParams) (int64, error) {
	return f.countResult, f.countErr
}

func validAuditEvent() sqlcgen.AuditEvent {
	var tid pgtype.UUID
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.AuditEvent{
		ID:         "01JTEST000000000000000000",
		TenantID:   tid,
		Type:       "endpoint.created",
		ActorID:    "user-123",
		ActorType:  "user",
		Resource:   "endpoint",
		ResourceID: "res-456",
		Action:     "create",
		Payload:    []byte("{}"),
		Metadata:   []byte("{}"),
		Timestamp:  pgtype.Timestamptz{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}
}

func TestAuditHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *fakeAuditQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns empty list",
			querier: &fakeAuditQuerier{
				listResult:  []sqlcgen.AuditEvent{},
				countResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "returns audit events",
			querier: &fakeAuditQuerier{
				listResult:  []sqlcgen.AuditEvent{validAuditEvent()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "store error returns 500",
			querier: &fakeAuditQuerier{
				listErr: fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
		{
			name:  "filter by actor_type",
			query: "?actor_type=user",
			querier: &fakeAuditQuerier{
				listResult:  []sqlcgen.AuditEvent{validAuditEvent()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by resource",
			query: "?resource=endpoint",
			querier: &fakeAuditQuerier{
				listResult:  []sqlcgen.AuditEvent{validAuditEvent()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by date range",
			query: "?from_date=2026-01-01T00:00:00Z&to_date=2026-12-31T23:59:59Z",
			querier: &fakeAuditQuerier{
				listResult:  []sqlcgen.AuditEvent{validAuditEvent()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "invalid from_date returns 400",
			query:      "?from_date=not-a-date",
			querier:    &fakeAuditQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
		{
			name:       "invalid to_date returns 400",
			query:      "?to_date=not-a-date",
			querier:    &fakeAuditQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
		{
			name: "count error returns 500",
			querier: &fakeAuditQuerier{
				listResult: []sqlcgen.AuditEvent{},
				countErr:   fmt.Errorf("count failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
		{
			name:       "invalid cursor returns 400",
			query:      "?cursor=bad-cursor",
			querier:    &fakeAuditQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewAuditHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit"+tt.query, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantLen >= 0 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body["data"], tt.wantLen)
			}
		})
	}
}

func TestAuditHandler_Export_CSV(t *testing.T) {
	q := &fakeAuditQuerier{
		listResult: []sqlcgen.AuditEvent{validAuditEvent()},
		onceOnly:   true,
	}
	h := v1.NewAuditHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?format=csv", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.Export(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/csv", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "audit-export-")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), ".csv")

	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	require.GreaterOrEqual(t, len(lines), 2, "expected header + at least 1 data row")
	assert.Equal(t, "id,timestamp,type,actor_id,actor_type,resource,resource_id,action,payload,metadata", lines[0])
}

func TestAuditHandler_Export_NDJSON(t *testing.T) {
	q := &fakeAuditQuerier{
		listResult: []sqlcgen.AuditEvent{validAuditEvent()},
		onceOnly:   true,
	}
	h := v1.NewAuditHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?format=json", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.Export(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/x-ndjson", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), ".json")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(rec.Body.String())), &parsed))
	assert.Equal(t, "01JTEST000000000000000000", parsed["id"])
}

func TestAuditHandler_Export_InvalidFormat(t *testing.T) {
	q := &fakeAuditQuerier{}
	h := v1.NewAuditHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?format=xml", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.Export(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_Export_EmptyResult(t *testing.T) {
	q := &fakeAuditQuerier{
		listResult: []sqlcgen.AuditEvent{},
	}
	h := v1.NewAuditHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?format=csv", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.Export(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	assert.Len(t, lines, 1, "expected only CSV header row")
	assert.Equal(t, "id,timestamp,type,actor_id,actor_type,resource,resource_id,action,payload,metadata", lines[0])
}
