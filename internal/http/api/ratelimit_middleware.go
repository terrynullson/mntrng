package api

import (
	"net"
	"net/http"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/example/hls-monitoring-platform/internal/ratelimit"
)

// authRateLimitPaths are the path prefixes that get rate-limited by IP.
var authRateLimitPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
	"/api/v1/auth/telegram/login",
}

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
	if r == nil {
		return ""
	}
	trustProxyHeaders := config.GetBool("TRUST_PROXY_HEADERS", false)
	if trustProxyHeaders {
		if x := r.Header.Get("X-Forwarded-For"); x != "" {
			if i := strings.Index(x, ","); i > 0 {
				return normalizeClientIP(strings.TrimSpace(x[:i]))
			}
			return normalizeClientIP(strings.TrimSpace(x))
		}
		if x := r.Header.Get("X-Real-IP"); x != "" {
			return normalizeClientIP(strings.TrimSpace(x))
		}
	}
	return normalizeClientIP(r.RemoteAddr)
}

func normalizeClientIP(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(value, "[]")
}
