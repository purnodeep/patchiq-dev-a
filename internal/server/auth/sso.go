package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"

	"github.com/skenzeriq/patchiq/internal/shared/organization"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// SSOConfig holds the OIDC/Zitadel SSO configuration.
type SSOConfig struct {
	ZitadelDomain string
	ZitadelSecure bool
	ClientID      string
	ClientSecret  string
	RedirectURI   string
	CookieName    string
	CookieDomain  string
	CookieSecure  bool
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	PostLoginURL  string
}

// SessionStore persists refresh tokens for SSO sessions.
type SessionStore interface {
	StoreRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error
	DeleteRefreshToken(ctx context.Context, userID string) error
}

// RoleStore loads role names for the current user.
type RoleStore interface {
	GetUserRoles(ctx context.Context, tenantID, userID string) ([]string, error)
}

// OrgScopeLookup is the optional dependency SSOHandler uses to populate the
// /auth/me response with organization and accessible-tenant information.
// Implementations wrap *store.Store.
type OrgScopeLookup interface {
	// GetOrganizationByID returns the name, slug, and type of an organization,
	// or an error if it does not exist.
	GetOrganizationByIDForMe(ctx context.Context, orgID string) (name, slug, orgType string, err error)
	// UserAccessibleTenantsForMe returns the (id, name, slug) triples for every
	// tenant the user can access in the given organization.
	UserAccessibleTenantsForMe(ctx context.Context, orgID, userID string) ([]AccessibleTenantInfo, error)
}

// AccessibleTenantInfo is a minimal projection of sqlcgen.Tenant suitable
// for returning in /auth/me.
type AccessibleTenantInfo struct {
	ID   string
	Name string
	Slug string
}

// SSOHandler implements the Login, Callback, Logout, and Me HTTP handlers
// for OIDC Authorization Code + PKCE flow with Zitadel.
type SSOHandler struct {
	cfg       SSOConfig
	sessions  SessionStore
	PermStore PermissionStore
	OrgPerms  OrgPermissionStore // optional; set by router to enable org-scoped /auth/me
	RoleStore RoleStore
	OrgScope  OrgScopeLookup // optional; enables organization + accessible_tenants in /auth/me
}

// NewSSOHandler creates an SSOHandler with the given config and session store.
func NewSSOHandler(cfg SSOConfig, sessions SessionStore) *SSOHandler {
	return &SSOHandler{cfg: cfg, sessions: sessions}
}

// scheme returns "https" if ZitadelSecure is true, otherwise "http".
func (h *SSOHandler) scheme() string {
	if h.cfg.ZitadelSecure {
		return "https"
	}
	return "http"
}

