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

type middlewareAuthStore struct {
	sessionByAccess map[string]domain.AuthSessionUser
}

func (s *middlewareAuthStore) GetUserByLoginOrEmail(ctx context.Context, identity string) (domain.UserRecord, error) {
	return domain.UserRecord{}, domain.ErrUserNotFound
}

func (s *middlewareAuthStore) GetUserByID(ctx context.Context, userID int64) (domain.UserRecord, error) {
	return domain.UserRecord{}, domain.ErrUserNotFound
}

func (s *middlewareAuthStore) CreateSession(ctx context.Context, user domain.AuthUser, accessTokenHash string, refreshTokenHash string, accessExpiresAt time.Time, refreshExpiresAt time.Time) (domain.AuthSession, error) {
	return domain.AuthSession{}, nil
}

func (s *middlewareAuthStore) GetSessionByAccessTokenHash(ctx context.Context, accessTokenHash string) (domain.AuthSessionUser, error) {
	item, ok := s.sessionByAccess[accessTokenHash]
	if !ok {
		return domain.AuthSessionUser{}, domain.ErrSessionNotFound
	}
	return item, nil
}

func (s *middlewareAuthStore) GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (domain.AuthSessionUser, error) {
	return domain.AuthSessionUser{}, domain.ErrSessionNotFound
}

func (s *middlewareAuthStore) RotateSessionTokens(ctx context.Context, sessionID int64, accessTokenHash string, refreshTokenHash string, accessExpiresAt time.Time, refreshExpiresAt time.Time) error {
	return nil
}

func (s *middlewareAuthStore) RevokeSessionByID(ctx context.Context, sessionID int64) error {
	return nil
}

func (s *middlewareAuthStore) RevokeSessionByRefreshToken(ctx context.Context, userID int64, refreshTokenHash string) error {
	return nil
}

func (s *middlewareAuthStore) HasPendingRegistration(ctx context.Context, identity string) (bool, error) {
	return false, nil
}

func (s *middlewareAuthStore) UpsertTelegramLink(ctx context.Context, user domain.AuthUser, telegramUserID int64, telegramUsername *string) error {
	return nil
}

func (s *middlewareAuthStore) GetUserByTelegramUserID(ctx context.Context, telegramUserID int64) (domain.UserRecord, error) {
	return domain.UserRecord{}, domain.ErrUserNotFound
}

func TestAuthMiddlewareDeniesRequestWithoutToken(t *testing.T) {
	server := &Server{authService: serviceapi.NewAuthService(&middlewareAuthStore{}, serviceapi.AuthConfig{})}
	middleware := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/projects", nil)
	response := httptest.NewRecorder()

	middleware.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "unauthorized" {
		t.Fatalf("expected code unauthorized, got %s", envelope.Code)
	}
}

func TestAuthMiddlewareAllowsValidToken(t *testing.T) {
	accessToken := "valid-access-token"
	hash := sha256.Sum256([]byte(accessToken))
	accessHash := hex.EncodeToString(hash[:])

	companyID := int64(10)
	store := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			accessHash: {
				Session: domain.AuthSession{
					ID:               7,
					AccessExpiresAt:  time.Now().Add(15 * time.Minute),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				},
				User: domain.AuthUser{
					ID:        44,
					CompanyID: &companyID,
					Role:      domain.RoleViewer,
					Status:    domain.UserStatusActive,
				},
			},
		},
	}

	server := &Server{authService: serviceapi.NewAuthService(store, serviceapi.AuthConfig{})}
	middleware := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/companies/10/projects", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()

	middleware.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", response.Code, response.Body.String())
	}
}

