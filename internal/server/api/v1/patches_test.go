package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePatchQuerier mocks PatchQuerier.
type fakePatchQuerier struct {
	listResult                []sqlcgen.ListPatchesFilteredRow
	listErr                   error
	countResult               int64
	countErr                  error
	getResult                 sqlcgen.Patch
	getErr                    error
	listCVEsResult            []sqlcgen.CVE
	listCVEsErr               error
	remediationResult         sqlcgen.GetPatchRemediationRow
	remediationErr            error
	affectedEndpointsResult   []sqlcgen.ListAffectedEndpointsForPatchRow
	affectedEndpointsErr      error
	affectedEndpointsCount    int64
	affectedEndpointsCountErr error
	deploymentsResult         []sqlcgen.ListDeploymentsForPatchRow
	deploymentsErr            error
	deploymentHistoryResult   []sqlcgen.ListDeploymentHistoryForPatchRow
	deploymentHistoryErr      error
	highestCVSSResult         float64
	highestCVSSErr            error
}

func (f *fakePatchQuerier) ListPatchesFiltered(_ context.Context, _ sqlcgen.ListPatchesFilteredParams) ([]sqlcgen.ListPatchesFilteredRow, error) {
	return f.listResult, f.listErr
}
func (f *fakePatchQuerier) CountPatchesFiltered(_ context.Context, _ sqlcgen.CountPatchesFilteredParams) (int64, error) {
	return f.countResult, f.countErr
}
func (f *fakePatchQuerier) CountPatchesBySeverity(_ context.Context, _ sqlcgen.CountPatchesBySeverityParams) ([]sqlcgen.CountPatchesBySeverityRow, error) {
	return nil, nil
}
func (f *fakePatchQuerier) GetPatchByID(_ context.Context, _ sqlcgen.GetPatchByIDParams) (sqlcgen.Patch, error) {
	return f.getResult, f.getErr
}
func (f *fakePatchQuerier) ListCVEsForPatch(_ context.Context, _ sqlcgen.ListCVEsForPatchParams) ([]sqlcgen.CVE, error) {
	return f.listCVEsResult, f.listCVEsErr
}
func (f *fakePatchQuerier) GetPatchRemediation(_ context.Context, _ sqlcgen.GetPatchRemediationParams) (sqlcgen.GetPatchRemediationRow, error) {
	return f.remediationResult, f.remediationErr
}
func (f *fakePatchQuerier) ListAffectedEndpointsForPatch(_ context.Context, _ sqlcgen.ListAffectedEndpointsForPatchParams) ([]sqlcgen.ListAffectedEndpointsForPatchRow, error) {
	return f.affectedEndpointsResult, f.affectedEndpointsErr
}
func (f *fakePatchQuerier) CountAffectedEndpointsForPatch(_ context.Context, _ sqlcgen.CountAffectedEndpointsForPatchParams) (int64, error) {
	return f.affectedEndpointsCount, f.affectedEndpointsCountErr
}
func (f *fakePatchQuerier) ListDeploymentsForPatch(_ context.Context, _ sqlcgen.ListDeploymentsForPatchParams) ([]sqlcgen.ListDeploymentsForPatchRow, error) {
	return f.deploymentsResult, f.deploymentsErr
}
func (f *fakePatchQuerier) ListDeploymentHistoryForPatch(_ context.Context, _ sqlcgen.ListDeploymentHistoryForPatchParams) ([]sqlcgen.ListDeploymentHistoryForPatchRow, error) {
	return f.deploymentHistoryResult, f.deploymentHistoryErr
}
func (f *fakePatchQuerier) GetPatchHighestCVSS(_ context.Context, _ sqlcgen.GetPatchHighestCVSSParams) (float64, error) {
	return f.highestCVSSResult, f.highestCVSSErr
}

