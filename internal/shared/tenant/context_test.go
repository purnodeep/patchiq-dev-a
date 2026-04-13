package tenant_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

func TestWithTenantID_RoundTrip(t *testing.T) {
	ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
	got, ok := tenant.TenantIDFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if got != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("got %q, want %q", got, "00000000-0000-0000-0000-000000000001")
	}
}

func TestTenantIDFromContext_Missing(t *testing.T) {
	_, ok := tenant.TenantIDFromContext(context.Background())
	if ok {
		t.Fatal("expected ok=false for empty context")
	}
}

func TestMustTenantID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing tenant ID")
		}
	}()
	tenant.MustTenantID(context.Background())
}

func TestMustTenantID_Returns(t *testing.T) {
	ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
	got := tenant.MustTenantID(ctx)
	if got != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("got %q, want %q", got, "00000000-0000-0000-0000-000000000001")
	}
}

func TestRequireTenantID_Success(t *testing.T) {
	ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
	got, err := tenant.RequireTenantID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("got %q, want %q", got, "00000000-0000-0000-0000-000000000001")
	}
}

func TestRequireTenantID_Missing(t *testing.T) {
	_, err := tenant.RequireTenantID(context.Background())
	if err == nil {
		t.Fatal("expected error for missing tenant ID, got nil")
	}
}
