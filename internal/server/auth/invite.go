package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// InviteQuerier defines the sqlc queries needed by InviteHandler.
type InviteQuerier interface {
	CreateInvitation(ctx context.Context, arg sqlcgen.CreateInvitationParams) (sqlcgen.Invitation, error)
	GetInvitationByCode(ctx context.Context, code pgtype.UUID) (sqlcgen.Invitation, error)
	ClaimInvitation(ctx context.Context, code pgtype.UUID) (sqlcgen.Invitation, error)
	ListInvitations(ctx context.Context, arg sqlcgen.ListInvitationsParams) ([]sqlcgen.Invitation, error)
	AssignUserRole(ctx context.Context, arg sqlcgen.AssignUserRoleParams) error
	GetRoleByID(ctx context.Context, arg sqlcgen.GetRoleByIDParams) (sqlcgen.Role, error)
	GetTenantByID(ctx context.Context, id pgtype.UUID) (sqlcgen.Tenant, error)
}

// ZitadelUserCreator is the subset of ZitadelClient used by InviteHandler.
type ZitadelUserCreator interface {
	CreateUser(ctx context.Context, email, name, password string) (string, error)
	Authenticate(ctx context.Context, email, password string) (*AuthResult, error)
	ExchangeToken(ctx context.Context, sessionToken string) (string, error)
}

// InviteHandler handles invite creation, validation, and registration.
type InviteHandler struct {
	q        InviteQuerier
	zitadel  ZitadelUserCreator
	eventBus domain.EventBus
	cfg      SessionConfig
	baseURL  string // frontend base URL for generating invite links
}

// NewInviteHandler creates an InviteHandler with the given dependencies.
func NewInviteHandler(q InviteQuerier, zitadel ZitadelUserCreator, eventBus domain.EventBus, cfg SessionConfig, baseURL string) *InviteHandler {
	return &InviteHandler{
		q:        q,
		zitadel:  zitadel,
		eventBus: eventBus,
		cfg:      cfg,
		baseURL:  baseURL,
	}
}

// createInviteRequest is the JSON body for POST /api/v1/auth/invite.
type createInviteRequest struct {
	Email  string `json:"email"`
	RoleID string `json:"role_id"`
}

// CreateInvite handles POST /api/v1/auth/invite (protected).
// Creates an invitation for the given email and role, scoped to the caller's tenant.
func (h *InviteHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	userID := user.MustUserID(ctx)

	var req createInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.WarnContext(ctx, "create invite: failed to decode request body", "error", err)
		writeAuthError(ctx, w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	if req.Email == "" || req.RoleID == "" {
		writeAuthError(ctx, w, http.StatusBadRequest, "Email and role_id are required.")
		return
	}

	tid, err := parsePgUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "create invite: invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		writeAuthError(ctx, w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}

	roleID, err := parsePgUUID(req.RoleID)
	if err != nil {
		writeAuthError(ctx, w, http.StatusBadRequest, "role_id is not a valid UUID.")
		return
	}

	inv, err := h.q.CreateInvitation(ctx, sqlcgen.CreateInvitationParams{
		TenantID:  tid,
		Email:     req.Email,
		RoleID:    roleID,
		InvitedBy: userID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create invite: failed to create invitation",
			"email", req.Email, "tenant_id", tenantID, "error", err)
		writeAuthError(ctx, w, http.StatusInternalServerError, "Failed to create invitation.")
		return
	}

	codeStr := pgUUIDToString(inv.Code)
	inviteURL := fmt.Sprintf("%s/register?code=%s", h.baseURL, codeStr)

	// Emit invitation.created event.
	h.emitEvent(ctx, events.InvitationCreated, "invitation", pgUUIDToString(inv.ID), tenantID,
		map[string]string{"email": req.Email, "role_id": req.RoleID, "invited_by": userID})

	slog.InfoContext(ctx, "create invite: invitation created",
		"email", req.Email, "code", codeStr, "tenant_id", tenantID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"code":       codeStr,
		"invite_url": inviteURL,
		"expires_at": inv.ExpiresAt.Time.Format(time.RFC3339),
	}); err != nil {
		slog.ErrorContext(ctx, "create invite: failed to encode response", "error", err)
	}
}

