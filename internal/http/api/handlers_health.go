package api

import (
	"context"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	response := healthResponse{
		Status:  "ok",
		Service: "api",
		Time:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := WriteJSON(w, http.StatusOK, response); err != nil {
		log.Printf("health response encode error: %v", err)
	}
}

type readyResponse struct {
	Ready bool `json:"ready"`
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowed(w, r, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		if err := WriteJSON(w, http.StatusServiceUnavailable, readyResponse{Ready: false}); err != nil {
			log.Printf("ready response encode error: %v", err)
		}
		return
	}
	if err := WriteJSON(w, http.StatusOK, readyResponse{Ready: true}); err != nil {
		log.Printf("ready response encode error: %v", err)
	}
}
