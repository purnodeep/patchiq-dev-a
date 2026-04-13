package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// patchListItem is the JSON shape for each item in the List() response.
// We use an explicit struct (instead of serialising ListPatchesFilteredRow directly)
// so that time fields are rendered as RFC-3339 strings rather than pgtype objects.
type patchListItem struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	Version                string  `json:"version"`
	Severity               string  `json:"severity"`
	OsFamily               string  `json:"os_family"`
	Status                 string  `json:"status"`
	CreatedAt              string  `json:"created_at"`
	ReleasedAt             string  `json:"released_at"`
	OsDistribution         *string `json:"os_distribution"`
	Description            *string `json:"description"`
	CveCount               int32   `json:"cve_count"`
	HighestCvssScore       float64 `json:"highest_cvss_score"`
	RemediationPct         int32   `json:"remediation_pct"`
	EndpointsDeployedCount int32   `json:"endpoints_deployed_count"`
	AffectedEndpointCount  int32   `json:"affected_endpoint_count"`
}

// PatchQuerier defines the sqlc queries needed by PatchHandler.
type PatchQuerier interface {
	ListPatchesFiltered(ctx context.Context, arg sqlcgen.ListPatchesFilteredParams) ([]sqlcgen.ListPatchesFilteredRow, error)
	CountPatchesFiltered(ctx context.Context, arg sqlcgen.CountPatchesFilteredParams) (int64, error)
	CountPatchesBySeverity(ctx context.Context, arg sqlcgen.CountPatchesBySeverityParams) ([]sqlcgen.CountPatchesBySeverityRow, error)
	GetPatchByID(ctx context.Context, arg sqlcgen.GetPatchByIDParams) (sqlcgen.Patch, error)
	ListCVEsForPatch(ctx context.Context, arg sqlcgen.ListCVEsForPatchParams) ([]sqlcgen.CVE, error)
	GetPatchRemediation(ctx context.Context, arg sqlcgen.GetPatchRemediationParams) (sqlcgen.GetPatchRemediationRow, error)
	ListAffectedEndpointsForPatch(ctx context.Context, arg sqlcgen.ListAffectedEndpointsForPatchParams) ([]sqlcgen.ListAffectedEndpointsForPatchRow, error)
	CountAffectedEndpointsForPatch(ctx context.Context, arg sqlcgen.CountAffectedEndpointsForPatchParams) (int64, error)
	ListDeploymentsForPatch(ctx context.Context, arg sqlcgen.ListDeploymentsForPatchParams) ([]sqlcgen.ListDeploymentsForPatchRow, error)
	ListDeploymentHistoryForPatch(ctx context.Context, arg sqlcgen.ListDeploymentHistoryForPatchParams) ([]sqlcgen.ListDeploymentHistoryForPatchRow, error)
	GetPatchHighestCVSS(ctx context.Context, arg sqlcgen.GetPatchHighestCVSSParams) (float64, error)
	ListEndpointsByTenant(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.Endpoint, error)
}

// PatchWriteQuerier extends PatchQuerier with write operations used by QuickDeploy.
type PatchWriteQuerier interface {
	PatchQuerier
	CreateQuickDeployment(ctx context.Context, arg sqlcgen.CreateQuickDeploymentParams) (sqlcgen.Deployment, error)
	CreateDeploymentWaveWithConfig(ctx context.Context, arg sqlcgen.CreateDeploymentWaveWithConfigParams) (sqlcgen.DeploymentWave, error)
	BulkCreateDeploymentTargets(ctx context.Context, arg []sqlcgen.BulkCreateDeploymentTargetsParams) (int64, error)
	SetDeploymentWaveTargetCount(ctx context.Context, arg sqlcgen.SetDeploymentWaveTargetCountParams) (sqlcgen.DeploymentWave, error)
	SetDeploymentTotalTargets(ctx context.Context, arg sqlcgen.SetDeploymentTotalTargetsParams) (sqlcgen.Deployment, error)
}

