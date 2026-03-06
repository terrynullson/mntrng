package api

import (
	"context"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/terrynullson/mntrng/internal/config"
	"github.com/terrynullson/mntrng/internal/domain"
	serviceapi "github.com/terrynullson/mntrng/internal/service/api"
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
	copy(out, items)
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
	absPath, pathErr := secureIncidentScreenshotPath(dataRoot, strings.TrimSpace(*item.SampleScreenshotPath))
	if pathErr != nil {
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
	if !info.Mode().IsRegular() {
		WriteJSONError(w, r, http.StatusNotFound, "not_found", "screenshot not found", map[string]interface{}{"incident_id": incidentID})
		return
	}
	header := make([]byte, 512)
	n, _ := io.ReadFull(file, header)
	contentType := strings.ToLower(strings.TrimSpace(http.DetectContentType(header[:n])))
	if !strings.HasPrefix(contentType, "image/jpeg") {
		WriteJSONError(w, r, http.StatusNotFound, "not_found", "screenshot not found", map[string]interface{}{"incident_id": incidentID})
		return
	}
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		WriteJSONError(w, r, http.StatusNotFound, "not_found", "screenshot not found", map[string]interface{}{"incident_id": incidentID})
		return
	}
	if ext := strings.ToLower(filepath.Ext(info.Name())); ext == ".jpg" || ext == ".jpeg" {
		contentType = mime.TypeByExtension(ext)
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func secureIncidentScreenshotPath(dataRoot string, screenshotPath string) (string, error) {
	raw := strings.TrimSpace(screenshotPath)
	if raw == "" {
		return "", errors.New("empty screenshot path")
	}
	if !filepath.IsAbs(raw) {
		return "", errors.New("path must be absolute")
	}
	incidentsRoot := filepath.Join(filepath.Clean(dataRoot), "screenshots", "incidents")
	rootAbs, err := filepath.Abs(incidentsRoot)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Clean(raw))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	rel = filepath.Clean(rel)
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", errors.New("path outside incidents root")
	}
	return targetAbs, nil
}
