package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDashboardQuerier mocks DashboardQuerier.
type fakeDashboardQuerier struct {
	result sqlcgen.GetDashboardSummaryRow
	err    error

	activeDeployments []sqlcgen.GetActiveDeploymentsRow
	activeDeployErr   error
	failedTrend       []sqlcgen.GetFailedDeploymentTrend7dRow
	failedTrendErr    error
	runningWorkflows  []sqlcgen.GetRunningWorkflowsRow
	runningWorkErr    error
	hubSyncResult     sqlcgen.HubSyncState
	hubSyncErr        error
	activityResult    []sqlcgen.GetDashboardActivityRow
	activityErr       error
	highestCVE        sqlcgen.GetHighestUnpatchedCVERow
	highestCVEErr     error
	cveByUUID         sqlcgen.GetCVEByUUIDRow
	cveByUUIDErr      error
	blastGroups       []sqlcgen.GetBlastRadiusGroupsRow
	blastGroupsErr    error
	endpointsRisk     []sqlcgen.GetTopEndpointsByRiskRow
	endpointsRiskErr  error
	slaDeadlines      []sqlcgen.ComplianceEvaluation
	slaDeadlinesErr   error
}

func (f *fakeDashboardQuerier) GetDashboardSummary(_ context.Context, _ pgtype.UUID) (sqlcgen.GetDashboardSummaryRow, error) {
	return f.result, f.err
}

func (f *fakeDashboardQuerier) GetActiveDeployments(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetActiveDeploymentsRow, error) {
	return f.activeDeployments, f.activeDeployErr
}

func (f *fakeDashboardQuerier) GetFailedDeploymentTrend7d(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetFailedDeploymentTrend7dRow, error) {
	return f.failedTrend, f.failedTrendErr
}

func (f *fakeDashboardQuerier) GetRunningWorkflows(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetRunningWorkflowsRow, error) {
	return f.runningWorkflows, f.runningWorkErr
}

func (f *fakeDashboardQuerier) GetHubSyncState(_ context.Context, _ pgtype.UUID) (sqlcgen.HubSyncState, error) {
	return f.hubSyncResult, f.hubSyncErr
}

func (f *fakeDashboardQuerier) GetDashboardActivity(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetDashboardActivityRow, error) {
	return f.activityResult, f.activityErr
}

func (f *fakeDashboardQuerier) GetHighestUnpatchedCVE(_ context.Context, _ pgtype.UUID) (sqlcgen.GetHighestUnpatchedCVERow, error) {
	return f.highestCVE, f.highestCVEErr
}

func (f *fakeDashboardQuerier) GetCVEByUUID(_ context.Context, _ sqlcgen.GetCVEByUUIDParams) (sqlcgen.GetCVEByUUIDRow, error) {
	return f.cveByUUID, f.cveByUUIDErr
}

func (f *fakeDashboardQuerier) GetBlastRadiusGroups(_ context.Context, _ sqlcgen.GetBlastRadiusGroupsParams) ([]sqlcgen.GetBlastRadiusGroupsRow, error) {
	return f.blastGroups, f.blastGroupsErr
}

func (f *fakeDashboardQuerier) GetTopEndpointsByRisk(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetTopEndpointsByRiskRow, error) {
	return f.endpointsRisk, f.endpointsRiskErr
}

func (f *fakeDashboardQuerier) GetExposureWindows(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetExposureWindowsRow, error) {
	return nil, nil
}

func (f *fakeDashboardQuerier) GetMTTR(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetMTTRRow, error) {
	return nil, nil
}

func (f *fakeDashboardQuerier) GetAttackPaths(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetAttackPathsRow, error) {
	return nil, nil
}

func (f *fakeDashboardQuerier) GetPolicyDrift(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetPolicyDriftRow, error) {
	return nil, nil
}

func (f *fakeDashboardQuerier) GetSLAForecast(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetSLAForecastRow, error) {
	return nil, nil
}

func (f *fakeDashboardQuerier) GetSLADeadlines(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetSLADeadlinesRow, error) {
	return nil, nil
}

func (f *fakeDashboardQuerier) GetSLATiers(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetSLATiersRow, error) {
	return nil, nil
}

func (f *fakeDashboardQuerier) GetRiskProjectionData(_ context.Context, _ pgtype.UUID) (sqlcgen.GetRiskProjectionDataRow, error) {
	return sqlcgen.GetRiskProjectionDataRow{}, nil
}

