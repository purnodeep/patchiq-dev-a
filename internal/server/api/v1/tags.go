package v1

import (
	"context"
	"encoding/json"
	"fmt"
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

// TagQuerier defines the sqlc queries needed by TagHandler. As of
// migration 060 tags are key=value; `Name` no longer exists.
type TagQuerier interface {
	ListTags(ctx context.Context, arg sqlcgen.ListTagsParams) ([]sqlcgen.ListTagsRow, error)
	ListDistinctTagKeys(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.ListDistinctTagKeysRow, error)
	CreateTag(ctx context.Context, arg sqlcgen.CreateTagParams) (sqlcgen.Tag, error)
	GetTagByID(ctx context.Context, arg sqlcgen.GetTagByIDParams) (sqlcgen.Tag, error)
	UpdateTag(ctx context.Context, arg sqlcgen.UpdateTagParams) (sqlcgen.Tag, error)
	DeleteTag(ctx context.Context, arg sqlcgen.DeleteTagParams) error
	BulkAssignTag(ctx context.Context, arg sqlcgen.BulkAssignTagParams) error
	RemoveTagFromEndpoint(ctx context.Context, arg sqlcgen.RemoveTagFromEndpointParams) error
	RemoveEndpointTagsByKey(ctx context.Context, arg sqlcgen.RemoveEndpointTagsByKeyParams) error
	IsKeyExclusive(ctx context.Context, arg sqlcgen.IsKeyExclusiveParams) (bool, error)
}

// TagHandler serves tag REST API endpoints.
type TagHandler struct {
	q        TagQuerier
	txb      TxBeginner
	eventBus domain.EventBus
}

// NewTagHandler creates a TagHandler.
func NewTagHandler(q TagQuerier, txb TxBeginner, eventBus domain.EventBus) *TagHandler {
	if q == nil {
		panic("tags: NewTagHandler called with nil querier")
	}
	if txb == nil {
		panic("tags: NewTagHandler called with nil txBeginner")
	}
	if eventBus == nil {
		panic("tags: NewTagHandler called with nil eventBus")
	}
	return &TagHandler{q: q, txb: txb, eventBus: eventBus}
}

// List handles GET /api/v1/tags. Accepts ?key=foo to filter by key.
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	tags, err := h.q.ListTags(ctx, sqlcgen.ListTagsParams{
		TenantID:  tid,
		KeyFilter: r.URL.Query().Get("key"),
	})
	if err != nil {
		slog.ErrorContext(ctx, "list tags", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tags")
		return
	}

	WriteJSON(w, http.StatusOK, tags)
}

// ListKeys handles GET /api/v1/tags/keys. Returns each distinct key with
// its value count so the UI can populate the key picker without paging
// through every (key,value) row.
func (h *TagHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}
	rows, err := h.q.ListDistinctTagKeys(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list distinct tag keys", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tag keys")
		return
	}
	WriteJSON(w, http.StatusOK, rows)
}

type createTagRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// Create handles POST /api/v1/tags. Key is normalised to lowercase to
// match the DB CHECK constraint; value is preserved as supplied because
// value case is user-meaningful (e.g. version strings, hostnames).
func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body createTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	key := strings.ToLower(strings.TrimSpace(body.Key))
	value := strings.TrimSpace(body.Value)
	if key == "" {
		WriteFieldError(w, http.StatusBadRequest, "VALIDATION_ERROR", "key is required", "key")
		return
	}
	if value == "" {
		WriteFieldError(w, http.StatusBadRequest, "VALIDATION_ERROR", "value is required", "value")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	tag, err := h.q.CreateTag(ctx, sqlcgen.CreateTagParams{
		TenantID:    tid,
		Key:         key,
		Value:       value,
		Description: textFromString(body.Description),
	})
	if err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "DUPLICATE_TAG", fmt.Sprintf("tag %s=%s already exists", key, value))
			return
		}
		slog.ErrorContext(ctx, "create tag", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create tag")
		return
	}

	emitEvent(ctx, h.eventBus, events.TagCreated, "tag", uuidToString(tag.ID), tenantID, tag)
	WriteJSON(w, http.StatusCreated, tag)
}

// Get handles GET /api/v1/tags/{id}.
func (h *TagHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid tag ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	tag, err := h.q.GetTagByID(ctx, sqlcgen.GetTagByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "tag not found")
			return
		}
		slog.ErrorContext(ctx, "get tag", "tag_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get tag")
		return
	}

	WriteJSON(w, http.StatusOK, tag)
}

type updateTagRequest struct {
	Description string `json:"description,omitempty"`
}

// Update handles PUT /api/v1/tags/{id}. Only description is mutable;
// key/value are immutable once created (the UI should delete + recreate
// if the user wants to change them — this matches cloud-native tag
// semantics where (key, value) is the identity).
func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid tag ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body updateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	tag, err := h.q.UpdateTag(ctx, sqlcgen.UpdateTagParams{
		ID:          id,
		Description: textFromString(body.Description),
		TenantID:    tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "tag not found")
			return
		}
		slog.ErrorContext(ctx, "update tag", "tag_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update tag")
		return
	}

	emitEvent(ctx, h.eventBus, events.TagUpdated, "tag", uuidToString(tag.ID), tenantID, tag)
	WriteJSON(w, http.StatusOK, tag)
}

