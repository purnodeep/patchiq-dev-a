package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/skenzeriq/patchiq/internal/agent/inventory"
)

// SoftwareProvider retrieves extended package information.
type SoftwareProvider interface {
	ExtendedPackages(ctx context.Context) ([]inventory.ExtendedPackageInfo, error)
}

// SoftwareHandler serves GET /api/v1/software.
type SoftwareHandler struct {
	provider SoftwareProvider
}

// NewSoftwareHandler creates a SoftwareHandler with the given provider.
func NewSoftwareHandler(provider SoftwareProvider) *SoftwareHandler {
	return &SoftwareHandler{provider: provider}
}

// Get handles GET /api/v1/software.
func (h *SoftwareHandler) Get(w http.ResponseWriter, r *http.Request) {
	pkgs, err := h.provider.ExtendedPackages(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "collect software", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to collect software info")
		return
	}
	WriteJSON(w, http.StatusOK, pkgs)
}
