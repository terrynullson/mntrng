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
	Service string `json:"service"`
	Time   string `json:"time"`
}

type errorEnvelope struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details"`
	RequestID string      `json:"request_id"`
}

func main() {
	port := config.GetString("API_PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSONError(
				w,
				r,
				http.StatusMethodNotAllowed,
				"method_not_allowed",
				"method is not allowed for this endpoint",
				map[string]interface{}{
					"method":          r.Method,
					"allowed_methods": []string{http.MethodGet},
				},
			)
			return
		}

		response := healthResponse{
			Status:  "ok",
			Service: "api",
			Time:    time.Now().UTC().Format(time.RFC3339),
		}
		if err := writeJSON(w, http.StatusOK, response); err != nil {
			log.Printf("health response encode error: %v", err)
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

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(payload)
}

func writeJSONError(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	code string,
	message string,
	details interface{},
) {
	err := writeJSON(w, statusCode, errorEnvelope{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: requestIDFromRequest(r),
	})
	if err != nil {
		log.Printf("error response encode failed: %v", err)
	}
}

func requestIDFromRequest(r *http.Request) string {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	return "req_" + time.Now().UTC().Format("20060102150405.000000000")
}
