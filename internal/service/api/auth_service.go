package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type AuthStore interface {
	GetUserByLoginOrEmail(ctx context.Context, identity string) (domain.UserRecord, error)
	GetUserByID(ctx context.Context, userID int64) (domain.UserRecord, error)
	CreateSession(ctx context.Context, user domain.AuthUser, accessTokenHash string, refreshTokenHash string, accessExpiresAt time.Time, refreshExpiresAt time.Time) (domain.AuthSession, error)
	GetSessionByAccessTokenHash(ctx context.Context, accessTokenHash string) (domain.AuthSessionUser, error)
	GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (domain.AuthSessionUser, error)
	RotateSessionTokens(ctx context.Context, sessionID int64, accessTokenHash string, refreshTokenHash string, accessExpiresAt time.Time, refreshExpiresAt time.Time) error
	RevokeSessionByID(ctx context.Context, sessionID int64) error
	RevokeSessionByRefreshToken(ctx context.Context, userID int64, refreshTokenHash string) error
	HasPendingRegistration(ctx context.Context, identity string) (bool, error)
	UpsertTelegramLink(ctx context.Context, user domain.AuthUser, telegramUserID int64, telegramUsername *string) error
	GetUserByTelegramUserID(ctx context.Context, telegramUserID int64) (domain.UserRecord, error)
}

type AuthConfig struct {
	AccessTTL          time.Duration
	RefreshTTL         time.Duration
	TelegramBotToken   string
	TelegramAuthMaxAge time.Duration
	AccessTokenBytes   int
	RefreshTokenBytes  int
}

type AuthService struct {
	store AuthStore
	cfg   AuthConfig
	nowFn func() time.Time
}

func NewAuthService(store AuthStore, cfg AuthConfig) *AuthService {
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	if cfg.RefreshTTL <= 0 {
		cfg.RefreshTTL = 30 * 24 * time.Hour
	}
	if cfg.TelegramAuthMaxAge <= 0 {
		cfg.TelegramAuthMaxAge = 10 * time.Minute
	}
	if cfg.AccessTokenBytes < 24 {
		cfg.AccessTokenBytes = 32
	}
	if cfg.RefreshTokenBytes < 24 {
		cfg.RefreshTokenBytes = 48
	}

	return &AuthService{
		store: store,
		cfg:   cfg,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *AuthService) Login(ctx context.Context, request domain.LoginRequest) (domain.AuthTokensResponse, error) {
	identity := strings.TrimSpace(request.LoginOrEmail)
	if identity == "" {
		return domain.AuthTokensResponse{}, NewValidationError("login_or_email is required", map[string]interface{}{"field": "login_or_email"})
	}
	if request.Password == "" {
		return domain.AuthTokensResponse{}, NewValidationError("password is required", map[string]interface{}{"field": "password"})
	}

	userRecord, err := s.store.GetUserByLoginOrEmail(ctx, identity)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			pending, pendingErr := s.store.HasPendingRegistration(ctx, identity)
			if pendingErr != nil {
				return domain.AuthTokensResponse{}, NewInternalError()
			}
			if pending {
				return domain.AuthTokensResponse{}, NewUnauthorizedError("registration request is pending approval", map[string]interface{}{})
			}
			return domain.AuthTokensResponse{}, NewUnauthorizedError("invalid credentials", map[string]interface{}{})
		}
		return domain.AuthTokensResponse{}, NewInternalError()
	}

	if userRecord.Status != domain.UserStatusActive {
		return domain.AuthTokensResponse{}, NewUnauthorizedError("user is not active", map[string]interface{}{"status": userRecord.Status})
	}
	if bcrypt.CompareHashAndPassword([]byte(userRecord.PasswordHash), []byte(request.Password)) != nil {
		return domain.AuthTokensResponse{}, NewUnauthorizedError("invalid credentials", map[string]interface{}{})
	}

	return s.issueTokens(ctx, userRecord.AuthUser)
}

