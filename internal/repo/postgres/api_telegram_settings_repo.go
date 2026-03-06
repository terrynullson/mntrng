package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

type APITelegramSettingsRepo struct {
	db *sql.DB
}

func NewAPITelegramSettingsRepo(db *sql.DB) *APITelegramSettingsRepo {
	return &APITelegramSettingsRepo{db: db}
}

func (r *APITelegramSettingsRepo) GetByCompanyID(ctx context.Context, companyID int64) (domain.TelegramDeliverySettings, error) {
	var out domain.TelegramDeliverySettings
	err := r.db.QueryRowContext(
		ctx,
		`SELECT is_enabled, chat_id, send_recovered, created_at, updated_at
         FROM telegram_delivery_settings WHERE company_id = $1`,
		companyID,
	).Scan(&out.IsEnabled, &out.ChatID, &out.SendRecovered, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TelegramDeliverySettings{}, domain.ErrTelegramDeliverySettingsNotFound
		}
		return domain.TelegramDeliverySettings{}, err
	}
	out.ChatID = strings.TrimSpace(out.ChatID)
	return out, nil
}

func (r *APITelegramSettingsRepo) Upsert(ctx context.Context, companyID int64, isEnabled *bool, chatID *string, sendRecovered *bool) (domain.TelegramDeliverySettings, error) {
	if chatID != nil && strings.TrimSpace(*chatID) == "" {
		return domain.TelegramDeliverySettings{}, domain.ErrTelegramDeliverySettingsInvalidInput
	}
	existing, err := r.GetByCompanyID(ctx, companyID)
	if err != nil && !errors.Is(err, domain.ErrTelegramDeliverySettingsNotFound) {
		return domain.TelegramDeliverySettings{}, err
	}

	exists := !errors.Is(err, domain.ErrTelegramDeliverySettingsNotFound)
	var isEnabledVal bool
	var chatIDVal string
	var sendRecoveredVal bool

	if exists {
		isEnabledVal = existing.IsEnabled
		chatIDVal = existing.ChatID
		sendRecoveredVal = existing.SendRecovered
		if isEnabled != nil {
			isEnabledVal = *isEnabled
		}
		if chatID != nil {
			chatIDVal = strings.TrimSpace(*chatID)
		}
		if sendRecovered != nil {
			sendRecoveredVal = *sendRecovered
		}
	} else {
		if chatID == nil || strings.TrimSpace(*chatID) == "" {
			return domain.TelegramDeliverySettings{}, domain.ErrTelegramDeliverySettingsInvalidInput
		}
		chatIDVal = strings.TrimSpace(*chatID)
		isEnabledVal = false
		sendRecoveredVal = false
		if isEnabled != nil {
			isEnabledVal = *isEnabled
		}
		if sendRecovered != nil {
			sendRecoveredVal = *sendRecovered
		}
	}

	var out domain.TelegramDeliverySettings
	if exists {
		err = r.db.QueryRowContext(
			ctx,
			`UPDATE telegram_delivery_settings
             SET is_enabled = $2, chat_id = $3, send_recovered = $4, updated_at = NOW()
             WHERE company_id = $1
             RETURNING is_enabled, chat_id, send_recovered, created_at, updated_at`,
			companyID, isEnabledVal, chatIDVal, sendRecoveredVal,
		).Scan(&out.IsEnabled, &out.ChatID, &out.SendRecovered, &out.CreatedAt, &out.UpdatedAt)
	} else {
		err = r.db.QueryRowContext(
			ctx,
			`INSERT INTO telegram_delivery_settings (company_id, is_enabled, chat_id, send_recovered)
             VALUES ($1, $2, $3, $4)
             RETURNING is_enabled, chat_id, send_recovered, created_at, updated_at`,
			companyID, isEnabledVal, chatIDVal, sendRecoveredVal,
		).Scan(&out.IsEnabled, &out.ChatID, &out.SendRecovered, &out.CreatedAt, &out.UpdatedAt)
	}
	if err != nil {
		if isForeignKeyViolation(err) {
			return domain.TelegramDeliverySettings{}, domain.ErrCompanyNotFound
		}
		return domain.TelegramDeliverySettings{}, err
	}
	out.ChatID = strings.TrimSpace(out.ChatID)
	return out, nil
}
