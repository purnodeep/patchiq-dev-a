package tenant_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

func TestMiddleware_ValidHeader(t *testing.T) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	var captured string

	handler := tenant.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := tenant.TenantIDFromContext(r.Context())
		if !ok {
			t.Fatal("tenant ID not in context")
		}
		captured = id
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(tenant.HeaderTenantID, tenantID)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if captured != tenantID {
		t.Errorf("captured = %q, want %q", captured, tenantID)
	}
}

func TestMiddleware_MissingHeader(t *testing.T) {
	handler := tenant.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected error message in response body")
	}
}

func TestMiddleware_InvalidUUID(t *testing.T) {
	handler := tenant.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(tenant.HeaderTenantID, "not-a-uuid")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
