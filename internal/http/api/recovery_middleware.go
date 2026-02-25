package api

import (
	"log"
	"net/http"
)

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered: path=%s method=%s panic=%v", r.URL.Path, r.Method, recovered)
				WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
