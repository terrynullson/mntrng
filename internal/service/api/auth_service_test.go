package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type authServiceStoreStub struct {
	getUserByLoginOrEmailFn       func(ctx context.Context, identity string) (domain.UserRecord, error)
	getUserByIDFn                 func(ctx context.Context, userID int64) (domain.UserRecord, error)
	createSessionFn               func(ctx context.Context, user domain.AuthUser, accessTokenHash string, refreshTokenHash string, accessExpiresAt time.Time, refreshExpiresAt time.Time) (domain.AuthSession, error)
	upsertTelegramLinkFn          func(ctx context.Context, user domain.AuthUser, telegramUserID int64, telegramUsername *string) error
	getUserByTelegramUserIDFn     func(ctx context.Context, telegramUserID int64) (domain.UserRecord, error)
	hasPendingRegistrationFn      func(ctx context.Context, identity string) (bool, error)
	revokeSessionByIDFn           func(ctx context.Context, sessionID int64) error
	revokeSessionByRefreshTokenFn func(ctx context.Context, userID int64, refreshTokenHash string) error
}

func (s *authServiceStoreStub) GetUserByLoginOrEmail(ctx context.Context, identity string) (domain.UserRecord, error) {
	if s.getUserByLoginOrEmailFn != nil {
		return s.getUserByLoginOrEmailFn(ctx, identity)
	}
	return domain.UserRecord{}, domain.ErrUserNotFound
}

func (s *authServiceStoreStub) GetUserByID(ctx context.Context, userID int64) (domain.UserRecord, error) {
	if s.getUserByIDFn != nil {
		return s.getUserByIDFn(ctx, userID)
	}
	return domain.UserRecord{}, domain.ErrUserNotFound
}

func (s *authServiceStoreStub) CreateSession(ctx context.Context, user domain.AuthUser, accessTokenHash string, refreshTokenHash string, accessExpiresAt time.Time, refreshExpiresAt time.Time) (domain.AuthSession, error) {
	if s.createSessionFn != nil {
		return s.createSessionFn(ctx, user, accessTokenHash, refreshTokenHash, accessExpiresAt, refreshExpiresAt)
	}
	return domain.AuthSession{}, nil
}

func (s *authServiceStoreStub) GetSessionByAccessTokenHash(ctx context.Context, accessTokenHash string) (domain.AuthSessionUser, error) {
	return domain.AuthSessionUser{}, domain.ErrSessionNotFound
}

func (s *authServiceStoreStub) GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (domain.AuthSessionUser, error) {
	return domain.AuthSessionUser{}, domain.ErrSessionNotFound
}

func (s *authServiceStoreStub) RotateSessionTokens(ctx context.Context, sessionID int64, accessTokenHash string, refreshTokenHash string, accessExpiresAt time.Time, refreshExpiresAt time.Time) error {
	return nil
}

func (s *authServiceStoreStub) RevokeSessionByID(ctx context.Context, sessionID int64) error {
	if s.revokeSessionByIDFn != nil {
		return s.revokeSessionByIDFn(ctx, sessionID)
	}
	return nil
}

func (s *authServiceStoreStub) RevokeSessionByRefreshToken(ctx context.Context, userID int64, refreshTokenHash string) error {
	if s.revokeSessionByRefreshTokenFn != nil {
		return s.revokeSessionByRefreshTokenFn(ctx, userID, refreshTokenHash)
	}
	return nil
}

func (s *authServiceStoreStub) HasPendingRegistration(ctx context.Context, identity string) (bool, error) {
	if s.hasPendingRegistrationFn != nil {
		return s.hasPendingRegistrationFn(ctx, identity)
	}
	return false, nil
}

func (s *authServiceStoreStub) UpsertTelegramLink(ctx context.Context, user domain.AuthUser, telegramUserID int64, telegramUsername *string) error {
	if s.upsertTelegramLinkFn != nil {
		return s.upsertTelegramLinkFn(ctx, user, telegramUserID, telegramUsername)
	}
	return nil
}

func (s *authServiceStoreStub) GetUserByTelegramUserID(ctx context.Context, telegramUserID int64) (domain.UserRecord, error) {
	if s.getUserByTelegramUserIDFn != nil {
		return s.getUserByTelegramUserIDFn(ctx, telegramUserID)
	}
	return domain.UserRecord{}, domain.ErrUserNotFound
}

