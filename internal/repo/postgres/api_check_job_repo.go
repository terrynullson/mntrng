package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

type APICheckJobRepo struct {
	db *sql.DB
}

func NewAPICheckJobRepo(db *sql.DB) *APICheckJobRepo {
	return &APICheckJobRepo{db: db}
}

func (r *APICheckJobRepo) EnqueueCheckJob(
	ctx context.Context,
	companyID int64,
	streamID int64,
	plannedAt time.Time,
) (domain.CheckJob, error) {
	var item domain.CheckJob
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO check_jobs (company_id, stream_id, planned_at)
         SELECT $1, $2, $3
         WHERE EXISTS (
             SELECT 1 FROM streams s
             WHERE s.company_id = $1 AND s.id = $2
         )
         RETURNING id, company_id, stream_id, planned_at, status, created_at, started_at, finished_at, error_message`,
		companyID,
		streamID,
		plannedAt,
	).Scan(
		&item.ID,
		&item.CompanyID,
		&item.StreamID,
		&item.PlannedAt,
		&item.Status,
		&item.CreatedAt,
		&item.StartedAt,
		&item.FinishedAt,
		&item.ErrorMessage,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || isForeignKeyViolation(err) {
			return domain.CheckJob{}, domain.ErrCheckJobStreamMissing
		}
		if isUniqueViolation(err) {
			return domain.CheckJob{}, domain.ErrCheckJobConflict
		}
		return domain.CheckJob{}, err
	}
	return item, nil
}

func (r *APICheckJobRepo) GetCheckJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckJob, error) {
	var item domain.CheckJob
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, stream_id, planned_at, status, created_at, started_at, finished_at, error_message
         FROM check_jobs
         WHERE company_id = $1 AND id = $2`,
		companyID,
		jobID,
	).Scan(
		&item.ID,
		&item.CompanyID,
		&item.StreamID,
		&item.PlannedAt,
		&item.Status,
		&item.CreatedAt,
		&item.StartedAt,
		&item.FinishedAt,
		&item.ErrorMessage,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CheckJob{}, domain.ErrCheckJobNotFound
		}
		return domain.CheckJob{}, err
	}
	return item, nil
}

func (r *APICheckJobRepo) StreamExistsForCheckJobs(ctx context.Context, companyID int64, streamID int64) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM streams WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func (r *APICheckJobRepo) ListCheckJobs(
	ctx context.Context,
	companyID int64,
	streamID int64,
	filter domain.CheckJobListFilter,
) ([]domain.CheckJob, error) {
	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", nextPlaceholder))
		args = append(args, *filter.Status)
		nextPlaceholder++
	}
	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("planned_at >= $%d", nextPlaceholder))
		args = append(args, *filter.From)
		nextPlaceholder++
	}
	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("planned_at <= $%d", nextPlaceholder))
		args = append(args, *filter.To)
		nextPlaceholder++
	}

	query := fmt.Sprintf(
		`SELECT id, company_id, stream_id, planned_at, status, created_at, started_at, finished_at, error_message
         FROM check_jobs
         WHERE %s
         ORDER BY planned_at DESC, id DESC`,
		strings.Join(conditions, " AND "),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.CheckJob, 0)
	for rows.Next() {
		var item domain.CheckJob
		if err := rows.Scan(
			&item.ID,
			&item.CompanyID,
			&item.StreamID,
			&item.PlannedAt,
			&item.Status,
			&item.CreatedAt,
			&item.StartedAt,
			&item.FinishedAt,
			&item.ErrorMessage,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
