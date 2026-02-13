package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
)

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

func main() {
	port := config.GetString("API_PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := healthResponse{
			Status: "ok",
			Time:   time.Now().UTC().Format(time.RFC3339),
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("healthz response encode error: %v", err)
		}
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("api skeleton listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("api server failed: %v", err)
	}
}
