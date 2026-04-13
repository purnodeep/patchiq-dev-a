package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// JWTConfig holds the configuration for JWT validation middleware.
type JWTConfig struct {
	// Issuer is the expected "iss" claim value (e.g. Zitadel issuer URL).
	Issuer string
	// JWKSURL is the URL to fetch the JSON Web Key Set from.
	JWKSURL string
	// CookieName is the name of the cookie containing the JWT.
	CookieName string
	// DefaultTenantID is used when the JWT does not contain an org ID claim.
	DefaultTenantID string
	// DevMode enables the header-based auth bypass (X-Tenant-ID + X-User-ID).
	// Must be explicitly set to true; defaults to false for safety.
	DevMode bool
}

// jwksCache holds a cached JWKS with expiry for thread-safe access.
type jwksCache struct {
	mu      sync.RWMutex
	keys    *jose.JSONWebKeySet
	fetched time.Time
	ttl     time.Duration
	url     string
}

func newJWKSCache(url string, ttl time.Duration) *jwksCache {
	return &jwksCache{url: url, ttl: ttl}
}

// get returns cached JWKS or fetches fresh if expired/missing.
func (c *jwksCache) get() (*jose.JSONWebKeySet, error) {
	c.mu.RLock()
	if c.keys != nil && time.Since(c.fetched) < c.ttl {
		keys := c.keys
		c.mu.RUnlock()
		return keys, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if c.keys != nil && time.Since(c.fetched) < c.ttl {
		return c.keys, nil
	}

	keys, err := fetchJWKS(c.url)
	if err != nil {
		return nil, err
	}
	c.keys = keys
	c.fetched = time.Now()
	return keys, nil
}

func fetchJWKS(url string) (*jose.JSONWebKeySet, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch JWKS from %s: unexpected status %d", url, resp.StatusCode)
	}

	var jwks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS from %s: %w", url, err)
	}
	return &jwks, nil
}

// zitadelClaims extends standard JWT claims with the Zitadel org ID.
type zitadelClaims struct {
	OrgID string `json:"urn:zitadel:iam:org:id"`
}

// localSigningKey is set once at startup to allow the JWT middleware
// to validate locally-signed HMAC-SHA256 tokens from direct login.
var (
	localSigningKey    []byte
	localSigningKeyMu  sync.Mutex
	localSigningKeySet bool
)

// SetLocalSigningKey sets the signing key exactly once. Subsequent calls are no-ops.
// Panics if called with an empty key to prevent silent auth failures.
func SetLocalSigningKey(key []byte) {
	if len(key) == 0 {
		panic("auth: SetLocalSigningKey called with empty key")
	}
	localSigningKeyMu.Lock()
	defer localSigningKeyMu.Unlock()
	if localSigningKeySet {
		return
	}
	localSigningKey = key
	localSigningKeySet = true
}

// LocalSigningKeyForTest exposes the current key for tests only.
func LocalSigningKeyForTest() []byte {
	localSigningKeyMu.Lock()
	defer localSigningKeyMu.Unlock()
	return localSigningKey
}

// ResetLocalSigningKeyForTest clears the signing key so tests can set a new one.
// Must only be called from tests.
func ResetLocalSigningKeyForTest() {
	localSigningKeyMu.Lock()
	defer localSigningKeyMu.Unlock()
	localSigningKey = nil
	localSigningKeySet = false
}

// validateLocalJWT validates an HMAC-SHA256 signed JWT minted by the
// direct login handler. Returns (sub, orgID, true) on success.
func validateLocalJWT(tokenStr string) (sub string, orgID string, ok bool) {
	if len(localSigningKey) == 0 {
		return "", "", false
	}

	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return "", "", false
	}

	// Check header is HS256.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", false
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "HS256" {
		return "", "", false
	}

	// Verify HMAC signature.
	mac := hmac.New(sha256.New, localSigningKey)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return "", "", false
	}

	// Decode claims.
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}

	var claims struct {
		Sub   string          `json:"sub"`
		OrgID json.RawMessage `json:"urn:zitadel:iam:org:id"`
		Exp   int64           `json:"exp"`
		Iss   string          `json:"iss"`
	}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return "", "", false
	}

	// Check issuer and expiry.
	if claims.Iss != "patchiq-local" {
		return "", "", false
	}
	if claims.Exp < time.Now().Unix() {
		return "", "", false
	}

	// Extract org ID — can be a nested object like {"orgId": {"roles": {}}}
	// where the key is the org ID.
	var orgIDStr string
	if len(claims.OrgID) > 0 {
		// Try as object — keys are org IDs.
		var orgMap map[string]any
		if err := json.Unmarshal(claims.OrgID, &orgMap); err == nil {
			for k := range orgMap {
				orgIDStr = k
				break
			}
		}
		// Try as plain string.
		if orgIDStr == "" {
			var s string
			if err := json.Unmarshal(claims.OrgID, &s); err == nil {
				orgIDStr = s
			}
		}
	}

	if claims.Sub == "" || orgIDStr == "" {
		return "", "", false
	}

	return claims.Sub, orgIDStr, true
}

