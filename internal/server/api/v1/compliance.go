package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/compliance"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// ComplianceQuerier defines the sqlc queries needed by ComplianceHandler.
type ComplianceQuerier interface {
	ListTenantFrameworks(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.ComplianceTenantFramework, error)
	GetTenantFramework(ctx context.Context, arg sqlcgen.GetTenantFrameworkParams) (sqlcgen.ComplianceTenantFramework, error)
	GetTenantFrameworkByID(ctx context.Context, arg sqlcgen.GetTenantFrameworkByIDParams) (sqlcgen.ComplianceTenantFramework, error)
	CreateTenantFramework(ctx context.Context, arg sqlcgen.CreateTenantFrameworkParams) (sqlcgen.ComplianceTenantFramework, error)
	UpdateTenantFramework(ctx context.Context, arg sqlcgen.UpdateTenantFrameworkParams) (sqlcgen.ComplianceTenantFramework, error)
	DeleteTenantFramework(ctx context.Context, arg sqlcgen.DeleteTenantFrameworkParams) error
	GetLatestFrameworkScore(ctx context.Context, arg sqlcgen.GetLatestFrameworkScoreParams) (sqlcgen.ComplianceScore, error)
	ListEndpointScoresByFramework(ctx context.Context, arg sqlcgen.ListEndpointScoresByFrameworkParams) ([]sqlcgen.ComplianceScore, error)
	ListEvaluationsByEndpoint(ctx context.Context, arg sqlcgen.ListEvaluationsByEndpointParams) ([]sqlcgen.ComplianceEvaluation, error)
	GetLastEvaluationTime(ctx context.Context, tenantID pgtype.UUID) (pgtype.Timestamptz, error)
	ListEnabledTenantFrameworks(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.ComplianceTenantFramework, error)
	GetOverallComplianceScore(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetOverallComplianceScoreRow, error)
	ListControlResultsByFramework(ctx context.Context, arg sqlcgen.ListControlResultsByFrameworkParams) ([]sqlcgen.ComplianceControlResult, error)
	ListOverdueControls(ctx context.Context, arg sqlcgen.ListOverdueControlsParams) ([]sqlcgen.ComplianceControlResult, error)
	GetControlCountsByFramework(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetControlCountsByFrameworkRow, error)
	GetCompliantEndpointCountsByFramework(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetCompliantEndpointCountsByFrameworkRow, error)
	ListScoreTrend(ctx context.Context, arg sqlcgen.ListScoreTrendParams) ([]sqlcgen.ComplianceScore, error)
	ListNonCompliantEndpointsByFramework(ctx context.Context, arg sqlcgen.ListNonCompliantEndpointsByFrameworkParams) ([]sqlcgen.ListNonCompliantEndpointsByFrameworkRow, error)
	GetCustomFramework(ctx context.Context, arg sqlcgen.GetCustomFrameworkParams) (sqlcgen.CustomComplianceFramework, error)
	ListCustomControls(ctx context.Context, arg sqlcgen.ListCustomControlsParams) ([]sqlcgen.CustomComplianceControl, error)
}

// ComplianceHandler serves compliance REST API endpoints.
type ComplianceHandler struct {
	q        ComplianceQuerier
	svc      *compliance.Service
	eventBus domain.EventBus
	store    *store.Store
}

// NewComplianceHandler creates a ComplianceHandler.
func NewComplianceHandler(q ComplianceQuerier, svc *compliance.Service, eventBus domain.EventBus, st *store.Store) *ComplianceHandler {
	if q == nil {
		panic("compliance: NewComplianceHandler called with nil querier")
	}
	if svc == nil {
		panic("compliance: NewComplianceHandler called with nil service")
	}
	if eventBus == nil {
		panic("compliance: NewComplianceHandler called with nil eventBus")
	}
	if st == nil {
		panic("compliance: NewComplianceHandler called with nil store")
	}
	return &ComplianceHandler{q: q, svc: svc, eventBus: eventBus, store: st}
}

// resolveFramework looks up a framework by ID, checking built-in first then custom.
// Returns a lightweight Framework-like info struct, or nil if not found.
func (h *ComplianceHandler) resolveFrameworkName(ctx context.Context, frameworkID string, tid pgtype.UUID) (name, version string, found bool) {
	if fw := compliance.GetFramework(frameworkID); fw != nil {
		return fw.Name, fw.Version, true
	}
	// Try custom framework.
	fwUUID, err := scanUUID(frameworkID)
	if err != nil {
		return "", "", false
	}
	dbFW, err := h.q.GetCustomFramework(ctx, sqlcgen.GetCustomFrameworkParams{ID: fwUUID, TenantID: tid})
	if err != nil {
		return "", "", false
	}
	return dbFW.Name, dbFW.Version, true
}

// ------------------------------------------------------------
// Response types
// ------------------------------------------------------------

type frameworkScoreSummary struct {
	FrameworkID        string  `json:"framework_id"`
	Name               string  `json:"name"`
	Score              *string `json:"score"`
	TotalCVEs          int32   `json:"total_cves"`
	Compliant          int32   `json:"compliant"`
	AtRisk             int32   `json:"at_risk"`
	NonCompliant       int32   `json:"non_compliant"`
	EvaluatedAt        *string `json:"evaluated_at"`
	Status             string  `json:"status"`
	TotalControls      int32   `json:"total_controls"`
	PassingControls    int32   `json:"passing_controls"`
	FailingControls    int32   `json:"failing_controls"`
	NaControls         int32   `json:"na_controls"`
	EndpointsCompliant int32   `json:"endpoints_compliant"`
	TotalEndpoints     int32   `json:"total_endpoints"`
	OverdueCount       int32   `json:"overdue_count"`
}

type complianceSummaryResponse struct {
	Frameworks      []frameworkScoreSummary `json:"frameworks"`
	LastEvaluatedAt *string                 `json:"last_evaluated_at"`
}

type frameworkListItem struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Version              string   `json:"version"`
	Description          string   `json:"description"`
	ApplicableIndustries []string `json:"applicable_industries"`
	Enabled              bool     `json:"enabled"`
	ScoringMethod        *string  `json:"scoring_method"`
	ConfigID             *string  `json:"config_id"`
}

