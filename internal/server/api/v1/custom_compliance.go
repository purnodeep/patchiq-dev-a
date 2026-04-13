package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/compliance"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// CustomComplianceQuerier defines the sqlc queries needed by CustomComplianceHandler.
type CustomComplianceQuerier interface {
	CreateCustomFramework(ctx context.Context, arg sqlcgen.CreateCustomFrameworkParams) (sqlcgen.CustomComplianceFramework, error)
	GetCustomFramework(ctx context.Context, arg sqlcgen.GetCustomFrameworkParams) (sqlcgen.CustomComplianceFramework, error)
	ListCustomFrameworks(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.ListCustomFrameworksRow, error)
	UpdateCustomFramework(ctx context.Context, arg sqlcgen.UpdateCustomFrameworkParams) (sqlcgen.CustomComplianceFramework, error)
	DeleteCustomFramework(ctx context.Context, arg sqlcgen.DeleteCustomFrameworkParams) error
	CreateCustomControl(ctx context.Context, arg sqlcgen.CreateCustomControlParams) (sqlcgen.CustomComplianceControl, error)
	ListCustomControls(ctx context.Context, arg sqlcgen.ListCustomControlsParams) ([]sqlcgen.CustomComplianceControl, error)
	DeleteCustomControls(ctx context.Context, arg sqlcgen.DeleteCustomControlsParams) error
}

// CustomComplianceHandler serves custom compliance framework REST API endpoints.
type CustomComplianceHandler struct {
	q        CustomComplianceQuerier
	eventBus domain.EventBus
	store    *store.Store
}

// NewCustomComplianceHandler creates a CustomComplianceHandler.
func NewCustomComplianceHandler(q CustomComplianceQuerier, eventBus domain.EventBus, st *store.Store) *CustomComplianceHandler {
	if q == nil {
		panic("custom_compliance: NewCustomComplianceHandler called with nil querier")
	}
	if eventBus == nil {
		panic("custom_compliance: NewCustomComplianceHandler called with nil eventBus")
	}
	if st == nil {
		panic("custom_compliance: NewCustomComplianceHandler called with nil store")
	}
	return &CustomComplianceHandler{q: q, eventBus: eventBus, store: st}
}

// ------------------------------------------------------------
// Request / response types
// ------------------------------------------------------------

type slaTierInput struct {
	Label   string  `json:"label"`
	Days    *int    `json:"days"`
	CVSSMin float64 `json:"cvss_min"`
	CVSSMax float64 `json:"cvss_max"`
}

type customControlInput struct {
	ControlID       string          `json:"control_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Category        string          `json:"category"`
	RemediationHint string          `json:"remediation_hint"`
	SLATiers        []slaTierInput  `json:"sla_tiers"`
	CheckType       string          `json:"check_type"`
	CheckConfig     json.RawMessage `json:"check_config,omitempty"`
}

type createCustomFrameworkRequest struct {
	Name          string               `json:"name"`
	Version       string               `json:"version"`
	Description   string               `json:"description"`
	ScoringMethod string               `json:"scoring_method"`
	Controls      []customControlInput `json:"controls"`
}

type updateCustomFrameworkRequest struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Description   string `json:"description"`
	ScoringMethod string `json:"scoring_method"`
}

type customControlResponse struct {
	ID              string          `json:"id"`
	FrameworkID     string          `json:"framework_id"`
	ControlID       string          `json:"control_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Category        string          `json:"category"`
	RemediationHint string          `json:"remediation_hint"`
	SLATiers        []slaTierInput  `json:"sla_tiers"`
	CheckType       string          `json:"check_type"`
	CheckConfig     json.RawMessage `json:"check_config,omitempty"`
	CreatedAt       string          `json:"created_at"`
}

