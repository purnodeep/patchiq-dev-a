package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFeedQuerier implements v1.FeedQuerier for testing.
type mockFeedQuerier struct {
	listFn             func(ctx context.Context) ([]sqlcgen.ListFeedSourcesWithSyncStateRow, error)
	getFn              func(ctx context.Context, id pgtype.UUID) (sqlcgen.FeedSource, error)
	updateFn           func(ctx context.Context, arg sqlcgen.UpdateFeedSourceParams) (sqlcgen.FeedSource, error)
	getWithSyncStateFn func(ctx context.Context, id pgtype.UUID) (sqlcgen.GetFeedSourceWithSyncStateByIDRow, error)
	listHistoryFn      func(ctx context.Context, arg sqlcgen.ListFeedSyncHistoryParams) ([]sqlcgen.FeedSyncHistory, error)
	countHistoryFn     func(ctx context.Context, feedSourceID pgtype.UUID) (int64, error)
	listRecentStatusFn func(ctx context.Context, feedSourceID pgtype.UUID) ([]sqlcgen.ListRecentFeedSyncStatusRow, error)
	getNewThisWeekFn   func(ctx context.Context, feedSourceID pgtype.UUID) (int64, error)
	getErrorRateFn     func(ctx context.Context, feedSourceID pgtype.UUID) (sqlcgen.GetFeedErrorRateRow, error)
}

func (m *mockFeedQuerier) ListFeedSourcesWithSyncState(ctx context.Context) ([]sqlcgen.ListFeedSourcesWithSyncStateRow, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockFeedQuerier) GetFeedSourceByID(ctx context.Context, id pgtype.UUID) (sqlcgen.FeedSource, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return sqlcgen.FeedSource{}, pgx.ErrNoRows
}

func (m *mockFeedQuerier) UpdateFeedSource(ctx context.Context, arg sqlcgen.UpdateFeedSourceParams) (sqlcgen.FeedSource, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, arg)
	}
	return sqlcgen.FeedSource{}, pgx.ErrNoRows
}

func (m *mockFeedQuerier) GetFeedSourceWithSyncStateByID(ctx context.Context, id pgtype.UUID) (sqlcgen.GetFeedSourceWithSyncStateByIDRow, error) {
	if m.getWithSyncStateFn != nil {
		return m.getWithSyncStateFn(ctx, id)
	}
	return sqlcgen.GetFeedSourceWithSyncStateByIDRow{}, pgx.ErrNoRows
}