type tenantFrameworkResponse struct {
	ID              string          `json:"id"`
	FrameworkID     string          `json:"framework_id"`
	Enabled         bool            `json:"enabled"`
	SlaOverrides    json.RawMessage `json:"sla_overrides"`
	ScoringMethod   string          `json:"scoring_method"`
	AtRiskThreshold *string         `json:"at_risk_threshold"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

type frameworkDetailResponse struct {
	Framework             frameworkListItem              `json:"framework"`
	Config                *tenantFrameworkResponse       `json:"config"`
	Score                 *scoreResponse                 `json:"score"`
	EndpointScores        []endpointScoreResponse        `json:"endpoint_scores"`
	Categories            []categoryBreakdown            `json:"categories"`
	NonCompliantEndpoints []nonCompliantEndpointResponse `json:"non_compliant_endpoints"`
}

type scoreResponse struct {
	Score               string `json:"score"`
	TotalCVEs           int32  `json:"total_cves"`
	CompliantCVEs       int32  `json:"compliant_cves"`
	AtRiskCVEs          int32  `json:"at_risk_cves"`
	NonCompliantCVEs    int32  `json:"non_compliant_cves"`
	LateRemediationCVEs int32  `json:"late_remediation_cves"`
	EvaluatedAt         string `json:"evaluated_at"`
}

type endpointScoreResponse struct {
	EndpointID          string `json:"endpoint_id"`
	Score               string `json:"score"`
	TotalCVEs           int32  `json:"total_cves"`
	CompliantCVEs       int32  `json:"compliant_cves"`
	AtRiskCVEs          int32  `json:"at_risk_cves"`
	NonCompliantCVEs    int32  `json:"non_compliant_cves"`
	LateRemediationCVEs int32  `json:"late_remediation_cves"`
	EvaluatedAt         string `json:"evaluated_at"`
}

type endpointEvaluationResponse struct {
	ID            string  `json:"id"`
	FrameworkID   string  `json:"framework_id"`
	ControlID     string  `json:"control_id"`
	CveID         string  `json:"cve_id"`
	State         string  `json:"state"`
	SlaDeadlineAt *string `json:"sla_deadline_at"`
	RemediatedAt  *string `json:"remediated_at"`
	DaysRemaining *int32  `json:"days_remaining"`
	EvaluatedAt   string  `json:"evaluated_at"`
}

type enableFrameworkRequest struct {
	FrameworkID     string          `json:"framework_id"`
	ScoringMethod   string          `json:"scoring_method"`
	SlaOverrides    json.RawMessage `json:"sla_overrides"`
	AtRiskThreshold *float64        `json:"at_risk_threshold"`
}

type updateFrameworkRequest struct {
	Enabled         *bool           `json:"enabled"`
	ScoringMethod   string          `json:"scoring_method"`
	SlaOverrides    json.RawMessage `json:"sla_overrides"`
	AtRiskThreshold *float64        `json:"at_risk_threshold"`
}

type overallScoreResponse struct {
	OverallScore     string  `json:"overall_score"`
	TotalCVEs        int64   `json:"total_cves"`
	CompliantCVEs    int64   `json:"compliant_cves"`
	AtRiskCVEs       int64   `json:"at_risk_cves"`
	NonCompliantCVEs int64   `json:"non_compliant_cves"`
	FrameworkCount   int64   `json:"framework_count"`
	Status           string  `json:"status"`
	LastEvaluatedAt  *string `json:"last_evaluated_at"`
}

type controlResultResponse struct {
	ControlID        string  `json:"control_id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Category         string  `json:"category"`
	Status           string  `json:"status"`
	PassingEndpoints int32   `json:"passing_endpoints"`
	TotalEndpoints   int32   `json:"total_endpoints"`
	RemediationHint  *string `json:"remediation_hint"`
	SlaDeadlineAt    *string `json:"sla_deadline_at"`
	DaysOverdue      *int32  `json:"days_overdue"`
	EvaluatedAt      string  `json:"evaluated_at"`
}

type overdueControlResponse struct {
	FrameworkID       string `json:"framework_id"`
	FrameworkName     string `json:"framework_name"`
	ControlID         string `json:"control_id"`
	ControlName       string `json:"control_name"`
	Status            string `json:"status"`
	SlaDeadlineAt     string `json:"sla_deadline_at"`
	DaysOverdue       int32  `json:"days_overdue"`
	AffectedEndpoints int32  `json:"affected_endpoints"`
}

type categoryBreakdown struct {
	Category string                  `json:"category"`
	Controls []controlResultResponse `json:"controls"`
}

type nonCompliantEndpointResponse struct {
	EndpointID          string `json:"endpoint_id"`
	Hostname            string `json:"hostname"`
	OsFamily            string `json:"os_family"`
	Score               string `json:"score"`
	TotalCVEs           int32  `json:"total_cves"`
	CompliantCVEs       int32  `json:"compliant_cves"`
	AtRiskCVEs          int32  `json:"at_risk_cves"`
	NonCompliantCVEs    int32  `json:"non_compliant_cves"`
	LateRemediationCVEs int32  `json:"late_remediation_cves"`
}

// ------------------------------------------------------------
// Handlers
// ------------------------------------------------------------

