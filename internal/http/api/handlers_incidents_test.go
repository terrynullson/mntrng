package api

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
	serviceapi "github.com/terrynullson/hls_mntrng/internal/service/api"
)

type mockIncidentStore struct {
	getResp domain.Incident
	getErr  error
}

func (m *mockIncidentStore) List(ctx context.Context, companyID int64, filter domain.IncidentListFilter) ([]domain.Incident, int64, error) {
	return nil, 0, nil
}

func (m *mockIncidentStore) GetByID(ctx context.Context, companyID int64, incidentID int64) (domain.Incident, error) {
	return m.getResp, m.getErr
}

func TestHandleGetIncidentScreenshot_200(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("APP_DATA_DIR", tempRoot)
	screenshotPath := filepath.Join(tempRoot, "screenshots", "incidents", "10", "5", "shot.jpg")
	if err := os.MkdirAll(filepath.Dir(screenshotPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeTestJPEG(screenshotPath); err != nil {
		t.Fatalf("write jpeg: %v", err)
	}

	store := &mockIncidentStore{
		getResp: domain.Incident{
			ID:                   5,
			CompanyID:            10,
			StreamID:             20,
			Status:               domain.IncidentStatusOpen,
			Severity:             domain.IncidentSeverityWarn,
			StartedAt:            time.Now().UTC(),
			LastEventAt:          time.Now().UTC(),
			SampleScreenshotPath: ptrString(screenshotPath),
		},
	}
	srv := &Server{incidentService: serviceapi.NewIncidentService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/incidents/5/screenshot", nil)
	rec := httptest.NewRecorder()

	srv.handleGetIncidentScreenshot(rec, req, 10, 5)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %q", got)
	}
}

func TestHandleGetIncidentScreenshot_403_PathTraversal(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("APP_DATA_DIR", tempRoot)
	invalidPath := filepath.Join(tempRoot, "screenshots", "incidents", "..", "..", "outside.jpg")

	store := &mockIncidentStore{
		getResp: domain.Incident{
			ID:                   5,
			CompanyID:            10,
			StreamID:             20,
			Status:               domain.IncidentStatusOpen,
			Severity:             domain.IncidentSeverityWarn,
			StartedAt:            time.Now().UTC(),
			LastEventAt:          time.Now().UTC(),
			SampleScreenshotPath: ptrString(invalidPath),
		},
	}
	srv := &Server{incidentService: serviceapi.NewIncidentService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/incidents/5/screenshot", nil)
	rec := httptest.NewRecorder()

	srv.handleGetIncidentScreenshot(rec, req, 10, 5)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "forbidden")
}

func TestHandleGetIncidentScreenshot_404_MissingFile(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("APP_DATA_DIR", tempRoot)
	missingPath := filepath.Join(tempRoot, "screenshots", "incidents", "10", "5", "missing.jpg")

	store := &mockIncidentStore{
		getResp: domain.Incident{
			ID:                   5,
			CompanyID:            10,
			StreamID:             20,
			Status:               domain.IncidentStatusOpen,
			Severity:             domain.IncidentSeverityWarn,
			StartedAt:            time.Now().UTC(),
			LastEventAt:          time.Now().UTC(),
			SampleScreenshotPath: ptrString(missingPath),
		},
	}
	srv := &Server{incidentService: serviceapi.NewIncidentService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/incidents/5/screenshot", nil)
	rec := httptest.NewRecorder()

	srv.handleGetIncidentScreenshot(rec, req, 10, 5)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func writeTestJPEG(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 120, G: 120, B: 120, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, nil); err != nil {
		return err
	}
	return os.WriteFile(path, buffer.Bytes(), 0o644)
}

func ptrString(value string) *string {
	return &value
}
