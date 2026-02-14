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
	return domain.ComputeWorkerAlertTransition(
		now,
		currentStatus,
		previousStatus,
		previousFailStreak,
		previousCooldownUntil,
		previousLastAlertAt,
		failStreakThreshold,
		alertCooldown,
		alertSendRecovered,
	)
}
