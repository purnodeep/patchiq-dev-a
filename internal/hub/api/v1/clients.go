package v1

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"golang.org/x/crypto/bcrypt"
)

// ClientQuerier abstracts the sqlcgen queries used by ClientHandler.
type ClientQuerier interface {
	CreateClient(ctx context.Context, arg sqlcgen.CreateClientParams) (sqlcgen.Client, error)
	GetClientByID(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error)
	GetClientByBootstrapToken(ctx context.Context, bootstrapToken string) (sqlcgen.Client, error)
	GetClientByAPIKeyHash(ctx context.Context, apiKeyHash pgtype.Text) (sqlcgen.Client, error)
	ListClients(ctx context.Context, arg sqlcgen.ListClientsParams) ([]sqlcgen.Client, error)
	CountClients(ctx context.Context, status pgtype.Text) (int64, error)
	CountPendingClients(ctx context.Context) (int64, error)
	UpdateClient(ctx context.Context, arg sqlcgen.UpdateClientParams) (sqlcgen.Client, error)
	ApproveClient(ctx context.Context, arg sqlcgen.ApproveClientParams) (sqlcgen.Client, error)
	DeclineClient(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error)
	SuspendClient(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error)
	DeleteClient(ctx context.Context, id pgtype.UUID) error
	UpdateClientSyncTime(ctx context.Context, arg sqlcgen.UpdateClientSyncTimeParams) error
	ListClientSyncHistory(ctx context.Context, arg sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error)
	CountClientSyncHistory(ctx context.Context, arg sqlcgen.CountClientSyncHistoryParams) (int64, error)
	GetClientEndpointTrend(ctx context.Context, arg sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error)
}

// ClientHandler serves client management endpoints.
type ClientHandler struct {
	queries         ClientQuerier
	eventBus        domain.EventBus
	defaultTenantID string
}

// NewClientHandler creates a new ClientHandler.
func NewClientHandler(queries ClientQuerier, eventBus domain.EventBus, defaultTenantID string) *ClientHandler {
	return &ClientHandler{queries: queries, eventBus: eventBus, defaultTenantID: defaultTenantID}
}

// hashBootstrapToken returns the SHA-256 hex digest of the given token.
// SHA-256 (not bcrypt) because bootstrap tokens are high-entropy (32 random bytes)
// and we need deterministic hashes for DB lookups (WHERE bootstrap_token = $hash).
func hashBootstrapToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type registerClientRequest struct {
	Hostname      string `json:"hostname"`
	Version       string `json:"version"`
	Os            string `json:"os"`
	EndpointCount int32  `json:"endpoint_count"`
	ContactEmail  string `json:"contact_email"`
}

// Register handles POST /api/v1/clients/register.
// Tenant context is optional — unauthenticated Patch Managers use the configured default tenant.
func (h *ClientHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if req.Hostname == "" {
		writeJSONError(w, http.StatusBadRequest, "hostname is required")
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		slog.ErrorContext(r.Context(), "generate bootstrap token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "generate bootstrap token: internal error")
		return
	}
	bootstrapToken := hex.EncodeToString(tokenBytes)

	// Use tenant from auth context if available, otherwise fall back to configured default.
	tenantID, ok := tenant.TenantIDFromContext(r.Context())
	if !ok || tenantID == "" {
		tenantID = h.defaultTenantID
		slog.InfoContext(r.Context(), "client registration using default tenant", "default_tenant_id", h.defaultTenantID, "hostname", req.Hostname)
	}
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "parse tenant ID", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "parse tenant ID: internal error")
		return
	}

	params := sqlcgen.CreateClientParams{
		TenantID:       tenantUUID,
		Hostname:       req.Hostname,
		EndpointCount:  req.EndpointCount,
		BootstrapToken: hashBootstrapToken(bootstrapToken),
	}
	if req.Version != "" {
		params.Version = pgtype.Text{String: req.Version, Valid: true}
	}
	if req.Os != "" {
		params.Os = pgtype.Text{String: req.Os, Valid: true}
	}
	if req.ContactEmail != "" {
		params.ContactEmail = pgtype.Text{String: req.ContactEmail, Valid: true}
	}

	client, err := h.queries.CreateClient(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "create client", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "create client: internal error")
		return
	}

	clientIDStr := uuidToString(client.ID)
	evt := domain.NewSystemEvent(events.ClientRegistered, tenantID, "client", clientIDStr, "register", map[string]any{
		"hostname": req.Hostname,
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit client.registered event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"bootstrap_token": bootstrapToken,
		"status":          "pending",
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode register client response", "error", err)
	}
}

