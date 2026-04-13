package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// testJWKS sets up an in-memory RSA key pair and JWKS HTTP server for testing.
func testJWKS(t *testing.T) (*rsa.PrivateKey, *httptest.Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{{
			Key:       &key.PublicKey,
			KeyID:     "test-key-1",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	t.Cleanup(srv.Close)
	return key, srv
}

// signToken creates a signed JWT with the given claims.
func signToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key-1"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestJWTMiddleware_DevBypass_RejectedWithoutDevMode(t *testing.T) {
	_, jwksSrv := testJWKS(t)

	cfg := auth.JWTConfig{
		Issuer:     jwksSrv.URL,
		JWKSURL:    jwksSrv.URL,
		CookieName: "piq_session",
		DevMode:    false,
	}
	mw := auth.NewJWTMiddleware(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-dev")
	req.Header.Set("X-User-ID", "user-dev")

	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d — dev bypass should be disabled without DevMode", rec.Code)
	}
}

func TestJWTMiddleware_DevBypass_AllowedWithDevMode(t *testing.T) {
	_, jwksSrv := testJWKS(t)

	cfg := auth.JWTConfig{
		Issuer:     jwksSrv.URL,
		JWKSURL:    jwksSrv.URL,
		CookieName: "piq_session",
		DevMode:    true,
	}
	mw := auth.NewJWTMiddleware(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-dev")
	req.Header.Set("X-User-ID", "user-dev")

	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := user.UserIDFromContext(r.Context())
		tid, _ := tenant.TenantIDFromContext(r.Context())
		fmt.Fprintf(w, "user=%s tenant=%s", uid, tid)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	if rec.Body.String() != "user=user-dev tenant=tenant-dev" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "user=user-dev tenant=tenant-dev")
	}
}

func TestJWTMiddleware(t *testing.T) {
	key, jwksSrv := testJWKS(t)
	issuer := jwksSrv.URL

	cfg := auth.JWTConfig{
		Issuer:     issuer,
		JWKSURL:    jwksSrv.URL,
		CookieName: "piq_session",
	}
	mw := auth.NewJWTMiddleware(cfg)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := user.UserIDFromContext(r.Context())
		tid, _ := tenant.TenantIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "user=%s tenant=%s", uid, tid)
	})

	handler := mw(nextHandler)

	tests := []struct {
		name       string
		cookie     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "missing cookie returns 401",
			cookie:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed token returns 401",
			cookie:     "not-a-jwt",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "expired token returns 401",
			cookie: signToken(t, key, map[string]any{
				"iss":                    issuer,
				"sub":                    "user-123",
				"exp":                    time.Now().Add(-1 * time.Hour).Unix(),
				"iat":                    time.Now().Add(-2 * time.Hour).Unix(),
				"urn:zitadel:iam:org:id": "org-456",
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong issuer returns 401",
			cookie: signToken(t, key, map[string]any{
				"iss":                    "https://evil.example.com",
				"sub":                    "user-123",
				"exp":                    time.Now().Add(1 * time.Hour).Unix(),
				"iat":                    time.Now().Unix(),
				"urn:zitadel:iam:org:id": "org-456",
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "missing sub claim returns 401",
			cookie: signToken(t, key, map[string]any{
				"iss":                    issuer,
				"exp":                    time.Now().Add(1 * time.Hour).Unix(),
				"iat":                    time.Now().Unix(),
				"urn:zitadel:iam:org:id": "org-456",
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "missing org_id claim returns 401",
			cookie: signToken(t, key, map[string]any{
				"iss": issuer,
				"sub": "user-123",
				"exp": time.Now().Add(1 * time.Hour).Unix(),
				"iat": time.Now().Unix(),
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "valid token sets user and tenant context",
			cookie: signToken(t, key, map[string]any{
				"iss":                    issuer,
				"sub":                    "user-123",
				"exp":                    time.Now().Add(1 * time.Hour).Unix(),
				"iat":                    time.Now().Unix(),
				"urn:zitadel:iam:org:id": "org-456",
			}),
			wantStatus: http.StatusOK,
			wantBody:   "user=user-123 tenant=org-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "piq_session", Value: tt.cookie})
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
