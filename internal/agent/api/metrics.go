package api

import (
	"log/slog"
	"net/http"
)

// MetricsProvider returns live system metrics.
type MetricsProvider interface {
	Metrics() (any, error)
}

// MetricsHandler serves GET /api/v1/metrics.
type MetricsHandler struct {
	provider MetricsProvider
}

// NewMetricsHandler creates a MetricsHandler with the given provider.
func NewMetricsHandler(provider MetricsProvider) *MetricsHandler {
	return &MetricsHandler{provider: provider}
}

// Get handles GET /api/v1/metrics.
func (h *MetricsHandler) Get(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.provider.Metrics()
	if err != nil {
		slog.ErrorContext(r.Context(), "collect metrics", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to collect system metrics")
		return
	}
	WriteJSON(w, http.StatusOK, metrics)
}
