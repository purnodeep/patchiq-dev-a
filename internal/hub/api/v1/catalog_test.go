package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCatalogQuerier implements v1.CatalogQuerier for testing.
type mockCatalogQuerier struct {
	createFn          func(ctx context.Context, arg sqlcgen.CreateCatalogEntryParams) (sqlcgen.PatchCatalog, error)
	getFn             func(ctx context.Context, id pgtype.UUID) (sqlcgen.PatchCatalog, error)
	listFn            func(ctx context.Context, arg sqlcgen.ListCatalogEntriesParams) ([]sqlcgen.PatchCatalog, error)
	countFn           func(ctx context.Context, arg sqlcgen.CountCatalogEntriesParams) (int64, error)
	listEnrichedFn    func(ctx context.Context, arg sqlcgen.ListCatalogEntriesEnrichedParams) ([]sqlcgen.ListCatalogEntriesEnrichedRow, error)
	countEnrichedFn   func(ctx context.Context, arg sqlcgen.CountCatalogEntriesEnrichedParams) (int64, error)
	updateFn          func(ctx context.Context, arg sqlcgen.UpdateCatalogEntryParams) (sqlcgen.PatchCatalog, error)
	softDeleteFn      func(ctx context.Context, id pgtype.UUID) error
	linkCVEFn         func(ctx context.Context, arg sqlcgen.LinkCatalogCVEParams) error
	unlinkAllCVEsFn   func(ctx context.Context, catalogID pgtype.UUID) error
	listCVEsFn        func(ctx context.Context, catalogID pgtype.UUID) ([]sqlcgen.ListCVEsForCatalogEntryRow, error)
	countCVEsFn       func(ctx context.Context, catalogID pgtype.UUID) (int64, error)
	getCatalogStatsFn func(ctx context.Context) (sqlcgen.GetCatalogStatsRow, error)
	countApprovedFn   func(ctx context.Context) (int64, error)
	countSyncedFn     func(ctx context.Context, catalogID pgtype.UUID) (int64, error)
}

func (m *mockCatalogQuerier) CreateCatalogEntry(ctx context.Context, arg sqlcgen.CreateCatalogEntryParams) (sqlcgen.PatchCatalog, error) {
	if m.createFn != nil {
		return m.createFn(ctx, arg)
	}
	return sqlcgen.PatchCatalog{}, nil
}

func (m *mockCatalogQuerier) GetCatalogEntryByID(ctx context.Context, id pgtype.UUID) (sqlcgen.PatchCatalog, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return sqlcgen.PatchCatalog{}, nil
}

func (m *mockCatalogQuerier) ListCatalogEntries(ctx context.Context, arg sqlcgen.ListCatalogEntriesParams) ([]sqlcgen.PatchCatalog, error) {
	if m.listFn != nil {
		return m.listFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockCatalogQuerier) CountCatalogEntries(ctx context.Context, arg sqlcgen.CountCatalogEntriesParams) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, arg)
	}
	return 0, nil
}

func (m *mockCatalogQuerier) ListCatalogEntriesEnriched(ctx context.Context, arg sqlcgen.ListCatalogEntriesEnrichedParams) ([]sqlcgen.ListCatalogEntriesEnrichedRow, error) {
	if m.listEnrichedFn != nil {
		return m.listEnrichedFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockCatalogQuerier) CountCatalogEntriesEnriched(ctx context.Context, arg sqlcgen.CountCatalogEntriesEnrichedParams) (int64, error) {
	if m.countEnrichedFn != nil {
		return m.countEnrichedFn(ctx, arg)
	}
	return 0, nil
}

func (m *mockCatalogQuerier) UpdateCatalogEntry(ctx context.Context, arg sqlcgen.UpdateCatalogEntryParams) (sqlcgen.PatchCatalog, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, arg)
	}
	return sqlcgen.PatchCatalog{}, nil
}

func (m *mockCatalogQuerier) SoftDeleteCatalogEntry(ctx context.Context, id pgtype.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return nil
}

func (m *mockCatalogQuerier) LinkCatalogCVE(ctx context.Context, arg sqlcgen.LinkCatalogCVEParams) error {
	if m.linkCVEFn != nil {
		return m.linkCVEFn(ctx, arg)
	}
	return nil
}

func (m *mockCatalogQuerier) UnlinkAllCatalogCVEs(ctx context.Context, catalogID pgtype.UUID) error {
	if m.unlinkAllCVEsFn != nil {
		return m.unlinkAllCVEsFn(ctx, catalogID)
	}
	return nil
}

func (m *mockCatalogQuerier) ListCVEsForCatalogEntry(ctx context.Context, catalogID pgtype.UUID) ([]sqlcgen.ListCVEsForCatalogEntryRow, error) {
	if m.listCVEsFn != nil {
		return m.listCVEsFn(ctx, catalogID)
	}
	return nil, nil
}