func TestDashboardHandler_Summary(t *testing.T) {
	tests := []struct {
		name       string
		querier    *fakeDashboardQuerier
		wantStatus int
		wantCheck  func(t *testing.T, body map[string]any)
	}{
		{
			name: "returns summary",
			querier: &fakeDashboardQuerier{
				result: sqlcgen.GetDashboardSummaryRow{
					EndpointsTotal:            150,
					EndpointsOnline:           142,
					PatchesAvailable:          45,
					PatchesCritical:           12,
					PatchesHigh:               18,
					CvesTotal:                 50,
					CvesUnpatched:             30,
					CvesCritical:              8,
					DeploymentsRunning:        3,
					DeploymentsCompletedToday: 7,
				},
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				t.Helper()
				assert.Equal(t, float64(150), body["total_endpoints"])
				assert.Equal(t, float64(142), body["active_endpoints"])
				assert.Equal(t, float64(45), body["total_patches"])
				assert.Equal(t, float64(12), body["critical_patches"])
				assert.Equal(t, float64(50), body["total_cves"])     // cves_total
				assert.Equal(t, float64(8), body["critical_cves"])   // cves_critical
				assert.Equal(t, float64(30), body["unpatched_cves"]) // cves_unpatched
				assert.Equal(t, float64(3), body["pending_deployments"])
				assert.Equal(t, float64(-1), body["compliance_rate"]) // -1 = N/A (no frameworks enabled)
			},
		},
		{
			name: "store error returns 500",
			querier: &fakeDashboardQuerier{
				err: fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewDashboardHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.Summary(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantCheck != nil {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				tt.wantCheck(t, body)
			}
		})
	}
}

func TestDashboardHandler_Summary_Enriched(t *testing.T) {
	depID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	wfID := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}

	tests := []struct {
		name       string
		querier    *fakeDashboardQuerier
		wantStatus int
		wantCheck  func(t *testing.T, body map[string]any)
	}{
		{
			name: "returns enriched summary with all new fields",
			querier: &fakeDashboardQuerier{
				result: sqlcgen.GetDashboardSummaryRow{
					EndpointsTotal:         100,
					EndpointsOnline:        90,
					EndpointsDegraded:      5,
					PatchesAvailable:       20,
					PatchesCritical:        4,
					PatchesHigh:            6,
					PatchesMedium:          7,
					PatchesLow:             3,
					CvesTotal:              25,
					CvesUnpatched:          10,
					CvesCritical:           2,
					DeploymentsRunning:     2,
					FailedDeploymentsCount: 3,
					OverdueSlaCount:        1,
					CompliancePct:          75.5,
				},
				activeDeployments: []sqlcgen.GetActiveDeploymentsRow{
					{ID: depID, PolicyName: pgtype.Text{String: "Patch All", Valid: true}, Status: "running", ProgressPct: 60},
					{ID: pgtype.UUID{Bytes: [16]byte{9}, Valid: true}, PolicyName: pgtype.Text{Valid: false}, Status: "running", ProgressPct: 30},
				},
				failedTrend: []sqlcgen.GetFailedDeploymentTrend7dRow{
					{Count: 0},
					{Count: 1},
					{Count: 2},
					{Count: 0},
					{Count: 3},
					{Count: 1},
					{Count: 0},
				},
				runningWorkflows: []sqlcgen.GetRunningWorkflowsRow{
					{ID: wfID, Name: "Auto Patch", CurrentStage: "stage-1"},
				},
				hubSyncResult: sqlcgen.HubSyncState{
					Status: "synced",
					HubUrl: "https://hub.example.com",
				},
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				t.Helper()
				assert.Equal(t, float64(5), body["endpoints_degraded"])
				assert.Equal(t, float64(6), body["patches_high"])
				assert.Equal(t, float64(7), body["patches_medium"])
				assert.Equal(t, float64(3), body["patches_low"])
				assert.Equal(t, float64(3), body["failed_deployments_count"])
				assert.Equal(t, float64(1), body["overdue_sla_count"])
				assert.Equal(t, "synced", body["hub_sync_status"])

				// active_deployments should be an array with two items (one with policy, one without)
				activeDeps, ok := body["active_deployments"].([]any)
				require.True(t, ok, "active_deployments should be array")
				assert.Len(t, activeDeps, 2)
				dep, ok := activeDeps[0].(map[string]any)
				require.True(t, ok, "active_deployments[0] should be map")
				assert.Equal(t, "Patch All", dep["name"])
				assert.Equal(t, "running", dep["status"])
				assert.Equal(t, float64(60), dep["progress_pct"])
				// Second deployment has NULL policy (LEFT JOIN) — name should be empty string
				dep2, ok := activeDeps[1].(map[string]any)
				require.True(t, ok, "active_deployments[1] should be map")
				assert.Equal(t, "", dep2["name"])
				assert.Equal(t, "running", dep2["status"])

				// failed_trend_7d should be an array of 7 ints
				trend, ok := body["failed_trend_7d"].([]any)
				require.True(t, ok, "failed_trend_7d should be array")
				assert.Len(t, trend, 7)

				// workflows_running should be an array with one item
				workflows, ok := body["workflows_running"].([]any)
				require.True(t, ok, "workflows_running should be array")
				assert.Len(t, workflows, 1)
				wf, ok := workflows[0].(map[string]any)
				require.True(t, ok, "workflows_running[0] should be map")
				assert.Equal(t, "Auto Patch", wf["name"])
			},
		},
		{
			name: "active deployments query error still returns summary with empty arrays",
			querier: &fakeDashboardQuerier{
				result: sqlcgen.GetDashboardSummaryRow{
					EndpointsTotal:  10,
					EndpointsOnline: 10,
				},
				activeDeployErr: fmt.Errorf("db timeout"),
				failedTrendErr:  fmt.Errorf("db timeout"),
				runningWorkErr:  fmt.Errorf("db timeout"),
				hubSyncErr:      fmt.Errorf("not found"),
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				t.Helper()
				// Should still return 200 with empty arrays
				activeDeps, ok := body["active_deployments"].([]any)
				require.True(t, ok, "active_deployments should be array")
				assert.Empty(t, activeDeps)

				trend, ok := body["failed_trend_7d"].([]any)
				require.True(t, ok, "failed_trend_7d should be array")
				assert.Empty(t, trend)

				workflows, ok := body["workflows_running"].([]any)
				require.True(t, ok, "workflows_running should be array")
				assert.Empty(t, workflows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewDashboardHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.Summary(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantCheck != nil {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				tt.wantCheck(t, body)
			}
		})
	}
}

func TestDashboardHandler_Activity(t *testing.T) {
	depID := pgtype.UUID{Bytes: [16]byte{3}, Valid: true}
	ts := pgtype.Timestamptz{Time: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), Valid: true}

	tests := []struct {
		name       string
		querier    *fakeDashboardQuerier
		wantStatus int
		wantCheck  func(t *testing.T, body map[string]any)
	}{
		{
			name: "returns activity items",
			querier: &fakeDashboardQuerier{
				activityResult: []sqlcgen.GetDashboardActivityRow{
					{
						ID:             depID,
						Type:           "deployment",
						Title:          pgtype.Text{String: "Patch All Servers", Valid: true},
						Status:         "running",
						TotalTargets:   10,
						CompletedCount: 6,
						FailedCount:    0,
						Timestamp:      ts,
					},
				},
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				t.Helper()
				items, ok := body["items"].([]any)
				require.True(t, ok, "items should be array")
				require.Len(t, items, 1)
				item, ok := items[0].(map[string]any)
				require.True(t, ok, "items[0] should be map")
				assert.Equal(t, "deployment", item["type"])
				assert.Equal(t, "Patch All Servers", item["title"])
				assert.Equal(t, "running", item["status"])
				assert.Equal(t, "6/10 endpoints", item["meta"])
				assert.Equal(t, "2026-01-15T10:00:00Z", item["timestamp"])
				detail, ok := item["detail"].(map[string]any)
				require.True(t, ok, "detail should be present for running deployment")
				assert.Equal(t, float64(60), detail["progress_pct"])
				assert.Equal(t, float64(10), detail["total"])
				assert.Equal(t, float64(6), detail["completed"])
			},
		},
		{
			name: "store error returns 500",
			querier: &fakeDashboardQuerier{
				activityErr: fmt.Errorf("database error"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewDashboardHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/activity", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.Activity(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantCheck != nil {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				tt.wantCheck(t, body)
			}
		})
	}
}

func TestDashboardHandler_BlastRadius(t *testing.T) {
	cveID := pgtype.UUID{Bytes: [16]byte{4}, Valid: true}

	tests := []struct {
		name       string
		querier    *fakeDashboardQuerier
		url        string
		wantStatus int
		wantCheck  func(t *testing.T, body map[string]any)
	}{
		{
			name: "returns blast radius with highest CVE",
			querier: &fakeDashboardQuerier{
				highestCVE: sqlcgen.GetHighestUnpatchedCVERow{
					ID:            cveID,
					CveID:         "CVE-2024-1234",
					CvssScore:     9.8,
					AffectedCount: 42,
				},
				blastGroups: []sqlcgen.GetBlastRadiusGroupsRow{
					{Name: "env=production", Os: "linux", HostCount: 30},
					{Name: "", Os: "windows", HostCount: 12},
				},
			},
			url:        "/api/v1/dashboard/blast-radius",
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				t.Helper()
				cve, ok := body["cve"].(map[string]any)
				require.True(t, ok, "cve should be present")
				assert.Equal(t, "CVE-2024-1234", cve["cve_id"])
				assert.Equal(t, float64(9.8), cve["cvss"])
				assert.Equal(t, float64(42), cve["affected_count"])
				groups, ok := body["groups"].([]any)
				require.True(t, ok, "groups should be array")
				assert.Len(t, groups, 2)
				g0, ok := groups[0].(map[string]any)
				require.True(t, ok, "groups[0] should be map")
				assert.Equal(t, "env=production", g0["name"])
				g1, ok := groups[1].(map[string]any)
				require.True(t, ok, "groups[1] should be map")
				// Empty tag falls back to OS family in the handler.
				assert.Equal(t, "windows", g1["name"])
			},
		},
		{
			name: "no CVE found returns empty response",
			querier: &fakeDashboardQuerier{
				highestCVEErr: pgx.ErrNoRows,
			},
			url:        "/api/v1/dashboard/blast-radius",
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				t.Helper()
				assert.Nil(t, body["cve"])
				groups, ok := body["groups"].([]any)
				require.True(t, ok)
				assert.Empty(t, groups)
			},
		},
		{
			name:       "invalid cve_id param returns 400",
			querier:    &fakeDashboardQuerier{},
			url:        "/api/v1/dashboard/blast-radius?cve_id=not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewDashboardHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.BlastRadius(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantCheck != nil {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				tt.wantCheck(t, body)
			}
		})
	}
}

