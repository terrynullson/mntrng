package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

// WorkerIncidentRepo manages incidents from the worker (create/update/resolve).
type WorkerIncidentRepo struct {
	db *sql.DB
}

// NewWorkerIncidentRepo returns a new WorkerIncidentRepo.
func NewWorkerIncidentRepo(db *sql.DB) *WorkerIncidentRepo {
	return &WorkerIncidentRepo{db: db}
}

// GetOpenByStream returns the open incident id for the stream in tenant scope, or false.
func (r *WorkerIncidentRepo) GetOpenByStream(ctx context.Context, companyID int64, streamID int64) (incident domain.Incident, ok bool, err error) {
	var diagCode sql.NullString
	var diagDetails []byte
	var resolvedAt sql.NullTime
	var failReason sql.NullString
	var screenshotPath sql.NullString
	var screenshotTakenAt sql.NullTime
	err = r.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, stream_id, status, severity, started_at, last_event_at, resolved_at,
                fail_reason, sample_screenshot_path, screenshot_taken_at, diag_code, COALESCE(diag_details, '{}'::jsonb)
         FROM incidents
         WHERE company_id = $1 AND stream_id = $2 AND status = $3`,
		companyID,
		streamID,
		domain.IncidentStatusOpen,
	).Scan(
		&incident.ID,
		&incident.CompanyID,
		&incident.StreamID,
		&incident.Status,
		&incident.Severity,
		&incident.StartedAt,
		&incident.LastEventAt,
		&resolvedAt,
		&failReason,
		&screenshotPath,
		&screenshotTakenAt,
		&diagCode,
		&diagDetails,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Incident{}, false, nil
		}
		return domain.Incident{}, false, err
	}
	if resolvedAt.Valid {
		incident.ResolvedAt = &resolvedAt.Time
	}
	if failReason.Valid {
		incident.FailReason = &failReason.String
	}
	if screenshotPath.Valid {
		incident.SampleScreenshotPath = &screenshotPath.String
	}
	if screenshotTakenAt.Valid {
		incident.ScreenshotTakenAt = &screenshotTakenAt.Time
	}
	if diagCode.Valid {
		code := diagCode.String
		incident.DiagCode = &code
	}
	if len(diagDetails) > 0 {
		incident.DiagDetails = json.RawMessage(diagDetails)
	}
	if incident.SampleScreenshotPath != nil && *incident.SampleScreenshotPath != "" {
		incident.HasScreenshot = true
	}
	return incident, true, nil
}

// Create creates a new open incident and writes audit log.
func (r *WorkerIncidentRepo) Create(
	ctx context.Context,
	companyID int64,
	streamID int64,
	severity string,
	failReason string,
	sampleScreenshotPath *string,
	lastCheckID *int64,
) (incidentID int64, err error) {
	now := time.Now().UTC()
	err = r.db.QueryRowContext(
		ctx,
		`INSERT INTO incidents (company_id, stream_id, status, severity, started_at, last_event_at, fail_reason, sample_screenshot_path, screenshot_taken_at, diag_code, diag_details, last_check_id)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, NULL, NULL, $9)
         RETURNING id`,
		companyID,
		streamID,
		domain.IncidentStatusOpen,
		severity,
		now,
		now,
		nullString(failReason),
		nullStringPtr(sampleScreenshotPath),
		lastCheckID,
	).Scan(&incidentID)
	if err != nil {
		return 0, err
	}
	auditErr := InsertAuditLog(
		ctx,
		r.db,
		companyID,
		domain.AuditActorTypeWorker,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeIncident,
		incidentID,
		domain.AuditActionIncidentOpen,
		map[string]interface{}{
			"stream_id":   streamID,
			"severity":    severity,
			"fail_reason": failReason,
		},
	)
	if auditErr != nil {
		return incidentID, auditErr
	}
	return incidentID, nil
}

// UpdateOpen updates last_event_at, fail_reason, sample_screenshot_path, last_check_id.
// Severity is only upgraded (warn -> fail), never downgraded.
func (r *WorkerIncidentRepo) UpdateOpen(
	ctx context.Context,
	incidentID int64,
	companyID int64,
	severity string,
	failReason string,
	sampleScreenshotPath *string,
	lastCheckID *int64,
) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE incidents
         SET last_event_at = $1,
             severity = CASE WHEN $2 = 'fail' THEN 'fail' ELSE severity END,
             fail_reason = $3,
             sample_screenshot_path = COALESCE($4, sample_screenshot_path),
             last_check_id = COALESCE($5, last_check_id)
         WHERE id = $6 AND company_id = $7 AND status = $8`,
		now,
		severity,
		nullString(failReason),
		sampleScreenshotPath,
		lastCheckID,
		incidentID,
		companyID,
		domain.IncidentStatusOpen,
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrIncidentNotFound
	}
	return nil
}

// Resolve sets status=resolved, resolved_at=now and writes audit log.
func (r *WorkerIncidentRepo) Resolve(ctx context.Context, incidentID int64, companyID int64, streamID int64) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE incidents SET status = $1, resolved_at = $2 WHERE id = $3 AND company_id = $4 AND status = $5`,
		domain.IncidentStatusResolved,
		now,
		incidentID,
		companyID,
		domain.IncidentStatusOpen,
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrIncidentNotFound
	}
	return InsertAuditLog(
		ctx,
		r.db,
		companyID,
		domain.AuditActorTypeWorker,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeIncident,
		incidentID,
		domain.AuditActionIncidentResolve,
		map[string]interface{}{"stream_id": streamID},
	)
}

func (r *WorkerIncidentRepo) UpdateDiagnostic(
	ctx context.Context,
	incidentID int64,
	companyID int64,
	streamID int64,
	sampleScreenshotPath *string,
	screenshotTakenAt time.Time,
	diagCode string,
	diagDetails map[string]interface{},
) error {
	payload, err := json.Marshal(diagDetails)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE incidents
         SET sample_screenshot_path = COALESCE($1, sample_screenshot_path),
             screenshot_taken_at = $2,
             diag_code = $3,
             diag_details = $4::jsonb
         WHERE id = $5 AND company_id = $6`,
		nullStringPtr(sampleScreenshotPath),
		screenshotTakenAt.UTC(),
		diagCode,
		string(payload),
		incidentID,
		companyID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrIncidentNotFound
	}
	return InsertAuditLog(
		ctx,
		r.db,
		companyID,
		domain.AuditActorTypeWorker,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeIncident,
		incidentID,
		domain.AuditActionIncidentDiagnosticUpdated,
		map[string]interface{}{
			"stream_id":        streamID,
			"diag_code":        diagCode,
			"screenshot_path":  sampleScreenshotPath,
			"screenshot_taken": screenshotTakenAt.UTC().Format(time.RFC3339),
		},
	)
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullStringPtr(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}
