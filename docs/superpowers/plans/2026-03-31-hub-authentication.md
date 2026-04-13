# Hub Authentication (PIQ-12) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace mock auth in web-hub with real Zitadel OIDC authentication (slim: direct login + JWT middleware + session cookies).

**Architecture:** Hub gets its own `internal/hub/auth/` package (no imports from `internal/server/`). Backend: Zitadel client for credential verification, HMAC-SHA256 JWT minting, cookie-based sessions, JWT validation middleware. Frontend: login page in web-hub SPA, AuthContext rewrite to call `/api/v1/auth/me`, dev fallback for Zitadel-less development.

**Tech Stack:** Go (chi/v5, HMAC-SHA256 JWT, Zitadel v2 API), React 19 (react-hook-form, Zod, TanStack Query), TypeScript

**Spec:** `docs/superpowers/specs/2026-03-31-hub-authentication-design.md`

---

## File Structure

### Backend (new files)

| File | Responsibility |
|------|---------------|
| `internal/hub/auth/session.go` | SessionConfig struct, InitSigningKey(), mintJWT() |
| `internal/hub/auth/zitadel.go` | Minimal Zitadel API client (Authenticate, GetSessionInfo) |
| `internal/hub/auth/jwt.go` | JWT validation middleware (cookie/header extraction, HMAC verification, context injection) |
| `internal/hub/auth/login.go` | Auth HTTP handlers (Login, Me, Logout) + writeAuthError helper |
| `internal/hub/events/topics.go` | Add `AuthLogin` and `AuthLogout` event constants |

### Backend (modified files)

| File | Change |
|------|--------|
| `configs/hub.yaml` | Bump `access_token_ttl` from `15m` to `24h` |
| `internal/hub/api/router.go` | Add auth routes, JWT middleware, CORS credentials, accept `loginHandler`/`jwtMW` params |
| `cmd/hub/main.go` | Initialize auth (Zitadel client, session config, login handler, JWT middleware), pass to router |

### Backend (test files)

| File | Tests |
|------|-------|
| `internal/hub/auth/session_test.go` | InitSigningKey, mintJWT (valid token, claims, expiry) |
| `internal/hub/auth/zitadel_test.go` | Authenticate success/failure, GetSessionInfo success/failure (httptest mock) |
| `internal/hub/auth/jwt_test.go` | Valid token, expired token, malformed token, missing cookie, dev fallback headers |
| `internal/hub/auth/login_test.go` | Login handler: success, invalid creds, missing fields, cookie set. Me handler. Logout handler. |

### Frontend (new files)

| File | Responsibility |
|------|---------------|
| `web-hub/src/api/hooks/useAuth.ts` | `useCurrentUser()` and `useLogout()` hooks |
| `web-hub/src/api/hooks/useLogin.ts` | `useLogin()` mutation hook |
| `web-hub/src/components/auth/AuthLayout.tsx` | Full-page centered login layout with branding |
| `web-hub/src/pages/login/LoginPage.tsx` | Login form (email, password, remember me, submit) |
| `web-hub/src/pages/login/index.ts` | Barrel export |

### Frontend (modified files)

| File | Change |
|------|--------|
| `web-hub/src/app/auth/AuthContext.tsx` | Rewrite: useCurrentUser(), loading state, dev fallback |
| `web-hub/src/app/routes.tsx` | Add `/login` route outside AppLayout |
| `web-hub/src/app/layout/TopBar.tsx` | Add logout dropdown to user avatar |

### Frontend (test files)

| File | Tests |
|------|-------|
| `web-hub/src/pages/login/__tests__/LoginPage.test.tsx` | Form validation, submit, error display, loading state |
| `web-hub/src/app/auth/__tests__/AuthContext.test.tsx` | Authenticated user, dev fallback, loading state |

---

## Task 1: Session Config and JWT Minting

**Files:**

- Create: `internal/hub/auth/session.go`
- Test: `internal/hub/auth/session_test.go`

- [ ] **Step 1: Write the failing tests for SessionConfig and mintJWT**

```go
// internal/hub/auth/session_test.go
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestInitSigningKey(t *testing.T) {
	t.Run("generates 32 byte key when empty", func(t *testing.T) {
		cfg := SessionConfig{}
		cfg.InitSigningKey()
		if len(cfg.SigningKey) != 32 {
			t.Fatalf("expected 32 byte key, got %d", len(cfg.SigningKey))
		}
	})

	t.Run("does not overwrite existing key", func(t *testing.T) {
		existing := []byte("my-existing-key-that-is-32bytes!")
		cfg := SessionConfig{SigningKey: existing}
		cfg.InitSigningKey()
		if string(cfg.SigningKey) != string(existing) {
			t.Fatal("InitSigningKey overwrote existing key")
		}
	})

	t.Run("generates unique keys", func(t *testing.T) {
		cfg1 := SessionConfig{}
		cfg1.InitSigningKey()
		cfg2 := SessionConfig{}
		cfg2.InitSigningKey()
		if string(cfg1.SigningKey) == string(cfg2.SigningKey) {
			t.Fatal("two calls generated identical keys")
		}
	})
}

func TestMintJWT(t *testing.T) {
	key := []byte("test-key-that-is-32-bytes-long!!")

	t.Run("produces valid 3-part JWT", func(t *testing.T) {
		token, err := mintJWT(key, "user-123", "tenant-456", "test@example.com", "Test User", time.Hour)
		if err != nil {
			t.Fatalf("mintJWT failed: %v", err)
		}
		parts := strings.SplitN(token, ".", 3)
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(parts))
		}
	})

	t.Run("header is HS256", func(t *testing.T) {
		token, _ := mintJWT(key, "user-123", "tenant-456", "test@example.com", "Test User", time.Hour)
		parts := strings.SplitN(token, ".", 3)
		headerBytes, _ := base64.RawURLEncoding.DecodeString(parts[0])
		var header struct {
			Alg string `json:"alg"`
			Typ string `json:"typ"`
		}
		if err := json.Unmarshal(headerBytes, &header); err != nil {
			t.Fatalf("failed to decode header: %v", err)
		}
		if header.Alg != "HS256" {
			t.Fatalf("expected alg HS256, got %s", header.Alg)
		}
	})

	t.Run("claims contain expected fields", func(t *testing.T) {
		token, _ := mintJWT(key, "user-123", "tenant-456", "test@example.com", "Test User", time.Hour)
		parts := strings.SplitN(token, ".", 3)
		claimsBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
		var claims struct {
			Sub   string `json:"sub"`
			Email string `json:"email"`
			Name  string `json:"name"`
			Iss   string `json:"iss"`
			Iat   int64  `json:"iat"`
			Exp   int64  `json:"exp"`
		}
		if err := json.Unmarshal(claimsBytes, &claims); err != nil {
			t.Fatalf("failed to decode claims: %v", err)
		}
		if claims.Sub != "user-123" {
			t.Errorf("expected sub user-123, got %s", claims.Sub)
		}
		if claims.Email != "test@example.com" {
			t.Errorf("expected email test@example.com, got %s", claims.Email)
		}
		if claims.Name != "Test User" {
			t.Errorf("expected name Test User, got %s", claims.Name)
		}
		if claims.Iss != "patchiq-hub" {
			t.Errorf("expected iss patchiq-hub, got %s", claims.Iss)
		}
		if claims.Exp <= claims.Iat {
			t.Error("exp should be after iat")
		}
	})

	t.Run("HMAC signature is valid", func(t *testing.T) {
		token, _ := mintJWT(key, "user-123", "tenant-456", "test@example.com", "Test User", time.Hour)
		parts := strings.SplitN(token, ".", 3)
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte(parts[0] + "." + parts[1]))
		expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
		if parts[2] != expected {
			t.Fatal("HMAC signature does not match")
		}
	})

	t.Run("tenant_id in claims", func(t *testing.T) {
		token, _ := mintJWT(key, "user-123", "tenant-456", "test@example.com", "Test User", time.Hour)
		parts := strings.SplitN(token, ".", 3)
		claimsBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
		var claims struct {
			TenantID string `json:"tenant_id"`
		}
		if err := json.Unmarshal(claimsBytes, &claims); err != nil {
			t.Fatalf("failed to decode claims: %v", err)
		}
		if claims.TenantID != "tenant-456" {
			t.Errorf("expected tenant_id tenant-456, got %s", claims.TenantID)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/ -run 'TestInitSigningKey|TestMintJWT' -v`
