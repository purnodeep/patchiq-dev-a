package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// TagKeyQuerier is the sqlc surface the TagKeyHandler needs.
type TagKeyQuerier interface {
	ListTagKeys(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.TagKey, error)
	UpsertTagKey(ctx context.Context, arg sqlcgen.UpsertTagKeyParams) (sqlcgen.TagKey, error)
	DeleteTagKey(ctx context.Context, arg sqlcgen.DeleteTagKeyParams) error
}

// TagKeyHandler serves /api/v1/tag-keys. tag_keys is a small metadata
// catalog (catalog.key, description, exclusive flag) that drives the
// AssignTag exclusive semantics and the UI key picker.
type TagKeyHandler struct {
	q        TagKeyQuerier
	eventBus domain.EventBus
}

func NewTagKeyHandler(q TagKeyQuerier, eventBus domain.EventBus) *TagKeyHandler {
	if q == nil {
		panic("tag_keys: NewTagKeyHandler called with nil querier")
	}
	if eventBus == nil {
		panic("tag_keys: NewTagKeyHandler called with nil eventBus")
	}
	return &TagKeyHandler{q: q, eventBus: eventBus}
}

// List handles GET /api/v1/tag-keys.
func (h *TagKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}
	rows, err := h.q.ListTagKeys(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list tag keys", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tag keys")
		return
	}
	WriteJSON(w, http.StatusOK, rows)
}

type upsertTagKeyRequest struct {
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	Exclusive   bool   `json:"exclusive"`
	ValueType   string `json:"value_type,omitempty"`
}

// Upsert handles POST /api/v1/tag-keys. Key is lowercased to satisfy the
// chk_tag_keys_key_lowercase CHECK constraint; the server normalises
// rather than rejecting mixed-case input so the UI doesn't have to do
// cosmetic validation.
func (h *TagKeyHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body upsertTagKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	key := strings.ToLower(strings.TrimSpace(body.Key))
	if key == "" {
		WriteFieldError(w, http.StatusBadRequest, "VALIDATION_ERROR", "key is required", "key")
		return
	}
	valueType := body.ValueType
	if valueType == "" {
		valueType = "string"
	}
	if valueType != "string" && valueType != "enum" {
		WriteFieldError(w, http.StatusBadRequest, "VALIDATION_ERROR", "value_type must be one of: string, enum", "value_type")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	row, err := h.q.UpsertTagKey(ctx, sqlcgen.UpsertTagKeyParams{
		TenantID:    tid,
		Key:         key,
		Description: textFromString(body.Description),
		Exclusive:   body.Exclusive,
		ValueType:   valueType,
	})
	if err != nil {
		slog.ErrorContext(ctx, "upsert tag key", "tenant_id", tenantID, "key", key, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to upsert tag key")
		return
	}
	emitEvent(ctx, h.eventBus, events.TagKeyUpserted, "tag_key", key, tenantID, row)
	WriteJSON(w, http.StatusOK, row)
}

// Delete handles DELETE /api/v1/tag-keys/{key}.
func (h *TagKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	key := strings.ToLower(chi.URLParam(r, "key"))
	if key == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "key path param is required")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}
	if err := h.q.DeleteTagKey(ctx, sqlcgen.DeleteTagKeyParams{TenantID: tid, Key: key}); err != nil {
		slog.ErrorContext(ctx, "delete tag key", "tenant_id", tenantID, "key", key, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete tag key")
		return
	}
	emitEvent(ctx, h.eventBus, events.TagKeyDeleted, "tag_key", key, tenantID, map[string]string{"key": key})
	w.WriteHeader(http.StatusNoContent)
}
