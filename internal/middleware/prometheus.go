package middleware

import (
	"net/http"
	"time"

	"github.com/crucial707/hci-asset/internal/metrics"
)

// Prometheus records request duration and count for each request.
// Wrap the handler chain with this after recovery and request ID so metrics reflect the actual request.
func Prometheus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		statusW := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(statusW, r)
		if r.URL.Path == "/metrics" {
			return
		}
		duration := time.Since(start).Seconds()
		path := r.URL.Path
		if path == "" {
			path = "/"
		}
		metrics.RecordRequest(r.Method, path, statusW.status, duration)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
