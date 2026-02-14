package api

import (
	"context"
	"log"
	"net/http"
	"time"

	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

func (s *Server) handleGetCheckResult(w http.ResponseWriter, r *http.Request, companyID int64, resultID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.checkResultService.GetCheckResult(ctx, companyID, resultID)
	if err != nil {
		writeServiceError(w, r, "get check result", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check result response encode error: %v", err)
	}
}

func (s *Server) handleListCheckResults(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := s.checkResultService.ListCheckResults(ctx, serviceapi.ListCheckResultsInput{
		CompanyID: companyID,
		StreamID:  streamID,
		StatusRaw: r.URL.Query().Get("status"),
		FromRaw:   r.URL.Query().Get("from"),
		ToRaw:     r.URL.Query().Get("to"),
	})
	if err != nil {
		writeServiceError(w, r, "list check results", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, checkResultListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list check results response encode error: %v", err)
	}
}

func (s *Server) handleGetCheckResultByJob(w http.ResponseWriter, r *http.Request, companyID int64, jobID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.checkResultService.GetCheckResultByJob(ctx, companyID, jobID)
	if err != nil {
		writeServiceError(w, r, "get check result by job", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check result by job response encode error: %v", err)
	}
}