func (f *fakePatchQuerier) ListEndpointsByTenant(_ context.Context, _ pgtype.UUID) ([]sqlcgen.Endpoint, error) {
	return nil, nil
}

func validPatchRow() sqlcgen.ListPatchesFilteredRow {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.ListPatchesFilteredRow{
		ID:       id,
		TenantID: tid,
		Name:     "openssl-update",
		Version:  "1.1.1w-1",
		Severity: "critical",
		OsFamily: "linux",
		Status:   "available",
		CveCount: 2,
	}
}

func validPatch() sqlcgen.Patch {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.Patch{
		ID:          id,
		TenantID:    tid,
		Name:        "openssl-update",
		Version:     "1.1.1w-1",
		Severity:    "critical",
		OsFamily:    "linux",
		Status:      "available",
		Description: pgtype.Text{String: "Security update for OpenSSL", Valid: true},
	}
}

// --- List Tests ---

func TestPatchHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *fakePatchQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns empty list",
			querier: &fakePatchQuerier{
				listResult:  []sqlcgen.ListPatchesFilteredRow{},
				countResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "returns patches",
			query: "?limit=10",
			querier: &fakePatchQuerier{
				listResult:  []sqlcgen.ListPatchesFilteredRow{validPatchRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "invalid cursor returns 400",
			query:      "?cursor=bad-cursor",
			querier:    &fakePatchQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
		{
			name: "store error returns 500",
			querier: &fakePatchQuerier{
				listErr: fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
		{
			name:  "filter by severity",
			query: "?severity=critical",
			querier: &fakePatchQuerier{
				listResult:  []sqlcgen.ListPatchesFilteredRow{validPatchRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by os_family",
			query: "?os_family=linux",
			querier: &fakePatchQuerier{
				listResult:  []sqlcgen.ListPatchesFilteredRow{validPatchRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by status",
			query: "?status=available",
			querier: &fakePatchQuerier{
				listResult:  []sqlcgen.ListPatchesFilteredRow{validPatchRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "search by name",
			query: "?search=openssl",
			querier: &fakePatchQuerier{
				listResult:  []sqlcgen.ListPatchesFilteredRow{validPatchRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "count error returns 500",
			querier: &fakePatchQuerier{
				listResult: []sqlcgen.ListPatchesFilteredRow{},
				countErr:   fmt.Errorf("count failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewPatchHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/patches"+tt.query, nil)
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

func TestPatchHandler_Get_CVEResponseIncludesExploitFields(t *testing.T) {
	q := &fakePatchQuerier{
		getResult: validPatch(),
		listCVEsResult: []sqlcgen.CVE{{
			CveID:            "CVE-2024-21412",
			Severity:         "critical",
			ExploitAvailable: true,
			CisaKevDueDate:   pgtype.Date{Valid: true},
			CvssV3Vector:     pgtype.Text{String: "CVSS:3.1/AV:N/AC:L", Valid: true},
		}},
		remediationResult: sqlcgen.GetPatchRemediationRow{EndpointsAffected: 1},
	}
	h := v1.NewPatchHandler(q)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patches/00000000-0000-0000-0000-000000000099", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	w := httptest.NewRecorder()
	h.Get(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	cves, ok := body["cves"].([]any)
	require.True(t, ok, "expected cves array")
	require.Len(t, cves, 1)
	cve, ok := cves[0].(map[string]any)
	require.True(t, ok, "expected cve object")
	assert.Equal(t, true, cve["exploit_available"])
	assert.Equal(t, true, cve["cisa_kev"])
	assert.Equal(t, "CVSS:3.1/AV:N/AC:L", cve["cvss_v3_vector"])
}

func TestPatchHandler_QuickDeploy_ReturnsIntent(t *testing.T) {
	t.Run("without pool returns 500 not configured", func(t *testing.T) {
		q := &fakePatchQuerier{getResult: validPatch()}
		h := v1.NewPatchHandler(q)

		body := `{"name":"Test Deploy","config_type":"install","endpoint_filter":"all"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/patches/00000000-0000-0000-0000-000000000099/deploy", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(tenantCtx(req.Context()))
		req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
		w := httptest.NewRecorder()
		h.QuickDeploy(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		var resp map[string]any
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, "NOT_CONFIGURED", resp["code"])
	})

	t.Run("invalid patch ID returns 400", func(t *testing.T) {
		q := &fakePatchQuerier{getResult: validPatch()}
		h := v1.NewPatchHandler(q)

		body := `{"name":"Test Deploy"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/patches/bad-uuid/deploy", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(tenantCtx(req.Context()))
		req = chiCtx(req, "id", "bad-uuid")
		w := httptest.NewRecorder()
		h.QuickDeploy(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("patch not found returns 404", func(t *testing.T) {
		q := &fakePatchQuerier{getErr: pgx.ErrNoRows}
		h := v1.NewPatchHandler(q)

		body := `{"name":"Test Deploy"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/patches/00000000-0000-0000-0000-000000000099/deploy", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(tenantCtx(req.Context()))
		req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
		w := httptest.NewRecorder()
		h.QuickDeploy(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})
}

// --- Get Tests ---

func TestPatchHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakePatchQuerier
		wantStatus int
	}{
		{
			name: "valid ID returns 200 with full CVE objects and sections",
			id:   "00000000-0000-0000-0000-000000000099",
			querier: &fakePatchQuerier{
				getResult: validPatch(),
				listCVEsResult: []sqlcgen.CVE{
					{CveID: "CVE-2024-0001", Severity: "critical", CvssV3Score: pgtype.Numeric{Valid: true}},
					{CveID: "CVE-2024-0002", Severity: "high", CvssV3Score: pgtype.Numeric{Valid: true}},
				},
				remediationResult: sqlcgen.GetPatchRemediationRow{
					EndpointsAffected: 5,
					EndpointsPatched:  3,
					EndpointsPending:  2,
					EndpointsFailed:   2,
				},
				affectedEndpointsResult: []sqlcgen.ListAffectedEndpointsForPatchRow{
					{Hostname: "web-01", OsFamily: "linux", Status: "vulnerable"},
				},
				affectedEndpointsCount: 1,
				deploymentsResult: []sqlcgen.ListDeploymentsForPatchRow{
					{Status: "completed", TotalTargets: 10, SuccessCount: 9, FailedCount: 1},
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakePatchQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000099",
			querier:    &fakePatchQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "db error returns 500",
			id:         "00000000-0000-0000-0000-000000000050",
			querier:    &fakePatchQuerier{getErr: fmt.Errorf("db error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewPatchHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/patches/"+tt.id, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

				// CVEs should be full objects, not string IDs
				cves, ok := body["cves"].([]any)
				require.True(t, ok, "expected cves array in response")
				assert.Len(t, cves, 2)
				cve0, ok := cves[0].(map[string]any)
				require.True(t, ok, "expected cve0 object")
				assert.Equal(t, "CVE-2024-0001", cve0["cve_id"])
				assert.Equal(t, "critical", cve0["severity"])

				// Remediation
				rem, ok := body["remediation"].(map[string]any)
				require.True(t, ok, "expected remediation object in response")
				assert.Equal(t, float64(5), rem["endpoints_affected"])
				assert.Equal(t, float64(3), rem["endpoints_patched"])
				assert.Equal(t, float64(2), rem["endpoints_pending"])
				assert.Equal(t, float64(2), rem["endpoints_failed"])

				// Affected endpoints
				ae, ok := body["affected_endpoints"].(map[string]any)
				require.True(t, ok, "expected affected_endpoints object in response")
				assert.Equal(t, float64(1), ae["total"])
				items, ok := ae["items"].([]any)
				require.True(t, ok)
				assert.Len(t, items, 1)
				item0, ok := items[0].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "web-01", item0["hostname"])

				// Deployment history
				history, ok := body["deployment_history"].([]any)
				require.True(t, ok, "expected deployment_history array in response")
				assert.Len(t, history, 0) // deploymentHistoryResult not set in this test case
			}
		})
	}
}

// TestPatchHandler_Get_ReturnsCVEObjects verifies CVEs are returned as full objects.
func TestPatchHandler_Get_ReturnsCVEObjects(t *testing.T) {
	vector := "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
	h := v1.NewPatchHandler(&fakePatchQuerier{
		getResult: validPatch(),
		listCVEsResult: []sqlcgen.CVE{
			{
				CveID:            "CVE-2024-1234",
				Severity:         "critical",
				ExploitAvailable: true,
				CvssV3Vector:     pgtype.Text{String: vector, Valid: true},
			},
		},
		remediationResult: sqlcgen.GetPatchRemediationRow{EndpointsAffected: 1},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patches/00000000-0000-0000-0000-000000000099", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	cves, ok := body["cves"].([]any)
	require.True(t, ok, "expected cves array")
	require.Len(t, cves, 1)

	cve, ok := cves[0].(map[string]any)
	require.True(t, ok, "expected cve object")
	assert.Equal(t, "CVE-2024-1234", cve["cve_id"])
	assert.Equal(t, "critical", cve["severity"])
	assert.Equal(t, true, cve["exploit_available"])
	assert.Equal(t, vector, cve["cvss_v3_vector"])
	assert.Equal(t, "Network", cve["attack_vector"])
}

// TestPatchHandler_Get_ReturnsAffectedEndpoints verifies affected_endpoints list is returned.
func TestPatchHandler_Get_ReturnsAffectedEndpoints(t *testing.T) {
	h := v1.NewPatchHandler(&fakePatchQuerier{
		getResult: validPatch(),
		affectedEndpointsResult: []sqlcgen.ListAffectedEndpointsForPatchRow{
			{
				Hostname:    "server-01",
				OsFamily:    "linux",
				Status:      "online",
				PatchStatus: "pending",
			},
		},
		remediationResult: sqlcgen.GetPatchRemediationRow{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patches/00000000-0000-0000-0000-000000000099", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	ae, ok := body["affected_endpoints"].(map[string]any)
	require.True(t, ok, "expected affected_endpoints object")
	items, ok := ae["items"].([]any)
	require.True(t, ok, "expected items array")
	require.Len(t, items, 1)

	ep, ok := items[0].(map[string]any)
	require.True(t, ok, "expected endpoint object")
	assert.Equal(t, "server-01", ep["hostname"])
	assert.Equal(t, "pending", ep["patch_status"])
}

// TestPatchHandler_Get_ReturnsDeploymentHistory verifies deployment_history is returned.
func TestPatchHandler_Get_ReturnsDeploymentHistory(t *testing.T) {
	var depID pgtype.UUID
	_ = depID.Scan("00000000-0000-0000-0000-000000000077")
	var createdBy pgtype.UUID
	_ = createdBy.Scan("00000000-0000-0000-0000-000000000001")

	h := v1.NewPatchHandler(&fakePatchQuerier{
		getResult: validPatch(),
		deploymentHistoryResult: []sqlcgen.ListDeploymentHistoryForPatchRow{
			{
				ID:           depID,
				Status:       "completed",
				CreatedBy:    createdBy,
				TotalTargets: 5,
				SuccessCount: 4,
				FailedCount:  1,
			},
		},
		remediationResult: sqlcgen.GetPatchRemediationRow{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patches/00000000-0000-0000-0000-000000000099", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	history, ok := body["deployment_history"].([]any)
	require.True(t, ok, "expected deployment_history array")
	require.Len(t, history, 1)

	entry, ok := history[0].(map[string]any)
	require.True(t, ok, "expected deployment entry object")
	assert.Equal(t, "completed", entry["status"])
	assert.Equal(t, float64(5), entry["total_targets"])
	assert.Equal(t, float64(4), entry["success_count"])
	assert.Equal(t, float64(1), entry["failed_count"])
}

// TestPatchHandler_QuickDeploy_NoPool verifies POST /patches/{id}/deploy returns 500
// when no transaction pool is configured (WithPool not called).
func TestPatchHandler_QuickDeploy_NoPool(t *testing.T) {
	h := v1.NewPatchHandler(&fakePatchQuerier{
		getResult: validPatch(),
	})

	body := `{"name":"Deploy openssl","description":"quick deploy","config_type":"install"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/patches/00000000-0000-0000-0000-000000000099/deploy", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.QuickDeploy(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "NOT_CONFIGURED", resp["code"])
}

// TestPatchHandler_QuickDeploy_PatchNotFound returns 404 when patch missing.
func TestPatchHandler_QuickDeploy_PatchNotFound(t *testing.T) {
	h := v1.NewPatchHandler(&fakePatchQuerier{
		getErr: pgx.ErrNoRows,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patches/00000000-0000-0000-0000-000000000099/deploy", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.QuickDeploy(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- filterEndpointsForDeploy Tests ---

func TestFilterEndpointsForDeploy_EndpointIDs(t *testing.T) {
	ep1 := v1.MakeEndpoint("00000000-0000-0000-0000-000000000001", "linux", "active")
	ep2 := v1.MakeEndpoint("00000000-0000-0000-0000-000000000002", "windows", "active")
	ep3 := v1.MakeEndpoint("00000000-0000-0000-0000-000000000003", "linux", "active")
	decommissioned := v1.MakeEndpoint("00000000-0000-0000-0000-000000000004", "linux", "decommissioned")

	tests := []struct {
		name        string
		endpoints   []sqlcgen.Endpoint
		endpointIDs []string
		osFamily    string
		wantLen     int
		wantIDs     []string
	}{
		{
			name:        "filters to exact endpoint IDs",
			endpoints:   []sqlcgen.Endpoint{ep1, ep2, ep3},
			endpointIDs: []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000003"},
			wantLen:     2,
			wantIDs:     []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000003"},
		},
		{
			name:        "excludes decommissioned even if ID is listed",
			endpoints:   []sqlcgen.Endpoint{ep1, decommissioned},
			endpointIDs: []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000004"},
			wantLen:     1,
			wantIDs:     []string{"00000000-0000-0000-0000-000000000001"},
		},
		{
			name:        "returns empty when no IDs match",
			endpoints:   []sqlcgen.Endpoint{ep1, ep2},
			endpointIDs: []string{"00000000-0000-0000-0000-000000000099"},
			wantLen:     0,
		},
		{
			name:      "falls back to os family filter when endpointIDs empty",
			endpoints: []sqlcgen.Endpoint{ep1, ep2, ep3},
			osFamily:  "linux",
			wantLen:   2,
			wantIDs:   []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000003"},
		},
		{
			name:      "os family all includes all non-decommissioned",
			endpoints: []sqlcgen.Endpoint{ep1, ep2, decommissioned},
			osFamily:  "all",
			wantLen:   2,
		},
		{
			name:      "os family windows filters correctly",
			endpoints: []sqlcgen.Endpoint{ep1, ep2, ep3},
			osFamily:  "windows",
			wantLen:   1,
			wantIDs:   []string{"00000000-0000-0000-0000-000000000002"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v1.FilterEndpointsForDeploy(tt.endpoints, tt.endpointIDs, tt.osFamily)
			require.Len(t, got, tt.wantLen)
			for i, wantID := range tt.wantIDs {
				var gotID pgtype.UUID
				_ = gotID.Scan(wantID)
				assert.Equal(t, gotID, got[i].ID, "endpoint %d ID mismatch", i)
			}
		})
	}
}