// Delete handles DELETE /api/v1/tags/{id}.
func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid tag ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	tag, err := h.q.GetTagByID(ctx, sqlcgen.GetTagByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "tag not found")
			return
		}
		slog.ErrorContext(ctx, "get tag for delete", "tag_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete tag")
		return
	}
	if err := h.q.DeleteTag(ctx, sqlcgen.DeleteTagParams{ID: id, TenantID: tid}); err != nil {
		slog.ErrorContext(ctx, "delete tag", "tag_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete tag")
		return
	}

	emitEvent(ctx, h.eventBus, events.TagDeleted, "tag", uuidToString(tag.ID), tenantID, tag)
	w.WriteHeader(http.StatusNoContent)
}

type assignTagRequest struct {
	EndpointIDs []string `json:"endpoint_ids"`
}

// Assign handles POST /api/v1/tags/{id}/assign. If the tag's key is
// flagged exclusive in tag_keys, any existing values for the same key on
// each target endpoint are stripped first so the endpoint ends up with
// exactly one value for that key (e.g. env=prod replaces env=staging).
func (h *TagHandler) Assign(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid tag ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body assignTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if len(body.EndpointIDs) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "endpoint_ids must not be empty")
		return
	}

	endpointUUIDs := make([]pgtype.UUID, 0, len(body.EndpointIDs))
	for _, eid := range body.EndpointIDs {
		uid, uidErr := scanUUID(eid)
		if uidErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", fmt.Sprintf("endpoint_id %q is not a valid UUID", eid))
			return
		}
		endpointUUIDs = append(endpointUUIDs, uid)
	}

	// Look up the tag so we know its key; we need the key to decide
	// whether to enforce exclusivity.
	tag, err := h.q.GetTagByID(ctx, sqlcgen.GetTagByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "tag not found")
			return
		}
		slog.ErrorContext(ctx, "get tag for assign", "tag_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign tag")
		return
	}

	// Atomic strip+assign so we never leave endpoints carrying two values
	// for an exclusive key even if the second step errors. The exclusivity
	// read happens *inside* the tx against txQ (not h.q) to close the
	// TOCTOU window where an admin could flip tag_keys.exclusive between
	// the check and the strip.
	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for assign tag", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign tag")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context for assign tag", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := sqlcgen.New(tx)

	// Exclusivity is opt-in per key and registered in tag_keys. Missing
	// row means the key is non-exclusive (default). Any other error is
	// fatal — falling back to non-exclusive would silently commit a
	// duplicate-value assign and break the invariant the exclusive flag
	// is designed to enforce.
	exclusive := false
	if ex, err := txQ.IsKeyExclusive(ctx, sqlcgen.IsKeyExclusiveParams{TenantID: tid, Key: tag.Key}); err == nil {
		exclusive = ex
	} else if !isNotFound(err) {
		slog.ErrorContext(ctx, "check key exclusivity", "key", tag.Key, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "EXCLUSIVITY_CHECK_FAILED", "failed to verify exclusivity for tag key")
		return
	}

	if exclusive {
		if err := txQ.RemoveEndpointTagsByKey(ctx, sqlcgen.RemoveEndpointTagsByKeyParams{
			TenantID:    tid,
			EndpointIds: endpointUUIDs,
			Key:         tag.Key,
		}); err != nil {
			slog.ErrorContext(ctx, "strip exclusive key before assign", "key", tag.Key, "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign tag")
			return
		}
	}

	if err := txQ.BulkAssignTag(ctx, sqlcgen.BulkAssignTagParams{
		EndpointIds: endpointUUIDs,
		TagID:       id,
		TenantID:    tid,
		Source:      "manual",
	}); err != nil {
		slog.ErrorContext(ctx, "bulk assign tag to endpoints", "tag_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign tag to endpoints")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit assign tag tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign tag")
		return
	}

	emitEvent(ctx, h.eventBus, events.EndpointTagged, "tag", uuidToString(id), tenantID, map[string]any{
		"tag_id":       uuidToString(id),
		"endpoint_ids": body.EndpointIDs,
		"exclusive":    exclusive,
	})
	w.WriteHeader(http.StatusNoContent)
}

// Unassign handles POST /api/v1/tags/{id}/unassign.
func (h *TagHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid tag ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body assignTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if len(body.EndpointIDs) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "endpoint_ids must not be empty")
		return
	}

	for _, eid := range body.EndpointIDs {
		uid, uidErr := scanUUID(eid)
		if uidErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", fmt.Sprintf("endpoint_id %q is not a valid UUID", eid))
			return
		}
		if err := h.q.RemoveTagFromEndpoint(ctx, sqlcgen.RemoveTagFromEndpointParams{
			EndpointID: uid,
			TagID:      id,
			TenantID:   tid,
		}); err != nil {
			slog.ErrorContext(ctx, "remove tag from endpoint", "tag_id", chi.URLParam(r, "id"), "endpoint_id", eid, "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to unassign tag from endpoint")
			return
		}
	}

	emitEvent(ctx, h.eventBus, events.EndpointUntagged, "tag", uuidToString(id), tenantID, map[string]any{
		"tag_id":       uuidToString(id),
		"endpoint_ids": body.EndpointIDs,
	})
	w.WriteHeader(http.StatusNoContent)
}