// PatchHandler serves patch REST API endpoints.
type PatchHandler struct {
	q        PatchQuerier
	pool     TxBeginner
	eventBus domain.EventBus
}

// NewPatchHandler creates a PatchHandler.
func NewPatchHandler(q PatchQuerier) *PatchHandler {
	if q == nil {
		panic("patches: NewPatchHandler called with nil querier")
	}
	return &PatchHandler{q: q}
}

// WithPool attaches a transaction pool to enable QuickDeploy persistence.
func (h *PatchHandler) WithPool(pool TxBeginner) *PatchHandler {
	h.pool = pool
	return h
}

// WithEventBus attaches an event bus to emit domain events from patch operations.
func (h *PatchHandler) WithEventBus(bus domain.EventBus) *PatchHandler {
	h.eventBus = bus
	return h
}

// List handles GET /api/v1/patches with pagination and filters.
func (h *PatchHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	cursorTime, cursorID, err := DecodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor")
		return
	}
	limit := ParseLimit(r.URL.Query().Get("limit"))

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var cursorTS pgtype.Timestamptz
	var cursorUUID pgtype.UUID
	if !cursorTime.IsZero() {
		cursorTS = pgtype.Timestamptz{Time: cursorTime, Valid: true}
		cursorUUID, err = scanUUID(cursorID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor: cursor ID is not a valid UUID")
			return
		}
	}

	// Validate sort parameters — only allow known columns.
	sortBy := r.URL.Query().Get("sort_by")
	sortDir := r.URL.Query().Get("sort_dir")
	validSorts := map[string]bool{"name": true, "severity": true, "cvss": true, "cves": true, "affected": true}
	if !validSorts[sortBy] {
		sortBy = ""
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = ""
	}

	params := sqlcgen.ListPatchesFilteredParams{
		TenantID:        tid,
		Severity:        r.URL.Query().Get("severity"),
		OsFamily:        r.URL.Query().Get("os_family"),
		OsDistribution:  r.URL.Query().Get("os_distribution"),
		Status:          r.URL.Query().Get("status"),
		Search:          r.URL.Query().Get("search"),
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageLimit:       limit,
		SortBy:          sortBy,
		SortDir:         sortDir,
	}

	patches, err := h.q.ListPatchesFiltered(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "list patches", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list patches")
		return
	}

	countParams := sqlcgen.CountPatchesFilteredParams{
		TenantID:       tid,
		Severity:       params.Severity,
		OsFamily:       params.OsFamily,
		OsDistribution: params.OsDistribution,
		Status:         params.Status,
		Search:         params.Search,
	}
	total, err := h.q.CountPatchesFiltered(ctx, countParams)
	if err != nil {
		slog.ErrorContext(ctx, "count patches", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count patches")
		return
	}

	var nextCursor string
	if len(patches) == int(limit) {
		last := patches[len(patches)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, uuidToString(last.ID))
	}

	items := make([]patchListItem, len(patches))
	for i, p := range patches {
		releasedAt := p.CreatedAt.Time.Format(time.RFC3339)
		if p.ReleasedAt.Valid {
			releasedAt = p.ReleasedAt.Time.Format(time.RFC3339)
		}
		items[i] = patchListItem{
			ID:                     uuidToString(p.ID),
			Name:                   p.Name,
			Version:                p.Version,
			Severity:               p.Severity,
			OsFamily:               p.OsFamily,
			Status:                 p.Status,
			CreatedAt:              p.CreatedAt.Time.Format(time.RFC3339),
			ReleasedAt:             releasedAt,
			OsDistribution:         nullableText(p.OsDistribution),
			Description:            nullableText(p.Description),
			CveCount:               p.CveCount,
			HighestCvssScore:       p.HighestCvssScore,
			RemediationPct:         p.RemediationPct,
			EndpointsDeployedCount: p.EndpointsDeployedCount,
			AffectedEndpointCount:  p.AffectedEndpointCount,
		}
	}
	WriteList(w, items, nextCursor, total)
}