func (s *AuthService) Refresh(ctx context.Context, request domain.RefreshRequest) (domain.AuthTokensResponse, error) {
	refreshToken := strings.TrimSpace(request.RefreshToken)
	if refreshToken == "" {
		return domain.AuthTokensResponse{}, NewValidationError("refresh_token is required", map[string]interface{}{"field": "refresh_token"})
	}

	hash := hashToken(refreshToken)
	sessionUser, err := s.store.GetSessionByRefreshTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.AuthTokensResponse{}, NewUnauthorizedError("invalid refresh token", map[string]interface{}{})
		}
		return domain.AuthTokensResponse{}, NewInternalError()
	}

	if serviceErr := s.validateRefreshSession(sessionUser); serviceErr != nil {
		return domain.AuthTokensResponse{}, serviceErr
	}

	tokens, issueErr := s.buildTokenPair(sessionUser.User)
	if issueErr != nil {
		return domain.AuthTokensResponse{}, NewInternalError()
	}

	if err := s.store.RotateSessionTokens(
		ctx,
		sessionUser.Session.ID,
		hashToken(tokens.AccessToken),
		hashToken(tokens.RefreshToken),
		tokens.AccessExpiresAt,
		tokens.RefreshExpiresAt,
	); err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.AuthTokensResponse{}, NewUnauthorizedError("refresh session is not available", map[string]interface{}{})
		}
		return domain.AuthTokensResponse{}, NewInternalError()
	}

	return toAuthTokensResponse(tokens), nil
}

func (s *AuthService) Logout(ctx context.Context, authContext domain.AuthContext, request domain.LogoutRequest) error {
	if request.RefreshToken != nil {
		refreshToken := strings.TrimSpace(*request.RefreshToken)
		if refreshToken == "" {
			return NewValidationError("refresh_token must not be empty", map[string]interface{}{"field": "refresh_token"})
		}
		err := s.store.RevokeSessionByRefreshToken(ctx, authContext.UserID, hashToken(refreshToken))
		if err != nil {
			if errors.Is(err, domain.ErrSessionNotFound) {
				return NewUnauthorizedError("refresh session is not available", map[string]interface{}{})
			}
			return NewInternalError()
		}
		return nil
	}

	err := s.store.RevokeSessionByID(ctx, authContext.SessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return NewUnauthorizedError("session is not available", map[string]interface{}{})
		}
		return NewInternalError()
	}
	return nil
}

func (s *AuthService) Me(ctx context.Context, userID int64) (domain.AuthUser, error) {
	userRecord, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.AuthUser{}, NewUnauthorizedError("user is not available", map[string]interface{}{})
		}
		return domain.AuthUser{}, NewInternalError()
	}
	if userRecord.Status != domain.UserStatusActive {
		return domain.AuthUser{}, NewUnauthorizedError("user is not active", map[string]interface{}{"status": userRecord.Status})
	}
	return userRecord.AuthUser, nil
}

func (s *AuthService) AuthenticateAccessToken(ctx context.Context, accessToken string) (domain.AuthContext, error) {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return domain.AuthContext{}, NewUnauthorizedError("access token is required", map[string]interface{}{})
	}

	sessionUser, err := s.store.GetSessionByAccessTokenHash(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.AuthContext{}, NewUnauthorizedError("invalid access token", map[string]interface{}{})
		}
		return domain.AuthContext{}, NewInternalError()
	}

	if serviceErr := s.validateAccessSession(sessionUser); serviceErr != nil {
		return domain.AuthContext{}, serviceErr
	}

	return domain.AuthContext{
		UserID:    sessionUser.User.ID,
		CompanyID: sessionUser.User.CompanyID,
		Role:      sessionUser.User.Role,
		SessionID: sessionUser.Session.ID,
	}, nil
}

func (s *AuthService) LinkTelegram(ctx context.Context, userID int64, payload map[string]string) error {
	userRecord, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return NewUnauthorizedError("user is not available", map[string]interface{}{})
		}
		return NewInternalError()
	}
	if userRecord.Status != domain.UserStatusActive {
		return NewForbiddenError("telegram linking is allowed only for active users", map[string]interface{}{"status": userRecord.Status})
	}

	telegramUserID, telegramUsername, verifyErr := verifyTelegramPayload(payload, s.cfg.TelegramBotToken, s.nowFn(), s.cfg.TelegramAuthMaxAge)
	if verifyErr != nil {
		return NewUnauthorizedError("invalid telegram auth payload", map[string]interface{}{"reason": verifyErr.Error()})
	}

	if err := s.store.UpsertTelegramLink(ctx, userRecord.AuthUser, telegramUserID, telegramUsername); err != nil {
		if errors.Is(err, domain.ErrTelegramLinkConflict) {
			return NewConflictError("telegram account is already linked", map[string]interface{}{})
		}
		return NewInternalError()
	}
	return nil
}

