package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/server/auth"
)

func TestMemorySessionStore(t *testing.T) {
	store := auth.NewMemorySessionStore()
	ctx := context.Background()

	t.Run("store and retrieve", func(t *testing.T) {
		err := store.StoreRefreshToken(ctx, "user-1", "token-abc", 1*time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		token, err := store.GetRefreshToken(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if token != "token-abc" {
			t.Errorf("token = %q, want %q", token, "token-abc")
		}
	})

	t.Run("delete", func(t *testing.T) {
		_ = store.StoreRefreshToken(ctx, "user-2", "token-xyz", 1*time.Hour)
		err := store.DeleteRefreshToken(ctx, "user-2")
		if err != nil {
			t.Fatal(err)
		}
		_, err = store.GetRefreshToken(ctx, "user-2")
		if err != auth.ErrSessionNotFound {
			t.Errorf("err = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("expired token not returned", func(t *testing.T) {
		_ = store.StoreRefreshToken(ctx, "user-3", "token-old", 1*time.Millisecond)
		time.Sleep(5 * time.Millisecond)
		_, err := store.GetRefreshToken(ctx, "user-3")
		if err != auth.ErrSessionNotFound {
			t.Errorf("err = %v, want ErrSessionNotFound for expired token", err)
		}
	})
}