// SeverityCounts handles GET /api/v1/patches/severity-counts.
func (h *PatchHandler) SeverityCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.CountPatchesBySeverity(ctx, sqlcgen.CountPatchesBySeverityParams{
		TenantID:       tid,
		OsFamily:       r.URL.Query().Get("os_family"),
		OsDistribution: r.URL.Query().Get("os_distribution"),
		Status:         r.URL.Query().Get("status"),
		Search:         r.URL.Query().Get("search"),
	})
	if err != nil {
		slog.ErrorContext(ctx, "count patches by severity", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count patches by severity")
		return
	}

	counts := map[string]int32{}
	for _, row := range rows {
		counts[row.Severity] = row.Count
	}
	WriteJSON(w, http.StatusOK, counts)
}

// cveResponse is the full CVE object returned in patch detail.
type cveResponse struct {
	ID               string  `json:"id"`
	CveID            string  `json:"cve_id"`
	CvssV3Score      *string `json:"cvss_v3_score"`
	Severity         string  `json:"severity"`
	PublishedAt      *string `json:"published_at"`
	ExploitAvailable bool    `json:"exploit_available"`
	CisaKev          bool    `json:"cisa_kev"`
	CvssV3Vector     *string `json:"cvss_v3_vector"`
	Description      *string `json:"description"`
	AttackVector     *string `json:"attack_vector"`
}

// affectedEndpointResponse represents an endpoint affected by a patch.
type affectedEndpointResponse struct {
	ID             string  `json:"id"`
	Hostname       string  `json:"hostname"`
	OsFamily       string  `json:"os_family"`
	AgentVersion   *string `json:"agent_version"`
	Status         string  `json:"status"`
	PatchStatus    string  `json:"patch_status"`
	LastDeployedAt *string `json:"last_deployed_at"`
}

// affectedEndpointsListResponse wraps affected endpoints with pagination metadata.
type affectedEndpointsListResponse struct {
	Total   int64                      `json:"total"`
	Items   []affectedEndpointResponse `json:"items"`
	HasMore bool                       `json:"has_more"`
}

// deploymentHistoryEntry represents a past deployment involving this patch.
type deploymentHistoryEntry struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	TriggeredBy  string  `json:"triggered_by"`
	StartedAt    *string `json:"started_at"`
	CompletedAt  *string `json:"completed_at"`
	TotalTargets int32   `json:"total_targets"`
	SuccessCount int32   `json:"success_count"`
	FailedCount  int32   `json:"failed_count"`
}

// patchDetailResponse is the JSON body for GET /patches/{id}.
type patchDetailResponse struct {
	ID                string                        `json:"id"`
	TenantID          string                        `json:"tenant_id"`
	Name              string                        `json:"name"`
	Version           string                        `json:"version"`
	Severity          string                        `json:"severity"`
	OsFamily          string                        `json:"os_family"`
	Status            string                        `json:"status"`
	OsDistribution    *string                       `json:"os_distribution"`
	PackageURL        *string                       `json:"package_url"`
	ChecksumSha256    *string                       `json:"checksum_sha256"`
	SourceRepo        *string                       `json:"source_repo"`
	Description       *string                       `json:"description"`
	CreatedAt         string                        `json:"created_at"`
	UpdatedAt         string                        `json:"updated_at"`
	ReleasedAt        string                        `json:"released_at"`
	FileSize          *int64                        `json:"file_size"`
	CVEs              []cveResponse                 `json:"cves"`
	Remediation       remediationResponse           `json:"remediation"`
	AffectedEndpoints affectedEndpointsListResponse `json:"affected_endpoints"`
	DeploymentHistory []deploymentHistoryEntry      `json:"deployment_history"`
	HighestCvssScore  float64                       `json:"highest_cvss_score"`
	AvgInstallTimeMs  *int64                        `json:"avg_install_time_ms"`
}

