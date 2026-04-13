package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/policy"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/server/targeting"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// PolicyQuerier defines the sqlc queries needed by PolicyHandler.
// As of Phase 2 (tags-replace-groups), endpoint targeting is delegated
// to targeting.Resolver rather than queried via policy_groups.
type PolicyQuerier interface {
	CreatePolicy(ctx context.Context, arg sqlcgen.CreatePolicyParams) (sqlcgen.Policy, error)
	GetPolicyByID(ctx context.Context, arg sqlcgen.GetPolicyByIDParams) (sqlcgen.Policy, error)
	ListPolicies(ctx context.Context, arg sqlcgen.ListPoliciesParams) ([]sqlcgen.Policy, error)
	CountPolicies(ctx context.Context, arg sqlcgen.CountPoliciesParams) (int64, error)
	UpdatePolicy(ctx context.Context, arg sqlcgen.UpdatePolicyParams) (sqlcgen.Policy, error)
	SoftDeletePolicy(ctx context.Context, arg sqlcgen.SoftDeletePolicyParams) (sqlcgen.Policy, error)
	ListPoliciesWithStats(ctx context.Context, arg sqlcgen.ListPoliciesWithStatsParams) ([]sqlcgen.ListPoliciesWithStatsRow, error)
	CountPoliciesFiltered(ctx context.Context, arg sqlcgen.CountPoliciesFilteredParams) (int64, error)
	BulkUpdatePolicyEnabled(ctx context.Context, arg sqlcgen.BulkUpdatePolicyEnabledParams) error
	BulkSoftDeletePolicies(ctx context.Context, arg sqlcgen.BulkSoftDeletePoliciesParams) error
	ListPolicyEvaluations(ctx context.Context, arg sqlcgen.ListPolicyEvaluationsParams) ([]sqlcgen.PolicyEvaluation, error)
	CreatePolicyEvaluation(ctx context.Context, arg sqlcgen.CreatePolicyEvaluationParams) (sqlcgen.PolicyEvaluation, error)
	UpdatePolicyEvalStats(ctx context.Context, arg sqlcgen.UpdatePolicyEvalStatsParams) error
	CountDeploymentsForPolicy(ctx context.Context, arg sqlcgen.CountDeploymentsForPolicyParams) (int64, error)
	ListDeploymentsForPolicy(ctx context.Context, arg sqlcgen.ListDeploymentsForPolicyParams) ([]sqlcgen.Deployment, error)
	ListEndpointsByIDs(ctx context.Context, arg sqlcgen.ListEndpointsByIDsParams) ([]sqlcgen.ListEndpointsByIDsRow, error)
	UpsertPolicyTagSelector(ctx context.Context, arg sqlcgen.UpsertPolicyTagSelectorParams) (sqlcgen.PolicyTagSelector, error)
	GetPolicyTagSelector(ctx context.Context, arg sqlcgen.GetPolicyTagSelectorParams) (sqlcgen.PolicyTagSelector, error)
	DeletePolicyTagSelector(ctx context.Context, arg sqlcgen.DeletePolicyTagSelectorParams) error
}

// PolicyEndpointResolver returns endpoint UUIDs from a policy's selector,
// letting tests substitute a stub. Implemented by targeting.Resolver.
type PolicyEndpointResolver interface {
	ResolveForPolicy(ctx context.Context, tenantID, policyID string) ([]uuid.UUID, error)
	Count(ctx context.Context, tenantID string, sel *targeting.Selector) (int, error)
}

// PolicyEvaluator evaluates a policy against its targeted endpoints.
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, tenantID, policyID string, now time.Time) ([]policy.EvaluationResult, error)
}

// PolicyHandler serves policy REST API endpoints.
type PolicyHandler struct {
	q         PolicyQuerier
	txb       TxBeginner
	eventBus  domain.EventBus
	evaluator PolicyEvaluator
	resolver  PolicyEndpointResolver
}

// NewPolicyHandler creates a PolicyHandler. resolver may be nil in tests
// that never exercise code paths surfacing matched endpoint counts.
func NewPolicyHandler(q PolicyQuerier, txb TxBeginner, eventBus domain.EventBus, evaluator PolicyEvaluator, resolver PolicyEndpointResolver) *PolicyHandler {
	if q == nil {
		panic("policies: NewPolicyHandler called with nil querier")
	}
	if txb == nil {
		panic("policies: NewPolicyHandler called with nil txBeginner")
	}
	if eventBus == nil {
		panic("policies: NewPolicyHandler called with nil eventBus")
	}
	if evaluator == nil {
		panic("policies: NewPolicyHandler called with nil evaluator")
	}
	return &PolicyHandler{q: q, txb: txb, eventBus: eventBus, evaluator: evaluator, resolver: resolver}
}

