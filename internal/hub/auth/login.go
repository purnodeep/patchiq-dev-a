package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// LoginHandler handles authentication endpoints: Login, Me, Logout.
type LoginHandler struct {
	zitadel  *ZitadelClient
	eventBus domain.EventBus
	cfg      SessionConfig
}

// NewLoginHandler creates a LoginHandler with the given Zitadel client, event bus, and session config.
func NewLoginHandler(zitadel *ZitadelClient, eventBus domain.EventBus, cfg SessionConfig) *LoginHandler {
	return &LoginHandler{
		zitadel:  zitadel,
		eventBus: eventBus,
		cfg:      cfg,
	}
}

// loginRequest is the JSON body expected by POST /api/v1/auth/login.
type loginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// loginResponse is the JSON body returned on a successful login.
type loginResponse struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
}

// Login handles POST /api/v1/auth/login.
// It authenticates the user against Zitadel, mints a JWT, sets an httpOnly cookie,
// emits an auth.login domain event, and returns user info.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(ctx, w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeAuthError(ctx, w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		writeAuthError(ctx, w, http.StatusBadRequest, "password is required")
		return
	}

	result, err := h.zitadel.Authenticate(ctx, req.Email, req.Password)
	if err != nil {
		writeAuthError(ctx, w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	sessionInfo, err := h.zitadel.GetSessionInfo(ctx, result.SessionID)
	if err != nil {
		writeAuthError(ctx, w, http.StatusInternalServerError, "failed to retrieve session info")
		return
	}

	// Determine tenant ID: prefer config default, fall back to org from session.
	tenantID := h.cfg.DefaultTenantID
	if tenantID == "" && sessionInfo.OrgID != "" {
		tenantID = sessionInfo.OrgID
	}

	// Hub uses email as the user identity.
	userID := req.Email

	// Choose TTL based on remember_me.
	ttl := h.cfg.AccessTokenTTL
	if req.RememberMe {
		ttl = h.cfg.RememberMeTTL
	}

	displayName := sessionInfo.DisplayName
	if displayName == "" {
		displayName = req.Email
	}

	token, err := mintJWT(h.cfg.SigningKey, userID, tenantID, req.Email, displayName, ttl)
	if err != nil {
		writeAuthError(ctx, w, http.StatusInternalServerError, "failed to mint session token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    token,
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	if h.eventBus != nil {
		_ = h.eventBus.Emit(ctx, domain.DomainEvent{
			ID:         domain.NewEventID(),
			Type:       events.AuthLogin,
			TenantID:   tenantID,
			ActorID:    userID,
			ActorType:  domain.ActorUser,
			Resource:   "auth",
			ResourceID: userID,
			Action:     "login",
			Payload:    map[string]string{"email": req.Email},
			Timestamp:  time.Now(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResponse{ //nolint:errcheck
		UserID:   userID,
		TenantID: tenantID,
		Name:     displayName,
		Email:    req.Email,
		Role:     h.cfg.defaultRole(),
	})
}

// Me handles GET /api/v1/auth/me.
// It returns the authenticated user's identity extracted from the JWT middleware context.
func (h *LoginHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, _ := user.UserIDFromContext(ctx)
	tenantID, _ := tenant.TenantIDFromContext(ctx)

	email := EmailFromContext(ctx)
	if email == "" {
		email = userID
	}
	name := NameFromContext(ctx)
	if name == "" {
		name = userID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"user_id":   userID,
		"tenant_id": tenantID,
		"email":     email,
		"name":      name,
		"role":      h.cfg.defaultRole(),
	})
}

// Logout handles POST /api/v1/auth/logout.
// It clears the session cookie, emits an auth.logout domain event, and returns a confirmation.
func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	if h.eventBus != nil {
		userID, _ := user.UserIDFromContext(ctx)
		tenantID, _ := tenant.TenantIDFromContext(ctx)
		_ = h.eventBus.Emit(ctx, domain.DomainEvent{
			ID:         domain.NewEventID(),
			Type:       events.AuthLogout,
			TenantID:   tenantID,
			ActorID:    userID,
			ActorType:  domain.ActorUser,
			Resource:   "auth",
			ResourceID: userID,
			Action:     "logout",
			Timestamp:  time.Now(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out"}) //nolint:errcheck
}
