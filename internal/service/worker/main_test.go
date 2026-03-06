package worker

import (
	"database/sql"
	"testing"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
)

func TestAggregateStatuses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "all ok",
			input:    []string{"OK", "OK", "OK"},
			expected: "OK",
		},
		{
			name:     "warn without fail",
			input:    []string{"OK", "WARN", "OK"},
			expected: "WARN",
		},
		{
			name:     "fail dominates",
			input:    []string{"OK", "WARN", "FAIL", "OK"},
			expected: "FAIL",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			actual := aggregateStatuses(testCase.input...)
			if actual != testCase.expected {
				t.Fatalf("aggregateStatuses(%v) = %s, expected %s", testCase.input, actual, testCase.expected)
			}
		})
	}
}

func TestComputeAlertTransition(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	cooldown := 10 * time.Minute
	previousAlertTime := sql.NullTime{Time: now.Add(-2 * time.Hour), Valid: true}

	t.Run("fail streak below threshold", func(t *testing.T) {
		result := domain.ComputeWorkerAlertTransition(
			now,
			"fail",
			"ok",
			0,
			sql.NullTime{},
			sql.NullTime{},
			2,
			cooldown,
			true,
		)

		if result.Decision.ShouldSend {
			t.Fatalf("expected should_send=false")
		}
		if result.Decision.Reason != "fail_streak_below_threshold" {
			t.Fatalf("unexpected reason: %s", result.Decision.Reason)
		}
		if result.NextFailStreak != 1 {
			t.Fatalf("expected next fail streak 1, got %d", result.NextFailStreak)
		}
	})

	t.Run("fail threshold met sends alert", func(t *testing.T) {
		result := domain.ComputeWorkerAlertTransition(
			now,
			"fail",
			"fail",
			1,
			sql.NullTime{},
			previousAlertTime,
			2,
			cooldown,
			true,
		)

		if !result.Decision.ShouldSend {
			t.Fatalf("expected should_send=true")
		}
		if result.Decision.EventType != "fail" {
			t.Fatalf("unexpected event type: %s", result.Decision.EventType)
		}
		if result.Decision.Reason != "fail_streak_threshold_met" {
			t.Fatalf("unexpected reason: %s", result.Decision.Reason)
		}
		if result.NextFailStreak != 2 {
			t.Fatalf("expected next fail streak 2, got %d", result.NextFailStreak)
		}
		if !result.NextCooldownUntil.Valid || !result.NextCooldownUntil.Time.Equal(now.Add(cooldown)) {
			t.Fatalf("unexpected cooldown_until: %#v", result.NextCooldownUntil)
		}
		if !result.NextLastAlertAt.Valid || !result.NextLastAlertAt.Time.Equal(now) {
			t.Fatalf("unexpected last_alert_at: %#v", result.NextLastAlertAt)
		}
	})

	t.Run("fail in cooldown does not send", func(t *testing.T) {
		cooldownUntil := sql.NullTime{Time: now.Add(2 * time.Minute), Valid: true}
		result := domain.ComputeWorkerAlertTransition(
			now,
			"fail",
			"fail",
			5,
			cooldownUntil,
			previousAlertTime,
			2,
			cooldown,
			true,
		)

		if result.Decision.ShouldSend {
			t.Fatalf("expected should_send=false")
		}
		if result.Decision.Reason != "cooldown_active" {
			t.Fatalf("unexpected reason: %s", result.Decision.Reason)
		}
		if result.NextFailStreak != 6 {
			t.Fatalf("expected next fail streak 6, got %d", result.NextFailStreak)
		}
		if !result.NextCooldownUntil.Valid || !result.NextCooldownUntil.Time.Equal(cooldownUntil.Time) {
			t.Fatalf("unexpected cooldown_until: %#v", result.NextCooldownUntil)
		}
	})

	t.Run("recovered suppressed by config", func(t *testing.T) {
		result := domain.ComputeWorkerAlertTransition(
			now,
			"ok",
			"fail",
			3,
			sql.NullTime{},
			previousAlertTime,
			2,
			cooldown,
			false,
		)

		if result.Decision.ShouldSend {
			t.Fatalf("expected should_send=false")
		}
		if result.Decision.Reason != "recovered_suppressed_by_config" {
			t.Fatalf("unexpected reason: %s", result.Decision.Reason)
		}
		if result.NextFailStreak != 0 {
			t.Fatalf("expected next fail streak 0, got %d", result.NextFailStreak)
		}
	})

	t.Run("recovered sends when enabled", func(t *testing.T) {
		result := domain.ComputeWorkerAlertTransition(
			now,
			"ok",
			"fail",
			2,
			sql.NullTime{},
			previousAlertTime,
			2,
			cooldown,
			true,
		)

		if !result.Decision.ShouldSend {
			t.Fatalf("expected should_send=true")
		}
		if result.Decision.EventType != "recovered" {
			t.Fatalf("unexpected event type: %s", result.Decision.EventType)
		}
		if result.Decision.Reason != "recovered_transition" {
			t.Fatalf("unexpected reason: %s", result.Decision.Reason)
		}
		if !result.NextCooldownUntil.Valid || !result.NextCooldownUntil.Time.Equal(now.Add(cooldown)) {
			t.Fatalf("unexpected cooldown_until: %#v", result.NextCooldownUntil)
		}
	})

	t.Run("recovered blocked by cooldown", func(t *testing.T) {
		cooldownUntil := sql.NullTime{Time: now.Add(3 * time.Minute), Valid: true}
		result := domain.ComputeWorkerAlertTransition(
			now,
			"ok",
			"fail",
			2,
			cooldownUntil,
			previousAlertTime,
			2,
			cooldown,
			true,
		)

		if result.Decision.ShouldSend {
			t.Fatalf("expected should_send=false")
		}
		if result.Decision.Reason != "cooldown_active" {
			t.Fatalf("unexpected reason: %s", result.Decision.Reason)
		}
	})

	t.Run("ok_to_warn sends when no cooldown", func(t *testing.T) {
		result := domain.ComputeWorkerAlertTransition(
			now,
			"warn",
			"ok",
			0,
			sql.NullTime{},
			previousAlertTime,
			2,
			cooldown,
			true,
		)
		if !result.Decision.ShouldSend {
			t.Fatalf("expected should_send=true")
		}
		if result.Decision.EventType != domain.WorkerAlertEventWarn {
			t.Fatalf("unexpected event type: %s", result.Decision.EventType)
		}
		if result.Decision.Reason != "ok_to_warn_transition" {
			t.Fatalf("unexpected reason: %s", result.Decision.Reason)
		}
		if !result.NextCooldownUntil.Valid || !result.NextCooldownUntil.Time.Equal(now.Add(cooldown)) {
			t.Fatalf("unexpected cooldown_until: %#v", result.NextCooldownUntil)
		}
	})

	t.Run("ok_to_warn blocked by cooldown", func(t *testing.T) {
		cooldownUntil := sql.NullTime{Time: now.Add(5 * time.Minute), Valid: true}
		result := domain.ComputeWorkerAlertTransition(
			now,
			"warn",
			"ok",
			0,
			cooldownUntil,
			previousAlertTime,
			2,
			cooldown,
			true,
		)
		if result.Decision.ShouldSend {
			t.Fatalf("expected should_send=false")
		}
		if result.Decision.Reason != "cooldown_active" {
			t.Fatalf("unexpected reason: %s", result.Decision.Reason)
		}
	})
}