// policyResponse is the JSON shape returned for policies.
// It wraps sqlcgen.Policy but formats pgtype.Time fields as "HH:MM" strings.
// `target_selector` replaces the legacy `group_ids`/`group_names` fields
// as the policy's endpoint scope.
type policyResponse struct {
	ID                     pgtype.UUID         `json:"id"`
	TenantID               pgtype.UUID         `json:"tenant_id"`
	Name                   string              `json:"name"`
	Description            pgtype.Text         `json:"description"`
	Enabled                bool                `json:"enabled"`
	Mode                   string              `json:"mode"`
	CreatedAt              pgtype.Timestamptz  `json:"created_at"`
	UpdatedAt              pgtype.Timestamptz  `json:"updated_at"`
	SelectionMode          string              `json:"selection_mode"`
	TargetSelector         *targeting.Selector `json:"target_selector"`
	MinSeverity            pgtype.Text         `json:"min_severity"`
	CveIds                 []string            `json:"cve_ids"`
	PackageRegex           pgtype.Text         `json:"package_regex"`
	ExcludePackages        []string            `json:"exclude_packages"`
	ScheduleType           string              `json:"schedule_type"`
	ScheduleCron           pgtype.Text         `json:"schedule_cron"`
	MwStart                *string             `json:"mw_start"`
	MwEnd                  *string             `json:"mw_end"`
	DeploymentStrategy     string              `json:"deployment_strategy"`
	DeletedAt              pgtype.Timestamptz  `json:"deleted_at"`
	SeverityFilter         []string            `json:"severity_filter"`
	TargetEndpointsCount   int                 `json:"target_endpoints_count"`
	LastEvaluatedAt        pgtype.Timestamptz  `json:"last_evaluated_at"`
	LastEvalPass           pgtype.Bool         `json:"last_eval_pass"`
	LastEvalEndpointCount  pgtype.Int4         `json:"last_eval_endpoint_count"`
	LastEvalCompliantCount pgtype.Int4         `json:"last_eval_compliant_count"`
	PolicyType             string              `json:"policy_type"`
	Timezone               string              `json:"timezone"`
	MwEnabled              bool                `json:"mw_enabled"`
}

// formatTimeOfDay converts pgtype.Time to "HH:MM" string pointer (nil if invalid).
func formatTimeOfDay(t pgtype.Time) *string {
	if !t.Valid {
		return nil
	}
	totalMinutes := t.Microseconds / 60_000_000
	h := totalMinutes / 60
	m := totalMinutes % 60
	s := fmt.Sprintf("%02d:%02d", h, m)
	return &s
}

// toPolicyResponse converts a sqlcgen.Policy to the API response shape.
func toPolicyResponse(p sqlcgen.Policy, selector *targeting.Selector) policyResponse {
	return policyResponse{
		ID:                     p.ID,
		TenantID:               p.TenantID,
		Name:                   p.Name,
		Description:            p.Description,
		Enabled:                p.Enabled,
		Mode:                   p.Mode,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
		SelectionMode:          p.SelectionMode,
		TargetSelector:         selector,
		MinSeverity:            p.MinSeverity,
		CveIds:                 p.CveIds,
		PackageRegex:           p.PackageRegex,
		ExcludePackages:        p.ExcludePackages,
		ScheduleType:           p.ScheduleType,
		ScheduleCron:           p.ScheduleCron,
		MwStart:                formatTimeOfDay(p.MwStart),
		MwEnd:                  formatTimeOfDay(p.MwEnd),
		DeploymentStrategy:     p.DeploymentStrategy,
		DeletedAt:              p.DeletedAt,
		SeverityFilter:         p.SeverityFilter,
		LastEvaluatedAt:        p.LastEvaluatedAt,
		LastEvalPass:           p.LastEvalPass,
		LastEvalEndpointCount:  p.LastEvalEndpointCount,
		LastEvalCompliantCount: p.LastEvalCompliantCount,
		PolicyType:             p.PolicyType,
		Timezone:               p.Timezone,
		MwEnabled:              p.MwEnabled,
	}
}

// toPolicyResponseWithStats converts a ListPoliciesWithStatsRow to the API response shape.
// targetEndpointsCount is supplied separately because the stats row's
// column is now hardcoded to 0 — the authoritative count comes from the
// tag selector resolver, which the caller runs once per row.
func toPolicyResponseWithStats(p sqlcgen.ListPoliciesWithStatsRow, selector *targeting.Selector, targetEndpointsCount int) policyResponse {
	return policyResponse{
		ID:                     p.ID,
		TenantID:               p.TenantID,
		Name:                   p.Name,
		Description:            p.Description,
		Enabled:                p.Enabled,
		Mode:                   p.Mode,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
		SelectionMode:          p.SelectionMode,
		TargetSelector:         selector,
		MinSeverity:            p.MinSeverity,
		CveIds:                 p.CveIds,
		PackageRegex:           p.PackageRegex,
		ExcludePackages:        p.ExcludePackages,
		ScheduleType:           p.ScheduleType,
		ScheduleCron:           p.ScheduleCron,
		MwStart:                formatTimeOfDay(p.MwStart),
		MwEnd:                  formatTimeOfDay(p.MwEnd),
		DeploymentStrategy:     p.DeploymentStrategy,
		DeletedAt:              p.DeletedAt,
		SeverityFilter:         p.SeverityFilter,
		TargetEndpointsCount:   targetEndpointsCount,
		LastEvaluatedAt:        p.LastEvaluatedAt,
		LastEvalPass:           p.LastEvalPass,
		LastEvalEndpointCount:  p.LastEvalEndpointCount,
		LastEvalCompliantCount: p.LastEvalCompliantCount,
		PolicyType:             p.PolicyType,
		Timezone:               p.Timezone,
		MwEnabled:              p.MwEnabled,
	}
}

var validSelectionModes = map[string]bool{
	"all_available": true,
	"by_severity":   true,
	"by_cve_list":   true,
	"by_regex":      true,
}

var validScheduleTypes = map[string]bool{
	"manual":    true,
	"recurring": true,
}

var validDeploymentStrategies = map[string]bool{
	"all_at_once": true,
	"rolling":     true,
}

var validSeverities = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
}

