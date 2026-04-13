package auth_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

func TestMe_WithPermissions(t *testing.T) {
	store := &mockPermissionStore{
		permissions: []auth.Permission{
			{Resource: "endpoints", Action: "read", Scope: "*"},
			{Resource: "deployments", Action: "create", Scope: "*"},
		},
	}

	h := auth.NewSSOHandler(auth.SSOConfig{}, nil)
	h.PermStore = store

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	ctx := user.WithUserID(req.Context(), "user-123")
	ctx = tenant.WithTenantID(ctx, "00000000-0000-0000-0000-000000000001")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		UserID      string `json:"user_id"`
		TenantID    string `json:"tenant_id"`
		Permissions []struct {
			Resource string `json:"resource"`
			Action   string `json:"action"`
			Scope    string `json:"scope"`
		} `json:"permissions"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.UserID != "user-123" {
		t.Errorf("user_id = %q, want %q", resp.UserID, "user-123")
	}
	if resp.TenantID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("tenant_id = %q, want %q", resp.TenantID, "00000000-0000-0000-0000-000000000001")
	}
	if len(resp.Permissions) != 2 {
		t.Fatalf("permissions count = %d, want 2", len(resp.Permissions))
	}
	if resp.Permissions[0].Resource != "endpoints" || resp.Permissions[0].Action != "read" {
		t.Errorf("permissions[0] = %+v, want endpoints:read:*", resp.Permissions[0])
	}
}

func TestMe_NilPermStore(t *testing.T) {
	h := auth.NewSSOHandler(auth.SSOConfig{}, nil)
	// PermStore is nil by default

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	ctx := user.WithUserID(req.Context(), "user-456")
	ctx = tenant.WithTenantID(ctx, "00000000-0000-0000-0000-000000000001")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		UserID      string      `json:"user_id"`
		Permissions interface{} `json:"permissions"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.UserID != "user-456" {
		t.Errorf("user_id = %q, want %q", resp.UserID, "user-456")
	}
	if resp.Permissions != nil {
		t.Errorf("permissions should be nil/omitted when PermStore is nil, got %v", resp.Permissions)
	}
}

func TestMe_PermStoreError(t *testing.T) {
	store := &mockPermissionStore{
		err: fmt.Errorf("db connection lost"),
	}

	h := auth.NewSSOHandler(auth.SSOConfig{}, nil)
	h.PermStore = store

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	ctx := user.WithUserID(req.Context(), "user-789")
	ctx = tenant.WithTenantID(ctx, "00000000-0000-0000-0000-000000000001")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Me(w, req)

	// Should still return 200 with user info, just no permissions
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		UserID      string      `json:"user_id"`
		Permissions interface{} `json:"permissions"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.UserID != "user-789" {
		t.Errorf("user_id = %q, want %q", resp.UserID, "user-789")
	}
	if resp.Permissions != nil {
		t.Errorf("permissions should be nil when store errors, got %v", resp.Permissions)
	}
}

func TestMe_Unauthenticated(t *testing.T) {
	h := auth.NewSSOHandler(auth.SSOConfig{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMe_EmptyTenantSkipsPermissions(t *testing.T) {
	store := &mockPermissionStore{
		permissions: []auth.Permission{
			{Resource: "*", Action: "*", Scope: "*"},
		},
	}

	h := auth.NewSSOHandler(auth.SSOConfig{}, nil)
	h.PermStore = store

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	// User ID set but no tenant ID
	ctx := user.WithUserID(req.Context(), "user-no-tenant")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Permissions interface{} `json:"permissions"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// PermStore is set but tenantID is empty, so permissions should be skipped
	if resp.Permissions != nil {
		t.Errorf("permissions should be nil when tenant is empty, got %v", resp.Permissions)
	}
}
