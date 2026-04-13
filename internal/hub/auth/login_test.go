package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// mockEventBus records emitted event types for test assertions.
type mockEventBus struct {
	emitted []string
}

func (m *mockEventBus) Emit(_ context.Context, e domain.DomainEvent) error {
	m.emitted = append(m.emitted, e.Type)
	return nil
}

func (m *mockEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (m *mockEventBus) Close() error                                    { return nil }

// zitadelMockServer returns an httptest.Server that mimics the Zitadel v2 API
// for session creation and retrieval. Only "correct@example.com" / "secret" succeeds.
func zitadelMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// POST /v2/sessions — authenticate
	mux.HandleFunc("/v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Checks struct {
				User struct {
					LoginName string `json:"loginName"`
				} `json:"user"`
				Password struct {
					Password string `json:"password"`
				} `json:"password"`
			} `json:"checks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.Checks.User.LoginName != "correct@example.com" || body.Checks.Password.Password != "secret" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "invalid credentials"}) //nolint:errcheck
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"sessionId":    "sess-123",
			"sessionToken": "tok-abc",
		})
	})

	// GET /v2/sessions/sess-123 — get session info
	mux.HandleFunc("/v2/sessions/sess-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"session": map[string]any{
				"factors": map[string]any{
					"user": map[string]any{
						"id":             "zitadel-user-id-001",
						"loginName":      "correct@example.com",
						"displayName":    "Correct User",
						"organizationId": "org-tenant-001",
					},
				},
			},
		})
	})

	return httptest.NewServer(mux)
}

// newTestHandler creates a LoginHandler backed by a mock Zitadel server.
func newTestHandler(t *testing.T, mockSrv *httptest.Server, bus domain.EventBus) *LoginHandler {
	t.Helper()
	cfg := SessionConfig{
		CookieName:      "hub_token",
		CookieDomain:    "localhost",
		CookieSecure:    false,
		AccessTokenTTL:  1 * time.Hour,
		RememberMeTTL:   30 * 24 * time.Hour,
		DefaultTenantID: "default-tenant",
	}
	cfg.InitSigningKey()
	zClient := NewZitadelClient(mockSrv.URL, "test-pat")
	return NewLoginHandler(zClient, bus, cfg)
}

// --- Login tests ---

func TestLoginHandler_Login_Success(t *testing.T) {
	mockSrv := zitadelMockServer(t)
	defer mockSrv.Close()

	bus := &mockEventBus{}
	h := newTestHandler(t, mockSrv, bus)

	body := `{"email":"correct@example.com","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify cookie is set
	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "hub_token" {
			found = true
			if !c.HttpOnly {
				t.Error("expected HttpOnly cookie")
			}
		}
	}
	if !found {
		t.Error("expected hub_token cookie to be set")
	}

	// Verify response body
	var resp loginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.UserID == "" {
		t.Error("expected non-empty UserID in response")
	}
	if resp.Email == "" {
		t.Error("expected non-empty Email in response")
	}

	// Verify event emitted
	if len(bus.emitted) != 1 || bus.emitted[0] != events.AuthLogin {
		t.Errorf("expected auth.login event, got %v", bus.emitted)
	}
}

func TestLoginHandler_Login_InvalidCredentials(t *testing.T) {
	mockSrv := zitadelMockServer(t)
	defer mockSrv.Close()

	h := newTestHandler(t, mockSrv, nil)

	body := `{"email":"correct@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLoginHandler_Login_MissingEmail(t *testing.T) {
	mockSrv := zitadelMockServer(t)
	defer mockSrv.Close()

	h := newTestHandler(t, mockSrv, nil)

	body := `{"email":"","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLoginHandler_Login_MissingPassword(t *testing.T) {
	mockSrv := zitadelMockServer(t)
	defer mockSrv.Close()

	h := newTestHandler(t, mockSrv, nil)

	body := `{"email":"correct@example.com","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLoginHandler_Login_RememberMe(t *testing.T) {
	mockSrv := zitadelMockServer(t)
	defer mockSrv.Close()

	h := newTestHandler(t, mockSrv, nil)

	body := `{"email":"correct@example.com","password":"secret","remember_me":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Should have a cookie with a longer MaxAge
	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "hub_token" {
			found = true
			// RememberMe TTL is 30 days, so MaxAge > 1 day
			if c.MaxAge <= int((24 * time.Hour).Seconds()) {
				t.Errorf("expected long MaxAge for remember_me, got %d", c.MaxAge)
			}
		}
	}
	if !found {
		t.Error("expected hub_token cookie to be set")
	}
}

// --- Me tests ---

func TestLoginHandler_Me(t *testing.T) {
	mockSrv := zitadelMockServer(t)
	defer mockSrv.Close()

	h := newTestHandler(t, mockSrv, nil)

	ctx := user.WithUserID(context.Background(), "user@example.com")
	ctx = tenant.WithTenantID(ctx, "tenant-abc")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Me(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["user_id"] != "user@example.com" {
		t.Errorf("expected user_id=user@example.com, got %q", resp["user_id"])
	}
	if resp["tenant_id"] != "tenant-abc" {
		t.Errorf("expected tenant_id=tenant-abc, got %q", resp["tenant_id"])
	}
}

// --- Logout tests ---

func TestLoginHandler_Logout(t *testing.T) {
	mockSrv := zitadelMockServer(t)
	defer mockSrv.Close()

	bus := &mockEventBus{}
	h := newTestHandler(t, mockSrv, bus)

	// Set an existing cookie so we can verify it gets cleared
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(""))
	req.AddCookie(&http.Cookie{Name: "hub_token", Value: "some-token"})

	ctx := user.WithUserID(req.Context(), "user@example.com")
	ctx = tenant.WithTenantID(ctx, "tenant-abc")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify cookie is cleared (MaxAge=-1)
	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "hub_token" {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("expected MaxAge=-1, got %d", c.MaxAge)
			}
		}
	}
	if !found {
		t.Error("expected hub_token cookie in Set-Cookie header")
	}

	// Verify event emitted
	if len(bus.emitted) != 1 || bus.emitted[0] != events.AuthLogout {
		t.Errorf("expected auth.logout event, got %v", bus.emitted)
	}

	// Verify response body
	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["message"] != "logged out" {
		t.Errorf("expected message=logged out, got %q", resp["message"])
	}
}
