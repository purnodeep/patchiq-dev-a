package organization

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// HeaderOrganizationID is the optional HTTP header that lets a client target
// an org explicitly (as opposed to letting it be derived from JWT claims).
// It is used for MSP dashboard endpoints that aggregate across tenants.
const HeaderOrganizationID = "X-Organization-ID"

// Middleware extracts an optional organization ID from the X-Organization-ID
// header and injects it into the context. Unlike the tenant middleware, a
// missing header is NOT an error — most endpoints derive the active org from
// the JWT session rather than the header. An invalid UUID, however, is a
// 400.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(HeaderOrganizationID)
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		parsed, err := uuid.Parse(raw)
		if err != nil {
			slog.WarnContext(r.Context(), "request rejected: invalid X-Organization-ID header",
				"method", r.Method,
				"path", r.URL.Path,
				"raw_org_id", raw,
			)
			writeError(w, http.StatusBadRequest, "invalid X-Organization-ID: not a valid UUID")
			return
		}
		ctx := WithOrgID(r.Context(), parsed.String())
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
