package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

func (r *WorkerRepo) ListCompanyIDsForRetention(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id
         FROM companies
         ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	companyIDs := make([]int64, 0)
	for rows.Next() {
		var companyID int64
		if err := rows.Scan(&companyID); err != nil {
			return nil, err
		}
		companyIDs = append(companyIDs, companyID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return companyIDs, nil
}

func (r *WorkerRepo) ListRetentionCandidates(
	ctx context.Context,
	companyID int64,
	cutoff time.Time,
	batchSize int,
) ([]domain.WorkerRetentionCandidate, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, screenshot_path
         FROM check_results
         WHERE company_id = $1
           AND created_at < $2
         ORDER BY created_at ASC, id ASC
         LIMIT $3`,
		companyID,
		cutoff,
		batchSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]domain.WorkerRetentionCandidate, 0, batchSize)
	for rows.Next() {
		var candidate domain.WorkerRetentionCandidate
		var screenshotPath sql.NullString
		if err := rows.Scan(&candidate.ID, &screenshotPath); err != nil {
			return nil, err
		}
		if screenshotPath.Valid {
			candidate.ScreenshotPath = strings.TrimSpace(screenshotPath.String)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (r *WorkerRepo) DeleteStaleCheckResult(ctx context.Context, companyID int64, resultID int64, cutoff time.Time) (int64, error) {
	result, err := r.db.ExecContext(
		ctx,
		`DELETE FROM check_results
         WHERE company_id = $1
           AND id = $2
           AND created_at < $3`,
		companyID,
		resultID,
		cutoff,
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