type remediationResponse struct {
	EndpointsAffected int32 `json:"endpoints_affected"`
	EndpointsPatched  int32 `json:"endpoints_patched"`
	EndpointsPending  int32 `json:"endpoints_pending"`
	EndpointsFailed   int32 `json:"endpoints_failed"`
}

// quickDeployRequest is the request body for POST /patches/{id}/deploy.
type quickDeployRequest struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	ConfigType      string   `json:"config_type"`
	EndpointFilter  string   `json:"endpoint_filter"`
	TargetEndpoints string   `json:"target_endpoints"`
	EndpointIDs     []string `json:"endpoint_ids"`
	ScheduledAt     *string  `json:"scheduled_at"`
}

// filterEndpointsForDeploy returns the subset of endpoints that should be
// targeted by a quick deploy. When endpointIDs is non-empty, only endpoints
// whose UUID string matches one of the provided IDs are included (and
// decommissioned endpoints are always excluded). When endpointIDs is empty,
// the legacy osFamily filter ("windows" / "linux" / anything-else=all) applies.
func filterEndpointsForDeploy(endpoints []sqlcgen.Endpoint, endpointIDs []string, osFamily string) []sqlcgen.Endpoint {
	out := endpoints[:0]
	if len(endpointIDs) > 0 {
		allowed := make(map[string]struct{}, len(endpointIDs))
		for _, id := range endpointIDs {
			allowed[id] = struct{}{}
		}
		for _, ep := range endpoints {
			if ep.Status == "decommissioned" {
				continue
			}
			epID := uuidToString(ep.ID)
			if _, ok := allowed[epID]; ok {
				out = append(out, ep)
			}
		}
		return out
	}
	for _, ep := range endpoints {
		if ep.Status == "decommissioned" {
			continue
		}
		switch osFamily {
		case "windows":
			if ep.OsFamily == "windows" {
				out = append(out, ep)
			}
		case "linux":
			if ep.OsFamily == "linux" {
				out = append(out, ep)
			}
		default:
			out = append(out, ep)
		}
	}
	return out
}

// extractAttackVector derives the attack vector label from a CVSS v3 vector string.
// Returns nil if vector is empty or AV component is not recognised.
func extractAttackVector(vector string) *string {
	if vector == "" {
		return nil
	}
	for _, part := range strings.Split(vector, "/") {
		if strings.HasPrefix(part, "AV:") {
			var label string
			switch part {
			case "AV:N":
				label = "Network"
			case "AV:L":
				label = "Local"
			case "AV:P":
				label = "Physical"
			case "AV:A":
				label = "Adjacent"
			default:
				return nil
			}
			return &label
		}
	}
	return nil
}

// buildCVEResponse converts a sqlcgen.CVE into a cveResponse.
func buildCVEResponse(c sqlcgen.CVE) cveResponse {
	resp := cveResponse{
		ID:               uuidToString(c.ID),
		CveID:            c.CveID,
		Severity:         c.Severity,
		ExploitAvailable: c.ExploitAvailable,
		CisaKev:          c.CisaKevDueDate.Valid,
	}
	// Convert CVSS score with proper precision
	if c.CvssV3Score.Valid {
		s := numericToString(c.CvssV3Score)
		resp.CvssV3Score = &s
	}
	if c.PublishedAt.Valid {
		s := c.PublishedAt.Time.Format(time.RFC3339)
		resp.PublishedAt = &s
	}
	if c.CvssV3Vector.Valid {
		v := c.CvssV3Vector.String
		resp.CvssV3Vector = &v
		resp.AttackVector = extractAttackVector(v)
	}
	if c.Description.Valid {
		d := c.Description.String
		resp.Description = &d
	}
	return resp
}

