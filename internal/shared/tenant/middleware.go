package tenant

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

const HeaderTenantID = "X-Tenant-ID"

// Middleware extracts the tenant ID from the X-Tenant-ID request header
// and injects it into the request context. Returns 400 if the header is
// missing or not a valid UUID.
//
// M0 stub: in M2+ this will also support JWT claims, API keys, and subdomain extraction.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(HeaderTenantID)
		if raw == "" {
			slog.WarnContext(r.Context(), "request rejected: missing X-Tenant-ID header",
				"method", r.Method,
				"path", r.URL.Path,
			)
			writeError(w, http.StatusBadRequest, "missing X-Tenant-ID header")
			return
		}

		parsed, err := uuid.Parse(raw)
		if err != nil {
			slog.WarnContext(r.Context(), "request rejected: invalid X-Tenant-ID header",
				"method", r.Method,
				"path", r.URL.Path,
				"raw_tenant_id", raw,
			)
			writeError(w, http.StatusBadRequest, "invalid X-Tenant-ID: not a valid UUID")
			return
		}

		ctx := WithTenantID(r.Context(), parsed.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		slog.Error("failed to write error response", "error", err, "status", status)
	}
}
