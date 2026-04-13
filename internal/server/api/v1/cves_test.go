package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCVEQuerier mocks CVEQuerier.
type fakeCVEQuerier struct {
	listResult          []sqlcgen.ListCVEsFilteredRow
	listErr             error
	countResult         int64
	countErr            error
	getResult           sqlcgen.CVE
	getErr              error
	affectedResult      []sqlcgen.ListAffectedEndpointsForCVERow
	affectedErr         error
	affectedCountResult int64
	affectedCountErr    error
	patchesResult       []sqlcgen.ListPatchesForCVEDetailRow
	patchesErr          error
	countBySeverityRows []sqlcgen.CountCVEsBySeverityRow
	countBySeverityErr  error
	countKEV            int32
	countKEVErr         error
	relatedCVEsResult   []sqlcgen.ListRelatedCVEsForCVERow
	relatedCVEsErr      error
}

func (f *fakeCVEQuerier) ListCVEsFiltered(_ context.Context, _ sqlcgen.ListCVEsFilteredParams) ([]sqlcgen.ListCVEsFilteredRow, error) {
	return f.listResult, f.listErr
}
func (f *fakeCVEQuerier) CountCVEsFiltered(_ context.Context, _ sqlcgen.CountCVEsFilteredParams) (int64, error) {
	return f.countResult, f.countErr
}
func (f *fakeCVEQuerier) GetCVEByID(_ context.Context, _ sqlcgen.GetCVEByIDParams) (sqlcgen.CVE, error) {
	return f.getResult, f.getErr
}
func (f *fakeCVEQuerier) ListAffectedEndpointsForCVE(_ context.Context, _ sqlcgen.ListAffectedEndpointsForCVEParams) ([]sqlcgen.ListAffectedEndpointsForCVERow, error) {
	return f.affectedResult, f.affectedErr
}
func (f *fakeCVEQuerier) CountAffectedEndpointsForCVE(_ context.Context, _ sqlcgen.CountAffectedEndpointsForCVEParams) (int64, error) {
	return f.affectedCountResult, f.affectedCountErr
}
func (f *fakeCVEQuerier) ListPatchesForCVEDetail(_ context.Context, _ sqlcgen.ListPatchesForCVEDetailParams) ([]sqlcgen.ListPatchesForCVEDetailRow, error) {
	return f.patchesResult, f.patchesErr
}
func (f *fakeCVEQuerier) CountCVEsBySeverity(_ context.Context, _ pgtype.UUID) ([]sqlcgen.CountCVEsBySeverityRow, error) {
	return f.countBySeverityRows, f.countBySeverityErr
}
func (f *fakeCVEQuerier) CountCVEsKEV(_ context.Context, _ pgtype.UUID) (int32, error) {
	return f.countKEV, f.countKEVErr
}
func (f *fakeCVEQuerier) CountCVEsExploit(_ context.Context, _ pgtype.UUID) (int32, error) {
	return 0, nil
}
func (f *fakeCVEQuerier) ListRelatedCVEsForCVE(_ context.Context, _ sqlcgen.ListRelatedCVEsForCVEParams) ([]sqlcgen.ListRelatedCVEsForCVERow, error) {
	return f.relatedCVEsResult, f.relatedCVEsErr
}

func validCVERow() sqlcgen.ListCVEsFilteredRow {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.ListCVEsFilteredRow{
		ID:                    id,
		TenantID:              tid,
		CveID:                 "CVE-2024-0001",
		Severity:              "critical",
		ExploitAvailable:      true,
		AffectedEndpointCount: 3,
		PatchAvailable:        true,
	}
}

func validCVE() sqlcgen.CVE {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.CVE{
		ID:                 id,
		TenantID:           tid,
		CveID:              "CVE-2024-0001",
		Severity:           "critical",
		Description:        pgtype.Text{String: "A critical vulnerability", Valid: true},
		ExploitAvailable:   true,
		AttackVector:       pgtype.Text{String: "NETWORK", Valid: true},
		CweID:              pgtype.Text{String: "CWE-79", Valid: true},
		Source:             "NVD",
		ExternalReferences: []byte(`[{"url":"https://nvd.nist.gov/vuln/detail/CVE-2024-0001","source":"NVD"}]`),
	}
}

// --- List Tests ---

func TestCVEHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *fakeCVEQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns empty list",
			querier: &fakeCVEQuerier{
				listResult:  []sqlcgen.ListCVEsFilteredRow{},
				countResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "returns CVEs",
			query: "?limit=10",
			querier: &fakeCVEQuerier{
				listResult:  []sqlcgen.ListCVEsFilteredRow{validCVERow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "invalid cursor returns 400",
			query:      "?cursor=bad-cursor",
			querier:    &fakeCVEQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
		{
			name: "store error returns 500",
			querier: &fakeCVEQuerier{
				listErr: fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
		{
			name:  "filter by severity",
			query: "?severity=critical",
			querier: &fakeCVEQuerier{
				listResult:  []sqlcgen.ListCVEsFilteredRow{validCVERow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by cisa_kev",
			query: "?cisa_kev=true",
			querier: &fakeCVEQuerier{
				listResult:  []sqlcgen.ListCVEsFilteredRow{validCVERow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "filter by exploit_available",
			query: "?exploit_available=true",
			querier: &fakeCVEQuerier{
				listResult:  []sqlcgen.ListCVEsFilteredRow{validCVERow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "count error returns 500",
			querier: &fakeCVEQuerier{
				listResult: []sqlcgen.ListCVEsFilteredRow{},
				countErr:   fmt.Errorf("count failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewCVEHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cves"+tt.query, nil)
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

// --- Get Tests ---

func TestCVEHandler_Get(t *testing.T) {
	var patchID pgtype.UUID
	_ = patchID.Scan("00000000-0000-0000-0000-000000000088")

	tests := []struct {
		name       string
		id         string
		querier    *fakeCVEQuerier
		wantStatus int
	}{
		{
			name: "valid ID returns 200",
			id:   "00000000-0000-0000-0000-000000000099",
			querier: &fakeCVEQuerier{
				getResult: validCVE(),
				affectedResult: []sqlcgen.ListAffectedEndpointsForCVERow{{
					ID:           patchID,
					Hostname:     "host-1",
					OsFamily:     "linux",
					OsVersion:    "22.04",
					IpAddress:    pgtype.Text{String: "10.0.0.1", Valid: true},
					Status:       "vulnerable",
					AgentVersion: pgtype.Text{String: "1.2.0", Valid: true},
					GroupNames:   []byte("web-servers"),
				}},
				affectedCountResult: 15,
				patchesResult: []sqlcgen.ListPatchesForCVEDetailRow{{
					ID:               patchID,
					Name:             "openssl-fix",
					Version:          "1.0.0",
					Severity:         "critical",
					OsFamily:         "linux",
					EndpointsCovered: 10,
					EndpointsPatched: 3,
				}},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeCVEQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000099",
			querier:    &fakeCVEQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "db error returns 500",
			id:         "00000000-0000-0000-0000-000000000060",
			querier:    &fakeCVEQuerier{getErr: fmt.Errorf("db error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewCVEHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cves/"+tt.id, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, "CVE-2024-0001", body["cve_id"])
				assert.Equal(t, "NETWORK", body["attack_vector"])
				assert.Equal(t, "CWE-79", body["cwe_id"])
				assert.Equal(t, "NVD", body["source"])

				extRefs, ok := body["external_references"].([]any)
				require.True(t, ok)
				assert.Len(t, extRefs, 1)

				ae, ok := body["affected_endpoints"].(map[string]any)
				require.True(t, ok, "expected affected_endpoints object")
				assert.Equal(t, float64(15), ae["count"])
				aeItems, ok := ae["items"].([]any)
				require.True(t, ok)
				assert.Len(t, aeItems, 1)
				assert.Equal(t, true, ae["has_more"])

				firstEndpoint, ok := aeItems[0].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "linux", firstEndpoint["os_family"])
				assert.Equal(t, "10.0.0.1", firstEndpoint["ip_address"])

				patches, ok := body["patches"].([]any)
				require.True(t, ok)
				assert.Len(t, patches, 1)
				firstPatch, ok := patches[0].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "linux", firstPatch["os_family"])
				assert.Equal(t, float64(10), firstPatch["endpoints_covered"])
			}
		})
	}
}

// --- Summary Tests ---

func TestCVEHandler_Summary(t *testing.T) {
	tests := []struct {
		name       string
		querier    *fakeCVEQuerier
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name: "returns severity counts and KEV count",
			querier: &fakeCVEQuerier{
				countBySeverityRows: []sqlcgen.CountCVEsBySeverityRow{
					{Severity: "critical", Count: 5},
					{Severity: "high", Count: 10},
					{Severity: "medium", Count: 20},
				},
				countKEV: 3,
			},
			wantStatus: http.StatusOK,
			wantBody: map[string]any{
				"total":     float64(35),
				"kev_count": float64(3),
			},
		},
		{
			name: "empty tenant returns zeroes",
			querier: &fakeCVEQuerier{
				countBySeverityRows: []sqlcgen.CountCVEsBySeverityRow{},
				countKEV:            0,
			},
			wantStatus: http.StatusOK,
			wantBody: map[string]any{
				"total":     float64(0),
				"kev_count": float64(0),
			},
		},
		{
			name: "severity count DB error",
			querier: &fakeCVEQuerier{
				countBySeverityErr: fmt.Errorf("db error"),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "KEV count DB error",
			querier: &fakeCVEQuerier{
				countBySeverityRows: []sqlcgen.CountCVEsBySeverityRow{},
				countKEVErr:         fmt.Errorf("db error"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewCVEHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cves/summary", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.Summary(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantBody != nil {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tt.wantBody["total"], body["total"])
				assert.Equal(t, tt.wantBody["kev_count"], body["kev_count"])
			}
		})
	}
}