// Summary handles GET /api/v1/compliance/summary.
func (h *ComplianceHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	enabledFrameworks, err := h.q.ListEnabledTenantFrameworks(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list enabled tenant frameworks", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list enabled frameworks")
		return
	}

	lastEval, err := h.q.GetLastEvaluationTime(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get last evaluation time", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get last evaluation time")
		return
	}

	fwSummaries := make([]frameworkScoreSummary, 0, len(enabledFrameworks))
	for _, tf := range enabledFrameworks {
		name, _, resolved := h.resolveFrameworkName(ctx, tf.FrameworkID, tid)
		if !resolved {
			name = tf.FrameworkID
		}

		item := frameworkScoreSummary{
			FrameworkID: tf.FrameworkID,
			Name:        name,
		}

		score, scoreErr := h.q.GetLatestFrameworkScore(ctx, sqlcgen.GetLatestFrameworkScoreParams{
			TenantID:    tid,
			FrameworkID: tf.FrameworkID,
		})
		if scoreErr == nil {
			scoreStr := numericToString(score.Score)
			evalAt := score.EvaluatedAt.Time.Format(time.RFC3339)
			item.Score = &scoreStr
			item.TotalCVEs = score.TotalCves
			item.Compliant = score.CompliantCves
			item.AtRisk = score.AtRiskCves
			item.NonCompliant = score.NonCompliantCves
			item.EvaluatedAt = &evalAt
		} else if !isNotFound(scoreErr) {
			slog.ErrorContext(ctx, "get latest framework score for summary", "framework_id", tf.FrameworkID, "tenant_id", tenantID, "error", scoreErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get framework score for "+tf.FrameworkID)
			return
		}

		fwSummaries = append(fwSummaries, item)
	}

	// Enrich with control counts per framework.
	controlCounts, ccErr := h.q.GetControlCountsByFramework(ctx, tid)
	if ccErr != nil {
		slog.ErrorContext(ctx, "get control counts by framework", "tenant_id", tenantID, "error", ccErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get control counts")
		return
	}
	ccIndex := make(map[string]sqlcgen.GetControlCountsByFrameworkRow, len(controlCounts))
	for _, cc := range controlCounts {
		ccIndex[cc.FrameworkID] = cc
	}

	// Enrich with compliant endpoint counts per framework.
	epCounts, epCErr := h.q.GetCompliantEndpointCountsByFramework(ctx, tid)
	if epCErr != nil {
		slog.ErrorContext(ctx, "get compliant endpoint counts by framework", "tenant_id", tenantID, "error", epCErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get endpoint counts")
		return
	}
	epIndex := make(map[string]sqlcgen.GetCompliantEndpointCountsByFrameworkRow, len(epCounts))
	for _, ep := range epCounts {
		epIndex[ep.FrameworkID] = ep
	}

	for i := range fwSummaries {
		fid := fwSummaries[i].FrameworkID
		if cc, ok := ccIndex[fid]; ok {
			fwSummaries[i].TotalControls = cc.TotalControls
			fwSummaries[i].PassingControls = cc.PassingControls
			fwSummaries[i].FailingControls = cc.FailingControls
			fwSummaries[i].NaControls = cc.NaControls
			fwSummaries[i].OverdueCount = cc.OverdueCount
		}
		if ep, ok := epIndex[fid]; ok {
			fwSummaries[i].EndpointsCompliant = ep.EndpointsCompliant
			fwSummaries[i].TotalEndpoints = ep.TotalEndpoints
		} else if cc, ok := ccIndex[fid]; ok && cc.MaxTotalEndpoints > 0 {
			// Non-SLA frameworks don't have per-endpoint scores; derive
			// the endpoint count from the control results instead.
			fwSummaries[i].TotalEndpoints = cc.MaxTotalEndpoints
		}
		// Derive status from score.
		if fwSummaries[i].Score != nil {
			fwSummaries[i].Status = deriveComplianceStatus(*fwSummaries[i].Score)
		}
	}

	var lastEvalStr *string
	if lastEval.Valid {
		s := lastEval.Time.Format(time.RFC3339)
		lastEvalStr = &s
	}

	WriteJSON(w, http.StatusOK, complianceSummaryResponse{
		Frameworks:      fwSummaries,
		LastEvaluatedAt: lastEvalStr,
	})
}

// ListFrameworks handles GET /api/v1/compliance/frameworks.
func (h *ComplianceHandler) ListFrameworks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	tenantConfigs, err := h.q.ListTenantFrameworks(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list tenant frameworks", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tenant frameworks")
		return
	}

	// Index tenant configs by framework_id for O(1) lookup.
	configIndex := make(map[string]sqlcgen.ComplianceTenantFramework, len(tenantConfigs))
	for _, tc := range tenantConfigs {
		configIndex[tc.FrameworkID] = tc
	}

	allFrameworks := compliance.ListFrameworks()
	builtInIDs := make(map[string]bool, len(allFrameworks))
	items := make([]frameworkListItem, 0, len(allFrameworks))
	for _, fw := range allFrameworks {
		builtInIDs[fw.ID] = true
		item := frameworkListItem{
			ID:                   fw.ID,
			Name:                 fw.Name,
			Version:              fw.Version,
			Description:          fw.Description,
			ApplicableIndustries: fw.ApplicableIndustries,
		}
		if tc, ok := configIndex[fw.ID]; ok {
			item.Enabled = tc.Enabled
			item.ScoringMethod = &tc.ScoringMethod
			cfgID := uuidToString(tc.ID)
			item.ConfigID = &cfgID
		}
		items = append(items, item)
	}

	// Append enabled custom frameworks (tenant configs whose framework_id
	// is not a built-in framework ID — i.e. custom framework UUIDs).
	for fwID, tc := range configIndex {
		if builtInIDs[fwID] {
			continue
		}
		cfgID := uuidToString(tc.ID)
		sm := tc.ScoringMethod
		items = append(items, frameworkListItem{
			ID:            fwID,
			Name:          fwID, // placeholder — frontend resolves custom names from custom-frameworks API
			Enabled:       tc.Enabled,
			ConfigID:      &cfgID,
			ScoringMethod: &sm,
		})
	}

	WriteJSON(w, http.StatusOK, items)
}

