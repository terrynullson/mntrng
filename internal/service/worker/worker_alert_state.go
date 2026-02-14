package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (w *worker) persistCheckResultWithRetry(ctx context.Context, job claimedJob, evaluation checkJobEvaluation) error {
	for attempt := 0; ; attempt++ {
		err := w.persistCheckResult(ctx, job, evaluation)
		if err == nil {
			return nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker persist retry attempt=%d job_id=%d backoff=%s err=%v", attempt+1, job.ID, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func (w *worker) persistCheckResult(ctx context.Context, job claimedJob, evaluation checkJobEvaluation) error {
	checksJSON, err := json.Marshal(evaluation.Checks)
	if err != nil {
		return err
	}

	if err := w.checkResultRepo.PersistCheckResult(ctx, job, evaluation.DBStatus, string(checksJSON)); err != nil {
		return err
	}

	log.Printf(
		"worker stored check_result: job_id=%d company_id=%d stream_id=%d status=%s checks=%v",
		job.ID,
		job.CompanyID,
		job.StreamID,
		evaluation.Aggregate,
		evaluation.Checks,
	)
	return nil
}

func (w *worker) applyAlertStateWithRetry(ctx context.Context, job claimedJob, resultStatus string) (alertDecision, error) {
	for attempt := 0; ; attempt++ {
		decision, err := w.applyAlertState(ctx, job, resultStatus)
		if err == nil {
			return decision, nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return alertDecision{}, err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker alert_state retry attempt=%d job_id=%d backoff=%s err=%v", attempt+1, job.ID, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return alertDecision{}, err
		}
	}
}

func (w *worker) applyAlertState(ctx context.Context, job claimedJob, resultStatus string) (alertDecision, error) {
	currentStatus, err := normalizeAlertStatus(resultStatus)
	if err != nil {
		return alertDecision{}, err
	}
	return w.alertStateRepo.ApplyAlertState(
		ctx,
		job.CompanyID,
		job.StreamID,
		currentStatus,
		w.alertFailStreak,
		w.alertCooldown,
		w.alertSendRecovered,
	)
}

func computeAlertTransition(
	now time.Time,
	currentStatus string,
	previousStatus string,
	previousFailStreak int,
	previousCooldownUntil sql.NullTime,
	previousLastAlertAt sql.NullTime,
	failStreakThreshold int,
	alertCooldown time.Duration,
	alertSendRecovered bool,
) alertTransitionResult {
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

func (w *worker) logAlertDecision(job claimedJob, decision alertDecision) {
	cooldownUntil := "null"
	if decision.CooldownUntil != nil {
		cooldownUntil = decision.CooldownUntil.Format(time.RFC3339)
	}
	log.Printf(
		"worker alert decision: company_id=%d stream_id=%d current_status=%s previous_status=%s fail_streak=%d fail_threshold=%d cooldown_until=%s should_send=%t event_type=%s reason=%s",
		job.CompanyID,
		job.StreamID,
		decision.CurrentStatus,
		decision.PreviousStatus,
		decision.FailStreak,
		w.alertFailStreak,
		cooldownUntil,
		decision.ShouldSend,
		decision.EventType,
		decision.Reason,
	)
}
