package api

import (
	"net/http"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/ratelimit"
)

// authRateLimitPaths are the path prefixes that get rate-limited by IP.
var authRateLimitPaths = []string{"/api/v1/auth/login", "/api/v1/auth/register"}

func rateLimitMiddleware(limiter ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			path := r.URL.Path
			var needLimit bool
			for _, p := range authRateLimitPaths {
				if path == p {
					needLimit = true
					break
				}
			}
			if !needLimit {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIP(r)
			allowed, err := limiter.Allow(r.Context(), ip)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				WriteJSONError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests, try again later", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		if i := strings.Index(x, ","); i > 0 {
			return strings.TrimSpace(x[:i])
		}
		return strings.TrimSpace(x)
	}
	if x := r.Header.Get("X-Real-IP"); x != "" {
		return strings.TrimSpace(x)
	}
	return r.RemoteAddr
}
