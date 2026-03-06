package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/terrynullson/mntrng/internal/bootstrap/apiapp"
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

	app, err := apiapp.NewRuntimeApp(runtimeConfig)
	if err != nil {
		log.Fatalf("api bootstrap failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("api skeleton listening on :%s", runtimeConfig.Port)
		if err := app.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("api server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("api received shutdown signal, draining connections (timeout %s)", apiShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), apiShutdownTimeout)
	defer cancel()
	if err := app.Shutdown(shutdownCtx); err != nil {
		log.Printf("api shutdown: %v", err)
	} else {
		log.Printf("api shutdown complete")
	}
}