func TestNormalizeTokenRef(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normalizes mixed symbols",
			input:    "  prod-bot 1  ",
			expected: "PROD_BOT_1",
		},
		{
			name:     "collapses separators",
			input:    "a---b...c",
			expected: "A_B_C",
		},
		{
			name:     "invalid all separators",
			input:    "***",
			expected: "",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			actual := normalizeTokenRef(testCase.input)
			if actual != testCase.expected {
				t.Fatalf("normalizeTokenRef(%q) = %q, expected %q", testCase.input, actual, testCase.expected)
			}
		})
	}
}

func TestTelegramTokenResolver(t *testing.T) {
	t.Run("env key resolver", func(t *testing.T) {
		key, err := telegramTokenEnvKey("my-prod bot")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if key != "TELEGRAM_BOT_TOKEN_MY_PROD_BOT" {
			t.Fatalf("unexpected key: %s", key)
		}
	})

	t.Run("env key resolver invalid ref", func(t *testing.T) {
		_, err := telegramTokenEnvKey("...")
		if err == nil {
			t.Fatalf("expected error for invalid ref")
		}
	})

	t.Run("default token fallback", func(t *testing.T) {
		w := &worker{telegramBotTokenDefault: "default-token"}
		token, err := w.resolveTelegramBotToken("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "default-token" {
			t.Fatalf("unexpected token: %s", token)
		}
	})

	t.Run("token by ref from env", func(t *testing.T) {
		t.Setenv("TELEGRAM_BOT_TOKEN_ALERTS_PRIMARY", "ref-token")
		w := &worker{telegramBotTokenDefault: "default-token"}
		token, err := w.resolveTelegramBotToken("alerts primary")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ref-token" {
			t.Fatalf("unexpected token: %s", token)
		}
	})

	t.Run("missing token for ref", func(t *testing.T) {
		w := &worker{telegramBotTokenDefault: "default-token"}
		_, err := w.resolveTelegramBotToken("missing_ref")
		if err == nil {
			t.Fatalf("expected error for missing ref token")
		}
	})
}

func TestThresholdBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("freeze threshold boundaries", func(t *testing.T) {
		if status := freezeStatusByThreshold(1.99, 2, 5); status != "OK" {
			t.Fatalf("expected OK, got %s", status)
		}
		if status := freezeStatusByThreshold(2, 2, 5); status != "WARN" {
			t.Fatalf("expected WARN, got %s", status)
		}
		if status := freezeStatusByThreshold(5, 2, 5); status != "FAIL" {
			t.Fatalf("expected FAIL, got %s", status)
		}
	})

	t.Run("blackframe threshold boundaries", func(t *testing.T) {
		if status := blackframeStatusByThreshold(0.89, 0.9, 0.98); status != "OK" {
			t.Fatalf("expected OK, got %s", status)
		}
		if status := blackframeStatusByThreshold(0.9, 0.9, 0.98); status != "WARN" {
			t.Fatalf("expected WARN, got %s", status)
		}
		if status := blackframeStatusByThreshold(0.98, 0.9, 0.98); status != "FAIL" {
			t.Fatalf("expected FAIL, got %s", status)
		}
	})

	t.Run("effective bitrate ratio boundaries", func(t *testing.T) {
		w := &worker{
			effectiveWarnRatio: 0.7,
			effectiveFailRatio: 0.4,
		}
		declared := declaredBitrateResult{DeclaredBPS: 1000}

		statusAtFailBoundary, _ := w.checkEffectiveBitrate(
			[]segmentSample{{Downloaded: true, DurationSec: 10, Bytes: 500}},
			declared,
		)
		if statusAtFailBoundary != "WARN" {
			t.Fatalf("expected WARN at ratio=0.4, got %s", statusAtFailBoundary)
		}

		statusBelowFailBoundary, _ := w.checkEffectiveBitrate(
			[]segmentSample{{Downloaded: true, DurationSec: 100, Bytes: 4987}},
			declared,
		)
		if statusBelowFailBoundary != "FAIL" {
			t.Fatalf("expected FAIL below ratio=0.4, got %s", statusBelowFailBoundary)
		}

		statusAtWarnBoundary, _ := w.checkEffectiveBitrate(
			[]segmentSample{{Downloaded: true, DurationSec: 10, Bytes: 875}},
			declared,
		)
		if statusAtWarnBoundary != "OK" {
			t.Fatalf("expected OK at ratio=0.7, got %s", statusAtWarnBoundary)
		}

		statusBelowWarnBoundary, _ := w.checkEffectiveBitrate(
			[]segmentSample{{Downloaded: true, DurationSec: 100, Bytes: 8737}},
			declared,
		)
		if statusBelowWarnBoundary != "WARN" {
			t.Fatalf("expected WARN below ratio=0.7, got %s", statusBelowWarnBoundary)
		}
	})
}
