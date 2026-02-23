package postgres

import (
	"context"
)

func (r *WorkerRepo) SaveAIIncidentResult(
	ctx context.Context,
	jobID int64,
	companyID int64,
	streamID int64,
	cause string,
	summary string,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO ai_incident_results (job_id, company_id, stream_id, cause, summary)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (job_id) DO UPDATE SET cause = $4, summary = $5`,
		jobID,
		companyID,
		streamID,
		nullIfEmpty(cause),
		nullIfEmpty(summary),
	)
	if err != nil {
		return err
	}
	return nil
}

// nullIfEmpty returns nil for empty string for nullable TEXT columns.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
