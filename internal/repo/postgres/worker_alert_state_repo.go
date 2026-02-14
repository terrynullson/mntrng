package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

type alertStateSnapshot struct {
	FailStreak    int
	CooldownUntil sql.NullTime
	LastAlertAt   sql.NullTime
	PreviousState string
}

func (r *WorkerRepo) ApplyAlertState(
	ctx context.Context,
	companyID int64,
	streamID int64,
	currentStatus string,
	failStreakThreshold int,
	alertCooldown time.Duration,
	alertSendRecovered bool,
) (domain.WorkerAlertDecision, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.WorkerAlertDecision{}, err
	}
	defer tx.Rollback()

	err = ensureAlertStateRow(ctx, tx, companyID, streamID)
	if err != nil {
		return domain.WorkerAlertDecision{}, err
	}

	snapshot, err := loadAlertStateSnapshot(ctx, tx, companyID, streamID)
	if err != nil {
		return domain.WorkerAlertDecision{}, err
	}

	now := time.Now().UTC()
	transition := domain.ComputeWorkerAlertTransition(
		now,
		currentStatus,
		snapshot.PreviousState,
		snapshot.FailStreak,
		snapshot.CooldownUntil,
		snapshot.LastAlertAt,
		failStreakThreshold,
		alertCooldown,
		alertSendRecovered,
	)
	decision := transition.Decision

	err = updateAlertStateSnapshot(
		ctx,
		tx,
		companyID,
		streamID,
		currentStatus,
		transition.NextFailStreak,
		transition.NextCooldownUntil,
		transition.NextLastAlertAt,
	)
	if err != nil {
		return domain.WorkerAlertDecision{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.WorkerAlertDecision{}, err
	}

	if transition.NextCooldownUntil.Valid {
		cooldownCopy := transition.NextCooldownUntil.Time.UTC()
		decision.CooldownUntil = &cooldownCopy
	}

	return decision, nil
}

func ensureAlertStateRow(ctx context.Context, tx *sql.Tx, companyID int64, streamID int64) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO alert_state (company_id, stream_id, fail_streak, cooldown_until, last_alert_at, last_status, created_at, updated_at)
         VALUES ($1, $2, 0, NULL, NULL, NULL, NOW(), NOW())
         ON CONFLICT (stream_id) DO NOTHING`,
		companyID,
		streamID,
	)
	return err
}

func loadAlertStateSnapshot(ctx context.Context, tx *sql.Tx, companyID int64, streamID int64) (alertStateSnapshot, error) {
	var snapshot alertStateSnapshot
	var previousStatusRaw sql.NullString
	scanErr := tx.QueryRowContext(
		ctx,
		`SELECT fail_streak, cooldown_until, last_alert_at, last_status
         FROM alert_state
         WHERE company_id = $1
           AND stream_id = $2
         FOR UPDATE`,
		companyID,
		streamID,
	).Scan(&snapshot.FailStreak, &snapshot.CooldownUntil, &snapshot.LastAlertAt, &previousStatusRaw)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return alertStateSnapshot{}, errors.New("alert_state row not found in tenant context")
		}
		return alertStateSnapshot{}, scanErr
	}

	if previousStatusRaw.Valid {
		normalizedPrevious, prevErr := normalizeAlertStatusForRepo(previousStatusRaw.String)
		if prevErr == nil {
			snapshot.PreviousState = normalizedPrevious
		}
	}

	return snapshot, nil
}

func updateAlertStateSnapshot(
	ctx context.Context,
	tx *sql.Tx,
	companyID int64,
	streamID int64,
	currentStatus string,
	nextFailStreak int,
	nextCooldownUntil sql.NullTime,
	nextLastAlertAt sql.NullTime,
) error {
	_, err := tx.ExecContext(
		ctx,
		`UPDATE alert_state
         SET fail_streak = $1,
             cooldown_until = $2,
             last_alert_at = $3,
             last_status = $4,
             updated_at = NOW()
	         WHERE company_id = $5
	           AND stream_id = $6`,
		nextFailStreak,
		nullTimeToValue(nextCooldownUntil),
		nullTimeToValue(nextLastAlertAt),
		currentStatus,
		companyID,
		streamID,
	)
	return err
}

func normalizeAlertStatusForRepo(statusRaw string) (string, error) {
	normalized := checkStatusToDBStatusForRepo(statusRaw)
	switch normalized {
	case domain.WorkerStatusDBOK, domain.WorkerStatusDBWarn, domain.WorkerStatusDBFail:
		return normalized, nil
	default:
		return "", errors.New("unsupported alert status: " + statusRaw)
	}
}

func checkStatusToDBStatusForRepo(status string) string {
	switch strings.TrimSpace(strings.ToUpper(status)) {
	case domain.WorkerStatusOK:
		return domain.WorkerStatusDBOK
	case domain.WorkerStatusWarn:
		return domain.WorkerStatusDBWarn
	case domain.WorkerStatusFail:
		return domain.WorkerStatusDBFail
	default:
		return strings.TrimSpace(strings.ToLower(status))
	}
}

func nullTimeToValue(value sql.NullTime) interface{} {
	if !value.Valid {
		return nil
	}
	return value.Time
}
