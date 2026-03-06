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

	"github.com/terrynullson/mntrng/internal/domain"
	serviceapi "github.com/terrynullson/mntrng/internal/service/api"
)

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

func TestHandleListStreamsWithLatestStatus_200(t *testing.T) {
	okStatus := "OK"
	store := &mockStreamStore{
		listWithStatusResp: []domain.StreamWithLatestStatus{
			{
				Stream:       domain.Stream{ID: 1, CompanyID: 10, ProjectID: 1, Name: "S1", URL: "https://a/b.m3u8", IsActive: true, CreatedAt: testTime, UpdatedAt: testTime},
				LatestStatus: &domain.StreamLatestStatus{StreamID: 1, Status: &okStatus, LastCheckAt: ptrTime(testTime)},
			},
		},
	}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams/operational-summary", nil)
	rec := httptest.NewRecorder()
	srv.handleListStreamsWithLatestStatus(rec, req, 10)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var out domain.StreamWithLatestStatusListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].Stream.Name != "S1" {
		t.Fatalf("unexpected items: %+v", out.Items)
	}
	if out.Items[0].LatestStatus == nil || *out.Items[0].LatestStatus.Status != "OK" {
		t.Fatalf("unexpected latest_status: %+v", out.Items[0].LatestStatus)
	}
}

func TestHandleListStreamLatestStatuses_200(t *testing.T) {
	okStatus := "OK"
	store := &mockStreamStore{
		latestResp: []domain.StreamLatestStatus{
			{StreamID: 1, Status: &okStatus, LastCheckAt: ptrTime(testTime)},
			{StreamID: 2, Status: nil, LastCheckAt: nil},
		},
	}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/streams/latest-statuses", nil)
	rec := httptest.NewRecorder()
	srv.handleListStreamLatestStatuses(rec, req, 10)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}

	var out domain.StreamLatestStatusListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("unexpected items len: %d", len(out.Items))
	}
	if out.Items[0].Status == nil || *out.Items[0].Status != "OK" {
		t.Fatalf("unexpected first status: %+v", out.Items[0])
	}
	if out.Items[1].Status != nil {
		t.Fatalf("expected nil status for second stream: %+v", out.Items[1])
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

func TestHandleCreateStreamInCompany_201(t *testing.T) {
	store := &mockStreamStore{
		createResp: domain.Stream{ID: 9, CompanyID: 10, ProjectID: 1, Name: "New by company", URL: "https://c/d.m3u8", IsActive: true, CreatedAt: testTime, UpdatedAt: testTime},
	}
	srv := &Server{streamService: serviceapi.NewStreamService(store)}
	body := []byte(`{"project_id":1,"name":"New by company","url":"https://c/d.m3u8","is_active":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/10/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCreateStreamInCompany(rec, req, 10)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateStreamInCompany_400_MissingProjectID(t *testing.T) {
	srv := &Server{streamService: serviceapi.NewStreamService(&mockStreamStore{})}
	body := []byte(`{"name":"New by company","url":"https://c/d.m3u8","is_active":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies/10/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCreateStreamInCompany(rec, req, 10)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "validation_error")
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

func TestStreams_403_TenantEscape_Get(t *testing.T) {
	companyID := int64(1)
	hash := sha256.Sum256([]byte("stream-tenant-get-token"))
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/streams/1", nil)
	req.Header.Set("Authorization", "Bearer stream-tenant-get-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "tenant_scope_required")
}

func ptrTime(value time.Time) *time.Time {
	v := value
	return &v
}
