package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/robfig/cron/v3"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// ScheduleQuerier defines the sqlc queries needed by ScheduleHandler.
type ScheduleQuerier interface {
	CreateDeploymentSchedule(ctx context.Context, arg sqlcgen.CreateDeploymentScheduleParams) (sqlcgen.DeploymentSchedule, error)
	GetDeploymentScheduleByID(ctx context.Context, arg sqlcgen.GetDeploymentScheduleByIDParams) (sqlcgen.DeploymentSchedule, error)
	ListDeploymentSchedulesByTenant(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.DeploymentSchedule, error)
	UpdateDeploymentSchedule(ctx context.Context, arg sqlcgen.UpdateDeploymentScheduleParams) (sqlcgen.DeploymentSchedule, error)
	DeleteDeploymentSchedule(ctx context.Context, arg sqlcgen.DeleteDeploymentScheduleParams) error
}

// ScheduleHandler serves deployment schedule REST API endpoints.
type ScheduleHandler struct {
	q        ScheduleQuerier
	eventBus domain.EventBus
}

// NewScheduleHandler creates a ScheduleHandler.
func NewScheduleHandler(q ScheduleQuerier, eventBus domain.EventBus) *ScheduleHandler {
	if q == nil {
		panic("schedules: NewScheduleHandler called with nil querier")
	}
	if eventBus == nil {
		panic("schedules: NewScheduleHandler called with nil eventBus")
	}
	return &ScheduleHandler{q: q, eventBus: eventBus}
}

// cronParser is the standard 5-field cron parser (minute, hour, dom, month, dow).
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// parseCronNextRun validates a cron expression and returns the next run time.
func parseCronNextRun(expr string) (time.Time, error) {
	sched, err := cronParser.Parse(expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cron expression %q: %w", expr, err)
	}
	return sched.Next(time.Now()), nil
}

// createScheduleRequest is the JSON body for POST /deployment-schedules.
type createScheduleRequest struct {
	PolicyID       string                  `json:"policy_id"`
	CronExpression string                  `json:"cron_expression"`
	WaveConfig     []deployment.WaveConfig `json:"wave_config,omitempty"`
	MaxConcurrent  *int32                  `json:"max_concurrent,omitempty"`
	Enabled        *bool                   `json:"enabled,omitempty"`
}

// updateScheduleRequest is the JSON body for PATCH /deployment-schedules/{id}.
type updateScheduleRequest struct {
	CronExpression *string                 `json:"cron_expression,omitempty"`
	WaveConfig     []deployment.WaveConfig `json:"wave_config,omitempty"`
	MaxConcurrent  *int32                  `json:"max_concurrent,omitempty"`
	Enabled        *bool                   `json:"enabled,omitempty"`
}