// Get handles GET /api/v1/patches/{id}.
func (h *PatchHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid patch ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	patch, err := h.q.GetPatchByID(ctx, sqlcgen.GetPatchByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "patch not found")
			return
		}
		slog.ErrorContext(ctx, "get patch", "patch_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get patch")
		return
	}

	cves, err := h.q.ListCVEsForPatch(ctx, sqlcgen.ListCVEsForPatchParams{PatchID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "patches: list CVEs for patch", "patch_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list CVEs for patch")
		return
	}

	remediation, err := h.q.GetPatchRemediation(ctx, sqlcgen.GetPatchRemediationParams{TenantID: tid, PatchID: id})
	if err != nil {
		slog.ErrorContext(ctx, "patches: get remediation", "patch_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get patch remediation")
		return
	}

	affectedEndpoints, err := h.q.ListAffectedEndpointsForPatch(ctx, sqlcgen.ListAffectedEndpointsForPatchParams{PatchID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "patches: list affected endpoints", "patch_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list affected endpoints for patch")
		return
	}

	deploymentHistory, err := h.q.ListDeploymentHistoryForPatch(ctx, sqlcgen.ListDeploymentHistoryForPatchParams{PatchID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "patches: list deployment history", "patch_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deployment history for patch")
		return
	}

	highestCVSS, err := h.q.GetPatchHighestCVSS(ctx, sqlcgen.GetPatchHighestCVSSParams{PatchID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "patches: get highest CVSS", "patch_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get highest CVSS for patch")
		return
	}

	cveObjs := make([]cveResponse, len(cves))
	for i, c := range cves {
		cveObjs[i] = buildCVEResponse(c)
	}

	epItems := make([]affectedEndpointResponse, len(affectedEndpoints))
	for i, e := range affectedEndpoints {
		epItems[i] = affectedEndpointResponse{
			ID:          uuidToString(e.ID),
			Hostname:    e.Hostname,
			OsFamily:    e.OsFamily,
			Status:      e.Status,
			PatchStatus: e.PatchStatus,
		}
		if e.AgentVersion.Valid {
			v := e.AgentVersion.String
			epItems[i].AgentVersion = &v
		}
		if ts, ok := e.LastDeployedAt.(pgtype.Timestamptz); ok && ts.Valid {
			s := ts.Time.Format(time.RFC3339)
			epItems[i].LastDeployedAt = &s
		}
	}

	histItems := make([]deploymentHistoryEntry, len(deploymentHistory))
	for i, d := range deploymentHistory {
		entry := deploymentHistoryEntry{
			ID:           uuidToString(d.ID),
			Status:       d.Status,
			TriggeredBy:  uuidToString(d.CreatedBy),
			TotalTargets: d.TotalTargets,
			SuccessCount: d.SuccessCount,
			FailedCount:  d.FailedCount,
		}
		if d.StartedAt.Valid {
			s := d.StartedAt.Time.Format(time.RFC3339)
			entry.StartedAt = &s
		}
		if d.CompletedAt.Valid {
			s := d.CompletedAt.Time.Format(time.RFC3339)
			entry.CompletedAt = &s
		}
		histItems[i] = entry
	}

	const affectedLimit = 50
	hasMore := len(epItems) == affectedLimit

	// Compute avg install time from completed deployment records.
	var avgInstallTimeMs *int64
	{
		var totalMs int64
		var count int64
		for _, d := range deploymentHistory {
			if d.Status == "success" && d.StartedAt.Valid && d.CompletedAt.Valid {
				durationMs := d.CompletedAt.Time.Sub(d.StartedAt.Time).Milliseconds()
				if durationMs > 0 {
					totalMs += durationMs
					count++
				}
			}
		}
		if count > 0 {
			avg := totalMs / count
			avgInstallTimeMs = &avg
		}
	}

	resp := patchDetailResponse{
		ID:             uuidToString(patch.ID),
		TenantID:       uuidToString(patch.TenantID),
		Name:           patch.Name,
		Version:        patch.Version,
		Severity:       patch.Severity,
		OsFamily:       patch.OsFamily,
		Status:         patch.Status,
		OsDistribution: nullableText(patch.OsDistribution),
		PackageURL:     nullableText(patch.PackageUrl),
		ChecksumSha256: nullableText(patch.ChecksumSha256),
		SourceRepo:     nullableText(patch.SourceRepo),
		Description:    nullableText(patch.Description),
		CreatedAt:      patch.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:      patch.UpdatedAt.Time.Format(time.RFC3339),
		ReleasedAt: func() string {
			if patch.ReleasedAt.Valid {
				return patch.ReleasedAt.Time.Format(time.RFC3339)
			}
			return patch.CreatedAt.Time.Format(time.RFC3339)
		}(),
		FileSize: nil,
		CVEs:     cveObjs,
		Remediation: remediationResponse{
			EndpointsAffected: remediation.EndpointsAffected,
			EndpointsPatched:  remediation.EndpointsPatched,
			EndpointsPending:  remediation.EndpointsPending,
			EndpointsFailed:   remediation.EndpointsFailed,
		},
		AffectedEndpoints: affectedEndpointsListResponse{
			Total:   int64(len(epItems)),
			Items:   epItems,
			HasMore: hasMore,
		},
		DeploymentHistory: histItems,
		HighestCvssScore:  highestCVSS,
		AvgInstallTimeMs:  avgInstallTimeMs,
	}

	WriteJSON(w, http.StatusOK, resp)
}

