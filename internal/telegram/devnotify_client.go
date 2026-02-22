package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const defaultDevLogNotifierTimeout = 5 * time.Second

type DevLogNotifier struct {
	enabled    bool
	botToken   string
	chatID     string
	httpClient *http.Client
}

func NewDevLogNotifier(enabled bool, botToken string, chatID string, httpClient *http.Client) *DevLogNotifier {
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: defaultDevLogNotifierTimeout}
	}

	return &DevLogNotifier{
		enabled:    enabled,
		botToken:   strings.TrimSpace(botToken),
		chatID:     strings.TrimSpace(chatID),
		httpClient: client,
	}
}

func (n *DevLogNotifier) NotifyCompletion(ctx context.Context, payload DevLogPayload) error {
	if n == nil || !n.enabled {
		return nil
	}
	if n.botToken == "" {
		return errors.New("dev log notifier token is not configured")
	}
	if n.chatID == "" {
		return errors.New("dev log notifier chat_id is not configured")
	}
	if err := ValidateDevLogPayload(payload); err != nil {
		return errors.New("dev log payload violates safety guardrails")
	}

	text := BuildDevLogMessage(payload)
	text = ensureUTF8(text)
	return n.sendMessage(ctx, text)
}

// ensureUTF8 replaces invalid UTF-8 runes so Telegram always receives valid UTF-8.
func ensureUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r != utf8.RuneError {
			b.WriteRune(r)
		} else {
			b.WriteRune('?')
		}
	}
	return b.String()
}

func (n *DevLogNotifier) sendMessage(ctx context.Context, text string) error {
	requestBody, err := json.Marshal(map[string]string{
		"chat_id": n.chatID,
		"text":    text,
	})
	if err != nil {
		return errors.New("failed to encode dev log payload")
	}

	requestURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		return errors.New("failed to create telegram request")
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	response, err := n.httpClient.Do(request)
	if err != nil {
		return errors.New("telegram request failed")
	}
	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram send failed with status=%d", response.StatusCode)
	}

	return nil
}