func TestAuthMiddlewareTenantEscapeDenied(t *testing.T) {
	accessToken := "tenant-access-token"
	hash := sha256.Sum256([]byte(accessToken))
	accessHash := hex.EncodeToString(hash[:])

	companyID := int64(1)
	store := &middlewareAuthStore{
		sessionByAccess: map[string]domain.AuthSessionUser{
			accessHash: {
				Session: domain.AuthSession{
					ID:               11,
					AccessExpiresAt:  time.Now().Add(10 * time.Minute),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				},
				User: domain.AuthUser{
					ID:        77,
					CompanyID: &companyID,
					Role:      domain.RoleCompanyAdmin,
					Status:    domain.UserStatusActive,
				},
			},
		},
	}

	server := &Server{authService: serviceapi.NewAuthService(store, serviceapi.AuthConfig{})}
	middleware := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/companies/2/projects", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()

	middleware.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "tenant_scope_required" {
		t.Fatalf("expected tenant_scope_required, got %s", envelope.Code)
	}
}

func TestEvaluateAccessPolicyRBACMatrix(t *testing.T) {
	companyID := int64(3)
	otherCompanyID := int64(5)

	testCases := []struct {
		name       string
		path       string
		method     string
		auth       domain.AuthContext
		expectDeny bool
		expectCode string
	}{
		{
			name:   "super admin can access admin endpoint",
			path:   "/api/v1/admin/registration-requests",
			method: http.MethodGet,
			auth: domain.AuthContext{
				Role: domain.RoleSuperAdmin,
			},
			expectDeny: false,
		},
		{
			name:   "company admin denied on admin endpoint",
			path:   "/api/v1/admin/registration-requests",
			method: http.MethodGet,
			auth: domain.AuthContext{
				Role:      domain.RoleCompanyAdmin,
				CompanyID: &companyID,
			},
			expectDeny: true,
			expectCode: "forbidden",
		},
		{
			name:   "viewer is read only",
			path:   "/api/v1/companies/3/projects",
			method: http.MethodPost,
			auth: domain.AuthContext{
				Role:      domain.RoleViewer,
				CompanyID: &companyID,
			},
			expectDeny: true,
			expectCode: "forbidden",
		},
		{
			name:   "viewer can read own tenant",
			path:   "/api/v1/companies/3/projects",
			method: http.MethodGet,
			auth: domain.AuthContext{
				Role:      domain.RoleViewer,
				CompanyID: &companyID,
			},
			expectDeny: false,
		},
		{
			name:   "company admin denied on unscoped companies collection",
			path:   "/api/v1/companies",
			method: http.MethodGet,
			auth: domain.AuthContext{
				Role:      domain.RoleCompanyAdmin,
				CompanyID: &companyID,
			},
			expectDeny: true,
			expectCode: "forbidden",
		},
		{
			name:   "viewer denied on unscoped companies collection",
			path:   "/api/v1/companies",
			method: http.MethodGet,
			auth: domain.AuthContext{
				Role:      domain.RoleViewer,
				CompanyID: &companyID,
			},
			expectDeny: true,
			expectCode: "forbidden",
		},
		{
			name:   "company admin tenant escape denied",
			path:   "/api/v1/companies/5/projects",
			method: http.MethodGet,
			auth: domain.AuthContext{
				Role:      domain.RoleCompanyAdmin,
				CompanyID: &companyID,
			},
			expectDeny: true,
			expectCode: "tenant_scope_required",
		},
		{
			name:   "company admin cannot delete company",
			path:   "/api/v1/companies/3",
			method: http.MethodDelete,
			auth: domain.AuthContext{
				Role:      domain.RoleCompanyAdmin,
				CompanyID: &companyID,
			},
			expectDeny: true,
			expectCode: "forbidden",
		},
		{
			name:   "super admin can access cross company",
			path:   "/api/v1/companies/5/projects",
			method: http.MethodGet,
			auth: domain.AuthContext{
				Role:      domain.RoleSuperAdmin,
				CompanyID: &otherCompanyID,
			},
			expectDeny: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			code, _, denied := evaluateAccessPolicy(request, testCase.auth)
			if denied != testCase.expectDeny {
				t.Fatalf("denied=%t expected=%t", denied, testCase.expectDeny)
			}
			if testCase.expectCode != "" && code != testCase.expectCode {
				t.Fatalf("code=%s expected=%s", code, testCase.expectCode)
			}
		})
	}
}
