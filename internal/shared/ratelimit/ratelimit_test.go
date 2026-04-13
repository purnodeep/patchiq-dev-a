package ratelimit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_AllowsUnderLimit(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	limit := int64(5)
	window := time.Minute

	handler := ratelimit.Middleware(store, limit, window, "test")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for i := 0; i < int(limit); i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}
}

func TestMiddleware_BlocksOverLimit(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	limit := int64(3)
	window := time.Minute

	handler := ratelimit.Middleware(store, limit, window, "api")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for i := 0; i < int(limit); i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "RATE_LIMITED", body["code"])
}

func TestMiddleware_KeyPrefixIsolation(t *testing.T) {
	store := ratelimit.NewMemoryStore()

	authHandler := ratelimit.Middleware(store, 1, time.Minute, "auth")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	)
	apiHandler := ratelimit.Middleware(store, 1, time.Minute, "api")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	)

	// Exhaust auth limit.
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "1.1.1.1:1111"
	rec := httptest.NewRecorder()
	authHandler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Auth should be blocked.
	req2 := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req2.RemoteAddr = "1.1.1.1:1111"
	rec2 := httptest.NewRecorder()
	authHandler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)

	// API should still work (different prefix).
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints", nil)
	req3.RemoteAddr = "1.1.1.1:1111"
	rec3 := httptest.NewRecorder()
	apiHandler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)
}

func TestMemoryStore_CleanupRemovesExpiredBuckets(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	// Force cleanup on every call.
	store.SetCleanupProbability(1)

	ctx := context.Background()

	// Create entries with a very short window.
	_, err := store.Increment(ctx, "key1", 1*time.Millisecond)
	require.NoError(t, err)
	_, err = store.Increment(ctx, "key2", 1*time.Millisecond)
	require.NoError(t, err)

	assert.Equal(t, 2, store.Len())

	// Wait for expiry.
	time.Sleep(5 * time.Millisecond)

	// Next increment triggers cleanup (probability=1 means every call).
	_, err = store.Increment(ctx, "key3", time.Minute)
	require.NoError(t, err)

	// key1 and key2 should be cleaned up, only key3 remains.
	assert.Equal(t, 1, store.Len())
}

func TestMemoryStore_CleanupDoesNotRemoveActiveBuckets(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	store.SetCleanupProbability(1)

	ctx := context.Background()

	_, err := store.Increment(ctx, "active", time.Hour)
	require.NoError(t, err)

	// Trigger cleanup via another increment.
	_, err = store.Increment(ctx, "another", time.Hour)
	require.NoError(t, err)

	// Both should still exist.
	assert.Equal(t, 2, store.Len())
}

func TestMemoryStore_ExpiredBucketResetsCounter(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	ctx := context.Background()

	count, err := store.Increment(ctx, "key", 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = store.Increment(ctx, "key", 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	time.Sleep(5 * time.Millisecond)

	// After expiry, counter resets.
	count, err = store.Increment(ctx, "key", 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
