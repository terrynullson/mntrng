package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
	serviceapi "github.com/terrynullson/hls_mntrng/internal/service/api"
	"golang.org/x/crypto/bcrypt"
)

type authHandlerStore struct {
	userByIdentity  map[string]domain.UserRecord
	userByID        map[int64]domain.UserRecord
	sessionByAccess map[string]domain.AuthSessionUser
	sessionByRef    map[string]domain.AuthSessionUser
	nextSessionID   int64
}

func (s *authHandlerStore) GetUserByLoginOrEmail(ctx context.Context, identity string) (domain.UserRecord, error) {
	item, ok := s.userByIdentity[strings.ToLower(strings.TrimSpace(identity))]
	if !ok {
		return domain.UserRecord{}, domain.ErrUserNotFound
	}
	return item, nil
}

func (s *authHandlerStore) GetUserByID(ctx context.Context, userID int64) (domain.UserRecord, error) {
	item, ok := s.userByID[userID]
	if !ok {
		return domain.UserRecord{}, domain.ErrUserNotFound
	}
	return item, nil
}

func (s *authHandlerStore) CreateSession(
	ctx context.Context,
	user domain.AuthUser,
	accessTokenHash string,
	refreshTokenHash string,
	accessExpiresAt time.Time,
	refreshExpiresAt time.Time,
) (domain.AuthSession, error) {
	s.nextSessionID++
	session := domain.AuthSession{
		ID:               s.nextSessionID,
		UserID:           user.ID,
		CompanyID:        user.CompanyID,
		AccessTokenHash:  accessTokenHash,
		RefreshTokenHash: refreshTokenHash,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}
	sessionUser := domain.AuthSessionUser{Session: session, User: user}
	s.sessionByAccess[accessTokenHash] = sessionUser
	s.sessionByRef[refreshTokenHash] = sessionUser
	return session, nil
}

func (s *authHandlerStore) GetSessionByAccessTokenHash(ctx context.Context, accessTokenHash string) (domain.AuthSessionUser, error) {
	item, ok := s.sessionByAccess[accessTokenHash]
	if !ok {
		return domain.AuthSessionUser{}, domain.ErrSessionNotFound
	}
	return item, nil
}

func (s *authHandlerStore) GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (domain.AuthSessionUser, error) {
	item, ok := s.sessionByRef[refreshTokenHash]
	if !ok {
		return domain.AuthSessionUser{}, domain.ErrSessionNotFound
	}
	return item, nil
}

func (s *authHandlerStore) RotateSessionTokens(
	ctx context.Context,
	sessionID int64,
	accessTokenHash string,
	refreshTokenHash string,
	accessExpiresAt time.Time,
	refreshExpiresAt time.Time,
) error {
	var current domain.AuthSessionUser
	found := false
	var oldAccessHash string
	var oldRefreshHash string
	for hash, item := range s.sessionByAccess {
		if item.Session.ID == sessionID {
			current = item
			oldAccessHash = hash
			found = true
			break
		}
	}
	if !found {
		return domain.ErrSessionNotFound
	}
	for hash, item := range s.sessionByRef {
		if item.Session.ID == sessionID {
			oldRefreshHash = hash
			break
		}
	}
	delete(s.sessionByAccess, oldAccessHash)
	delete(s.sessionByRef, oldRefreshHash)

	current.Session.AccessTokenHash = accessTokenHash
	current.Session.RefreshTokenHash = refreshTokenHash
	current.Session.AccessExpiresAt = accessExpiresAt
	current.Session.RefreshExpiresAt = refreshExpiresAt
	s.sessionByAccess[accessTokenHash] = current
	s.sessionByRef[refreshTokenHash] = current
	return nil
}

func (s *authHandlerStore) RevokeSessionByID(ctx context.Context, sessionID int64) error {
	return nil
}

func (s *authHandlerStore) RevokeSessionByRefreshToken(ctx context.Context, userID int64, refreshTokenHash string) error {
	return nil
}

func (s *authHandlerStore) HasPendingRegistration(ctx context.Context, identity string) (bool, error) {
	return false, nil
}

func (s *authHandlerStore) UpsertTelegramLink(ctx context.Context, user domain.AuthUser, telegramUserID int64, telegramUsername *string) error {
	return nil
}

func (s *authHandlerStore) GetUserByTelegramUserID(ctx context.Context, telegramUserID int64) (domain.UserRecord, error) {
	return domain.UserRecord{}, domain.ErrUserNotFound
}

