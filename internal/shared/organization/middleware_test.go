package organization_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/organization"
)

func TestMiddleware_ValidHeader(t *testing.T) {
	var captured string
	handler := organization.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := organization.OrgIDFromContext(r.Context())
		if !ok {
			t.Fatal("org ID not in context")
		}
		captured = id
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(organization.HeaderOrganizationID, testOrgID)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if captured != testOrgID {
		t.Errorf("captured = %q, want %q", captured, testOrgID)
	}
}

func TestMiddleware_MissingHeader_PassThrough(t *testing.T) {
	called := false
	handler := organization.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := organization.OrgIDFromContext(r.Context()); ok {
			t.Error("org ID unexpectedly in context")
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler should be called when header is missing")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMiddleware_InvalidUUID(t *testing.T) {
	handler := organization.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(organization.HeaderOrganizationID, "not-a-uuid")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
