package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDashboardQuerier implements v1.DashboardQuerier for testing.
type mockDashboardQuerier struct {
	statsFn            func(ctx context.Context) (sqlcgen.GetDashboardStatsRow, error)
	licenseBreakdownFn func(ctx context.Context) ([]sqlcgen.GetDashboardLicenseBreakdownRow, error)
	catalogGrowthFn    func(ctx context.Context, days int32) ([]sqlcgen.GetDashboardCatalogGrowthRow, error)
	clientSummaryFn    func(ctx context.Context) ([]sqlcgen.GetDashboardClientSummaryRow, error)
}

func (m *mockDashboardQuerier) GetDashboardStats(ctx context.Context) (sqlcgen.GetDashboardStatsRow, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx)
	}
	return sqlcgen.GetDashboardStatsRow{}, nil
}

func (m *mockDashboardQuerier) GetDashboardLicenseBreakdown(ctx context.Context) ([]sqlcgen.GetDashboardLicenseBreakdownRow, error) {
	if m.licenseBreakdownFn != nil {
		return m.licenseBreakdownFn(ctx)
	}
	return nil, nil
}

func (m *mockDashboardQuerier) GetDashboardCatalogGrowth(ctx context.Context, days int32) ([]sqlcgen.GetDashboardCatalogGrowthRow, error) {
	if m.catalogGrowthFn != nil {
		return m.catalogGrowthFn(ctx, days)
	}
	return nil, nil
}

func (m *mockDashboardQuerier) GetDashboardClientSummary(ctx context.Context) ([]sqlcgen.GetDashboardClientSummaryRow, error) {
	if m.clientSummaryFn != nil {
		return m.clientSummaryFn(ctx)
	}
	return nil, nil
}

func (m *mockDashboardQuerier) ListAuditEventsByTenant(_ context.Context, _ sqlcgen.ListAuditEventsByTenantParams) ([]sqlcgen.AuditEvent, error) {
	return nil, nil
}

func TestGetDashboardStats_Success(t *testing.T) {
	querier := &mockDashboardQuerier{
		statsFn: func(_ context.Context) (sqlcgen.GetDashboardStatsRow, error) {
			return sqlcgen.GetDashboardStatsRow{
				TotalCatalogEntries: 42,
				ActiveFeeds:         3,
				ConnectedClients:    10,
				PendingClients:      2,
				ActiveLicenses:      8,
			}, nil
		},
	}
	h := v1.NewDashboardHandler(querier)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	rec := httptest.NewRecorder()

	h.Stats(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(42), body["total_catalog_entries"])
	assert.Equal(t, float64(3), body["active_feeds"])
	assert.Equal(t, float64(10), body["connected_clients"])
	assert.Equal(t, float64(2), body["pending_clients"])
	assert.Equal(t, float64(8), body["active_licenses"])
}

func TestGetDashboardStats_DBError(t *testing.T) {
	querier := &mockDashboardQuerier{
		statsFn: func(_ context.Context) (sqlcgen.GetDashboardStatsRow, error) {
			return sqlcgen.GetDashboardStatsRow{}, errors.New("db down")
		},
	}
	h := v1.NewDashboardHandler(querier)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	rec := httptest.NewRecorder()

	h.Stats(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "get dashboard stats")
}

func TestGetLicenseBreakdown_Success(t *testing.T) {
	querier := &mockDashboardQuerier{
		licenseBreakdownFn: func(_ context.Context) ([]sqlcgen.GetDashboardLicenseBreakdownRow, error) {
			return []sqlcgen.GetDashboardLicenseBreakdownRow{
				{Tier: "enterprise", Status: "active", Count: 3, TotalEndpoints: 500},
				{Tier: "standard", Status: "expired", Count: 1, TotalEndpoints: 50},
			}, nil
		},
	}
	h := v1.NewDashboardHandler(querier)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/license-breakdown", nil)
	rec := httptest.NewRecorder()

	h.LicenseBreakdown(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 2)
	assert.Equal(t, "enterprise", body[0]["tier"])
	assert.Equal(t, "active", body[0]["status"])
	assert.Equal(t, float64(3), body[0]["count"])
	assert.Equal(t, float64(500), body[0]["total_endpoints"])
}

