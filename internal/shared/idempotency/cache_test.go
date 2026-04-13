package idempotency_test

import (
	"context"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/idempotency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_GetSet(t *testing.T) {
	store := idempotency.NewMemoryStore()
	ctx := context.Background()

	want := idempotency.CachedResponse{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"ok":true}`),
	}

	err := store.Set(ctx, "tenant-1", "key-abc", want, time.Minute)
	require.NoError(t, err)

	got, found, err := store.Get(ctx, "tenant-1", "key-abc")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, want.StatusCode, got.StatusCode)
	assert.Equal(t, want.ContentType, got.ContentType)
	assert.Equal(t, want.Body, got.Body)
}

func TestMemoryStore_Miss(t *testing.T) {
	store := idempotency.NewMemoryStore()
	ctx := context.Background()

	_, found, err := store.Get(ctx, "tenant-1", "nonexistent-key")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestMemoryStore_DifferentTenants(t *testing.T) {
	store := idempotency.NewMemoryStore()
	ctx := context.Background()

	resp1 := idempotency.CachedResponse{StatusCode: 200, ContentType: "application/json", Body: []byte(`{"tenant":"one"}`)}
	resp2 := idempotency.CachedResponse{StatusCode: 201, ContentType: "application/json", Body: []byte(`{"tenant":"two"}`)}

	require.NoError(t, store.Set(ctx, "tenant-1", "shared-key", resp1, time.Minute))
	require.NoError(t, store.Set(ctx, "tenant-2", "shared-key", resp2, time.Minute))

	got1, found1, err := store.Get(ctx, "tenant-1", "shared-key")
	require.NoError(t, err)
	assert.True(t, found1)
	assert.Equal(t, resp1.StatusCode, got1.StatusCode)
	assert.Equal(t, resp1.Body, got1.Body)

	got2, found2, err := store.Get(ctx, "tenant-2", "shared-key")
	require.NoError(t, err)
	assert.True(t, found2)
	assert.Equal(t, resp2.StatusCode, got2.StatusCode)
	assert.Equal(t, resp2.Body, got2.Body)
}

func TestMemoryStore_DifferentKeys(t *testing.T) {
	store := idempotency.NewMemoryStore()
	ctx := context.Background()

	resp1 := idempotency.CachedResponse{StatusCode: 200, ContentType: "text/plain", Body: []byte("first")}
	resp2 := idempotency.CachedResponse{StatusCode: 202, ContentType: "text/plain", Body: []byte("second")}

	require.NoError(t, store.Set(ctx, "tenant-1", "key-1", resp1, time.Minute))
	require.NoError(t, store.Set(ctx, "tenant-1", "key-2", resp2, time.Minute))

	got1, found1, err := store.Get(ctx, "tenant-1", "key-1")
	require.NoError(t, err)
	assert.True(t, found1)
	assert.Equal(t, resp1.Body, got1.Body)

	got2, found2, err := store.Get(ctx, "tenant-1", "key-2")
	require.NoError(t, err)
	assert.True(t, found2)
	assert.Equal(t, resp2.Body, got2.Body)
}
