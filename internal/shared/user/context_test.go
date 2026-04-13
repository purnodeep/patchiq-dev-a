package user_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/user"
)

func TestUserIDContext(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		wantOK bool
	}{
		{"round-trips user ID", "user-123", true},
		{"returns false for empty context", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.userID != "" {
				ctx = user.WithUserID(ctx, tt.userID)
			}
			got, ok := user.UserIDFromContext(ctx)
			if ok != tt.wantOK {
				t.Fatalf("UserIDFromContext() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got != tt.userID {
				t.Errorf("UserIDFromContext() = %q, want %q", got, tt.userID)
			}
		})
	}
}

func TestWithUserIDPanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("WithUserID(\"\") did not panic")
		}
	}()
	user.WithUserID(context.Background(), "")
}

func TestMustUserID(t *testing.T) {
	ctx := user.WithUserID(context.Background(), "user-456")
	if got := user.MustUserID(ctx); got != "user-456" {
		t.Errorf("MustUserID() = %q, want %q", got, "user-456")
	}
}

func TestMustUserIDPanicsWhenMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustUserID() did not panic on empty context")
		}
	}()
	user.MustUserID(context.Background())
}

func TestRequireUserID_Success(t *testing.T) {
	ctx := user.WithUserID(context.Background(), "user-789")
	got, err := user.RequireUserID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "user-789" {
		t.Errorf("got %q, want %q", got, "user-789")
	}
}

func TestRequireUserID_Missing(t *testing.T) {
	_, err := user.RequireUserID(context.Background())
	if err == nil {
		t.Fatal("expected error for missing user ID, got nil")
	}
}
