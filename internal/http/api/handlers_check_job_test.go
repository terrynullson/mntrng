package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
	serviceapi "github.com/terrynullson/hls_mntrng/internal/service/api"
)

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

func TestHandleTriggerStreamCheck_202(t *testing.T) {
	store := &mockCheckJobStore{
		enqueueResp: domain.CheckJob{ID: 103, CompanyID: 10, StreamID: 1, PlannedAt: testTime, Status: "queued", CreatedAt: testTime, StartedAt: nil, FinishedAt: nil, ErrorMessage: nil},
	}
	srv := &Server{checkJobService: serviceapi.NewCheckJobService(store)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/10/streams/1/check", nil)
	rec := httptest.NewRecorder()
	srv.handleTriggerStreamCheck(rec, req, 10, 1)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d %s", rec.Code, rec.Body.String())
	}
	var out domain.EnqueueCheckJobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Job.ID != 103 {
		t.Fatalf("unexpected job id: %d", out.Job.ID)
	}
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

func TestCheckJobs_403_TenantEscape_Enqueue(t *testing.T) {
	companyID := int64(1)
	hash := sha256.Sum256([]byte("checkjob-tenant-token"))
	authStore := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			hex.EncodeToString(hash[:]): {
				Session: domain.AuthSession{ID: 1, AccessExpiresAt: time.Now().Add(15 * time.Minute), RefreshExpiresAt: time.Now().Add(24 * time.Hour)},
				User:    domain.AuthUser{ID: 1, CompanyID: &companyID, Role: domain.RoleViewer, Status: domain.UserStatusActive},
			},
		},
	}
	srv := &Server{
		authService:     serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		checkJobService: serviceapi.NewCheckJobService(&mockCheckJobStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	body := []byte(`{"planned_at":"2026-02-01T12:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/2/streams/1/check-jobs", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer checkjob-tenant-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "tenant_scope_required")
}

func TestCheckJobs_403_TenantEscape_List(t *testing.T) {
	companyID := int64(1)
	hash := sha256.Sum256([]byte("checkjob-tenant-token-list"))
	authStore := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			hex.EncodeToString(hash[:]): {
				Session: domain.AuthSession{ID: 1, AccessExpiresAt: time.Now().Add(15 * time.Minute), RefreshExpiresAt: time.Now().Add(24 * time.Hour)},
				User:    domain.AuthUser{ID: 1, CompanyID: &companyID, Role: domain.RoleViewer, Status: domain.UserStatusActive},
			},
		},
	}
	srv := &Server{
		authService:     serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		checkJobService: serviceapi.NewCheckJobService(&mockCheckJobStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/streams/1/check-jobs", nil)
	req.Header.Set("Authorization", "Bearer checkjob-tenant-token-list")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "tenant_scope_required")
}

func TestCheckJobs_403_TenantEscape_Get(t *testing.T) {
	companyID := int64(1)
	hash := sha256.Sum256([]byte("checkjob-tenant-token-get"))
	authStore := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			hex.EncodeToString(hash[:]): {
				Session: domain.AuthSession{ID: 1, AccessExpiresAt: time.Now().Add(15 * time.Minute), RefreshExpiresAt: time.Now().Add(24 * time.Hour)},
				User:    domain.AuthUser{ID: 1, CompanyID: &companyID, Role: domain.RoleViewer, Status: domain.UserStatusActive},
			},
		},
	}
	srv := &Server{
		authService:     serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		checkJobService: serviceapi.NewCheckJobService(&mockCheckJobStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/check-jobs/1", nil)
	req.Header.Set("Authorization", "Bearer checkjob-tenant-token-get")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "tenant_scope_required")
}
