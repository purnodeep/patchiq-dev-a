package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// RegistrationQuerier defines the sqlc queries needed by RegistrationHandler.
type RegistrationQuerier interface {
	CreateRegistration(ctx context.Context, arg sqlcgen.CreateRegistrationParams) (sqlcgen.AgentRegistration, error)
	ListRegistrationsByTenant(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.AgentRegistration, error)
	RevokeRegistration(ctx context.Context, arg sqlcgen.RevokeRegistrationParams) (sqlcgen.AgentRegistration, error)
}

// RegistrationHandler serves registration token REST API endpoints.
type RegistrationHandler struct {
	q        RegistrationQuerier
	eventBus domain.EventBus
}

// NewRegistrationHandler creates a RegistrationHandler.
func NewRegistrationHandler(q RegistrationQuerier, eventBus domain.EventBus) *RegistrationHandler {
	return &RegistrationHandler{q: q, eventBus: eventBus}
}

// registrationResponse is the JSON response for a single registration.
type registrationResponse struct {
	ID                string  `json:"id"`
	TenantID          string  `json:"tenant_id"`
	EndpointID        *string `json:"endpoint_id"`
	RegistrationToken string  `json:"registration_token"`
	Status            string  `json:"status"`
	RegisteredAt      *string `json:"registered_at"`
	CreatedAt         string  `json:"created_at"`
	ExpiresAt         string  `json:"expires_at"`
}

func toRegistrationResponse(r sqlcgen.AgentRegistration) registrationResponse {
	resp := registrationResponse{
		ID:                uuidToString(r.ID),
		TenantID:          uuidToString(r.TenantID),
		RegistrationToken: r.RegistrationToken,
		Status:            r.Status,
	}
	if r.EndpointID.Valid {
		s := uuidToString(r.EndpointID)
		resp.EndpointID = &s
	}
	if r.RegisteredAt.Valid {
		s := r.RegisteredAt.Time.Format(time.RFC3339)
		resp.RegisteredAt = &s
	}
	if r.CreatedAt.Valid {
		resp.CreatedAt = r.CreatedAt.Time.Format(time.RFC3339)
	}
	if r.ExpiresAt.Valid {
		resp.ExpiresAt = r.ExpiresAt.Time.Format(time.RFC3339)
	}
	return resp
}

// Create handles POST /api/v1/registrations.
func (h *RegistrationHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	token := uuid.New().String()
	expiresAt := pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true}
	reg, err := h.q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
		TenantID:          tid,
		RegistrationToken: token,
		ExpiresAt:         expiresAt,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create registration failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create registration token")
		return
	}

	resp := toRegistrationResponse(reg)
	emitEvent(ctx, h.eventBus, events.RegistrationCreated, "registration", resp.ID, tenantID, resp)
	WriteJSON(w, http.StatusCreated, resp)
}

// List handles GET /api/v1/registrations.
func (h *RegistrationHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	regs, err := h.q.ListRegistrationsByTenant(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list registrations failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list registrations")
		return
	}

	statusFilter := r.URL.Query().Get("status")
	items := make([]registrationResponse, 0, len(regs))
	for _, reg := range regs {
		if statusFilter != "" && reg.Status != statusFilter {
			continue
		}
		items = append(items, toRegistrationResponse(reg))
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"total_count": len(items),
	})
}

// Revoke handles DELETE /api/v1/registrations/{id}.
func (h *RegistrationHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid registration ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	reg, err := h.q.RevokeRegistration(ctx, sqlcgen.RevokeRegistrationParams{
		ID:       id,
		TenantID: tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "registration not found or already revoked")
			return
		}
		slog.ErrorContext(ctx, "revoke registration failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to revoke registration")
		return
	}

	resp := toRegistrationResponse(reg)
	emitEvent(ctx, h.eventBus, events.RegistrationRevoked, "registration", resp.ID, tenantID, resp)
	WriteJSON(w, http.StatusOK, resp)
}