func (s *AuthService) TelegramLogin(ctx context.Context, payload map[string]string) (domain.AuthTokensResponse, error) {
	telegramUserID, _, verifyErr := verifyTelegramPayload(payload, s.cfg.TelegramBotToken, s.nowFn(), s.cfg.TelegramAuthMaxAge)
	if verifyErr != nil {
		return domain.AuthTokensResponse{}, NewUnauthorizedError("invalid telegram auth payload", map[string]interface{}{"reason": verifyErr.Error()})
	}

	userRecord, err := s.store.GetUserByTelegramUserID(ctx, telegramUserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrTelegramLinkNotFound) {
			return domain.AuthTokensResponse{}, NewUnauthorizedError("telegram account is not linked", map[string]interface{}{})
		}
		return domain.AuthTokensResponse{}, NewInternalError()
	}

	if userRecord.Status != domain.UserStatusActive {
		return domain.AuthTokensResponse{}, NewForbiddenError("telegram login is allowed only for active users", map[string]interface{}{"status": userRecord.Status})
	}

	return s.issueTokens(ctx, userRecord.AuthUser)
}

func (s *AuthService) issueTokens(ctx context.Context, user domain.AuthUser) (domain.AuthTokensResponse, error) {
	tokens, err := s.buildTokenPair(user)
	if err != nil {
		return domain.AuthTokensResponse{}, NewInternalError()
	}

	_, err = s.store.CreateSession(
		ctx,
		user,
		hashToken(tokens.AccessToken),
		hashToken(tokens.RefreshToken),
		tokens.AccessExpiresAt,
		tokens.RefreshExpiresAt,
	)
	if err != nil {
		return domain.AuthTokensResponse{}, NewInternalError()
	}

	return toAuthTokensResponse(tokens), nil
}

func (s *AuthService) buildTokenPair(user domain.AuthUser) (issuedTokens, error) {
	now := s.nowFn()
	accessToken, err := generateToken(s.cfg.AccessTokenBytes)
	if err != nil {
		return issuedTokens{}, err
	}
	refreshToken, err := generateToken(s.cfg.RefreshTokenBytes)
	if err != nil {
		return issuedTokens{}, err
	}
	return issuedTokens{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  now.Add(s.cfg.AccessTTL),
		RefreshExpiresAt: now.Add(s.cfg.RefreshTTL),
		User:             user,
	}, nil
}

func (s *AuthService) validateAccessSession(sessionUser domain.AuthSessionUser) *ServiceError {
	now := s.nowFn()
	if sessionUser.User.Status != domain.UserStatusActive {
		return NewUnauthorizedError("user is not active", map[string]interface{}{"status": sessionUser.User.Status})
	}
	if sessionUser.Session.RevokedAt != nil {
		return NewUnauthorizedError("session is revoked", map[string]interface{}{})
	}
	if now.After(sessionUser.Session.AccessExpiresAt) {
		return NewUnauthorizedError("access token is expired", map[string]interface{}{})
	}
	return nil
}

func (s *AuthService) validateRefreshSession(sessionUser domain.AuthSessionUser) *ServiceError {
	now := s.nowFn()
	if sessionUser.User.Status != domain.UserStatusActive {
		return NewUnauthorizedError("user is not active", map[string]interface{}{"status": sessionUser.User.Status})
	}
	if sessionUser.Session.RevokedAt != nil {
		return NewUnauthorizedError("session is revoked", map[string]interface{}{})
	}
	if now.After(sessionUser.Session.RefreshExpiresAt) {
		return NewUnauthorizedError("session is expired", map[string]interface{}{})
	}
	return nil
}

type issuedTokens struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	User             domain.AuthUser
}

func toAuthTokensResponse(tokens issuedTokens) domain.AuthTokensResponse {
	expiresIn := int64(time.Until(tokens.AccessExpiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}
	return domain.AuthTokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		User:         tokens.User,
	}
}

func generateToken(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
