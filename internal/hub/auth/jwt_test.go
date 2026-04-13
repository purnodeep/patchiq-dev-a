package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

var testSigningKey = []byte("test-signing-key-for-jwt-tests")

func testMiddleware() func(http.Handler) http.Handler {
	return NewJWTMiddleware(JWTMiddlewareConfig{
		CookieName: "hub_session",
		SigningKey: testSigningKey,
	})
}

func newHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestJWT_ValidTokenInCookieSetsContext(t *testing.T) {
	token, err := mintJWT(testSigningKey, "user-123", "tenant-abc", "test@example.com", "Test User", time.Hour)
	if err != nil {
		t.Fatalf("mintJWT: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "hub_session", Value: token})

	var gotUserID, gotTenantID string
	handler := testMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = user.UserIDFromContext(r.Context())
		gotTenantID, _ = tenant.TenantIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	if gotUserID != "user-123" {
		t.Errorf("user ID: want user-123, got %q", gotUserID)
	}
	if gotTenantID != "tenant-abc" {
		t.Errorf("tenant ID: want tenant-abc, got %q", gotTenantID)
	}
}

func TestJWT_ExpiredTokenReturns401(t *testing.T) {
	token, err := mintJWT(testSigningKey, "user-123", "tenant-abc", "test@example.com", "Test User", -time.Minute)
	if err != nil {
		t.Fatalf("mintJWT: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "hub_session", Value: token})

	rec := httptest.NewRecorder()
	testMiddleware()(newHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rec.Code)
	}
}

func TestJWT_MissingCookieReturns401(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	testMiddleware()(newHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rec.Code)
	}
}

func TestJWT_MalformedTokenReturns401(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "hub_session", Value: "not.a.valid.jwt.token"})

	rec := httptest.NewRecorder()
	testMiddleware()(newHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rec.Code)
	}
}

func TestJWT_WrongSigningKeyReturns401(t *testing.T) {
	wrongKey := []byte("wrong-signing-key-entirely-different")
	token, err := mintJWT(wrongKey, "user-123", "tenant-abc", "test@example.com", "Test User", time.Hour)
	if err != nil {
		t.Fatalf("mintJWT: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "hub_session", Value: token})

	rec := httptest.NewRecorder()
	testMiddleware()(newHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rec.Code)
	}
}

func TestJWT_ValidTokenInAuthorizationHeader(t *testing.T) {
	token, err := mintJWT(testSigningKey, "user-456", "tenant-xyz", "other@example.com", "Other User", time.Hour)
	if err != nil {
		t.Fatalf("mintJWT: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	var gotUserID, gotTenantID string
	handler := testMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = user.UserIDFromContext(r.Context())
		gotTenantID, _ = tenant.TenantIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	if gotUserID != "user-456" {
		t.Errorf("user ID: want user-456, got %q", gotUserID)
	}
	if gotTenantID != "tenant-xyz" {
		t.Errorf("tenant ID: want tenant-xyz, got %q", gotTenantID)
	}
}

func TestJWT_DevFallbackHeadersBypassJWT_DevModeEnabled(t *testing.T) {
	mw := NewJWTMiddleware(JWTMiddlewareConfig{
		CookieName: "hub_session",
		SigningKey: testSigningKey,
		DevMode:    true,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-dev")
	req.Header.Set("X-User-ID", "user-dev")

	var gotUserID, gotTenantID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = user.UserIDFromContext(r.Context())
		gotTenantID, _ = tenant.TenantIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	if gotUserID != "user-dev" {
		t.Errorf("user ID: want user-dev, got %q", gotUserID)
	}
	if gotTenantID != "tenant-dev" {
		t.Errorf("tenant ID: want tenant-dev, got %q", gotTenantID)
	}
}

func TestJWT_DevFallbackHeaders_RejectedWithoutDevMode(t *testing.T) {
	mw := NewJWTMiddleware(JWTMiddlewareConfig{
		CookieName: "hub_session",
		SigningKey: testSigningKey,
		DevMode:    false,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-dev")
	req.Header.Set("X-User-ID", "user-dev")

	rec := httptest.NewRecorder()
	mw(newHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d — dev bypass should be disabled without DevMode", rec.Code)
	}
}
