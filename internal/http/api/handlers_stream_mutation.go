package api

import (
	"context"
	"log"
	"net/http"
	"time"

	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

func (s *Server) handlePatchStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	var request patchStreamRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.streamService.PatchStream(ctx, serviceapi.PatchStreamRequest{
		CompanyID:  companyID,
		StreamID:   streamID,
		Name:       request.Name,
		SourceType: request.SourceType,
		SourceURL:  request.SourceURL,
		URL:        request.URL,
		IsActive:   request.IsActive,
	})
	if err != nil {
		writeServiceError(w, r, "patch stream", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch stream response encode error: %v", err)
	}
}

func (s *Server) handleDeleteStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.streamService.DeleteStream(ctx, companyID, streamID); err != nil {
		writeServiceError(w, r, "delete stream", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
