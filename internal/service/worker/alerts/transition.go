package alerts

import (
	"database/sql"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func ComputeTransition(
	now time.Time,
	currentStatus string,
	previousStatus string,
	previousFailStreak int,
	previousCooldownUntil sql.NullTime,
	previousLastAlertAt sql.NullTime,
	failStreakThreshold int,
	alertCooldown time.Duration,
	alertSendRecovered bool,
) domain.WorkerAlertTransitionResult {
	nowUTC := now.UTC()
	cooldownActive := previousCooldownUntil.Valid && previousCooldownUntil.Time.After(nowUTC)

	newFailStreak := 0
	if currentStatus == domain.WorkerStatusDBFail {
		if previousStatus == domain.WorkerStatusDBFail {
			newFailStreak = previousFailStreak + 1
		} else {
			newFailStreak = 1
		}
	}

	decision := domain.WorkerAlertDecision{
		ShouldSend:     false,
		EventType:      "",
		Reason:         "no_alert_condition",
		CurrentStatus:  currentStatus,
		PreviousStatus: previousStatus,
		FailStreak:     newFailStreak,
		CooldownUntil:  nil,
	}

	nextCooldownUntil := previousCooldownUntil
	nextLastAlertAt := previousLastAlertAt

	if currentStatus == domain.WorkerStatusDBFail {
		if newFailStreak < failStreakThreshold {
			decision.Reason = "fail_streak_below_threshold"
		} else if cooldownActive {
			decision.Reason = "cooldown_active"
		} else {
			decision.ShouldSend = true
			decision.EventType = domain.WorkerAlertEventFail
			decision.Reason = "fail_streak_threshold_met"
			nextLastAlertAt = sql.NullTime{Time: nowUTC, Valid: true}
			nextCooldownUntil = sql.NullTime{Time: nowUTC.Add(alertCooldown), Valid: true}
		}
	} else if currentStatus == domain.WorkerStatusDBOK && previousStatus == domain.WorkerStatusDBFail {
		if !alertSendRecovered {
			decision.Reason = "recovered_suppressed_by_config"
		} else if cooldownActive {
			decision.Reason = "cooldown_active"
		} else {
			decision.ShouldSend = true
			decision.EventType = domain.WorkerAlertEventRecovered
			decision.Reason = "recovered_transition"
			nextLastAlertAt = sql.NullTime{Time: nowUTC, Valid: true}
			nextCooldownUntil = sql.NullTime{Time: nowUTC.Add(alertCooldown), Valid: true}
		}
	}

	return domain.WorkerAlertTransitionResult{
		Decision:          decision,
		NextFailStreak:    newFailStreak,
		NextCooldownUntil: nextCooldownUntil,
		NextLastAlertAt:   nextLastAlertAt,
	}
}
