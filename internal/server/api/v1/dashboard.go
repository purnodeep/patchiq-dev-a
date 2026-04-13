package v1

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// DashboardQuerier defines the sqlc queries needed by DashboardHandler.
type DashboardQuerier interface {
	GetDashboardSummary(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetDashboardSummaryRow, error)
	GetActiveDeployments(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetActiveDeploymentsRow, error)
	GetFailedDeploymentTrend7d(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetFailedDeploymentTrend7dRow, error)
	GetRunningWorkflows(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetRunningWorkflowsRow, error)
	GetHubSyncState(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.HubSyncState, error)
	GetDashboardActivity(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetDashboardActivityRow, error)
	GetHighestUnpatchedCVE(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetHighestUnpatchedCVERow, error)
	GetCVEByUUID(ctx context.Context, arg sqlcgen.GetCVEByUUIDParams) (sqlcgen.GetCVEByUUIDRow, error)
	GetBlastRadiusGroups(ctx context.Context, arg sqlcgen.GetBlastRadiusGroupsParams) ([]sqlcgen.GetBlastRadiusGroupsRow, error)
	GetTopEndpointsByRisk(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetTopEndpointsByRiskRow, error)
	GetExposureWindows(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetExposureWindowsRow, error)
	GetMTTR(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetMTTRRow, error)
	GetAttackPaths(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetAttackPathsRow, error)
	GetPolicyDrift(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetPolicyDriftRow, error)
	GetSLAForecast(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetSLAForecastRow, error)
	GetSLADeadlines(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetSLADeadlinesRow, error)
	GetSLATiers(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetSLATiersRow, error)
	GetRiskProjectionData(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetRiskProjectionDataRow, error)
}

// DashboardHandler serves dashboard REST API endpoints.
type DashboardHandler struct {
	q DashboardQuerier
}

// NewDashboardHandler creates a DashboardHandler.
func NewDashboardHandler(q DashboardQuerier) *DashboardHandler {
	if q == nil {
		panic("dashboard: NewDashboardHandler called with nil querier")
	}
	return &DashboardHandler{q: q}
}

type activeDeploymentItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	ProgressPct int32  `json:"progress_pct"`
}

type runningWorkflowItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CurrentStage string `json:"current_stage"`
}

// dashboardSummaryResponse is the JSON body for GET /dashboard/summary.
// Field names must match the OpenAPI spec (DashboardSummary schema).
type dashboardSummaryResponse struct {
	TotalEndpoints         int32                  `json:"total_endpoints"`
	ActiveEndpoints        int32                  `json:"active_endpoints"`
	EndpointsDegraded      int32                  `json:"endpoints_degraded"`
	TotalPatches           int32                  `json:"total_patches"`
	CriticalPatches        int32                  `json:"critical_patches"`
	PatchesHigh            int32                  `json:"patches_high"`
	PatchesMedium          int32                  `json:"patches_medium"`
	PatchesLow             int32                  `json:"patches_low"`
	TotalCVEs              int32                  `json:"total_cves"`
	CriticalCVEs           int32                  `json:"critical_cves"`
	UnpatchedCVEs          int32                  `json:"unpatched_cves"`
	PendingDeployments     int32                  `json:"pending_deployments"`
	ComplianceRate         float64                `json:"compliance_rate"`
	FrameworkCount         int32                  `json:"framework_count"`
	ActiveDeployments      []activeDeploymentItem `json:"active_deployments"`
	OverdueSLACount        int32                  `json:"overdue_sla_count"`
	FailedDeploymentsCount int32                  `json:"failed_deployments_count"`
	FailedTrend7d          []int32                `json:"failed_trend_7d"`
	WorkflowsRunningCount  int32                  `json:"workflows_running_count"`
	WorkflowsRunning       []runningWorkflowItem  `json:"workflows_running"`
	HubSyncStatus          string                 `json:"hub_sync_status"`
	HubLastSyncAt          *string                `json:"hub_last_sync_at"`
	HubURL                 string                 `json:"hub_url"`
}