func (m *mockCatalogQuerier) CountCVEsForCatalogEntry(ctx context.Context, catalogID pgtype.UUID) (int64, error) {
	if m.countCVEsFn != nil {
		return m.countCVEsFn(ctx, catalogID)
	}
	return 0, nil
}

func (m *mockCatalogQuerier) GetCatalogStats(ctx context.Context) (sqlcgen.GetCatalogStatsRow, error) {
	if m.getCatalogStatsFn != nil {
		return m.getCatalogStatsFn(ctx)
	}
	return sqlcgen.GetCatalogStatsRow{}, nil
}

func (m *mockCatalogQuerier) CountApprovedClients(ctx context.Context) (int64, error) {
	if m.countApprovedFn != nil {
		return m.countApprovedFn(ctx)
	}
	return 0, nil
}

func (m *mockCatalogQuerier) CountSyncedClientsForCatalogEntry(ctx context.Context, catalogID pgtype.UUID) (int64, error) {
	if m.countSyncedFn != nil {
		return m.countSyncedFn(ctx, catalogID)
	}
	return 0, nil
}

func (m *mockCatalogQuerier) ListSyncsForCatalogEntry(ctx context.Context, catalogID pgtype.UUID) ([]sqlcgen.ListSyncsForCatalogEntryRow, error) {
	return nil, nil
}

func (m *mockCatalogQuerier) ListApprovedClientsBasic(ctx context.Context) ([]sqlcgen.ListApprovedClientsBasicRow, error) {
	return nil, nil
}

func (m *mockCatalogQuerier) GetFeedSourceByID(ctx context.Context, id pgtype.UUID) (sqlcgen.FeedSource, error) {
	return sqlcgen.FeedSource{}, nil
}

// mockEventBus implements domain.EventBus for testing.
type mockEventBus struct {
	emitted []domain.DomainEvent
	emitErr error
}

func (m *mockEventBus) Emit(_ context.Context, event domain.DomainEvent) error {
	m.emitted = append(m.emitted, event)
	return m.emitErr
}

func (m *mockEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (m *mockEventBus) Close() error                                    { return nil }

// testUUID returns a valid pgtype.UUID from a fixed string pattern.
func testUUID(n byte) pgtype.UUID {
	var u pgtype.UUID
	u.Valid = true
	for i := range u.Bytes {
		u.Bytes[i] = n
	}
	return u
}

func testUUIDString(n byte) string {
	u := testUUID(n)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

// withTenantCtx adds tenant ID to the request context.
func withTenantCtx(r *http.Request) *http.Request {
	ctx := tenant.WithTenantID(r.Context(), "00000000-0000-0000-0000-000000000001")
	return r.WithContext(ctx)
}

// withChiURLParam adds a chi URL parameter to the request context.
func withChiURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func TestCatalogCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]any
		querier    *mockCatalogQuerier
		bus        *mockEventBus
		wantStatus int
		wantErr    string
	}{
		{
			name: "valid create without CVEs",
			body: map[string]any{
				"name": "KB5001", "vendor": "Microsoft", "os_family": "windows",
				"version": "21H2", "severity": "critical",
			},
			querier: &mockCatalogQuerier{
				createFn: func(_ context.Context, arg sqlcgen.CreateCatalogEntryParams) (sqlcgen.PatchCatalog, error) {
					return sqlcgen.PatchCatalog{
						ID: testUUID(1), Name: arg.Name, Vendor: arg.Vendor,
						OsFamily: arg.OsFamily, Version: arg.Version, Severity: arg.Severity,
						FeedSourceID:  testUUID(5),
						SourceUrl:     "https://example.com/patch.msi",
						InstallerType: "msi",
					}, nil
				},
			},
			bus:        &mockEventBus{},
			wantStatus: http.StatusCreated,
		},
		{
			name: "valid create with CVE IDs",
			body: map[string]any{
				"name": "KB5001", "vendor": "Microsoft", "os_family": "windows",
				"version": "21H2", "severity": "critical",
				"cve_ids": []string{testUUIDString(2)},
			},
			querier: &mockCatalogQuerier{
				createFn: func(_ context.Context, arg sqlcgen.CreateCatalogEntryParams) (sqlcgen.PatchCatalog, error) {
					return sqlcgen.PatchCatalog{
						ID: testUUID(1), Name: arg.Name, Vendor: arg.Vendor,
						OsFamily: arg.OsFamily, Version: arg.Version, Severity: arg.Severity,
					}, nil
				},
			},
			bus:        &mockEventBus{},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing required field name",
			body:       map[string]any{"vendor": "Microsoft", "os_family": "windows", "version": "21H2", "severity": "critical"},
			querier:    &mockCatalogQuerier{},
			bus:        &mockEventBus{},
			wantStatus: http.StatusBadRequest,
			wantErr:    "name",
		},
		{
			name:       "missing required field vendor",
			body:       map[string]any{"name": "KB5001", "os_family": "windows", "version": "21H2", "severity": "critical"},
			querier:    &mockCatalogQuerier{},
			bus:        &mockEventBus{},
			wantStatus: http.StatusBadRequest,
			wantErr:    "vendor",
		},
		{
			name: "db error on create",
			body: map[string]any{
				"name": "KB5001", "vendor": "Microsoft", "os_family": "windows",
				"version": "21H2", "severity": "critical",
			},
			querier: &mockCatalogQuerier{
				createFn: func(_ context.Context, _ sqlcgen.CreateCatalogEntryParams) (sqlcgen.PatchCatalog, error) {
					return sqlcgen.PatchCatalog{}, errors.New("db down")
				},
			},
			bus:        &mockEventBus{},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewCatalogHandler(tt.querier, tt.bus)
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/catalog", bytes.NewReader(body))
			req = withTenantCtx(req)
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantErr != "" {
				assert.Contains(t, rec.Body.String(), tt.wantErr)
			}
			if tt.wantStatus == http.StatusCreated {
				assert.Len(t, tt.bus.emitted, 1)
				var resp map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				entry, ok := resp["entry"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, entry, "feed_source_id")
				assert.Contains(t, entry, "source_url")
				assert.Contains(t, entry, "installer_type")
			}
		})
	}
}

