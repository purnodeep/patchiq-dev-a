package auth

import (
	"net/http"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/ratelimit"
)

// RateLimitStore is kept as a type alias so existing call sites compile.
type RateLimitStore = ratelimit.Store

// MemoryRateLimitStore is kept as a type alias so existing call sites compile.
type MemoryRateLimitStore = ratelimit.MemoryStore

// NewMemoryRateLimitStore returns a new in-memory rate limit store.
func NewMemoryRateLimitStore() *MemoryRateLimitStore {
	return ratelimit.NewMemoryStore()
}

// RateLimitMiddleware returns HTTP middleware that limits requests per client IP
// using the "auth" key prefix. This is the original auth-specific rate limiter.
func RateLimitMiddleware(store RateLimitStore, limit int64, window time.Duration) func(http.Handler) http.Handler {
	return ratelimit.Middleware(store, limit, window, "auth")
}
