package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEventBus records emitted events.
type fakeEventBus struct {
	events []domain.DomainEvent
}

func (f *fakeEventBus) Emit(_ context.Context, event domain.DomainEvent) error {
	f.events = append(f.events, event)
	return nil
}
func (f *fakeEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (f *fakeEventBus) Close() error                                    { return nil }

// fakeScanner mocks EndpointScanner.
type fakeScanner struct {
	called bool
	err    error
}

func (f *fakeScanner) ScanSingle(_ context.Context, _, _ pgtype.UUID, _, _ string) (pgtype.UUID, error) {
	f.called = true
	return pgtype.UUID{}, f.err
}

// fakeEndpointQuerier mocks EndpointQuerier.
type fakeEndpointQuerier struct {
	listResult         []sqlcgen.ListEndpointsRow
	listErr            error
	countResult        int64
	countErr           error
	getResult          sqlcgen.GetEndpointByIDRow
	getErr             error
	updateResult       sqlcgen.Endpoint
	updateErr          error
	softDeleteResult   sqlcgen.Endpoint
	softDeleteErr      error
	inventoryResult    sqlcgen.EndpointInventory
	inventoryErr       error
	cveCounts          []sqlcgen.CountEndpointCVEsByStatusRow
	cveCountsErr       error
	nicResult          []sqlcgen.EndpointNetworkInterface
	nicErr             error
	packagesResult     []sqlcgen.ListEndpointPackagesByEndpointRow
	packagesErr        error
	deployTargetResult []sqlcgen.DeploymentTarget
	deployTargetErr    error
}

func (f *fakeEndpointQuerier) ListEndpoints(_ context.Context, _ sqlcgen.ListEndpointsParams) ([]sqlcgen.ListEndpointsRow, error) {
	return f.listResult, f.listErr
}
func (f *fakeEndpointQuerier) CountEndpoints(_ context.Context, _ sqlcgen.CountEndpointsParams) (int64, error) {
	return f.countResult, f.countErr
}
func (f *fakeEndpointQuerier) GetEndpointByID(_ context.Context, _ sqlcgen.GetEndpointByIDParams) (sqlcgen.GetEndpointByIDRow, error) {
	return f.getResult, f.getErr
}
func (f *fakeEndpointQuerier) UpdateEndpoint(_ context.Context, _ sqlcgen.UpdateEndpointParams) (sqlcgen.Endpoint, error) {
	return f.updateResult, f.updateErr
}
func (f *fakeEndpointQuerier) SoftDeleteEndpoint(_ context.Context, _ sqlcgen.SoftDeleteEndpointParams) (sqlcgen.Endpoint, error) {
	return f.softDeleteResult, f.softDeleteErr
}
func (f *fakeEndpointQuerier) GetLatestEndpointInventory(_ context.Context, _ sqlcgen.GetLatestEndpointInventoryParams) (sqlcgen.EndpointInventory, error) {
	return f.inventoryResult, f.inventoryErr
}
func (f *fakeEndpointQuerier) CountEndpointCVEsByStatus(_ context.Context, _ sqlcgen.CountEndpointCVEsByStatusParams) ([]sqlcgen.CountEndpointCVEsByStatusRow, error) {
	return f.cveCounts, f.cveCountsErr
}
func (f *fakeEndpointQuerier) ListEndpointCVEsAffected(_ context.Context, _ sqlcgen.ListEndpointCVEsAffectedParams) ([]sqlcgen.ListEndpointCVEsAffectedRow, error) {
	return nil, nil
}
func (f *fakeEndpointQuerier) ListEndpointNetworkInterfaces(_ context.Context, _ sqlcgen.ListEndpointNetworkInterfacesParams) ([]sqlcgen.EndpointNetworkInterface, error) {
	return f.nicResult, f.nicErr
}
func (f *fakeEndpointQuerier) ListEndpointPackagesByEndpoint(_ context.Context, _ sqlcgen.ListEndpointPackagesByEndpointParams) ([]sqlcgen.ListEndpointPackagesByEndpointRow, error) {
	return f.packagesResult, f.packagesErr
}
func (f *fakeEndpointQuerier) ListDeploymentTargetsByEndpoint(_ context.Context, _ sqlcgen.ListDeploymentTargetsByEndpointParams) ([]sqlcgen.DeploymentTarget, error) {
	return f.deployTargetResult, f.deployTargetErr
}
func (f *fakeEndpointQuerier) ListPatchesForEndpoint(_ context.Context, _ sqlcgen.ListPatchesForEndpointParams) ([]sqlcgen.ListPatchesForEndpointRow, error) {
	return nil, nil
}
func (f *fakeEndpointQuerier) ListAvailablePatchesForEndpointByOS(_ context.Context, _ sqlcgen.ListAvailablePatchesForEndpointByOSParams) ([]sqlcgen.ListAvailablePatchesForEndpointByOSRow, error) {
	return nil, nil
}
func (f *fakeEndpointQuerier) ListEndpointsForExport(_ context.Context, _ sqlcgen.ListEndpointsForExportParams) ([]sqlcgen.ListEndpointsForExportRow, error) {
	return nil, nil
}
func (f *fakeEndpointQuerier) ListTagsForEndpoint(_ context.Context, _ sqlcgen.ListTagsForEndpointParams) ([]sqlcgen.Tag, error) {
	return nil, nil
}
func (f *fakeEndpointQuerier) ListAvailablePatchesForEndpointByPackage(_ context.Context, _ sqlcgen.ListAvailablePatchesForEndpointByPackageParams) ([]sqlcgen.ListAvailablePatchesForEndpointByPackageRow, error) {
	return nil, nil
}
func (f *fakeEndpointQuerier) ListAuditEventsByEndpoint(_ context.Context, _ sqlcgen.ListAuditEventsByEndpointParams) ([]sqlcgen.AuditEvent, error) {
	return nil, nil
}
func (f *fakeEndpointQuerier) CountAuditEventsByEndpoint(_ context.Context, _ sqlcgen.CountAuditEventsByEndpointParams) (int64, error) {
	return 0, nil
}

func (f *fakeEndpointQuerier) GetActiveRunScanByAgent(_ context.Context, _ sqlcgen.GetActiveRunScanByAgentParams) (sqlcgen.Command, error) {
	return sqlcgen.Command{}, pgx.ErrNoRows
}

func tenantCtx(ctx context.Context) context.Context {
	return tenant.WithTenantID(ctx, "00000000-0000-0000-0000-000000000001")
}

func chiCtx(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func validEndpoint() sqlcgen.Endpoint {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.Endpoint{
		ID:        id,
		TenantID:  tid,
		Hostname:  "test-host",
		OsFamily:  "linux",
		OsVersion: "Ubuntu 24.04",
		Status:    "active",
	}
}

func validEndpointRow() sqlcgen.ListEndpointsRow {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.ListEndpointsRow{
		ID:                  id,
		TenantID:            tid,
		Hostname:            "test-host",
		OsFamily:            "linux",
		OsVersion:           "Ubuntu 24.04",
		Status:              "active",
		IpAddress:           pgtype.Text{String: "10.0.0.1", Valid: true},
		CveCount:            3,
		PendingPatchesCount: 5,
		CompliancePct:       pgtype.Float8{Float64: 87.5, Valid: true},
		CpuUsagePercent:     pgtype.Int2{Int16: 45, Valid: true},
	}
}

func validGetEndpointByIDRow() sqlcgen.GetEndpointByIDRow {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.GetEndpointByIDRow{
		ID:        id,
		TenantID:  tid,
		Hostname:  "test-host",
		OsFamily:  "linux",
		OsVersion: "Ubuntu 24.04",
		Status:    "active",
	}
}

// --- List Tests ---

func TestEndpointHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *fakeEndpointQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns empty list",
			querier: &fakeEndpointQuerier{
				listResult:  []sqlcgen.ListEndpointsRow{},
				countResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "returns endpoints",
			query: "?limit=10",
			querier: &fakeEndpointQuerier{
				listResult:  []sqlcgen.ListEndpointsRow{validEndpointRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "invalid cursor returns 400",
			query:      "?cursor=bad-cursor",
			querier:    &fakeEndpointQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
		{
			name: "store error returns 500",
			querier: &fakeEndpointQuerier{
				listErr: fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
		{
			name:  "filter by status",
			query: "?status=active",
			querier: &fakeEndpointQuerier{
				listResult:  []sqlcgen.ListEndpointsRow{validEndpointRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by os_family",
			query: "?os_family=linux",
			querier: &fakeEndpointQuerier{
				listResult:  []sqlcgen.ListEndpointsRow{validEndpointRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by search",
			query: "?search=test",
			querier: &fakeEndpointQuerier{
				listResult:  []sqlcgen.ListEndpointsRow{validEndpointRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by group_id",
			query: "?group_id=00000000-0000-0000-0000-000000000088",
			querier: &fakeEndpointQuerier{
				listResult:  []sqlcgen.ListEndpointsRow{validEndpointRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewEndpointHandler(tt.querier, &fakeEventBus{}, &fakeScanner{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints"+tt.query, nil)
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

func TestListEndpoints_EnrichedFields(t *testing.T) {
	row := validEndpointRow()
	querier := &fakeEndpointQuerier{
		listResult:  []sqlcgen.ListEndpointsRow{row},
		countResult: 1,
	}

	h := v1.NewEndpointHandler(querier, &fakeEventBus{}, &fakeScanner{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	data, ok := body["data"].([]any)
	require.True(t, ok, "expected data to be a list")
	require.Len(t, data, 1)

	item, ok := data[0].(map[string]any)
	require.True(t, ok, "expected data item to be a map")

	assert.Equal(t, float64(3), item["cve_count"], "expected cve_count to be 3")
	assert.Equal(t, float64(5), item["pending_patches_count"], "expected pending_patches_count to be 5")
	assert.Equal(t, "10.0.0.1", item["ip_address"], "expected ip_address to be 10.0.0.1")
	assert.Equal(t, float64(87.5), item["compliance_pct"], "expected compliance_pct to be 87.5")
	assert.Equal(t, float64(45), item["cpu_usage_percent"], "expected cpu_usage_percent to be 45")
	assert.Equal(t, []any{}, item["tags"], "expected tags to be empty array")
}

// --- Get Tests ---

func TestEndpointHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeEndpointQuerier
		wantStatus int
	}{
		{
			name:       "valid ID returns 200",
			id:         "00000000-0000-0000-0000-000000000099",
			querier:    &fakeEndpointQuerier{getResult: validGetEndpointByIDRow()},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeEndpointQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000099",
			querier:    &fakeEndpointQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewEndpointHandler(tt.querier, &fakeEventBus{}, &fakeScanner{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/"+tt.id, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Update Tests ---

func TestEndpointHandler_Update(t *testing.T) {
	eb := &fakeEventBus{}
	updated := validEndpoint()
	updated.Hostname = "new-host"

	tests := []struct {
		name       string
		id         string
		body       any
		querier    *fakeEndpointQuerier
		wantStatus int
	}{
		{
			name:       "valid update returns 200",
			id:         "00000000-0000-0000-0000-000000000099",
			body:       map[string]string{"hostname": "new-host", "os_family": "linux", "os_version": "Ubuntu 24.04"},
			querier:    &fakeEndpointQuerier{updateResult: updated},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			body:       map[string]string{},
			querier:    &fakeEndpointQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000099",
			body:       map[string]string{"hostname": "new-host", "os_family": "linux", "os_version": "Ubuntu 24.04"},
			querier:    &fakeEndpointQuerier{updateErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := v1.NewEndpointHandler(tt.querier, eb, &fakeScanner{})
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/endpoints/"+tt.id, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Update(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Len(t, eb.events, 1)
				assert.Equal(t, "endpoint.updated", eb.events[0].Type)
			}
		})
	}
}

// --- Delete Tests ---

func TestEndpointHandler_Delete(t *testing.T) {
	eb := &fakeEventBus{}

	tests := []struct {
		name       string
		id         string
		querier    *fakeEndpointQuerier
		wantStatus int
	}{
		{
			name:       "valid delete returns 204",
			id:         "00000000-0000-0000-0000-000000000099",
			querier:    &fakeEndpointQuerier{softDeleteResult: validEndpoint()},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeEndpointQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000099",
			querier:    &fakeEndpointQuerier{softDeleteErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := v1.NewEndpointHandler(tt.querier, eb, &fakeScanner{})
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/endpoints/"+tt.id, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Delete(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusNoContent {
				assert.Len(t, eb.events, 1)
				assert.Equal(t, "endpoint.deleted", eb.events[0].Type)
			}
		})
	}
}

// --- Get Detail Enriched Tests ---

func TestGetEndpointDetail_EnrichedFields(t *testing.T) {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")

	epRow := sqlcgen.GetEndpointByIDRow{
		ID:              id,
		TenantID:        tid,
		Hostname:        "test-host",
		OsFamily:        "linux",
		OsVersion:       "Ubuntu 24.04",
		Status:          "active",
		CpuModel:        pgtype.Text{String: "Intel Xeon", Valid: true},
		CpuCores:        pgtype.Int4{Int32: 16, Valid: true},
		CpuUsagePercent: pgtype.Int2{Int16: 75, Valid: true},
		MemoryTotalMb:   pgtype.Int8{Int64: 65536, Valid: true},
		MemoryUsedMb:    pgtype.Int8{Int64: 32768, Valid: true},
		EnrolledAt:      pgtype.Timestamptz{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		CertExpiry:      pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}

	var nic1id, nic2id pgtype.UUID
	_ = nic1id.Scan("00000000-0000-0000-0000-000000000011")
	_ = nic2id.Scan("00000000-0000-0000-0000-000000000022")

	nics := []sqlcgen.EndpointNetworkInterface{
		{
			ID:        nic1id,
			TenantID:  tid,
			Name:      "eth0",
			IpAddress: pgtype.Text{String: "10.0.0.1", Valid: true},
			Status:    "up",
		},
		{
			ID:       nic2id,
			TenantID: tid,
			Name:     "lo",
			Status:   "up",
		},
	}

	querier := &fakeEndpointQuerier{
		getResult:    epRow,
		nicResult:    nics,
		inventoryErr: pgx.ErrNoRows,
	}

	h := v1.NewEndpointHandler(querier, &fakeEventBus{}, &fakeScanner{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/00000000-0000-0000-0000-000000000099", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.Equal(t, "Intel Xeon", body["cpu_model"], "expected cpu_model")
	assert.Equal(t, float64(16), body["cpu_cores"], "expected cpu_cores")
	assert.Equal(t, float64(75), body["cpu_usage_percent"], "expected cpu_usage_percent")
	assert.Equal(t, float64(65536), body["memory_total_mb"], "expected memory_total_mb")
	assert.Equal(t, float64(32768), body["memory_used_mb"], "expected memory_used_mb")
	assert.NotNil(t, body["enrolled_at"], "expected enrolled_at to be set")
	assert.NotNil(t, body["cert_expiry"], "expected cert_expiry to be set")

	nicsArr, ok := body["network_interfaces"].([]any)
	require.True(t, ok, "expected network_interfaces to be an array")
	assert.Len(t, nicsArr, 2, "expected 2 network interfaces")
}

// --- Scan Tests ---

func TestEndpointHandler_Scan(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		querier     *fakeEndpointQuerier
		scanErr     error
		wantStatus  int
		wantScanned bool
	}{
		{
			name:        "valid scan returns 202",
			id:          "00000000-0000-0000-0000-000000000099",
			querier:     &fakeEndpointQuerier{getResult: validGetEndpointByIDRow()},
			wantStatus:  http.StatusAccepted,
			wantScanned: true,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000099",
			querier:    &fakeEndpointQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "scan scheduler error returns 500",
			id:         "00000000-0000-0000-0000-000000000099",
			querier:    &fakeEndpointQuerier{getResult: validGetEndpointByIDRow()},
			scanErr:    assert.AnError,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := &fakeScanner{err: tt.scanErr}
			h := v1.NewEndpointHandler(tt.querier, &fakeEventBus{}, scanner)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints/"+tt.id+"/scan", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Scan(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantScanned {
				assert.True(t, scanner.called, "expected ScanSingle to be called")
			}
		})
	}
}

// --- ListPackages Tests ---

func TestListEndpointPackages(t *testing.T) {
	var id, tid, invID pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000011")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = invID.Scan("00000000-0000-0000-0000-000000000022")

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	pkg1 := sqlcgen.ListEndpointPackagesByEndpointRow{
		ID:          id,
		PackageName: "curl",
		Version:     "7.88.1",
		Arch:        pgtype.Text{String: "amd64", Valid: true},
		Source:      pgtype.Text{String: "apt", Valid: true},
		CreatedAt:   now,
	}

	var id2 pgtype.UUID
	_ = id2.Scan("00000000-0000-0000-0000-000000000033")
	pkg2 := sqlcgen.ListEndpointPackagesByEndpointRow{
		ID:          id2,
		PackageName: "openssl",
		Version:     "3.0.2",
		CreatedAt:   now,
	}

	tests := []struct {
		name       string
		endpointID string
		querier    *fakeEndpointQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name:       "returns packages",
			endpointID: "00000000-0000-0000-0000-000000000099",
			querier: &fakeEndpointQuerier{
				packagesResult: []sqlcgen.ListEndpointPackagesByEndpointRow{pkg1, pkg2},
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "returns empty list",
			endpointID: "00000000-0000-0000-0000-000000000099",
			querier: &fakeEndpointQuerier{
				packagesResult: []sqlcgen.ListEndpointPackagesByEndpointRow{},
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "invalid endpoint id returns 400",
			endpointID: "not-a-uuid",
			querier:    &fakeEndpointQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
		{
			name:       "store error returns 500",
			endpointID: "00000000-0000-0000-0000-000000000099",
			querier: &fakeEndpointQuerier{
				packagesErr: fmt.Errorf("database error"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewEndpointHandler(tt.querier, &fakeEventBus{}, &fakeScanner{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/"+tt.endpointID+"/packages", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.endpointID)
			rec := httptest.NewRecorder()

			h.ListPackages(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body["data"], tt.wantLen)
			}
		})
	}
}

func TestListEndpointPackages_Fields(t *testing.T) {
	var id, tid, invID pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000011")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = invID.Scan("00000000-0000-0000-0000-000000000022")

	pkg := sqlcgen.ListEndpointPackagesByEndpointRow{
		ID:          id,
		PackageName: "curl",
		Version:     "7.88.1",
		Arch:        pgtype.Text{String: "amd64", Valid: true},
		Source:      pgtype.Text{String: "apt", Valid: true},
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	querier := &fakeEndpointQuerier{packagesResult: []sqlcgen.ListEndpointPackagesByEndpointRow{pkg}}
	h := v1.NewEndpointHandler(querier, &fakeEventBus{}, &fakeScanner{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/00000000-0000-0000-0000-000000000099/packages", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.ListPackages(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	data, ok := body["data"].([]any)
	require.True(t, ok, "data should be []any")
	require.Len(t, data, 1)
	item, ok := data[0].(map[string]any)
	require.True(t, ok, "item should be map[string]any")
	assert.Equal(t, "curl", item["package_name"])
	assert.Equal(t, "7.88.1", item["version"])
	assert.Equal(t, "amd64", item["arch"])
	assert.Equal(t, "apt", item["source"])
	assert.NotEmpty(t, item["created_at"])
}

// --- ListDeploymentHistory Tests ---

func TestListEndpointDeployments(t *testing.T) {
	var id, tid, depID, patchID pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000011")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = depID.Scan("00000000-0000-0000-0000-000000000044")
	_ = patchID.Scan("00000000-0000-0000-0000-000000000055")

	started := time.Now().Add(-2 * time.Minute)
	completed := time.Now()

	target := sqlcgen.DeploymentTarget{
		ID:           id,
		TenantID:     tid,
		DeploymentID: depID,
		EndpointID:   id,
		PatchID:      patchID,
		Status:       "succeeded",
		StartedAt:    pgtype.Timestamptz{Time: started, Valid: true},
		CompletedAt:  pgtype.Timestamptz{Time: completed, Valid: true},
		CreatedAt:    pgtype.Timestamptz{Time: started, Valid: true},
	}

	tests := []struct {
		name       string
		endpointID string
		querier    *fakeEndpointQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name:       "returns deployment targets",
			endpointID: "00000000-0000-0000-0000-000000000099",
			querier: &fakeEndpointQuerier{
				deployTargetResult: []sqlcgen.DeploymentTarget{target},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "returns empty list",
			endpointID: "00000000-0000-0000-0000-000000000099",
			querier: &fakeEndpointQuerier{
				deployTargetResult: []sqlcgen.DeploymentTarget{},
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "invalid endpoint id returns 400",
			endpointID: "not-a-uuid",
			querier:    &fakeEndpointQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
		{
			name:       "store error returns 500",
			endpointID: "00000000-0000-0000-0000-000000000099",
			querier: &fakeEndpointQuerier{
				deployTargetErr: fmt.Errorf("database error"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewEndpointHandler(tt.querier, &fakeEventBus{}, &fakeScanner{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/"+tt.endpointID+"/deployments", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.endpointID)
			rec := httptest.NewRecorder()

			h.ListDeploymentHistory(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body["data"], tt.wantLen)
			}
		})
	}
}

func TestListEndpointDeployments_Fields(t *testing.T) {
	var id, tid, depID, patchID pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000011")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = depID.Scan("00000000-0000-0000-0000-000000000044")
	_ = patchID.Scan("00000000-0000-0000-0000-000000000055")

	started := time.Now().Add(-2 * time.Minute)
	completed := time.Now()

	target := sqlcgen.DeploymentTarget{
		ID:           id,
		TenantID:     tid,
		DeploymentID: depID,
		EndpointID:   id,
		PatchID:      patchID,
		Status:       "succeeded",
		StartedAt:    pgtype.Timestamptz{Time: started, Valid: true},
		CompletedAt:  pgtype.Timestamptz{Time: completed, Valid: true},
		ErrorMessage: pgtype.Text{},
		CreatedAt:    pgtype.Timestamptz{Time: started, Valid: true},
	}

	querier := &fakeEndpointQuerier{deployTargetResult: []sqlcgen.DeploymentTarget{target}}
	h := v1.NewEndpointHandler(querier, &fakeEventBus{}, &fakeScanner{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/00000000-0000-0000-0000-000000000099/deployments", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.ListDeploymentHistory(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	data, ok := body["data"].([]any)
	require.True(t, ok, "data should be []any")
	require.Len(t, data, 1)
	item, ok := data[0].(map[string]any)
	require.True(t, ok, "item should be map[string]any")
	assert.Equal(t, "succeeded", item["status"])
	assert.NotEmpty(t, item["deployment_id"])
	assert.NotNil(t, item["started_at"])
	assert.NotNil(t, item["completed_at"])
	assert.NotNil(t, item["duration_seconds"])
}