// dashboardActivityResponse is the JSON body for GET /dashboard/activity.
type dashboardActivityResponse struct {
	Items []activityItem `json:"items"`
}

type activityItem struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Status    string          `json:"status"`
	Meta      string          `json:"meta"`
	Detail    *activityDetail `json:"detail,omitempty"`
	Timestamp string          `json:"timestamp"`
}

type activityDetail struct {
	DeploymentID string `json:"deployment_id,omitempty"`
	ProgressPct  int32  `json:"progress_pct"`
	Total        int32  `json:"total"`
	Completed    int32  `json:"completed"`
}

// blastRadiusResponse is the JSON body for GET /dashboard/blast-radius.
type blastRadiusResponse struct {
	CVE    *blastRadiusCVE    `json:"cve"`
	Groups []blastRadiusGroup `json:"groups"`
}

type blastRadiusCVE struct {
	ID            string  `json:"id"`
	CVEID         string  `json:"cve_id"`
	CVSS          float64 `json:"cvss"`
	AffectedCount int32   `json:"affected_count"`
}

type blastRadiusGroup struct {
	Name      string `json:"name"`
	OS        string `json:"os"`
	HostCount int32  `json:"host_count"`
}

// endpointsRiskItem is one entry in GET /dashboard/endpoints-risk.
type endpointsRiskItem struct {
	Hostname  string `json:"hostname"`
	CVECount  int32  `json:"cve_count"`
	RiskScore int32  `json:"risk_score"`
}

// Summary handles GET /api/v1/dashboard/summary.
func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	row, err := h.q.GetDashboardSummary(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get dashboard summary", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get dashboard summary")
		return
	}

	// Compliance: use SQL-computed value. Return -1 when not meaningful
	// (no frameworks enabled or no CVE data) so the UI can show "N/A".
	frameworkCount := int32(row.FrameworksEnabled)
	complianceRate := row.CompliancePct
	if frameworkCount == 0 || row.CvesTotal == 0 {
		complianceRate = -1 // signals "not applicable" to frontend
	}

	// Non-critical queries: log warn on error, continue with empty defaults.
	activeDeps, err := h.q.GetActiveDeployments(ctx, tid)
	if err != nil {
		slog.WarnContext(ctx, "get active deployments", "tenant_id", tenantID, "error", err)
		activeDeps = nil
	}

	failedTrend, err := h.q.GetFailedDeploymentTrend7d(ctx, tid)
	if err != nil {
		slog.WarnContext(ctx, "get failed deployment trend", "tenant_id", tenantID, "error", err)
		failedTrend = nil
	}

	runningWFs, err := h.q.GetRunningWorkflows(ctx, tid)
	if err != nil {
		slog.WarnContext(ctx, "get running workflows", "tenant_id", tenantID, "error", err)
		runningWFs = nil
	}

	hubSync, hubSyncErr := h.q.GetHubSyncState(ctx, tid)
	if hubSyncErr != nil {
		slog.WarnContext(ctx, "get hub sync state", "tenant_id", tenantID, "error", hubSyncErr)
	}

	// Map active deployments.
	activeDepItems := make([]activeDeploymentItem, 0, len(activeDeps))
	for _, d := range activeDeps {
		activeDepItems = append(activeDepItems, activeDeploymentItem{
			ID:          uuidToString(d.ID),
			Name:        d.PolicyName.String,
			Status:      d.Status,
			ProgressPct: d.ProgressPct,
		})
	}

	// Map failed trend to int slice.
	trendCounts := make([]int32, 0, len(failedTrend))
	for _, t := range failedTrend {
		trendCounts = append(trendCounts, t.Count)
	}

	// Map running workflows.
	wfItems := make([]runningWorkflowItem, 0, len(runningWFs))
	for _, wf := range runningWFs {
		wfItems = append(wfItems, runningWorkflowItem{
			ID:           uuidToString(wf.ID),
			Name:         wf.Name,
			CurrentStage: fmt.Sprintf("%v", wf.CurrentStage),
		})
	}

	// Hub sync fields.
	hubSyncStatus := ""
	hubURL := ""
	var hubLastSyncAt *string
	if hubSyncErr == nil {
		hubSyncStatus = hubSync.Status
		hubURL = hubSync.HubUrl
		if hubSync.LastSyncAt.Valid {
			s := hubSync.LastSyncAt.Time.UTC().Format(time.RFC3339)
			hubLastSyncAt = &s
		}
	}

	resp := dashboardSummaryResponse{
		TotalEndpoints:         row.EndpointsTotal,
		ActiveEndpoints:        row.EndpointsOnline,
		EndpointsDegraded:      row.EndpointsDegraded,
		TotalPatches:           row.PatchesAvailable,
		CriticalPatches:        row.PatchesCritical,
		PatchesHigh:            row.PatchesHigh,
		PatchesMedium:          row.PatchesMedium,
		PatchesLow:             row.PatchesLow,
		TotalCVEs:              row.CvesTotal,
		CriticalCVEs:           row.CvesCritical,
		UnpatchedCVEs:          row.CvesUnpatched,
		PendingDeployments:     row.DeploymentsRunning,
		ComplianceRate:         complianceRate,
		FrameworkCount:         frameworkCount,
		ActiveDeployments:      activeDepItems,
		OverdueSLACount:        row.OverdueSlaCount,
		FailedDeploymentsCount: row.FailedDeploymentsCount,
		FailedTrend7d:          trendCounts,
		WorkflowsRunningCount:  int32(len(wfItems)),
		WorkflowsRunning:       wfItems,
		HubSyncStatus:          hubSyncStatus,
		HubLastSyncAt:          hubLastSyncAt,
		HubURL:                 hubURL,
	}

	WriteJSON(w, http.StatusOK, resp)
}