func (m *mockFeedQuerier) ListFeedSyncHistory(ctx context.Context, arg sqlcgen.ListFeedSyncHistoryParams) ([]sqlcgen.FeedSyncHistory, error) {
	if m.listHistoryFn != nil {
		return m.listHistoryFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockFeedQuerier) CountFeedSyncHistory(ctx context.Context, feedSourceID pgtype.UUID) (int64, error) {
	if m.countHistoryFn != nil {
		return m.countHistoryFn(ctx, feedSourceID)
	}
	return 0, nil
}

func (m *mockFeedQuerier) ListRecentFeedSyncStatus(ctx context.Context, feedSourceID pgtype.UUID) ([]sqlcgen.ListRecentFeedSyncStatusRow, error) {
	if m.listRecentStatusFn != nil {
		return m.listRecentStatusFn(ctx, feedSourceID)
	}
	return nil, nil
}

func (m *mockFeedQuerier) GetFeedNewThisWeek(ctx context.Context, feedSourceID pgtype.UUID) (int64, error) {
	if m.getNewThisWeekFn != nil {
		return m.getNewThisWeekFn(ctx, feedSourceID)
	}
	return 0, nil
}

func (m *mockFeedQuerier) GetFeedErrorRate(ctx context.Context, feedSourceID pgtype.UUID) (sqlcgen.GetFeedErrorRateRow, error) {
	if m.getErrorRateFn != nil {
		return m.getErrorRateFn(ctx, feedSourceID)
	}
	return sqlcgen.GetFeedErrorRateRow{}, nil
}

// mockRiverEnqueuer implements v1.RiverEnqueuer for testing.
type mockRiverEnqueuer struct {
	insertFn func(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
	inserted []river.JobArgs
}

func (m *mockRiverEnqueuer) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	m.inserted = append(m.inserted, args)
	if m.insertFn != nil {
		return m.insertFn(ctx, args, opts)
	}
	return &rivertype.JobInsertResult{}, nil
}

func TestFeedList(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)

	tests := []struct {
		name       string
		querier    *mockFeedQuerier
		wantStatus int
		wantCount  int
	}{
		{
			name: "returns feed sources with sync state",
			querier: &mockFeedQuerier{
				listFn: func(_ context.Context) ([]sqlcgen.ListFeedSourcesWithSyncStateRow, error) {
					return []sqlcgen.ListFeedSourcesWithSyncStateRow{
						{
							ID:                  testUUID(1),
							Name:                "nvd",
							DisplayName:         "National Vulnerability Database",
							Enabled:             true,
							SyncIntervalSeconds: 21600,
							LastSyncAt:          pgtype.Timestamptz{Time: now, Valid: true},
							NextSyncAt:          pgtype.Timestamptz{Time: now.Add(6 * time.Hour), Valid: true},
							Status:              pgtype.Text{String: "idle", Valid: true},
							ErrorCount:          pgtype.Int4{Int32: 0, Valid: true},
							LastError:           pgtype.Text{Valid: false},
							EntriesIngested:     pgtype.Int8{Int64: 4523, Valid: true},
							Cursor:              pgtype.Text{String: "2026-03-07T12:00:00Z", Valid: true},
						},
						{
							ID:                  testUUID(2),
							Name:                "github_advisories",
							DisplayName:         "GitHub Security Advisories",
							Enabled:             true,
							SyncIntervalSeconds: 3600,
							LastSyncAt:          pgtype.Timestamptz{Valid: false},
							NextSyncAt:          pgtype.Timestamptz{Time: now.Add(time.Hour), Valid: true},
							Status:              pgtype.Text{String: "error", Valid: true},
							ErrorCount:          pgtype.Int4{Int32: 3, Valid: true},
							LastError:           pgtype.Text{String: "rate limited", Valid: true},
							EntriesIngested:     pgtype.Int8{Int64: 0, Valid: true},
							Cursor:              pgtype.Text{Valid: false},
						},
					}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "returns empty array when no sources",
			querier: &mockFeedQuerier{
				listFn: func(_ context.Context) ([]sqlcgen.ListFeedSourcesWithSyncStateRow, error) {
					return []sqlcgen.ListFeedSourcesWithSyncStateRow{}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "returns 500 on query error",
			querier: &mockFeedQuerier{
				listFn: func(_ context.Context) ([]sqlcgen.ListFeedSourcesWithSyncStateRow, error) {
					return nil, errors.New("db connection lost")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantCount:  -1, // not checked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewFeedHandler(tt.querier, nil, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var body []json.RawMessage
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body, tt.wantCount)

				// Verify JSON shape for non-empty response.
				if tt.wantCount > 0 {
					var first map[string]any
					require.NoError(t, json.Unmarshal(body[0], &first))
					assert.Contains(t, first, "id")
					assert.Contains(t, first, "name")
					assert.Contains(t, first, "display_name")
					assert.Contains(t, first, "enabled")
					assert.Contains(t, first, "sync_interval_seconds")
					assert.Contains(t, first, "status")
					assert.Contains(t, first, "error_count")
					assert.Contains(t, first, "entries_ingested")
					assert.Contains(t, first, "last_sync_at")
					assert.Contains(t, first, "next_sync_at")

					// First source has valid last_sync_at, should not be null.
					assert.NotNil(t, first["last_sync_at"])

					// Second source has null last_sync_at and non-null last_error.
					var second map[string]any
					require.NoError(t, json.Unmarshal(body[1], &second))
					assert.Nil(t, second["last_sync_at"])
					assert.NotNil(t, second["last_error"])
					assert.Equal(t, "rate limited", second["last_error"])
				}
			}

			if tt.wantStatus == http.StatusInternalServerError {
				var errBody map[string]string
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errBody))
				assert.Contains(t, errBody["error"], "list feed sources")
			}
		})
	}
}

func TestUpdateFeed_Enable(t *testing.T) {
	enabled := true
	interval := int32(3600)
	querier := &mockFeedQuerier{
		updateFn: func(_ context.Context, arg sqlcgen.UpdateFeedSourceParams) (sqlcgen.FeedSource, error) {
			return sqlcgen.FeedSource{
				ID:                  arg.ID,
				Name:                "nvd",
				DisplayName:         "National Vulnerability Database",
				Enabled:             true,
				SyncIntervalSeconds: 3600,
			}, nil
		},
	}

	h := v1.NewFeedHandler(querier, nil, nil)

	body := `{"enabled": true, "sync_interval_seconds": 3600}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/feeds/"+testUUIDString(1), strings.NewReader(body))

	// Wire chi URL params and tenant context.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testUUIDString(1))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = tenant.WithTenantID(ctx, "00000000-0000-0000-0000-000000000001")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["enabled"])
	assert.Equal(t, float64(interval), resp["sync_interval_seconds"])
	assert.Equal(t, "nvd", resp["name"])
	_ = enabled // suppress unused warning
}

func TestUpdateFeed_NotFound(t *testing.T) {
	querier := &mockFeedQuerier{
		updateFn: func(_ context.Context, _ sqlcgen.UpdateFeedSourceParams) (sqlcgen.FeedSource, error) {
			return sqlcgen.FeedSource{}, pgx.ErrNoRows
		},
	}

	h := v1.NewFeedHandler(querier, nil, nil)

	body := `{"enabled": false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/feeds/"+testUUIDString(1), strings.NewReader(body))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testUUIDString(1))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = tenant.WithTenantID(ctx, "00000000-0000-0000-0000-000000000001")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "feed source not found")
}

func TestTriggerFeedSync_Success(t *testing.T) {
	querier := &mockFeedQuerier{
		getFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.FeedSource, error) {
			return sqlcgen.FeedSource{
				ID:   testUUID(1),
				Name: "nvd",
			}, nil
		},
	}
	enqueuer := &mockRiverEnqueuer{}

	h := v1.NewFeedHandler(querier, enqueuer, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feeds/"+testUUIDString(1)+"/sync", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testUUIDString(1))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.TriggerSync(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "accepted", resp["status"])
	assert.Equal(t, "nvd", resp["feed_name"])

	// Verify a job was enqueued.
	require.Len(t, enqueuer.inserted, 1)
	assert.Equal(t, "hub_feed_sync", enqueuer.inserted[0].Kind())
}

func TestTriggerFeedSync_FeedNotFound(t *testing.T) {
	querier := &mockFeedQuerier{
		getFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.FeedSource, error) {
			return sqlcgen.FeedSource{}, pgx.ErrNoRows
		},
	}
	enqueuer := &mockRiverEnqueuer{}

	h := v1.NewFeedHandler(querier, enqueuer, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feeds/"+testUUIDString(1)+"/sync", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testUUIDString(1))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.TriggerSync(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "feed source not found")

	// No job should have been enqueued.
	assert.Empty(t, enqueuer.inserted)
}
