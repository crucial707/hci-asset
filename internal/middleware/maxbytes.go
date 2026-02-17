package middleware

import (
	"net/http"
)

// DefaultMaxBodyBytes is the default maximum request body size (1 MiB).
const DefaultMaxBodyBytes = 1 << 20

// MaxBytes limits the request body size. If the body exceeds maxBytes, the client
// receives 413 Request Entity Too Large. Apply to routes that accept a body (POST, PUT, PATCH).
func MaxBytes(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