Expected: Compilation error — package `auth` does not exist yet.

- [ ] **Step 3: Implement session.go**

```go
// internal/hub/auth/session.go
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// SessionConfig holds cookie and TTL settings for hub login sessions.
type SessionConfig struct {
	CookieName     string
	CookieDomain   string
	CookieSecure   bool
	AccessTokenTTL time.Duration
	RememberMeTTL  time.Duration
	SigningKey      []byte
	DefaultTenantID string
	PostLoginURL   string
}

// InitSigningKey generates a random 32-byte HMAC signing key if none is set.
func (c *SessionConfig) InitSigningKey() {
	if len(c.SigningKey) == 0 {
		c.SigningKey = make([]byte, 32)
		if _, err := rand.Read(c.SigningKey); err != nil {
			panic("hub auth: failed to generate signing key: " + err.Error())
		}
	}
}

// mintJWT creates an HMAC-SHA256 signed JWT with user claims.
func mintJWT(key []byte, sub, tenantID, email, name string, ttl time.Duration) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	now := time.Now()
	claims := fmt.Sprintf(`{"sub":"%s","tenant_id":"%s","email":"%s","name":"%s","iat":%d,"exp":%d,"iss":"patchiq-hub"}`,
		sub, tenantID, email, name, now.Unix(), now.Add(ttl).Unix())
	payload := base64.RawURLEncoding.EncodeToString([]byte(claims))

	sigInput := header + "." + payload
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(sigInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return sigInput + "." + sig, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/ -run 'TestInitSigningKey|TestMintJWT' -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hub/auth/session.go internal/hub/auth/session_test.go
git commit -m "feat(hub): add session config and JWT minting for hub auth (PIQ-12)"
```

---

## Task 2: Minimal Zitadel Client

**Files:**

- Create: `internal/hub/auth/zitadel.go`
- Test: `internal/hub/auth/zitadel_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/hub/auth/zitadel_test.go
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestZitadelClientAuthenticate(t *testing.T) {
	t.Run("successful authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v2/sessions" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer test-pat" {
				t.Fatalf("unexpected auth header: %s", r.Header.Get("Authorization"))
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			checks := body["checks"].(map[string]any)
			userCheck := checks["user"].(map[string]any)
			if userCheck["loginName"] != "admin@test.com" {
				t.Fatalf("unexpected loginName: %v", userCheck["loginName"])
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"sessionId":    "sess-123",
				"sessionToken": "tok-456",
			})
		}))
		defer server.Close()

		client := NewZitadelClient(server.URL, "test-pat")
		result, err := client.Authenticate(context.Background(), "admin@test.com", "password123")
		if err != nil {
			t.Fatalf("Authenticate failed: %v", err)
		}
		if result.SessionID != "sess-123" {
			t.Errorf("expected session ID sess-123, got %s", result.SessionID)
		}
		if result.SessionToken != "tok-456" {
			t.Errorf("expected session token tok-456, got %s", result.SessionToken)
		}
	})

	t.Run("invalid credentials returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "invalid credentials"})
		}))
		defer server.Close()

		client := NewZitadelClient(server.URL, "test-pat")
		_, err := client.Authenticate(context.Background(), "bad@test.com", "wrong")
		if err == nil {
			t.Fatal("expected error for invalid credentials")
		}
	})

	t.Run("server error returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewZitadelClient(server.URL, "test-pat")
		_, err := client.Authenticate(context.Background(), "admin@test.com", "password123")
		if err == nil {
			t.Fatal("expected error for server error")
		}
	})
}

func TestZitadelClientGetSessionInfo(t *testing.T) {
	t.Run("successful session info retrieval", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v2/sessions/sess-123" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"factors": map[string]any{
						"user": map[string]any{
							"id":             "user-789",
							"loginName":      "admin@test.com",
							"displayName":    "Admin User",
							"organizationId": "org-001",
						},
					},
				},
			})
		}))
		defer server.Close()

		client := NewZitadelClient(server.URL, "test-pat")
		info, err := client.GetSessionInfo(context.Background(), "sess-123")
		if err != nil {
			t.Fatalf("GetSessionInfo failed: %v", err)
		}
		if info.UserID != "user-789" {
			t.Errorf("expected user ID user-789, got %s", info.UserID)
		}
		if info.LoginName != "admin@test.com" {
			t.Errorf("expected login name admin@test.com, got %s", info.LoginName)
		}
		if info.DisplayName != "Admin User" {
			t.Errorf("expected display name Admin User, got %s", info.DisplayName)
		}
	})

	t.Run("missing user ID returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"factors": map[string]any{
						"user": map[string]any{},
					},
				},
			})
		}))
		defer server.Close()

		client := NewZitadelClient(server.URL, "test-pat")
		_, err := client.GetSessionInfo(context.Background(), "sess-empty")
		if err == nil {
			t.Fatal("expected error for missing user ID")
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/ -run 'TestZitadelClient' -v`
Expected: Compilation error — `NewZitadelClient` not defined.

- [ ] **Step 3: Implement zitadel.go**

```go
// internal/hub/auth/zitadel.go
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AuthResult holds the result of a successful Zitadel authentication.
type AuthResult struct {
	SessionID    string
	SessionToken string
}

// SessionInfo holds user info extracted from a Zitadel session.
type SessionInfo struct {
	UserID      string
	LoginName   string
	DisplayName string
	OrgID       string
}

// ZitadelClient communicates with Zitadel's v2 APIs for user authentication.
// Uses a service account PAT for authorization.
type ZitadelClient struct {
	baseURL    string
	httpClient *http.Client
	pat        string
}

// NewZitadelClient creates a client for Zitadel's v2 APIs.
// baseURL is the full scheme+host (e.g. "http://localhost:8085").
// pat is a Personal Access Token for the service account.
func NewZitadelClient(baseURL string, pat string) *ZitadelClient {
	return &ZitadelClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		pat: pat,
	}
}

// Authenticate verifies email+password via POST /v2/sessions.
func (c *ZitadelClient) Authenticate(ctx context.Context, email, password string) (*AuthResult, error) {
	reqBody := map[string]any{
		"checks": map[string]any{
			"user": map[string]any{
				"loginName": email,
			},
			"password": map[string]any{
				"password": password,
			},
		},
	}

	body, err := c.doJSON(ctx, http.MethodPost, "/v2/sessions", reqBody)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	var resp struct {
		SessionID    string `json:"sessionId"`
		SessionToken string `json:"sessionToken"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("authenticate: decode response: %w", err)
	}

	return &AuthResult{
		SessionID:    resp.SessionID,
		SessionToken: resp.SessionToken,
	}, nil
}