// Login initiates the OIDC Authorization Code + PKCE flow by redirecting
// the user to Zitadel's authorize endpoint.
func (h *SSOHandler) Login(w http.ResponseWriter, r *http.Request) {
	verifier, err := generateRandomBase64URL(32)
	if err != nil {
		slog.ErrorContext(r.Context(), "sso login: failed to generate PKCE verifier", "error", err)
		writeAuthError(r.Context(), w, http.StatusInternalServerError, "failed to initiate login")
		return
	}

	challenge := s256Challenge(verifier)

	state, err := generateRandomBase64URL(16)
	if err != nil {
		slog.ErrorContext(r.Context(), "sso login: failed to generate state", "error", err)
		writeAuthError(r.Context(), w, http.StatusInternalServerError, "failed to initiate login")
		return
	}

	// Store verifier|state in a short-lived httpOnly cookie for the callback.
	http.SetCookie(w, &http.Cookie{
		Name:     "piq_pkce",
		Value:    verifier + "|" + state,
		Path:     "/api/v1/auth",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	params := url.Values{
		"client_id":             {h.cfg.ClientID},
		"redirect_uri":          {h.cfg.RedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile email urn:zitadel:iam:org:id urn:zitadel:iam:org:project:roles"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}

	authorizeURL := fmt.Sprintf("%s://%s/oauth/v2/authorize?%s",
		h.scheme(), h.cfg.ZitadelDomain, params.Encode())

	slog.InfoContext(r.Context(), "sso login: redirecting to Zitadel authorize endpoint",
		"client_id", h.cfg.ClientID,
	)

	http.Redirect(w, r, authorizeURL, http.StatusFound)
}

// tokenResponse represents the JSON response from Zitadel's token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Callback handles the OIDC callback, exchanging the authorization code for tokens.
func (h *SSOHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		slog.WarnContext(r.Context(), "sso callback: missing authorization code")
		writeAuthError(r.Context(), w, http.StatusBadRequest, "missing authorization code")
		return
	}

	// Retrieve and validate PKCE cookie.
	pkceCookie, err := r.Cookie("piq_pkce")
	if err != nil {
		slog.WarnContext(r.Context(), "sso callback: missing PKCE cookie", "error", err)
		writeAuthError(r.Context(), w, http.StatusBadRequest, "missing PKCE state: restart login")
		return
	}

	parts := strings.SplitN(pkceCookie.Value, "|", 2)
	if len(parts) != 2 {
		slog.WarnContext(r.Context(), "sso callback: malformed PKCE cookie")
		writeAuthError(r.Context(), w, http.StatusBadRequest, "invalid PKCE state: restart login")
		return
	}

	verifier, savedState := parts[0], parts[1]
	if state != savedState {
		slog.WarnContext(r.Context(), "sso callback: state mismatch",
			"expected", savedState, "got", state)
		writeAuthError(r.Context(), w, http.StatusBadRequest, "state mismatch: possible CSRF")
		return
	}

	// Clear PKCE cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "piq_pkce",
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Exchange code for tokens.
	tokenURL := fmt.Sprintf("%s://%s/oauth/v2/token", h.scheme(), h.cfg.ZitadelDomain)
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {h.cfg.RedirectURI},
		"client_id":     {h.cfg.ClientID},
		"code_verifier": {verifier},
	}

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		slog.ErrorContext(r.Context(), "sso callback: token exchange request failed",
			"error", err, "token_url", tokenURL)
		writeAuthError(r.Context(), w, http.StatusBadGateway, "failed to exchange authorization code")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.ErrorContext(r.Context(), "sso callback: token endpoint returned error",
			"status", resp.StatusCode, "body", string(body))
		writeAuthError(r.Context(), w, http.StatusBadGateway, "token exchange failed")
		return
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		slog.ErrorContext(r.Context(), "sso callback: failed to decode token response", "error", err)
		writeAuthError(r.Context(), w, http.StatusInternalServerError, "failed to parse token response")
		return
	}

	// Set access token as httpOnly cookie.
	maxAge := tok.ExpiresIn
	if h.cfg.AccessTTL > 0 {
		maxAge = int(h.cfg.AccessTTL.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    tok.AccessToken,
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	slog.InfoContext(r.Context(), "sso callback: token exchange successful")

	redirectURL := h.cfg.PostLoginURL
	if redirectURL == "" {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// Logout clears the session cookie and redirects to Zitadel's end-session endpoint.
func (h *SSOHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear session cookie.
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

	slog.InfoContext(r.Context(), "sso logout: session cookie cleared")

	endSessionURL := fmt.Sprintf("%s://%s/oidc/v1/end_session",
		h.scheme(), h.cfg.ZitadelDomain)

	http.Redirect(w, r, endSessionURL, http.StatusFound)
}

// meResponse is the JSON response body for the /auth/me endpoint.
type meResponse struct {
	UserID            string            `json:"user_id"`
	TenantID          string            `json:"tenant_id,omitempty"`
	Name              string            `json:"name,omitempty"`
	Email             string            `json:"email,omitempty"`
	PreferredUsername string            `json:"preferred_username,omitempty"`
	Roles             []string          `json:"roles,omitempty"`
	Permissions       []permissionEntry `json:"permissions,omitempty"`
	// Organization / MSP fields, populated when OrgScope is configured.
	Organization      *organizationRef   `json:"organization,omitempty"`
	ActiveTenantID    string             `json:"active_tenant_id,omitempty"`
	AccessibleTenants []accessibleTenant `json:"accessible_tenants,omitempty"`
	OrgPermissions    []permissionEntry  `json:"org_permissions,omitempty"`
}

// organizationRef is the minimal organization projection in /auth/me.
type organizationRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"`
}

// accessibleTenant is a minimal tenant projection in /auth/me.
type accessibleTenant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type permissionEntry struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Scope    string `json:"scope"`
}

// oidcProfileClaims holds the OIDC profile/email claims extracted from the JWT.
type oidcProfileClaims struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
}

// Me returns the current user's identity from request context as JSON.
// It also decodes the session JWT to extract OIDC profile claims (name, email,
// preferred_username). Returns 401 if no user identity is present in context.
func (h *SSOHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := user.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeAuthError(r.Context(), w, http.StatusUnauthorized, "not authenticated")
		return
	}

	resp := meResponse{UserID: userID}

	tenantID, ok := tenant.TenantIDFromContext(r.Context())
	if ok && tenantID != "" {
		resp.TenantID = tenantID
	}

	// Decode the session cookie JWT to extract OIDC profile claims.
	// The JWT signature has already been validated by the JWT middleware,
	// so we only need to parse the payload here — no key material required.
	cookieName := h.cfg.CookieName
	if cookieName == "" {
		cookieName = "piq_session"
	}
	if cookie, err := r.Cookie(cookieName); err == nil {
		tok, err := jwtParseInsecure(cookie.Value)
		if err != nil {
			slog.WarnContext(r.Context(), "sso me: failed to parse session JWT for profile claims",
				"error", err,
			)
		} else {
			var profile oidcProfileClaims
			if err := tok.UnsafeClaimsWithoutVerification(&profile); err != nil {
				slog.WarnContext(r.Context(), "sso me: failed to extract profile claims from JWT",
					"error", err,
				)
			} else {
				resp.Name = profile.Name
				resp.Email = profile.Email
				resp.PreferredUsername = profile.PreferredUsername
			}
		}
	}

	// Fallback: use user_id as display name when OIDC claims are unavailable
	if resp.Name == "" {
		resp.Name = userID
	}

	if h.PermStore != nil && resp.TenantID != "" {
		perms, err := h.PermStore.GetUserPermissions(r.Context(), resp.TenantID, userID)
		if err != nil {
			slog.WarnContext(r.Context(), "sso me: failed to load user permissions", "error", err)
		} else {
			entries := make([]permissionEntry, len(perms))
			for i, p := range perms {
				entries[i] = permissionEntry(p)
			}
			resp.Permissions = entries
		}
	}

	if h.RoleStore != nil && resp.TenantID != "" {
		roles, err := h.RoleStore.GetUserRoles(r.Context(), resp.TenantID, userID)
		if err != nil {
			slog.WarnContext(r.Context(), "sso me: failed to load user roles", "error", err)
		} else {
			resp.Roles = roles
		}
	}

	// Populate organization + accessible tenants if the org scope resolver
	// is wired up. This enables the MSP operator tenant switcher and the
	// cross-tenant dashboard in the frontend. Missing org info is NOT an
	// error — single-tenant deployments simply leave these fields empty.
	if h.OrgScope != nil && resp.TenantID != "" {
		orgID, orgOK := organization.OrgIDFromContext(r.Context())
		if orgOK && orgID != "" {
			name, slug, orgType, err := h.OrgScope.GetOrganizationByIDForMe(r.Context(), orgID)
			if err != nil {
				slog.WarnContext(r.Context(), "sso me: failed to load organization", "org_id", orgID, "error", err)
			} else {
				resp.Organization = &organizationRef{ID: orgID, Name: name, Slug: slug, Type: orgType}
				resp.ActiveTenantID = resp.TenantID
				tenants, err := h.OrgScope.UserAccessibleTenantsForMe(r.Context(), orgID, userID)
				if err != nil {
					slog.WarnContext(r.Context(), "sso me: failed to list accessible tenants",
						"org_id", orgID, "user_id", userID, "error", err)
				} else {
					resp.AccessibleTenants = make([]accessibleTenant, len(tenants))
					for i, t := range tenants {
						resp.AccessibleTenants[i] = accessibleTenant(t)
					}
				}
			}
		}

		if h.OrgPerms != nil && resp.Organization != nil {
			orgPerms, err := h.OrgPerms.GetUserOrgPermissions(r.Context(), resp.Organization.ID, userID)
			if err != nil {
				slog.WarnContext(r.Context(), "sso me: failed to load org permissions", "error", err)
			} else {
				entries := make([]permissionEntry, len(orgPerms))
				for i, p := range orgPerms {
					entries[i] = permissionEntry(p)
				}
				resp.OrgPermissions = entries
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "sso me: failed to encode response", "error", err)
	}
}

// generateRandomBase64URL generates n random bytes and returns them as a
// base64url-encoded string (no padding).
func generateRandomBase64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// s256Challenge computes the S256 PKCE challenge from a code verifier.
func s256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// jwtParseInsecure parses a signed JWT token string without verifying its
// signature. This is safe to call only after the JWT middleware has already
// validated the token's signature, issuer and expiry on the same request.
func jwtParseInsecure(raw string) (*josejwt.JSONWebToken, error) {
	return josejwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
}
