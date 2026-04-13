package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// SessionConfig holds cookie and TTL settings for login sessions.
type SessionConfig struct {
	CookieName      string
	CookieDomain    string
	CookieSecure    bool
	AccessTokenTTL  time.Duration
	RememberMeTTL   time.Duration
	SigningKey      []byte // HMAC key for locally-signed JWTs
	DefaultTenantID string // PatchIQ tenant UUID to use (Zitadel org IDs are not UUIDs)
}

// InitSigningKey generates a random 32-byte HMAC signing key if none is set.
func (c *SessionConfig) InitSigningKey() {
	if len(c.SigningKey) == 0 {
		c.SigningKey = make([]byte, 32)
		if _, err := rand.Read(c.SigningKey); err != nil {
			panic("auth: failed to generate signing key: " + err.Error())
		}
	}
}

// localJWTClaims is a typed struct for building JWT claim payloads safely
// via json.Marshal, preventing injection through user-controlled fields.
type localJWTClaims struct {
	Sub   string                    `json:"sub"`
	OrgID map[string]map[string]any `json:"urn:zitadel:iam:org:id"`
	Email string                    `json:"email"`
	Name  string                    `json:"name"`
	Iat   int64                     `json:"iat"`
	Exp   int64                     `json:"exp"`
	Iss   string                    `json:"iss"`
}

// mintJWT creates an HMAC-SHA256 signed JWT with the given claims.
// This is used for direct login sessions where we can't get a Zitadel-signed JWT.
func mintJWT(key []byte, sub, orgID, email, name string, ttl time.Duration) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	now := time.Now()
	claims := localJWTClaims{
		Sub:   sub,
		OrgID: map[string]map[string]any{orgID: {"roles": map[string]any{}}},
		Email: email,
		Name:  name,
		Iat:   now.Unix(),
		Exp:   now.Add(ttl).Unix(),
		Iss:   "patchiq-local",
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

// LoginHandler handles POST /api/v1/auth/login and POST /api/v1/auth/forgot-password.
type LoginHandler struct {
	zitadel  *ZitadelClient
	eventBus domain.EventBus
	cfg      SessionConfig
}

// NewLoginHandler creates a LoginHandler with the given dependencies.
func NewLoginHandler(zitadel *ZitadelClient, eventBus domain.EventBus, cfg SessionConfig) *LoginHandler {
	return &LoginHandler{
		zitadel:  zitadel,
		eventBus: eventBus,
		cfg:      cfg,
	}
}

// loginRequest is the JSON body for POST /api/v1/auth/login.
type loginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// loginResponse is the JSON body returned on successful login.
type loginResponse struct {
	UserID   string `json:"user_id,omitempty"`
	TenantID string `json:"tenant_id,omitempty"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email"`
}

// Login handles POST /api/v1/auth/login.
// Flow: decode body -> validate -> authenticate via Zitadel -> exchange token -> set cookie -> emit event -> respond.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.WarnContext(ctx, "login: failed to decode request body", "error", err)
		writeAuthError(ctx, w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeAuthError(ctx, w, http.StatusBadRequest, "Email and password are required.")
		return
	}

	// Authenticate via Zitadel.
	authResult, err := h.zitadel.Authenticate(ctx, req.Email, req.Password)
	if err != nil {
		slog.WarnContext(ctx, "login: authentication failed",
			"email", req.Email,
			"error", err,
		)
		writeAuthError(ctx, w, http.StatusUnauthorized,
			"That email/password combination didn't work. Try again?")
		return
	}

	// Fetch session details to get user info (user ID, org ID, display name).
	sessionInfo, err := h.zitadel.GetSessionInfo(ctx, authResult.SessionID)
	if err != nil {
		slog.ErrorContext(ctx, "login: failed to get session info",
			"email", req.Email,
			"session_id", authResult.SessionID,
			"error", err,
		)
		writeAuthError(ctx, w, http.StatusInternalServerError,
			"Something went wrong. Please try again.")
		return
	}

	// Map Zitadel org ID to PatchIQ tenant UUID. Zitadel uses snowflake IDs
	// (e.g., "364916209130471430") which are not UUIDs. Use the configured
	// default tenant ID for PatchIQ's RLS-based tenant isolation.
	tenantID := h.cfg.DefaultTenantID
	if tenantID == "" {
		tenantID = sessionInfo.OrgID // fallback to raw org ID
	}

	// Use the login email as the user ID to match PatchIQ's user_roles table
	// which stores email-based identifiers (not Zitadel numeric IDs).
	userID := req.Email

	// Mint a locally-signed JWT with user claims.
	// Signing key is initialized once at startup (cmd/server/main.go),
	// not per-request, to avoid race conditions.
	ttl := h.cfg.AccessTokenTTL
	if req.RememberMe && h.cfg.RememberMeTTL > 0 {
		ttl = h.cfg.RememberMeTTL
	}

	token, err := mintJWT(h.cfg.SigningKey, userID, tenantID, req.Email, sessionInfo.DisplayName, ttl)
	if err != nil {
		slog.ErrorContext(ctx, "login: failed to mint JWT",
			"email", req.Email,
			"error", err,
		)
		writeAuthError(ctx, w, http.StatusInternalServerError,
			"Something went wrong. Please try again.")
		return
	}

	// Set session cookie.
	cookieName := h.cfg.CookieName
	if cookieName == "" {
		cookieName = "piq_session"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Emit auth.login event.
	if err := h.eventBus.Emit(ctx, domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       events.AuthLogin,
		ActorID:    userID,
		ActorType:  domain.ActorUser,
		Resource:   "auth",
		ResourceID: userID,
		Action:     "login",
		TenantID:   tenantID,
		Payload:    map[string]string{"email": req.Email, "user_id": userID},
		Timestamp:  time.Now(),
	}); err != nil {
		slog.ErrorContext(ctx, "login: failed to emit auth.login event",
			"email", req.Email,
			"error", err,
		)
	}

	slog.InfoContext(ctx, "login: user authenticated successfully",
		"email", req.Email,
		"user_id", userID,
		"tenant_id", tenantID,
		"remember_me", req.RememberMe,
	)

	// Return user info.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(loginResponse{
		UserID:   userID,
		TenantID: tenantID,
		Name:     sessionInfo.DisplayName,
		Email:    req.Email,
	}); err != nil {
		slog.ErrorContext(ctx, "login: failed to encode response", "error", err)
	}
}

// forgotPasswordRequest is the JSON body for POST /api/v1/auth/forgot-password.
type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles POST /api/v1/auth/forgot-password.
// Always returns 200 to prevent email enumeration.
func (h *LoginHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.WarnContext(ctx, "forgot-password: failed to decode request body", "error", err)
		writeAuthError(ctx, w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	if req.Email == "" {
		writeAuthError(ctx, w, http.StatusBadRequest, "Email is required.")
		return
	}

	// Call Zitadel to trigger password reset. Always returns nil even if user doesn't exist.
	if err := h.zitadel.ResetPassword(ctx, req.Email); err != nil {
		slog.ErrorContext(ctx, "forgot-password: failed to trigger password reset",
			"email", req.Email,
			"error", err,
		)
		// Still return 200 to prevent enumeration.
	}

	slog.InfoContext(ctx, "forgot-password: reset requested",
		"email", req.Email,
	)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "If that email exists, we've sent a reset link.",
	}); err != nil {
		slog.ErrorContext(ctx, "forgot-password: failed to encode response", "error", err)
	}
}