// QuickDeploy handles POST /api/v1/patches/{id}/deploy.
// Creates a deployment for the given patch targeting all matching endpoints.
func (h *PatchHandler) QuickDeploy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid patch ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	patch, err := h.q.GetPatchByID(ctx, sqlcgen.GetPatchByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "patch not found")
			return
		}
		slog.ErrorContext(ctx, "patches: quick deploy get patch", "patch_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get patch")
		return
	}

	var req quickDeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if h.pool == nil {
		WriteError(w, http.StatusInternalServerError, "NOT_CONFIGURED", "deployment persistence not configured")
		return
	}

	wq, ok := h.q.(PatchWriteQuerier)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "NOT_CONFIGURED", "deployment persistence not configured")
		return
	}

	// Determine deployment name.
	name := req.Name
	if name == "" {
		name = patch.Name + " - Deployment"
	}

	// Resolve default single-wave config (100%, no delay).
	waveConfigs, err := deployment.ParseWaveConfig(nil)
	if err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy parse wave config", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse wave config")
		return
	}
	waveConfigJSON, err := json.Marshal(waveConfigs)
	if err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy marshal wave config", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to marshal wave config")
		return
	}

	// List all active endpoints for this tenant.
	endpoints, err := h.q.ListEndpointsByTenant(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy list endpoints", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoints")
		return
	}

	// Filter endpoints: by specific IDs when provided, otherwise by OS family.
	osFamily := req.TargetEndpoints
	if osFamily == "" {
		osFamily = req.EndpointFilter
	}
	filtered := filterEndpointsForDeploy(endpoints, req.EndpointIDs, osFamily)

	// Begin transaction.
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy begin tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to begin transaction")
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.WarnContext(ctx, "rollback quick deploy tx", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy set tenant ctx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := sqlcgen.New(tx)

	// Ensure txQ satisfies PatchWriteQuerier (it will at compile time via wq usage).
	_ = wq

	dep, err := txQ.CreateQuickDeployment(ctx, sqlcgen.CreateQuickDeploymentParams{
		TenantID:   tid,
		PatchID:    pgtype.UUID{Bytes: id.Bytes, Valid: true},
		Name:       pgtype.Text{String: name, Valid: true},
		Status:     string(deployment.StatusCreated),
		WaveConfig: waveConfigJSON,
	})
	if err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy create deployment", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment")
		return
	}

	// Create a single wave (100%).
	wc := waveConfigs[0]
	wave, err := txQ.CreateDeploymentWaveWithConfig(ctx, sqlcgen.CreateDeploymentWaveWithConfigParams{
		TenantID:          tid,
		DeploymentID:      dep.ID,
		WaveNumber:        1,
		Status:            string(deployment.WavePending),
		Percentage:        int32(wc.Percentage),
		SuccessThreshold:  pgtype.Numeric{Int: big.NewInt(int64(wc.SuccessThreshold * 100)), Exp: -2, Valid: true},
		ErrorRateMax:      pgtype.Numeric{Int: big.NewInt(int64(wc.ErrorRateMax * 100)), Exp: -2, Valid: true},
		DelayAfterMinutes: int32(wc.DelayMinutes),
	})
	if err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy create wave", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment wave")
		return
	}

	// Create targets for each endpoint (batch INSERT).
	bulkParams := make([]sqlcgen.BulkCreateDeploymentTargetsParams, len(filtered))
	for i, ep := range filtered {
		bulkParams[i] = sqlcgen.BulkCreateDeploymentTargetsParams{
			TenantID:     tid,
			DeploymentID: dep.ID,
			EndpointID:   ep.ID,
			PatchID:      id,
			Status:       string(deployment.TargetPending),
			WaveID:       wave.ID,
		}
	}
	if _, err := txQ.BulkCreateDeploymentTargets(ctx, bulkParams); err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy create targets", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "target_count", len(bulkParams), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment targets")
		return
	}

	targetCount := int32(len(filtered))
	if _, err := txQ.SetDeploymentWaveTargetCount(ctx, sqlcgen.SetDeploymentWaveTargetCountParams{
		ID:          wave.ID,
		TargetCount: targetCount,
		TenantID:    tid,
	}); err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy set wave target count", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set wave target count")
		return
	}

	dep, err = txQ.SetDeploymentTotalTargets(ctx, sqlcgen.SetDeploymentTotalTargetsParams{
		ID:           dep.ID,
		TotalTargets: targetCount,
		TenantID:     tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy set total targets", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set total targets")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "patches: quick deploy commit tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit deployment")
		return
	}

	emitEvent(ctx, h.eventBus, events.DeploymentCreated, "deployment", uuidToString(dep.ID), tenantID, map[string]any{
		"patch_id":      uuidToString(id),
		"total_targets": dep.TotalTargets,
		"source":        "quick_deploy",
	})

	WriteJSON(w, http.StatusOK, map[string]any{
		"id":            uuidToString(dep.ID),
		"status":        dep.Status,
		"total_targets": dep.TotalTargets,
		"name":          name,
	})
}

