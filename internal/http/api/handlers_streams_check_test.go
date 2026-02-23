package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

var testTime = time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

// --- Stream mocks and tests ---

type mockStreamStore struct {
	listResp   []domain.Stream
	listErr    error
	getResp    domain.Stream
	getErr     error
	createResp domain.Stream
	createErr  error
	patchResp  domain.Stream
	patchErr   error
	deleteErr  error
}

func (m *mockStreamStore) ListStreams(ctx context.Context, companyID int64, filter domain.StreamListFilter) ([]domain.Stream, error) {
	return m.listResp, m.listErr
}
func (m *mockStreamStore) GetStream(ctx context.Context, companyID int64, streamID int64) (domain.Stream, error) {
	return m.getResp, m.getErr
}
func (m *mockStreamStore) CreateStream(ctx context.Context, companyID int64, projectID int64, name string, url string, isActive bool) (domain.Stream, error) {
	return m.createResp, m.createErr
}
func (m *mockStreamStore) PatchStream(ctx context.Context, companyID int64, streamID int64, patch domain.StreamPatchInput) (domain.Stream, error) {
	return m.patchResp, m.patchErr
}
func (m *mockStreamStore) DeleteStream(ctx context.Context, companyID int64, streamID int64) error {
	return m.deleteErr
}

func TestHandleListStreams_200(t *testing.T) {
	store := &mockStreamStore{
		listResp: []domain.Stream{{ID: 1, CompanyID: 10, ProjectID: 1, Name: "S1", URL: "https://a/b.m3u8", IsActive: true, CreatedAt: testTime, UpdatedAt: testTime}},
	}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams", nil)
	rec := httptest.NewRecorder()
	srv.handleListStreams(rec, req, 10)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var out domain.StreamListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 1 || out.Items[0].Name != "S1" {
		t.Fatalf("unexpected items: %+v", out.Items)
	}
}

func TestHandleGetStream_200(t *testing.T) {
	store := &mockStreamStore{
		getResp: domain.Stream{ID: 2, CompanyID: 10, ProjectID: 1, Name: "S2", URL: "https://x/y.m3u8", IsActive: true, CreatedAt: testTime, UpdatedAt: testTime},
	}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams/2", nil)
	rec := httptest.NewRecorder()
	srv.handleGetStream(rec, req, 10, 2)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var out domain.Stream
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != 2 || out.Name != "S2" {
		t.Fatalf("unexpected stream: %+v", out)
	}
}