// EnableFramework handles POST /api/v1/compliance/frameworks.
func (h *ComplianceHandler) EnableFramework(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var req enableFrameworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.FrameworkID == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "framework_id is required")
		return
	}

	fw := compliance.GetFramework(req.FrameworkID)
	isCustom := false
	if fw == nil {
		// Check if it's a custom framework UUID.
		fwUUID, parseErr := scanUUID(req.FrameworkID)
		if parseErr == nil {
			_, dbErr := h.q.GetCustomFramework(ctx, sqlcgen.GetCustomFrameworkParams{ID: fwUUID, TenantID: tid})
			if dbErr == nil {
				isCustom = true
			}
		}
		if !isCustom {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "unknown framework_id: "+req.FrameworkID)
			return
		}
	}

	if req.AtRiskThreshold != nil && (*req.AtRiskThreshold <= 0.0 || *req.AtRiskThreshold > 1.0) {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "at_risk_threshold must be between 0.0 (exclusive) and 1.0 (inclusive)")
		return
	}

	scoringMethod := req.ScoringMethod
	if scoringMethod == "" {
		if fw != nil {
			scoringMethod = fw.DefaultScoringMethod
		} else {
			scoringMethod = "average"
		}
	}

	if scoringMethod != "strictest" && scoringMethod != "average" && scoringMethod != "worst_case" && scoringMethod != "weighted" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "scoring_method must be one of: strictest, average, worst_case, weighted")
		return
	}

	// Use canonical framework ID for built-in frameworks to prevent duplicates
	// from differing casing (e.g. "cis" vs "CIS").
	frameworkIDForDB := req.FrameworkID
	if fw != nil {
		frameworkIDForDB = fw.ID
	}

	// Check if tenant already has this framework enabled (by canonical ID).
	_, existErr := h.q.GetTenantFramework(ctx, sqlcgen.GetTenantFrameworkParams{
		TenantID:    tid,
		FrameworkID: frameworkIDForDB,
	})
	if existErr == nil {
		WriteError(w, http.StatusConflict, "ALREADY_ENABLED", "framework is already enabled for this tenant")
		return
	}

	slaOverrides := []byte("{}")
	if len(req.SlaOverrides) > 0 {
		slaOverrides = req.SlaOverrides
	}

	var atRiskThreshold pgtype.Numeric
	if req.AtRiskThreshold != nil {
		atRiskThreshold = float64ToNumeric(*req.AtRiskThreshold)
	} else {
		atRiskThreshold = float64ToNumeric(0.75) // default: 75% at-risk threshold
	}

	created, err := h.q.CreateTenantFramework(ctx, sqlcgen.CreateTenantFrameworkParams{
		TenantID:        tid,
		FrameworkID:     frameworkIDForDB,
		Enabled:         true,
		SlaOverrides:    slaOverrides,
		ScoringMethod:   scoringMethod,
		AtRiskThreshold: atRiskThreshold,
	})
	if err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "ALREADY_EXISTS", "framework already enabled for this tenant")
			return
		}
		slog.ErrorContext(ctx, "create tenant framework", "framework_id", req.FrameworkID, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enable framework")
		return
	}

	emitEvent(ctx, h.eventBus, events.ComplianceFrameworkEnabled, "compliance_framework", uuidToString(created.ID), tenantID, map[string]string{
		"framework_id": frameworkIDForDB,
	})

	WriteJSON(w, http.StatusCreated, mapTenantFrameworkResponse(created))
}

