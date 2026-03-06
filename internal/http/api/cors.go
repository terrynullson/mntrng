package api

import (
	"net/http"
	"strings"

	"github.com/terrynullson/hls_mntrng/internal/config"
)

const corsAllowedOriginsKey = "CORS_ALLOWED_ORIGINS"

// corsMiddleware sets CORS headers when origin is in the allowed list.
// CORS_ALLOWED_ORIGINS is comma-separated (e.g. "https://app.example.com,http://localhost:3000").
// Empty or unset means no Access-Control-Allow-Origin is set (same-origin only).
func corsMiddleware(next http.Handler) http.Handler {
	raw := config.GetString(corsAllowedOriginsKey, "")
	var allowed map[string]bool
	if raw != "" {
		allowed = make(map[string]bool)
		for _, o := range strings.Split(raw, ",") {
			origin := strings.TrimSpace(o)
			if origin != "" {
				allowed[origin] = true
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed != nil && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