// GetSessionInfo fetches session details and extracts user info.
func (c *ZitadelClient) GetSessionInfo(ctx context.Context, sessionID string) (*SessionInfo, error) {
	path := fmt.Sprintf("/v2/sessions/%s", sessionID)
	body, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get session info: %w", err)
	}

	var resp struct {
		Session struct {
			Factors struct {
				User struct {
					ID             string `json:"id"`
					LoginName      string `json:"loginName"`
					DisplayName    string `json:"displayName"`
					OrganizationID string `json:"organizationId"`
				} `json:"user"`
			} `json:"factors"`
		} `json:"session"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("get session info: decode: %w", err)
	}

	user := resp.Session.Factors.User
	if user.ID == "" {
		return nil, fmt.Errorf("get session info: no user ID in session")
	}

	return &SessionInfo{
		UserID:      user.ID,
		LoginName:   user.LoginName,
		DisplayName: user.DisplayName,
		OrgID:       user.OrganizationID,
	}, nil
}

// doJSON sends a JSON request to Zitadel and returns the response body.
func (c *ZitadelClient) doJSON(ctx context.Context, method, path string, reqBody any) ([]byte, error) {
	var bodyReader io.Reader
	if reqBody != nil {
		jsonBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.pat != "" {
		req.Header.Set("Authorization", "Bearer "+c.pat)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read response from %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("Zitadel %s %s returned %d: %s", method, path, resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("Zitadel %s %s returned %d: %s", method, path, resp.StatusCode, string(body))
	}

	return body, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/ -run 'TestZitadelClient' -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hub/auth/zitadel.go internal/hub/auth/zitadel_test.go
git commit -m "feat(hub): add minimal Zitadel client for hub auth (PIQ-12)"
```

---

## Task 3: JWT Validation Middleware

**Files:**

- Create: `internal/hub/auth/jwt.go`
- Test: `internal/hub/auth/jwt_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/hub/auth/jwt_test.go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

func TestJWTMiddleware(t *testing.T) {
	key := []byte("test-key-that-is-32-bytes-long!!")
	cfg := JWTMiddlewareConfig{
		CookieName: "piq_hub_session",
		SigningKey:  key,
	}
	mw := NewJWTMiddleware(cfg)

	// Handler that extracts user/tenant from context and writes them back.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := user.UserIDFromContext(r.Context())
		tid, _ := tenant.TenantIDFromContext(r.Context())
		w.Write([]byte(uid + "|" + tid))
	})

	t.Run("valid token in cookie sets context", func(t *testing.T) {
		token, _ := mintJWT(key, "user-123", "tenant-456", "test@example.com", "Test User", time.Hour)

		req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "piq_hub_session", Value: token})
		rec := httptest.NewRecorder()

		mw(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "user-123|tenant-456" {
			t.Fatalf("expected user-123|tenant-456, got %s", rec.Body.String())
		}
	})

	t.Run("expired token returns 401", func(t *testing.T) {
		token, _ := mintJWT(key, "user-123", "tenant-456", "test@example.com", "Test User", -time.Hour)

		req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "piq_hub_session", Value: token})
		rec := httptest.NewRecorder()

		mw(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("missing cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
		rec := httptest.NewRecorder()

		mw(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("malformed token returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "piq_hub_session", Value: "not.a.jwt"})
		rec := httptest.NewRecorder()

		mw(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("token with wrong key returns 401", func(t *testing.T) {
		wrongKey := []byte("wrong-key-that-is-32-bytes-long!")
		token, _ := mintJWT(wrongKey, "user-123", "tenant-456", "test@example.com", "Test User", time.Hour)

		req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "piq_hub_session", Value: token})
		rec := httptest.NewRecorder()

		mw(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("valid token in Authorization header", func(t *testing.T) {
		token, _ := mintJWT(key, "user-abc", "tenant-def", "header@example.com", "Header User", time.Hour)

		req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		mw(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "user-abc|tenant-def" {
			t.Fatalf("expected user-abc|tenant-def, got %s", rec.Body.String())
		}
	})

	t.Run("dev fallback with X-Tenant-ID and X-User-ID headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/catalog", nil)
		req.Header.Set("X-Tenant-ID", "dev-tenant")
		req.Header.Set("X-User-ID", "dev-user")
		rec := httptest.NewRecorder()

		mw(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "dev-user|dev-tenant" {
			t.Fatalf("expected dev-user|dev-tenant, got %s", rec.Body.String())
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/ -run 'TestJWTMiddleware' -v`
Expected: Compilation error — `JWTMiddlewareConfig` and `NewJWTMiddleware` not defined.

- [ ] **Step 3: Implement jwt.go**

```go
// internal/hub/auth/jwt.go
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// JWTMiddlewareConfig holds config for the hub JWT validation middleware.
type JWTMiddlewareConfig struct {
	CookieName string
	SigningKey  []byte
}

// NewJWTMiddleware returns chi middleware that validates HMAC-SHA256 JWTs
// from cookies or Authorization headers. On success, it injects user ID and
// tenant ID into the request context.
func NewJWTMiddleware(cfg JWTMiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r, cfg.CookieName)

			if tokenStr == "" {
				// Dev fallback: if X-Tenant-ID and X-User-ID headers are present,
				// skip JWT validation. Allows Vite dev proxy without Zitadel.
				tid := r.Header.Get("X-Tenant-ID")
				uid := r.Header.Get("X-User-ID")
				if tid != "" && uid != "" {
					ctx := user.WithUserID(r.Context(), uid)
					ctx = tenant.WithTenantID(ctx, tid)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				slog.WarnContext(r.Context(), "hub jwt: missing session token",
					"cookie_name", cfg.CookieName,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "missing session token")
				return
			}

			sub, tenantID, ok := validateHubJWT(tokenStr, cfg.SigningKey)
			if !ok {
				writeAuthError(r.Context(), w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := user.WithUserID(r.Context(), sub)
			ctx = tenant.WithTenantID(ctx, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken gets the JWT from the cookie (primary) or Authorization header (fallback).
func extractToken(r *http.Request, cookieName string) string {
	if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// validateHubJWT validates an HMAC-SHA256 JWT minted by the hub login handler.
// Returns (sub, tenant_id, true) on success.
func validateHubJWT(tokenStr string, key []byte) (sub string, tenantID string, ok bool) {
	if len(key) == 0 {
		return "", "", false
	}

	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return "", "", false
	}

	// Verify header is HS256.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", false
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "HS256" {
		return "", "", false
	}

	// Verify HMAC signature.
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return "", "", false
	}

	// Decode and validate claims.
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}

	var claims struct {
		Sub      string `json:"sub"`
		TenantID string `json:"tenant_id"`
		Exp      int64  `json:"exp"`
		Iss      string `json:"iss"`
	}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return "", "", false
	}

	if claims.Iss != "patchiq-hub" {
		return "", "", false
	}
	if claims.Exp < time.Now().Unix() {
		return "", "", false
	}
	if claims.Sub == "" || claims.TenantID == "" {
		return "", "", false
	}

	return claims.Sub, claims.TenantID, true
}

// writeAuthError writes a JSON error response for auth failures.
func writeAuthError(ctx context.Context, w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"code":    "AUTH_ERROR",
		"message": msg,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to write auth error response", "error", err, "status", status)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/ -run 'TestJWTMiddleware' -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hub/auth/jwt.go internal/hub/auth/jwt_test.go
git commit -m "feat(hub): add JWT validation middleware for hub auth (PIQ-12)"
```

---

## Task 4: Auth Handlers (Login, Me, Logout)

**Files:**

- Create: `internal/hub/auth/login.go`
- Modify: `internal/hub/events/topics.go`
- Test: `internal/hub/auth/login_test.go`

- [ ] **Step 1: Add auth event topics**

Add to `internal/hub/events/topics.go`:

```go
// Add these constants after the existing ones:
AuthLogin  = "auth.login"
AuthLogout = "auth.logout"
```

And add them to the `AllTopics()` slice.

- [ ] **Step 2: Write the failing tests**

```go
// internal/hub/auth/login_test.go
package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// mockEventBus implements domain.EventBus for testing.
type mockEventBus struct {
	emitted []string
}

func (m *mockEventBus) Emit(_ context.Context, e domain.DomainEvent) error {
	m.emitted = append(m.emitted, e.Type)
	return nil
}
func (m *mockEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (m *mockEventBus) Close() error                                    { return nil }

func TestLoginHandler(t *testing.T) {
	// Mock Zitadel server that accepts admin@test.com / correct-password.
	zitadelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/sessions":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			checks := body["checks"].(map[string]any)
			pw := checks["password"].(map[string]any)
			if pw["password"] != "correct-password" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"message": "invalid credentials"})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{
				"sessionId":    "sess-123",
				"sessionToken": "tok-456",
			})
		case "/v2/sessions/sess-123":
			json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]any{
					"factors": map[string]any{
						"user": map[string]any{
							"id":             "zitadel-user-1",
							"loginName":      "admin@test.com",
							"displayName":    "Admin User",
							"organizationId": "org-001",
						},
					},
				},
			})
		}
	}))
	defer zitadelServer.Close()

	zitadelClient := NewZitadelClient(zitadelServer.URL, "test-pat")
	eventBus := &mockEventBus{}
	cfg := SessionConfig{
		CookieName:      "piq_hub_session",
		CookieDomain:    "localhost",
		CookieSecure:    false,
		AccessTokenTTL:  24 * time.Hour,
		RememberMeTTL:   168 * time.Hour,
		DefaultTenantID: "00000000-0000-0000-0000-000000000001",
	}
	cfg.InitSigningKey()

	handler := NewLoginHandler(zitadelClient, eventBus, cfg)

	t.Run("successful login sets cookie and returns user", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"email":       "admin@test.com",
			"password":    "correct-password",
			"remember_me": false,
		})
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Check cookie was set.
		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "piq_hub_session" {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("expected piq_hub_session cookie to be set")
		}
		if !sessionCookie.HttpOnly {
			t.Error("expected HttpOnly cookie")
		}

		// Check response body.
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["email"] != "admin@test.com" {
			t.Errorf("expected email admin@test.com, got %s", resp["email"])
		}
		if resp["name"] != "Admin User" {
			t.Errorf("expected name Admin User, got %s", resp["name"])
		}
	})

	t.Run("invalid credentials returns 401", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"email":    "admin@test.com",
			"password": "wrong-password",
		})
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("missing email returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"password": "something",
		})
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing password returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"email": "admin@test.com",
		})
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestMeHandler(t *testing.T) {
	handler := NewLoginHandler(nil, nil, SessionConfig{})

	t.Run("returns user info from context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		ctx := user.WithUserID(req.Context(), "user-123")
		ctx = tenant.WithTenantID(ctx, "tenant-456")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.Me(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["user_id"] != "user-123" {
			t.Errorf("expected user_id user-123, got %s", resp["user_id"])
		}
		if resp["tenant_id"] != "tenant-456" {
			t.Errorf("expected tenant_id tenant-456, got %s", resp["tenant_id"])
		}
		if resp["role"] != "admin" {
			t.Errorf("expected role admin, got %s", resp["role"])
		}
	})
}

func TestLogoutHandler(t *testing.T) {
	eventBus := &mockEventBus{}
	cfg := SessionConfig{
		CookieName:   "piq_hub_session",
		CookieDomain: "localhost",
	}
	handler := NewLoginHandler(nil, eventBus, cfg)

	t.Run("clears cookie", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		ctx := user.WithUserID(req.Context(), "user-123")
		ctx = tenant.WithTenantID(ctx, "tenant-456")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.Logout(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "piq_hub_session" {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("expected cookie to be set (with MaxAge -1)")
		}
		if sessionCookie.MaxAge != -1 {
			t.Errorf("expected MaxAge -1 to clear cookie, got %d", sessionCookie.MaxAge)
		}
	})
}
```

**Note:** The test file needs an import for `context` and `github.com/skenzeriq/patchiq/internal/shared/domain`. The `mockEventBus` should satisfy `domain.EventBus`. Adjust the import block to:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/ -run 'TestLoginHandler|TestMeHandler|TestLogoutHandler' -v`
Expected: Compilation error — `NewLoginHandler`, `Login`, `Me`, `Logout` not defined.

- [ ] **Step 4: Implement login.go**

```go
// internal/hub/auth/login.go
package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// LoginHandler handles authentication endpoints for the hub.
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

type loginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

type loginResponse struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Login handles POST /api/v1/auth/login.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.WarnContext(ctx, "hub login: failed to decode request body", "error", err)
		writeAuthError(ctx, w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeAuthError(ctx, w, http.StatusBadRequest, "Email and password are required.")
		return
	}

	authResult, err := h.zitadel.Authenticate(ctx, req.Email, req.Password)
	if err != nil {
		slog.WarnContext(ctx, "hub login: authentication failed",
			"email", req.Email,
			"error", err,
		)
		writeAuthError(ctx, w, http.StatusUnauthorized,
			"That email/password combination didn't work. Try again?")
		return
	}

	sessionInfo, err := h.zitadel.GetSessionInfo(ctx, authResult.SessionID)
	if err != nil {
		slog.ErrorContext(ctx, "hub login: failed to get session info",
			"email", req.Email,
			"session_id", authResult.SessionID,
			"error", err,
		)
		writeAuthError(ctx, w, http.StatusInternalServerError,
			"Something went wrong. Please try again.")
		return
	}

	tenantID := h.cfg.DefaultTenantID
	if tenantID == "" {
		tenantID = sessionInfo.OrgID
	}

	userID := req.Email

	ttl := h.cfg.AccessTokenTTL
	if req.RememberMe && h.cfg.RememberMeTTL > 0 {
		ttl = h.cfg.RememberMeTTL
	}

	token, err := mintJWT(h.cfg.SigningKey, userID, tenantID, req.Email, sessionInfo.DisplayName, ttl)
	if err != nil {
		slog.ErrorContext(ctx, "hub login: failed to mint JWT",
			"email", req.Email,
			"error", err,
		)
		writeAuthError(ctx, w, http.StatusInternalServerError,
			"Something went wrong. Please try again.")
		return
	}

	cookieName := h.cfg.CookieName
	if cookieName == "" {
		cookieName = "piq_hub_session"
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

	if h.eventBus != nil {
		if emitErr := h.eventBus.Emit(ctx, domain.DomainEvent{
			ID:        domain.NewEventID(),
			Type:      events.AuthLogin,
			Resource:  "auth",
			Action:    "login",
			TenantID:  tenantID,
			Payload:   map[string]string{"email": req.Email, "user_id": userID},
			Timestamp: time.Now(),
		}); emitErr != nil {
			slog.ErrorContext(ctx, "hub login: failed to emit auth.login event",
				"email", req.Email,
				"error", emitErr,
			)
		}
	}

	slog.InfoContext(ctx, "hub login: user authenticated",
		"email", req.Email,
		"user_id", userID,
		"tenant_id", tenantID,
		"remember_me", req.RememberMe,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{
		UserID:   userID,
		TenantID: tenantID,
		Name:     sessionInfo.DisplayName,
		Email:    req.Email,
		Role:     "admin",
	})
}

// Me handles GET /api/v1/auth/me — returns current user from JWT context.
func (h *LoginHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := user.UserIDFromContext(r.Context())
	tenantID, _ := tenant.TenantIDFromContext(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"user_id":   userID,
		"tenant_id": tenantID,
		"email":     userID,
		"name":      userID,
		"role":      "admin",
	})
}

