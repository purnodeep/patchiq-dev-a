package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/skenzeriq/patchiq/internal/agent/inventory"
)

// HardwareProvider collects hardware information.
type HardwareProvider interface {
	CollectHardware(ctx context.Context) (*inventory.HardwareInfo, error)
}

// HardwareHandler serves GET /api/v1/hardware.
type HardwareHandler struct {
	provider HardwareProvider
}

// NewHardwareHandler creates a HardwareHandler with the given provider.
func NewHardwareHandler(provider HardwareProvider) *HardwareHandler {
	return &HardwareHandler{provider: provider}
}

// Get handles GET /api/v1/hardware.
func (h *HardwareHandler) Get(w http.ResponseWriter, r *http.Request) {
	info, err := h.provider.CollectHardware(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "collect hardware", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to collect hardware info")
		return
	}
	WriteJSON(w, http.StatusOK, info)
}
