package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/example/hls-monitoring-platform/internal/domain"
	transport "github.com/example/hls-monitoring-platform/internal/telegram"
)

func (w *worker) processTelegramDelivery(ctx context.Context, job claimedJob, evaluation checkJobEvaluation, decision alertDecision) {
	if !decision.ShouldSend {
		log.Printf(
			"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s",
			job.CompanyID,
			job.StreamID,
			decision.EventType,
			decision.ShouldSend,
			"skipped",
			"decision_false",
		)
		return
	}

	settings, found, err := w.loadTelegramDeliverySettings(ctx, job.CompanyID)
	if err != nil {
		log.Printf(
			"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s err=%v",
			job.CompanyID,
			job.StreamID,
			decision.EventType,
			decision.ShouldSend,
			"failed",
			"settings_load_error",
			err,
		)
		return
	}
	if !found {
		log.Printf(
			"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s",
			job.CompanyID,
			job.StreamID,
			decision.EventType,
			decision.ShouldSend,
			"skipped",
			"settings_not_found",
		)
		return
	}
	if !settings.IsEnabled {
		log.Printf(
			"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s",
			job.CompanyID,
			job.StreamID,
			decision.EventType,
			decision.ShouldSend,
			"skipped",
			"settings_disabled",
		)
		return
	}
	if settings.ChatID == "" {
		log.Printf(
			"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s",
			job.CompanyID,
			job.StreamID,
			decision.EventType,
			decision.ShouldSend,
			"skipped",
			"chat_id_missing",
		)
		return
	}
	if decision.EventType == domain.WorkerAlertEventRecovered && !settings.SendRecovered {
		log.Printf(
			"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s",
			job.CompanyID,
			job.StreamID,
			decision.EventType,
			decision.ShouldSend,
			"skipped",
			"recovered_disabled_for_company",
		)
		return
	}

	token, err := w.resolveTelegramBotToken(settings.BotTokenRef)
	if err != nil {
		log.Printf(
			"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s err=%v",
			job.CompanyID,
			job.StreamID,
			decision.EventType,
			decision.ShouldSend,
			"failed",
			"token_resolve_error",
			err,
		)
		return
	}

	messageText := buildTelegramMessage(job, evaluation, decision)
	sendErr := w.sendTelegramMessageWithRetry(ctx, token, settings.ChatID, messageText)
	if sendErr != nil {
		log.Printf(
			"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s",
			job.CompanyID,
			job.StreamID,
			decision.EventType,
			decision.ShouldSend,
			"failed",
			"send_error",
		)
		return
	}

	log.Printf(
		"worker telegram delivery: company_id=%d stream_id=%d event_type=%s should_send=%t delivery_result=%s reason=%s",
		job.CompanyID,
		job.StreamID,
		decision.EventType,
		decision.ShouldSend,
		"sent",
		"ok",
	)
}

func (w *worker) loadTelegramDeliverySettings(ctx context.Context, companyID int64) (telegramDeliverySettings, bool, error) {
	return w.telegramSettingsRepo.LoadTelegramDeliverySettings(ctx, companyID)
}

func (w *worker) resolveTelegramBotToken(botTokenRef string) (string, error) {
	ref := strings.TrimSpace(botTokenRef)
	if ref == "" {
		token := strings.TrimSpace(w.telegramBotTokenDefault)
		if token == "" {
			return "", errors.New("telegram default bot token is not configured")
		}
		return token, nil
	}

	normalizedRef := normalizeTokenRef(ref)
	if normalizedRef == "" {
		return "", errors.New("telegram bot token ref is invalid")
	}

	envKey, err := telegramTokenEnvKey(ref)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(config.GetString(envKey, ""))
	if token == "" {
		return "", errors.New("telegram bot token ref is not configured in env")
	}
	return token, nil
}

func telegramTokenEnvKey(botTokenRef string) (string, error) {
	normalizedRef := normalizeTokenRef(botTokenRef)
	if normalizedRef == "" {
		return "", errors.New("telegram bot token ref is invalid")
	}
	return "TELEGRAM_BOT_TOKEN_" + normalizedRef, nil
}

func normalizeTokenRef(value string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	var builder strings.Builder
	lastUnderscore := false

	for _, ch := range trimmed {
		isAlphaNum := (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
		if isAlphaNum {
			builder.WriteRune(ch)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteRune('_')
			lastUnderscore = true
		}
	}

	normalized := strings.Trim(builder.String(), "_")
	return normalized
}

func (w *worker) sendTelegramMessageWithRetry(ctx context.Context, botToken string, chatID string, text string) error {
	client := transport.NewClient(nil)
	for attempt := 0; ; attempt++ {
		sendCtx, cancel := context.WithTimeout(ctx, w.telegramHTTPTimeout)
		err := client.SendMessage(sendCtx, botToken, chatID, text)
		cancel()
		if err == nil {
			return nil
		}
		if !transport.IsRetryableError(err) || attempt >= w.telegramRetryMax {
			return err
		}

		backoff := w.telegramRetryBackoff * time.Duration(1<<attempt)
		log.Printf("worker telegram retry attempt=%d backoff=%s", attempt+1, backoff)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func buildTelegramMessage(job claimedJob, evaluation checkJobEvaluation, decision alertDecision) string {
	eventType := strings.ToUpper(strings.TrimSpace(decision.EventType))
	if eventType == "" {
		eventType = "ALERT"
	}
	return fmt.Sprintf(
		"HLS monitor alert\nEvent: %s\nCompany ID: %d\nStream ID: %d\nJob ID: %d\nStatus: %s\nDecision reason: %s",
		eventType,
		job.CompanyID,
		job.StreamID,
		job.ID,
		strings.ToUpper(evaluation.DBStatus),
		decision.Reason,
	)
}