// ValidateInvite handles GET /api/v1/auth/invite/{code} (public).
// Looks up the invitation by code using the bypass pool (no tenant context required).
func (h *InviteHandler) ValidateInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	codeStr := chi.URLParam(r, "code")

	codeUUID, err := parsePgUUID(codeStr)
	if err != nil {
		writeAuthError(ctx, w, http.StatusNotFound, "This invite link is invalid or has expired.")
		return
	}

	inv, err := h.q.GetInvitationByCode(ctx, codeUUID)
	if err != nil {
		slog.InfoContext(ctx, "validate invite: invitation not found or expired",
			"code", codeStr, "error", err)
		writeAuthError(ctx, w, http.StatusNotFound, "This invite link is invalid or has expired.")
		return
	}

	// Look up role name and tenant name using the invitation's tenant context.
	roleName, tenantName, err := h.lookupInviteContext(ctx, inv)
	if err != nil {
		slog.ErrorContext(ctx, "validate invite: failed to look up context",
			"code", codeStr, "error", err)
		// Still return basic info even if lookup fails.
		roleName = ""
		tenantName = ""
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"email":       inv.Email,
		"tenant_name": tenantName,
		"role_name":   roleName,
		"expires_at":  inv.ExpiresAt.Time.Format(time.RFC3339),
	}); err != nil {
		slog.ErrorContext(ctx, "validate invite: failed to encode response", "error", err)
	}
}

