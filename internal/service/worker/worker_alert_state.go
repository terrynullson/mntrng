package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	workeralerts "github.com/example/hls-monitoring-platform/internal/service/worker/alerts"
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

	result, err := w.db.ExecContext(
		ctx,
		`INSERT INTO check_results (company_id, job_id, stream_id, status, checks)
         VALUES ($1, $2, $3, $4, $5::jsonb)
         ON CONFLICT (job_id) DO NOTHING`,
		job.CompanyID,
		job.ID,
		job.StreamID,
		evaluation.DBStatus,
		string(checksJSON),
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		var existingCount int
		countErr := w.db.QueryRowContext(
			ctx,
			`SELECT COUNT(1)
             FROM check_results
             WHERE company_id = $1
               AND job_id = $2
               AND stream_id = $3`,
			job.CompanyID,
			job.ID,
			job.StreamID,
		).Scan(&existingCount)
		if countErr != nil {
			return countErr
		}
		if existingCount == 0 {
			return errors.New("check_result conflict without matching tenant row")
		}
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

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return alertDecision{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO alert_state (company_id, stream_id, fail_streak, cooldown_until, last_alert_at, last_status, created_at, updated_at)
         VALUES ($1, $2, 0, NULL, NULL, NULL, NOW(), NOW())
         ON CONFLICT (stream_id) DO NOTHING`,
		job.CompanyID,
		job.StreamID,
	)
	if err != nil {
		return alertDecision{}, err
	}

	var previousFailStreak int
	var previousCooldownUntil sql.NullTime
	var previousLastAlertAt sql.NullTime
	var previousStatusRaw sql.NullString
	scanErr := tx.QueryRowContext(
		ctx,
		`SELECT fail_streak, cooldown_until, last_alert_at, last_status
         FROM alert_state
         WHERE company_id = $1
           AND stream_id = $2
         FOR UPDATE`,
		job.CompanyID,
		job.StreamID,
	).Scan(&previousFailStreak, &previousCooldownUntil, &previousLastAlertAt, &previousStatusRaw)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return alertDecision{}, errors.New("alert_state row not found in tenant context")
		}
		return alertDecision{}, scanErr
	}

	previousStatus := ""
	if previousStatusRaw.Valid {
		normalizedPrevious, prevErr := normalizeAlertStatus(previousStatusRaw.String)
		if prevErr == nil {
			previousStatus = normalizedPrevious
		}
	}

	now := time.Now().UTC()
	transition := workeralerts.ComputeTransition(
		now,
		currentStatus,
		previousStatus,
		previousFailStreak,
		previousCooldownUntil,
		previousLastAlertAt,
		w.alertFailStreak,
		w.alertCooldown,
		w.alertSendRecovered,
	)
	decision := transition.Decision

	_, err = tx.ExecContext(
		ctx,
		`UPDATE alert_state
         SET fail_streak = $1,
             cooldown_until = $2,
             last_alert_at = $3,
             last_status = $4,
             updated_at = NOW()
	         WHERE company_id = $5
	           AND stream_id = $6`,
		transition.NextFailStreak,
		nullTimeToValue(transition.NextCooldownUntil),
		nullTimeToValue(transition.NextLastAlertAt),
		currentStatus,
		job.CompanyID,
		job.StreamID,
	)
	if err != nil {
		return alertDecision{}, err
	}

	if err := tx.Commit(); err != nil {
		return alertDecision{}, err
	}

	if transition.NextCooldownUntil.Valid {
		cooldownCopy := transition.NextCooldownUntil.Time.UTC()
		decision.CooldownUntil = &cooldownCopy
	}

	return decision, nil
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
	return workeralerts.ComputeTransition(
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
