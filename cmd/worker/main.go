package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/terrynullson/mntrng/internal/bootstrap/workerapp"
)

func main() {
	runtimeConfig := workerapp.LoadRuntimeConfig()
	if err := runtimeConfig.Validate(); err != nil {
		log.Fatal(err)
	}
	workerapp.LogRuntimeConfig(runtimeConfig)

	app, err := workerapp.NewRuntimeApp(runtimeConfig)
	if err != nil {
		log.Fatalf("worker bootstrap failed: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("worker db close: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	app.Run(ctx)
}
