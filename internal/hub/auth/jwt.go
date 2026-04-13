package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// JWTMiddlewareConfig holds configuration for the JWT validation middleware.
type JWTMiddlewareConfig struct {
	CookieName string
	SigningKey []byte
	// DevMode enables the header-based auth bypass (X-Tenant-ID + X-User-ID).
	// Must be explicitly set to true; defaults to false for safety.
	DevMode bool
}

// NewJWTMiddleware returns a chi-compatible HTTP middleware that validates
// HMAC-SHA256 JWTs issued by the hub (issuer: "patchiq-hub").
//
// Token lookup order:
//  1. Cookie named CookieName (primary)
//  2. Authorization: Bearer <token> header (fallback)
//  3. X-Tenant-ID + X-User-ID headers (dev fallback, only when no token found)
//
// On success the validated sub and tenant_id claims are injected into the
// request context via user.WithUserID and tenant.WithTenantID.
func NewJWTMiddleware(cfg JWTMiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r, cfg.CookieName)

			// Try JWT validation first.
			if tokenStr != "" {
				if identity, ok := validateHubJWT(tokenStr, cfg.SigningKey); ok {
					ctx := tenant.WithTenantID(r.Context(), identity.TenantID)
					ctx = user.WithUserID(ctx, identity.Sub)
					ctx = WithEmail(ctx, identity.Email)
					ctx = WithName(ctx, identity.Name)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// Token present but invalid — reject immediately, do not fall through
				// to dev bypass. If a token was sent, the caller intended JWT auth.
				slog.WarnContext(r.Context(), "jwt middleware: token validation failed",
					"method", r.Method, "path", r.URL.Path)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// Dev fallback: only active when DevMode is explicitly enabled
			// and no token was provided at all.
			if cfg.DevMode {
				tenantID := r.Header.Get("X-Tenant-ID")
				userID := r.Header.Get("X-User-ID")
				if tenantID != "" && userID != "" {
					slog.WarnContext(r.Context(), "jwt middleware: dev bypass active — header-based auth",
						"user_id", userID, "tenant_id", tenantID, "path", r.URL.Path)
					ctx := tenant.WithTenantID(r.Context(), tenantID)
					ctx = user.WithUserID(ctx, userID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			writeAuthError(r.Context(), w, http.StatusUnauthorized, "authentication required")
		})
	}
}

// extractToken returns a JWT string from the named cookie or, as a fallback,
// the Authorization: Bearer header. Returns "" when no token is found.
func extractToken(r *http.Request, cookieName string) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}

	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return ""
}

// Context keys for auth claims.
type emailCtxKey struct{}
type nameCtxKey struct{}

// WithEmail stores the user's email in context.
func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailCtxKey{}, email)
}

// EmailFromContext extracts the email from context.
func EmailFromContext(ctx context.Context) string {
	v, _ := ctx.Value(emailCtxKey{}).(string)
	return v
}

// WithName stores the user's display name in context.
func WithName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, nameCtxKey{}, name)
}

// NameFromContext extracts the display name from context.
func NameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(nameCtxKey{}).(string)
	return v
}

// hubClaims is the set of JWT claims we validate and extract.
type hubClaims struct {
	Sub      string `json:"sub"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Iss      string `json:"iss"`
	Exp      int64  `json:"exp"`
}

// jwtIdentity holds the full set of claims extracted from a validated JWT.
type jwtIdentity struct {
	Sub      string
	TenantID string
	Email    string
	Name     string
}

// validateHubJWT validates a compact JWT signed with HMAC-SHA256.
// It checks the algorithm (HS256), issuer ("patchiq-hub"), expiry, and
// HMAC signature. On success it returns the full identity.
func validateHubJWT(tokenStr string, key []byte) (*jwtIdentity, bool) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, false
	}

	// --- validate header ---
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, false
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, false
	}
	if !strings.EqualFold(header.Alg, "HS256") {
		return nil, false
	}

	// --- verify HMAC signature ---
	sigInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(sigInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, false
	}

	// --- decode payload ---
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	var claims hubClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, false
	}

	// --- validate claims ---
	if claims.Iss != "patchiq-hub" {
		return nil, false
	}
	if time.Now().Unix() > claims.Exp {
		return nil, false
	}
	if claims.Sub == "" || claims.TenantID == "" {
		return nil, false
	}

	return &jwtIdentity{
		Sub:      claims.Sub,
		TenantID: claims.TenantID,
		Email:    claims.Email,
		Name:     claims.Name,
	}, true
}

// writeAuthError writes a JSON authentication error response.
func writeAuthError(ctx context.Context, w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"code":    "AUTH_ERROR",
		"message": msg,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to write auth error response", "error", err, "status", status)
	}
}