func TestDashboardHandler_SLADeadlines(t *testing.T) {
	ts := pgtype.Timestamptz{Time: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC), Valid: true}
	evalTs := pgtype.Timestamptz{Time: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC), Valid: true}

	tests := []struct {
		name       string
		querier    *fakeDashboardQuerier
		wantStatus int
		wantCheck  func(t *testing.T, body []any)
	}{
		{
			name: "returns SLA deadline items",
			querier: &fakeDashboardQuerier{
				slaDeadlines: []sqlcgen.ComplianceEvaluation{
					{
						FrameworkID:   "CIS",
						State:         "AT_RISK",
						SlaDeadlineAt: ts,
						EvaluatedAt:   evalTs,
					},
				},
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body []any) {
				t.Helper()
				require.Len(t, body, 1)
				item, ok := body[0].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "CIS", item["framework_id"])
				assert.Equal(t, "CIS", item["framework_name"])
				assert.Equal(t, "AT_RISK", item["state"])
				assert.Equal(t, "2026-04-20T12:00:00Z", item["sla_deadline_at"])
				assert.Equal(t, "2026-04-10T12:00:00Z", item["evaluated_at"])
			},
		},
		{
			name:       "empty result returns empty array",
			querier:    &fakeDashboardQuerier{},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body []any) {
				t.Helper()
				assert.Empty(t, body)
			},
		},
		{
			name: "store error returns 500",
			querier: &fakeDashboardQuerier{
				slaDeadlinesErr: fmt.Errorf("db error"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewDashboardHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/sla-deadlines", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.SLADeadlines(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantCheck != nil {
				var body []any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				tt.wantCheck(t, body)
			}
		})
	}
}

func TestDashboardHandler_EndpointsRisk(t *testing.T) {
	tests := []struct {
		name       string
		querier    *fakeDashboardQuerier
		wantStatus int
		wantCheck  func(t *testing.T, body []any)
	}{
		{
			name: "returns endpoints risk list",
			querier: &fakeDashboardQuerier{
				endpointsRisk: []sqlcgen.GetTopEndpointsByRiskRow{
					{Hostname: "server-01", CveCount: 15, RiskScore: 95},
					{Hostname: "server-02", CveCount: 8, RiskScore: 72},
				},
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body []any) {
				t.Helper()
				require.Len(t, body, 2)
				e0, ok := body[0].(map[string]any)
				require.True(t, ok, "body[0] should be map")
				assert.Equal(t, "server-01", e0["hostname"])
				assert.Equal(t, float64(15), e0["cve_count"])
				assert.Equal(t, float64(95), e0["risk_score"])
			},
		},
		{
			name: "store error returns 500",
			querier: &fakeDashboardQuerier{
				endpointsRiskErr: fmt.Errorf("db error"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewDashboardHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/endpoints-risk", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.EndpointsRisk(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantCheck != nil {
				var body []any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				tt.wantCheck(t, body)
			}
		})
	}
}
