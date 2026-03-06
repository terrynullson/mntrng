package api

import (
	"context"
	"errors"
	"strings"

	"github.com/terrynullson/mntrng/internal/domain"
)

type TelegramSettingsStore interface {
	GetByCompanyID(ctx context.Context, companyID int64) (domain.TelegramDeliverySettings, error)
	Upsert(ctx context.Context, companyID int64, isEnabled *bool, chatID *string, sendRecovered *bool) (domain.TelegramDeliverySettings, error)
}

type TelegramSettingsService struct {
	store TelegramSettingsStore
}

func NewTelegramSettingsService(store TelegramSettingsStore) *TelegramSettingsService {
	return &TelegramSettingsService{store: store}
}

func (s *TelegramSettingsService) GetTelegramDeliverySettings(ctx context.Context, companyID int64) (domain.TelegramDeliverySettings, error) {
	out, err := s.store.GetByCompanyID(ctx, companyID)
	if err == nil {
		return out, nil
	}
	if errors.Is(err, domain.ErrTelegramDeliverySettingsNotFound) {
		return domain.TelegramDeliverySettings{}, NewNotFoundError("telegram delivery settings not found", map[string]interface{}{"company_id": companyID})
	}
	return domain.TelegramDeliverySettings{}, NewInternalError()
}

func (s *TelegramSettingsService) PatchTelegramDeliverySettings(ctx context.Context, companyID int64, patch domain.PatchTelegramDeliverySettingsRequest) (domain.TelegramDeliverySettings, error) {
	_, err := s.store.GetByCompanyID(ctx, companyID)
	exists := err == nil
	if err != nil && !errors.Is(err, domain.ErrTelegramDeliverySettingsNotFound) {
		return domain.TelegramDeliverySettings{}, NewInternalError()
	}
	if !exists && (patch.ChatID == nil || strings.TrimSpace(*patch.ChatID) == "") {
		return domain.TelegramDeliverySettings{}, NewValidationError("chat_id is required when creating settings", map[string]interface{}{"field": "chat_id"})
	}

	out, err := s.store.Upsert(ctx, companyID, patch.IsEnabled, patch.ChatID, patch.SendRecovered)
	if err == nil {
		return out, nil
	}
	if errors.Is(err, domain.ErrTelegramDeliverySettingsNotFound) {
		return domain.TelegramDeliverySettings{}, NewNotFoundError("telegram delivery settings not found", map[string]interface{}{"company_id": companyID})
	}
	if errors.Is(err, domain.ErrTelegramDeliverySettingsInvalidInput) {
		return domain.TelegramDeliverySettings{}, NewValidationError("chat_id must be non-empty when provided", map[string]interface{}{"field": "chat_id"})
	}
	if errors.Is(err, domain.ErrCompanyNotFound) {
		return domain.TelegramDeliverySettings{}, NewNotFoundError("company not found", map[string]interface{}{"company_id": companyID})
	}
	return domain.TelegramDeliverySettings{}, NewInternalError()
}
