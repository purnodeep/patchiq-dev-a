package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
)

func TestSSOHandler_Login(t *testing.T) {
	cfg := auth.SSOConfig{
		ZitadelDomain: "localhost:8085",
		ZitadelSecure: false,
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:8080/api/v1/auth/callback",
		CookieName:    "piq_session",
		CookieDomain:  "localhost",
		CookieSecure:  false,
	}
	h := auth.NewSSOHandler(cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "authorize") {
		t.Errorf("redirect location should contain authorize endpoint, got %s", loc)
	}
	if !strings.Contains(loc, "client_id=test-client") {
		t.Errorf("redirect should contain client_id, got %s", loc)
	}
	if !strings.Contains(loc, "code_challenge") {
		t.Errorf("redirect should contain PKCE code_challenge, got %s", loc)
	}
}

func TestSSOHandler_Logout(t *testing.T) {
	cfg := auth.SSOConfig{
		ZitadelDomain: "localhost:8085",
		ZitadelSecure: false,
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:8080/api/v1/auth/callback",
		CookieName:    "piq_session",
		CookieDomain:  "localhost",
		CookieSecure:  false,
	}
	h := auth.NewSSOHandler(cfg, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.Logout(rec, req)

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "piq_session" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("logout should clear piq_session cookie")
	}
}

func TestSSOHandler_Me_Unauthenticated(t *testing.T) {
	cfg := auth.SSOConfig{CookieName: "piq_session"}
	h := auth.NewSSOHandler(cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
