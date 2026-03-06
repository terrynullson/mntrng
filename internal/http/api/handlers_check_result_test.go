package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
	serviceapi "github.com/terrynullson/mntrng/internal/service/api"
)

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
		authService:        serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
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
		authService:        serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
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

func TestCheckResults_403_TenantEscape_GetByID(t *testing.T) {
	companyID := int64(1)
	hash := sha256.Sum256([]byte("result-tenant-get-id-token"))
	authStore := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			hex.EncodeToString(hash[:]): {
				Session: domain.AuthSession{ID: 1, AccessExpiresAt: time.Now().Add(15 * time.Minute), RefreshExpiresAt: time.Now().Add(24 * time.Hour)},
				User:    domain.AuthUser{ID: 1, CompanyID: &companyID, Role: domain.RoleViewer, Status: domain.UserStatusActive},
			},
		},
	}
	srv := &Server{
		authService:        serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		checkResultService: serviceapi.NewCheckResultService(&mockCheckResultStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/check-results/1", nil)
	req.Header.Set("Authorization", "Bearer result-tenant-get-id-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "tenant_scope_required")
}

func TestCheckResults_403_TenantEscape_GetByJob(t *testing.T) {
	companyID := int64(1)
	hash := sha256.Sum256([]byte("result-tenant-get-job-token"))
	authStore := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			hex.EncodeToString(hash[:]): {
				Session: domain.AuthSession{ID: 1, AccessExpiresAt: time.Now().Add(15 * time.Minute), RefreshExpiresAt: time.Now().Add(24 * time.Hour)},
				User:    domain.AuthUser{ID: 1, CompanyID: &companyID, Role: domain.RoleViewer, Status: domain.UserStatusActive},
			},
		},
	}
	srv := &Server{
		authService:        serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		checkResultService: serviceapi.NewCheckResultService(&mockCheckResultStore{}),
	}
	router := NewRouter(srv.RouterHandlers())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/check-jobs/1/result", nil)
	req.Header.Set("Authorization", "Bearer result-tenant-get-job-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "tenant_scope_required")
}
