package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/terrynullson/mntrng/internal/bootstrap/apiapp"
	"github.com/terrynullson/mntrng/internal/bootstrap/infra"
	"github.com/terrynullson/mntrng/internal/config"
)

const apiShutdownTimeout = 15 * time.Second

func main() {
	runtimeConfig := apiapp.LoadRuntimeConfig()
	if err := runtimeConfig.Validate(); err != nil {
		log.Fatal(err)
	}

	if err := config.ValidateAPIRuntimeSafety(); err != nil {
		log.Fatalf("api startup config validation failed: %v", err)
	}

	db, err := infra.OpenPostgres(runtimeConfig.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()
	infra.ConfigureDBPoolFromEnv(db, infra.DBPoolDefaults{
		MaxOpenConns:       30,
		MaxIdleConns:       10,
		ConnMaxLifetimeMin: 30,
		ConnMaxIdleTimeMin: 10,
	})
	if err := infra.PingDB(context.Background(), db, runtimeConfig.DBPingTimeout); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	limiter := infra.NewAuthRateLimiter(runtimeConfig.RedisAddr, runtimeConfig.AuthRateLimitPerMin, runtimeConfig.RedisPingTimeout)

	if err := infra.InitTelemetry(context.Background()); err != nil {
		log.Printf("telemetry init (optional): %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = infra.ShutdownTelemetry(shutdownCtx)
	}()

	server := apiapp.NewHTTPServer(":"+runtimeConfig.Port, db, limiter, runtimeConfig)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("api skeleton listening on :%s", runtimeConfig.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("api server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("api received shutdown signal, draining connections (timeout %s)", apiShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), apiShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("api shutdown: %v", err)
	} else {
		log.Printf("api shutdown complete")
	}
}
