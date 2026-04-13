package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseRenew_Success(t *testing.T) {
	bus := &mockEventBus{}
	futureDate := time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339)
	tier := "enterprise"
	maxEP := int32(500)

	querier := &mockLicenseQuerier{
		renewFn: func(_ context.Context, arg sqlcgen.RenewLicenseParams) (sqlcgen.License, error) {
			assert.True(t, arg.NewTier.Valid)
			assert.Equal(t, "enterprise", arg.NewTier.String)
			assert.True(t, arg.NewMaxEndpoints.Valid)
			assert.Equal(t, int32(500), arg.NewMaxEndpoints.Int32)
			return sqlcgen.License{
				ID:           arg.ID,
				Tier:         "enterprise",
				MaxEndpoints: 500,
				ExpiresAt:    arg.ExpiresAt,
				CustomerName: "Test Co",
			}, nil
		},
	}
	h := v1.NewLicenseHandler(querier, bus)

	body, _ := json.Marshal(map[string]any{
		"tier":          tier,
		"max_endpoints": maxEP,
		"expires_at":    futureDate,
	})
	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/licenses/"+id+"/renew", bytes.NewReader(body))
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Renew(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp, "license")

	assert.Len(t, bus.emitted, 1)
}

func TestLicenseRenew_OnlyExpiresAt(t *testing.T) {
	bus := &mockEventBus{}
	futureDate := time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339)

	querier := &mockLicenseQuerier{
		renewFn: func(_ context.Context, arg sqlcgen.RenewLicenseParams) (sqlcgen.License, error) {
			assert.False(t, arg.NewTier.Valid, "tier should not be set when omitted")
			assert.False(t, arg.NewMaxEndpoints.Valid, "max_endpoints should not be set when omitted")
			return sqlcgen.License{ID: arg.ID, Tier: "professional", CustomerName: "Test Co", ExpiresAt: arg.ExpiresAt}, nil
		},
	}
	h := v1.NewLicenseHandler(querier, bus)

	body, _ := json.Marshal(map[string]any{
		"expires_at": futureDate,
	})
	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/licenses/"+id+"/renew", bytes.NewReader(body))
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Renew(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLicenseRenew_Validation(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{
			name:    "missing expires_at",
			body:    map[string]any{},
			wantErr: "expires_at is required",
		},
		{
			name:    "past expires_at",
			body:    map[string]any{"expires_at": "2020-01-01T00:00:00Z"},
			wantErr: "expires_at must be in the future",
		},
		{
			name:    "invalid tier",
			body:    map[string]any{"expires_at": time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339), "tier": "invalid"},
			wantErr: "tier must be one of",
		},
		{
			name:    "zero max_endpoints",
			body:    map[string]any{"expires_at": time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339), "max_endpoints": 0},
			wantErr: "max_endpoints must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewLicenseHandler(&mockLicenseQuerier{}, &mockEventBus{})
			body, _ := json.Marshal(tt.body)
			id := testUUIDString(1)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/licenses/"+id+"/renew", bytes.NewReader(body))
			req = withChiURLParam(req, "id", id)
			req = withTenantCtx(req)
			rec := httptest.NewRecorder()

			h.Renew(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantErr)
		})
	}
}

func TestLicenseRenew_NotFound(t *testing.T) {
	querier := &mockLicenseQuerier{
		renewFn: func(_ context.Context, _ sqlcgen.RenewLicenseParams) (sqlcgen.License, error) {
			return sqlcgen.License{}, pgx.ErrNoRows
		},
	}
	h := v1.NewLicenseHandler(querier, &mockEventBus{})

	body, _ := json.Marshal(map[string]any{
		"expires_at": time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339),
	})
	id := testUUIDString(99)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/licenses/"+id+"/renew", bytes.NewReader(body))
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Renew(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "license not found")
}