// Activity handles GET /api/v1/dashboard/activity.
func (h *DashboardHandler) Activity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetDashboardActivity(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get dashboard activity", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get dashboard activity")
		return
	}

	items := make([]activityItem, 0, len(rows))
	for _, row := range rows {
		item := activityItem{
			ID:     uuidToString(row.ID),
			Type:   row.Type,
			Title:  row.Title.String,
			Status: row.Status,
		}

		deploymentID := uuidToString(row.ID)
		item.Detail = &activityDetail{DeploymentID: deploymentID}

		if row.TotalTargets > 0 {
			progressPct := int32(float64(row.CompletedCount) / float64(row.TotalTargets) * 100)
			item.Meta = fmt.Sprintf("%d/%d endpoints", row.CompletedCount, row.TotalTargets)
			switch row.Status {
			case "running", "completed", "failed":
				item.Detail = &activityDetail{
					DeploymentID: deploymentID,
					ProgressPct:  progressPct,
					Total:        row.TotalTargets,
					Completed:    row.CompletedCount,
				}
			}
		}

		if row.Timestamp.Valid {
			item.Timestamp = row.Timestamp.Time.UTC().Format(time.RFC3339)
		}

		items = append(items, item)
	}

	WriteJSON(w, http.StatusOK, dashboardActivityResponse{Items: items})
}

