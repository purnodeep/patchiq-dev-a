package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	store := auth.NewMemoryRateLimitStore()
	limit := int64(5)
	window := time.Minute

	handler := auth.RateLimitMiddleware(store, limit, window)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for i := 0; i < int(limit); i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	store := auth.NewMemoryRateLimitStore()
	limit := int64(3)
	window := time.Minute

	handler := auth.RateLimitMiddleware(store, limit, window)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Exhaust the limit.
	for i := 0; i < int(limit); i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}

	// Next request should be blocked.
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "RATE_LIMITED", body["code"])
	assert.Contains(t, body["message"], "too many requests")
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	store := auth.NewMemoryRateLimitStore()
	limit := int64(2)
	window := time.Minute

	handler := auth.RateLimitMiddleware(store, limit, window)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Exhaust limit for IP-A.
	for i := 0; i < int(limit); i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "1.1.1.1:1111"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// IP-A is now blocked.
	reqA := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	reqA.RemoteAddr = "1.1.1.1:1111"
	recA := httptest.NewRecorder()
	handler.ServeHTTP(recA, reqA)
	assert.Equal(t, http.StatusTooManyRequests, recA.Code, "IP-A should be blocked")

	// IP-B should still be allowed.
	reqB := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	reqB.RemoteAddr = "2.2.2.2:2222"
	recB := httptest.NewRecorder()
	handler.ServeHTTP(recB, reqB)
	assert.Equal(t, http.StatusOK, recB.Code, "IP-B should still be allowed")
}

func TestRateLimiter_UsesRemoteAddrIgnoresXFF(t *testing.T) {
	store := auth.NewMemoryRateLimitStore()
	limit := int64(1)
	window := time.Minute

	handler := auth.RateLimitMiddleware(store, limit, window)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// First request uses RemoteAddr for rate key, X-Forwarded-For is ignored.
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second request from same RemoteAddr is blocked (even with different XFF).
	req2 := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req2.RemoteAddr = "10.0.0.1:8888"
	req2.Header.Set("X-Forwarded-For", "198.51.100.99")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)

	// Different RemoteAddr is allowed (proves keying is on RemoteAddr IP).
	req3 := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req3.RemoteAddr = "10.0.0.2:9999"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)
}