type customFrameworkResponse struct {
	ID            string                  `json:"id"`
	Name          string                  `json:"name"`
	Version       string                  `json:"version"`
	Description   string                  `json:"description"`
	ScoringMethod string                  `json:"scoring_method"`
	ControlCount  int64                   `json:"control_count"`
	CreatedAt     string                  `json:"created_at"`
	UpdatedAt     string                  `json:"updated_at"`
	Controls      []customControlResponse `json:"controls,omitempty"`
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func mapCustomFrameworkResponse(fw sqlcgen.CustomComplianceFramework) customFrameworkResponse {
	desc := ""
	if fw.Description.Valid {
		desc = fw.Description.String
	}
	return customFrameworkResponse{
		ID:            uuidToString(fw.ID),
		Name:          fw.Name,
		Version:       fw.Version,
		Description:   desc,
		ScoringMethod: fw.ScoringMethod,
		CreatedAt:     fw.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     fw.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func mapCustomFrameworkListRow(row sqlcgen.ListCustomFrameworksRow) customFrameworkResponse {
	desc := ""
	if row.Description.Valid {
		desc = row.Description.String
	}
	return customFrameworkResponse{
		ID:            uuidToString(row.ID),
		Name:          row.Name,
		Version:       row.Version,
		Description:   desc,
		ScoringMethod: row.ScoringMethod,
		ControlCount:  row.ControlCount,
		CreatedAt:     row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func mapCustomControlResponse(c sqlcgen.CustomComplianceControl) customControlResponse {
	desc := ""
	if c.Description.Valid {
		desc = c.Description.String
	}
	hint := ""
	if c.RemediationHint.Valid {
		hint = c.RemediationHint.String
	}

	var tiers []slaTierInput
	if len(c.SlaTiers) > 0 {
		_ = json.Unmarshal(c.SlaTiers, &tiers)
	}
	if tiers == nil {
		tiers = []slaTierInput{}
	}

	var checkConfig json.RawMessage
	if len(c.CheckConfig) > 0 && string(c.CheckConfig) != "{}" {
		checkConfig = c.CheckConfig
	}

	return customControlResponse{
		ID:              uuidToString(c.ID),
		FrameworkID:     uuidToString(c.FrameworkID),
		ControlID:       c.ControlID,
		Name:            c.Name,
		Description:     desc,
		Category:        c.Category,
		RemediationHint: hint,
		SLATiers:        tiers,
		CheckType:       c.CheckType,
		CheckConfig:     checkConfig,
		CreatedAt:       c.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func validateScoringMethod(method string) bool {
	switch method {
	case "strictest", "average", "worst_case", "weighted":
		return true
	}
	return false
}

// ------------------------------------------------------------
// Handlers
// ------------------------------------------------------------

// List handles GET /api/v1/compliance/custom-frameworks.
func (h *CustomComplianceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	frameworks, err := h.q.ListCustomFrameworks(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: list custom frameworks", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list custom frameworks")
		return
	}

	items := make([]customFrameworkResponse, 0, len(frameworks))
	for _, fw := range frameworks {
		items = append(items, mapCustomFrameworkListRow(fw))
	}

	WriteJSON(w, http.StatusOK, items)
}

// Create handles POST /api/v1/compliance/custom-frameworks.
func (h *CustomComplianceHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var req createCustomFrameworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	if req.Version == "" {
		req.Version = "1.0"
	}

	if req.ScoringMethod == "" {
		req.ScoringMethod = "average"
	}

	if !validateScoringMethod(req.ScoringMethod) {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "scoring_method must be one of: strictest, average, worst_case, weighted")
		return
	}

	var desc pgtype.Text
	if req.Description != "" {
		desc = pgtype.Text{String: req.Description, Valid: true}
	}

	tx, err := h.store.Pool().Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: begin transaction", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to start transaction")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txq := sqlcgen.New(tx)

	// Set RLS context
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "custom_compliance: set tenant context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	fw, err := txq.CreateCustomFramework(ctx, sqlcgen.CreateCustomFrameworkParams{
		TenantID:      tid,
		Name:          req.Name,
		Version:       req.Version,
		Description:   desc,
		ScoringMethod: req.ScoringMethod,
	})
	if err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "ALREADY_EXISTS", "a custom framework with this name already exists")
			return
		}
		slog.ErrorContext(ctx, "custom_compliance: create framework", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create custom framework")
		return
	}

	controls, err := bulkInsertControls(ctx, txq, tid, fw.ID, req.Controls)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: insert controls", "framework_id", uuidToString(fw.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to insert controls: "+err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "custom_compliance: commit transaction", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit transaction")
		return
	}

	emitEvent(ctx, h.eventBus, events.CustomComplianceFrameworkCreated, "custom_compliance_framework", uuidToString(fw.ID), tenantID, map[string]any{
		"name":          fw.Name,
		"control_count": len(controls),
	})

	resp := mapCustomFrameworkResponse(fw)
	resp.Controls = controls
	WriteJSON(w, http.StatusCreated, resp)
}

