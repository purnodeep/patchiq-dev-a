package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// RoleQuerier defines the sqlc queries needed by RoleHandler.
type RoleQuerier interface {
	ListRolesWithCount(ctx context.Context, arg sqlcgen.ListRolesWithCountParams) ([]sqlcgen.ListRolesWithCountRow, error)
	CountRoles(ctx context.Context, arg sqlcgen.CountRolesParams) (int64, error)
	GetRoleByID(ctx context.Context, arg sqlcgen.GetRoleByIDParams) (sqlcgen.Role, error)
	CreateRole(ctx context.Context, arg sqlcgen.CreateRoleParams) (sqlcgen.Role, error)
	UpdateRole(ctx context.Context, arg sqlcgen.UpdateRoleParams) (sqlcgen.Role, error)
	DeleteRole(ctx context.Context, arg sqlcgen.DeleteRoleParams) (int64, error)
	ListRolePermissions(ctx context.Context, arg sqlcgen.ListRolePermissionsParams) ([]sqlcgen.RolePermission, error)
	DeleteRolePermissions(ctx context.Context, arg sqlcgen.DeleteRolePermissionsParams) error
	CreateRolePermission(ctx context.Context, arg sqlcgen.CreateRolePermissionParams) error
}

// TxQuerierFactory creates a RoleQuerier bound to a transaction.
type TxQuerierFactory func(pgx.Tx) RoleQuerier

// RoleHandler serves role REST API endpoints.
type RoleHandler struct {
	q        RoleQuerier
	txb      TxBeginner
	txQF     TxQuerierFactory
	eventBus domain.EventBus
}

// NewRoleHandler creates a RoleHandler. q, txb, and eventBus are required (panics if nil).
// txQF is optional; if nil, defaults to sqlcgen.New.
func NewRoleHandler(q RoleQuerier, txb TxBeginner, eventBus domain.EventBus, txQF TxQuerierFactory) *RoleHandler {
	if q == nil {
		panic("roles: NewRoleHandler called with nil querier")
	}
	if txb == nil {
		panic("roles: NewRoleHandler called with nil txBeginner")
	}
	if eventBus == nil {
		panic("roles: NewRoleHandler called with nil eventBus")
	}
	if txQF == nil {
		txQF = func(tx pgx.Tx) RoleQuerier { return sqlcgen.New(tx) }
	}
	return &RoleHandler{q: q, txb: txb, txQF: txQF, eventBus: eventBus}
}

// Get handles GET /api/v1/roles/{id}.
func (h *RoleHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid role ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	role, err := h.q.GetRoleByID(ctx, sqlcgen.GetRoleByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "role not found")
			return
		}
		slog.ErrorContext(ctx, "get role", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get role")
		return
	}

	WriteJSON(w, http.StatusOK, role)
}

// List handles GET /api/v1/roles with pagination and search.
func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
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

	roles, err := h.q.ListRolesWithCount(ctx, sqlcgen.ListRolesWithCountParams{
		TenantID: tid,
		Column2:  r.URL.Query().Get("search"),
		Column3:  cursorTS,
		Column4:  cursorUUID,
		Limit:    limit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list roles", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list roles")
		return
	}

	total, err := h.q.CountRoles(ctx, sqlcgen.CountRolesParams{
		TenantID: tid,
		Column2:  r.URL.Query().Get("search"),
	})
	if err != nil {
		slog.ErrorContext(ctx, "count roles", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count roles")
		return
	}

	var nextCursor string
	if len(roles) == int(limit) {
		last := roles[len(roles)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, uuidToString(last.ID))
	}

	WriteList(w, roles, nextCursor, total)
}

type roleRequest struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	ParentRoleID string            `json:"parent_role_id,omitempty"`
	Permissions  []permissionInput `json:"permissions,omitempty"`
}