// GetFrameworkDetail handles GET /api/v1/compliance/frameworks/{frameworkId}.
func (h *ComplianceHandler) GetFrameworkDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	frameworkID := chi.URLParam(r, "frameworkId")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	fwName, fwVersion, fwFound := h.resolveFrameworkName(ctx, frameworkID, tid)
	if !fwFound {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "framework not found: "+frameworkID)
		return
	}
	fw := compliance.GetFramework(frameworkID) // may be nil for custom — that's ok

	// Use canonical framework ID for DB queries to avoid alias mismatches.
	canonicalFwID := frameworkID
	if fw != nil {
		canonicalFwID = fw.ID
	}

	// For custom frameworks, load controls from DB so we can populate
	// name/description in control result responses.
	type customCtrlInfo struct{ Name, Description string }
	customCtrlMap := map[string]customCtrlInfo{}
	if fw == nil {
		fwUUID, _ := scanUUID(frameworkID)
		if ctrls, err := h.q.ListCustomControls(ctx, sqlcgen.ListCustomControlsParams{TenantID: tid, FrameworkID: fwUUID}); err == nil {
			for _, c := range ctrls {
				desc := ""
				if c.Description.Valid {
					desc = c.Description.String
				}
				customCtrlMap[c.ControlID] = customCtrlInfo{Name: c.Name, Description: desc}
			}
		}
	}

	fwDesc := ""
	var fwIndustries []string
	if fw != nil {
		fwDesc = fw.Description
		fwIndustries = fw.ApplicableIndustries
	}

	item := frameworkListItem{
		ID:                   canonicalFwID,
		Name:                 fwName,
		Version:              fwVersion,
		Description:          fwDesc,
		ApplicableIndustries: fwIndustries,
	}

	resp := frameworkDetailResponse{
		Framework:      item,
		EndpointScores: []endpointScoreResponse{},
	}

	// Look up tenant config for this framework.
	tc, tcErr := h.q.GetTenantFramework(ctx, sqlcgen.GetTenantFrameworkParams{
		TenantID:    tid,
		FrameworkID: canonicalFwID,
	})
	if tcErr == nil {
		item.Enabled = tc.Enabled
		item.ScoringMethod = &tc.ScoringMethod
		cfgID := uuidToString(tc.ID)
		item.ConfigID = &cfgID
		resp.Framework = item

		config := mapTenantFrameworkResponse(tc)
		resp.Config = &config
	} else if !isNotFound(tcErr) {
		slog.ErrorContext(ctx, "get tenant framework config", "framework_id", canonicalFwID, "tenant_id", tenantID, "error", tcErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get framework configuration")
		return
	}

	// Get latest tenant-level score.
	score, scoreErr := h.q.GetLatestFrameworkScore(ctx, sqlcgen.GetLatestFrameworkScoreParams{
		TenantID:    tid,
		FrameworkID: canonicalFwID,
	})
	if scoreErr == nil {
		s := mapScoreResponse(score)
		resp.Score = &s
	} else if !isNotFound(scoreErr) {
		slog.ErrorContext(ctx, "get latest framework score", "framework_id", canonicalFwID, "tenant_id", tenantID, "error", scoreErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get framework score")
		return
	}

	// Get per-endpoint scores (sorted worst first).
	endpointScores, epErr := h.q.ListEndpointScoresByFramework(ctx, sqlcgen.ListEndpointScoresByFrameworkParams{
		TenantID:    tid,
		FrameworkID: canonicalFwID,
	})
	if epErr == nil {
		epItems := make([]endpointScoreResponse, len(endpointScores))
		for i, es := range endpointScores {
			epItems[i] = endpointScoreResponse{
				EndpointID:          uuidToString(es.ScopeID),
				Score:               numericToString(es.Score),
				TotalCVEs:           es.TotalCves,
				CompliantCVEs:       es.CompliantCves,
				AtRiskCVEs:          es.AtRiskCves,
				NonCompliantCVEs:    es.NonCompliantCves,
				LateRemediationCVEs: es.LateRemediationCves,
				EvaluatedAt:         es.EvaluatedAt.Time.Format(time.RFC3339),
			}
		}
		resp.EndpointScores = epItems
	} else if !isNotFound(epErr) {
		slog.ErrorContext(ctx, "list endpoint scores by framework", "framework_id", canonicalFwID, "tenant_id", tenantID, "error", epErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoint scores")
		return
	}

	// Get control results grouped by category.
	controlResults, crErr := h.q.ListControlResultsByFramework(ctx, sqlcgen.ListControlResultsByFrameworkParams{
		TenantID:    tid,
		FrameworkID: canonicalFwID,
	})
	if crErr == nil {
		categoryMap := make(map[string][]controlResultResponse)
		var categoryOrder []string
		for _, cr := range controlResults {
			item := mapControlResultResponse(cr, fw)
			// Enrich with custom control name/description when built-in framework is nil.
			if fw == nil {
				if cc, ok := customCtrlMap[cr.ControlID]; ok {
					if cc.Name != "" {
						item.Name = cc.Name
					}
					if cc.Description != "" {
						item.Description = cc.Description
					}
				}
			}
			if _, exists := categoryMap[item.Category]; !exists {
				categoryOrder = append(categoryOrder, item.Category)
			}
			categoryMap[item.Category] = append(categoryMap[item.Category], item)
		}
		categories := make([]categoryBreakdown, 0, len(categoryOrder))
		for _, cat := range categoryOrder {
			categories = append(categories, categoryBreakdown{
				Category: cat,
				Controls: categoryMap[cat],
			})
		}
		resp.Categories = categories
	} else if isNotFound(crErr) {
		slog.DebugContext(ctx, "no control results found for framework", "framework_id", canonicalFwID)
	} else {
		slog.ErrorContext(ctx, "list control results by framework", "framework_id", canonicalFwID, "tenant_id", tenantID, "error", crErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list control results")
		return
	}

	// Get non-compliant endpoints.
	ncEndpoints, ncErr := h.q.ListNonCompliantEndpointsByFramework(ctx, sqlcgen.ListNonCompliantEndpointsByFrameworkParams{
		TenantID:    tid,
		FrameworkID: canonicalFwID,
	})
	if ncErr == nil {
		ncItems := make([]nonCompliantEndpointResponse, len(ncEndpoints))
		for i, ep := range ncEndpoints {
			ncItems[i] = nonCompliantEndpointResponse{
				EndpointID:          uuidToString(ep.EndpointID),
				Hostname:            ep.Hostname,
				OsFamily:            ep.OsFamily,
				Score:               numericToString(ep.Score),
				TotalCVEs:           ep.TotalCves,
				CompliantCVEs:       ep.CompliantCves,
				AtRiskCVEs:          ep.AtRiskCves,
				NonCompliantCVEs:    ep.NonCompliantCves,
				LateRemediationCVEs: ep.LateRemediationCves,
			}
		}
		resp.NonCompliantEndpoints = ncItems
	} else if isNotFound(ncErr) {
		slog.DebugContext(ctx, "no non-compliant endpoints found for framework", "framework_id", canonicalFwID)
	} else {
		slog.ErrorContext(ctx, "list non-compliant endpoints by framework", "framework_id", canonicalFwID, "tenant_id", tenantID, "error", ncErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list non-compliant endpoints")
		return
	}

	if resp.Categories == nil {
		resp.Categories = []categoryBreakdown{}
	}
	if resp.NonCompliantEndpoints == nil {
		resp.NonCompliantEndpoints = []nonCompliantEndpointResponse{}
	}

	WriteJSON(w, http.StatusOK, resp)
}

// UpdateFramework handles PUT /api/v1/compliance/frameworks/{id}.
func (h *ComplianceHandler) UpdateFramework(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	configID, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid framework config ID")
		return
	}

	// Fetch existing config to use as defaults for partial update.
	existing, err := h.q.GetTenantFrameworkByID(ctx, sqlcgen.GetTenantFrameworkByIDParams{
		ID:       configID,
		TenantID: tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "framework configuration not found")
			return
		}
		slog.ErrorContext(ctx, "get tenant framework by ID for update", "config_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get framework configuration")
		return
	}

	var req updateFrameworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.AtRiskThreshold != nil && (*req.AtRiskThreshold <= 0.0 || *req.AtRiskThreshold > 1.0) {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "at_risk_threshold must be between 0.0 (exclusive) and 1.0 (inclusive)")
		return
	}

	if req.ScoringMethod != "" && req.ScoringMethod != "strictest" && req.ScoringMethod != "average" && req.ScoringMethod != "worst_case" && req.ScoringMethod != "weighted" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "scoring_method must be one of: strictest, average, worst_case, weighted")
		return
	}

	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	scoringMethod := existing.ScoringMethod
	if req.ScoringMethod != "" {
		scoringMethod = req.ScoringMethod
	}

	slaOverrides := existing.SlaOverrides
	if len(req.SlaOverrides) > 0 {
		slaOverrides = req.SlaOverrides
	}

	atRiskThreshold := existing.AtRiskThreshold
	if req.AtRiskThreshold != nil {
		atRiskThreshold = float64ToNumeric(*req.AtRiskThreshold)
	}

	updated, err := h.q.UpdateTenantFramework(ctx, sqlcgen.UpdateTenantFrameworkParams{
		ID:              configID,
		TenantID:        tid,
		Enabled:         enabled,
		ScoringMethod:   scoringMethod,
		SlaOverrides:    slaOverrides,
		AtRiskThreshold: atRiskThreshold,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "framework configuration not found")
			return
		}
		slog.ErrorContext(ctx, "update tenant framework", "config_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update framework configuration")
		return
	}

	emitEvent(ctx, h.eventBus, events.ComplianceFrameworkUpdated, "compliance_framework", uuidToString(updated.ID), tenantID, map[string]string{
		"framework_id": updated.FrameworkID,
	})

	WriteJSON(w, http.StatusOK, mapTenantFrameworkResponse(updated))
}