// Get handles GET /api/v1/compliance/custom-frameworks/{id}.
func (h *CustomComplianceHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	id := chi.URLParam(r, "id")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	fwID, err := scanUUID(id)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid framework ID")
		return
	}

	fw, err := h.q.GetCustomFramework(ctx, sqlcgen.GetCustomFrameworkParams{ID: fwID, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "custom framework not found")
			return
		}
		slog.ErrorContext(ctx, "custom_compliance: get framework", "framework_id", id, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get custom framework")
		return
	}

	dbControls, err := h.q.ListCustomControls(ctx, sqlcgen.ListCustomControlsParams{TenantID: tid, FrameworkID: fwID})
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: list controls for framework", "framework_id", id, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get controls")
		return
	}

	resp := mapCustomFrameworkResponse(fw)
	resp.Controls = make([]customControlResponse, 0, len(dbControls))
	for _, c := range dbControls {
		resp.Controls = append(resp.Controls, mapCustomControlResponse(c))
	}

	WriteJSON(w, http.StatusOK, resp)
}

// Update handles PUT /api/v1/compliance/custom-frameworks/{id}.
func (h *CustomComplianceHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	id := chi.URLParam(r, "id")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	fwID, err := scanUUID(id)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid framework ID")
		return
	}

	var req updateCustomFrameworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	if req.Version == "" {
		req.Version = "1.0"
	}

	if req.ScoringMethod == "" {
		req.ScoringMethod = "average"
	}

	if !validateScoringMethod(req.ScoringMethod) {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "scoring_method must be one of: strictest, average, worst_case, weighted")
		return
	}

	var desc pgtype.Text
	if req.Description != "" {
		desc = pgtype.Text{String: req.Description, Valid: true}
	}

	fw, err := h.q.UpdateCustomFramework(ctx, sqlcgen.UpdateCustomFrameworkParams{
		ID:            fwID,
		TenantID:      tid,
		Name:          req.Name,
		Version:       req.Version,
		Description:   desc,
		ScoringMethod: req.ScoringMethod,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "custom framework not found")
			return
		}
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "ALREADY_EXISTS", "a custom framework with this name already exists")
			return
		}
		slog.ErrorContext(ctx, "custom_compliance: update framework", "framework_id", id, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update custom framework")
		return
	}

	emitEvent(ctx, h.eventBus, events.CustomComplianceFrameworkUpdated, "custom_compliance_framework", uuidToString(fw.ID), tenantID, map[string]any{
		"name": fw.Name,
	})

	WriteJSON(w, http.StatusOK, mapCustomFrameworkResponse(fw))
}

// Delete handles DELETE /api/v1/compliance/custom-frameworks/{id}.
func (h *CustomComplianceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	id := chi.URLParam(r, "id")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	fwID, err := scanUUID(id)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid framework ID")
		return
	}

	// Verify exists before deleting so we can return a proper 404.
	if _, err := h.q.GetCustomFramework(ctx, sqlcgen.GetCustomFrameworkParams{ID: fwID, TenantID: tid}); err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "custom framework not found")
			return
		}
		slog.ErrorContext(ctx, "custom_compliance: get framework for delete", "framework_id", id, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify custom framework")
		return
	}

	if err := h.q.DeleteCustomFramework(ctx, sqlcgen.DeleteCustomFrameworkParams{ID: fwID, TenantID: tid}); err != nil {
		slog.ErrorContext(ctx, "custom_compliance: delete framework", "framework_id", id, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete custom framework")
		return
	}

	emitEvent(ctx, h.eventBus, events.CustomComplianceFrameworkDeleted, "custom_compliance_framework", id, tenantID, map[string]any{
		"framework_id": id,
	})

	w.WriteHeader(http.StatusNoContent)
}