// deployCriticalRequest is the request body for POST /endpoints/{id}/deploy-critical.
type deployCriticalRequest struct {
	PatchIDs []string `json:"patch_ids"`
	Name     string   `json:"name"`
}

// DeployCritical handles POST /api/v1/endpoints/{id}/deploy-critical.
// Creates ONE deployment targeting the specified endpoint with all provided patches,
// resulting in one deployment row instead of N separate rows (one per patch).
func (h *PatchHandler) DeployCritical(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	endpointID, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var req deployCriticalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if len(req.PatchIDs) == 0 {
		WriteError(w, http.StatusBadRequest, "NO_PATCHES", "patch_ids must not be empty")
		return
	}

	patchUUIDs := make([]pgtype.UUID, 0, len(req.PatchIDs))
	for _, pid := range req.PatchIDs {
		u, err := scanUUID(pid)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_PATCH_ID", fmt.Sprintf("invalid patch ID %q: not a valid UUID", pid))
			return
		}
		patchUUIDs = append(patchUUIDs, u)
	}

	if h.pool == nil {
		WriteError(w, http.StatusInternalServerError, "NOT_CONFIGURED", "deployment persistence not configured")
		return
	}
	if _, ok := h.q.(PatchWriteQuerier); !ok {
		WriteError(w, http.StatusInternalServerError, "NOT_CONFIGURED", "deployment persistence not configured")
		return
	}

	name := req.Name
	if name == "" {
		name = fmt.Sprintf("Critical patch deployment (%d patches)", len(req.PatchIDs))
	}

	waveConfigs, err := deployment.ParseWaveConfig(nil)
	if err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical parse wave config", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse wave config")
		return
	}
	waveConfigJSON, err := json.Marshal(waveConfigs)
	if err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical marshal wave config", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to marshal wave config")
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical begin tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to begin transaction")
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.WarnContext(ctx, "rollback deploy critical tx", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical set tenant ctx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := sqlcgen.New(tx)

	dep, err := txQ.CreateQuickDeployment(ctx, sqlcgen.CreateQuickDeploymentParams{
		TenantID:   tid,
		PatchID:    pgtype.UUID{}, // null — multi-patch deployment
		Name:       pgtype.Text{String: name, Valid: true},
		Status:     string(deployment.StatusCreated),
		WaveConfig: waveConfigJSON,
	})
	if err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical create deployment", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment")
		return
	}

	wc := waveConfigs[0]
	wave, err := txQ.CreateDeploymentWaveWithConfig(ctx, sqlcgen.CreateDeploymentWaveWithConfigParams{
		TenantID:          tid,
		DeploymentID:      dep.ID,
		WaveNumber:        1,
		Status:            string(deployment.WavePending),
		Percentage:        int32(wc.Percentage),
		SuccessThreshold:  pgtype.Numeric{Int: big.NewInt(int64(wc.SuccessThreshold * 100)), Exp: -2, Valid: true},
		ErrorRateMax:      pgtype.Numeric{Int: big.NewInt(int64(wc.ErrorRateMax * 100)), Exp: -2, Valid: true},
		DelayAfterMinutes: int32(wc.DelayMinutes),
	})
	if err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical create wave", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment wave")
		return
	}

	bulkParams := make([]sqlcgen.BulkCreateDeploymentTargetsParams, len(patchUUIDs))
	for i, patchID := range patchUUIDs {
		bulkParams[i] = sqlcgen.BulkCreateDeploymentTargetsParams{
			TenantID:     tid,
			DeploymentID: dep.ID,
			EndpointID:   endpointID,
			PatchID:      patchID,
			Status:       string(deployment.TargetPending),
			WaveID:       wave.ID,
		}
	}
	if _, err := txQ.BulkCreateDeploymentTargets(ctx, bulkParams); err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical create targets", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "target_count", len(bulkParams), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment targets")
		return
	}

	targetCount := int32(len(patchUUIDs))
	if _, err := txQ.SetDeploymentWaveTargetCount(ctx, sqlcgen.SetDeploymentWaveTargetCountParams{
		ID:          wave.ID,
		TargetCount: targetCount,
		TenantID:    tid,
	}); err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical set wave target count", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set wave target count")
		return
	}

	dep, err = txQ.SetDeploymentTotalTargets(ctx, sqlcgen.SetDeploymentTotalTargetsParams{
		ID:           dep.ID,
		TotalTargets: targetCount,
		TenantID:     tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical set total targets", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set total targets")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "patches: deploy critical commit tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit deployment")
		return
	}

	emitEvent(ctx, h.eventBus, events.DeploymentCreated, "deployment", uuidToString(dep.ID), tenantID, map[string]any{
		"endpoint_id":   uuidToString(endpointID),
		"patch_count":   len(req.PatchIDs),
		"total_targets": dep.TotalTargets,
		"source":        "deploy_critical",
	})

	WriteJSON(w, http.StatusOK, map[string]any{
		"id":            uuidToString(dep.ID),
		"status":        dep.Status,
		"total_targets": dep.TotalTargets,
		"name":          name,
	})
}