// RegistrationStatus handles GET /api/v1/clients/registration-status.
// Auth via X-Bootstrap-Token header.
func (h *ClientHandler) RegistrationStatus(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Bootstrap-Token")
	if token == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing X-Bootstrap-Token header")
		return
	}

	client, err := h.queries.GetClientByBootstrapToken(r.Context(), hashBootstrapToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "client not found for bootstrap token")
			return
		}
		slog.ErrorContext(r.Context(), "get client by bootstrap token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get client by bootstrap token: internal error")
		return
	}

	resp := map[string]any{
		"status": client.Status,
	}
	if client.Status == "approved" && client.ApiKeyHash.Valid {
		// The API key hash is stored, not the plaintext. The plaintext was returned once on approve.
		// For registration status, we just indicate approved; the key was returned at approve time.
		resp["api_key"] = nil
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "encode registration status response", "error", err)
	}
}

// List handles GET /api/v1/clients.
func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := queryParamInt(r, "limit", 50)
	limit = min(limit, 100)
	if limit < 1 {
		limit = 50
	}
	offset := queryParamInt(r, "offset", 0)

	listParams := sqlcgen.ListClientsParams{
		QueryLimit:  int32(limit),
		QueryOffset: int32(offset),
	}
	var countStatus pgtype.Text

	if v := r.URL.Query().Get("status"); v != "" {
		listParams.Status = pgtype.Text{String: v, Valid: true}
		countStatus = pgtype.Text{String: v, Valid: true}
	}

	clients, err := h.queries.ListClients(r.Context(), listParams)
	if err != nil {
		slog.ErrorContext(r.Context(), "list clients", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list clients: internal error")
		return
	}

	total, err := h.queries.CountClients(r.Context(), countStatus)
	if err != nil {
		slog.ErrorContext(r.Context(), "count clients", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count clients: internal error")
		return
	}

	if clients == nil {
		clients = []sqlcgen.Client{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"clients": clients,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode list clients response", "error", err)
	}
}

// Get handles GET /api/v1/clients/{id}.
func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	client, err := h.queries.GetClientByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "client not found")
			return
		}
		slog.ErrorContext(r.Context(), "get client", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get client: internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"client": client}); err != nil {
		slog.ErrorContext(r.Context(), "encode get client response", "error", err)
	}
}

type updateClientRequest struct {
	Hostname     string `json:"hostname"`
	SyncInterval *int32 `json:"sync_interval"`
	Notes        string `json:"notes"`
}

// Update handles PUT /api/v1/clients/{id}.
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	var req updateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	params := sqlcgen.UpdateClientParams{ID: id}
	if req.Hostname != "" {
		params.Hostname = pgtype.Text{String: req.Hostname, Valid: true}
	}
	if req.SyncInterval != nil {
		params.SyncInterval = pgtype.Int4{Int32: *req.SyncInterval, Valid: true}
	}
	if req.Notes != "" {
		params.Notes = pgtype.Text{String: req.Notes, Valid: true}
	}

	client, err := h.queries.UpdateClient(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "client not found")
			return
		}
		slog.ErrorContext(r.Context(), "update client", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "update client: internal error")
		return
	}

	clientIDStr := uuidToString(client.ID)
	evt := domain.NewSystemEvent(events.ClientUpdated, uuidToString(client.TenantID), "client", clientIDStr, "update", nil)
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit client.updated event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"client": client}); err != nil {
		slog.ErrorContext(r.Context(), "encode update client response", "error", err)
	}
}

