package config

import (
	"fmt"
	"strings"
)

func IsProduction() bool {
	switch strings.ToLower(GetString("APP_ENV", "")) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func ValidateAPIRuntimeSafety() error {
	if !IsProduction() {
		return nil
	}

	if GetBool("API_METRICS_PUBLIC", false) {
		return fmt.Errorf("unsafe production config: API_METRICS_PUBLIC must be false")
	}
	if !GetBool("AUTH_COOKIE_SECURE", true) {
		return fmt.Errorf("unsafe production config: AUTH_COOKIE_SECURE must be true")
	}

	sameSite := strings.ToLower(strings.TrimSpace(GetString("AUTH_COOKIE_SAMESITE", "strict")))
	if sameSite == "none" {
		return fmt.Errorf("unsafe production config: AUTH_COOKIE_SAMESITE=none is not allowed")
	}

	if GetBool("BOOTSTRAP_SEED_ENABLED", false) {
		return fmt.Errorf("unsafe production config: BOOTSTRAP_SEED_ENABLED must be false")
	}

	for _, origin := range strings.Split(GetString("CORS_ALLOWED_ORIGINS", ""), ",") {
		trimmed := strings.ToLower(strings.TrimSpace(origin))
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "http://") &&
			!strings.Contains(trimmed, "localhost") &&
			!strings.Contains(trimmed, "127.0.0.1") {
			return fmt.Errorf("unsafe production config: CORS origin must use https in production: %s", origin)
		}
	}

	return nil
}