// DisableFramework handles DELETE /api/v1/compliance/frameworks/{id}.
func (h *ComplianceHandler) DisableFramework(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	configID, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid framework config ID")
		return
	}

	// Fetch existing to get framework_id for the event payload.
	existing, err := h.q.GetTenantFrameworkByID(ctx, sqlcgen.GetTenantFrameworkByIDParams{
		ID:       configID,
		TenantID: tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "framework configuration not found")
			return
		}
		slog.ErrorContext(ctx, "get tenant framework for disable", "config_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get framework configuration")
		return
	}

	err = h.q.DeleteTenantFramework(ctx, sqlcgen.DeleteTenantFrameworkParams{
		ID:       configID,
		TenantID: tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "delete tenant framework", "config_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to disable framework")
		return
	}

	emitEvent(ctx, h.eventBus, events.ComplianceFrameworkDisabled, "compliance_framework", uuidToString(configID), tenantID, map[string]string{
		"framework_id": existing.FrameworkID,
	})

	w.WriteHeader(http.StatusNoContent)
}

// GetEndpointCompliance handles GET /api/v1/compliance/endpoints/{id}.
func (h *ComplianceHandler) GetEndpointCompliance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	endpointID, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID")
		return
	}

	evals, err := h.q.ListEvaluationsByEndpoint(ctx, sqlcgen.ListEvaluationsByEndpointParams{
		TenantID:   tid,
		EndpointID: endpointID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list evaluations by endpoint", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoint evaluations")
		return
	}

	items := make([]endpointEvaluationResponse, len(evals))
	for i, e := range evals {
		items[i] = endpointEvaluationResponse{
			ID:          uuidToString(e.ID),
			FrameworkID: e.FrameworkID,
			ControlID:   e.ControlID,
			CveID:       e.CveID,
			State:       e.State,
			EvaluatedAt: e.EvaluatedAt.Time.Format(time.RFC3339),
		}
		if e.SlaDeadlineAt.Valid {
			s := e.SlaDeadlineAt.Time.Format(time.RFC3339)
			items[i].SlaDeadlineAt = &s
		}
		if e.RemediatedAt.Valid {
			s := e.RemediatedAt.Time.Format(time.RFC3339)
			items[i].RemediatedAt = &s
		}
		if e.DaysRemaining.Valid {
			d := e.DaysRemaining.Int32
			items[i].DaysRemaining = &d
		}
	}

	WriteJSON(w, http.StatusOK, items)
}

type triggerEvaluationResponse struct {
	Status              string `json:"status"`
	FrameworksEvaluated int    `json:"frameworks_evaluated"`
	TotalEvaluations    int    `json:"total_evaluations"`
}

// TriggerEvaluation handles POST /api/v1/compliance/evaluate.
func (h *ComplianceHandler) TriggerEvaluation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	// Begin transaction to set app.current_tenant_id for RLS policies.
	tx, err := h.store.BeginTx(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin evaluation transaction", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to start evaluation")
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err.Error() != "tx is closed" {
			slog.WarnContext(ctx, "compliance: rollback after evaluation", "error", err)
		}
	}()

	txQ := sqlcgen.New(tx)
	result, err := h.svc.RunEvaluation(ctx, tid, txQ)
	if err != nil {
		slog.ErrorContext(ctx, "trigger compliance evaluation", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "EVALUATION_FAILED", "compliance evaluation failed")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit evaluation transaction", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit evaluation results")
		return
	}

	if result.RunID != "" {
		emitEvent(ctx, h.eventBus, events.ComplianceEvaluationCompleted, "compliance_evaluation", result.RunID, tenantID, map[string]string{
			"frameworks_evaluated": fmt.Sprintf("%d", result.FrameworksEvaluated),
			"total_evaluations":    fmt.Sprintf("%d", result.TotalEvaluations),
		})
	}

	WriteJSON(w, http.StatusOK, triggerEvaluationResponse{
		Status:              "completed",
		FrameworksEvaluated: result.FrameworksEvaluated,
		TotalEvaluations:    result.TotalEvaluations,
	})
}

// TriggerFrameworkEvaluation handles POST /api/v1/compliance/frameworks/{frameworkId}/evaluate.
func (h *ComplianceHandler) TriggerFrameworkEvaluation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	frameworkID := chi.URLParam(r, "frameworkId")

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	tx, err := h.store.BeginTx(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin framework evaluation transaction", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to start evaluation")
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err.Error() != "tx is closed" {
			slog.WarnContext(ctx, "compliance: rollback after framework evaluation", "error", err)
		}
	}()

	txQ := sqlcgen.New(tx)
	result, err := h.svc.RunFrameworkEvaluation(ctx, tid, frameworkID, txQ)
	if err != nil {
		slog.ErrorContext(ctx, "trigger framework evaluation", "tenant_id", tenantID, "framework_id", frameworkID, "error", err)
		WriteError(w, http.StatusInternalServerError, "EVALUATION_FAILED", "framework evaluation failed")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit framework evaluation transaction", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit evaluation results")
		return
	}

	if result.RunID != "" {
		emitEvent(ctx, h.eventBus, events.ComplianceEvaluationCompleted, "compliance_evaluation", result.RunID, tenantID, map[string]string{
			"framework_id":         frameworkID,
			"frameworks_evaluated": fmt.Sprintf("%d", result.FrameworksEvaluated),
			"total_evaluations":    fmt.Sprintf("%d", result.TotalEvaluations),
		})
	}

	WriteJSON(w, http.StatusOK, triggerEvaluationResponse{
		Status:              "completed",
		FrameworksEvaluated: result.FrameworksEvaluated,
		TotalEvaluations:    result.TotalEvaluations,
	})
}

