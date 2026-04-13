package api

import (
	"context"
	"log/slog"
	"net/http"
)

// HistoryEntry represents a completed patch operation.
type HistoryEntry struct {
	ID              string  `json:"id"`
	PatchName       string  `json:"patch_name"`
	PatchVersion    string  `json:"patch_version"`
	Action          string  `json:"action"`
	Result          string  `json:"result"`
	ErrorMessage    *string `json:"error_message,omitempty"`
	CompletedAt     string  `json:"completed_at"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
	Size            *string `json:"size,omitempty"`
	RebootRequired  bool    `json:"reboot_required"`
	Stdout          *string `json:"stdout,omitempty"`
	Stderr          *string `json:"stderr,omitempty"`
	ExitCode        *int    `json:"exit_code,omitempty"`
	Attempt         int     `json:"attempt"`
}

// HistoryStore defines the data access interface for patch history.
type HistoryStore interface {
	ListHistory(ctx context.Context, limit int, cursor string, dateRange string) ([]HistoryEntry, string, int64, error)
	InsertHistory(ctx context.Context, entry HistoryEntry) error
}

// HistoryHandler serves history-related HTTP endpoints.
type HistoryHandler struct {
	store HistoryStore
}

// NewHistoryHandler creates a HistoryHandler.
func NewHistoryHandler(store HistoryStore) *HistoryHandler {
	return &HistoryHandler{store: store}
}

// List handles GET requests for patch history.
func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r)
	cursor := r.URL.Query().Get("cursor")
	dateRange := r.URL.Query().Get("date_range")

	entries, next, total, err := h.store.ListHistory(r.Context(), limit, cursor, dateRange)
	if err != nil {
		slog.Error("list history", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to list history")
		return
	}

	WriteList(w, entries, next, total)
}
