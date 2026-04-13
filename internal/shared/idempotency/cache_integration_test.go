//go:build integration

package idempotency_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valkey-io/valkey-go"

	"github.com/skenzeriq/patchiq/internal/shared/idempotency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupValkeyStore starts a Valkey container and returns a ValkeyStore backed by
// it. Container teardown is registered via t.Cleanup.
func setupValkeyStore(t *testing.T) *idempotency.ValkeyStore {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "valkey/valkey:9-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "start valkey container")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate valkey container: %v", err)
		}
	})

	endpoint, err := container.Endpoint(ctx, "")
	require.NoError(t, err, "get valkey container endpoint")

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{endpoint},
	})
	require.NoError(t, err, "create valkey client")

	t.Cleanup(func() {
		client.Close()
	})

	return idempotency.NewValkeyStore(client)
}

func TestValkeyStore_GetSet(t *testing.T) {
	store := setupValkeyStore(t)
	ctx := context.Background()

	want := idempotency.CachedResponse{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"ok":true}`),
	}

	err := store.Set(ctx, "tenant-1", "key-abc", want, time.Minute)
	require.NoError(t, err, "Set should succeed")

	got, found, err := store.Get(ctx, "tenant-1", "key-abc")
	require.NoError(t, err, "Get should succeed")
	assert.True(t, found, "key should be found")
	assert.Equal(t, want.StatusCode, got.StatusCode)
	assert.Equal(t, want.ContentType, got.ContentType)
	assert.Equal(t, want.Body, got.Body)
}

func TestValkeyStore_Miss(t *testing.T) {
	store := setupValkeyStore(t)
	ctx := context.Background()

	_, found, err := store.Get(ctx, "tenant-1", "nonexistent-key")
	require.NoError(t, err, "Get on missing key should not error")
	assert.False(t, found, "nonexistent key should not be found")
}

func TestValkeyStore_TenantIsolation(t *testing.T) {
	store := setupValkeyStore(t)
	ctx := context.Background()

	resp1 := idempotency.CachedResponse{StatusCode: 200, ContentType: "application/json", Body: []byte(`{"tenant":"one"}`)}
	resp2 := idempotency.CachedResponse{StatusCode: 201, ContentType: "application/json", Body: []byte(`{"tenant":"two"}`)}

	require.NoError(t, store.Set(ctx, "tenant-1", "shared-key", resp1, time.Minute))
	require.NoError(t, store.Set(ctx, "tenant-2", "shared-key", resp2, time.Minute))

	got1, found1, err := store.Get(ctx, "tenant-1", "shared-key")
	require.NoError(t, err)
	assert.True(t, found1, "tenant-1 key should be found")
	assert.Equal(t, resp1.StatusCode, got1.StatusCode)
	assert.Equal(t, resp1.Body, got1.Body)

	got2, found2, err := store.Get(ctx, "tenant-2", "shared-key")
	require.NoError(t, err)
	assert.True(t, found2, "tenant-2 key should be found")
	assert.Equal(t, resp2.StatusCode, got2.StatusCode)
	assert.Equal(t, resp2.Body, got2.Body)

	// Cross-check: tenant-1's value must not bleed into tenant-2 and vice versa.
	assert.NotEqual(t, got1.Body, got2.Body, "tenant responses must be isolated")
}

func TestValkeyStore_TTLExpiry(t *testing.T) {
	store := setupValkeyStore(t)
	ctx := context.Background()

	resp := idempotency.CachedResponse{
		StatusCode:  200,
		ContentType: "text/plain",
		Body:        []byte("ephemeral"),
	}

	require.NoError(t, store.Set(ctx, "tenant-1", "expiring-key", resp, time.Second))

	// Confirm the key exists immediately after Set.
	_, found, err := store.Get(ctx, "tenant-1", "expiring-key")
	require.NoError(t, err)
	assert.True(t, found, "key should be present before TTL expires")

	// Wait for TTL to expire.
	time.Sleep(2 * time.Second)

	_, found, err = store.Get(ctx, "tenant-1", "expiring-key")
	require.NoError(t, err, "Get after TTL should not error")
	assert.False(t, found, "key should be gone after TTL expiry")
}