// GetOverallScore handles GET /api/v1/compliance/score.
func (h *ComplianceHandler) GetOverallScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	row, err := h.q.GetOverallComplianceScore(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get overall compliance score", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get overall compliance score")
		return
	}

	scoreVal := numericToFloat64(row.OverallScore)
	status := deriveComplianceStatusFromFloat(scoreVal)

	lastEval, err := h.q.GetLastEvaluationTime(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get last evaluation time for overall score", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get last evaluation time")
		return
	}

	var lastEvalStr *string
	if lastEval.Valid {
		s := lastEval.Time.Format(time.RFC3339)
		lastEvalStr = &s
	}

	WriteJSON(w, http.StatusOK, overallScoreResponse{
		OverallScore:     numericToString(row.OverallScore),
		TotalCVEs:        row.TotalCves,
		CompliantCVEs:    row.CompliantCves,
		AtRiskCVEs:       row.AtRiskCves,
		NonCompliantCVEs: row.NonCompliantCves,
		FrameworkCount:   row.FrameworkCount,
		Status:           status,
		LastEvaluatedAt:  lastEvalStr,
	})
}

// ListFrameworkControls handles GET /api/v1/compliance/frameworks/{frameworkId}/controls.
func (h *ComplianceHandler) ListFrameworkControls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	frameworkID := chi.URLParam(r, "frameworkId")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	_, _, ctrlFwOK := h.resolveFrameworkName(ctx, frameworkID, tid)
	if !ctrlFwOK {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "framework not found: "+frameworkID)
		return
	}
	fw := compliance.GetFramework(frameworkID) // nil for custom — mapControlResultResponse handles this

	// Use canonical framework ID for DB queries to avoid alias mismatches.
	canonicalFwID := frameworkID
	if fw != nil {
		canonicalFwID = fw.ID
	}

	// For custom frameworks, load controls from DB for name/description enrichment.
	type customCtrlInfo2 struct{ Name, Description string }
	customCtrlLookup := map[string]customCtrlInfo2{}
	if fw == nil {
		fwUUID, _ := scanUUID(frameworkID)
		if ctrls, err := h.q.ListCustomControls(ctx, sqlcgen.ListCustomControlsParams{TenantID: tid, FrameworkID: fwUUID}); err == nil {
			for _, c := range ctrls {
				desc := ""
				if c.Description.Valid {
					desc = c.Description.String
				}
				customCtrlLookup[c.ControlID] = customCtrlInfo2{Name: c.Name, Description: desc}
			}
		}
	}

	results, err := h.q.ListControlResultsByFramework(ctx, sqlcgen.ListControlResultsByFrameworkParams{
		TenantID:    tid,
		FrameworkID: canonicalFwID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list control results by framework", "framework_id", canonicalFwID, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list control results")
		return
	}

	statusFilter := r.URL.Query().Get("status")
	searchFilter := strings.ToLower(r.URL.Query().Get("search"))

	items := make([]controlResultResponse, 0, len(results))
	for _, cr := range results {
		item := mapControlResultResponse(cr, fw)
		if fw == nil {
			if cc, ok := customCtrlLookup[cr.ControlID]; ok {
				if cc.Name != "" {
					item.Name = cc.Name
				}
				if cc.Description != "" {
					item.Description = cc.Description
				}
			}
		}

		if statusFilter != "" && item.Status != statusFilter {
			continue
		}
		if searchFilter != "" && !strings.Contains(strings.ToLower(item.ControlID), searchFilter) && !strings.Contains(strings.ToLower(item.Name), searchFilter) {
			continue
		}

		items = append(items, item)
	}

	WriteJSON(w, http.StatusOK, items)
}

// ListOverdueControls handles GET /api/v1/compliance/overdue.
func (h *ComplianceHandler) ListOverdueControls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit := int32(50)
	offset := int32(0)
	if v, err := strconv.ParseInt(limitStr, 10, 32); err == nil && v > 0 && v <= 200 {
		limit = int32(v)
	}
	if v, err := strconv.ParseInt(offsetStr, 10, 32); err == nil && v >= 0 {
		offset = int32(v)
	}

	controls, err := h.q.ListOverdueControls(ctx, sqlcgen.ListOverdueControlsParams{
		TenantID:     tid,
		ResultLimit:  limit,
		ResultOffset: offset,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list overdue controls", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list overdue controls")
		return
	}

	items := make([]overdueControlResponse, 0, len(controls))
	for _, cr := range controls {
		fwName := cr.FrameworkID
		ctrlName := cr.ControlID

		// Resolve framework name — works for both built-in and custom frameworks
		if name, _, found := h.resolveFrameworkName(ctx, cr.FrameworkID, tid); found {
			fwName = name
		}

		// Resolve control name — try built-in registry first
		if fw := compliance.GetFramework(cr.FrameworkID); fw != nil {
			if ctrl := fw.GetControl(cr.ControlID); ctrl != nil {
				ctrlName = ctrl.Name
			}
		} else {
			// Custom framework — look up control name from database
			fwUUID, fwErr := scanUUID(cr.FrameworkID)
			if fwErr == nil {
				ctrls, cErr := h.q.ListCustomControls(ctx, sqlcgen.ListCustomControlsParams{
					FrameworkID: fwUUID,
					TenantID:    tid,
				})
				if cErr == nil {
					for _, cc := range ctrls {
						if cc.ControlID == cr.ControlID {
							ctrlName = cc.Name
							break
						}
					}
				}
			}
		}

		var daysOverdue int32
		if cr.DaysOverdue.Valid {
			daysOverdue = cr.DaysOverdue.Int32
		}

		var slaStr string
		if cr.SlaDeadlineAt.Valid {
			slaStr = cr.SlaDeadlineAt.Time.Format(time.RFC3339)
		}

		items = append(items, overdueControlResponse{
			FrameworkID:       cr.FrameworkID,
			FrameworkName:     fwName,
			ControlID:         cr.ControlID,
			ControlName:       ctrlName,
			Status:            cr.Status,
			SlaDeadlineAt:     slaStr,
			DaysOverdue:       daysOverdue,
			AffectedEndpoints: cr.TotalEndpoints - cr.PassingEndpoints,
		})
	}

	WriteJSON(w, http.StatusOK, items)
}