// UpdateControls handles PUT /api/v1/compliance/custom-frameworks/{id}/controls.
// This is a bulk-replace: all existing controls for the framework are deleted and replaced.
func (h *CustomComplianceHandler) UpdateControls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	id := chi.URLParam(r, "id")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	fwID, err := scanUUID(id)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid framework ID")
		return
	}

	// Verify framework exists.
	if _, err := h.q.GetCustomFramework(ctx, sqlcgen.GetCustomFrameworkParams{ID: fwID, TenantID: tid}); err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "custom framework not found")
			return
		}
		slog.ErrorContext(ctx, "custom_compliance: get framework for controls update", "framework_id", id, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify custom framework")
		return
	}

	var inputs []customControlInput
	if err := json.NewDecoder(r.Body).Decode(&inputs); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body: expected array of controls")
		return
	}

	tx, err := h.store.Pool().Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: begin transaction for controls update", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to start transaction")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txq := sqlcgen.New(tx)

	// Set RLS context
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "custom_compliance: set tenant context for controls update", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	// Delete existing controls first.
	if err := txq.DeleteCustomControls(ctx, sqlcgen.DeleteCustomControlsParams{TenantID: tid, FrameworkID: fwID}); err != nil {
		slog.ErrorContext(ctx, "custom_compliance: delete existing controls", "framework_id", id, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete existing controls")
		return
	}

	controls, err := bulkInsertControls(ctx, txq, tid, fwID, inputs)
	if err != nil {
		slog.ErrorContext(ctx, "custom_compliance: insert replacement controls", "framework_id", id, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to insert controls: "+err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "custom_compliance: commit controls update", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit controls update")
		return
	}

	emitEvent(ctx, h.eventBus, events.CustomComplianceControlsUpdated, "custom_compliance_framework", id, tenantID, map[string]any{
		"framework_id":  id,
		"control_count": len(controls),
	})

	WriteJSON(w, http.StatusOK, controls)
}

// ------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------

// bulkInsertControls validates and inserts a slice of control inputs for a framework.
// It returns the list of created control responses (or an error on first failure).
func bulkInsertControls(
	ctx context.Context,
	q CustomComplianceQuerier,
	tenantID pgtype.UUID,
	frameworkID pgtype.UUID,
	inputs []customControlInput,
) ([]customControlResponse, error) {
	results := make([]customControlResponse, 0, len(inputs))
	for _, input := range inputs {
		if input.ControlID == "" || input.Name == "" {
			continue // skip invalid entries silently
		}

		if !isValidCheckType(input.CheckType) {
			return nil, fmt.Errorf("invalid check_type %q for control %s", input.CheckType, input.ControlID)
		}

		category := input.Category
		if category == "" {
			category = "General"
		}

		tiersJSON, err := json.Marshal(input.SLATiers)
		if err != nil {
			return nil, err
		}
		if string(tiersJSON) == "null" {
			tiersJSON = []byte("[]")
		}

		checkConfigJSON := []byte("{}")
		if len(input.CheckConfig) > 0 {
			checkConfigJSON = input.CheckConfig
		}

		var desc pgtype.Text
		if input.Description != "" {
			desc = pgtype.Text{String: input.Description, Valid: true}
		}

		var hint pgtype.Text
		if input.RemediationHint != "" {
			hint = pgtype.Text{String: input.RemediationHint, Valid: true}
		}

		ctrl, err := q.CreateCustomControl(ctx, sqlcgen.CreateCustomControlParams{
			TenantID:        tenantID,
			FrameworkID:     frameworkID,
			ControlID:       input.ControlID,
			Name:            input.Name,
			Description:     desc,
			Category:        category,
			RemediationHint: hint,
			SlaTiers:        tiersJSON,
			CheckType:       input.CheckType,
			CheckConfig:     checkConfigJSON,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, mapCustomControlResponse(ctrl))
	}
	return results, nil
}

// isValidCheckType returns true if ct is empty (defaults to SLA) or a known check type.
func isValidCheckType(ct string) bool {
	if ct == "" || ct == "sla" {
		return true
	}
	for _, t := range compliance.AvailableCheckTypes {
		if t.Type == ct {
			return true
		}
	}
	return false
}

// ListCheckTypes handles GET /api/v1/compliance/check-types.
func (h *CustomComplianceHandler) ListCheckTypes(w http.ResponseWriter, r *http.Request) {
	types := make([]compliance.CheckTypeDef, 0, len(compliance.CheckTypeDefs))
	for _, def := range compliance.CheckTypeDefs {
		types = append(types, def)
	}
	sort.Slice(types, func(i, j int) bool { return types[i].Type < types[j].Type })
	WriteJSON(w, http.StatusOK, types)
}
