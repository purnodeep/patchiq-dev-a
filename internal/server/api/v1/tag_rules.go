package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// TagRuleQuerier defines the sqlc queries needed by TagRuleHandler.
type TagRuleQuerier interface {
	CreateTagRule(ctx context.Context, arg sqlcgen.CreateTagRuleParams) (sqlcgen.TagRule, error)
	GetTagRuleByID(ctx context.Context, arg sqlcgen.GetTagRuleByIDParams) (sqlcgen.TagRule, error)
	ListTagRules(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.TagRule, error)
	ListEnabledTagRules(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.TagRule, error)
	UpdateTagRule(ctx context.Context, arg sqlcgen.UpdateTagRuleParams) (sqlcgen.TagRule, error)
	DeleteTagRule(ctx context.Context, arg sqlcgen.DeleteTagRuleParams) error
}

// TagRuleHandler serves tag rule REST API endpoints.
type TagRuleHandler struct {
	q        TagRuleQuerier
	eventBus domain.EventBus
}

// NewTagRuleHandler creates a TagRuleHandler.
func NewTagRuleHandler(q TagRuleQuerier, eventBus domain.EventBus) *TagRuleHandler {
	if q == nil {
		panic("tag_rules: NewTagRuleHandler called with nil querier")
	}
	if eventBus == nil {
		panic("tag_rules: NewTagRuleHandler called with nil eventBus")
	}
	return &TagRuleHandler{q: q, eventBus: eventBus}
}

// List handles GET /api/v1/tag-rules.
func (h *TagRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rules, err := h.q.ListTagRules(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list tag rules", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tag rules")
		return
	}

	WriteJSON(w, http.StatusOK, rules)
}

type createTagRuleRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Condition   json.RawMessage `json:"condition"`
	TagsToApply []string        `json:"tags_to_apply"`
	Enabled     bool            `json:"enabled"`
	Priority    int             `json:"priority"`
}

// Create handles POST /api/v1/tag-rules.
func (h *TagRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body createTagRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if body.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if len(body.Condition) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "condition is required")
		return
	}
	if !json.Valid(body.Condition) {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "condition must be valid JSON")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	tagIDs, err := parseUUIDs(body.TagsToApply)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "tags_to_apply contains invalid UUID")
		return
	}

	rule, err := h.q.CreateTagRule(ctx, sqlcgen.CreateTagRuleParams{
		TenantID:    tid,
		Name:        body.Name,
		Description: textFromString(body.Description),
		Condition:   body.Condition,
		TagsToApply: tagIDs,
		Enabled:     body.Enabled,
		Priority:    int32(body.Priority),
	})
	if err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "DUPLICATE_TAG_RULE", "a tag rule with that name already exists")
			return
		}
		slog.ErrorContext(ctx, "create tag rule", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create tag rule")
		return
	}

	emitEvent(ctx, h.eventBus, events.TagRuleCreated, "tag_rule", uuidToString(rule.ID), tenantID, rule)
	WriteJSON(w, http.StatusCreated, rule)
}

// Get handles GET /api/v1/tag-rules/{id}.
func (h *TagRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid tag rule ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rule, err := h.q.GetTagRuleByID(ctx, sqlcgen.GetTagRuleByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "tag rule not found")
			return
		}
		slog.ErrorContext(ctx, "get tag rule", "tag_rule_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get tag rule")
		return
	}

	WriteJSON(w, http.StatusOK, rule)
}

type updateTagRuleRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Condition   json.RawMessage `json:"condition"`
	TagsToApply []string        `json:"tags_to_apply"`
	Enabled     bool            `json:"enabled"`
	Priority    int             `json:"priority"`
}

// Update handles PUT /api/v1/tag-rules/{id}.
func (h *TagRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid tag rule ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body updateTagRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if body.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if len(body.Condition) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "condition is required")
		return
	}
	if !json.Valid(body.Condition) {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "condition must be valid JSON")
		return
	}

	tagIDs, err := parseUUIDs(body.TagsToApply)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "tags_to_apply contains invalid UUID")
		return
	}

	rule, err := h.q.UpdateTagRule(ctx, sqlcgen.UpdateTagRuleParams{
		ID:          id,
		TenantID:    tid,
		Name:        body.Name,
		Description: textFromString(body.Description),
		Condition:   body.Condition,
		TagsToApply: tagIDs,
		Enabled:     body.Enabled,
		Priority:    int32(body.Priority),
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "tag rule not found")
			return
		}
		slog.ErrorContext(ctx, "update tag rule", "tag_rule_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update tag rule")
		return
	}

	emitEvent(ctx, h.eventBus, events.TagRuleUpdated, "tag_rule", uuidToString(rule.ID), tenantID, rule)
	WriteJSON(w, http.StatusOK, rule)
}

// Delete handles DELETE /api/v1/tag-rules/{id}.
func (h *TagRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid tag rule ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	// Verify tag rule exists before deleting.
	rule, err := h.q.GetTagRuleByID(ctx, sqlcgen.GetTagRuleByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "tag rule not found")
			return
		}
		slog.ErrorContext(ctx, "get tag rule for delete", "tag_rule_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete tag rule")
		return
	}

	if err := h.q.DeleteTagRule(ctx, sqlcgen.DeleteTagRuleParams{ID: id, TenantID: tid}); err != nil {
		slog.ErrorContext(ctx, "delete tag rule", "tag_rule_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete tag rule")
		return
	}

	emitEvent(ctx, h.eventBus, events.TagRuleDeleted, "tag_rule", uuidToString(rule.ID), tenantID, rule)
	w.WriteHeader(http.StatusNoContent)
}

// parseUUIDs converts a slice of UUID strings into pgtype.UUID values.
func parseUUIDs(ss []string) ([]pgtype.UUID, error) {
	result := make([]pgtype.UUID, 0, len(ss))
	for _, s := range ss {
		u, err := scanUUID(s)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, nil
}
