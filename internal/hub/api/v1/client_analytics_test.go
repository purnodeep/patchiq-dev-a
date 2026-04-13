package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientHandler_SyncHistory(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		query      string
		querier    *mockClientQuerier
		wantStatus int
		wantTotal  float64
		wantItems  int
	}{
		{
			name:  "returns sync history with defaults",
			id:    testUUIDString(1),
			query: "",
			querier: &mockClientQuerier{
				listSyncHistoryFn: func(_ context.Context, arg sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error) {
					assert.Equal(t, int32(50), arg.QueryLimit)
					assert.Equal(t, int32(0), arg.QueryOffset)
					return []sqlcgen.ClientSyncHistory{
						{ID: testUUID(10), ClientID: testUUID(1), Status: "success"},
						{ID: testUUID(11), ClientID: testUUID(1), Status: "success"},
					}, nil
				},
				countSyncHistoryFn: func(_ context.Context, _ sqlcgen.CountClientSyncHistoryParams) (int64, error) {
					return 2, nil
				},
			},
			wantStatus: http.StatusOK,
			wantTotal:  2,
			wantItems:  2,
		},
		{
			name:  "respects limit and offset",
			id:    testUUIDString(1),
			query: "?limit=10&offset=5",
			querier: &mockClientQuerier{
				listSyncHistoryFn: func(_ context.Context, arg sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error) {
					assert.Equal(t, int32(10), arg.QueryLimit)
					assert.Equal(t, int32(5), arg.QueryOffset)
					return nil, nil
				},
				countSyncHistoryFn: func(_ context.Context, _ sqlcgen.CountClientSyncHistoryParams) (int64, error) {
					return 0, nil
				},
			},
			wantStatus: http.StatusOK,
			wantTotal:  0,
			wantItems:  0,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "bad-uuid",
			querier:    &mockClientQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "list error returns 500",
			id:   testUUIDString(1),
			querier: &mockClientQuerier{
				listSyncHistoryFn: func(_ context.Context, _ sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error) {
					return nil, errors.New("db down")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "count error returns 500",
			id:   testUUIDString(1),
			querier: &mockClientQuerier{
				listSyncHistoryFn: func(_ context.Context, _ sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error) {
					return []sqlcgen.ClientSyncHistory{}, nil
				},
				countSyncHistoryFn: func(_ context.Context, _ sqlcgen.CountClientSyncHistoryParams) (int64, error) {
					return 0, errors.New("db down")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewClientHandler(tt.querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+tt.id+"/sync-history"+tt.query, nil)
			req = withChiURLParam(req, "id", tt.id)
			req = withTenantCtx(req)
			rec := httptest.NewRecorder()

			h.SyncHistory(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tt.wantTotal, body["total"])
				items, ok := body["items"].([]any)
				require.True(t, ok)
				assert.Len(t, items, tt.wantItems)
			}
		})
	}
}

func TestClientHandler_EndpointTrend(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		query      string
		querier    *mockClientQuerier
		wantStatus int
		wantPoints int
	}{
		{
			name:  "returns trend with default days",
			id:    testUUIDString(1),
			query: "",
			querier: &mockClientQuerier{
				getEndpointTrendFn: func(_ context.Context, arg sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error) {
					assert.Equal(t, int32(90), arg.Days)
					return []sqlcgen.GetClientEndpointTrendRow{
						{Date: pgtype.Date{Valid: true}, EndpointCount: 10},
						{Date: pgtype.Date{Valid: true}, EndpointCount: 15},
					}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantPoints: 2,
		},
		{
			name:  "respects days parameter",
			id:    testUUIDString(1),
			query: "?days=30",
			querier: &mockClientQuerier{
				getEndpointTrendFn: func(_ context.Context, arg sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error) {
					assert.Equal(t, int32(30), arg.Days)
					return nil, nil
				},
			},
			wantStatus: http.StatusOK,
			wantPoints: 0,
		},
		{
			name:  "caps days at 365",
			id:    testUUIDString(1),
			query: "?days=999",
			querier: &mockClientQuerier{
				getEndpointTrendFn: func(_ context.Context, arg sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error) {
					assert.Equal(t, int32(365), arg.Days)
					return nil, nil
				},
			},
			wantStatus: http.StatusOK,
			wantPoints: 0,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "bad-uuid",
			querier:    &mockClientQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "db error returns 500",
			id:   testUUIDString(1),
			querier: &mockClientQuerier{
				getEndpointTrendFn: func(_ context.Context, _ sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error) {
					return nil, errors.New("db down")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewClientHandler(tt.querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+tt.id+"/endpoint-trend"+tt.query, nil)
			req = withChiURLParam(req, "id", tt.id)
			req = withTenantCtx(req)
			rec := httptest.NewRecorder()

			h.EndpointTrend(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				points, ok := body["points"].([]any)
				require.True(t, ok)
				assert.Len(t, points, tt.wantPoints)
			}
		})
	}
}
