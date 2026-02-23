package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	httpapi "github.com/example/hls-monitoring-platform/internal/http/api"
	"github.com/example/hls-monitoring-platform/internal/ratelimit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

const apiShutdownTimeout = 15 * time.Second

func main() {
	prometheus.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{Namespace: "hls_api"}),
	)

	port := config.GetString("API_PORT", "8080")
	databaseURL := config.GetString("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	authPerMin := config.IntAtLeast(config.GetInt("RATE_LIMIT_AUTH_PER_MIN", 10), 1)
	var limiter ratelimit.Limiter
	redisAddr := config.GetString("REDIS_ADDR", "")
	if redisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
		pingCtx2, pingCancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		defer pingCancel2()
		if err := rdb.Ping(pingCtx2).Err(); err != nil {
			log.Printf("redis ping failed, using in-memory rate limiter: %v", err)
			limiter = ratelimit.NewInMemLimiter(authPerMin)
		} else {
			limiter = ratelimit.NewRedisLimiter(rdb, authPerMin)
		}
	} else {
		limiter = ratelimit.NewInMemLimiter(authPerMin)
	}

	server := httpapi.NewHTTPServer(":"+port, db, limiter)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("api skeleton listening on :%s", port)
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