// BlastRadius handles GET /api/v1/dashboard/blast-radius.
func (h *DashboardHandler) BlastRadius(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	empty := blastRadiusResponse{CVE: nil, Groups: []blastRadiusGroup{}}

	var cveRow sqlcgen.GetHighestUnpatchedCVERow
	cveIDParam := r.URL.Query().Get("cve_id")
	if cveIDParam != "" {
		cveUUID, scanErr := scanUUID(cveIDParam)
		if scanErr != nil {
			slog.WarnContext(ctx, "blast radius: invalid cve_id parameter", "tenant_id", tenantID, "cve_id", cveIDParam, "error", scanErr)
			WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "invalid cve_id parameter")
			return
		}
		row, qErr := h.q.GetCVEByUUID(ctx, sqlcgen.GetCVEByUUIDParams{ID: cveUUID, TenantID: tid})
		if isNotFound(qErr) {
			WriteJSON(w, http.StatusOK, empty)
			return
		}
		if qErr != nil {
			slog.ErrorContext(ctx, "get CVE by UUID", "tenant_id", tenantID, "cve_id", cveIDParam, "error", qErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get CVE")
			return
		}
		cveRow = sqlcgen.GetHighestUnpatchedCVERow{
			ID:            row.ID,
			CveID:         row.CveID,
			CvssScore:     row.CvssScore,
			AffectedCount: row.AffectedCount,
		}
	} else {
		row, qErr := h.q.GetHighestUnpatchedCVE(ctx, tid)
		if isNotFound(qErr) {
			WriteJSON(w, http.StatusOK, empty)
			return
		}
		if qErr != nil {
			slog.ErrorContext(ctx, "get highest unpatched CVE", "tenant_id", tenantID, "error", qErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get highest CVE")
			return
		}
		cveRow = row
	}

	groupRows, err := h.q.GetBlastRadiusGroups(ctx, sqlcgen.GetBlastRadiusGroupsParams{
		CveID:    cveRow.ID,
		TenantID: tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "get blast radius groups", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get blast radius groups")
		return
	}

	groups := make([]blastRadiusGroup, 0, len(groupRows))
	for _, g := range groupRows {
		name := g.Os
		if g.Name != "" {
			name = g.Name
		}
		groups = append(groups, blastRadiusGroup{
			Name:      name,
			OS:        g.Os,
			HostCount: g.HostCount,
		})
	}

	WriteJSON(w, http.StatusOK, blastRadiusResponse{
		CVE: &blastRadiusCVE{
			ID:            uuidToString(cveRow.ID),
			CVEID:         cveRow.CveID,
			CVSS:          cveRow.CvssScore,
			AffectedCount: cveRow.AffectedCount,
		},
		Groups: groups,
	})
}

// EndpointsRisk handles GET /api/v1/dashboard/endpoints-risk.
func (h *DashboardHandler) EndpointsRisk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetTopEndpointsByRisk(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get top endpoints by risk", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get endpoints risk")
		return
	}

	items := make([]endpointsRiskItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, endpointsRiskItem{
			Hostname:  row.Hostname,
			CVECount:  row.CveCount,
			RiskScore: row.RiskScore,
		})
	}

	WriteJSON(w, http.StatusOK, items)
}

// --- Response types for new dashboard endpoints ---

type exposureWindowItem struct {
	ID            string  `json:"id"`
	CVEID         string  `json:"cve_id"`
	Severity      string  `json:"severity"`
	CVSS          float64 `json:"cvss"`
	AffectedCount int32   `json:"affected_count"`
	FirstSeen     string  `json:"first_seen"`
	PatchedAt     *string `json:"patched_at"`
}

type mttrItem struct {
	Week     string  `json:"week"`
	Severity string  `json:"severity"`
	AvgHours float64 `json:"avg_hours"`
}

type attackPathNode struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	OS            string `json:"os"`
	CriticalCount int32  `json:"critical_count"`
	HighCount     int32  `json:"high_count"`
	IsOnline      bool   `json:"is_online"`
}

type attackPathEdge struct {
	SourceID       string `json:"source_id"`
	TargetID       string `json:"target_id"`
	SharedCVECount int32  `json:"shared_cve_count"`
}

type attackPathsResponse struct {
	Nodes []attackPathNode `json:"nodes"`
	Edges []attackPathEdge `json:"edges"`
}

type driftItem struct {
	ID              string  `json:"id"`
	Hostname        string  `json:"hostname"`
	OS              string  `json:"os"`
	UnpatchedCount  int32   `json:"unpatched_count"`
	TotalCVECount   int32   `json:"total_cve_count"`
	DriftScore      int32   `json:"drift_score"`
	LastCompliantAt *string `json:"last_compliant_at"`
}