// registerRequest is the JSON body for POST /api/v1/auth/register.
type registerRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Register handles POST /api/v1/auth/register (public).
// Flow: validate body -> re-validate invite (TOCTOU) -> create user in Zitadel ->
// claim invite -> assign role -> authenticate -> set cookie -> emit events -> respond.
func (h *InviteHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.WarnContext(ctx, "register: failed to decode request body", "error", err)
		writeAuthError(ctx, w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	// Validate required fields.
	if req.Code == "" || req.Name == "" || req.Password == "" {
		writeAuthError(ctx, w, http.StatusBadRequest, "Code, name, and password are required.")
		return
	}

	// Validate password strength.
	if len(req.Password) < 8 {
		writeAuthError(ctx, w, http.StatusBadRequest, "Password must be at least 8 characters.")
		return
	}

	// Re-validate invite code (TOCTOU protection).
	codeUUID, err := parsePgUUID(req.Code)
	if err != nil {
		writeAuthError(ctx, w, http.StatusNotFound, "This invite link is invalid or has expired.")
		return
	}

	inv, err := h.q.GetInvitationByCode(ctx, codeUUID)
	if err != nil {
		slog.InfoContext(ctx, "register: invitation not found or expired",
			"code", req.Code, "error", err)
		writeAuthError(ctx, w, http.StatusNotFound, "This invite link is invalid or has expired.")
		return
	}

	tenantID := pgUUIDToString(inv.TenantID)

	// Create user in Zitadel.
	zitadelUserID, err := h.zitadel.CreateUser(ctx, inv.Email, req.Name, req.Password)
	if err != nil {
		slog.ErrorContext(ctx, "register: failed to create user in Zitadel",
			"email", inv.Email, "error", err)
		writeAuthError(ctx, w, http.StatusInternalServerError, "Failed to create account. Please try again.")
		return
	}

	// Claim the invitation atomically.
	_, err = h.q.ClaimInvitation(ctx, codeUUID)
	if err != nil {
		slog.ErrorContext(ctx, "register: failed to claim invitation",
			"code", req.Code, "error", err)
		writeAuthError(ctx, w, http.StatusNotFound, "This invite link is invalid or has expired.")
		return
	}

	// Assign role via user_roles table.
	if err := h.q.AssignUserRole(ctx, sqlcgen.AssignUserRoleParams{
		TenantID: inv.TenantID,
		UserID:   zitadelUserID,
		RoleID:   inv.RoleID,
	}); err != nil {
		slog.ErrorContext(ctx, "register: failed to assign role",
			"user_id", zitadelUserID, "role_id", pgUUIDToString(inv.RoleID),
			"tenant_id", tenantID, "error", err)
		// Non-fatal for the user: account is created, role can be fixed by admin.
	}

	// Authenticate + exchange token to get a JWT for the new user.
	var jwt string
	authResult, err := h.zitadel.Authenticate(ctx, inv.Email, req.Password)
	if err != nil {
		slog.ErrorContext(ctx, "register: failed to authenticate new user",
			"email", inv.Email, "error", err)
	} else {
		jwt, err = h.zitadel.ExchangeToken(ctx, authResult.SessionToken)
		if err != nil {
			slog.ErrorContext(ctx, "register: failed to exchange token",
				"email", inv.Email, "error", err)
		}
	}

	// Set session cookie if we got a JWT.
	if jwt != "" {
		cookieName := h.cfg.CookieName
		if cookieName == "" {
			cookieName = "piq_session"
		}
		maxAge := int(h.cfg.AccessTokenTTL.Seconds())
		if maxAge == 0 {
			maxAge = int((24 * time.Hour).Seconds())
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    jwt,
			Path:     "/",
			Domain:   h.cfg.CookieDomain,
			MaxAge:   maxAge,
			HttpOnly: true,
			Secure:   h.cfg.CookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
	}

	// Emit events: invitation.claimed, user.registered, auth.login.
	invID := pgUUIDToString(inv.ID)
	h.emitEvent(ctx, events.InvitationClaimed, "invitation", invID, tenantID,
		map[string]string{"email": inv.Email, "user_id": zitadelUserID})
	h.emitEvent(ctx, events.UserRegistered, "user", zitadelUserID, tenantID,
		map[string]string{"email": inv.Email, "name": req.Name})
	h.emitEvent(ctx, events.AuthLogin, "auth", zitadelUserID, tenantID,
		map[string]string{"email": inv.Email})

	slog.InfoContext(ctx, "register: user registered successfully",
		"email", inv.Email, "user_id", zitadelUserID, "tenant_id", tenantID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"user_id": zitadelUserID,
		"email":   inv.Email,
		"name":    req.Name,
	}); err != nil {
		slog.ErrorContext(ctx, "register: failed to encode response", "error", err)
	}
}

// lookupInviteContext resolves role name and tenant name from the invitation.
func (h *InviteHandler) lookupInviteContext(ctx context.Context, inv sqlcgen.Invitation) (roleName, tenantName string, err error) {
	t, err := h.q.GetTenantByID(ctx, inv.TenantID)
	if err != nil {
		return "", "", fmt.Errorf("look up tenant: %w", err)
	}
	tenantName = t.Name

	role, err := h.q.GetRoleByID(ctx, sqlcgen.GetRoleByIDParams{
		ID:       inv.RoleID,
		TenantID: inv.TenantID,
	})
	if err != nil {
		return "", tenantName, fmt.Errorf("look up role: %w", err)
	}
	roleName = role.Name

	return roleName, tenantName, nil
}

// emitEvent publishes a domain event. Errors are logged but not propagated.
func (h *InviteHandler) emitEvent(ctx context.Context, eventType, resource, resourceID, tenantID string, payload any) {
	if h.eventBus == nil {
		slog.ErrorContext(ctx, "invite: event bus is nil — domain event not emitted",
			"event_type", eventType, "resource", resource, "resource_id", resourceID)
		return
	}
	event := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       eventType,
		TenantID:   tenantID,
		ActorID:    "system",
		ActorType:  domain.ActorSystem,
		Resource:   resource,
		ResourceID: resourceID,
		Action:     eventType,
		Payload:    payload,
		Timestamp:  time.Now(),
	}
	if err := h.eventBus.Emit(ctx, event); err != nil {
		slog.ErrorContext(ctx, "invite: failed to emit domain event",
			"event_type", eventType, "resource", resource,
			"resource_id", resourceID, "tenant_id", tenantID, "error", err)
	}
}

// parsePgUUID parses a string UUID into the pgtype representation used by sqlc.
func parsePgUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

// pgUUIDToString converts a pgtype.UUID to its canonical string form.
func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
