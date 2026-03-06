package apiapp

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/terrynullson/mntrng/internal/bootstrap/infra"
)

type RuntimeApp struct {
	server *http.Server
	db     *sql.DB
}

func NewRuntimeApp(runtime RuntimeConfig) (*RuntimeApp, error) {
	db, err := infra.OpenPostgres(runtime.DatabaseURL)
	if err != nil {
		return nil, err
	}

	infra.ConfigureDBPoolFromEnv(db, infra.DBPoolDefaults{
		MaxOpenConns:       30,
		MaxIdleConns:       10,
		ConnMaxLifetimeMin: 30,
		ConnMaxIdleTimeMin: 10,
	})
	if err := infra.PingDB(context.Background(), db, runtime.DBPingTimeout); err != nil {
		_ = db.Close()
		return nil, err
	}

	limiter := infra.NewAuthRateLimiter(runtime.RedisAddr, runtime.AuthRateLimitPerMin, runtime.RedisPingTimeout)
	if err := infra.InitTelemetry(context.Background()); err != nil {
		log.Printf("telemetry init (optional): %v", err)
	}

	server := NewHTTPServer(":"+runtime.Port, db, limiter, runtime)
	return &RuntimeApp{
		server: server,
		db:     db,
	}, nil
}

func (a *RuntimeApp) ListenAndServe() error {
	return a.server.ListenAndServe()
}

func (a *RuntimeApp) Shutdown(ctx context.Context) error {
	shutdownErr := a.server.Shutdown(ctx)
	if closeErr := a.db.Close(); shutdownErr == nil {
		shutdownErr = closeErr
	}

	telemetryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = infra.ShutdownTelemetry(telemetryCtx)
	return shutdownErr
}
