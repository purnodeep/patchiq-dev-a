package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// SessionConfig holds cookie and TTL settings for hub login sessions.
type SessionConfig struct {
	CookieName      string
	CookieDomain    string
	CookieSecure    bool
	AccessTokenTTL  time.Duration
	RememberMeTTL   time.Duration
	SigningKey      []byte
	DefaultTenantID string
	PostLoginURL    string
	// DefaultRole is returned for all hub users until a proper role system is implemented.
	// WARNING: This role is informational only — it is NOT enforced at the route level.
	// TODO(#319): replace with per-user role lookup and enforce via RBAC middleware.
	DefaultRole string
}

var defaultRoleOnce sync.Once

// defaultRole returns the configured default role, falling back to "admin".
func (c *SessionConfig) defaultRole() string {
	if c.DefaultRole != "" {
		return c.DefaultRole
	}
	defaultRoleOnce.Do(func() {
		slog.Warn("hub auth: DefaultRole not configured, falling back to admin")
	})
	return "admin"
}

// InitSigningKey generates a random 32-byte HMAC signing key if none is set.
func (c *SessionConfig) InitSigningKey() {
	if len(c.SigningKey) == 0 {
		c.SigningKey = make([]byte, 32)
		if _, err := rand.Read(c.SigningKey); err != nil {
			panic("hub auth: failed to generate signing key: " + err.Error())
		}
	}
}

// hubJWTClaims is a typed struct for building JWT claim payloads safely
// via json.Marshal, preventing injection through user-controlled fields.
type hubJWTClaims struct {
	Sub      string `json:"sub"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
	Iss      string `json:"iss"`
}

// mintJWT creates an HMAC-SHA256 signed JWT with user claims.
func mintJWT(key []byte, sub, tenantID, email, name string, ttl time.Duration) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	now := time.Now()
	claims := hubJWTClaims{
		Sub:      sub,
		TenantID: tenantID,
		Email:    email,
		Name:     name,
		Iat:      now.Unix(),
		Exp:      now.Add(ttl).Unix(),
		Iss:      "patchiq-hub",
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal JWT claims: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	sigInput := header + "." + payload
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(sigInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return sigInput + "." + sig, nil
}

// MintJWTForTest exposes mintJWT for package-level testing.
// It is only intended for use in _test.go files.
func MintJWTForTest(key []byte, sub, tenantID, email, name string, ttl time.Duration) (string, error) {
	return mintJWT(key, sub, tenantID, email, name, ttl)
}