func TestLoginPendingRegistrationDenied(t *testing.T) {
	store := &authServiceStoreStub{
		getUserByLoginOrEmailFn: func(ctx context.Context, identity string) (domain.UserRecord, error) {
			return domain.UserRecord{}, domain.ErrUserNotFound
		},
		hasPendingRegistrationFn: func(ctx context.Context, identity string) (bool, error) {
			return true, nil
		},
	}

	service := NewAuthService(store, AuthConfig{})
	_, err := service.Login(context.Background(), domain.LoginRequest{LoginOrEmail: "pending@example.com", Password: "secret123"})
	if err == nil {
		t.Fatal("expected error")
	}

	serviceErr, ok := AsServiceError(err)
	if !ok {
		t.Fatalf("expected service error, got %T", err)
	}
	if serviceErr.Code != "unauthorized" {
		t.Fatalf("expected unauthorized code, got %s", serviceErr.Code)
	}
}

func TestLoginDisabledUserDenied(t *testing.T) {
	password := "secret123"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	store := &authServiceStoreStub{
		getUserByLoginOrEmailFn: func(ctx context.Context, identity string) (domain.UserRecord, error) {
			return domain.UserRecord{
				AuthUser: domain.AuthUser{
					ID:     10,
					Login:  "disabled-user",
					Email:  "disabled@example.com",
					Role:   domain.RoleViewer,
					Status: domain.UserStatusDisabled,
				},
				PasswordHash: string(passwordHash),
			}, nil
		},
	}

	service := NewAuthService(store, AuthConfig{})
	_, err = service.Login(context.Background(), domain.LoginRequest{LoginOrEmail: "disabled-user", Password: password})
	if err == nil {
		t.Fatal("expected error")
	}

	serviceErr, ok := AsServiceError(err)
	if !ok {
		t.Fatalf("expected service error, got %T", err)
	}
	if serviceErr.Code != "unauthorized" {
		t.Fatalf("expected unauthorized code, got %s", serviceErr.Code)
	}
}

func TestLoginRejectedOrUnknownIdentityDenied(t *testing.T) {
	store := &authServiceStoreStub{
		getUserByLoginOrEmailFn: func(ctx context.Context, identity string) (domain.UserRecord, error) {
			return domain.UserRecord{}, domain.ErrUserNotFound
		},
		hasPendingRegistrationFn: func(ctx context.Context, identity string) (bool, error) {
			return false, nil
		},
	}

	service := NewAuthService(store, AuthConfig{})
	_, err := service.Login(context.Background(), domain.LoginRequest{LoginOrEmail: "rejected@example.com", Password: "secret123"})
	if err == nil {
		t.Fatal("expected error")
	}

	serviceErr, ok := AsServiceError(err)
	if !ok {
		t.Fatalf("expected service error, got %T", err)
	}
	if serviceErr.Code != "unauthorized" {
		t.Fatalf("expected unauthorized code, got %s", serviceErr.Code)
	}
}

func TestLogoutRevokeBehavior(t *testing.T) {
	var revokedSessionID int64
	var revokedUserID int64
	var revokedRefreshHash string

	store := &authServiceStoreStub{
		revokeSessionByIDFn: func(ctx context.Context, sessionID int64) error {
			revokedSessionID = sessionID
			return nil
		},
		revokeSessionByRefreshTokenFn: func(ctx context.Context, userID int64, refreshTokenHash string) error {
			revokedUserID = userID
			revokedRefreshHash = refreshTokenHash
			return nil
		},
	}

	service := NewAuthService(store, AuthConfig{})
	authContext := domain.AuthContext{UserID: 99, SessionID: 42}

	if err := service.Logout(context.Background(), authContext, domain.LogoutRequest{}); err != nil {
		t.Fatalf("logout by session id failed: %v", err)
	}
	if revokedSessionID != 42 {
		t.Fatalf("expected revoked session id 42, got %d", revokedSessionID)
	}

	refreshToken := "refresh-token"
	if err := service.Logout(context.Background(), authContext, domain.LogoutRequest{RefreshToken: &refreshToken}); err != nil {
		t.Fatalf("logout by refresh token failed: %v", err)
	}
	if revokedUserID != 99 {
		t.Fatalf("expected revoked user id 99, got %d", revokedUserID)
	}

	hash := sha256.Sum256([]byte(refreshToken))
	expectedHash := hex.EncodeToString(hash[:])
	if revokedRefreshHash != expectedHash {
		t.Fatalf("expected hashed refresh token %s, got %s", expectedHash, revokedRefreshHash)
	}
}
