package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/skenzeriq/patchiq/internal/shared/user"
)

const HeaderUserID = "X-User-ID"

// maxUserIDLen is the maximum accepted length for the X-User-ID header value.
const maxUserIDLen = 128

// UserMiddleware extracts the user ID from the X-User-ID request header
// and injects it into the request context. Returns 400 if the header is
// missing or invalid.
func UserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(HeaderUserID)
		if raw == "" {
			slog.WarnContext(r.Context(), "request rejected: missing X-User-ID header",
				"method", r.Method,
				"path", r.URL.Path,
			)
			writeAuthError(r.Context(), w, http.StatusBadRequest, "missing X-User-ID header")
			return
		}

		if len(raw) > maxUserIDLen {
			slog.WarnContext(r.Context(), "request rejected: X-User-ID header too long",
				"method", r.Method,
				"path", r.URL.Path,
				"length", len(raw),
			)
			writeAuthError(r.Context(), w, http.StatusBadRequest, "invalid X-User-ID header")
			return
		}

		for _, b := range []byte(raw) {
			if b < 0x20 || b == 0x7f {
				writeAuthError(r.Context(), w, http.StatusBadRequest, "invalid X-User-ID header: contains control characters")
				return
			}
		}

		ctx := user.WithUserID(r.Context(), raw)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission returns HTTP middleware that verifies the user holds
// a permission covering resource:action:* (e.g., an exact match, a wildcard
// action, or super-admin). Scope-narrowed checks should use Evaluator.HasPermission
// directly in handlers. Returns 401 for missing identity, 403 if denied,
// 500 on evaluation error.
func RequirePermission(eval *Evaluator, resource, action string) func(http.Handler) http.Handler {
	required := Permission{Resource: resource, Action: action, Scope: "*"}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := user.UserIDFromContext(r.Context())
			if !ok || userID == "" {
				slog.WarnContext(r.Context(), "permission check: missing user ID in context",
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "missing user identity")
				return
			}

			allowed, err := eval.HasPermission(r.Context(), required)
			if err != nil {
				if errors.Is(err, ErrMissingTenantID) || errors.Is(err, ErrMissingUserID) {
					writeAuthError(r.Context(), w, http.StatusUnauthorized, "missing identity context")
					return
				}
				slog.ErrorContext(r.Context(), "permission check failed",
					"error", err,
					"user_id", userID,
					"required", required.String(),
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusInternalServerError, "permission check failed")
				return
			}

			if !allowed {
				slog.WarnContext(r.Context(), "permission denied",
					"user_id", userID,
					"required", required.String(),
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusForbidden, "permission denied: requires "+required.String())
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthError(ctx context.Context, w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"code":    "AUTH_ERROR",
		"message": msg,
		"details": []any{},
	}); err != nil {
		slog.ErrorContext(ctx, "failed to write error response", "error", err, "status", status)
	}
}
