package v1

import (
	"log/slog"
	"net/http"

	serverlicense "github.com/skenzeriq/patchiq/internal/server/license"
)

// EndpointCounter provides current endpoint count for license status.
type EndpointCounter interface {
	CountAllEndpoints() (int, error)
}

// LicenseHandler serves the license status API.
type LicenseHandler struct {
	svc     *serverlicense.Service
	counter EndpointCounter
}

// NewLicenseHandler creates a LicenseHandler.
func NewLicenseHandler(svc *serverlicense.Service, counter EndpointCounter) *LicenseHandler {
	return &LicenseHandler{svc: svc, counter: counter}
}

// Status handles GET /api/v1/license/status.
func (h *LicenseHandler) Status(w http.ResponseWriter, r *http.Request) {
	status := h.svc.Status()

	if h.counter != nil {
		count, err := h.counter.CountAllEndpoints()
		if err != nil {
			slog.ErrorContext(r.Context(), "count endpoints for license status", "error", err)
		} else {
			status.EndpointUsage.Current = count
		}
	}

	WriteJSON(w, http.StatusOK, status)
}
