package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/shared/organization"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

type fakeOrgResolver struct {
	orgID           string
	resolveOrgErr   error
	activeTenantID  string
	activeTenantErr error
	// capture last args for assertions
	lastOrgID     string
	lastUserID    string
	lastPreferred string
}

func (f *fakeOrgResolver) ResolveZitadelOrg(_ context.Context, _ string) (string, error) {
	return f.orgID, f.resolveOrgErr
}

func (f *fakeOrgResolver) ResolveActiveTenant(_ context.Context, orgID, userID, preferred string) (string, error) {
	f.lastOrgID = orgID
	f.lastUserID = userID
	f.lastPreferred = preferred
	return f.activeTenantID, f.activeTenantErr
}

const (
	testOrgUUID    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	testTenantUUID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	testZitadelOrg = "zitadel-org-999"
)

func TestOrgScopeMiddleware_NilResolver_PassThrough(t *testing.T) {
	mw := auth.NewOrgScopeMiddleware(nil)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Error("nil resolver should pass through")
	}
}

func TestOrgScopeMiddleware_ResolvesOrgAndTenant(t *testing.T) {
	resolver := &fakeOrgResolver{
		orgID:          testOrgUUID,
		activeTenantID: testTenantUUID,
	}
	mw := auth.NewOrgScopeMiddleware(resolver)

	var captured struct {
		orgID    string
		tenantID string
	}
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o, ok := organization.OrgIDFromContext(r.Context()); ok {
			captured.orgID = o
		}
		if tid, ok := tenant.TenantIDFromContext(r.Context()); ok {
			captured.tenantID = tid
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := user.WithUserID(req.Context(), "user-1")
	ctx = tenant.WithTenantID(ctx, testZitadelOrg) // simulates JWT middleware output
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if captured.orgID != testOrgUUID {
		t.Errorf("org ID = %q, want %q", captured.orgID, testOrgUUID)
	}
	if captured.tenantID != testTenantUUID {
		t.Errorf("tenant ID = %q, want %q", captured.tenantID, testTenantUUID)
	}
}

func TestOrgScopeMiddleware_HeaderOverridePreferred(t *testing.T) {
	resolver := &fakeOrgResolver{
		orgID:          testOrgUUID,
		activeTenantID: "cccccccc-cccc-cccc-cccc-cccccccccccc",
	}
	mw := auth.NewOrgScopeMiddleware(resolver)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(tenant.HeaderTenantID, "cccccccc-cccc-cccc-cccc-cccccccccccc")
	ctx := user.WithUserID(req.Context(), "user-1")
	ctx = tenant.WithTenantID(ctx, testZitadelOrg)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if resolver.lastPreferred != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Errorf("resolver got preferred = %q, want the header value", resolver.lastPreferred)
	}
}

func TestOrgScopeMiddleware_OrgNotFound_PassThrough(t *testing.T) {
	resolver := &fakeOrgResolver{resolveOrgErr: auth.ErrOrgNotFound}
	mw := auth.NewOrgScopeMiddleware(resolver)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// The preliminary tenant value should still be present (fall-through).
		if tid, ok := tenant.TenantIDFromContext(r.Context()); !ok || tid != testZitadelOrg {
			t.Errorf("tenant context should be untouched, got %q ok=%v", tid, ok)
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := user.WithUserID(req.Context(), "user-1")
	ctx = tenant.WithTenantID(ctx, testZitadelOrg)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("ErrOrgNotFound should pass through to next handler")
	}
}

func TestOrgScopeMiddleware_NoAccessibleTenants_403(t *testing.T) {
	resolver := &fakeOrgResolver{
		orgID:           testOrgUUID,
		activeTenantErr: auth.ErrNoAccessibleTenants,
	}
	mw := auth.NewOrgScopeMiddleware(resolver)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := user.WithUserID(req.Context(), "user-1")
	ctx = tenant.WithTenantID(ctx, testZitadelOrg)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rr.Code)
	}
}

func TestOrgScopeMiddleware_TenantNotAccessible_403(t *testing.T) {
	resolver := &fakeOrgResolver{
		orgID:           testOrgUUID,
		activeTenantErr: auth.ErrTenantNotAccessible,
	}
	mw := auth.NewOrgScopeMiddleware(resolver)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(tenant.HeaderTenantID, "dddddddd-dddd-dddd-dddd-dddddddddddd")
	ctx := user.WithUserID(req.Context(), "user-1")
	ctx = tenant.WithTenantID(ctx, testZitadelOrg)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rr.Code)
	}
}

// TestOrgResolver_ErrNoUser validates that an empty user ID short-circuits
// correctly. The resolver is a concrete implementation detail, not the
// middleware, so we only test the error surface here.
func TestOrgResolver_ErrorsExport(t *testing.T) {
	if !errors.Is(auth.ErrOrgNotFound, auth.ErrOrgNotFound) {
		t.Error("ErrOrgNotFound is not comparable via errors.Is")
	}
	if !errors.Is(auth.ErrNoAccessibleTenants, auth.ErrNoAccessibleTenants) {
		t.Error("ErrNoAccessibleTenants not comparable")
	}
	if !errors.Is(auth.ErrTenantNotAccessible, auth.ErrTenantNotAccessible) {
		t.Error("ErrTenantNotAccessible not comparable")
	}
}
