package api

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

func TestTelegramLoginApprovedActiveSuccess(t *testing.T) {
	botToken := "123456:telegram_token"
	now := time.Now().UTC()
	companyID := int64(3)
	telegramUserID := int64(777)
	payload := validTelegramPayload(botToken, now, telegramUserID, "alice")

	store := &authServiceStoreStub{
		getUserByTelegramUserIDFn: func(ctx context.Context, requestedTelegramUserID int64) (domain.UserRecord, error) {
			if requestedTelegramUserID != telegramUserID {
				t.Fatalf("unexpected telegram_user_id=%d", requestedTelegramUserID)
			}
			return domain.UserRecord{
				AuthUser: domain.AuthUser{
					ID:        91,
					CompanyID: &companyID,
					Login:     "alice",
					Email:     "alice@example.com",
					Role:      domain.RoleViewer,
					Status:    domain.UserStatusActive,
				},
			}, nil
		},
		createSessionFn: func(ctx context.Context, user domain.AuthUser, accessTokenHash string, refreshTokenHash string, accessExpiresAt time.Time, refreshExpiresAt time.Time) (domain.AuthSession, error) {
			return domain.AuthSession{ID: 10, UserID: user.ID}, nil
		},
	}

	service := NewAuthService(store, AuthConfig{
		TelegramBotToken:   botToken,
		TelegramAuthMaxAge: 10 * time.Minute,
	})
	service.nowFn = func() time.Time { return now }

	result, err := service.TelegramLogin(context.Background(), payload)
	if err != nil {
		t.Fatalf("telegram login failed: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("expected tokens to be issued, got %+v", result)
	}
	if result.User.ID != 91 || result.User.Status != domain.UserStatusActive {
		t.Fatalf("unexpected user in response: %+v", result.User)
	}
}

func TestTelegramLoginDeniedCases(t *testing.T) {
	botToken := "123456:telegram_token"
	now := time.Now().UTC()
	payload := validTelegramPayload(botToken, now, 777, "alice")

	t.Run("pending_or_rejected_unlinked_denied", func(t *testing.T) {
		store := &authServiceStoreStub{
			getUserByTelegramUserIDFn: func(ctx context.Context, telegramUserID int64) (domain.UserRecord, error) {
				return domain.UserRecord{}, domain.ErrTelegramLinkNotFound
			},
		}
		service := NewAuthService(store, AuthConfig{
			TelegramBotToken:   botToken,
			TelegramAuthMaxAge: 10 * time.Minute,
		})
		service.nowFn = func() time.Time { return now }

		_, err := service.TelegramLogin(context.Background(), payload)
		assertServiceErrorCode(t, err, "unauthorized")
	})

	t.Run("disabled_user_denied", func(t *testing.T) {
		store := &authServiceStoreStub{
			getUserByTelegramUserIDFn: func(ctx context.Context, telegramUserID int64) (domain.UserRecord, error) {
				return domain.UserRecord{
					AuthUser: domain.AuthUser{
						ID:     11,
						Login:  "disabled",
						Status: domain.UserStatusDisabled,
						Role:   domain.RoleViewer,
					},
				}, nil
			},
		}
		service := NewAuthService(store, AuthConfig{
			TelegramBotToken:   botToken,
			TelegramAuthMaxAge: 10 * time.Minute,
		})
		service.nowFn = func() time.Time { return now }

		_, err := service.TelegramLogin(context.Background(), payload)
		assertServiceErrorCode(t, err, "forbidden")
	})
}

func TestTelegramLoginInvalidSignatureNoSecretLeak(t *testing.T) {
	botToken := "123456:telegram_secret_token"
	now := time.Now().UTC()
	payload := map[string]string{
		"id":        "777",
		"auth_date": fmt.Sprintf("%d", now.Unix()),
		"hash":      "invalid_hash",
	}

	service := NewAuthService(&authServiceStoreStub{}, AuthConfig{
		TelegramBotToken:   botToken,
		TelegramAuthMaxAge: 10 * time.Minute,
	})
	service.nowFn = func() time.Time { return now }

	_, err := service.TelegramLogin(context.Background(), payload)
	serviceErr := assertServiceErrorCode(t, err, "unauthorized")
	reason := fmt.Sprintf("%v", serviceErr.Details["reason"])
	if !strings.Contains(reason, "hash_mismatch") {
		t.Fatalf("unexpected reason=%s", reason)
	}
	if strings.Contains(reason, botToken) {
		t.Fatalf("reason must not contain bot token")
	}
}

func TestLinkTelegramGuardsAndRelink(t *testing.T) {
	botToken := "123456:telegram_token"
	now := time.Now().UTC()
	payload := validTelegramPayload(botToken, now, 777, "alice")
	companyID := int64(8)
	var linkCalls int
	var linkedTelegramUserID int64
	var linkedUsername string

	store := &authServiceStoreStub{
		getUserByIDFn: func(ctx context.Context, userID int64) (domain.UserRecord, error) {
			if userID == 300 {
				return domain.UserRecord{
					AuthUser: domain.AuthUser{
						ID:        300,
						CompanyID: &companyID,
						Login:     "active",
						Role:      domain.RoleCompanyAdmin,
						Status:    domain.UserStatusActive,
					},
				}, nil
			}
			return domain.UserRecord{
				AuthUser: domain.AuthUser{
					ID:     userID,
					Login:  "disabled",
					Status: domain.UserStatusDisabled,
					Role:   domain.RoleViewer,
				},
			}, nil
		},
		upsertTelegramLinkFn: func(ctx context.Context, user domain.AuthUser, telegramUserID int64, telegramUsername *string) error {
			linkCalls++
			linkedTelegramUserID = telegramUserID
			if telegramUsername != nil {
				linkedUsername = *telegramUsername
			}
			return nil
		},
	}

	service := NewAuthService(store, AuthConfig{
		TelegramBotToken:   botToken,
		TelegramAuthMaxAge: 10 * time.Minute,
	})
	service.nowFn = func() time.Time { return now }

	if err := service.LinkTelegram(context.Background(), 300, payload); err != nil {
		t.Fatalf("active user link failed: %v", err)
	}
	if err := service.LinkTelegram(context.Background(), 300, payload); err != nil {
		t.Fatalf("active user relink failed: %v", err)
	}
	if linkCalls != 2 {
		t.Fatalf("expected 2 link calls for link/relink, got %d", linkCalls)
	}
	if linkedTelegramUserID != 777 || linkedUsername != "alice" {
		t.Fatalf("unexpected link payload id=%d username=%s", linkedTelegramUserID, linkedUsername)
	}

	err := service.LinkTelegram(context.Background(), 301, payload)
	assertServiceErrorCode(t, err, "forbidden")
}

func TestLinkTelegramInvalidPayloadNoSecretLeak(t *testing.T) {
	botToken := "123456:telegram_secret_token"
	now := time.Now().UTC()
	companyID := int64(8)

	store := &authServiceStoreStub{
		getUserByIDFn: func(ctx context.Context, userID int64) (domain.UserRecord, error) {
			return domain.UserRecord{
				AuthUser: domain.AuthUser{
					ID:        userID,
					CompanyID: &companyID,
					Login:     "active",
					Role:      domain.RoleViewer,
					Status:    domain.UserStatusActive,
				},
			}, nil
		},
	}
	service := NewAuthService(store, AuthConfig{
		TelegramBotToken:   botToken,
		TelegramAuthMaxAge: 10 * time.Minute,
	})
	service.nowFn = func() time.Time { return now }

	payload := map[string]string{
		"id":        "777",
		"auth_date": fmt.Sprintf("%d", now.Unix()),
		"hash":      "invalid_hash",
	}
	err := service.LinkTelegram(context.Background(), 300, payload)
	serviceErr := assertServiceErrorCode(t, err, "unauthorized")
	reason := fmt.Sprintf("%v", serviceErr.Details["reason"])
	if !strings.Contains(reason, "hash_mismatch") {
		t.Fatalf("unexpected reason=%s", reason)
	}
	if strings.Contains(reason, botToken) {
		t.Fatalf("reason must not contain bot token")
	}
}

func validTelegramPayload(botToken string, now time.Time, telegramUserID int64, username string) map[string]string {
	payload := map[string]string{
		"id":         fmt.Sprintf("%d", telegramUserID),
		"first_name": "Alice",
		"username":   username,
		"auth_date":  fmt.Sprintf("%d", now.Unix()),
	}
	payload["hash"] = signTelegramPayload(botToken, payload)
	return payload
}

func assertServiceErrorCode(t *testing.T, err error, expectedCode string) *ServiceError {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error code=%s", expectedCode)
	}
	serviceErr, ok := AsServiceError(err)
	if !ok {
		t.Fatalf("expected service error, got %T", err)
	}
	if serviceErr.Code != expectedCode {
		t.Fatalf("expected code=%s got=%s", expectedCode, serviceErr.Code)
	}
	return serviceErr
}
