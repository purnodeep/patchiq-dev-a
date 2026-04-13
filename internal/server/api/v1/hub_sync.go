package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/server/workers"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// HubSyncQuerier defines the sqlc queries needed by HubSyncAPIHandler.
type HubSyncQuerier interface {
	GetHubSyncState(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.HubSyncState, error)
	UpsertHubSyncState(ctx context.Context, arg sqlcgen.UpsertHubSyncStateParams) (sqlcgen.HubSyncState, error)
}

// RiverEnqueuer abstracts River's Insert method for testing.
type RiverEnqueuer interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// HubSyncAPIHandler handles Hub sync configuration and trigger endpoints.
type HubSyncAPIHandler struct {
	queries     HubSyncQuerier
	riverClient RiverEnqueuer
	eventBus    domain.EventBus
}

// NewHubSyncAPIHandler creates a new HubSyncAPIHandler.
func NewHubSyncAPIHandler(queries HubSyncQuerier, riverClient RiverEnqueuer, eventBus domain.EventBus) *HubSyncAPIHandler {
	return &HubSyncAPIHandler{
		queries:     queries,
		riverClient: riverClient,
		eventBus:    eventBus,
	}
}

// syncStatusResponse is the JSON response for GET /api/v1/sync/status.
type syncStatusResponse struct {
	ID              string  `json:"id"`
	HubURL          string  `json:"hub_url"`
	Status          string  `json:"status"`
	LastSyncAt      *string `json:"last_sync_at"`
	NextSyncAt      *string `json:"next_sync_at"`
	SyncInterval    int32   `json:"sync_interval"`
	EntriesReceived int32   `json:"entries_received"`
	LastEntryCount  int32   `json:"last_entry_count"`
	LastError       *string `json:"last_error"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func toSyncStatusResponse(s sqlcgen.HubSyncState) syncStatusResponse {
	resp := syncStatusResponse{
		ID:              uuid.UUID(s.ID.Bytes).String(),
		HubURL:          s.HubUrl,
		Status:          s.Status,
		SyncInterval:    s.SyncInterval,
		EntriesReceived: s.EntriesReceived,
		LastEntryCount:  s.LastEntryCount,
	}
	if s.LastSyncAt.Valid {
		t := s.LastSyncAt.Time.UTC().Format(time.RFC3339)
		resp.LastSyncAt = &t
	}
	if s.NextSyncAt.Valid {
		t := s.NextSyncAt.Time.UTC().Format(time.RFC3339)
		resp.NextSyncAt = &t
	}
	if s.LastError.Valid {
		resp.LastError = &s.LastError.String
	}
	if s.CreatedAt.Valid {
		resp.CreatedAt = s.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	if s.UpdatedAt.Valid {
		resp.UpdatedAt = s.UpdatedAt.Time.UTC().Format(time.RFC3339)
	}
	return resp
}

// Status returns the current Hub sync state for the tenant.
// GET /api/v1/sync/status
func (h *HubSyncAPIHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_tenant_id", "invalid tenant ID in context")
		return
	}

	state, err := h.queries.GetHubSyncState(ctx, tid)
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "not_found", "hub sync not configured for this tenant")
			return
		}
		slog.ErrorContext(ctx, "hub sync status: query failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve hub sync state")
		return
	}

	WriteJSON(w, http.StatusOK, toSyncStatusResponse(state))
}

// Trigger enqueues a CatalogSyncJobArgs to trigger an immediate sync.
// POST /api/v1/sync/trigger
func (h *HubSyncAPIHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_tenant_id", "invalid tenant ID in context")
		return
	}

	// Verify sync state exists before enqueuing
	if _, err := h.queries.GetHubSyncState(ctx, tid); err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "not_found", "hub sync not configured; set config first")
			return
		}
		slog.ErrorContext(ctx, "hub sync trigger: query failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to check hub sync state")
		return
	}

	_, err = h.riverClient.Insert(ctx, workers.CatalogSyncJobArgs{TenantID: tenantID}, &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		slog.ErrorContext(ctx, "hub sync trigger: enqueue failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to enqueue sync job")
		return
	}

	emitEvent(ctx, h.eventBus, events.HubSyncTriggered, "hub_sync", tenantID, tenantID, nil)

	WriteJSON(w, http.StatusAccepted, map[string]string{
		"message": "catalog sync job enqueued",
	})
}

// syncConfigRequest is the JSON body for PUT /api/v1/sync/config.
type syncConfigRequest struct {
	HubURL       string `json:"hub_url"`
	APIKey       string `json:"api_key"`
	SyncInterval *int32 `json:"sync_interval"`
}

// UpdateConfig creates or updates the Hub sync configuration for the tenant.
// PUT /api/v1/sync/config
func (h *HubSyncAPIHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var req syncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_body", "invalid JSON request body")
		return
	}

	if req.HubURL == "" {
		WriteError(w, http.StatusBadRequest, "validation_error", "hub_url is required")
		return
	}
	if req.APIKey == "" {
		WriteError(w, http.StatusBadRequest, "validation_error", "api_key is required")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_tenant_id", "invalid tenant ID in context")
		return
	}

	syncInterval := int32(21600) // default 6 hours
	if req.SyncInterval != nil && *req.SyncInterval > 0 {
		syncInterval = *req.SyncInterval
	}

	state, err := h.queries.UpsertHubSyncState(ctx, sqlcgen.UpsertHubSyncStateParams{
		TenantID:     tid,
		HubUrl:       req.HubURL,
		ApiKey:       req.APIKey,
		SyncInterval: syncInterval,
	})
	if err != nil {
		slog.ErrorContext(ctx, "hub sync config: upsert failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save hub sync configuration")
		return
	}

	emitEvent(ctx, h.eventBus, events.HubSyncConfigUpdated, "hub_sync", tenantID, tenantID, map[string]any{
		"hub_url":       req.HubURL,
		"sync_interval": syncInterval,
	})

	WriteJSON(w, http.StatusOK, toSyncStatusResponse(state))
}