// scheduleResponse is the clean API response type for a deployment schedule.
type scheduleResponse struct {
	ID             string                  `json:"id"`
	PolicyID       string                  `json:"policy_id"`
	CronExpression string                  `json:"cron_expression"`
	WaveConfig     []deployment.WaveConfig `json:"wave_config,omitempty"`
	MaxConcurrent  *int32                  `json:"max_concurrent,omitempty"`
	Enabled        bool                    `json:"enabled"`
	LastRunAt      *time.Time              `json:"last_run_at,omitempty"`
	NextRunAt      time.Time               `json:"next_run_at"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

// toScheduleResponse converts a sqlcgen.DeploymentSchedule to a clean API response.
func toScheduleResponse(s sqlcgen.DeploymentSchedule) scheduleResponse {
	resp := scheduleResponse{
		ID:             uuidToString(s.ID),
		PolicyID:       uuidToString(s.PolicyID),
		CronExpression: s.CronExpression,
		Enabled:        s.Enabled,
		NextRunAt:      s.NextRunAt.Time,
		CreatedAt:      s.CreatedAt.Time,
		UpdatedAt:      s.UpdatedAt.Time,
	}
	if len(s.WaveConfig) > 0 {
		var wc []deployment.WaveConfig
		if err := json.Unmarshal(s.WaveConfig, &wc); err == nil {
			resp.WaveConfig = wc
		}
	}
	if s.MaxConcurrent.Valid {
		resp.MaxConcurrent = &s.MaxConcurrent.Int32
	}
	if s.LastRunAt.Valid {
		resp.LastRunAt = &s.LastRunAt.Time
	}
	return resp
}

// Create handles POST /api/v1/deployment-schedules.
func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if body.PolicyID == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_POLICY_ID", "policy_id is required")
		return
	}
	if body.CronExpression == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_CRON_EXPRESSION", "cron_expression is required")
		return
	}

	policyID, err := scanUUID(body.PolicyID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_POLICY_ID", "policy_id is not a valid UUID")
		return
	}

	nextRun, err := parseCronNextRun(body.CronExpression)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_CRON_EXPRESSION", "cron_expression is not a valid cron expression")
		return
	}

	// Marshal wave config to JSON for storage.
	var waveConfigJSON []byte
	if len(body.WaveConfig) > 0 {
		waveConfigJSON, err = json.Marshal(body.WaveConfig)
		if err != nil {
			slog.ErrorContext(ctx, "marshal wave config", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to marshal wave config")
			return
		}
	}

	var maxConcurrent pgtype.Int4
	if body.MaxConcurrent != nil {
		maxConcurrent = pgtype.Int4{Int32: *body.MaxConcurrent, Valid: true}
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	schedule, err := h.q.CreateDeploymentSchedule(ctx, sqlcgen.CreateDeploymentScheduleParams{
		TenantID:       tid,
		PolicyID:       policyID,
		CronExpression: body.CronExpression,
		WaveConfig:     waveConfigJSON,
		MaxConcurrent:  maxConcurrent,
		Enabled:        enabled,
		NextRunAt:      pgtype.Timestamptz{Time: nextRun, Valid: true},
		CreatedBy:      pgtype.UUID{}, // system user for now
	})
	if err != nil {
		slog.ErrorContext(ctx, "create deployment schedule", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment schedule")
		return
	}

	emitEvent(ctx, h.eventBus, events.ScheduleCreated, "deployment_schedule", uuidToString(schedule.ID), tenantID, schedule)

	WriteJSON(w, http.StatusCreated, toScheduleResponse(schedule))
}

// List handles GET /api/v1/deployment-schedules.
func (h *ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	schedules, err := h.q.ListDeploymentSchedulesByTenant(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list deployment schedules", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deployment schedules")
		return
	}

	resp := make([]scheduleResponse, len(schedules))
	for i, s := range schedules {
		resp[i] = toScheduleResponse(s)
	}

	WriteJSON(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/deployment-schedules/{id}.
func (h *ScheduleHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid schedule ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	schedule, err := h.q.GetDeploymentScheduleByID(ctx, sqlcgen.GetDeploymentScheduleByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "deployment schedule not found")
			return
		}
		slog.ErrorContext(ctx, "get deployment schedule", "schedule_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get deployment schedule")
		return
	}

	WriteJSON(w, http.StatusOK, toScheduleResponse(schedule))
}

// Update handles PATCH /api/v1/deployment-schedules/{id}.
func (h *ScheduleHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid schedule ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	// First fetch the existing schedule to merge fields.
	existing, err := h.q.GetDeploymentScheduleByID(ctx, sqlcgen.GetDeploymentScheduleByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "deployment schedule not found")
			return
		}
		slog.ErrorContext(ctx, "get deployment schedule for update", "schedule_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get deployment schedule")
		return
	}

	// Determine cron expression: use new value or keep existing.
	cronExpr := existing.CronExpression
	if body.CronExpression != nil {
		if *body.CronExpression == "" {
			WriteError(w, http.StatusBadRequest, "INVALID_CRON_EXPRESSION", "cron_expression cannot be empty")
			return
		}
		cronExpr = *body.CronExpression
	}

	// Validate cron expression if it changed.
	if body.CronExpression != nil {
		if _, err := parseCronNextRun(cronExpr); err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_CRON_EXPRESSION", "cron_expression is not a valid cron expression")
			return
		}
	}

	// Wave config: use new value or keep existing.
	waveConfig := existing.WaveConfig
	if body.WaveConfig != nil {
		waveConfig, err = json.Marshal(body.WaveConfig)
		if err != nil {
			slog.ErrorContext(ctx, "marshal wave config", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to marshal wave config")
			return
		}
	}

	// Max concurrent: use new value or keep existing.
	maxConcurrent := existing.MaxConcurrent
	if body.MaxConcurrent != nil {
		maxConcurrent = pgtype.Int4{Int32: *body.MaxConcurrent, Valid: true}
	}

	// Enabled: use new value or keep existing.
	enabled := existing.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	// The sqlc-generated UpdateDeploymentScheduleParams uses Column2 (interface{})
	// for the cron expression due to COALESCE(NULLIF($2, ''), ...).
	// Pass empty string to keep existing, or the new value to update.
	var cronColumn2 interface{} = ""
	if body.CronExpression != nil {
		cronColumn2 = *body.CronExpression
	}

	schedule, err := h.q.UpdateDeploymentSchedule(ctx, sqlcgen.UpdateDeploymentScheduleParams{
		ID:            id,
		Column2:       cronColumn2,
		WaveConfig:    waveConfig,
		MaxConcurrent: maxConcurrent,
		Enabled:       enabled,
		TenantID:      tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "deployment schedule not found")
			return
		}
		slog.ErrorContext(ctx, "update deployment schedule", "schedule_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update deployment schedule")
		return
	}

	emitEvent(ctx, h.eventBus, events.ScheduleUpdated, "deployment_schedule", uuidToString(schedule.ID), tenantID, schedule)

	WriteJSON(w, http.StatusOK, toScheduleResponse(schedule))
}

// Delete handles DELETE /api/v1/deployment-schedules/{id}.
func (h *ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid schedule ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	err = h.q.DeleteDeploymentSchedule(ctx, sqlcgen.DeleteDeploymentScheduleParams{ID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "delete deployment schedule", "schedule_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete deployment schedule")
		return
	}

	emitEvent(ctx, h.eventBus, events.ScheduleDeleted, "deployment_schedule", uuidToString(id), tenantID, nil)

	w.WriteHeader(http.StatusNoContent)
}
