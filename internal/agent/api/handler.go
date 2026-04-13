package api

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// HandlerDeps holds the dependencies for the agent API router.
type HandlerDeps struct {
	Status         StatusProvider
	Patches        PatchStore
	History        HistoryStore
	Logs           LogStore
	Settings       SettingsProvider
	Hardware       HardwareProvider
	Software       SoftwareProvider
	Services       ServicesProvider
	Metrics        MetricsProvider
	SettingsUpdate SettingsUpdater
	LogWriter      LogWriter
	ScanTrigger    ScanTrigger
	APIKey         string // shared secret for authenticating local API requests
}

// bearerAuth returns middleware that validates the Authorization: Bearer <key> header.
// If apiKey is empty, the middleware is a no-op (no auth configured).
func bearerAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if auth == "" {
				slog.Warn("agent API: missing Authorization header", "remote", r.RemoteAddr)
				WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")
			if token == auth {
				slog.Warn("agent API: malformed Authorization header", "remote", r.RemoteAddr)
				WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "malformed authorization header"})
				return
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
				slog.Warn("agent API: invalid API key", "remote", r.RemoteAddr)
				WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NewRouter creates a chi router with all agent API endpoints.
func NewRouter(deps HandlerDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	statusH := NewStatusHandler(deps.Status)
	patchesH := NewPatchesHandler(deps.Patches)
	historyH := NewHistoryHandler(deps.History)
	logsH := NewLogsHandler(deps.Logs)
	settingsH := NewSettingsHandler(deps.Settings)
	hardwareH := NewHardwareHandler(deps.Hardware)
	softwareH := NewSoftwareHandler(deps.Software)
	servicesH := NewServicesHandler(deps.Services)
	metricsH := NewMetricsHandler(deps.Metrics)
	settingsUpdateH := NewSettingsUpdateHandler(deps.SettingsUpdate)
	scanH := NewScanHandler(deps.LogWriter, deps.ScanTrigger)

	if deps.APIKey == "" {
		slog.Warn("agent API: no API key configured, /api/v1 endpoints are unauthenticated")
	}

	// Health endpoint is unauthenticated for monitoring/probes.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(bearerAuth(deps.APIKey))

		r.Get("/status", statusH.Get)
		r.Get("/patches/pending", patchesH.ListPending)
		r.Get("/history", historyH.List)
		r.Get("/logs", logsH.List)
		r.Get("/settings", settingsH.Get)
		r.Get("/hardware", hardwareH.Get)
		r.Get("/software", softwareH.Get)
		r.Get("/services", servicesH.Get)
		r.Get("/metrics", metricsH.Get)
		r.Put("/settings", settingsUpdateH.Update)
		r.Post("/scan", scanH.Trigger)
	})

	return r
}
