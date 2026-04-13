package bodylimit

import (
	"net/http"
)

// DefaultMaxBodySize is the default maximum request body size (10MB).
const DefaultMaxBodySize = 10 * 1024 * 1024 // 10MB

// Middleware returns a chi-compatible middleware that limits request body size.
// It wraps the request body with http.MaxBytesReader so that downstream handlers
// receive an error (and the server returns 413) if the payload exceeds maxBytes.
// Safe methods (GET, HEAD, OPTIONS) are skipped.
func Middleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
