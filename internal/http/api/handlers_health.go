package api

import (
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
