package middleware

import (
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// IPRateLimiter limits requests per client IP using a token bucket per IP.
type IPRateLimiter struct {
	ips   map[string]*rate.Limiter
	mu    sync.RWMutex
	limit rate.Limit
	burst int
}

// NewIPRateLimiter creates a per-IP rate limiter. limit is events per second (e.g. rate.Every(time.Minute) for 1/min);
// for N per minute use rate.Limit(float64(N)/60.0). burst is max tokens per bucket.
func NewIPRateLimiter(limit rate.Limit, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		ips:   make(map[string]*rate.Limiter),
		limit: limit,
		burst: burst,
	}
}

func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	lim, ok := l.ips[ip]
	l.mu.RUnlock()
	if ok {
		return lim
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	// Double-check after acquiring write lock
	if lim, ok = l.ips[ip]; ok {
		return lim
	}
	lim = rate.NewLimiter(l.limit, l.burst)
	l.ips[ip] = lim
	return lim
}

// clientIP returns the client IP from X-Forwarded-For, X-Real-IP, or RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First value is the client when behind a single proxy
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// RemoteAddr is "host:port"; use as-is for uniqueness (or strip port)
	return r.RemoteAddr
}

// Middleware returns a chi-compatible middleware that returns 429 when the client IP exceeds the rate.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		lim := l.getLimiter(ip)
		if !lim.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"too many requests"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AuthRateLimiter returns a limiter suitable for login/register: 10 requests per minute per IP, burst 5.
func AuthRateLimiter() *IPRateLimiter {
	// 10 per minute = 10/60 per second
	return NewIPRateLimiter(rate.Limit(10.0/60.0), 5)
}
