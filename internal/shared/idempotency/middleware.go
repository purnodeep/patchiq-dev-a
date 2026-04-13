package idempotency

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

const (
	HeaderIdempotencyKey = "Idempotency-Key"
	DefaultTTL           = 24 * time.Hour
)

// Middleware returns an HTTP middleware that enforces idempotency for non-safe methods.
// Responses with 2xx status codes are cached under (tenantID, idempotency-key).
// Subsequent requests with the same key replay the cached response without calling the handler.
//
// NOTE(PIQ-10): This middleware does not provide in-flight request deduplication.
// Two concurrent requests with the same key may both execute. Database-level
// constraints (optimistic locking) provide the final safety net. In-flight
// locking can be added in M1 if needed.
func Middleware(store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Step 1: skip safe methods.
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			// Step 2: skip if no idempotency key header.
			key := r.Header.Get(HeaderIdempotencyKey)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// Step 3: require tenant ID.
			tenantID, ok := tenant.TenantIDFromContext(ctx)
			if !ok {
				// This should never happen: tenant middleware runs before idempotency
				// middleware in the chi middleware chain. Fail closed rather than
				// silently degrading the exactly-once guarantee.
				slog.ErrorContext(ctx, "idempotency middleware: no tenant ID in context, middleware ordering defect",
					"idempotency_key", key,
				)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}

			// Step 4: check cache.
			cached, found, err := store.Get(ctx, tenantID, key)
			if err != nil {
				slog.ErrorContext(ctx, "idempotency middleware: store get error, passing through",
					"idempotency_key", key,
					"tenant_id", tenantID,
					"error", err,
				)
				next.ServeHTTP(w, r)
				return
			}

			if found {
				slog.InfoContext(ctx, "idempotency middleware: replaying cached response",
					"idempotency_key", key,
					"tenant_id", tenantID,
					"cached_status", cached.StatusCode,
				)
				w.Header().Set("Content-Type", cached.ContentType)
				w.WriteHeader(cached.StatusCode)
				_, _ = w.Write(cached.Body)
				return
			}

			// Step 5: cache miss — wrap the ResponseWriter and call handler.
			rec := &responseRecorder{
				ResponseWriter: w,
				statusCode:     0, // resolved on first WriteHeader or after ServeHTTP
				body:           &bytes.Buffer{},
			}
			next.ServeHTTP(rec, r)

			// Default to 200 if WriteHeader was never called (Go's implicit default).
			status := rec.statusCode
			if status == 0 {
				status = http.StatusOK
			}

			// Step 6: only cache 2xx responses.
			if status >= 200 && status < 300 {
				resp := CachedResponse{
					StatusCode:  status,
					ContentType: w.Header().Get("Content-Type"),
					Body:        rec.body.Bytes(),
				}
				// NOTE: The response has already been written to the client at this
				// point (responseRecorder forwards writes in real time). We cannot
				// change the status code. The operation completed but won't be
				// deduplicated on retries until the cache recovers.
				if setErr := store.Set(ctx, tenantID, key, resp, DefaultTTL); setErr != nil {
					slog.ErrorContext(ctx, "idempotency middleware: store set error",
						"idempotency_key", key,
						"tenant_id", tenantID,
						"error", setErr,
					)
				}
			}
		})
	}
}

// responseRecorder wraps an http.ResponseWriter to capture the status code and body.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// WriteHeader captures the status code and forwards it to the underlying writer.
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write forwards bytes to the underlying writer first, then captures only the
// bytes that were successfully written. bytes.Buffer.Write never errors.
func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	if n > 0 {
		_, _ = r.body.Write(b[:n]) // bytes.Buffer.Write never errors
	}
	return n, err
}
