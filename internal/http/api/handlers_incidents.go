package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/example/hls-monitoring-platform/internal/domain"
	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

type incidentResponse = domain.Incident

type incidentListResponse struct {
	Items      []incidentResponse `json:"items"`
	NextCursor *string            `json:"next_cursor,omitempty"`
	Total      int64              `json:"total"`
}

func (s *Server) handleListIncidents(w http.ResponseWriter, r *http.Request, companyID int64) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowed(w, r, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	q := r.URL.Query()
	input := serviceapi.ListIncidentsInput{
		CompanyID: companyID,
		Status:    q.Get("status"),
		Severity:  q.Get("severity"),
		StreamID:  q.Get("stream_id"),
		Q:         q.Get("q"),
		Page:      q.Get("page"),
		PageSize:  q.Get("page_size"),
	}
	items, total, nextCursor, err := s.incidentService.List(ctx, input)
	if err != nil {
		writeServiceError(w, r, "list incidents", err)
		return
	}
	out := make([]incidentResponse, len(items))
	for i := range items {
		out[i] = items[i]
	}
	if err := WriteJSON(w, http.StatusOK, incidentListResponse{Items: out, NextCursor: nextCursor, Total: total}); err != nil {
		log.Printf("list incidents response encode error: %v", err)
	}
}

func (s *Server) handleGetIncident(w http.ResponseWriter, r *http.Request, companyID int64, incidentID int64) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowed(w, r, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := s.incidentService.Get(ctx, companyID, incidentID)
	if err != nil {
		writeServiceError(w, r, "get incident", err)
		return
	}
	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get incident response encode error: %v", err)
	}
}

func (s *Server) handleGetIncidentScreenshot(w http.ResponseWriter, r *http.Request, companyID int64, incidentID int64) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowed(w, r, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := s.incidentService.Get(ctx, companyID, incidentID)
	if err != nil {
		writeServiceError(w, r, "get incident screenshot", err)
		return
	}
	if item.SampleScreenshotPath == nil || strings.TrimSpace(*item.SampleScreenshotPath) == "" {
		WriteJSONError(w, r, http.StatusNotFound, "not_found", "screenshot not found", map[string]interface{}{"incident_id": incidentID})
		return
	}
	dataRoot := filepath.Clean(config.GetString("APP_DATA_DIR", "/data"))
	absPath := filepath.Clean(strings.TrimSpace(*item.SampleScreenshotPath))
	incidentsRoot := filepath.Join(dataRoot, "screenshots", "incidents")
	if !strings.HasPrefix(absPath, incidentsRoot) {
		WriteJSONError(w, r, http.StatusForbidden, "forbidden", "invalid screenshot path", map[string]interface{}{"incident_id": incidentID})
		return
	}
	file, openErr := os.Open(absPath)
	if openErr != nil {
		WriteJSONError(w, r, http.StatusNotFound, "not_found", "screenshot not found", map[string]interface{}{"incident_id": incidentID})
		return
	}
	defer file.Close()
	info, statErr := file.Stat()
	if statErr != nil {
		WriteJSONError(w, r, http.StatusNotFound, "not_found", "screenshot not found", map[string]interface{}{"incident_id": incidentID})
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
