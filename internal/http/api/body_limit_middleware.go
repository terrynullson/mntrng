package api

import (
	"net/http"

	"github.com/example/hls-monitoring-platform/internal/config"
)

func requestBodyLimitMiddleware(next http.Handler) http.Handler {
	maxBodyBytes := int64(config.IntAtLeast(config.GetInt("API_MAX_BODY_BYTES", 1<<20), 1024))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !methodAllowsBody(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if r.ContentLength > maxBodyBytes {
			WriteJSONError(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "request body is too large", map[string]interface{}{
				"max_bytes": maxBodyBytes,
			})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func methodAllowsBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
