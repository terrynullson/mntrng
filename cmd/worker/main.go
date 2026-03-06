package main

import (
	"context"
	"database/sql"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/terrynullson/mntrng/internal/bootstrap/workerapp"
	"github.com/terrynullson/mntrng/internal/config"
	_ "github.com/lib/pq"
)

func main() {
	runtimeConfig := workerapp.LoadRuntimeConfig()
	if err := runtimeConfig.Validate(); err != nil {
		log.Fatal(err)
	}
	workerapp.LogRuntimeConfig(runtimeConfig)

	databaseURL := config.GetString("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()
	workerapp.ConfigureDBPool(db)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	workerapp.StartMetricsServer(ctx, runtimeConfig.MetricsPort, runtimeConfig.MetricsToken)

	app := workerapp.NewApp(db, runtimeConfig)
	app.Run(ctx)
}
