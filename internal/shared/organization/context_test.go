package organization_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/organization"
)

const testOrgID = "11111111-1111-1111-1111-111111111111"

func TestWithOrgID_RoundTrip(t *testing.T) {
	ctx := organization.WithOrgID(context.Background(), testOrgID)
	got, ok := organization.OrgIDFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != testOrgID {
		t.Errorf("got %q, want %q", got, testOrgID)
	}
}

func TestOrgIDFromContext_Missing(t *testing.T) {
	if _, ok := organization.OrgIDFromContext(context.Background()); ok {
		t.Fatal("expected ok=false for empty context")
	}
}

func TestWithOrgID_EmptyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty org ID")
		}
	}()
	organization.WithOrgID(context.Background(), "")
}

func TestRequireOrgID_Success(t *testing.T) {
	ctx := organization.WithOrgID(context.Background(), testOrgID)
	got, err := organization.RequireOrgID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != testOrgID {
		t.Errorf("got %q, want %q", got, testOrgID)
	}
}

func TestRequireOrgID_Missing(t *testing.T) {
	if _, err := organization.RequireOrgID(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestMustOrgID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	organization.MustOrgID(context.Background())
}

func TestMustOrgID_Returns(t *testing.T) {
	ctx := organization.WithOrgID(context.Background(), testOrgID)
	if got := organization.MustOrgID(ctx); got != testOrgID {
		t.Errorf("got %q, want %q", got, testOrgID)
	}
}