func TestHandleLoginSetsHttpOnlyAuthCookies(t *testing.T) {
	t.Setenv("AUTH_COOKIE_SECURE", "false")
	password := "StrongPass123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := domain.UserRecord{
		AuthUser: domain.AuthUser{
			ID:     101,
			Email:  "admin@example.com",
			Login:  "admin",
			Role:   domain.RoleSuperAdmin,
			Status: domain.UserStatusActive,
		},
		PasswordHash: string(hash),
	}
	store := &authHandlerStore{
		userByIdentity: map[string]domain.UserRecord{
			"admin":             user,
			"admin@example.com": user,
		},
		userByID:        map[int64]domain.UserRecord{user.ID: user},
		sessionByAccess: make(map[string]domain.AuthSessionUser),
		sessionByRef:    make(map[string]domain.AuthSessionUser),
	}
	srv := &Server{
		authService:    serviceapi.NewAuthService(store, serviceapi.AuthConfig{}),
		authAccessTTL:  15 * time.Minute,
		authRefreshTTL: 30 * 24 * time.Hour,
	}

	body := []byte(`{"login_or_email":"admin","password":"StrongPass123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	accessCookie := findCookieByName(cookies, defaultAccessCookieName)
	refreshCookie := findCookieByName(cookies, defaultRefreshCookieName)
	if accessCookie == nil || refreshCookie == nil {
		t.Fatalf("expected auth cookies to be set, cookies=%v", cookies)
	}
	if !accessCookie.HttpOnly || !refreshCookie.HttpOnly {
		t.Fatalf("expected HttpOnly cookies, got access=%t refresh=%t", accessCookie.HttpOnly, refreshCookie.HttpOnly)
	}
}

func TestHandleRefreshAcceptsRefreshTokenFromCookie(t *testing.T) {
	t.Setenv("AUTH_COOKIE_SECURE", "false")
	refreshToken := "existing-refresh-token"
	refreshHash := hashForTest(refreshToken)
	companyID := int64(7)
	user := domain.AuthUser{
		ID:        11,
		CompanyID: &companyID,
		Email:     "viewer@example.com",
		Login:     "viewer",
		Role:      domain.RoleViewer,
		Status:    domain.UserStatusActive,
	}
	store := &authHandlerStore{
		userByIdentity: map[string]domain.UserRecord{},
		userByID:       map[int64]domain.UserRecord{user.ID: {AuthUser: user}},
		sessionByAccess: map[string]domain.AuthSessionUser{
			hashForTest("old-access"): {
				Session: domain.AuthSession{
					ID:               77,
					UserID:           user.ID,
					CompanyID:        user.CompanyID,
					AccessTokenHash:  hashForTest("old-access"),
					RefreshTokenHash: refreshHash,
					AccessExpiresAt:  time.Now().UTC().Add(-1 * time.Minute),
					RefreshExpiresAt: time.Now().UTC().Add(24 * time.Hour),
				},
				User: user,
			},
		},
		sessionByRef: map[string]domain.AuthSessionUser{
			refreshHash: {
				Session: domain.AuthSession{
					ID:               77,
					UserID:           user.ID,
					CompanyID:        user.CompanyID,
					AccessTokenHash:  hashForTest("old-access"),
					RefreshTokenHash: refreshHash,
					AccessExpiresAt:  time.Now().UTC().Add(-1 * time.Minute),
					RefreshExpiresAt: time.Now().UTC().Add(24 * time.Hour),
				},
				User: user,
			},
		},
		nextSessionID: 77,
	}
	srv := &Server{
		authService: serviceapi.NewAuthService(store, serviceapi.AuthConfig{
			AccessTTL:  15 * time.Minute,
			RefreshTTL: 30 * 24 * time.Hour,
		}),
		authAccessTTL:  15 * time.Minute,
		authRefreshTTL: 30 * 24 * time.Hour,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  defaultRefreshCookieName,
		Value: refreshToken,
	})
	rec := httptest.NewRecorder()
	srv.handleRefresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response domain.AuthTokensResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	if strings.TrimSpace(response.AccessToken) == "" || strings.TrimSpace(response.RefreshToken) == "" {
		t.Fatalf("expected rotated tokens in response")
	}

	cookies := rec.Result().Cookies()
	refreshCookie := findCookieByName(cookies, defaultRefreshCookieName)
	if refreshCookie == nil {
		t.Fatalf("expected refresh cookie in response")
	}
	if refreshCookie.Value != response.RefreshToken {
		t.Fatalf("cookie refresh token mismatch")
	}
}

func hashForTest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func findCookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
