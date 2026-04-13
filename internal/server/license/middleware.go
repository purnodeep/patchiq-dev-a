package license

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// RequireFeature returns chi middleware that gates access based on the current license.
func RequireFeature(svc *Service, feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !svc.HasFeature(feature) {
				slog.WarnContext(r.Context(), "feature not licensed",
					"feature", feature,
					"method", r.Method,
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				if err := json.NewEncoder(w).Encode(map[string]string{
					"error":   "feature_not_licensed",
					"feature": feature,
					"tier":    svc.CurrentTier(),
				}); err != nil {
					slog.ErrorContext(r.Context(), "encode license error response", "error", err)
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
