package config

import (
	"errors"
	"strings"
	"time"
)

const (
	defaultDevLogTelegramEnabled = false
	defaultDevLogHTTPTimeout     = 5 * time.Second
)

type DevLogNotifyConfig struct {
	Enabled     bool
	Token       string
	ChatID      string
	HTTPTimeout time.Duration
}

func LoadDevLogNotifyConfig() DevLogNotifyConfig {
	return DevLogNotifyConfig{
		Enabled:     GetBool("DEV_LOG_TELEGRAM_ENABLED", defaultDevLogTelegramEnabled),
		Token:       GetString("DEV_LOG_TELEGRAM_TOKEN", ""),
		ChatID:      GetString("DEV_LOG_TELEGRAM_CHAT_ID", ""),
		HTTPTimeout: defaultDevLogHTTPTimeout,
	}
}

func (c DevLogNotifyConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Token) == "" {
		return errors.New("DEV_LOG_TELEGRAM_TOKEN is required when DEV_LOG_TELEGRAM_ENABLED=true")
	}
	if strings.TrimSpace(c.ChatID) == "" {
		return errors.New("DEV_LOG_TELEGRAM_CHAT_ID is required when DEV_LOG_TELEGRAM_ENABLED=true")
	}
	return nil
}
