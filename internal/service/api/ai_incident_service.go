package api

import (
	"context"
	"errors"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

// AIIncidentStore reads AI incident results in tenant scope.
type AIIncidentStore interface {
	GetByCompanyStreamJob(ctx context.Context, companyID int64, streamID int64, jobID int64) (domain.AIIncidentResponse, error)
}

type AIIncidentService struct {
	store AIIncidentStore
}

func NewAIIncidentService(store AIIncidentStore) *AIIncidentService {
	return &AIIncidentService{store: store}
}

// Get returns cause and summary for the given job in tenant scope (company_id, stream_id).
// Returns NotFound when no AI incident result exists for that job.
func (s *AIIncidentService) Get(ctx context.Context, companyID int64, streamID int64, jobID int64) (domain.AIIncidentResponse, error) {
	out, err := s.store.GetByCompanyStreamJob(ctx, companyID, streamID, jobID)
	if err == nil {
		return out, nil
	}
	if errors.Is(err, domain.ErrAIIncidentNotFound) {
		return domain.AIIncidentResponse{}, NewNotFoundError(
			"ai incident result not found for job",
			map[string]interface{}{"company_id": companyID, "stream_id": streamID, "job_id": jobID},
		)
	}
	return domain.AIIncidentResponse{}, NewInternalError()
}