// NewJWTMiddleware returns HTTP middleware that validates a JWT from the
// configured cookie, verifies its signature against JWKS, checks issuer and
// expiry, and sets user ID and tenant ID in the request context.
func NewJWTMiddleware(cfg JWTConfig) func(http.Handler) http.Handler {
	cache := newJWKSCache(cfg.JWKSURL, 5*time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cfg.CookieName)
			if err != nil {
				// Dev fallback: only active when DevMode is explicitly enabled.
				if cfg.DevMode {
					tid := r.Header.Get("X-Tenant-ID")
					uid := r.Header.Get("X-User-ID")
					if tid != "" && uid != "" {
						slog.WarnContext(r.Context(), "jwt middleware: dev bypass active — header-based auth",
							"user_id", uid, "tenant_id", tid, "path", r.URL.Path)
						ctx := user.WithUserID(r.Context(), uid)
						ctx = tenant.WithTenantID(ctx, tid)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}

				slog.WarnContext(r.Context(), "jwt middleware: missing session cookie",
					"cookie_name", cfg.CookieName,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "missing session cookie")
				return
			}

			// Try parsing as a locally-signed HMAC JWT first (from direct login).
			if sub, orgID, ok := validateLocalJWT(cookie.Value); ok {
				ctx := user.WithUserID(r.Context(), sub)
				ctx = tenant.WithTenantID(ctx, orgID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Fall back to Zitadel JWKS-validated RS256 JWT (from SSO flow).
			tok, err := jwt.ParseSigned(cookie.Value, []jose.SignatureAlgorithm{jose.RS256})
			if err != nil {
				slog.WarnContext(r.Context(), "jwt middleware: failed to parse token",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "invalid token")
				return
			}

			// Fetch JWKS and find matching key.
			jwks, err := cache.get()
			if err != nil {
				slog.ErrorContext(r.Context(), "jwt middleware: failed to fetch JWKS",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "unable to validate token")
				return
			}

			// Get kid from token headers.
			if len(tok.Headers) == 0 {
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "invalid token: missing headers")
				return
			}
			kid := tok.Headers[0].KeyID
			keys := jwks.Key(kid)
			if len(keys) == 0 {
				slog.WarnContext(r.Context(), "jwt middleware: no matching key found in JWKS",
					"kid", kid,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "invalid token: unknown signing key")
				return
			}

			// Verify signature and extract claims.
			var stdClaims jwt.Claims
			var customClaims zitadelClaims
			if err := tok.Claims(keys[0].Key, &stdClaims, &customClaims); err != nil {
				slog.WarnContext(r.Context(), "jwt middleware: failed to verify token signature",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "invalid token signature")
				return
			}

			// Validate standard claims: issuer and expiry.
			expected := jwt.Expected{
				Issuer: cfg.Issuer,
				Time:   time.Now(),
			}
			if err := stdClaims.Validate(expected); err != nil {
				slog.WarnContext(r.Context(), "jwt middleware: token validation failed",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "token validation failed")
				return
			}

			// Extract sub and org ID.
			sub := stdClaims.Subject
			if sub == "" {
				slog.WarnContext(r.Context(), "jwt middleware: missing sub claim",
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "invalid token: missing subject")
				return
			}

			orgID := customClaims.OrgID
			if orgID == "" && cfg.DefaultTenantID != "" {
				orgID = cfg.DefaultTenantID
			} else if orgID == "" {
				slog.WarnContext(r.Context(), "jwt middleware: missing org ID claim and no default tenant",
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "invalid token: missing organization ID")
				return
			}

			ctx := user.WithUserID(r.Context(), sub)
			ctx = tenant.WithTenantID(ctx, orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
