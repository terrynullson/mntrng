package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
)

type CheckJobListFilter = domain.CheckJobListFilter

type CheckJobStore interface {
	EnqueueCheckJob(ctx context.Context, companyID int64, streamID int64, plannedAt time.Time) (domain.CheckJob, error)
	GetCheckJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckJob, error)
	StreamExistsForCheckJobs(ctx context.Context, companyID int64, streamID int64) (bool, error)
	ListCheckJobs(ctx context.Context, companyID int64, streamID int64, filter CheckJobListFilter) ([]domain.CheckJob, error)
}

type EnqueueCheckJobInput struct {
	CompanyID    int64
	StreamID     int64
	PlannedAtRaw string
}

type ListCheckJobsInput struct {
	CompanyID int64
	StreamID  int64
	StatusRaw string
	FromRaw   string
	ToRaw     string
}

type CheckJobService struct {
	store CheckJobStore
}

func NewCheckJobService(store CheckJobStore) *CheckJobService {
	return &CheckJobService{store: store}
}

func (s *CheckJobService) EnqueueCheckJob(ctx context.Context, input EnqueueCheckJobInput) (domain.CheckJob, error) {
	plannedAtRaw := strings.TrimSpace(input.PlannedAtRaw)
	if plannedAtRaw == "" {
		return domain.CheckJob{}, NewValidationError("planned_at is required", map[string]interface{}{"field": "planned_at"})
	}

	plannedAt, err := time.Parse(time.RFC3339, plannedAtRaw)
	if err != nil {
		return domain.CheckJob{}, NewValidationError(
			"planned_at must be RFC3339 timestamp",
			map[string]interface{}{"field": "planned_at", "value": plannedAtRaw},
		)
	}

	item, err := s.store.EnqueueCheckJob(ctx, input.CompanyID, input.StreamID, plannedAt.UTC())
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrCheckJobStreamMissing) {
		return domain.CheckJob{}, NewNotFoundError(
			"stream not found for company",
			map[string]interface{}{"company_id": input.CompanyID, "stream_id": input.StreamID},
		)
	}
	if errors.Is(err, domain.ErrCheckJobConflict) {
		return domain.CheckJob{}, NewConflictError(
			"check job already exists for stream and planned_at",
			map[string]interface{}{
				"company_id": input.CompanyID,
				"stream_id":  input.StreamID,
				"planned_at": plannedAtRaw,
			},
		)
	}
	return domain.CheckJob{}, NewInternalError()
}

func (s *CheckJobService) GetCheckJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckJob, error) {
	item, err := s.store.GetCheckJob(ctx, companyID, jobID)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrCheckJobNotFound) {
		return domain.CheckJob{}, NewNotFoundError(
			"check job not found",
			map[string]interface{}{"company_id": companyID, "job_id": jobID},
		)
	}
	return domain.CheckJob{}, NewInternalError()
}

func (s *CheckJobService) ListCheckJobs(ctx context.Context, input ListCheckJobsInput) ([]domain.CheckJob, error) {
	streamExists, err := s.store.StreamExistsForCheckJobs(ctx, input.CompanyID, input.StreamID)
	if err != nil {
		return nil, NewInternalError()
	}
	if !streamExists {
		return nil, NewNotFoundError(
			"stream not found for company",
			map[string]interface{}{"company_id": input.CompanyID, "stream_id": input.StreamID},
		)
	}

	filter, err := parseCheckJobListFilter(input.StatusRaw, input.FromRaw, input.ToRaw)
	if err != nil {
		return nil, err
	}

	items, err := s.store.ListCheckJobs(ctx, input.CompanyID, input.StreamID, filter)
	if err != nil {
		return nil, NewInternalError()
	}
	return items, nil
}

func parseCheckJobListFilter(statusRaw string, fromRaw string, toRaw string) (CheckJobListFilter, error) {
	filter := CheckJobListFilter{}

	statusRaw = strings.TrimSpace(statusRaw)
	if statusRaw != "" {
		status, ok := normalizeCheckJobStatus(statusRaw)
		if !ok {
			return CheckJobListFilter{}, NewValidationError(
				"invalid status filter",
				map[string]interface{}{"status": statusRaw, "allowed": []string{"queued", "running", "done", "failed"}},
			)
		}
		filter.Status = &status
	}

	fromRaw = strings.TrimSpace(fromRaw)
	if fromRaw != "" {
		fromTime, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			return CheckJobListFilter{}, NewValidationError("invalid from filter", map[string]interface{}{"from": fromRaw})
		}
		fromTime = fromTime.UTC()
		filter.From = &fromTime
	}

	toRaw = strings.TrimSpace(toRaw)
	if toRaw != "" {
		toTime, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			return CheckJobListFilter{}, NewValidationError("invalid to filter", map[string]interface{}{"to": toRaw})
		}
		toTime = toTime.UTC()
		filter.To = &toTime
	}

	return filter, nil
}

func normalizeCheckJobStatus(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "queued":
		return "queued", true
	case "running":
		return "running", true
	case "done":
		return "done", true
	case "failed":
		return "failed", true
	default:
		return "", false
	}
}
