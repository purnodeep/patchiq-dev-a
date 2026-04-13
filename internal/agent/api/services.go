package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/skenzeriq/patchiq/internal/agent/inventory"
)

// ServicesProvider retrieves system service information.
type ServicesProvider interface {
	CollectServices(ctx context.Context) ([]inventory.ServiceInfo, error)
}

// ServicesHandler serves GET /api/v1/services.
type ServicesHandler struct {
	provider ServicesProvider
}

// NewServicesHandler creates a ServicesHandler with the given provider.
func NewServicesHandler(provider ServicesProvider) *ServicesHandler {
	return &ServicesHandler{provider: provider}
}

// Get handles GET /api/v1/services.
func (h *ServicesHandler) Get(w http.ResponseWriter, r *http.Request) {
	services, err := h.provider.CollectServices(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "collect services", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to collect services info")
		return
	}
	WriteJSON(w, http.StatusOK, services)
}