func TestGetLicenseBreakdown_DBError(t *testing.T) {
	querier := &mockDashboardQuerier{
		licenseBreakdownFn: func(_ context.Context) ([]sqlcgen.GetDashboardLicenseBreakdownRow, error) {
			return nil, errors.New("db down")
		},
	}
	h := v1.NewDashboardHandler(querier)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/license-breakdown", nil)
	rec := httptest.NewRecorder()

	h.LicenseBreakdown(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "get license breakdown")
}

func TestGetCatalogGrowth_Success(t *testing.T) {
	querier := &mockDashboardQuerier{
		catalogGrowthFn: func(_ context.Context, days int32) ([]sqlcgen.GetDashboardCatalogGrowthRow, error) {
			assert.Equal(t, int32(90), days)
			return []sqlcgen.GetDashboardCatalogGrowthRow{
				{Day: pgtype.Date{Time: time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC), Valid: true}, EntriesAdded: 10},
				{Day: pgtype.Date{Time: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), Valid: true}, EntriesAdded: 5},
			}, nil
		},
	}
	h := v1.NewDashboardHandler(querier)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/catalog-growth?days=90", nil)
	rec := httptest.NewRecorder()

	h.CatalogGrowth(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 2)
	assert.Equal(t, "2026-03-14", body[0]["day"])
	assert.Equal(t, float64(10), body[0]["entries_added"])
}

func TestGetCatalogGrowth_DBError(t *testing.T) {
	querier := &mockDashboardQuerier{
		catalogGrowthFn: func(_ context.Context, _ int32) ([]sqlcgen.GetDashboardCatalogGrowthRow, error) {
			return nil, errors.New("db down")
		},
	}
	h := v1.NewDashboardHandler(querier)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/catalog-growth", nil)
	rec := httptest.NewRecorder()

	h.CatalogGrowth(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "get catalog growth")
}

func TestGetCatalogGrowth_ClampsBoundary(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"zero days", "/api/v1/dashboard/catalog-growth?days=0"},
		{"negative days", "/api/v1/dashboard/catalog-growth?days=-1"},
		{"over max", "/api/v1/dashboard/catalog-growth?days=500"},
		{"missing param", "/api/v1/dashboard/catalog-growth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := &mockDashboardQuerier{
				catalogGrowthFn: func(_ context.Context, days int32) ([]sqlcgen.GetDashboardCatalogGrowthRow, error) {
					assert.Equal(t, int32(90), days, "days should be clamped to 90")
					return []sqlcgen.GetDashboardCatalogGrowthRow{}, nil
				},
			}
			h := v1.NewDashboardHandler(querier)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()

			h.CatalogGrowth(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestGetClientSummary_Success(t *testing.T) {
	querier := &mockDashboardQuerier{
		clientSummaryFn: func(_ context.Context) ([]sqlcgen.GetDashboardClientSummaryRow, error) {
			return []sqlcgen.GetDashboardClientSummaryRow{
				{
					ID:            pgtype.UUID{Bytes: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, Valid: true},
					Hostname:      "prod-east",
					Status:        "approved",
					EndpointCount: 150,
					LastSyncAt:    pgtype.Timestamptz{Time: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Valid: true},
					Version:       pgtype.Text{String: "1.2.0", Valid: true},
					Os:            pgtype.Text{String: "linux", Valid: true},
				},
			}, nil
		},
	}
	h := v1.NewDashboardHandler(querier)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/clients", nil)
	rec := httptest.NewRecorder()

	h.ClientSummary(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 1)
	assert.Equal(t, "prod-east", body[0]["hostname"])
	assert.Equal(t, "approved", body[0]["status"])
	assert.Equal(t, float64(150), body[0]["endpoint_count"])
	assert.Equal(t, "1.2.0", body[0]["version"])
}

func TestGetClientSummary_DBError(t *testing.T) {
	querier := &mockDashboardQuerier{
		clientSummaryFn: func(_ context.Context) ([]sqlcgen.GetDashboardClientSummaryRow, error) {
			return nil, errors.New("db down")
		},
	}
	h := v1.NewDashboardHandler(querier)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/clients", nil)
	rec := httptest.NewRecorder()

	h.ClientSummary(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "get client summary")
}