// Logout handles POST /api/v1/auth/logout — clears the session cookie.
func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookieName := h.cfg.CookieName
	if cookieName == "" {
		cookieName = "piq_hub_session"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	if h.eventBus != nil {
		userID, _ := user.UserIDFromContext(r.Context())
		tenantID, _ := tenant.TenantIDFromContext(r.Context())
		if emitErr := h.eventBus.Emit(r.Context(), domain.DomainEvent{
			ID:        domain.NewEventID(),
			Type:      events.AuthLogout,
			Resource:  "auth",
			Action:    "logout",
			TenantID:  tenantID,
			Payload:   map[string]string{"user_id": userID},
			Timestamp: time.Now(),
		}); emitErr != nil {
			slog.ErrorContext(r.Context(), "hub logout: failed to emit event", "error", emitErr)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out"})
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/ -run 'TestLoginHandler|TestMeHandler|TestLogoutHandler' -v`
Expected: All tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hub/auth/login.go internal/hub/auth/login_test.go internal/hub/events/topics.go
git commit -m "feat(hub): add login, me, and logout handlers for hub auth (PIQ-12)"
```

---

## Task 5: Wire Auth Into Hub Router and Main

**Files:**

- Modify: `internal/hub/api/router.go`
- Modify: `cmd/hub/main.go`
- Modify: `configs/hub.yaml`

- [ ] **Step 1: Update configs/hub.yaml — bump access_token_ttl**

Change line 50 in `configs/hub.yaml`:

```yaml
    access_token_ttl: 24h
```

- [ ] **Step 2: Update router.go — accept auth params and wire routes**

Modify `internal/hub/api/router.go`:

1. Update `NewRouter` signature to accept `jwtMW func(http.Handler) http.Handler` and `loginHandler` as parameters. Import `internal/hub/auth` as `hubauth`.

2. Change CORS config: set `AllowCredentials: true`.

3. Add auth routes before the tenant-scoped `/api/v1` group:
   - `POST /api/v1/auth/login` — no JWT, no tenant middleware (public)
   - Inside a JWT-protected group (no tenant middleware): `GET /api/v1/auth/me` and `POST /api/v1/auth/logout`

4. Add JWT middleware to the existing `/api/v1` tenant-scoped route group (before tenant middleware).

The updated `NewRouter` function signature:

```go
func NewRouter(pool *pgxpool.Pool, eventBus domain.EventBus, syncAPIKey string, startTime time.Time, version string, idempotencyStore idempotency.Store, corsOrigins []string, riverClient v1.RiverEnqueuer, binaryStore v1.BinaryStore, binaryBucket string, jwtMW func(http.Handler) http.Handler, loginHandler *hubauth.LoginHandler) chi.Router {
```

Key changes inside the function:

```go
// CORS: add AllowCredentials
r.Use(cors.Handler(cors.Options{
    AllowedOrigins:   origins,
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Tenant-ID", "X-Request-ID", "Idempotency-Key"},
    ExposedHeaders:   []string{"Link"},
    AllowCredentials: true,
    MaxAge:           300,
}))

// Auth routes (outside tenant middleware).
if loginHandler != nil {
    r.Post("/api/v1/auth/login", loginHandler.Login)
    // /me and /logout require a valid JWT.
    if jwtMW != nil {
        r.Group(func(r chi.Router) {
            r.Use(jwtMW)
            r.Get("/api/v1/auth/me", loginHandler.Me)
            r.Post("/api/v1/auth/logout", loginHandler.Logout)
        })
    }
}

// Existing /api/v1 group — add JWT middleware before tenant middleware.
r.Route("/api/v1", func(r chi.Router) {
    if jwtMW != nil {
        r.Use(jwtMW)
    }
    r.Use(tenant.Middleware)
    r.Use(idempotency.Middleware(idempotencyStore))
    // ... rest of routes unchanged
})
```

- [ ] **Step 3: Update cmd/hub/main.go — initialize auth**

Add auth initialization between step 7 (MinIO) and step 8 (HTTP server). Add import for `hubauth "github.com/skenzeriq/patchiq/internal/hub/auth"`.

```go
// 7.5. Authentication (Zitadel OIDC — slim: login + JWT only)
var jwtMW func(http.Handler) http.Handler
var loginHandler *hubauth.LoginHandler

if cfg.IAM.Zitadel.ClientID != "" {
    scheme := "https"
    if !cfg.IAM.Zitadel.Secure {
        scheme = "http"
    }
    zitadelBaseURL := fmt.Sprintf("%s://%s", scheme, cfg.IAM.Zitadel.Domain)
    zitadelClient := hubauth.NewZitadelClient(zitadelBaseURL, cfg.IAM.Zitadel.ServiceAccountKey)

    sessionCfg := hubauth.SessionConfig{
        CookieName:      cfg.IAM.Session.CookieName,
        CookieDomain:    cfg.IAM.Session.CookieDomain,
        CookieSecure:    cfg.IAM.Session.CookieSecure,
        AccessTokenTTL:  cfg.IAM.Session.AccessTTL,
        RememberMeTTL:   cfg.IAM.Session.RememberMeTTL,
        DefaultTenantID: "00000000-0000-0000-0000-000000000001",
        PostLoginURL:    cfg.IAM.Session.PostLoginURL,
    }
    sessionCfg.InitSigningKey()

    loginHandler = hubauth.NewLoginHandler(zitadelClient, eventBus, sessionCfg)

    jwtMW = hubauth.NewJWTMiddleware(hubauth.JWTMiddlewareConfig{
        CookieName: cfg.IAM.Session.CookieName,
        SigningKey:  sessionCfg.SigningKey,
    })

    slog.Info("hub auth initialized", "zitadel_domain", cfg.IAM.Zitadel.Domain)
} else {
    slog.Warn("hub auth not configured — using header-based auth stubs (not suitable for production)")
}
```

Update the `api.NewRouter()` call to pass the new parameters:

```go
router := api.NewRouter(pool, eventBus, syncAPIKey, startTime, version, idempotencyStore, cfg.Hub.CORSOrigins, riverClient, binaryStore, minIOCfg.Bucket, jwtMW, loginHandler)
```

- [ ] **Step 4: Verify build compiles**

Run: `cd /home/heramb/skenzeriq/patchiq && go build ./cmd/hub/`
Expected: Build succeeds.

- [ ] **Step 5: Run all hub auth tests**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/auth/... -v`
Expected: All tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hub/api/router.go cmd/hub/main.go configs/hub.yaml
git commit -m "feat(hub): wire auth into hub router and main startup (PIQ-12)"
```

---

## Task 6: Frontend — Auth Hooks (useAuth, useLogin)

**Files:**

- Create: `web-hub/src/api/hooks/useAuth.ts`
- Create: `web-hub/src/api/hooks/useLogin.ts`

- [ ] **Step 1: Create useAuth.ts**

```typescript
// web-hub/src/api/hooks/useAuth.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface AuthUser {
  user_id: string;
  tenant_id: string;
  email?: string;
  name?: string;
  role?: string;
}

export function useCurrentUser() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: async (): Promise<AuthUser> => {
      const res = await fetch('/api/v1/auth/me', {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`auth/me failed: ${res.status}`);
      return res.json() as Promise<AuthUser>;
    },
    retry: false,
    staleTime: 5 * 60 * 1000,
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        throw new Error(`Logout failed (status ${res.status})`);
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['auth'] });
      window.location.href = '/login';
    },
  });
}
```

- [ ] **Step 2: Create useLogin.ts**

```typescript
// web-hub/src/api/hooks/useLogin.ts
import { useMutation } from '@tanstack/react-query';

interface LoginRequest {
  email: string;
  password: string;
  remember_me: boolean;
}

interface LoginResponse {
  user_id: string;
  tenant_id: string;
  name: string;
  email: string;
  role: string;
}

export function useLogin() {
  return useMutation<LoginResponse, Error, LoginRequest>({
    mutationFn: async (data) => {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const err = await res
          .json()
          .catch(() => ({ message: 'Something went wrong. Please try again.' }));
        throw new Error(err.message || 'Login failed');
      }
      return res.json() as Promise<LoginResponse>;
    },
  });
}
```

- [ ] **Step 3: Commit**

```bash
git add web-hub/src/api/hooks/useAuth.ts web-hub/src/api/hooks/useLogin.ts
git commit -m "feat(web-hub): add useAuth and useLogin hooks for hub auth (PIQ-12)"
```

---

## Task 7: Frontend — AuthLayout Component

**Files:**

- Create: `web-hub/src/components/auth/AuthLayout.tsx`

- [ ] **Step 1: Create AuthLayout.tsx**

Same structure as PM's `AuthLayout` (`web/src/components/auth/AuthLayout.tsx`) with Hub-specific branding:
- Title: "PatchIQ Hub" instead of "PatchIQ"
- Subtitle: "Centralized patch catalog, feed aggregation, and fleet management for your PatchIQ deployment."
- Feature bullets: "Multi-source feed aggregation", "Centralized patch catalog", "Fleet license management"

```typescript
// web-hub/src/components/auth/AuthLayout.tsx
import { ShieldCheck } from 'lucide-react';

interface AuthLayoutProps {
  children: React.ReactNode;
}

export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div
      style={{
        display: 'flex',
        minHeight: '100vh',
        background: 'var(--bg-page)',
      }}
    >
      {/* Left panel — branding, hidden on mobile */}
      <div
        style={{
          padding: '3rem',
          background: 'var(--bg-inset)',
          borderRight: '1px solid var(--border)',
          position: 'relative',
          overflow: 'hidden',
          width: '50%',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
        }}
        className="hidden lg:flex"
      >
        <div
          style={{
            position: 'absolute',
            inset: 0,
            backgroundImage:
              'linear-gradient(var(--border) 1px, transparent 1px), linear-gradient(90deg, var(--border) 1px, transparent 1px)',
            backgroundSize: '32px 32px',
            opacity: 0.4,
          }}
        />
        <div
          style={{
            position: 'relative',
            zIndex: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '1.5rem',
            maxWidth: '360px',
            textAlign: 'center',
          }}
        >
          <div
            style={{
              display: 'flex',
              height: '64px',
              width: '64px',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: '16px',
              background: 'var(--accent)',
              boxShadow: '0 0 0 8px rgba(16,185,129,0.12)',
            }}
          >
            <ShieldCheck style={{ height: '32px', width: '32px', color: '#fff' }} />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <h1
              style={{
                fontSize: '28px',
                fontWeight: 700,
                letterSpacing: '-0.025em',
                color: 'var(--text-emphasis)',
                fontFamily: 'var(--font-sans)',
                margin: 0,
              }}
            >
              PatchIQ Hub
            </h1>
            <p style={{ fontSize: '15px', color: 'var(--text-secondary)', lineHeight: 1.6, margin: 0 }}>
              Centralized patch catalog, feed aggregation, and fleet management for your PatchIQ
              deployment.
            </p>
          </div>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: '12px',
              width: '100%',
              marginTop: '8px',
            }}
          >
            {[
              'Multi-source feed aggregation',
              'Centralized patch catalog',
              'Fleet license management',
            ].map((feature) => (
              <div
                key={feature}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '10px',
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: '8px',
                  padding: '10px 14px',
                  textAlign: 'left',
                }}
              >
                <div
                  style={{
                    width: '6px',
                    height: '6px',
                    borderRadius: '50%',
                    background: 'var(--accent)',
                    flexShrink: 0,
                  }}
                />
                <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>{feature}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Right panel — form content */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '2rem',
        }}
      >
        <div style={{ width: '100%', maxWidth: '420px' }}>
          {/* Mobile logo */}
          <div
            style={{
              alignItems: 'center',
              justifyContent: 'center',
              gap: 8,
              marginBottom: 32,
            }}
            className="flex lg:hidden"
          >
            <div
              style={{
                display: 'flex',
                height: '32px',
                width: '32px',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '8px',
                background: 'var(--accent)',
              }}
            >
              <ShieldCheck style={{ height: '18px', width: '18px', color: '#fff' }} />
            </div>
            <span
              style={{
                fontSize: '18px',
                fontWeight: 700,
                color: 'var(--text-emphasis)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              PatchIQ Hub
            </span>
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add web-hub/src/components/auth/AuthLayout.tsx
git commit -m "feat(web-hub): add AuthLayout component for login page (PIQ-12)"
```

---

## Task 8: Frontend — Login Page

**Files:**

- Create: `web-hub/src/pages/login/LoginPage.tsx`
- Create: `web-hub/src/pages/login/index.ts`

- [ ] **Step 1: Create LoginPage.tsx**

Matches PM's LoginPage aesthetic (`web/src/pages/login/LoginPage.tsx`) with these differences:
- No SSO button
- No "Forgot password?" link
- No "Have an invite? Sign up" section

```typescript
// web-hub/src/pages/login/LoginPage.tsx
import { useState, forwardRef } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useNavigate } from 'react-router';
import { Eye, EyeOff } from 'lucide-react';
import { AuthLayout } from '../../components/auth/AuthLayout';
import { useLogin } from '../../api/hooks/useLogin';

const loginSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Please enter a valid email address'),
  password: z.string().min(1, 'Password is required'),
  remember_me: z.boolean(),
});

type LoginFormValues = z.infer<typeof loginSchema>;

const inputStyle: React.CSSProperties = {
  width: '100%',
  height: '42px',
  padding: '0 12px',
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  fontSize: '13px',
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color 0.15s, box-shadow 0.15s',
  boxSizing: 'border-box',
};

const inputFocusStyle: React.CSSProperties = {
  borderColor: 'var(--accent)',
  boxShadow: '0 0 0 2px rgba(16,185,129,0.15)',
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '12px',
  fontWeight: 500,
  color: 'var(--text-secondary)',
  marginBottom: '6px',
  fontFamily: 'var(--font-sans)',
};

const errorStyle: React.CSSProperties = {
  fontSize: '11px',
  color: '#ef4444',
  marginTop: '4px',
  fontFamily: 'var(--font-sans)',
};

const FocusInput = forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  function FocusInput({ style, ...rest }, ref) {
    const [focused, setFocused] = useState(false);
    return (
      <input
        ref={ref}
        style={{ ...inputStyle, ...(focused ? inputFocusStyle : {}), ...style }}
        onFocus={() => setFocused(true)}
        onBlur={(e) => {
          setFocused(false);
          rest.onBlur?.(e);
        }}
        {...rest}
      />
    );
  },
);

export function LoginPage() {
  const navigate = useNavigate();
  const login = useLogin();
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: '',
      password: '',
      remember_me: false,
    },
  });

  const onSubmit = (data: LoginFormValues) => {
    login.mutate(data, {
      onSuccess: () => {
        void navigate('/');
      },
    });
  };

  return (
    <AuthLayout>
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: '12px',
          padding: '32px',
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        <div style={{ textAlign: 'center', marginBottom: '28px' }}>
          <h1
            style={{
              fontSize: '22px',
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              letterSpacing: '-0.02em',
              fontFamily: 'var(--font-sans)',
              marginBottom: '6px',
            }}
          >
            Welcome back
          </h1>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)', fontFamily: 'var(--font-sans)' }}>
            Sign in to the Hub Manager to continue
          </p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} noValidate style={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
          {login.isError && (
            <div
              style={{
                borderRadius: '6px',
                background: 'rgba(239,68,68,0.08)',
                border: '1px solid rgba(239,68,68,0.25)',
                padding: '10px 12px',
                fontSize: '13px',
                color: '#ef4444',
                fontFamily: 'var(--font-sans)',
              }}
            >
              {login.error.message}
            </div>
          )}

          <div>
            <label htmlFor="email" style={labelStyle}>Email</label>
            <FocusInput
              id="email"
              type="email"
              placeholder="you@company.com"
              autoComplete="email"
              aria-invalid={!!errors.email}
              {...register('email')}
            />
            {errors.email && <p style={errorStyle}>{errors.email.message}</p>}
          </div>

          <div>
            <label htmlFor="password" style={labelStyle}>Password</label>
            <div style={{ position: 'relative' }}>
              <FocusInput
                id="password"
                type={showPassword ? 'text' : 'password'}
                placeholder="Enter your password"
                autoComplete="current-password"
                aria-invalid={!!errors.password}
                style={{ paddingRight: '42px' }}
                {...register('password')}
              />
              <button
                type="button"
                style={{
                  position: 'absolute',
                  right: '12px',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: 0,
                  display: 'flex',
                  color: 'var(--text-muted)',
                }}
                onClick={() => setShowPassword(!showPassword)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? (
                  <EyeOff style={{ width: '16px', height: '16px' }} />
                ) : (
                  <Eye style={{ width: '16px', height: '16px' }} />
                )}
              </button>
            </div>
            {errors.password && <p style={errorStyle}>{errors.password.message}</p>}
          </div>

          <div style={{ display: 'flex', alignItems: 'center' }}>
            <label
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                fontSize: '13px',
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                fontFamily: 'var(--font-sans)',
              }}
            >
              <input
                type="checkbox"
                style={{
                  width: '14px',
                  height: '14px',
                  accentColor: 'var(--accent)',
                  cursor: 'pointer',
                }}
                {...register('remember_me')}
              />
              Remember me
            </label>
          </div>

          <button
            type="submit"
            disabled={login.isPending}
            style={{
              width: '100%',
              height: '44px',
              background: login.isPending ? 'rgba(16,185,129,0.6)' : 'var(--accent)',
              border: 'none',
              borderRadius: '6px',
              fontSize: '14px',
              fontWeight: 600,
              color: '#fff',
              cursor: login.isPending ? 'not-allowed' : 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              fontFamily: 'var(--font-sans)',
              transition: 'opacity 0.15s',
            }}
          >
            {login.isPending ? (
              <>
                <span
                  style={{
                    width: '16px',
                    height: '16px',
                    borderRadius: '50%',
                    border: '2px solid rgba(255,255,255,0.4)',
                    borderTopColor: '#fff',
                    animation: 'spin 0.7s linear infinite',
                    display: 'inline-block',
                  }}
                />
                Signing in...
              </>
            ) : (
              'Sign in'
            )}
          </button>
        </form>
      </div>
    </AuthLayout>
  );
}
```

- [ ] **Step 2: Create barrel export**

```typescript
// web-hub/src/pages/login/index.ts
export { LoginPage } from './LoginPage';
```

- [ ] **Step 3: Commit**

```bash
git add web-hub/src/pages/login/LoginPage.tsx web-hub/src/pages/login/index.ts
git commit -m "feat(web-hub): add login page for hub auth (PIQ-12)"
```

---

## Task 9: Frontend — Rewrite AuthContext and Wire Routes

**Files:**

- Modify: `web-hub/src/app/auth/AuthContext.tsx`
- Modify: `web-hub/src/app/routes.tsx`
- Modify: `web-hub/src/app/layout/TopBar.tsx`

- [ ] **Step 1: Rewrite AuthContext.tsx**

Replace the entire file content:

```typescript
// web-hub/src/app/auth/AuthContext.tsx
import { createContext, useContext } from 'react';
import { useCurrentUser } from '../../api/hooks/useAuth';

interface AuthUser {
  user_id: string;
  tenant_id?: string;
  email?: string;
  name?: string;
  role?: string;
}

interface AuthContextValue {
  user: AuthUser;
}

const AuthContext = createContext<AuthContextValue | null>(null);

interface AuthProviderProps {
  children: React.ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const { data: user, isLoading, isError } = useCurrentUser();

  if (isLoading) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>;
  }

  // In dev mode, if auth fails, use a stub user instead of redirecting to login
  const devUser: AuthUser = {
    user_id: 'dev-user',
    tenant_id: '00000000-0000-0000-0000-000000000001',
    email: 'dev@patchiq.local',
    name: 'Dev User',
    role: 'admin',
  };

  const effectiveUser = user ?? (isError ? devUser : null);

  if (!effectiveUser) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>;
  }

  return <AuthContext.Provider value={{ user: effectiveUser }}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextValue => {
  const ctx = useContext(AuthContext);
  if (ctx === null) {
    throw new Error('useAuth must be used within an <AuthProvider>');
  }
  return ctx;
};
```

- [ ] **Step 2: Update routes.tsx — add /login route outside AppLayout**

The login page must be outside `AppLayout` (no sidebar, no topbar). Update `web-hub/src/app/routes.tsx`:

```typescript
import { createBrowserRouter, Navigate } from 'react-router';
import { AppLayout } from './layout/AppLayout';
import { DashboardPage } from '../pages/dashboard/DashboardPage';
import { CatalogPage } from '../pages/catalog/CatalogPage';
import { CatalogDetailPage } from '../pages/catalog/CatalogDetailPage';
import { FeedsPage } from '../pages/feeds/FeedsPage';
import { FeedDetailPage } from '../pages/feeds/FeedDetailPage';
import { LicensesPage } from '../pages/licenses/LicensesPage';
import { LicenseDetailPage } from '../pages/licenses/LicenseDetailPage';
import { ClientsPage } from '../pages/clients/ClientsPage';
import { ClientDetailPage } from '../pages/clients/ClientDetailPage';
import { SettingsPage } from '../pages/settings/SettingsPage';
import { GeneralSettingsPage } from '../pages/settings/GeneralSettingsPage';
import { IAMSettingsPage } from '../pages/settings/IAMSettingsPage';
import { FeedConfigSettingsPage } from '../pages/settings/FeedConfigSettingsPage';
import { APIWebhookSettingsPage } from '../pages/settings/APIWebhookSettingsPage';
import { DeploymentsPage } from '../pages/deployments/DeploymentsPage';
import { LoginPage } from '../pages/login';

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    element: <AppLayout />,
    children: [
      { path: '/', element: <DashboardPage /> },
      { path: '/catalog', element: <CatalogPage /> },
      { path: '/catalog/:id', element: <CatalogDetailPage /> },
      { path: '/feeds', element: <FeedsPage /> },
      { path: '/feeds/:id', element: <FeedDetailPage /> },
      { path: '/licenses', element: <LicensesPage /> },
      { path: '/licenses/:id', element: <LicenseDetailPage /> },
      { path: '/clients', element: <ClientsPage /> },
      { path: '/clients/:id', element: <ClientDetailPage /> },
      { path: '/deployments', element: <DeploymentsPage /> },
      {
        path: '/settings',
        element: <SettingsPage />,
        children: [
          { index: true, element: <Navigate to="/settings/general" replace /> },
          { path: 'general', element: <GeneralSettingsPage /> },
          { path: 'iam', element: <IAMSettingsPage /> },
          { path: 'feeds', element: <FeedConfigSettingsPage /> },
          { path: 'api', element: <APIWebhookSettingsPage /> },
        ],
      },
    ],
  },
]);
```

- [ ] **Step 3: Update TopBar.tsx — add logout to user avatar**

Add a dropdown menu to the user avatar in `web-hub/src/app/layout/TopBar.tsx`. Import `useLogout` and `DropdownMenu` from `@patchiq/ui`. Add `LogOut` icon from lucide-react.

Replace the user avatar `<div>` section (the `<div className="flex items-center gap-2">` containing the avatar circle and name) with a dropdown menu:

```typescript
import { NavLink, useLocation } from 'react-router';
import { Bell, Sun, Moon, LogOut } from 'lucide-react';
import {
  SidebarTrigger,
  Button,
  useTheme,
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@patchiq/ui';
import { useAuth } from '../auth/AuthContext';
import { useLogout } from '../../api/hooks/useAuth';
```

Replace the user avatar section with:

```tsx
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <button
      className="flex items-center gap-2 rounded-md px-2 py-1 hover:bg-accent/10 transition-colors"
      style={{ background: 'none', border: 'none', cursor: 'pointer' }}
    >
      <div
        className="flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold"
        style={{
          background: 'var(--accent-subtle)',
          color: 'var(--accent)',
        }}
      >
        {initials}
      </div>
      <span
        className="text-sm font-medium hidden sm:inline"
        style={{ color: 'var(--text-primary)' }}
      >
        {user.name}
      </span>
    </button>
  </DropdownMenuTrigger>
  <DropdownMenuContent align="end" className="w-48">
    <div className="px-2 py-1.5">
      <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{user.name}</p>
      <p className="text-xs" style={{ color: 'var(--text-muted)' }}>{user.email}</p>
    </div>
    <DropdownMenuSeparator />
    <DropdownMenuItem
      onClick={() => logout.mutate()}
      className="text-destructive focus:text-destructive"
    >
      <LogOut className="mr-2 h-4 w-4" />
      Sign out
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

Add `const logout = useLogout();` after the existing `useAuth()` call inside the `TopBar` component.

- [ ] **Step 4: Verify frontend builds**

Run: `cd /home/heramb/skenzeriq/patchiq/web-hub && pnpm tsc --noEmit`
Expected: No type errors.

- [ ] **Step 5: Commit**

```bash
git add web-hub/src/app/auth/AuthContext.tsx web-hub/src/app/routes.tsx web-hub/src/app/layout/TopBar.tsx
git commit -m "feat(web-hub): rewrite AuthContext, add login route, wire logout (PIQ-12)"
```

---

## Task 10: Frontend Tests

**Files:**

- Create: `web-hub/src/pages/login/__tests__/LoginPage.test.tsx`
- Create: `web-hub/src/app/auth/__tests__/AuthContext.test.tsx`

- [ ] **Step 1: Write LoginPage tests**

```typescript
// web-hub/src/pages/login/__tests__/LoginPage.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { LoginPage } from '../LoginPage';

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
};

describe('LoginPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders email and password fields', () => {
    render(<LoginPage />, { wrapper: createWrapper() });
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
    expect(screen.getByLabelText('Password')).toBeInTheDocument();
  });

  it('shows validation errors for empty fields', async () => {
    const user = userEvent.setup();
    render(<LoginPage />, { wrapper: createWrapper() });

    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    await waitFor(() => {
      expect(screen.getByText('Email is required')).toBeInTheDocument();
      expect(screen.getByText('Password is required')).toBeInTheDocument();
    });
  });

  it('shows email validation error for invalid email', async () => {
    const user = userEvent.setup();
    render(<LoginPage />, { wrapper: createWrapper() });

    await user.type(screen.getByLabelText('Email'), 'not-an-email');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    await waitFor(() => {
      expect(screen.getByText('Please enter a valid email address')).toBeInTheDocument();
    });
  });

  it('toggles password visibility', async () => {
    const user = userEvent.setup();
    render(<LoginPage />, { wrapper: createWrapper() });

    const passwordInput = screen.getByLabelText('Password');
    expect(passwordInput).toHaveAttribute('type', 'password');

    await user.click(screen.getByLabelText('Show password'));
    expect(passwordInput).toHaveAttribute('type', 'text');

    await user.click(screen.getByLabelText('Hide password'));
    expect(passwordInput).toHaveAttribute('type', 'password');
  });

  it('shows server error on failed login', async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ message: 'Invalid credentials' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    render(<LoginPage />, { wrapper: createWrapper() });

    await user.type(screen.getByLabelText('Email'), 'admin@test.com');
    await user.type(screen.getByLabelText('Password'), 'wrong');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    await waitFor(() => {
      expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
    });
  });
});
```

- [ ] **Step 2: Write AuthContext tests**

```typescript
// web-hub/src/app/auth/__tests__/AuthContext.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { AuthProvider, useAuth } from '../AuthContext';

