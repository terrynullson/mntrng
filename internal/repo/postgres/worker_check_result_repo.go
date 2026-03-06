package postgres

import (
	"context"
	"errors"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

func (r *WorkerRepo) PersistCheckResult(
	ctx context.Context,
	job domain.WorkerClaimedJob,
	dbStatus string,
	checksJSON string,
) error {
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO check_results (company_id, job_id, stream_id, status, checks)
         VALUES ($1, $2, $3, $4, $5::jsonb)
         ON CONFLICT (job_id) DO NOTHING`,
		job.CompanyID,
		job.ID,
		job.StreamID,
		dbStatus,
		checksJSON,
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
		countErr := r.db.QueryRowContext(
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

	return nil
}
