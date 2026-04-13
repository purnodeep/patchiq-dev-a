package v1_test

import (
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

// mockSyncQuerier implements v1.SyncQuerier for testing.
type mockSyncQuerier struct {
	listUpdatedFn    func(ctx context.Context, updatedAt pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error)
	listDeletedFn    func(ctx context.Context, deletedAt pgtype.Timestamptz) ([]pgtype.UUID, error)
	listCVEUpdatedFn func(ctx context.Context, updatedAt pgtype.Timestamptz) ([]sqlcgen.CVEFeed, error)
	getClientFn      func(ctx context.Context, apiKeyHash pgtype.Text) (sqlcgen.Client, error)
	updatedSummaries *sqlcgen.UpdateClientSummariesParams
	insertedHistory  *sqlcgen.InsertClientSyncHistoryParams
}

func (m *mockSyncQuerier) ListCatalogEntriesUpdatedSince(ctx context.Context, updatedAt pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error) {
	if m.listUpdatedFn != nil {
		return m.listUpdatedFn(ctx, updatedAt)
	}
	return nil, nil
}

func (m *mockSyncQuerier) ListCatalogEntriesDeletedSince(ctx context.Context, deletedAt pgtype.Timestamptz) ([]pgtype.UUID, error) {
	if m.listDeletedFn != nil {
		return m.listDeletedFn(ctx, deletedAt)
	}
	return nil, nil
}

func (m *mockSyncQuerier) GetClientByAPIKeyHash(ctx context.Context, apiKeyHash pgtype.Text) (sqlcgen.Client, error) {
	if m.getClientFn != nil {
		return m.getClientFn(ctx, apiKeyHash)
	}
	// Default: return pgx.ErrNoRows to simulate no client found.
	return sqlcgen.Client{}, pgx.ErrNoRows
}

func (m *mockSyncQuerier) UpdateClientSummaries(_ context.Context, arg sqlcgen.UpdateClientSummariesParams) (sqlcgen.Client, error) {
	m.updatedSummaries = &arg
	return sqlcgen.Client{}, nil
}

func (m *mockSyncQuerier) InsertClientSyncHistory(_ context.Context, arg sqlcgen.InsertClientSyncHistoryParams) (sqlcgen.ClientSyncHistory, error) {
	m.insertedHistory = &arg
	return sqlcgen.ClientSyncHistory{}, nil
}

func (m *mockSyncQuerier) ListCVEFeedsUpdatedSince(ctx context.Context, arg sqlcgen.ListCVEFeedsUpdatedSinceParams) ([]sqlcgen.CVEFeed, error) {
	if m.listCVEUpdatedFn != nil {
		return m.listCVEUpdatedFn(ctx, arg.UpdatedAt)
	}
	return nil, nil
}

func (m *mockSyncQuerier) ListCatalogCVELinks(_ context.Context, _ []pgtype.UUID) ([]sqlcgen.ListCatalogCVELinksRow, error) {
	return nil, nil
}

func TestSync(t *testing.T) {
	const validAPIKey = "test-sync-api-key-secret"

	testUUID := pgtype.UUID{Valid: true}
	testUUID.Bytes = [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now().UTC()
	testEntry := sqlcgen.PatchCatalog{
		ID:        testUUID,
		Name:      "KB5001234",
		Vendor:    "Microsoft",
		OsFamily:  "windows",
		Version:   "1.0.0",
		Severity:  "critical",
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}

	tests := []struct {
		name           string
		apiKey         string
		authHeader     string
		queryParams    string
		mock           *mockSyncQuerier
		wantStatus     int
		wantEntries    bool
		wantDeletedIDs bool
	}{
		{
			name:        "valid request with entries and deleted IDs",
			apiKey:      validAPIKey,
			authHeader:  "Bearer " + validAPIKey,
			queryParams: "since=2026-01-01T00:00:00Z",
			mock: &mockSyncQuerier{
				listUpdatedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error) {
					return []sqlcgen.PatchCatalog{testEntry}, nil
				},
				listDeletedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]pgtype.UUID, error) {
					return []pgtype.UUID{testUUID}, nil
				},
			},
			wantStatus:     http.StatusOK,
			wantEntries:    true,
			wantDeletedIDs: true,
		},
		{
			name:        "missing since parameter",
			apiKey:      validAPIKey,
			authHeader:  "Bearer " + validAPIKey,
			queryParams: "",
			mock:        &mockSyncQuerier{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid since format",
			apiKey:      validAPIKey,
			authHeader:  "Bearer " + validAPIKey,
			queryParams: "since=not-a-date",
			mock:        &mockSyncQuerier{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "missing authorization header",
			apiKey:      validAPIKey,
			authHeader:  "",
			queryParams: "since=2026-01-01T00:00:00Z",
			mock:        &mockSyncQuerier{},
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "wrong API key",
			apiKey:      validAPIKey,
			authHeader:  "Bearer wrong-key",
			queryParams: "since=2026-01-01T00:00:00Z",
			mock:        &mockSyncQuerier{},
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "empty API key rejects all requests",
			apiKey:      "",
			authHeader:  "Bearer ",
			queryParams: "since=2026-01-01T00:00:00Z",
			mock:        &mockSyncQuerier{},
			wantStatus:  http.StatusServiceUnavailable,
		},
		{
			name:        "no changes since timestamp",
			apiKey:      validAPIKey,
			authHeader:  "Bearer " + validAPIKey,
			queryParams: "since=2026-03-04T00:00:00Z",
			mock: &mockSyncQuerier{
				listUpdatedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error) {
					return nil, nil
				},
				listDeletedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]pgtype.UUID, error) {
					return nil, nil
				},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := v1.NewSyncHandler(tt.mock, tt.apiKey, bus)

			url := "/api/v1/sync"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.Sync(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var resp struct {
					Entries    []sqlcgen.PatchCatalog `json:"entries"`
					DeletedIDs []string               `json:"deleted_ids"`
					ServerTime string                 `json:"server_time"`
				}
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)

				require.NotEmpty(t, resp.ServerTime, "server_time must be present")
				_, parseErr := time.Parse(time.RFC3339Nano, resp.ServerTime)
				require.NoError(t, parseErr, "server_time must be valid RFC3339Nano")

				if tt.wantEntries {
					assert.Len(t, resp.Entries, 1)
					assert.Equal(t, "KB5001234", resp.Entries[0].Name)
				} else {
					assert.Empty(t, resp.Entries)
				}

				if tt.wantDeletedIDs {
					assert.Len(t, resp.DeletedIDs, 1)
				} else {
					assert.Empty(t, resp.DeletedIDs)
				}

				// Verify sync.completed event was emitted.
				require.Len(t, bus.emitted, 1)
				assert.Equal(t, "sync.completed", bus.emitted[0].Type)
			}
		})
	}
}

