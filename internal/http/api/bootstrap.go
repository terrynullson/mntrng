package api

import (
	"database/sql"
	"net/http"
	"time"
)

func NewHTTPServer(addr string, db *sql.DB) *http.Server {
	server := NewServer(db)
	router := NewRouter(server.RouterHandlers())
	handler := securityHeaders(router)

	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
