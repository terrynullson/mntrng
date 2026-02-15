package config

import (
	"strings"
	"testing"
)

func TestLoadDevLogNotifyConfigDefaults(t *testing.T) {
	t.Setenv("DEV_LOG_TELEGRAM_ENABLED", "")
	t.Setenv("DEV_LOG_TELEGRAM_TOKEN", "")
	t.Setenv("DEV_LOG_TELEGRAM_CHAT_ID", "")

	cfg := LoadDevLogNotifyConfig()
	if cfg.Enabled {
		t.Fatalf("expected disabled by default")
	}
	if cfg.Token != "" {
		t.Fatalf("expected empty token by default")
	}
	if cfg.ChatID != "" {
		t.Fatalf("expected empty chat id by default")
	}
}

func TestDevLogNotifyConfigValidate(t *testing.T) {
	testCases := []struct {
		name      string
		enabled   string
		token     string
		chatID    string
		expectErr bool
	}{
		{name: "disabled", enabled: "false", token: "", chatID: "", expectErr: false},
		{name: "enabled missing token", enabled: "true", token: "", chatID: "123", expectErr: true},
		{name: "enabled missing chat", enabled: "true", token: "token", chatID: "", expectErr: true},
		{name: "enabled valid", enabled: "true", token: "token", chatID: "123", expectErr: false},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("DEV_LOG_TELEGRAM_ENABLED", testCase.enabled)
			t.Setenv("DEV_LOG_TELEGRAM_TOKEN", testCase.token)
			t.Setenv("DEV_LOG_TELEGRAM_CHAT_ID", testCase.chatID)

			cfg := LoadDevLogNotifyConfig()
			err := cfg.Validate()
			if testCase.expectErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !testCase.expectErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestDevLogNotifyConfigValidateDoesNotLeakToken(t *testing.T) {
	secretToken := "super-secret-devlog-token"
	t.Setenv("DEV_LOG_TELEGRAM_ENABLED", "true")
	t.Setenv("DEV_LOG_TELEGRAM_TOKEN", secretToken)
	t.Setenv("DEV_LOG_TELEGRAM_CHAT_ID", "")

	cfg := LoadDevLogNotifyConfig()
	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if strings.Contains(err.Error(), secretToken) {
		t.Fatalf("validation error leaked token: %q", err.Error())
	}
}
