package idempotency_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skenzeriq/patchiq/internal/shared/idempotency"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

const testTenantID = "tenant-abc"

// newJSONHandler returns a handler that increments counter, writes a JSON body, and responds with statusCode.
func newJSONHandler(counter *atomic.Int64, statusCode int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	})
}

// withTenant wraps a request with a tenant context.
func withTenant(r *http.Request, tenantID string) *http.Request {
	return r.WithContext(tenant.WithTenantID(r.Context(), tenantID))
}

// errorStore is a Store implementation that always returns errors, used to test
// graceful degradation when the cache backend is unavailable.
type errorStore struct{}

func (e *errorStore) Get(_ context.Context, _, _ string) (idempotency.CachedResponse, bool, error) {
	return idempotency.CachedResponse{}, false, errors.New("connection refused")
}

func (e *errorStore) Set(_ context.Context, _, _ string, _ idempotency.CachedResponse, _ time.Duration) error {
	return errors.New("connection refused")
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, store *idempotency.MemoryStore)
	}{
		{
			name: "GET request passes through",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusOK))

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req = withTenant(req, testTenantID)
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)

				assert.Equal(t, int64(1), counter.Load(), "handler should be called for GET")
				assert.Equal(t, http.StatusOK, rr.Code)
			},
		},
		{
			name: "HEAD request passes through",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusOK))

				req := httptest.NewRequest(http.MethodHead, "/", nil)
				req = withTenant(req, testTenantID)
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)

				assert.Equal(t, int64(1), counter.Load(), "handler should be called for HEAD")
			},
		},
		{
			name: "OPTIONS request passes through",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusOK))

				req := httptest.NewRequest(http.MethodOptions, "/", nil)
				req = withTenant(req, testTenantID)
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)

				assert.Equal(t, int64(1), counter.Load(), "handler should be called for OPTIONS")
			},
		},
		{
			name: "POST without Idempotency-Key header passes through",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusCreated))

				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req = withTenant(req, testTenantID)
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)

				assert.Equal(t, int64(1), counter.Load(), "handler should be called without idempotency key")
				assert.Equal(t, http.StatusCreated, rr.Code)
			},
		},
		{
			name: "POST with key first request calls handler and returns response",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusCreated))

				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req.Header.Set(idempotency.HeaderIdempotencyKey, "key-first-001")
				req = withTenant(req, testTenantID)
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)

				assert.Equal(t, int64(1), counter.Load())
				assert.Equal(t, http.StatusCreated, rr.Code)
				assert.Equal(t, `{"id":"123"}`, rr.Body.String())
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "POST with same key second request returns cached response without calling handler",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusCreated))

				// First request
				req1 := httptest.NewRequest(http.MethodPost, "/", nil)
				req1.Header.Set(idempotency.HeaderIdempotencyKey, "key-dup-001")
				req1 = withTenant(req1, testTenantID)
				rr1 := httptest.NewRecorder()
				handler.ServeHTTP(rr1, req1)

				require.Equal(t, int64(1), counter.Load(), "handler should be called on first request")

				// Second request with same key
				req2 := httptest.NewRequest(http.MethodPost, "/", nil)
				req2.Header.Set(idempotency.HeaderIdempotencyKey, "key-dup-001")
				req2 = withTenant(req2, testTenantID)
				rr2 := httptest.NewRecorder()
				handler.ServeHTTP(rr2, req2)

				assert.Equal(t, int64(1), counter.Load(), "handler must NOT be called on second request")
				assert.Equal(t, http.StatusCreated, rr2.Code)
				assert.Equal(t, `{"id":"123"}`, rr2.Body.String())
				assert.Equal(t, "application/json", rr2.Header().Get("Content-Type"))
			},
		},
		{
			name: "POST with different keys calls handler independently",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusCreated))

				for _, key := range []string{"key-a", "key-b"} {
					req := httptest.NewRequest(http.MethodPost, "/", nil)
					req.Header.Set(idempotency.HeaderIdempotencyKey, key)
					req = withTenant(req, testTenantID)
					rr := httptest.NewRecorder()
					handler.ServeHTTP(rr, req)
				}

				assert.Equal(t, int64(2), counter.Load(), "handler should be called once per unique key")
			},
		},
		{
			name: "DELETE with key is processed by middleware",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusOK))

				// First DELETE
				req1 := httptest.NewRequest(http.MethodDelete, "/resource/1", nil)
				req1.Header.Set(idempotency.HeaderIdempotencyKey, "del-key-001")
				req1 = withTenant(req1, testTenantID)
				rr1 := httptest.NewRecorder()
				handler.ServeHTTP(rr1, req1)

				require.Equal(t, int64(1), counter.Load())

				// Second DELETE same key — should be cached
				req2 := httptest.NewRequest(http.MethodDelete, "/resource/1", nil)
				req2.Header.Set(idempotency.HeaderIdempotencyKey, "del-key-001")
				req2 = withTenant(req2, testTenantID)
				rr2 := httptest.NewRecorder()
				handler.ServeHTTP(rr2, req2)

				assert.Equal(t, int64(1), counter.Load(), "DELETE handler must not be called again for same key")
				assert.Equal(t, http.StatusOK, rr2.Code)
			},
		},
		{
			name: "Non-2xx response is not cached, retry executes handler again",
			run: func(t *testing.T, store *idempotency.MemoryStore) {
				var counter atomic.Int64
				handler := idempotency.Middleware(store)(newJSONHandler(&counter, http.StatusInternalServerError))

				// First request — 500 response
				req1 := httptest.NewRequest(http.MethodPost, "/", nil)
				req1.Header.Set(idempotency.HeaderIdempotencyKey, "key-err-001")
				req1 = withTenant(req1, testTenantID)
				rr1 := httptest.NewRecorder()
				handler.ServeHTTP(rr1, req1)

				require.Equal(t, int64(1), counter.Load())
				require.Equal(t, http.StatusInternalServerError, rr1.Code)

				// Second request with same key — handler must be called again since 500 was not cached
				req2 := httptest.NewRequest(http.MethodPost, "/", nil)
				req2.Header.Set(idempotency.HeaderIdempotencyKey, "key-err-001")
				req2 = withTenant(req2, testTenantID)
				rr2 := httptest.NewRecorder()
				handler.ServeHTTP(rr2, req2)

				assert.Equal(t, int64(2), counter.Load(), "handler must be called again after non-2xx response")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := idempotency.NewMemoryStore()
			tc.run(t, store)
		})
	}
}