// Approve handles POST /api/v1/clients/{id}/approve.
func (h *ClientHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	// Generate API key (32 bytes = 64 hex chars).
	apiKeyBytes := make([]byte, 32)
	if _, err := rand.Read(apiKeyBytes); err != nil {
		slog.ErrorContext(r.Context(), "generate API key", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "generate API key: internal error")
		return
	}
	apiKeyPlaintext := hex.EncodeToString(apiKeyBytes)

	// Hash the API key with bcrypt for storage.
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKeyPlaintext), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(r.Context(), "hash API key", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "hash API key: internal error")
		return
	}

	client, err := h.queries.ApproveClient(r.Context(), sqlcgen.ApproveClientParams{
		ID:         id,
		ApiKeyHash: pgtype.Text{String: string(hash), Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "client not found or not in pending status")
			return
		}
		slog.ErrorContext(r.Context(), "approve client", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "approve client: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	clientIDStr := uuidToString(client.ID)
	evt := domain.NewSystemEvent(events.ClientApproved, tenantID, "client", clientIDStr, "approve", map[string]any{
		"hostname": client.Hostname,
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit client.approved event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"api_key": apiKeyPlaintext,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode approve client response", "error", err)
	}
}

// Decline handles POST /api/v1/clients/{id}/decline.
func (h *ClientHandler) Decline(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	client, err := h.queries.DeclineClient(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "client not found or not in pending status")
			return
		}
		slog.ErrorContext(r.Context(), "decline client", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "decline client: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	clientIDStr := uuidToString(client.ID)
	evt := domain.NewSystemEvent(events.ClientDeclined, tenantID, "client", clientIDStr, "decline", map[string]any{
		"hostname": client.Hostname,
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit client.declined event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"status": "declined"}); err != nil {
		slog.ErrorContext(r.Context(), "encode decline client response", "error", err)
	}
}

// Suspend handles POST /api/v1/clients/{id}/suspend.
func (h *ClientHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	client, err := h.queries.SuspendClient(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "client not found or not in approved status")
			return
		}
		slog.ErrorContext(r.Context(), "suspend client", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "suspend client: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	clientIDStr := uuidToString(client.ID)
	evt := domain.NewSystemEvent(events.ClientSuspended, tenantID, "client", clientIDStr, "suspend", map[string]any{
		"hostname": client.Hostname,
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit client.suspended event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"status": "suspended"}); err != nil {
		slog.ErrorContext(r.Context(), "encode suspend client response", "error", err)
	}
}

// PendingCount handles GET /api/v1/clients/pending-count.
func (h *ClientHandler) PendingCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.queries.CountPendingClients(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "count pending clients", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count pending clients: internal error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"count": count}); err != nil {
		slog.ErrorContext(r.Context(), "encode pending count response", "error", err)
	}
}

// Delete handles DELETE /api/v1/clients/{id}.
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	if err := h.queries.DeleteClient(r.Context(), id); err != nil {
		slog.ErrorContext(r.Context(), "delete client", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "delete client: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	clientIDStr := uuidToString(id)
	evt := domain.NewSystemEvent(events.ClientRemoved, tenantID, "client", clientIDStr, "delete", nil)
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit client.removed event", "error", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// SyncHistory handles GET /api/v1/clients/{id}/sync-history.
func (h *ClientHandler) SyncHistory(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "parse tenant id", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "parse tenant id: internal error")
		return
	}

	limit := queryParamInt(r, "limit", 50)
	limit = min(limit, 100)
	if limit < 1 {
		limit = 50
	}
	offset := queryParamInt(r, "offset", 0)

	items, err := h.queries.ListClientSyncHistory(r.Context(), sqlcgen.ListClientSyncHistoryParams{
		TenantID:    tenantUUID,
		ClientID:    clientID,
		QueryLimit:  int32(limit),
		QueryOffset: int32(offset),
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list client sync history", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list client sync history: internal error")
		return
	}

	total, err := h.queries.CountClientSyncHistory(r.Context(), sqlcgen.CountClientSyncHistoryParams{
		TenantID: tenantUUID,
		ClientID: clientID,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "count client sync history", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count client sync history: internal error")
		return
	}

	if items == nil {
		items = []sqlcgen.ClientSyncHistory{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode sync history response", "error", err)
	}
}

// EndpointTrend handles GET /api/v1/clients/{id}/endpoint-trend.
func (h *ClientHandler) EndpointTrend(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client id: %s", err))
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "parse tenant id", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "parse tenant id: internal error")
		return
	}

	days := queryParamInt(r, "days", 90)
	if days < 1 {
		days = 90
	}
	if days > 365 {
		days = 365
	}

	points, err := h.queries.GetClientEndpointTrend(r.Context(), sqlcgen.GetClientEndpointTrendParams{
		TenantID: tenantUUID,
		ClientID: clientID,
		Days:     int32(days),
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "get client endpoint trend", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get client endpoint trend: internal error")
		return
	}

	if points == nil {
		points = []sqlcgen.GetClientEndpointTrendRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"points": points,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode endpoint trend response", "error", err)
	}
}
