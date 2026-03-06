package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

func (r *WorkerRepo) LoadTelegramDeliverySettings(ctx context.Context, companyID int64) (domain.WorkerTelegramDeliverySettings, bool, error) {
	var settings domain.WorkerTelegramDeliverySettings
	var botTokenRef sql.NullString
	err := r.db.QueryRowContext(
		ctx,
		`SELECT is_enabled, chat_id, send_recovered, bot_token_ref
         FROM telegram_delivery_settings
         WHERE company_id = $1`,
		companyID,
	).Scan(&settings.IsEnabled, &settings.ChatID, &settings.SendRecovered, &botTokenRef)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.WorkerTelegramDeliverySettings{}, false, nil
		}
		return domain.WorkerTelegramDeliverySettings{}, false, err
	}
	if botTokenRef.Valid {
		settings.BotTokenRef = strings.TrimSpace(botTokenRef.String)
	}
	settings.ChatID = strings.TrimSpace(settings.ChatID)
	return settings, true, nil
}
