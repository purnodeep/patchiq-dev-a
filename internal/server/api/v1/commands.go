package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// CommandsQuerier is the subset of store methods required by CommandsHandler.
type CommandsQuerier interface {
	GetCommandByID(ctx context.Context, arg sqlcgen.GetCommandByIDParams) (sqlcgen.Command, error)
}

// CommandsHandler serves command-lookup endpoints used by the PM UI to poll
// the status of commands it has triggered (e.g. the Scan Now button).
type CommandsHandler struct {
	q CommandsQuerier
}

// NewCommandsHandler constructs a CommandsHandler.
func NewCommandsHandler(q CommandsQuerier) *CommandsHandler {
	if q == nil {
		panic("commands: NewCommandsHandler called with nil querier")
	}
	return &CommandsHandler{q: q}
}

// Get handles GET /api/v1/commands/{id}.
func (h *CommandsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid command ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	cmd, err := h.q.GetCommandByID(ctx, sqlcgen.GetCommandByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "command not found")
			return
		}
		slog.ErrorContext(ctx, "get command", "command_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get command")
		return
	}

	resp := map[string]any{
		"id":            uuidToString(cmd.ID),
		"agent_id":      uuidToString(cmd.AgentID),
		"type":          cmd.Type,
		"status":        cmd.Status,
		"created_at":    nullableTime(cmd.CreatedAt),
		"delivered_at":  nullableTime(cmd.DeliveredAt),
		"completed_at":  nullableTime(cmd.CompletedAt),
		"error_message": nullableText(cmd.ErrorMessage),
	}
	WriteJSON(w, http.StatusOK, resp)
}

// nullableTime returns a serializable *time.Time (nil if the timestamptz is null).
// Declared here rather than in helpers.go because commands.go is the only current caller.
func nullableTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
