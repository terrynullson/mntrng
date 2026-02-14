package api

import (
	"database/sql"
	"net/http"
	"time"
)

func NewHTTPServer(addr string, db *sql.DB) *http.Server {
	server := NewServer(db)
	router := NewRouter(server.RouterHandlers())

	return &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
