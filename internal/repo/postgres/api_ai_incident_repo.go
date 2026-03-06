package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

type APIAIIncidentRepo struct {
	db *sql.DB
}

func NewAPIAIIncidentRepo(db *sql.DB) *APIAIIncidentRepo {
	return &APIAIIncidentRepo{db: db}
}

// GetByCompanyStreamJob returns cause and summary for the given job in tenant scope.
// Returns domain.ErrAIIncidentNotFound when no row exists for company_id, stream_id, job_id.
func (r *APIAIIncidentRepo) GetByCompanyStreamJob(ctx context.Context, companyID int64, streamID int64, jobID int64) (domain.AIIncidentResponse, error) {
	var cause, summary sql.NullString
	err := r.db.QueryRowContext(
		ctx,
		`SELECT cause, summary FROM ai_incident_results
         WHERE company_id = $1 AND stream_id = $2 AND job_id = $3`,
		companyID,
		streamID,
		jobID,
	).Scan(&cause, &summary)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AIIncidentResponse{}, domain.ErrAIIncidentNotFound
		}
		return domain.AIIncidentResponse{}, err
	}
	out := domain.AIIncidentResponse{}
	if cause.Valid {
		out.Cause = cause.String
	}
	if summary.Valid {
		out.Summary = summary.String
	}
	return out, nil
}