func TestCatalogList(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *mockCatalogQuerier
		wantStatus int
		wantTotal  float64
	}{
		{
			name:  "default pagination",
			query: "",
			querier: &mockCatalogQuerier{
				listEnrichedFn: func(_ context.Context, arg sqlcgen.ListCatalogEntriesEnrichedParams) ([]sqlcgen.ListCatalogEntriesEnrichedRow, error) {
					assert.Equal(t, int32(50), arg.QueryLimit)
					assert.Equal(t, int32(0), arg.QueryOffset)
					return []sqlcgen.ListCatalogEntriesEnrichedRow{{
						ID:            testUUID(1),
						Name:          "KB5001",
						FeedSourceID:  testUUID(5),
						SourceUrl:     "https://example.com/patch.msi",
						InstallerType: "msi",
					}}, nil
				},
				countEnrichedFn: func(_ context.Context, _ sqlcgen.CountCatalogEntriesEnrichedParams) (int64, error) {
					return 1, nil
				},
				countApprovedFn: func(_ context.Context) (int64, error) {
					return 5, nil
				},
			},
			wantStatus: http.StatusOK,
			wantTotal:  1,
		},
		{
			name:  "with filters",
			query: "?os_family=windows&severity=critical&search=KB&limit=10&offset=5",
			querier: &mockCatalogQuerier{
				listEnrichedFn: func(_ context.Context, arg sqlcgen.ListCatalogEntriesEnrichedParams) ([]sqlcgen.ListCatalogEntriesEnrichedRow, error) {
					assert.Equal(t, "windows", arg.OsFamily.String)
					assert.Equal(t, "critical", arg.Severity.String)
					assert.Equal(t, "KB", arg.Search.String)
					assert.Equal(t, int32(10), arg.QueryLimit)
					assert.Equal(t, int32(5), arg.QueryOffset)
					return nil, nil
				},
				countEnrichedFn: func(_ context.Context, _ sqlcgen.CountCatalogEntriesEnrichedParams) (int64, error) {
					return 0, nil
				},
			},
			wantStatus: http.StatusOK,
			wantTotal:  0,
		},
		{
			name:  "limit capped at 100",
			query: "?limit=999",
			querier: &mockCatalogQuerier{
				listEnrichedFn: func(_ context.Context, arg sqlcgen.ListCatalogEntriesEnrichedParams) ([]sqlcgen.ListCatalogEntriesEnrichedRow, error) {
					assert.Equal(t, int32(100), arg.QueryLimit)
					return nil, nil
				},
				countEnrichedFn: func(_ context.Context, _ sqlcgen.CountCatalogEntriesEnrichedParams) (int64, error) {
					return 0, nil
				},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewCatalogHandler(tt.querier, &mockEventBus{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog"+tt.query, nil)
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantTotal > 0 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tt.wantTotal, body["total"])
				entries, ok := body["entries"].([]any)
				require.True(t, ok)
				entry, ok := entries[0].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, entry, "feed_source_id")
				assert.Contains(t, entry, "source_url")
				assert.Contains(t, entry, "installer_type")
			}
		})
	}
}

