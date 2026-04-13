package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skenzeriq/patchiq/internal/server/auth"
)

// RegisterAuthRoutes mounts SSO auth routes (public, no JWT required).
// Uses explicit paths instead of r.Route() to avoid creating a catch-all
// subrouter that would shadow /api/v1/auth/me in the JWT-protected group.
func RegisterAuthRoutes(r chi.Router, ssoHandler *auth.SSOHandler) {
	r.Get("/api/v1/auth/sso", ssoHandler.Login)
	r.Get("/api/v1/auth/callback", ssoHandler.Callback)
	r.Post("/api/v1/auth/logout", ssoHandler.Logout)
}

// RegisterLoginRoutes mounts public login, forgot-password, register, and
// invite validation routes. These are rate-limited but do not require JWT.
// Uses explicit paths (like RegisterAuthRoutes) to avoid subrouter shadowing.
func RegisterLoginRoutes(r chi.Router, loginHandler *auth.LoginHandler, inviteHandler *auth.InviteHandler, rateLimiter func(http.Handler) http.Handler) {
	if loginHandler != nil {
		r.With(rateLimiter).Post("/api/v1/auth/login", loginHandler.Login)
		r.With(rateLimiter).Post("/api/v1/auth/forgot-password", loginHandler.ForgotPassword)
	}
	if inviteHandler != nil {
		r.With(rateLimiter).Post("/api/v1/auth/register", inviteHandler.Register)
		r.Get("/api/v1/auth/invite/{code}", inviteHandler.ValidateInvite)
	}
}

// RegisterAuthenticatedAuthRoutes mounts auth routes that require JWT (inside /api/v1 group).
func RegisterAuthenticatedAuthRoutes(r chi.Router, ssoHandler *auth.SSOHandler) {
	r.Get("/auth/me", ssoHandler.Me)
}
