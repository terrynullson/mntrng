package api

import (
	"context"
	"log"
	"net/http"
	"time"

	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

func (s *Server) handleEnqueueCheckJob(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	var request enqueueCheckJobRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.checkJobService.EnqueueCheckJob(ctx, serviceapi.EnqueueCheckJobInput{
		CompanyID:    companyID,
		StreamID:     streamID,
		PlannedAtRaw: request.PlannedAt,
	})
	if err != nil {
		writeServiceError(w, r, "enqueue check job", err)
		return
	}

	if err := WriteJSON(w, http.StatusAccepted, enqueueCheckJobResponse{Job: item}); err != nil {
		log.Printf("enqueue check job response encode error: %v", err)
	}
}

func (s *Server) handleTriggerStreamCheck(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.checkJobService.EnqueueCheckJob(ctx, serviceapi.EnqueueCheckJobInput{
		CompanyID:    companyID,
		StreamID:     streamID,
		PlannedAtRaw: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		writeServiceError(w, r, "enqueue check job", err)
		return
	}

	if err := WriteJSON(w, http.StatusAccepted, enqueueCheckJobResponse{Job: item}); err != nil {
		log.Printf("enqueue check job response encode error: %v", err)
	}
}

func (s *Server) handleGetCheckJob(w http.ResponseWriter, r *http.Request, companyID int64, jobID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.checkJobService.GetCheckJob(ctx, companyID, jobID)
	if err != nil {
		writeServiceError(w, r, "get check job", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check job response encode error: %v", err)
	}
}

func (s *Server) handleListCheckJobs(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := s.checkJobService.ListCheckJobs(ctx, serviceapi.ListCheckJobsInput{
		CompanyID: companyID,
		StreamID:  streamID,
		StatusRaw: r.URL.Query().Get("status"),
		FromRaw:   r.URL.Query().Get("from"),
		ToRaw:     r.URL.Query().Get("to"),
	})
	if err != nil {
		writeServiceError(w, r, "list check jobs", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, checkJobListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list check jobs response encode error: %v", err)
	}
}

func (s *Server) handleGetAIIncident(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64, jobID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.aiIncidentService.Get(ctx, companyID, streamID, jobID)
	if err != nil {
		writeServiceError(w, r, "get ai incident", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get ai incident response encode error: %v", err)
	}
}