func TestHandleGetStream_404(t *testing.T) {
	store := &mockStreamStore{getErr: domain.ErrStreamNotFound}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams/99", nil)
	rec := httptest.NewRecorder()
	srv.handleGetStream(rec, req, 10, 99)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func TestHandleCreateStream_201(t *testing.T) {
	store := &mockStreamStore{
		createResp: domain.Stream{ID: 3, CompanyID: 10, ProjectID: 1, Name: "New", URL: "https://c/d.m3u8", IsActive: true, CreatedAt: testTime, UpdatedAt: testTime},
	}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	body := []byte(`{"name":"New","url":"https://c/d.m3u8","is_active":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/10/projects/1/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCreateStream(rec, req, 10, 1)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	var out domain.Stream
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "New" {
		t.Fatalf("unexpected name: %s", out.Name)
	}
}

func TestHandleCreateStream_404_ProjectMiss(t *testing.T) {
	store := &mockStreamStore{createErr: domain.ErrStreamProjectMiss}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	body := []byte(`{"name":"X","url":"https://x/y.m3u8"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/10/projects/99/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCreateStream(rec, req, 10, 99)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func TestHandlePatchStream_200(t *testing.T) {
	store := &mockStreamStore{
		patchResp: domain.Stream{ID: 4, CompanyID: 10, ProjectID: 1, Name: "Updated", URL: "https://e/f.m3u8", IsActive: false, CreatedAt: testTime, UpdatedAt: testTime},
	}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	body := []byte(`{"name":"Updated","url":"https://e/f.m3u8","is_active":false}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/companies/10/streams/4", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handlePatchStream(rec, req, 10, 4)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePatchStream_404(t *testing.T) {
	store := &mockStreamStore{patchErr: domain.ErrStreamNotFound}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	body := []byte(`{"name":"X","url":"https://x/y.m3u8"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/companies/10/streams/99", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handlePatchStream(rec, req, 10, 99)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteStream_204(t *testing.T) {
	store := &mockStreamStore{}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/companies/10/streams/5", nil)
	rec := httptest.NewRecorder()
	srv.handleDeleteStream(rec, req, 10, 5)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteStream_404(t *testing.T) {
	store := &mockStreamStore{deleteErr: domain.ErrStreamNotFound}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/companies/10/streams/99", nil)
	rec := httptest.NewRecorder()
	srv.handleDeleteStream(rec, req, 10, 99)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestStreams_401_NoAuth(t *testing.T) {
	authStore := &middlewareAuthStore{sessionByAccess: nil}
	srv := &Server{
		authService:   serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		streamService: serviceapi.NewStreamService(&mockStreamStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/streams", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "unauthorized")
}

func TestStreams_403_TenantEscape(t *testing.T) {
	companyID := int64(1)
	hash := sha256.Sum256([]byte("stream-tenant-token"))
	authStore := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			hex.EncodeToString(hash[:]): {
				Session: domain.AuthSession{ID: 1, AccessExpiresAt: time.Now().Add(15 * time.Minute), RefreshExpiresAt: time.Now().Add(24 * time.Hour)},
				User:    domain.AuthUser{ID: 1, CompanyID: &companyID, Role: domain.RoleViewer, Status: domain.UserStatusActive},
			},
		},
	}
	srv := &Server{
		authService:   serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		streamService: serviceapi.NewStreamService(&mockStreamStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/streams", nil)
	req.Header.Set("Authorization", "Bearer stream-tenant-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "tenant_scope_required")
}

func assertErrorCode(t *testing.T, body []byte, code string) {
	t.Helper()
	var e domain.ErrorEnvelope
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if e.Code != code {
		t.Fatalf("expected code %q, got %q", code, e.Code)
	}
}

// --- Check job mocks and tests ---

type mockCheckJobStore struct {
	enqueueResp     domain.CheckJob
	enqueueErr      error
	getResp         domain.CheckJob
	getErr          error
	streamExists    bool
	streamExistsErr error
	listResp        []domain.CheckJob
	listErr         error
}

func (m *mockCheckJobStore) EnqueueCheckJob(ctx context.Context, companyID int64, streamID int64, plannedAt time.Time) (domain.CheckJob, error) {
	return m.enqueueResp, m.enqueueErr
}
func (m *mockCheckJobStore) GetCheckJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckJob, error) {
	return m.getResp, m.getErr
}
func (m *mockCheckJobStore) StreamExistsForCheckJobs(ctx context.Context, companyID int64, streamID int64) (bool, error) {
	return m.streamExists, m.streamExistsErr
}
func (m *mockCheckJobStore) ListCheckJobs(ctx context.Context, companyID int64, streamID int64, filter domain.CheckJobListFilter) ([]domain.CheckJob, error) {
	return m.listResp, m.listErr
}

func TestHandleEnqueueCheckJob_202(t *testing.T) {
	store := &mockCheckJobStore{
		enqueueResp: domain.CheckJob{ID: 100, CompanyID: 10, StreamID: 1, PlannedAt: testTime, Status: "queued", CreatedAt: testTime, StartedAt: nil, FinishedAt: nil, ErrorMessage: nil},
	}
	srv := &Server{checkJobService: serviceapi.NewCheckJobService(store)}
	body := []byte(`{"planned_at":"2026-02-01T12:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/10/streams/1/check-jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleEnqueueCheckJob(rec, req, 10, 1)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d %s", rec.Code, rec.Body.String())
	}
	var out domain.EnqueueCheckJobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Job.ID != 100 || out.Job.Status != "queued" {
		t.Fatalf("unexpected job: %+v", out.Job)
	}
}

func TestHandleEnqueueCheckJob_404_StreamMiss(t *testing.T) {
	store := &mockCheckJobStore{enqueueErr: domain.ErrCheckJobStreamMissing}
	srv := &Server{checkJobService: serviceapi.NewCheckJobService(store)}
	body := []byte(`{"planned_at":"2026-02-01T12:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/10/streams/99/check-jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleEnqueueCheckJob(rec, req, 10, 99)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func TestHandleGetCheckJob_200(t *testing.T) {
	store := &mockCheckJobStore{
		getResp: domain.CheckJob{ID: 101, CompanyID: 10, StreamID: 1, PlannedAt: testTime, Status: "done", CreatedAt: testTime, StartedAt: &testTime, FinishedAt: &testTime, ErrorMessage: nil},
	}
	srv := &Server{checkJobService: serviceapi.NewCheckJobService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/check-jobs/101", nil)
	rec := httptest.NewRecorder()
	srv.handleGetCheckJob(rec, req, 10, 101)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetCheckJob_404(t *testing.T) {
	store := &mockCheckJobStore{getErr: domain.ErrCheckJobNotFound}
	srv := &Server{checkJobService: serviceapi.NewCheckJobService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/check-jobs/999", nil)
	rec := httptest.NewRecorder()
	srv.handleGetCheckJob(rec, req, 10, 999)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func TestHandleListCheckJobs_200(t *testing.T) {
	store := &mockCheckJobStore{streamExists: true, listResp: []domain.CheckJob{
		{ID: 102, CompanyID: 10, StreamID: 1, PlannedAt: testTime, Status: "done", CreatedAt: testTime, StartedAt: nil, FinishedAt: nil, ErrorMessage: nil},
	}}
	srv := &Server{checkJobService: serviceapi.NewCheckJobService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams/1/check-jobs", nil)
	rec := httptest.NewRecorder()
	srv.handleListCheckJobs(rec, req, 10, 1)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListCheckJobs_404_StreamMiss(t *testing.T) {
	store := &mockCheckJobStore{streamExists: false}
	srv := &Server{checkJobService: serviceapi.NewCheckJobService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams/99/check-jobs", nil)
	rec := httptest.NewRecorder()
	srv.handleListCheckJobs(rec, req, 10, 99)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func TestCheckJobs_401_NoAuth(t *testing.T) {
	authStore := &middlewareAuthStore{sessionByAccess: nil}
	srv := &Server{
		authService:     serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		checkJobService: serviceapi.NewCheckJobService(&mockCheckJobStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/check-jobs/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "unauthorized")
}

// --- Check result mocks and tests ---

type mockCheckResultStore struct {
	getByIDResp   domain.CheckResult
	getByIDErr    error
	getByJobResp  domain.CheckResult
	getByJobErr   error
	streamExists  bool
	streamExistsErr error
	listResp      []domain.CheckResult
	listErr       error
}

func (m *mockCheckResultStore) GetCheckResultByID(ctx context.Context, companyID int64, resultID int64) (domain.CheckResult, error) {
	return m.getByIDResp, m.getByIDErr
}
func (m *mockCheckResultStore) GetCheckResultByJobID(ctx context.Context, companyID int64, jobID int64) (domain.CheckResult, error) {
	return m.getByJobResp, m.getByJobErr
}
func (m *mockCheckResultStore) StreamExistsForCheckResults(ctx context.Context, companyID int64, streamID int64) (bool, error) {
	return m.streamExists, m.streamExistsErr
}
func (m *mockCheckResultStore) ListCheckResults(ctx context.Context, companyID int64, streamID int64, filter domain.CheckResultListFilter) ([]domain.CheckResult, error) {
	return m.listResp, m.listErr
}

func TestHandleGetCheckResult_200(t *testing.T) {
	store := &mockCheckResultStore{
		getByIDResp: domain.CheckResult{ID: 1, CompanyID: 10, JobID: 100, StreamID: 1, Status: "OK", Checks: json.RawMessage(`{}`), ScreenshotPath: nil, CreatedAt: testTime},
	}
	srv := &Server{checkResultService: serviceapi.NewCheckResultService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/check-results/1", nil)
	rec := httptest.NewRecorder()
	srv.handleGetCheckResult(rec, req, 10, 1)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetCheckResult_404(t *testing.T) {
	store := &mockCheckResultStore{getByIDErr: domain.ErrCheckResultNotFound}
	srv := &Server{checkResultService: serviceapi.NewCheckResultService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/check-results/999", nil)
	rec := httptest.NewRecorder()
	srv.handleGetCheckResult(rec, req, 10, 999)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func TestHandleGetCheckResultByJob_200(t *testing.T) {
	store := &mockCheckResultStore{
		getByJobResp: domain.CheckResult{ID: 2, CompanyID: 10, JobID: 101, StreamID: 1, Status: "FAIL", Checks: json.RawMessage(`{}`), ScreenshotPath: nil, CreatedAt: testTime},
	}
	srv := &Server{checkResultService: serviceapi.NewCheckResultService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/check-jobs/101/result", nil)
	rec := httptest.NewRecorder()
	srv.handleGetCheckResultByJob(rec, req, 10, 101)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetCheckResultByJob_404(t *testing.T) {
	store := &mockCheckResultStore{getByJobErr: domain.ErrCheckResultByJobNotFound}
	srv := &Server{checkResultService: serviceapi.NewCheckResultService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/check-jobs/999/result", nil)
	rec := httptest.NewRecorder()
	srv.handleGetCheckResultByJob(rec, req, 10, 999)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func TestHandleListCheckResults_200(t *testing.T) {
	store := &mockCheckResultStore{
		streamExists: true,
		listResp:     []domain.CheckResult{{ID: 3, CompanyID: 10, JobID: 102, StreamID: 1, Status: "WARN", Checks: json.RawMessage(`{}`), ScreenshotPath: nil, CreatedAt: testTime}},
	}
	srv := &Server{checkResultService: serviceapi.NewCheckResultService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams/1/check-results", nil)
	rec := httptest.NewRecorder()
	srv.handleListCheckResults(rec, req, 10, 1)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListCheckResults_404_StreamMiss(t *testing.T) {
	store := &mockCheckResultStore{streamExists: false}
	srv := &Server{checkResultService: serviceapi.NewCheckResultService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams/99/check-results", nil)
	rec := httptest.NewRecorder()
	srv.handleListCheckResults(rec, req, 10, 99)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "not_found")
}

func TestCheckResults_401_NoAuth(t *testing.T) {
	authStore := &middlewareAuthStore{sessionByAccess: nil}
	srv := &Server{
		authService:       serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		checkResultService: serviceapi.NewCheckResultService(&mockCheckResultStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/streams/1/check-results", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "unauthorized")
}

func TestCheckResults_403_TenantEscape(t *testing.T) {
	companyID := int64(1)
	hash := sha256.Sum256([]byte("result-tenant-token"))
	authStore := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			hex.EncodeToString(hash[:]): {
				Session: domain.AuthSession{ID: 1, AccessExpiresAt: time.Now().Add(15 * time.Minute), RefreshExpiresAt: time.Now().Add(24 * time.Hour)},
				User:    domain.AuthUser{ID: 1, CompanyID: &companyID, Role: domain.RoleViewer, Status: domain.UserStatusActive},
			},
		},
	}
	srv := &Server{
		authService:       serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		checkResultService: serviceapi.NewCheckResultService(&mockCheckResultStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/streams/1/check-results", nil)
	req.Header.Set("Authorization", "Bearer result-tenant-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "tenant_scope_required")
}