var validPolicyModes = map[string]bool{
	"manual":    true,
	"automatic": true,
	"advisory":  true,
}

var validPolicyTypes = map[string]bool{
	"patch":      true,
	"deploy":     true,
	"compliance": true,
}

type createPolicyRequest struct {
	Name               string   `json:"name"`
	Description        string   `json:"description,omitempty"`
	Enabled            *bool    `json:"enabled,omitempty"`
	Mode               string   `json:"mode,omitempty"`
	SelectionMode      string   `json:"selection_mode"`
	MinSeverity        string   `json:"min_severity,omitempty"`
	CVEIDs             []string `json:"cve_ids,omitempty"`
	PackageRegex       string   `json:"package_regex,omitempty"`
	ExcludePackages    []string `json:"exclude_packages,omitempty"`
	ScheduleType       string   `json:"schedule_type,omitempty"`
	ScheduleCron       string   `json:"schedule_cron,omitempty"`
	MwStart            string   `json:"mw_start,omitempty"`
	MwEnd              string   `json:"mw_end,omitempty"`
	DeploymentStrategy string   `json:"deployment_strategy,omitempty"`
	// TargetSelector is the key=value tag AST that scopes the policy. A
	// nil selector means "match every non-decommissioned endpoint in the
	// tenant" — dangerous by default, the UI should warn before submit.
	TargetSelector *targeting.Selector `json:"target_selector,omitempty"`
	PolicyType     string              `json:"policy_type,omitempty"`
	Timezone       string              `json:"timezone,omitempty"`
	MwEnabled      *bool               `json:"mw_enabled,omitempty"`
}

type updatePolicyRequest = createPolicyRequest

// validatePolicyRequest validates the common fields for create and update.
// Returns (code, message, field) on validation failure, or ("", "", "") when valid.
func validatePolicyRequest(body *createPolicyRequest) (string, string, string) {
	if body.Name == "" {
		return "VALIDATION_ERROR", "name is required", "name"
	}
	if !validSelectionModes[body.SelectionMode] {
		return "VALIDATION_ERROR", "selection_mode must be one of: all_available, by_severity, by_cve_list, by_regex", "selection_mode"
	}
	if body.Mode != "" && !validPolicyModes[body.Mode] {
		return "VALIDATION_ERROR", "mode must be one of: manual, automatic, advisory", "mode"
	}
	switch body.SelectionMode {
	case "by_severity":
		if body.MinSeverity == "" {
			return "VALIDATION_ERROR", "min_severity is required when selection_mode is by_severity", "min_severity"
		}
		if !validSeverities[body.MinSeverity] {
			return "VALIDATION_ERROR", "min_severity must be one of: critical, high, medium, low", "min_severity"
		}
	case "by_cve_list":
		if len(body.CVEIDs) == 0 {
			return "VALIDATION_ERROR", "cve_ids must be non-empty when selection_mode is by_cve_list", "cve_ids"
		}
	case "by_regex":
		if body.PackageRegex == "" {
			return "VALIDATION_ERROR", "package_regex is required when selection_mode is by_regex", "package_regex"
		}
		if _, err := regexp.Compile(body.PackageRegex); err != nil {
			return "VALIDATION_ERROR", "package_regex is not a valid regular expression", "package_regex"
		}
	}
	if body.ScheduleType != "" && !validScheduleTypes[body.ScheduleType] {
		return "VALIDATION_ERROR", "schedule_type must be one of: manual, recurring", "schedule_type"
	}
	if body.DeploymentStrategy != "" && !validDeploymentStrategies[body.DeploymentStrategy] {
		return "VALIDATION_ERROR", "deployment_strategy must be one of: all_at_once, rolling", "deployment_strategy"
	}
	if body.MwStart != "" {
		if _, err := parseTimeOfDay(body.MwStart); err != nil {
			return "VALIDATION_ERROR", "mw_start must be in HH:MM format (e.g., 09:00)", "mw_start"
		}
	}
	if body.MwEnd != "" {
		if _, err := parseTimeOfDay(body.MwEnd); err != nil {
			return "VALIDATION_ERROR", "mw_end must be in HH:MM format (e.g., 17:00)", "mw_end"
		}
	}
	if (body.MwStart != "") != (body.MwEnd != "") {
		return "VALIDATION_ERROR", "mw_start and mw_end must both be provided or both be omitted", "mw_start"
	}
	if body.PolicyType != "" && !validPolicyTypes[body.PolicyType] {
		return "VALIDATION_ERROR", "policy_type must be one of: patch, deploy, compliance", "policy_type"
	}
	if body.Timezone != "" {
		if _, err := time.LoadLocation(body.Timezone); err != nil {
			return "VALIDATION_ERROR", "unknown timezone: " + body.Timezone, "timezone"
		}
	}
	if body.PolicyType == "compliance" && body.Mode != "" && body.Mode != "advisory" {
		return "VALIDATION_ERROR", "compliance policies must use advisory mode", "mode"
	}
	if body.PolicyType == "deploy" && body.Mode == "advisory" {
		return "VALIDATION_ERROR", "deploy policies cannot use advisory mode", "mode"
	}
	return "", "", ""
}

// parseTimeOfDay parses an "HH:MM" string into pgtype.Time.
// Returns a zero-value (invalid) pgtype.Time for empty input.
func parseTimeOfDay(s string) (pgtype.Time, error) {
	if s == "" {
		return pgtype.Time{}, nil
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return pgtype.Time{}, err
	}
	micros := int64(t.Hour())*3600000000 + int64(t.Minute())*60000000
	return pgtype.Time{Microseconds: micros, Valid: true}, nil
}