func TestLicenseRenew_InvalidUUID(t *testing.T) {
	h := v1.NewLicenseHandler(&mockLicenseQuerier{}, &mockEventBus{})
	body, _ := json.Marshal(map[string]any{
		"expires_at": time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/licenses/bad-uuid/renew", bytes.NewReader(body))
	req = withChiURLParam(req, "id", "bad-uuid")
	rec := httptest.NewRecorder()

	h.Renew(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLicenseUsageHistory_Success(t *testing.T) {
	querier := &mockLicenseQuerier{
		getByIDFn: func(_ context.Context, id pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error) {
			return sqlcgen.GetLicenseByIDRow{ID: id, MaxEndpoints: 100, Tier: "professional"}, nil
		},
		usageHistoryFn: func(_ context.Context, arg sqlcgen.GetLicenseUsageHistoryParams) ([]sqlcgen.GetLicenseUsageHistoryRow, error) {
			assert.Equal(t, int32(90), arg.Days)
			return []sqlcgen.GetLicenseUsageHistoryRow{
				{Date: pgtype.Date{Time: time.Now().AddDate(0, 0, -1), Valid: true}, EndpointCount: 42},
			}, nil
		},
	}
	h := v1.NewLicenseHandler(querier, &mockEventBus{})

	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/"+id+"/usage-history", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.UsageHistory(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(100), resp["max_endpoints"])
	points, ok := resp["points"].([]any)
	require.True(t, ok)
	assert.Len(t, points, 1)
}

func TestLicenseUsageHistory_CustomDays(t *testing.T) {
	querier := &mockLicenseQuerier{
		getByIDFn: func(_ context.Context, id pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error) {
			return sqlcgen.GetLicenseByIDRow{ID: id, MaxEndpoints: 50}, nil
		},
		usageHistoryFn: func(_ context.Context, arg sqlcgen.GetLicenseUsageHistoryParams) ([]sqlcgen.GetLicenseUsageHistoryRow, error) {
			assert.Equal(t, int32(30), arg.Days)
			return nil, nil
		},
	}
	h := v1.NewLicenseHandler(querier, &mockEventBus{})

	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/"+id+"/usage-history?days=30", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.UsageHistory(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	// Nil result should be returned as empty array.
	points, ok := resp["points"].([]any)
	require.True(t, ok)
	assert.Empty(t, points)
}

func TestLicenseUsageHistory_NotFound(t *testing.T) {
	querier := &mockLicenseQuerier{
		getByIDFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error) {
			return sqlcgen.GetLicenseByIDRow{}, pgx.ErrNoRows
		},
	}
	h := v1.NewLicenseHandler(querier, &mockEventBus{})

	id := testUUIDString(99)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/"+id+"/usage-history", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.UsageHistory(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLicenseUsageHistory_InvalidUUID(t *testing.T) {
	h := v1.NewLicenseHandler(&mockLicenseQuerier{}, &mockEventBus{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/bad-uuid/usage-history", nil)
	req = withChiURLParam(req, "id", "bad-uuid")
	rec := httptest.NewRecorder()

	h.UsageHistory(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLicenseAuditTrail_Success(t *testing.T) {
	idStr := testUUIDString(1)
	// The handler converts the UUID to a string for the resource_id query.
	// testUUIDString produces the same format as uuidToString.
	expectedResourceID := testUUIDString(1)

	querier := &mockLicenseQuerier{
		listAuditByResourceIDFn: func(_ context.Context, arg sqlcgen.ListAuditEventsByResourceIDParams) ([]sqlcgen.AuditEvent, error) {
			assert.Equal(t, "license", arg.Resource)
			assert.Equal(t, expectedResourceID, arg.ResourceID)
			assert.Equal(t, int32(50), arg.QueryLimit)
			return []sqlcgen.AuditEvent{
				{ID: "evt-1", Resource: "license", ResourceID: arg.ResourceID, Action: "create"},
				{ID: "evt-2", Resource: "license", ResourceID: arg.ResourceID, Action: "renew"},
			}, nil
		},
		countAuditByResourceIDFn: func(_ context.Context, arg sqlcgen.CountAuditEventsByResourceIDParams) (int64, error) {
			assert.Equal(t, "license", arg.Resource)
			return 2, nil
		},
	}
	h := v1.NewLicenseHandler(querier, &mockEventBus{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/"+idStr+"/audit-trail", nil)
	req = withChiURLParam(req, "id", idStr)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.AuditTrail(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["total"])
	items, ok := resp["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestLicenseAuditTrail_CustomLimit(t *testing.T) {
	querier := &mockLicenseQuerier{
		listAuditByResourceIDFn: func(_ context.Context, arg sqlcgen.ListAuditEventsByResourceIDParams) ([]sqlcgen.AuditEvent, error) {
			assert.Equal(t, int32(10), arg.QueryLimit)
			assert.Equal(t, int32(5), arg.QueryOffset)
			return nil, nil
		},
		countAuditByResourceIDFn: func(_ context.Context, _ sqlcgen.CountAuditEventsByResourceIDParams) (int64, error) {
			return 0, nil
		},
	}
	h := v1.NewLicenseHandler(querier, &mockEventBus{})

	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/"+id+"/audit-trail?limit=10&offset=5", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.AuditTrail(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total"])
	items, ok := resp["items"].([]any)
	require.True(t, ok)
	assert.Empty(t, items)
}

func TestLicenseAuditTrail_InvalidUUID(t *testing.T) {
	h := v1.NewLicenseHandler(&mockLicenseQuerier{}, &mockEventBus{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/bad-uuid/audit-trail", nil)
	req = withChiURLParam(req, "id", "bad-uuid")
	rec := httptest.NewRecorder()

	h.AuditTrail(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