function TestConsumer() {
  const { user } = useAuth();
  return <div data-testid="user">{user.name ?? user.email ?? user.user_id}</div>;
}

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('AuthProvider', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('shows loading state initially', () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => new Promise(() => {})); // never resolves
    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      </Wrapper>,
    );
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders children with authenticated user', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          user_id: 'user-123',
          tenant_id: 'tenant-456',
          email: 'admin@test.com',
          name: 'Admin User',
          role: 'admin',
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      </Wrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('user')).toHaveTextContent('Admin User');
    });
  });

  it('falls back to dev user on auth failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response('', { status: 401 }),
    );

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      </Wrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('user')).toHaveTextContent('Dev User');
    });
  });
});
```

- [ ] **Step 3: Run frontend tests**

Run: `cd /home/heramb/skenzeriq/patchiq/web-hub && pnpm vitest run src/pages/login/__tests__/LoginPage.test.tsx src/app/auth/__tests__/AuthContext.test.tsx`
Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add web-hub/src/pages/login/__tests__/LoginPage.test.tsx web-hub/src/app/auth/__tests__/AuthContext.test.tsx
git commit -m "test(web-hub): add LoginPage and AuthContext tests for hub auth (PIQ-12)"
```

---

## Task 11: Integration Verification

- [ ] **Step 1: Run all backend tests**

Run: `cd /home/heramb/skenzeriq/patchiq && go test ./internal/hub/... -v -count=1`
Expected: All tests PASS, including new auth tests.

- [ ] **Step 2: Run all frontend tests**

Run: `cd /home/heramb/skenzeriq/patchiq/web-hub && pnpm vitest run`
Expected: All tests PASS.

- [ ] **Step 3: Verify hub binary builds**

Run: `cd /home/heramb/skenzeriq/patchiq && go build ./cmd/hub/`
Expected: Build succeeds.

- [ ] **Step 4: Verify frontend builds**

Run: `cd /home/heramb/skenzeriq/patchiq/web-hub && pnpm tsc --noEmit && pnpm build`
Expected: No type errors, build succeeds.

- [ ] **Step 5: Verify full lint passes**

Run: `cd /home/heramb/skenzeriq/patchiq && make lint`
Expected: No lint errors from new code.
