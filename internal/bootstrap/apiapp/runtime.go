package apiapp

import (
	"fmt"
	"time"

	"github.com/terrynullson/mntrng/internal/config"
)

type RuntimeConfig struct {
	Port                     string
	DatabaseURL              string
	RedisAddr                string
	AuthRateLimitPerMin      int
	RedisPingTimeout         time.Duration
	DBPingTimeout            time.Duration
	AuthAccessTTL            time.Duration
	AuthRefreshTTL           time.Duration
	TelegramBotTokenDefault  string
	SuperAdminTelegramChatID string
	TelegramHTTPTimeout      time.Duration
	TelegramAuthMaxAge       time.Duration
}

func LoadRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Port:                     config.GetString("API_PORT", "8080"),
		DatabaseURL:              config.GetString("DATABASE_URL", ""),
		RedisAddr:                config.GetString("REDIS_ADDR", ""),
		AuthRateLimitPerMin:      config.IntAtLeast(config.GetInt("RATE_LIMIT_AUTH_PER_MIN", 10), 1),
		RedisPingTimeout:         2 * time.Second,
		DBPingTimeout:            5 * time.Second,
		AuthAccessTTL:            time.Duration(config.IntAtLeast(config.GetInt("AUTH_ACCESS_TTL_MIN", 15), 1)) * time.Minute,
		AuthRefreshTTL:           time.Duration(config.IntAtLeast(config.GetInt("AUTH_REFRESH_TTL_DAYS", 30), 1)) * 24 * time.Hour,
		TelegramBotTokenDefault:  config.GetString("TELEGRAM_BOT_TOKEN_DEFAULT", ""),
		SuperAdminTelegramChatID: config.GetString("SUPER_ADMIN_TELEGRAM_CHAT_ID", ""),
		TelegramHTTPTimeout:      time.Duration(config.IntAtLeast(config.GetInt("TELEGRAM_HTTP_TIMEOUT_MS", 5000), 1)) * time.Millisecond,
		TelegramAuthMaxAge:       time.Duration(config.GetInt("AUTH_TELEGRAM_MAX_AGE_SEC", 600)) * time.Second,
	}
}

func (cfg RuntimeConfig) Validate() error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	return nil
}
