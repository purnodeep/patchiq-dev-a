package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// fakeEventBus captures emitted events for test assertions.
type fakeEventBus struct {
	events []domain.DomainEvent
}

func (f *fakeEventBus) Emit(_ context.Context, event domain.DomainEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (f *fakeEventBus) Close() error                                    { return nil }

func newTestLoginHandler(zitadelHandler http.HandlerFunc) (*LoginHandler, *httptest.Server, *fakeEventBus) {
	srv := httptest.NewServer(zitadelHandler)
	client := NewZitadelClient(srv.URL, "test-pat")
	client.SetOIDCCredentials("test-client", "test-secret")
	bus := &fakeEventBus{}

	cfg := SessionConfig{
		CookieName:     "piq_session",
		CookieDomain:   "localhost",
		CookieSecure:   false,
		AccessTokenTTL: 24 * time.Hour,
		RememberMeTTL:  7 * 24 * time.Hour,
	}

	handler := NewLoginHandler(client, bus, cfg)
	return handler, srv, bus
}

func zitadelSessionHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/sessions" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId":    "sess-123",
				"sessionToken": "tok-abc",
			})
		case r.URL.Path == "/v2/sessions/sess-123" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"factors": map[string]any{
						"user": map[string]any{
							"id":             "user-456",
							"loginName":      "alice@example.com",
							"displayName":    "Alice",
							"organizationId": "org-789",
						},
					},
				},
			})
		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func TestLoginHandler_Success(t *testing.T) {
	handler, srv, bus := newTestLoginHandler(zitadelSessionHandler(t))
	defer srv.Close()

	body := `{"email":"alice@example.com","password":"correct-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "alice@example.com", resp["email"])
	assert.Equal(t, "alice@example.com", resp["user_id"])
	assert.Equal(t, "org-789", resp["tenant_id"])

	// Verify cookie is set.
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "piq_session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "piq_session cookie should be set")
	assert.NotEmpty(t, sessionCookie.Value)
	assert.True(t, sessionCookie.HttpOnly)
	assert.Equal(t, "/", sessionCookie.Path)
	assert.Equal(t, int(24*time.Hour/time.Second), sessionCookie.MaxAge)

	// Verify event emitted.
	require.Len(t, bus.events, 1)
	assert.Equal(t, "auth.login", bus.events[0].Type)
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    7,
			"message": "invalid credentials",
		})
	})
	defer srv.Close()

	body := `{"email":"alice@example.com","password":"wrong-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "That email/password combination didn't work. Try again?", resp["message"])
}

func TestLoginHandler_MissingFields(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach Zitadel")
	})
	defer srv.Close()

	tests := []struct {
		name string
		body string
	}{
		{"empty body", `{}`},
		{"missing password", `{"email":"alice@example.com"}`},
		{"missing email", `{"password":"secret123"}`},
		{"empty email", `{"email":"","password":"secret123"}`},
		{"empty password", `{"email":"alice@example.com","password":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.Login(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func TestLoginHandler_RememberMe(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(zitadelSessionHandler(t))
	defer srv.Close()

	body := `{"email":"alice@example.com","password":"correct-password","remember_me":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "piq_session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "piq_session cookie should be set")
	assert.Equal(t, int(7*24*time.Hour/time.Second), sessionCookie.MaxAge)
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach Zitadel")
	})
	defer srv.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestLoginHandler_TokenExchangeFailure(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/sessions":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId":    "sess-123",
				"sessionToken": "tok-abc",
			})
		case "/oauth/v2/token":
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "invalid_grant",
			})
		}
	})
	defer srv.Close()

	body := `{"email":"alice@example.com","password":"correct-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "Something went wrong. Please try again.", resp["message"])
}

func TestForgotPasswordHandler_Success(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/users":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{
					{"userId": "user-111"},
				},
			})
		case "/v2/users/user-111/password_reset":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	})
	defer srv.Close()

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ForgotPassword(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "If that email exists, we've sent a reset link.", resp["message"])
}

func TestForgotPasswordHandler_NonexistentEmail(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/users":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{},
			})
		default:
			t.Fatalf("should not call password_reset for nonexistent user")
		}
	})
	defer srv.Close()

	body := `{"email":"unknown@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ForgotPassword(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "If that email exists, we've sent a reset link.", resp["message"])
}

func TestForgotPasswordHandler_MissingEmail(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach Zitadel")
	})
	defer srv.Close()

	tests := []struct {
		name string
		body string
	}{
		{"empty body", `{}`},
		{"empty email", `{"email":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.ForgotPassword(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func TestForgotPasswordHandler_InvalidJSON(t *testing.T) {
	handler, srv, _ := newTestLoginHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach Zitadel")
	})
	defer srv.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ForgotPassword(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestMintJWT_NoClaimInjection(t *testing.T) {
	key := []byte("test-key-for-injection-test")
	// Attacker-controlled name with JSON injection payload.
	maliciousName := `","role":"superadmin`
	token, err := mintJWT(key, "user-1", "org-1", "evil@example.com", maliciousName, time.Hour)
	require.NoError(t, err)

	// Decode payload and verify no injected "role" claim exists.
	parts := strings.SplitN(token, ".", 3)
	require.Len(t, parts, 3)

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(payloadBytes, &claims))

	// The name should be the literal malicious string, not parsed as separate JSON keys.
	assert.Equal(t, maliciousName, claims["name"])
	// There must NOT be an injected "role" key.
	_, hasRole := claims["role"]
	assert.False(t, hasRole, "JWT claim injection: attacker injected a 'role' claim")
}