type slaForecastItem struct {
	ID               string `json:"id"`
	Hostname         string `json:"hostname"`
	Severity         string `json:"severity"`
	SLAWindowHours   int32  `json:"sla_window_hours"`
	RemainingSeconds int32  `json:"remaining_seconds"`
	OldestOpenSince  string `json:"oldest_open_since"`
}

type slaDeadlineItem struct {
	EndpointID       string `json:"endpoint_id"`
	Hostname         string `json:"hostname"`
	Severity         string `json:"severity"`
	PatchName        string `json:"patch_name"`
	RemainingSeconds int32  `json:"remaining_seconds"`
}

type slaTierItem struct {
	Severity string `json:"severity"`
	Total    int32  `json:"total"`
	Overdue  int32  `json:"overdue"`
}

type riskProjectionScenarios struct {
	DeployAll  []float64 `json:"deploy_all"`
	Trajectory []float64 `json:"trajectory"`
	DoNothing  []float64 `json:"do_nothing"`
}

type riskProjectionResponse struct {
	CurrentRiskPct float64                 `json:"current_risk_pct"`
	Scenarios      riskProjectionScenarios `json:"scenarios"`
}

// formatTimestamp safely converts an interface{} (from sqlc aggregate) to RFC3339 string.
// Returns empty string if value is nil or not a time.Time.
func formatTimestamp(v interface{}) string {
	if v == nil {
		return ""
	}
	if t, ok := v.(time.Time); ok {
		return t.UTC().Format(time.RFC3339)
	}
	return ""
}

// formatTimestampPtr returns a *string (nil if value is nil/not a time.Time).
func formatTimestampPtr(v interface{}) *string {
	if v == nil {
		return nil
	}
	if t, ok := v.(time.Time); ok {
		s := t.UTC().Format(time.RFC3339)
		return &s
	}
	return nil
}

