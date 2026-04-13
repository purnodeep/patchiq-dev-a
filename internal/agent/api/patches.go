package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
)

// PendingPatch represents a patch awaiting installation.
type PendingPatch struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Severity    string   `json:"severity"`
	Status      string   `json:"status"`
	QueuedAt    string   `json:"queued_at"`
	Size        *string  `json:"size,omitempty"`
	CVSSScore   *float64 `json:"cvss_score,omitempty"`
	CVEIDs      []string `json:"cve_ids"`
	PublishedAt *string  `json:"published_at,omitempty"`
	Source      *string  `json:"source,omitempty"`
}

// PatchStore defines the data access interface for pending patches.
type PatchStore interface {
	ListPending(ctx context.Context, limit int, cursor string) ([]PendingPatch, string, int64, error)
}

// PatchesHandler serves patch-related HTTP endpoints.
type PatchesHandler struct {
	store PatchStore
}

// NewPatchesHandler creates a PatchesHandler.
func NewPatchesHandler(store PatchStore) *PatchesHandler {
	return &PatchesHandler{store: store}
}

// ListPending handles GET requests for pending patches.
func (h *PatchesHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r)
	cursor := r.URL.Query().Get("cursor")

	patches, next, total, err := h.store.ListPending(r.Context(), limit, cursor)
	if err != nil {
		slog.Error("list pending patches", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to list pending patches")
		return
	}

	WriteList(w, patches, next, total)
}

// parseLimit extracts and clamps the limit query parameter.
func parseLimit(r *http.Request) int {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}
	return limit
}
