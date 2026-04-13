package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// UserRoleQuerier defines the sqlc queries needed by UserRoleHandler.
type UserRoleQuerier interface {
	AssignUserRole(ctx context.Context, arg sqlcgen.AssignUserRoleParams) error
	RevokeUserRole(ctx context.Context, arg sqlcgen.RevokeUserRoleParams) (int64, error)
	ListUserRoles(ctx context.Context, arg sqlcgen.ListUserRolesParams) ([]sqlcgen.Role, error)
}

// UserRoleHandler serves user-role assignment REST API endpoints.
type UserRoleHandler struct {
	q        UserRoleQuerier
	eventBus domain.EventBus
}

// NewUserRoleHandler creates a UserRoleHandler with the given querier and event bus.
func NewUserRoleHandler(q UserRoleQuerier, eventBus domain.EventBus) *UserRoleHandler {
	if q == nil {
		panic("user_roles: NewUserRoleHandler called with nil querier")
	}
	if eventBus == nil {
		panic("user_roles: NewUserRoleHandler called with nil eventBus")
	}
	return &UserRoleHandler{q: q, eventBus: eventBus}
}

// assignRequest is the JSON body for POST /users/{id}/roles.
type assignRequest struct {
	RoleID string `json:"role_id"`
}

// Assign handles POST /users/{id}/roles — assigns a role to a user.
func (h *UserRoleHandler) Assign(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	userID, _ := url.PathUnescape(chi.URLParam(r, "id"))
	if userID == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_USER_ID", "missing user ID")
		return
	}

	var body assignRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "assign user role: invalid request body")
		return
	}

	if body.RoleID == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_ROLE_ID", "assign user role: role_id is required")
		return
	}

	roleUUID, err := scanUUID(body.RoleID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ROLE_ID", "assign user role: invalid role_id format")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	// AssignUserRole uses ON CONFLICT DO NOTHING, making duplicate assignments idempotent (200 OK).
	if err := h.q.AssignUserRole(ctx, sqlcgen.AssignUserRoleParams{
		TenantID: tid,
		UserID:   userID,
		RoleID:   roleUUID,
	}); err != nil {
		slog.ErrorContext(ctx, "assign user role: store error", "user_id", userID, "role_id", body.RoleID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "assign user role: unexpected error")
		return
	}

	emitEvent(ctx, h.eventBus, events.UserRoleAssigned, "user_role", userID, tenantID, map[string]string{
		"user_id": userID,
		"role_id": body.RoleID,
	})

	WriteJSON(w, http.StatusOK, map[string]string{"status": "role_assigned"})
}

// Revoke handles DELETE /users/{id}/roles/{roleId} — revokes a role from a user.
func (h *UserRoleHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	userID, _ := url.PathUnescape(chi.URLParam(r, "id"))
	if userID == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_USER_ID", "missing user ID")
		return
	}
	roleIDStr := chi.URLParam(r, "roleId")

	roleUUID, err := scanUUID(roleIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ROLE_ID", "revoke user role: invalid role_id format")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.RevokeUserRole(ctx, sqlcgen.RevokeUserRoleParams{
		TenantID: tid,
		UserID:   userID,
		RoleID:   roleUUID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "revoke user role: store error", "user_id", userID, "role_id", roleIDStr, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "revoke user role: unexpected error")
		return
	}

	if rows == 0 {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "user role assignment not found")
		return
	}

	emitEvent(ctx, h.eventBus, events.UserRoleRevoked, "user_role", userID, tenantID, map[string]string{
		"user_id": userID,
		"role_id": roleIDStr,
	})

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /users/{id}/roles — lists all roles assigned to a user.
func (h *UserRoleHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	userID, _ := url.PathUnescape(chi.URLParam(r, "id"))
	if userID == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_USER_ID", "missing user ID")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	roles, err := h.q.ListUserRoles(ctx, sqlcgen.ListUserRolesParams{
		UserID:   userID,
		TenantID: tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list user roles: store error", "user_id", userID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "list user roles: unexpected error")
		return
	}

	WriteJSON(w, http.StatusOK, roles)
}
