package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/config"
	"github.com/terrynullson/hls_mntrng/internal/ratelimit"
	"github.com/terrynullson/hls_mntrng/internal/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewHTTPServer(addr string, db *sql.DB, limiter ratelimit.Limiter) *http.Server {
	server := NewServer(db)
	router := NewRouter(server.RouterHandlers())
	handler := securityHeaders(router)
	handler = corsMiddleware(handler)
	handler = requestBodyLimitMiddleware(handler)
	if limiter != nil {
		handler = rateLimitMiddleware(limiter)(handler)
	}
	handler = recoveryMiddleware(handler)
	handler = withHTTPObservability(handler)
	if telemetry.Enabled() {
		handler = otelhttp.NewHandler(handler, "api")
	}

	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: durationFromEnv("API_READ_HEADER_TIMEOUT_SEC", 5),
		ReadTimeout:       durationFromEnv("API_READ_TIMEOUT_SEC", 15),
		WriteTimeout:      durationFromEnv("API_WRITE_TIMEOUT_SEC", 30),
		IdleTimeout:       durationFromEnv("API_IDLE_TIMEOUT_SEC", 60),
		MaxHeaderBytes:    config.IntAtLeast(config.GetInt("API_MAX_HEADER_BYTES", 1<<20), 1024),
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		if config.GetBool("API_HSTS_ENABLED", false) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func durationFromEnv(key string, fallbackSec int) time.Duration {
	seconds := config.IntAtLeast(config.GetInt(key, fallbackSec), 1)
	return time.Duration(seconds) * time.Second
}