// Update handles PUT /api/v1/roles/{id}.
// System roles cannot be updated — the SQL WHERE clause filters is_system=false.
func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid role ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body roleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if body.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	for i, p := range body.Permissions {
		if p.Resource == "" || p.Action == "" || p.Scope == "" {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR",
				fmt.Sprintf("permission[%d]: resource, action, and scope are required", i))
			return
		}
	}

	var parentRoleID pgtype.UUID
	if body.ParentRoleID != "" {
		parentRoleID, err = scanUUID(body.ParentRoleID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_PARENT_ROLE_ID", "parent_role_id is not a valid UUID")
			return
		}
	}

	// Run update + permission replace inside a transaction for atomicity.
	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for update role", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update role")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context for update role", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := h.txQF(tx)

	role, err := txQ.UpdateRole(ctx, sqlcgen.UpdateRoleParams{
		ID:           id,
		Name:         body.Name,
		Description:  body.Description,
		ParentRoleID: parentRoleID,
		TenantID:     tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "role not found")
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				WriteError(w, http.StatusConflict, "DUPLICATE", "a role with this name already exists")
				return
			case "23503":
				WriteError(w, http.StatusBadRequest, "INVALID_PARENT_ROLE", "parent role does not exist")
				return
			}
		}
		slog.ErrorContext(ctx, "update role", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update role")
		return
	}

	// Replace permissions: delete all existing, then insert new ones.
	if err := txQ.DeleteRolePermissions(ctx, sqlcgen.DeleteRolePermissionsParams{
		RoleID:   id,
		TenantID: tid,
	}); err != nil {
		slog.ErrorContext(ctx, "delete role permissions", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update role permissions")
		return
	}
	for _, p := range body.Permissions {
		if err := txQ.CreateRolePermission(ctx, sqlcgen.CreateRolePermissionParams{
			TenantID: tid,
			RoleID:   id,
			Resource: p.Resource,
			Action:   p.Action,
			Scope:    p.Scope,
		}); err != nil {
			slog.ErrorContext(ctx, "create role permission", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create role permission")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit update role tx", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update role")
		return
	}

	emitEvent(ctx, h.eventBus, events.RoleUpdated, "role", uuidToString(role.ID), tenantID, role)
	WriteJSON(w, http.StatusOK, role)
}

type permissionInput struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Scope    string `json:"scope"`
}

// Create handles POST /api/v1/roles.
func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body roleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if body.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	for i, p := range body.Permissions {
		if p.Resource == "" || p.Action == "" || p.Scope == "" {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR",
				fmt.Sprintf("permission[%d]: resource, action, and scope are required", i))
			return
		}
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var parentRoleID pgtype.UUID
	if body.ParentRoleID != "" {
		parentRoleID, err = scanUUID(body.ParentRoleID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_PARENT_ROLE_ID", "parent_role_id is not a valid UUID")
			return
		}
	}

	// Run create + permissions inside a transaction for atomicity.
	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for create role", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create role")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context for create role", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := h.txQF(tx)

	role, err := txQ.CreateRole(ctx, sqlcgen.CreateRoleParams{
		TenantID:     tid,
		Name:         body.Name,
		Description:  body.Description,
		ParentRoleID: parentRoleID,
		IsSystem:     false,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				WriteError(w, http.StatusConflict, "DUPLICATE", "a role with this name already exists")
				return
			case "23503":
				WriteError(w, http.StatusBadRequest, "INVALID_PARENT_ROLE", "parent role does not exist")
				return
			}
		}
		slog.ErrorContext(ctx, "create role", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create role")
		return
	}

	for _, p := range body.Permissions {
		if err := txQ.CreateRolePermission(ctx, sqlcgen.CreateRolePermissionParams{
			TenantID: tid,
			RoleID:   role.ID,
			Resource: p.Resource,
			Action:   p.Action,
			Scope:    p.Scope,
		}); err != nil {
			slog.ErrorContext(ctx, "create role permission", "role_id", uuidToString(role.ID), "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create role permission")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit create role tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create role")
		return
	}

	emitEvent(ctx, h.eventBus, events.RoleCreated, "role", uuidToString(role.ID), tenantID, role)
	WriteJSON(w, http.StatusCreated, role)
}

// GetPermissions handles GET /api/v1/roles/{id}/permissions.
func (h *RoleHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid role ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	// Verify role exists.
	if _, err := h.q.GetRoleByID(ctx, sqlcgen.GetRoleByIDParams{ID: id, TenantID: tid}); err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "role not found")
			return
		}
		slog.ErrorContext(ctx, "get role for permissions", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get role")
		return
	}

	perms, err := h.q.ListRolePermissions(ctx, sqlcgen.ListRolePermissionsParams{RoleID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list role permissions", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list role permissions")
		return
	}

	WriteJSON(w, http.StatusOK, perms)
}

// Delete handles DELETE /api/v1/roles/{id}.
// Only custom roles can be deleted; system roles are filtered by the SQL WHERE clause
// (is_system = false), so attempting to delete one returns 403.
func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid role ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.DeleteRole(ctx, sqlcgen.DeleteRoleParams{ID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "delete role", "role_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete role")
		return
	}
	if rows == 0 {
		// Distinguish "not found" from "system role" by checking if the role exists.
		existing, getErr := h.q.GetRoleByID(ctx, sqlcgen.GetRoleByIDParams{ID: id, TenantID: tid})
		if getErr == nil && existing.IsSystem {
			WriteError(w, http.StatusForbidden, "SYSTEM_ROLE", "system roles cannot be deleted")
			return
		}
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "role not found")
		return
	}

	emitEvent(ctx, h.eventBus, events.RoleDeleted, "role", chi.URLParam(r, "id"), tenantID, nil)
	w.WriteHeader(http.StatusNoContent)
}
