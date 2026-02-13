package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
)

func main() {
	heartbeatSeconds := config.GetInt("WORKER_HEARTBEAT_SEC", 15)
	heartbeatInterval := time.Duration(heartbeatSeconds) * time.Second

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("worker skeleton started, heartbeat interval=%s", heartbeatInterval)

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("worker skeleton stopped")
			os.Exit(0)
		case currentTime := <-ticker.C:
			log.Printf("worker skeleton heartbeat: %s", currentTime.UTC().Format(time.RFC3339))
		}
	}
}
