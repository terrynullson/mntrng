package apiapp

import (
	"database/sql"
	"net/http"

	httpapi "github.com/terrynullson/mntrng/internal/http/api"
	"github.com/terrynullson/mntrng/internal/ratelimit"
)

func NewHTTPServer(addr string, db *sql.DB, limiter ratelimit.Limiter, runtime RuntimeConfig) *http.Server {
	adapters := buildAdapters(db, runtime)
	ports := buildPorts(adapters, runtime)
	server := httpapi.NewServer(db, ports, httpapi.AuthTTLConfig{
		AccessTTL:  runtime.AuthAccessTTL,
		RefreshTTL: runtime.AuthRefreshTTL,
	})
	return httpapi.NewHTTPServer(addr, server, limiter)
}
