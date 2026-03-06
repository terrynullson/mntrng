package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/terrynullson/mntrng/internal/domain"
)

type APICheckResultRepo struct {
	db *sql.DB
}

func NewAPICheckResultRepo(db *sql.DB) *APICheckResultRepo {
	return &APICheckResultRepo{db: db}
}

func (r *APICheckResultRepo) GetCheckResultByID(ctx context.Context, companyID int64, resultID int64) (domain.CheckResult, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, job_id, stream_id, status, checks, screenshot_path, created_at
         FROM check_results
         WHERE company_id = $1 AND id = $2`,
		companyID,
		resultID,
	)
	item, err := scanCheckResult(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CheckResult{}, domain.ErrCheckResultNotFound
		}
		return domain.CheckResult{}, err
	}
	return item, nil
}

func (r *APICheckResultRepo) GetCheckResultByJobID(ctx context.Context, companyID int64, jobID int64) (domain.CheckResult, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, job_id, stream_id, status, checks, screenshot_path, created_at
         FROM check_results
         WHERE company_id = $1 AND job_id = $2`,
		companyID,
		jobID,
	)
	item, err := scanCheckResult(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CheckResult{}, domain.ErrCheckResultByJobNotFound
		}
		return domain.CheckResult{}, err
	}
	return item, nil
}

func (r *APICheckResultRepo) StreamExistsForCheckResults(ctx context.Context, companyID int64, streamID int64) (bool, error) {
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

func (r *APICheckResultRepo) ListCheckResults(
	ctx context.Context,
	companyID int64,
	streamID int64,
	filter domain.CheckResultListFilter,
) ([]domain.CheckResult, error) {
	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", nextPlaceholder))
		args = append(args, *filter.Status)
		nextPlaceholder++
	}
	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", nextPlaceholder))
		args = append(args, *filter.From)
		nextPlaceholder++
	}
	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", nextPlaceholder))
		args = append(args, *filter.To)
		nextPlaceholder++
	}

	query := fmt.Sprintf(
		`SELECT id, company_id, job_id, stream_id, status, checks, screenshot_path, created_at
         FROM check_results
         WHERE %s
         ORDER BY created_at DESC, id DESC`,
		strings.Join(conditions, " AND "),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.CheckResult, 0)
	for rows.Next() {
		item, err := scanCheckResult(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanCheckResult(scanner rowScanner) (domain.CheckResult, error) {
	var item domain.CheckResult
	var dbStatus string
	var checksRaw []byte

	err := scanner.Scan(
		&item.ID,
		&item.CompanyID,
		&item.JobID,
		&item.StreamID,
		&dbStatus,
		&checksRaw,
		&item.ScreenshotPath,
		&item.CreatedAt,
	)
	if err != nil {
		return domain.CheckResult{}, err
	}

	if len(checksRaw) == 0 {
		checksRaw = []byte("{}")
	}
	item.Checks = json.RawMessage(checksRaw)
	item.Status = formatCheckResultStatus(dbStatus)
	return item, nil
}

func formatCheckResultStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ok":
		return "OK"
	case "warn":
		return "WARN"
	case "fail":
		return "FAIL"
	default:
		return strings.ToUpper(strings.TrimSpace(raw))
	}
}