// GetFrameworkTrend handles GET /api/v1/compliance/frameworks/{frameworkId}/trend.
func (h *ComplianceHandler) GetFrameworkTrend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	frameworkID := chi.URLParam(r, "frameworkId")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	_, _, trendFwOK := h.resolveFrameworkName(ctx, frameworkID, tid)
	if !trendFwOK {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "framework not found: "+frameworkID)
		return
	}

	// Use canonical framework ID for DB queries to avoid alias mismatches.
	canonicalFwID := frameworkID
	if fw := compliance.GetFramework(frameworkID); fw != nil {
		canonicalFwID = fw.ID
	}

	scores, err := h.q.ListScoreTrend(ctx, sqlcgen.ListScoreTrendParams{
		TenantID:    tid,
		FrameworkID: canonicalFwID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list score trend", "framework_id", frameworkID, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list score trend")
		return
	}

	type trendPoint struct {
		Score       string `json:"score"`
		EvaluatedAt string `json:"evaluated_at"`
	}

	items := make([]trendPoint, len(scores))
	for i, s := range scores {
		items[i] = trendPoint{
			Score:       numericToString(s.Score),
			EvaluatedAt: s.EvaluatedAt.Time.Format(time.RFC3339),
		}
	}

	WriteJSON(w, http.StatusOK, items)
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func mapControlResultResponse(cr sqlcgen.ComplianceControlResult, fw *compliance.Framework) controlResultResponse {
	item := controlResultResponse{
		ControlID:        cr.ControlID,
		Name:             cr.ControlID,
		Description:      "",
		Category:         cr.Category,
		Status:           cr.Status,
		PassingEndpoints: cr.PassingEndpoints,
		TotalEndpoints:   cr.TotalEndpoints,
		EvaluatedAt:      cr.EvaluatedAt.Time.Format(time.RFC3339),
	}

	if fw != nil {
		if ctrl := fw.GetControl(cr.ControlID); ctrl != nil {
			item.Name = ctrl.Name
			item.Description = ctrl.Description
		}
	}

	if cr.RemediationHint.Valid {
		item.RemediationHint = &cr.RemediationHint.String
	}
	if cr.SlaDeadlineAt.Valid {
		s := cr.SlaDeadlineAt.Time.Format(time.RFC3339)
		item.SlaDeadlineAt = &s
	}
	if cr.DaysOverdue.Valid {
		d := cr.DaysOverdue.Int32
		item.DaysOverdue = &d
	}

	return item
}

func numericToFloat64(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil {
		slog.Error("failed to convert numeric to float64", "error", err)
		return 0
	}
	if f.Valid {
		return f.Float64
	}
	return 0
}

func deriveComplianceStatusFromFloat(score float64) string {
	switch {
	case score >= 95:
		return "compliant"
	case score >= 80:
		return "needs_improvement"
	default:
		return "non_compliant"
	}
}

func deriveComplianceStatus(scoreStr string) string {
	var score float64
	if _, err := fmt.Sscanf(scoreStr, "%f", &score); err != nil {
		slog.Error("failed to parse compliance score string", "score", scoreStr, "error", err)
		return "non_compliant"
	}
	return deriveComplianceStatusFromFloat(score)
}

func mapTenantFrameworkResponse(tf sqlcgen.ComplianceTenantFramework) tenantFrameworkResponse {
	slaOverrides := json.RawMessage(tf.SlaOverrides)
	if len(slaOverrides) == 0 {
		slaOverrides = json.RawMessage("{}")
	}

	var atRisk *string
	if tf.AtRiskThreshold.Valid {
		s := numericToString(tf.AtRiskThreshold)
		atRisk = &s
	}

	return tenantFrameworkResponse{
		ID:              uuidToString(tf.ID),
		FrameworkID:     tf.FrameworkID,
		Enabled:         tf.Enabled,
		SlaOverrides:    slaOverrides,
		ScoringMethod:   tf.ScoringMethod,
		AtRiskThreshold: atRisk,
		CreatedAt:       tf.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       tf.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func mapScoreResponse(s sqlcgen.ComplianceScore) scoreResponse {
	return scoreResponse{
		Score:               numericToString(s.Score),
		TotalCVEs:           s.TotalCves,
		CompliantCVEs:       s.CompliantCves,
		AtRiskCVEs:          s.AtRiskCves,
		NonCompliantCVEs:    s.NonCompliantCves,
		LateRemediationCVEs: s.LateRemediationCves,
		EvaluatedAt:         s.EvaluatedAt.Time.Format(time.RFC3339),
	}
}

// numericToString converts a pgtype.Numeric to its string representation.
func numericToString(n pgtype.Numeric) string {
	if !n.Valid || n.Int == nil {
		return "0"
	}
	// Reconstruct the decimal: value = Int * 10^Exp
	f := new(big.Float).SetInt(n.Int)
	if n.Exp > 0 {
		mul := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil)
		f.Mul(f, new(big.Float).SetInt(mul))
	} else if n.Exp < 0 {
		div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n.Exp)), nil)
		f.Quo(f, new(big.Float).SetInt(div))
	}
	return f.Text('f', -1)
}

// float64ToNumeric converts a float64 to a pgtype.Numeric.
func float64ToNumeric(f float64) pgtype.Numeric {
	// Represent as integer with exponent: multiply by 100 for 2 decimal places.
	intVal := int64(math.Round(f * 100))
	return pgtype.Numeric{
		Int:   big.NewInt(intVal),
		Exp:   -2,
		Valid: true,
	}
}
