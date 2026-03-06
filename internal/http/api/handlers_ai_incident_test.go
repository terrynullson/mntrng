package api

import (
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

type mockAIIncidentStore struct {
	resp domain.AIIncidentResponse
	err  error
}

func (m *mockAIIncidentStore) GetByCompanyStreamJob(ctx context.Context, companyID int64, streamID int64, jobID int64) (domain.AIIncidentResponse, error) {
	return m.resp, m.err
}

func TestHandleGetAIIncident_200(t *testing.T) {
	store := &mockAIIncidentStore{
		resp: domain.AIIncidentResponse{Cause: "playlist timeout", Summary: "HLS playlist did not respond in time"},
		err:  nil,
	}
	server := &Server{aiIncidentService: serviceapi.NewAIIncidentService(store)}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/streams/2/check-jobs/3/ai-incident", nil)
	rec := httptest.NewRecorder()

	server.handleGetAIIncident(rec, req, 1, 2, 3)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var out domain.AIIncidentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Cause != "playlist timeout" || out.Summary != "HLS playlist did not respond in time" {
		t.Fatalf("unexpected body: cause=%q summary=%q", out.Cause, out.Summary)
	}
}

func TestHandleGetAIIncident_404(t *testing.T) {
	store := &mockAIIncidentStore{
		resp: domain.AIIncidentResponse{},
		err:  domain.ErrAIIncidentNotFound,
	}
	server := &Server{aiIncidentService: serviceapi.NewAIIncidentService(store)}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/streams/2/check-jobs/99/ai-incident", nil)
	rec := httptest.NewRecorder()

	server.handleGetAIIncident(rec, req, 1, 2, 99)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "not_found" {
		t.Fatalf("expected code not_found, got %s", envelope.Code)
	}
}

func TestGetAIIncident_401_NoAuth(t *testing.T) {
	authStore := &middlewareAuthStore{sessionByAccess: nil}
	server := &Server{
		authService:       serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		aiIncidentService: serviceapi.NewAIIncidentService(&mockAIIncidentStore{}),
	}
	router := NewRouter(server.RouterHandlers())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/streams/1/check-jobs/1/ai-incident", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "unauthorized" {
		t.Fatalf("expected code unauthorized, got %s", envelope.Code)
	}
}

func TestGetAIIncident_403_TenantEscape(t *testing.T) {
	accessToken := "ai-incident-tenant-token"
	hash := sha256.Sum256([]byte(accessToken))
	accessHash := hex.EncodeToString(hash[:])

	companyID := int64(1)
	authStore := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			accessHash: {
				Session: domain.AuthSession{
					ID:               1,
					AccessExpiresAt:  time.Now().Add(15 * time.Minute),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				},
				User: domain.AuthUser{
					ID:        1,
					CompanyID: &companyID,
					Role:      domain.RoleViewer,
					Status:    domain.UserStatusActive,
				},
			},
		},
	}
	server := &Server{
		authService:       serviceapi.NewAuthService(authStore, serviceapi.AuthConfig{}),
		aiIncidentService: serviceapi.NewAIIncidentService(&mockAIIncidentStore{}),
	}
	router := NewRouter(server.RouterHandlers())

	// User company_id=1, path company_id=2 -> 403
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/streams/1/check-jobs/1/ai-incident", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "tenant_scope_required" {
		t.Fatalf("expected code tenant_scope_required, got %s", envelope.Code)
	}
}
