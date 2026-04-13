package v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode json response", "error", err)
	}
}

// WriteError writes a JSON error response matching the ErrorResponse schema.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
		"details": []any{},
	})
}

// WriteFieldError writes a JSON validation error response with a field indicator.
func WriteFieldError(w http.ResponseWriter, status int, code, message, field string) {
	WriteJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
		"field":   field,
		"details": []any{},
	})
}

// ListResponse is the standard paginated list envelope.
type ListResponse struct {
	Data       any   `json:"data"`
	NextCursor any   `json:"next_cursor"`
	TotalCount int64 `json:"total_count"`
}

// WriteList writes a paginated list response.
// nextCursor should be "" for the last page (will be serialized as null).
func WriteList(w http.ResponseWriter, data any, nextCursor string, totalCount int64) {
	var cursor any = nextCursor
	if nextCursor == "" {
		cursor = nil
	}
	WriteJSON(w, http.StatusOK, ListResponse{
		Data:       data,
		NextCursor: cursor,
		TotalCount: totalCount,
	})
}
