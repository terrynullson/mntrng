package api

import (
	"context"
	"log"
	"net/http"
	"time"

	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

func (s *Server) handleCreateStream(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	var request createStreamRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.streamService.CreateStream(ctx, serviceapi.CreateStreamInput{
		CompanyID: companyID,
		ProjectID: projectID,
		Name:      request.Name,
		URL:       request.URL,
		IsActive:  request.IsActive,
	})
	if err != nil {
		writeServiceError(w, r, "create stream", err)
		return
	}

	if err := WriteJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create stream response encode error: %v", err)
	}
}

func (s *Server) handleListStreams(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := s.streamService.ListStreams(ctx, serviceapi.ListStreamsInput{
		CompanyID:    companyID,
		ProjectIDRaw: r.URL.Query().Get("project_id"),
		IsActiveRaw:  r.URL.Query().Get("is_active"),
	})
	if err != nil {
		writeServiceError(w, r, "list streams", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, streamListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list streams response encode error: %v", err)
	}
}

func (s *Server) handleGetStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.streamService.GetStream(ctx, companyID, streamID)
	if err != nil {
		writeServiceError(w, r, "get stream", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get stream response encode error: %v", err)
	}
}