func TestCatalogGet(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *mockCatalogQuerier
		wantStatus int
	}{
		{
			name: "valid ID returns entry with CVEs",
			id:   testUUIDString(1),
			querier: &mockCatalogQuerier{
				getFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.PatchCatalog, error) {
					return sqlcgen.PatchCatalog{
						ID: testUUID(1), Name: "KB5001",
						FeedSourceID:  testUUID(5),
						SourceUrl:     "https://example.com/patch.msi",
						InstallerType: "msi",
					}, nil
				},
				listCVEsFn: func(_ context.Context, _ pgtype.UUID) ([]sqlcgen.ListCVEsForCatalogEntryRow, error) {
					return []sqlcgen.ListCVEsForCatalogEntryRow{{CveID: "CVE-2024-0001"}}, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &mockCatalogQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found returns 404",
			id:   testUUIDString(99),
			querier: &mockCatalogQuerier{
				getFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.PatchCatalog, error) {
					return sqlcgen.PatchCatalog{}, pgx.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewCatalogHandler(tt.querier, &mockEventBus{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/"+tt.id, nil)
			req = withChiURLParam(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Contains(t, body, "cves")
				assert.Contains(t, body, "feed_source_id")
				assert.Contains(t, body, "source_url")
				assert.Contains(t, body, "installer_type")
				assert.Contains(t, body, "synced_count")
				assert.Contains(t, body, "total_clients")
				assert.Contains(t, body, "syncs")
			}
		})
	}
}

func TestCatalogUpdate(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       map[string]any
		querier    *mockCatalogQuerier
		bus        *mockEventBus
		wantStatus int
	}{
		{
			name: "valid update",
			id:   testUUIDString(1),
			body: map[string]any{
				"name": "KB5002", "vendor": "Microsoft", "os_family": "windows",
				"version": "22H2", "severity": "high",
			},
			querier: &mockCatalogQuerier{
				updateFn: func(_ context.Context, arg sqlcgen.UpdateCatalogEntryParams) (sqlcgen.PatchCatalog, error) {
					return sqlcgen.PatchCatalog{
						ID: arg.ID, Name: arg.Name, Vendor: arg.Vendor,
						FeedSourceID:  testUUID(5),
						SourceUrl:     "https://example.com/patch.msi",
						InstallerType: "msi",
					}, nil
				},
			},
			bus:        &mockEventBus{},
			wantStatus: http.StatusOK,
		},
		{
			name: "not found returns 404",
			id:   testUUIDString(99),
			body: map[string]any{
				"name": "KB5002", "vendor": "Microsoft", "os_family": "windows",
				"version": "22H2", "severity": "high",
			},
			querier: &mockCatalogQuerier{
				updateFn: func(_ context.Context, _ sqlcgen.UpdateCatalogEntryParams) (sqlcgen.PatchCatalog, error) {
					return sqlcgen.PatchCatalog{}, pgx.ErrNoRows
				},
			},
			bus:        &mockEventBus{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "bad-uuid",
			body:       map[string]any{"name": "KB5002", "vendor": "Microsoft", "os_family": "windows", "version": "22H2", "severity": "high"},
			querier:    &mockCatalogQuerier{},
			bus:        &mockEventBus{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewCatalogHandler(tt.querier, tt.bus)
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/catalog/"+tt.id, bytes.NewReader(body))
			req = withChiURLParam(req, "id", tt.id)
			req = withTenantCtx(req)
			rec := httptest.NewRecorder()

			h.Update(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Len(t, tt.bus.emitted, 1)
				var resp map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				entry, ok := resp["entry"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, entry, "feed_source_id")
				assert.Contains(t, entry, "source_url")
				assert.Contains(t, entry, "installer_type")
			}
		})
	}
}

func TestCatalogDelete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *mockCatalogQuerier
		bus        *mockEventBus
		wantStatus int
	}{
		{
			name:       "valid delete returns 204",
			id:         testUUIDString(1),
			querier:    &mockCatalogQuerier{},
			bus:        &mockEventBus{},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "bad",
			querier:    &mockCatalogQuerier{},
			bus:        &mockEventBus{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "non-existent still returns 204",
			id:   testUUIDString(99),
			querier: &mockCatalogQuerier{
				softDeleteFn: func(_ context.Context, _ pgtype.UUID) error {
					return nil // soft delete is idempotent
				},
			},
			bus:        &mockEventBus{},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewCatalogHandler(tt.querier, tt.bus)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/catalog/"+tt.id, nil)
			req = withChiURLParam(req, "id", tt.id)
			req = withTenantCtx(req)
			rec := httptest.NewRecorder()

			h.Delete(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusNoContent {
				assert.Len(t, tt.bus.emitted, 1)
			}
		})
	}
}
