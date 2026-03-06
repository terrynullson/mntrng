package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/terrynullson/mntrng/internal/bootstrap/infra"
	"github.com/terrynullson/mntrng/internal/bootstrap/workerapp"
)

func main() {
	runtimeConfig := workerapp.LoadRuntimeConfig()
	if err := runtimeConfig.Validate(); err != nil {
		log.Fatal(err)
	}
	workerapp.LogRuntimeConfig(runtimeConfig)

	db, err := infra.OpenPostgres(runtimeConfig.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()
	infra.ConfigureDBPoolFromEnv(db, infra.DBPoolDefaults{
		MaxOpenConns:       20,
		MaxIdleConns:       10,
		ConnMaxLifetimeMin: 30,
		ConnMaxIdleTimeMin: 10,
	})
	if err := infra.PingDB(context.Background(), db, runtimeConfig.DBPingTimeout); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	workerapp.StartMetricsServer(ctx, runtimeConfig.MetricsPort, runtimeConfig.MetricsToken)

	app := workerapp.NewApp(db, runtimeConfig)
	app.Run(ctx)
}
