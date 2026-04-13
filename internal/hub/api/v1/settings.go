package v1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// SettingsQuerier abstracts the sqlcgen queries used by SettingsHandler.
type SettingsQuerier interface {
	ListHubConfigByTenant(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.HubConfig, error)
	GetHubConfig(ctx context.Context, arg sqlcgen.GetHubConfigParams) (sqlcgen.HubConfig, error)
	UpsertHubConfig(ctx context.Context, arg sqlcgen.UpsertHubConfigParams) (sqlcgen.HubConfig, error)
}

// SettingsHandler serves settings endpoints.
type SettingsHandler struct {
	queries  SettingsQuerier
	eventBus domain.EventBus
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(queries SettingsQuerier, eventBus domain.EventBus) *SettingsHandler {
	return &SettingsHandler{queries: queries, eventBus: eventBus}
}

// List handles GET /api/v1/settings.
func (h *SettingsHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.MustTenantID(r.Context())
	tid, err := parseUUID(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}

	configs, err := h.queries.ListHubConfigByTenant(r.Context(), tid)
	if err != nil {
		slog.ErrorContext(r.Context(), "list settings", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list settings: internal error")
		return
	}

	result := make(map[string]json.RawMessage, len(configs))
	for _, c := range configs {
		result[c.Key] = c.Value
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.ErrorContext(r.Context(), "encode settings response", "error", err)
	}
}

// Get handles GET /api/v1/settings/{key}.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.MustTenantID(r.Context())
	tid, err := parseUUID(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}

	key := chi.URLParam(r, "key")
	if key == "" {
		writeJSONError(w, http.StatusBadRequest, "key is required")
		return
	}

	config, err := h.queries.GetHubConfig(r.Context(), sqlcgen.GetHubConfigParams{
		TenantID: tid,
		Key:      key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "setting not found: "+key)
			return
		}
		slog.ErrorContext(r.Context(), "get setting", "error", err, "key", key)
		writeJSONError(w, http.StatusInternalServerError, "get setting: internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"key":   config.Key,
		"value": json.RawMessage(config.Value),
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode setting response", "error", err)
	}
}

type upsertSettingRequest struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// Upsert handles PUT /api/v1/settings.
func (h *SettingsHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.MustTenantID(r.Context())
	tid, err := parseUUID(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}

	var req upsertSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Key == "" {
		writeJSONError(w, http.StatusBadRequest, "key is required")
		return
	}
	if len(req.Value) == 0 {
		writeJSONError(w, http.StatusBadRequest, "value is required")
		return
	}

	// Fetch old value for the event payload (best-effort).
	var oldValue json.RawMessage
	existing, err := h.queries.GetHubConfig(r.Context(), sqlcgen.GetHubConfigParams{
		TenantID: tid,
		Key:      req.Key,
	})
	if err == nil {
		oldValue = existing.Value
	}

	config, err := h.queries.UpsertHubConfig(r.Context(), sqlcgen.UpsertHubConfigParams{
		TenantID:  tid,
		Key:       req.Key,
		Value:     req.Value,
		UpdatedBy: tid, // TODO(PIQ-245): use actual user ID from auth context
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "upsert setting", "error", err, "key", req.Key)
		writeJSONError(w, http.StatusInternalServerError, "upsert setting: internal error")
		return
	}

	// Emit config.updated event.
	evt := domain.NewSystemEvent(events.ConfigUpdated, tenantID, "hub_config", config.Key, "upsert", map[string]any{
		"key":       config.Key,
		"old_value": oldValue,
		"new_value": req.Value,
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit config.updated event", "error", err, "key", req.Key)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"key":   config.Key,
		"value": json.RawMessage(config.Value),
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode upsert setting response", "error", err)
	}
}