// TestMiddleware_StoreErrors verifies graceful degradation when the cache backend is unavailable.
// When store.Get fails the middleware passes the request through. When store.Set fails after a
// successful handler call the response is still returned to the client (fail-open on cache write).
func TestMiddleware_StoreErrors(t *testing.T) {
	var counter atomic.Int64
	handler := idempotency.Middleware(&errorStore{})(newJSONHandler(&counter, http.StatusCreated))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(idempotency.HeaderIdempotencyKey, "key-store-err-001")
	req = withTenant(req, testTenantID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Handler must be called and response returned correctly despite store errors.
	assert.Equal(t, int64(1), counter.Load(), "handler should be called when store.Get fails")
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, `{"id":"123"}`, rr.Body.String())
}

// TestMiddleware_NoTenantID verifies that the middleware returns 500 when no tenant ID is
// present in the context. This guards against middleware ordering bugs.
func TestMiddleware_NoTenantID(t *testing.T) {
	var counter atomic.Int64
	handler := idempotency.Middleware(idempotency.NewMemoryStore())(newJSONHandler(&counter, http.StatusCreated))

	// Request without tenant context.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(idempotency.HeaderIdempotencyKey, "key-no-tenant")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, int64(0), counter.Load(), "handler must not be called when tenant ID is missing")
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
