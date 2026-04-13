package api

import (
	"context"
	"log/slog"
	"net/http"
)

// LogEntry represents a log record.
type LogEntry struct {
	ID        string  `json:"id"`
	Level     string  `json:"level"`
	Message   string  `json:"message"`
	Source    *string `json:"source,omitempty"`
	Timestamp string  `json:"timestamp"`
}

// LogStore defines the data access interface for logs.
type LogStore interface {
	ListLogs(ctx context.Context, limit int, cursor string, level string) ([]LogEntry, string, int64, error)
}

// LogsHandler serves log-related HTTP endpoints.
type LogsHandler struct {
	store LogStore
}

// NewLogsHandler creates a LogsHandler.
func NewLogsHandler(store LogStore) *LogsHandler {
	return &LogsHandler{store: store}
}

// List handles GET requests for logs.
func (h *LogsHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r)
	cursor := r.URL.Query().Get("cursor")
	level := r.URL.Query().Get("level")

	entries, next, total, err := h.store.ListLogs(r.Context(), limit, cursor, level)
	if err != nil {
		slog.Error("list logs", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to list logs")
		return
	}

	WriteList(w, entries, next, total)
}
