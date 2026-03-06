package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

// APIIncidentRepo manages incidents for API (tenant-scoped).
type APIIncidentRepo struct {
	db *sql.DB
}

// NewAPIIncidentRepo returns a new APIIncidentRepo.
func NewAPIIncidentRepo(db *sql.DB) *APIIncidentRepo {
	return &APIIncidentRepo{db: db}
}

// List returns incidents for company with filter and pagination.
func (r *APIIncidentRepo) List(
	ctx context.Context,
	companyID int64,
	filter domain.IncidentListFilter,
) ([]domain.Incident, int64, error) {
	conditions := []string{"i.company_id = $1"}
	args := []interface{}{companyID}
	nextPlaceholder := 2

	if filter.Status != nil && *filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("i.status = $%d", nextPlaceholder))
		args = append(args, *filter.Status)
		nextPlaceholder++
	}
	if filter.Severity != nil && *filter.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("i.severity = $%d", nextPlaceholder))
		args = append(args, *filter.Severity)
		nextPlaceholder++
	}
	if filter.StreamID != nil {
		conditions = append(conditions, fmt.Sprintf("i.stream_id = $%d", nextPlaceholder))
		args = append(args, *filter.StreamID)
		nextPlaceholder++
	}
	if filter.Q != "" {
		conditions = append(conditions, fmt.Sprintf("(s.name ILIKE $%d OR i.fail_reason ILIKE $%d)", nextPlaceholder, nextPlaceholder))
		args = append(args, "%"+filter.Q+"%")
		nextPlaceholder++
	}

	whereClause := strings.Join(conditions, " AND ")
	baseQuery := `FROM incidents i JOIN streams s ON s.id = i.stream_id AND s.company_id = i.company_id WHERE ` + whereClause

	var total int64
	countQuery := `SELECT COUNT(*) ` + baseQuery
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := filter.Page
	if page < 0 {
		page = 0
	}
	offset := page * pageSize

	listQuery := fmt.Sprintf(
		`SELECT i.id, i.company_id, i.stream_id, s.name,
                i.status, i.severity, i.started_at, i.last_event_at, i.resolved_at,
                i.fail_reason, i.sample_screenshot_path, i.last_check_id, i.screenshot_taken_at, i.diag_code, COALESCE(i.diag_details, '{}'::jsonb)
         %s
         ORDER BY i.last_event_at DESC
         LIMIT $%d OFFSET $%d`,
		baseQuery, nextPlaceholder, nextPlaceholder+1,
	)
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []domain.Incident
	for rows.Next() {
		var inc domain.Incident
		var resolvedAt sql.NullTime
		var failReason, screenshotPath sql.NullString
		var lastCheckID sql.NullInt64
		var screenshotTakenAt sql.NullTime
		var diagCode sql.NullString
		var diagDetails []byte
		if err := rows.Scan(
			&inc.ID,
			&inc.CompanyID,
			&inc.StreamID,
			&inc.StreamName,
			&inc.Status,
			&inc.Severity,
			&inc.StartedAt,
			&inc.LastEventAt,
			&resolvedAt,
			&failReason,
			&screenshotPath,
			&lastCheckID,
			&screenshotTakenAt,
			&diagCode,
			&diagDetails,
		); err != nil {
			return nil, 0, err
		}
		if resolvedAt.Valid {
			inc.ResolvedAt = &resolvedAt.Time
		}
		if failReason.Valid {
			inc.FailReason = &failReason.String
		}
		if screenshotPath.Valid {
			inc.SampleScreenshotPath = &screenshotPath.String
			inc.HasScreenshot = strings.TrimSpace(screenshotPath.String) != ""
		}
		if lastCheckID.Valid {
			inc.LastCheckID = &lastCheckID.Int64
		}
		if screenshotTakenAt.Valid {
			inc.ScreenshotTakenAt = &screenshotTakenAt.Time
		}
		if diagCode.Valid {
			inc.DiagCode = &diagCode.String
		}
		if len(diagDetails) > 0 {
			inc.DiagDetails = diagDetails
		}
		items = append(items, inc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns incident by id in tenant scope.
func (r *APIIncidentRepo) GetByID(ctx context.Context, companyID int64, incidentID int64) (domain.Incident, error) {
	var inc domain.Incident
	var resolvedAt sql.NullTime
	var failReason, screenshotPath sql.NullString
	var lastCheckID sql.NullInt64
	var screenshotTakenAt sql.NullTime
	var diagCode sql.NullString
	var diagDetails []byte
	err := r.db.QueryRowContext(
		ctx,
		`SELECT i.id, i.company_id, i.stream_id, s.name,
                i.status, i.severity, i.started_at, i.last_event_at, i.resolved_at,
                i.fail_reason, i.sample_screenshot_path, i.last_check_id, i.screenshot_taken_at, i.diag_code, COALESCE(i.diag_details, '{}'::jsonb)
         FROM incidents i
         JOIN streams s ON s.id = i.stream_id AND s.company_id = i.company_id
         WHERE i.company_id = $1 AND i.id = $2`,
		companyID,
		incidentID,
	).Scan(
		&inc.ID,
		&inc.CompanyID,
		&inc.StreamID,
		&inc.StreamName,
		&inc.Status,
		&inc.Severity,
		&inc.StartedAt,
		&inc.LastEventAt,
		&resolvedAt,
		&failReason,
		&screenshotPath,
		&lastCheckID,
		&screenshotTakenAt,
		&diagCode,
		&diagDetails,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Incident{}, domain.ErrIncidentNotFound
		}
		return domain.Incident{}, err
	}
	if resolvedAt.Valid {
		inc.ResolvedAt = &resolvedAt.Time
	}
	if failReason.Valid {
		inc.FailReason = &failReason.String
	}
	if screenshotPath.Valid {
		inc.SampleScreenshotPath = &screenshotPath.String
		inc.HasScreenshot = strings.TrimSpace(screenshotPath.String) != ""
	}
	if lastCheckID.Valid {
		inc.LastCheckID = &lastCheckID.Int64
	}
	if screenshotTakenAt.Valid {
		inc.ScreenshotTakenAt = &screenshotTakenAt.Time
	}
	if diagCode.Valid {
		inc.DiagCode = &diagCode.String
	}
	if len(diagDetails) > 0 {
		inc.DiagDetails = diagDetails
	}
	return inc, nil
}
