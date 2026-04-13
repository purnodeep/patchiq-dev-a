package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// RoleMappingQuerier is the DB interface required by RoleMappingHandler.
type RoleMappingQuerier interface {
	ListRoleMappings(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.ListRoleMappingsRow, error)
	UpsertRoleMapping(ctx context.Context, arg sqlcgen.UpsertRoleMappingParams) (sqlcgen.RoleMapping, error)
	DeleteRoleMappingsByTenant(ctx context.Context, tenantID pgtype.UUID) error
}

// RoleMappingTxQuerierFactory creates a RoleMappingQuerier bound to a transaction.
type RoleMappingTxQuerierFactory func(pgx.Tx) RoleMappingQuerier

// RoleMappingHandler handles GET/PUT /api/v1/settings/role-mapping.
type RoleMappingHandler struct {
	q        RoleMappingQuerier
	txb      TxBeginner
	txQF     RoleMappingTxQuerierFactory
	eventBus domain.EventBus
}

// NewRoleMappingHandler creates a new role mapping handler.
// txQF is optional; if nil, defaults to sqlcgen.New.
func NewRoleMappingHandler(q RoleMappingQuerier, txb TxBeginner, eventBus domain.EventBus, txQF RoleMappingTxQuerierFactory) *RoleMappingHandler {
	if q == nil {
		panic("settings_rolemapping: NewRoleMappingHandler called with nil querier")
	}
	if txb == nil {
		panic("settings_rolemapping: NewRoleMappingHandler called with nil txBeginner")
	}
	if eventBus == nil {
		panic("settings_rolemapping: NewRoleMappingHandler called with nil eventBus")
	}
	if txQF == nil {
		txQF = func(tx pgx.Tx) RoleMappingQuerier { return sqlcgen.New(tx) }
	}
	return &RoleMappingHandler{q: q, txb: txb, txQF: txQF, eventBus: eventBus}
}

// RoleMappingEntry represents a single role mapping.
type RoleMappingEntry struct {
	ExternalRole  string `json:"external_role"`
	PatchIQRoleID string `json:"patchiq_role_id"`
	RoleName      string `json:"role_name,omitempty"`
}

// Get returns all role mappings for the tenant.
func (h *RoleMappingHandler) Get(w http.ResponseWriter, r *http.Request) {
	tid := tenant.MustTenantID(r.Context())
	slog.InfoContext(r.Context(), "role mapping get", "tenant_id", tid)

	pgTID, err := scanUUID(tid)
	if err != nil {
		slog.ErrorContext(r.Context(), "role mapping get: invalid tenant id in context", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.ListRoleMappings(r.Context(), pgTID)
	if err != nil {
		slog.ErrorContext(r.Context(), "role mapping get: query failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list role mappings")
		return
	}

	entries := make([]RoleMappingEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, RoleMappingEntry{
			ExternalRole:  row.ExternalRole,
			PatchIQRoleID: uuidToString(row.PatchiqRoleID),
			RoleName:      row.RoleName,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"data": entries,
	})
}

// Update replaces the role mappings for the tenant.
func (h *RoleMappingHandler) Update(w http.ResponseWriter, r *http.Request) {
	tid := tenant.MustTenantID(r.Context())

	var req struct {
		Mappings []RoleMappingEntry `json:"mappings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	pgTID, err := scanUUID(tid)
	if err != nil {
		slog.ErrorContext(r.Context(), "role mapping update: invalid tenant id in context", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	// Validate all role IDs before starting the transaction
	parsedRoleIDs := make([]pgtype.UUID, len(req.Mappings))
	for i, m := range req.Mappings {
		roleID, parseErr := scanUUID(m.PatchIQRoleID)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_ROLE_ID", "invalid patchiq_role_id: "+m.PatchIQRoleID)
			return
		}
		parsedRoleIDs[i] = roleID
	}

	// Delete + upsert inside a transaction so membership replacement is atomic
	ctx := r.Context()
	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "role mapping update: begin tx failed", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update role mappings")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tid); err != nil {
		slog.ErrorContext(ctx, "role mapping update: set tenant config failed", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update role mappings")
		return
	}

	txQ := h.txQF(tx)

	if err := txQ.DeleteRoleMappingsByTenant(ctx, pgTID); err != nil {
		slog.ErrorContext(ctx, "role mapping update: delete existing failed", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to clear existing role mappings")
		return
	}

	results := make([]RoleMappingEntry, 0, len(req.Mappings))
	for i, m := range req.Mappings {
		_, upsertErr := txQ.UpsertRoleMapping(ctx, sqlcgen.UpsertRoleMappingParams{
			TenantID:      pgTID,
			ExternalRole:  m.ExternalRole,
			PatchiqRoleID: parsedRoleIDs[i],
		})
		if upsertErr != nil {
			slog.ErrorContext(ctx, "role mapping update: upsert failed",
				"external_role", m.ExternalRole, "error", upsertErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update role mapping")
			return
		}
		results = append(results, RoleMappingEntry{
			ExternalRole:  m.ExternalRole,
			PatchIQRoleID: m.PatchIQRoleID,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "role mapping update: commit failed", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update role mappings")
		return
	}

	emitEvent(r.Context(), h.eventBus, events.SettingsRoleMappingUpdated, "settings", tid, tid, map[string]any{
		"count": len(req.Mappings),
	})

	slog.InfoContext(r.Context(), "role mapping update", "tenant_id", tid, "count", len(req.Mappings))
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": results,
	})
}