// policyDefaults resolves default values from a request body.
type policyDefaults struct {
	Enabled            bool
	Mode               string
	ScheduleType       string
	DeploymentStrategy string
	MwStart            pgtype.Time
	MwEnd              pgtype.Time
	PolicyType         string
	Timezone           string
	MwEnabled          bool
}

func resolvePolicyDefaults(body *createPolicyRequest) policyDefaults {
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	mode := body.Mode
	if mode == "" {
		mode = "manual"
	}
	scheduleType := body.ScheduleType
	if scheduleType == "" {
		scheduleType = "manual"
	}
	deploymentStrategy := body.DeploymentStrategy
	if deploymentStrategy == "" {
		deploymentStrategy = "all_at_once"
	}
	// Errors already validated in validatePolicyRequest.
	mwStart, _ := parseTimeOfDay(body.MwStart)
	mwEnd, _ := parseTimeOfDay(body.MwEnd)
	policyType := body.PolicyType
	if policyType == "" {
		policyType = "patch"
	}
	tz := body.Timezone
	if tz == "" {
		tz = "UTC"
	}
	mwEnabled := false
	if body.MwEnabled != nil {
		mwEnabled = *body.MwEnabled
	}
	if policyType == "compliance" {
		mode = "advisory"
	}
	return policyDefaults{
		Enabled:            enabled,
		Mode:               mode,
		ScheduleType:       scheduleType,
		DeploymentStrategy: deploymentStrategy,
		MwStart:            mwStart,
		MwEnd:              mwEnd,
		PolicyType:         policyType,
		Timezone:           tz,
		MwEnabled:          mwEnabled,
	}
}

// severityFilterFromPolicy derives the severity_filter TEXT[] column value
// used by the deployment evaluator from the policy's selection_mode + min_severity.
//
// NOTE: This duplicates the logic in deployment.BuildSeverityFilter intentionally.
// This version runs at write-time (policy create/update) to persist the derived
// severity filter into the severity_filter column so the evaluator can read it
// directly. The deployment.BuildSeverityFilter version is the runtime fallback
// used when the column is empty (e.g., policies created before the column existed).
// Do not consolidate without updating both call sites and verifying migration coverage.
func severityFilterFromPolicy(selectionMode, minSeverity string) []string {
	if selectionMode != "by_severity" || minSeverity == "" {
		return nil
	}
	rank := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	allSevs := []string{"low", "medium", "high", "critical"}
	minRank := rank[minSeverity]
	var result []string
	for _, s := range allSevs {
		if rank[s] >= minRank {
			result = append(result, s)
		}
	}
	return result
}

// buildCreateParams converts a validated request into sqlc params.
func buildCreateParams(body *createPolicyRequest, tid pgtype.UUID) sqlcgen.CreatePolicyParams {
	d := resolvePolicyDefaults(body)
	return sqlcgen.CreatePolicyParams{
		TenantID:           tid,
		Name:               body.Name,
		Description:        textFromString(body.Description),
		Enabled:            d.Enabled,
		Mode:               d.Mode,
		SelectionMode:      body.SelectionMode,
		MinSeverity:        textFromString(body.MinSeverity),
		CveIds:             body.CVEIDs,
		PackageRegex:       textFromString(body.PackageRegex),
		ExcludePackages:    body.ExcludePackages,
		ScheduleType:       d.ScheduleType,
		ScheduleCron:       textFromString(body.ScheduleCron),
		MwStart:            d.MwStart,
		MwEnd:              d.MwEnd,
		DeploymentStrategy: d.DeploymentStrategy,
		PolicyType:         d.PolicyType,
		Timezone:           d.Timezone,
		MwEnabled:          d.MwEnabled,
	}
}