// ExposureWindows handles GET /api/v1/dashboard/exposure-windows.
func (h *DashboardHandler) ExposureWindows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetExposureWindows(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get exposure windows", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get exposure windows")
		return
	}

	items := make([]exposureWindowItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, exposureWindowItem{
			ID:            uuidToString(row.ID),
			CVEID:         row.CveID,
			Severity:      row.Severity,
			CVSS:          row.CvssScore,
			AffectedCount: row.AffectedCount,
			FirstSeen:     formatTimestamp(row.FirstSeen),
			PatchedAt:     formatTimestampPtr(row.PatchedAt),
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// MTTR handles GET /api/v1/dashboard/mttr.
func (h *DashboardHandler) MTTR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetMTTR(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get mttr", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get MTTR data")
		return
	}

	items := make([]mttrItem, 0, len(rows))
	for _, row := range rows {
		week := ""
		if row.Week.Valid {
			week = row.Week.Time.Format("2006-01-02")
		}
		items = append(items, mttrItem{
			Week:     week,
			Severity: row.Severity,
			AvgHours: row.AvgHours,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// AttackPaths handles GET /api/v1/dashboard/attack-paths.
func (h *DashboardHandler) AttackPaths(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetAttackPaths(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get attack paths", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get attack paths")
		return
	}

	nodes := make([]attackPathNode, 0)
	edges := make([]attackPathEdge, 0)
	for _, row := range rows {
		switch row.RowType {
		case "node":
			nodes = append(nodes, attackPathNode{
				ID:            row.ID,
				Hostname:      row.Label,
				OS:            row.Os,
				CriticalCount: row.CriticalCount,
				HighCount:     row.HighCount,
				IsOnline:      row.IsOnline,
			})
		case "edge":
			edges = append(edges, attackPathEdge{
				SourceID:       row.SourceID,
				TargetID:       row.TargetID,
				SharedCVECount: row.SharedCount,
			})
		}
	}

	WriteJSON(w, http.StatusOK, attackPathsResponse{Nodes: nodes, Edges: edges})
}

// Drift handles GET /api/v1/dashboard/drift.
func (h *DashboardHandler) Drift(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetPolicyDrift(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get policy drift", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get policy drift data")
		return
	}

	items := make([]driftItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, driftItem{
			ID:              uuidToString(row.ID),
			Hostname:        row.Hostname,
			OS:              row.OsFamily,
			UnpatchedCount:  row.UnpatchedCount,
			TotalCVECount:   row.TotalCveCount,
			DriftScore:      row.DriftScore,
			LastCompliantAt: formatTimestampPtr(row.LastCompliantAt),
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// SLAForecast handles GET /api/v1/dashboard/sla-forecast.
func (h *DashboardHandler) SLAForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetSLAForecast(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get SLA forecast", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get SLA forecast")
		return
	}

	items := make([]slaForecastItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, slaForecastItem{
			ID:               uuidToString(row.ID),
			Hostname:         row.Hostname,
			Severity:         row.Severity,
			SLAWindowHours:   row.SlaWindowHours,
			RemainingSeconds: row.RemainingSeconds,
			OldestOpenSince:  formatTimestamp(row.OldestOpenSince),
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// SLADeadlines handles GET /api/v1/dashboard/sla-deadlines.
func (h *DashboardHandler) SLADeadlines(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetSLADeadlines(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get SLA deadlines", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get SLA deadlines")
		return
	}

	items := make([]slaDeadlineItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, slaDeadlineItem{
			EndpointID:       uuidToString(row.EndpointID),
			Hostname:         row.Hostname,
			Severity:         row.Severity,
			PatchName:        row.PatchName,
			RemainingSeconds: row.RemainingSeconds,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// SLATiers handles GET /api/v1/dashboard/sla-tiers.
func (h *DashboardHandler) SLATiers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.GetSLATiers(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get SLA tiers", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get SLA tiers")
		return
	}

	items := make([]slaTierItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, slaTierItem{
			Severity: row.Severity,
			Total:    row.Total,
			Overdue:  row.Overdue,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// RiskProjection handles GET /api/v1/dashboard/risk-projection.
func (h *DashboardHandler) RiskProjection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	row, err := h.q.GetRiskProjectionData(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get risk projection data", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get risk projection data")
		return
	}

	currentRisk := row.CurrentRiskPct
	avgPatches := row.AvgDailyPatches
	avgNewCVEs := row.AvgDailyNewCves
	totalAffected := float64(row.TotalAffected)
	totalCVEs := float64(row.TotalCves)

	const days = 31

	// Deploy All: risk drops to 0 linearly over 14 days.
	deployAll := make([]float64, days)
	for i := 0; i < days; i++ {
		if i >= 14 {
			deployAll[i] = 0
		} else {
			deployAll[i] = currentRisk * (1 - float64(i)/14.0)
		}
	}

	// Trajectory: risk changes based on net velocity (patches minus new CVEs).
	trajectory := make([]float64, days)
	affected := totalAffected
	for i := 0; i < days; i++ {
		if totalCVEs > 0 {
			trajectory[i] = affected * 100.0 / totalCVEs
		}
		affected = affected - avgPatches + avgNewCVEs
		if affected < 0 {
			affected = 0
		}
		totalCVEs += avgNewCVEs
	}

	// Do Nothing: risk increases based on new CVE inflow only.
	doNothing := make([]float64, days)
	affected = totalAffected
	totalForDoNothing := totalCVEs
	for i := 0; i < days; i++ {
		if totalForDoNothing > 0 {
			doNothing[i] = affected * 100.0 / totalForDoNothing
		}
		affected += avgNewCVEs
		totalForDoNothing += avgNewCVEs
	}

	WriteJSON(w, http.StatusOK, riskProjectionResponse{
		CurrentRiskPct: currentRisk,
		Scenarios: riskProjectionScenarios{
			DeployAll:  deployAll,
			Trajectory: trajectory,
			DoNothing:  doNothing,
		},
	})
}
