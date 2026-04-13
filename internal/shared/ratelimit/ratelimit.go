// Package ratelimit provides HTTP rate limiting middleware with an in-memory
// backend. For multi-instance deployments, swap in a Valkey-backed store.
package ratelimit

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"sync"
	"time"
)

// Store abstracts the counter backend (Valkey or in-memory).
type Store interface {
	// Increment atomically increments the counter for key within the given
	// window and returns the new count. The counter resets after window expires.
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
}

// Middleware returns HTTP middleware that limits requests per client IP.
// When the limit is exceeded it responds with 429 Too Many Requests and a JSON
// error body. The keyPrefix distinguishes different rate limit scopes
// (e.g., "auth" vs "api").
func Middleware(store Store, limit int64, window time.Duration, keyPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)
			key := "ratelimit:" + keyPrefix + ":" + ip

			count, err := store.Increment(r.Context(), key, window)
			if err != nil {
				slog.ErrorContext(r.Context(), "rate limiter store error, allowing request",
					"error", err,
					"ip", ip,
					"path", r.URL.Path,
				)
				// Fail open: allow the request if the store is unavailable.
				next.ServeHTTP(w, r)
				return
			}

			if count > limit {
				slog.WarnContext(r.Context(), "rate limit exceeded",
					"ip", ip,
					"count", count,
					"limit", limit,
					"key_prefix", keyPrefix,
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				if encErr := json.NewEncoder(w).Encode(map[string]any{
					"code":    "RATE_LIMITED",
					"message": "too many requests, please try again later",
					"details": []any{},
				}); encErr != nil {
					slog.ErrorContext(r.Context(), "failed to write rate limit response", "error", encErr)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP extracts the client IP from r.RemoteAddr. This assumes chi's
// middleware.RealIP (or equivalent) has already set RemoteAddr from trusted
// proxy headers — do NOT re-parse X-Forwarded-For here to avoid spoofing.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// MemoryStore is an in-memory Store suitable for single-instance deployments.
// It uses probabilistic cleanup: on every Increment call there is a 1-in-100
// chance of sweeping expired entries. This avoids the need for a background
// goroutine while bounding memory growth.
type MemoryStore struct {
	mu      sync.Mutex
	buckets map[string]*bucket

	// cleanupProbability is the 1-in-N chance of cleanup per Increment call.
	// Defaults to 100. Exposed for testing.
	cleanupProbability int
}

type bucket struct {
	count   int64
	expires time.Time
}

// NewMemoryStore returns a new in-memory rate limit store with probabilistic
// cleanup (1-in-100 chance per Increment call).
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		buckets:            make(map[string]*bucket),
		cleanupProbability: 100,
	}
}

// Increment atomically increments the counter for key. If the window has
// expired the counter resets to 1. Probabilistically cleans expired entries.
func (s *MemoryStore) Increment(_ context.Context, key string, window time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Probabilistic cleanup: 1-in-N chance.
	if s.cleanupProbability > 0 && rand.IntN(s.cleanupProbability) == 0 {
		s.sweepExpiredLocked(now)
	}

	b, ok := s.buckets[key]
	if !ok || now.After(b.expires) {
		s.buckets[key] = &bucket{count: 1, expires: now.Add(window)}
		return 1, nil
	}

	b.count++
	return b.count, nil
}

// Len returns the number of buckets currently tracked. Useful for testing.
func (s *MemoryStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.buckets)
}

// SetCleanupProbability sets the 1-in-N cleanup probability. Use 1 to clean
// on every call (useful for tests). Use 0 to disable probabilistic cleanup.
func (s *MemoryStore) SetCleanupProbability(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupProbability = n
}

// sweepExpiredLocked removes all expired buckets. Caller must hold s.mu.
func (s *MemoryStore) sweepExpiredLocked(now time.Time) {
	for key, b := range s.buckets {
		if now.After(b.expires) {
			delete(s.buckets, key)
		}
	}
}
