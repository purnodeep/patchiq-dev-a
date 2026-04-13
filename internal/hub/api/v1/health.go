package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Pinger checks database connectivity.
type Pinger interface {
	Ping(ctx context.Context) error
}

// CheckFunc is a named health check that returns nil on success.
type CheckFunc func(ctx context.Context) error

// HealthHandler serves health and readiness endpoints.
type HealthHandler struct {
	checks    map[string]CheckFunc
	startTime time.Time
	version   string
}

// NewHealthHandler creates a new HealthHandler.
// If pinger is nil, the database check reports the pool as not initialized.
// Additional checks (e.g. valkey) can be passed via the checks map.
func NewHealthHandler(pinger Pinger, startTime time.Time, version string, checks map[string]CheckFunc) *HealthHandler {
	merged := make(map[string]CheckFunc)
	if pinger != nil {
		merged["database"] = pinger.Ping
	} else {
		merged["database"] = func(_ context.Context) error {
			return errDBPoolNotInitialized
		}
	}
	for k, v := range checks {
		merged[k] = v
	}
	return &HealthHandler{checks: merged, startTime: startTime, version: version}
}

var errDBPoolNotInitialized = &dbPoolError{}

type dbPoolError struct{}

func (e *dbPoolError) Error() string { return "database pool not initialized" }

// Health returns 200 OK with uptime and version, indicating the process is alive.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"uptime":  time.Since(h.startTime).String(),
		"version": h.version,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode health response", "error", err)
	}
}

// Ready returns 200 if all checks pass, 503 if any fail.
// Response includes per-check status details.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	results := make(map[string]string, len(h.checks))
	allOK := true

	for name, check := range h.checks {
		if err := check(r.Context()); err != nil {
			results[name] = "error"
			allOK = false
			slog.ErrorContext(r.Context(), "readiness check failed", "check", name, "error", err)
		} else {
			results[name] = "ok"
		}
	}

	status := "ready"
	httpStatus := http.StatusOK
	if !allOK {
		status = "unavailable"
		httpStatus = http.StatusServiceUnavailable
	}

	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status": status,
		"checks": results,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode ready response", "error", err)
	}
}
