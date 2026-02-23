package domain

import (
	"database/sql"
	"time"
)

func ComputeWorkerAlertTransition(
	now time.Time,
	currentStatus string,
	previousStatus string,
	previousFailStreak int,
	previousCooldownUntil sql.NullTime,
	previousLastAlertAt sql.NullTime,
	failStreakThreshold int,
	alertCooldown time.Duration,
	alertSendRecovered bool,
) WorkerAlertTransitionResult {
	nowUTC := now.UTC()
	cooldownActive := previousCooldownUntil.Valid && previousCooldownUntil.Time.After(nowUTC)

	newFailStreak := 0
	if currentStatus == WorkerStatusDBFail {
		if previousStatus == WorkerStatusDBFail {
			newFailStreak = previousFailStreak + 1
		} else {
			newFailStreak = 1
		}
	}

	decision := WorkerAlertDecision{
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

	if currentStatus == WorkerStatusDBFail {
		if newFailStreak < failStreakThreshold {
			decision.Reason = "fail_streak_below_threshold"
		} else if cooldownActive {
			decision.Reason = "cooldown_active"
		} else {
			decision.ShouldSend = true
			decision.EventType = WorkerAlertEventFail
			decision.Reason = "fail_streak_threshold_met"
			nextLastAlertAt = sql.NullTime{Time: nowUTC, Valid: true}
			nextCooldownUntil = sql.NullTime{Time: nowUTC.Add(alertCooldown), Valid: true}
		}
	} else if currentStatus == WorkerStatusDBWarn && previousStatus == WorkerStatusDBOK {
		if cooldownActive {
			decision.Reason = "cooldown_active"
		} else {
			decision.ShouldSend = true
			decision.EventType = WorkerAlertEventWarn
			decision.Reason = "ok_to_warn_transition"
			nextLastAlertAt = sql.NullTime{Time: nowUTC, Valid: true}
			nextCooldownUntil = sql.NullTime{Time: nowUTC.Add(alertCooldown), Valid: true}
		}
	} else if currentStatus == WorkerStatusDBOK && previousStatus == WorkerStatusDBFail {
		if !alertSendRecovered {
			decision.Reason = "recovered_suppressed_by_config"
		} else if cooldownActive {
			decision.Reason = "cooldown_active"
		} else {
			decision.ShouldSend = true
			decision.EventType = WorkerAlertEventRecovered
			decision.Reason = "recovered_transition"
			nextLastAlertAt = sql.NullTime{Time: nowUTC, Valid: true}
			nextCooldownUntil = sql.NullTime{Time: nowUTC.Add(alertCooldown), Valid: true}
		}
	}

	return WorkerAlertTransitionResult{
		Decision:          decision,
		NextFailStreak:    newFailStreak,
		NextCooldownUntil: nextCooldownUntil,
		NextLastAlertAt:   nextLastAlertAt,
	}
}