func TestSync_EmitsSyncCompletedEvent(t *testing.T) {
	const validAPIKey = "test-key"
	bus := &mockEventBus{}
	querier := &mockSyncQuerier{
		listUpdatedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error) {
			return []sqlcgen.PatchCatalog{}, nil
		},
		listDeletedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]pgtype.UUID, error) {
			return nil, nil
		},
	}

	handler := v1.NewSyncHandler(querier, validAPIKey, bus)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since=2026-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+validAPIKey)

	rec := httptest.NewRecorder()
	handler.Sync(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, bus.emitted, 1)

	evt := bus.emitted[0]
	assert.Equal(t, "sync.completed", evt.Type)
	assert.Equal(t, "sync", evt.Resource)
	assert.Equal(t, "completed", evt.Action)

	// Verify payload.
	payload, ok := evt.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 0, payload["entries_count"])
	assert.Equal(t, 0, payload["deleted_count"])
	assert.Equal(t, "2026-01-01T00:00:00Z", payload["since"])
}

func TestSync_NilEventBus(t *testing.T) {
	const validAPIKey = "test-key"
	querier := &mockSyncQuerier{
		listUpdatedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error) {
			return []sqlcgen.PatchCatalog{}, nil
		},
		listDeletedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]pgtype.UUID, error) {
			return nil, nil
		},
	}

	// Pass nil eventBus — should not panic.
	handler := v1.NewSyncHandler(querier, validAPIKey, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since=2026-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+validAPIKey)

	rec := httptest.NewRecorder()
	handler.Sync(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSyncCVEs(t *testing.T) {
	const validAPIKey = "test-sync-api-key"

	testUUID := pgtype.UUID{Valid: true}
	testUUID.Bytes = [16]byte{0x01}

	testCVE := sqlcgen.CVEFeed{
		ID:           testUUID,
		CveID:        "CVE-2024-21762",
		Severity:     "critical",
		Source:       "nvd",
		CvssV3Vector: pgtype.Text{String: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", Valid: true},
	}

	tests := []struct {
		name       string
		authHeader string
		query      string
		mock       *mockSyncQuerier
		wantStatus int
		wantCVEs   int
	}{
		{
			name:       "valid request returns CVEs",
			authHeader: "Bearer " + validAPIKey,
			query:      "since=2026-01-01T00:00:00Z",
			mock: &mockSyncQuerier{
				listCVEUpdatedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.CVEFeed, error) {
					return []sqlcgen.CVEFeed{testCVE}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCVEs:   1,
		},
		{
			name:       "missing auth returns 401",
			authHeader: "",
			query:      "since=2026-01-01T00:00:00Z",
			mock:       &mockSyncQuerier{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong key returns 401",
			authHeader: "Bearer wrong",
			query:      "since=2026-01-01T00:00:00Z",
			mock:       &mockSyncQuerier{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing since returns 400",
			authHeader: "Bearer " + validAPIKey,
			query:      "",
			mock:       &mockSyncQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty result returns empty array",
			authHeader: "Bearer " + validAPIKey,
			query:      "since=2026-01-01T00:00:00Z",
			mock: &mockSyncQuerier{
				listCVEUpdatedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.CVEFeed, error) {
					return nil, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCVEs:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := v1.NewSyncHandler(tt.mock, validAPIKey, bus)

			url := "/api/v1/sync/cves"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.SyncCVEs(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var resp struct {
					CVEs       []json.RawMessage `json:"cves"`
					ServerTime string            `json:"server_time"`
				}
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Len(t, resp.CVEs, tt.wantCVEs)
				assert.NotEmpty(t, resp.ServerTime)
			}
		})
	}
}
