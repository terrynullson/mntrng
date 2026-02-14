package postgres

import (
	"context"
	"database/sql"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (r *WorkerRepo) ClaimNextQueuedJob(ctx context.Context) (domain.WorkerClaimedJob, bool, error) {
	row := r.db.QueryRowContext(
		ctx,
		`WITH candidate AS (
             SELECT id, company_id
             FROM check_jobs
             WHERE status = $1
             ORDER BY planned_at ASC, id ASC
             FOR UPDATE SKIP LOCKED
             LIMIT 1
         )
         UPDATE check_jobs AS j
         SET status = $2,
             started_at = NOW(),
             finished_at = NULL,
             error_message = NULL
         FROM candidate AS c
         WHERE j.id = c.id
           AND j.company_id = c.company_id
           AND j.status = $1
         RETURNING j.id, j.company_id, j.stream_id, j.planned_at`,
		domain.WorkerJobStatusQueued,
		domain.WorkerJobStatusRunning,
	)

	var job domain.WorkerClaimedJob
	err := row.Scan(&job.ID, &job.CompanyID, &job.StreamID, &job.PlannedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.WorkerClaimedJob{}, false, nil
		}
		return domain.WorkerClaimedJob{}, false, err
	}
	return job, true, nil
}

func (r *WorkerRepo) FinalizeJob(ctx context.Context, job domain.WorkerClaimedJob, status string, errorMessage string) (int64, error) {
	var nullableErrorMessage interface{}
	if errorMessage == "" {
		nullableErrorMessage = nil
	} else {
		nullableErrorMessage = errorMessage
	}

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE check_jobs
         SET status = $1,
             finished_at = NOW(),
             error_message = $2
         WHERE id = $3
           AND company_id = $4
           AND status = $5`,
		status,
		nullableErrorMessage,
		job.ID,
		job.CompanyID,
		domain.WorkerJobStatusRunning,
	)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}
