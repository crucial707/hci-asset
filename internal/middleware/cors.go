package middleware

import (
	"net/http"
	"strings"
)

// DefaultCORSAllowedMethods is the default set of methods allowed for CORS.
var DefaultCORSAllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

// DefaultCORSAllowedHeaders is the default set of request headers allowed for CORS.
var DefaultCORSAllowedHeaders = []string{"Accept", "Authorization", "Content-Type"}

// CORS returns a middleware that sets CORS response headers and handles OPTIONS preflight
// when origins is non-nil. When origins is nil or empty, the middleware is a no-op.
func CORS(origins []string) func(http.Handler) http.Handler {
	if len(origins) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	originSet := make(map[string]bool)
	for _, o := range origins {
		originSet[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && originSet[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(DefaultCORSAllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(DefaultCORSAllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
