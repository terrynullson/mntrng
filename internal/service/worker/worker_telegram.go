package worker

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/example/hls-monitoring-platform/internal/domain"
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
	var settings telegramDeliverySettings
	var botTokenRef sql.NullString
	err := w.db.QueryRowContext(
		ctx,
		`SELECT is_enabled, chat_id, send_recovered, bot_token_ref
         FROM telegram_delivery_settings
         WHERE company_id = $1`,
		companyID,
	).Scan(&settings.IsEnabled, &settings.ChatID, &settings.SendRecovered, &botTokenRef)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return telegramDeliverySettings{}, false, nil
		}
		return telegramDeliverySettings{}, false, err
	}
	if botTokenRef.Valid {
		settings.BotTokenRef = strings.TrimSpace(botTokenRef.String)
	}
	settings.ChatID = strings.TrimSpace(settings.ChatID)
	return settings, true, nil
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
	for attempt := 0; ; attempt++ {
		sendCtx, cancel := context.WithTimeout(ctx, w.telegramHTTPTimeout)
		err := sendTelegramMessage(sendCtx, botToken, chatID, text)
		cancel()
		if err == nil {
			return nil
		}
		if !isRetryableTelegramError(err) || attempt >= w.telegramRetryMax {
			return err
		}

		backoff := w.telegramRetryBackoff * time.Duration(1<<attempt)
		log.Printf("worker telegram retry attempt=%d backoff=%s", attempt+1, backoff)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

type telegramHTTPError struct {
	StatusCode int
}

func (e telegramHTTPError) Error() string {
	return fmt.Sprintf("telegram sendMessage returned status=%d", e.StatusCode)
}

func sendTelegramMessage(ctx context.Context, botToken string, chatID string, text string) error {
	requestBody, err := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}

	requestURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return urlErr.Err
		}
		return err
	}
	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return telegramHTTPError{StatusCode: response.StatusCode}
	}
	return nil
}

func isRetryableTelegramError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var httpErr telegramHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusTooManyRequests || httpErr.StatusCode >= http.StatusInternalServerError
	}
	return false
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