// Create handles POST /api/v1/policies.
func (h *PolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body createPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	if code, msg, field := validatePolicyRequest(&body); code != "" {
		if field != "" {
			WriteFieldError(w, http.StatusBadRequest, code, msg, field)
		} else {
			WriteError(w, http.StatusBadRequest, code, msg)
		}
		return
	}

	// Validate the tag selector up front, outside of any transaction, so a
	// malformed AST is a clean 400 rather than an opaque 500 from a rolled-
	// back tx. A nil selector is legal and means "match every endpoint".
	if body.TargetSelector != nil {
		if err := targeting.Validate(*body.TargetSelector); err != nil {
			if errors.Is(err, targeting.ErrMalformedSelector) {
				WriteFieldError(w, http.StatusBadRequest, "INVALID_SELECTOR", err.Error(), "target_selector")
				return
			}
			slog.ErrorContext(ctx, "validate policy selector", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusBadRequest, "INVALID_SELECTOR", err.Error())
			return
		}
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	params := buildCreateParams(&body, tid)

	// Atomic: create policy + (optionally) persist tag selector + sync
	// severity_filter. Rolling back on any step avoids a half-created
	// policy with no selector attached.
	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for create policy", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create policy")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context for create policy", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := sqlcgen.New(tx)
	pol, err := txQ.CreatePolicy(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "create policy", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create policy")
		return
	}

	if body.TargetSelector != nil {
		raw, merr := json.Marshal(body.TargetSelector)
		if merr != nil {
			slog.ErrorContext(ctx, "marshal policy selector", "policy_id", uuidToString(pol.ID), "error", merr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to persist target selector")
			return
		}
		if _, err := txQ.UpsertPolicyTagSelector(ctx, sqlcgen.UpsertPolicyTagSelectorParams{
			PolicyID:   pol.ID,
			TenantID:   tid,
			Expression: raw,
		}); err != nil {
			slog.ErrorContext(ctx, "upsert policy selector", "policy_id", uuidToString(pol.ID), "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to persist target selector")
			return
		}
	}

	if sf := severityFilterFromPolicy(body.SelectionMode, body.MinSeverity); sf != nil {
		if _, err := tx.Exec(ctx, "UPDATE policies SET severity_filter = $1 WHERE id = $2 AND tenant_id = $3", sf, pol.ID, tid); err != nil {
			slog.ErrorContext(ctx, "set severity_filter on policy", "policy_id", uuidToString(pol.ID), "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create policy")
			return
		}
		pol.SeverityFilter = sf
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit create policy tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create policy")
		return
	}

	emitEvent(ctx, h.eventBus, events.PolicyCreated, "policy", uuidToString(pol.ID), tenantID, pol)
	if body.TargetSelector != nil {
		emitEvent(ctx, h.eventBus, events.PolicyTargetSelectorUpdated, "policy", uuidToString(pol.ID), tenantID, map[string]any{
			"policy_id": uuidToString(pol.ID),
			"selector":  body.TargetSelector,
		})
	}
	resp := toPolicyResponse(pol, body.TargetSelector)
	WriteJSON(w, http.StatusCreated, resp)
}

// List handles GET /api/v1/policies with pagination, search, enabled, and mode filters.
func (h *PolicyHandler) List(w http.ResponseWriter, r *http.Request) {
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

	search := r.URL.Query().Get("search")
	enabledFilter := r.URL.Query().Get("enabled")
	modeFilter := r.URL.Query().Get("mode")
	typeFilter := r.URL.Query().Get("type")

	policies, err := h.q.ListPoliciesWithStats(ctx, sqlcgen.ListPoliciesWithStatsParams{
		TenantID:        tid,
		Search:          search,
		EnabledFilter:   enabledFilter,
		ModeFilter:      modeFilter,
		TypeFilter:      typeFilter,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageLimit:       limit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list policies", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list policies")
		return
	}

	total, err := h.q.CountPoliciesFiltered(ctx, sqlcgen.CountPoliciesFilteredParams{
		TenantID:      tid,
		Search:        search,
		EnabledFilter: enabledFilter,
		ModeFilter:    modeFilter,
		TypeFilter:    typeFilter,
	})
	if err != nil {
		slog.ErrorContext(ctx, "count policies", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count policies")
		return
	}

	var nextCursor string
	if len(policies) == int(limit) {
		last := policies[len(policies)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, uuidToString(last.ID))
	}

	// Load the tag selector and live target count for each policy. Doing
	// this per-row is N+M queries (one selector SELECT + one Count per
	// policy). At M2 scale (dozens of policies per page) this is fine;
	// if it becomes a hot path, batch the selector SELECTs first and
	// feed a single compiled resolver call per unique expression.
	respItems := make([]policyResponse, len(policies))
	for i, p := range policies {
		sel, selErr := h.loadPolicySelector(ctx, tid, p.ID)
		if selErr != nil {
			slog.ErrorContext(ctx, "load policy selector", "policy_id", uuidToString(p.ID), "tenant_id", tenantID, "error", selErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load policy selector")
			return
		}
		count := 0
		if h.resolver != nil {
			if c, cErr := h.resolver.Count(ctx, tenantID, sel); cErr != nil {
				slog.WarnContext(ctx, "resolver count failed, reporting zero", "policy_id", uuidToString(p.ID), "error", cErr)
			} else {
				count = c
			}
		}
		respItems[i] = toPolicyResponseWithStats(p, sel, count)
	}
	WriteList(w, respItems, nextCursor, total)
}

// loadPolicySelector returns the stored tag selector for a policy, or nil
// if none is attached ("match all"). Returns an error only for true
// load failures — ErrNoRows and an empty Expression are both translated
// to (nil, nil).
func (h *PolicyHandler) loadPolicySelector(ctx context.Context, tid, policyID pgtype.UUID) (*targeting.Selector, error) {
	row, err := h.q.GetPolicyTagSelector(ctx, sqlcgen.GetPolicyTagSelectorParams{
		PolicyID: policyID,
		TenantID: tid,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	// A fake / zero-value row is treated as "no selector stored" rather
	// than a decode error. In production the DB's NOT NULL constraint on
	// expression prevents this branch from ever firing.
	if len(row.Expression) == 0 {
		return nil, nil
	}
	var sel targeting.Selector
	if err := json.Unmarshal(row.Expression, &sel); err != nil {
		return nil, fmt.Errorf("decode policy selector: %w", err)
	}
	return &sel, nil
}

// endpointSummary is a compact endpoint representation for the policy detail view.
type endpointSummary struct {
	ID       pgtype.UUID `json:"id"`
	Hostname string      `json:"hostname"`
	OsFamily string      `json:"os_family"`
	Status   string      `json:"status"`
}

// policyDetailResponse combines policy info with its matched endpoints
// and recent activity. The legacy Groups field is removed; tag targeting
// is surfaced via policyResponse.TargetSelector.
type policyDetailResponse struct {
	policyResponse
	MatchedEndpoints  []endpointSummary          `json:"matched_endpoints"`
	RecentEvaluations []sqlcgen.PolicyEvaluation `json:"recent_evaluations"`
	RecentDeployments []sqlcgen.Deployment       `json:"recent_deployments"`
	DeploymentCount   int64                      `json:"deployment_count"`
}

// Get handles GET /api/v1/policies/{id}.
func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	pol, err := h.q.GetPolicyByID(ctx, sqlcgen.GetPolicyByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "policy not found")
			return
		}
		slog.ErrorContext(ctx, "get policy", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get policy")
		return
	}

	selector, err := h.loadPolicySelector(ctx, tid, id)
	if err != nil {
		slog.ErrorContext(ctx, "load policy selector", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load policy selector")
		return
	}

	// Resolve matched endpoints via the tag selector. We do NOT suppress
	// ErrPolicyNotFound here: the policy fetch above confirms the row
	// existed a moment ago, so a concurrent delete is the only way we'd
	// hit the sentinel and that is a 404, not a "healthy response with
	// no endpoints". Passing the sentinel into the 500 branch below is
	// intentional — it is a TOCTOU race, not an expected state.
	var epSummaries []endpointSummary
	if h.resolver != nil {
		ids, rerr := h.resolver.ResolveForPolicy(ctx, tenantID, chi.URLParam(r, "id"))
		if rerr != nil {
			if errors.Is(rerr, targeting.ErrPolicyNotFound) {
				slog.WarnContext(ctx, "policy deleted between Get fetch and resolver", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID)
				WriteError(w, http.StatusNotFound, "NOT_FOUND", "policy not found")
				return
			}
			slog.ErrorContext(ctx, "resolve endpoints for policy", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", rerr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoints for policy")
			return
		}
		if len(ids) > 0 {
			pgIDs := make([]pgtype.UUID, len(ids))
			for i, id := range ids {
				pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
			}
			rows, herr := h.q.ListEndpointsByIDs(ctx, sqlcgen.ListEndpointsByIDsParams{TenantID: tid, Ids: pgIDs})
			if herr != nil {
				slog.ErrorContext(ctx, "hydrate policy endpoints", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", herr)
				WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoints for policy")
				return
			}
			epSummaries = make([]endpointSummary, len(rows))
			for i, ep := range rows {
				epSummaries[i] = endpointSummary{
					ID:       ep.ID,
					Hostname: ep.Hostname,
					OsFamily: ep.OsFamily,
					Status:   ep.Status,
				}
			}
		}
	}

	// Fetch recent evaluations.
	evals, err := h.q.ListPolicyEvaluations(ctx, sqlcgen.ListPolicyEvaluationsParams{TenantID: tid, PolicyID: id, PageLimit: 20})
	if err != nil {
		slog.ErrorContext(ctx, "list policy evaluations", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list policy evaluations")
		return
	}

	// Fetch recent deployments + count.
	deployments, err := h.q.ListDeploymentsForPolicy(ctx, sqlcgen.ListDeploymentsForPolicyParams{TenantID: tid, PolicyID: id, PageLimit: 20})
	if err != nil {
		slog.ErrorContext(ctx, "list deployments for policy", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deployments for policy")
		return
	}
	deploymentCount, err := h.q.CountDeploymentsForPolicy(ctx, sqlcgen.CountDeploymentsForPolicyParams{TenantID: tid, PolicyID: id})
	if err != nil {
		slog.ErrorContext(ctx, "count deployments for policy", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count deployments for policy")
		return
	}

	resp := policyDetailResponse{
		policyResponse:    toPolicyResponse(pol, selector),
		MatchedEndpoints:  epSummaries,
		RecentEvaluations: evals,
		RecentDeployments: deployments,
		DeploymentCount:   deploymentCount,
	}
	resp.TargetEndpointsCount = len(epSummaries)
	WriteJSON(w, http.StatusOK, resp)
}

// Update handles PUT /api/v1/policies/{id}.
func (h *PolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	// Read the body once so we can do a "null vs absent" peek on the
	// target_selector field before decoding into the typed struct.
	// json.Decoder collapses both to nil, which would silently ignore a
	// user who explicitly sent `"target_selector": null` expecting to
	// clear the policy's selector. We reject that explicitly until a
	// dedicated DELETE /policies/{id}/target_selector endpoint ships.
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "failed to read request body")
		return
	}
	var rawFields map[string]json.RawMessage
	if jerr := json.Unmarshal(rawBody, &rawFields); jerr != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if rawSel, present := rawFields["target_selector"]; present && string(rawSel) == "null" {
		WriteFieldError(w, http.StatusBadRequest, "INVALID_SELECTOR",
			"target_selector: null is not supported — omit the field to leave the selector unchanged",
			"target_selector")
		return
	}

	var body updatePolicyRequest
	if err := json.Unmarshal(rawBody, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	if code, msg, field := validatePolicyRequest(&body); code != "" {
		if field != "" {
			WriteFieldError(w, http.StatusBadRequest, code, msg, field)
		} else {
			WriteError(w, http.StatusBadRequest, code, msg)
		}
		return
	}

	if body.TargetSelector != nil {
		if err := targeting.Validate(*body.TargetSelector); err != nil {
			if errors.Is(err, targeting.ErrMalformedSelector) {
				WriteFieldError(w, http.StatusBadRequest, "INVALID_SELECTOR", err.Error(), "target_selector")
				return
			}
			slog.ErrorContext(ctx, "validate policy selector", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusBadRequest, "INVALID_SELECTOR", err.Error())
			return
		}
	}

	d := resolvePolicyDefaults(&body)

	// Wrap update + severity filter + group reassignment in a single transaction.
	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for update policy", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update policy")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context for update policy", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := sqlcgen.New(tx)

	// Step 1: Update the policy row.
	pol, err := txQ.UpdatePolicy(ctx, sqlcgen.UpdatePolicyParams{
		ID:                 id,
		TenantID:           tid,
		Name:               body.Name,
		Description:        textFromString(body.Description),
		Enabled:            d.Enabled,
		Mode:               d.Mode,
		SelectionMode:      body.SelectionMode,
		MinSeverity:        textFromString(body.MinSeverity),
		CveIds:             body.CVEIDs,
		PackageRegex:       textFromString(body.PackageRegex),
		ExcludePackages:    body.ExcludePackages,
		ScheduleType:       d.ScheduleType,
		ScheduleCron:       textFromString(body.ScheduleCron),
		MwStart:            d.MwStart,
		MwEnd:              d.MwEnd,
		DeploymentStrategy: d.DeploymentStrategy,
		PolicyType:         d.PolicyType,
		Timezone:           d.Timezone,
		MwEnabled:          d.MwEnabled,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "policy not found")
			return
		}
		slog.ErrorContext(ctx, "update policy", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update policy")
		return
	}

	// Step 2: Sync severity_filter for the deployment evaluator.
	if sf := severityFilterFromPolicy(body.SelectionMode, body.MinSeverity); sf != nil {
		if _, err := tx.Exec(ctx, "UPDATE policies SET severity_filter = $1 WHERE id = $2 AND tenant_id = $3", sf, id, tid); err != nil {
			slog.ErrorContext(ctx, "sync severity_filter on update", "policy_id", id, "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update policy severity filter")
			return
		}
		pol.SeverityFilter = sf
	}

	// Step 3: Replace the tag selector. Absent = leave existing; nil with
	// an explicit "target_selector": null would clear it — but json.Decode
	// collapses both cases into TargetSelector == nil, so we treat nil as
	// "no change". The dedicated DELETE /policies/{id}/target_selector
	// endpoint (future) will expose "clear selector" explicitly; until
	// then users can PATCH a trivial match-all selector.
	if body.TargetSelector != nil {
		raw, merr := json.Marshal(body.TargetSelector)
		if merr != nil {
			slog.ErrorContext(ctx, "marshal policy selector", "policy_id", uuidToString(pol.ID), "error", merr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to persist target selector")
			return
		}
		if _, err := txQ.UpsertPolicyTagSelector(ctx, sqlcgen.UpsertPolicyTagSelectorParams{
			PolicyID:   id,
			TenantID:   tid,
			Expression: raw,
		}); err != nil {
			slog.ErrorContext(ctx, "upsert policy selector", "policy_id", uuidToString(pol.ID), "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to persist target selector")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit update policy tx", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update policy")
		return
	}

	emitEvent(ctx, h.eventBus, events.PolicyUpdated, "policy", uuidToString(pol.ID), tenantID, pol)
	if body.TargetSelector != nil {
		emitEvent(ctx, h.eventBus, events.PolicyTargetSelectorUpdated, "policy", uuidToString(pol.ID), tenantID, map[string]any{
			"policy_id": uuidToString(pol.ID),
			"selector":  body.TargetSelector,
		})
	}
	updateResp := toPolicyResponse(pol, body.TargetSelector)
	WriteJSON(w, http.StatusOK, updateResp)
}

// Delete handles DELETE /api/v1/policies/{id} (soft delete).
func (h *PolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	pol, err := h.q.SoftDeletePolicy(ctx, sqlcgen.SoftDeletePolicyParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "policy not found")
			return
		}
		slog.ErrorContext(ctx, "soft delete policy", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete policy")
		return
	}

	emitEvent(ctx, h.eventBus, events.PolicyDeleted, "policy", uuidToString(pol.ID), tenantID, pol)
	w.WriteHeader(http.StatusNoContent)
}

// Toggle handles PATCH /api/v1/policies/{id} — partial update for enabled toggle.
func (h *PolicyHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Enabled == nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "request must include enabled field")
		return
	}

	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for toggle policy", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to toggle policy")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context for toggle policy", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	if _, err := tx.Exec(ctx, "UPDATE policies SET enabled = $1, updated_at = now() WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL", *body.Enabled, id, tid); err != nil {
		slog.ErrorContext(ctx, "toggle policy enabled", "policy_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to toggle policy")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit toggle policy", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to toggle policy")
		return
	}

	pol, err := h.q.GetPolicyByID(ctx, sqlcgen.GetPolicyByIDParams{ID: id, TenantID: tid})
	if err != nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "policy not found")
		return
	}

	emitEvent(ctx, h.eventBus, events.PolicyUpdated, "policy", uuidToString(pol.ID), tenantID, pol)
	sel, err := h.loadPolicySelector(ctx, tid, pol.ID)
	if err != nil {
		slog.ErrorContext(ctx, "load policy selector after toggle", "policy_id", uuidToString(pol.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load policy after toggle")
		return
	}
	WriteJSON(w, http.StatusOK, toPolicyResponse(pol, sel))
}

// bulkPolicyRequest is the JSON body for POST /api/v1/policies/bulk.
type bulkPolicyRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

// BulkAction handles POST /api/v1/policies/bulk.
func (h *PolicyHandler) BulkAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body bulkPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	if len(body.IDs) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "ids must be non-empty")
		return
	}

	validActions := map[string]bool{"enable": true, "disable": true, "delete": true}
	if !validActions[body.Action] {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "action must be one of: enable, disable, delete")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	uuids := make([]pgtype.UUID, 0, len(body.IDs))
	for _, idStr := range body.IDs {
		uid, err := scanUUID(idStr)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID: "+idStr)
			return
		}
		uuids = append(uuids, uid)
	}

	switch body.Action {
	case "enable":
		if err := h.q.BulkUpdatePolicyEnabled(ctx, sqlcgen.BulkUpdatePolicyEnabledParams{
			Enabled:  true,
			TenantID: tid,
			Ids:      uuids,
		}); err != nil {
			slog.ErrorContext(ctx, "bulk enable policies", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to bulk enable policies")
			return
		}
	case "disable":
		if err := h.q.BulkUpdatePolicyEnabled(ctx, sqlcgen.BulkUpdatePolicyEnabledParams{
			Enabled:  false,
			TenantID: tid,
			Ids:      uuids,
		}); err != nil {
			slog.ErrorContext(ctx, "bulk disable policies", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to bulk disable policies")
			return
		}
	case "delete":
		if err := h.q.BulkSoftDeletePolicies(ctx, sqlcgen.BulkSoftDeletePoliciesParams{
			TenantID: tid,
			Ids:      uuids,
		}); err != nil {
			slog.ErrorContext(ctx, "bulk delete policies", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to bulk delete policies")
			return
		}
	}

	// Emit individual events per policy.
	eventType := events.PolicyUpdated
	if body.Action == "delete" {
		eventType = events.PolicyDeleted
	}
	for _, idStr := range body.IDs {
		emitEvent(ctx, h.eventBus, eventType, "policy", idStr, tenantID, map[string]any{
			"bulk_action": body.Action,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]any{"affected": len(body.IDs)})
}

// Evaluate handles POST /api/v1/policies/{id}/evaluate.
func (h *PolicyHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id := chi.URLParam(r, "id")
	policyUUID, err := scanUUID(id)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID: not a valid UUID")
		return
	}

	evalStart := time.Now()
	results, err := h.evaluator.Evaluate(ctx, tenantID, id, evalStart)
	if err != nil {
		switch {
		case errors.Is(err, policy.ErrPolicyDisabled):
			WriteError(w, http.StatusUnprocessableEntity, "POLICY_DISABLED", "policy is disabled")
		case errors.Is(err, policy.ErrPolicyNotFound):
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "policy not found")
		case errors.Is(err, policy.ErrOutsideMaintenanceWindow):
			WriteError(w, http.StatusUnprocessableEntity, "OUTSIDE_MAINTENANCE_WINDOW", "policy evaluation skipped: outside maintenance window")
		default:
			slog.ErrorContext(ctx, "evaluate policy", "policy_id", id, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "policy evaluation failed")
		}
		return
	}
	evalDuration := time.Since(evalStart)

	// Compute stats.
	inScope := int32(len(results))
	totalPatches := int32(countTotalPatches(results))
	compliant := int32(0)
	nonCompliant := int32(0)
	for _, r := range results {
		if len(r.Patches) == 0 {
			compliant++
		} else {
			nonCompliant++
		}
	}
	pass := nonCompliant == 0

	// Persist evaluation record.
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context for eval persistence", "tenant_id", tenantID, "error", err)
		// Continue — eval succeeded, just can't persist.
	} else {
		_, evalErr := h.q.CreatePolicyEvaluation(ctx, sqlcgen.CreatePolicyEvaluationParams{
			TenantID:          tid,
			PolicyID:          policyUUID,
			MatchedPatches:    totalPatches,
			InScopeEndpoints:  inScope,
			CompliantCount:    compliant,
			NonCompliantCount: nonCompliant,
			DurationMs:        int32(evalDuration.Milliseconds()),
			Pass:              pass,
		})
		if evalErr != nil {
			slog.ErrorContext(ctx, "persist policy evaluation", "policy_id", id, "error", evalErr)
		}

		statsErr := h.q.UpdatePolicyEvalStats(ctx, sqlcgen.UpdatePolicyEvalStatsParams{
			LastEvaluatedAt:        pgtype.Timestamptz{Time: evalStart, Valid: true},
			LastEvalPass:           pgtype.Bool{Bool: pass, Valid: true},
			LastEvalEndpointCount:  pgtype.Int4{Int32: inScope, Valid: true},
			LastEvalCompliantCount: pgtype.Int4{Int32: compliant, Valid: true},
			ID:                     policyUUID,
			TenantID:               tid,
		})
		if statsErr != nil {
			slog.ErrorContext(ctx, "update policy eval stats", "policy_id", id, "error", statsErr)
		}

		emitEvent(ctx, h.eventBus, events.PolicyEvaluationRecorded, "policy", id, tenantID, map[string]any{
			"in_scope":      inScope,
			"compliant":     compliant,
			"non_compliant": nonCompliant,
			"pass":          pass,
			"duration_ms":   evalDuration.Milliseconds(),
			"total_patches": totalPatches,
		})
	}

	emitEvent(ctx, h.eventBus, events.PolicyEvaluated, "policy", id, tenantID, map[string]any{
		"endpoint_count": len(results),
	})

	WriteJSON(w, http.StatusOK, map[string]any{
		"policy_id": id,
		"results":   results,
		"summary": map[string]int{
			"endpoint_count": len(results),
			"total_patches":  countTotalPatches(results),
		},
	})
}

func countTotalPatches(results []policy.EvaluationResult) int {
	total := 0
	for _, r := range results {
		total += len(r.Patches)
	}
	return total
}
