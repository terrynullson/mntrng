package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

type CheckResultListFilter = domain.CheckResultListFilter

type CheckResultStore interface {
	GetCheckResultByID(ctx context.Context, companyID int64, resultID int64) (domain.CheckResult, error)
	GetCheckResultByJobID(ctx context.Context, companyID int64, jobID int64) (domain.CheckResult, error)
	StreamExistsForCheckResults(ctx context.Context, companyID int64, streamID int64) (bool, error)
	ListCheckResults(ctx context.Context, companyID int64, streamID int64, filter CheckResultListFilter) ([]domain.CheckResult, error)
}

type ListCheckResultsInput struct {
	CompanyID int64
	StreamID  int64
	StatusRaw string
	FromRaw   string
	ToRaw     string
}

type CheckResultService struct {
	store CheckResultStore
}

func NewCheckResultService(store CheckResultStore) *CheckResultService {
	return &CheckResultService{store: store}
}

func (s *CheckResultService) GetCheckResult(ctx context.Context, companyID int64, resultID int64) (domain.CheckResult, error) {
	item, err := s.store.GetCheckResultByID(ctx, companyID, resultID)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrCheckResultNotFound) {
		return domain.CheckResult{}, NewNotFoundError(
			"check result not found",
			map[string]interface{}{"company_id": companyID, "result_id": resultID},
		)
	}
	return domain.CheckResult{}, NewInternalError()
}

func (s *CheckResultService) GetCheckResultByJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckResult, error) {
	item, err := s.store.GetCheckResultByJobID(ctx, companyID, jobID)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrCheckResultByJobNotFound) {
		return domain.CheckResult{}, NewNotFoundError(
			"check result not found for job",
			map[string]interface{}{"company_id": companyID, "job_id": jobID},
		)
	}
	return domain.CheckResult{}, NewInternalError()
}

func (s *CheckResultService) ListCheckResults(ctx context.Context, input ListCheckResultsInput) ([]domain.CheckResult, error) {
	streamExists, err := s.store.StreamExistsForCheckResults(ctx, input.CompanyID, input.StreamID)
	if err != nil {
		return nil, NewInternalError()
	}
	if !streamExists {
		return nil, NewNotFoundError(
			"stream not found for company",
			map[string]interface{}{"company_id": input.CompanyID, "stream_id": input.StreamID},
		)
	}

	filter, err := parseCheckResultListFilter(input.StatusRaw, input.FromRaw, input.ToRaw)
	if err != nil {
		return nil, err
	}

	items, err := s.store.ListCheckResults(ctx, input.CompanyID, input.StreamID, filter)
	if err != nil {
		return nil, NewInternalError()
	}
	return items, nil
}

func parseCheckResultListFilter(statusRaw string, fromRaw string, toRaw string) (CheckResultListFilter, error) {
	filter := CheckResultListFilter{}

	statusRaw = strings.TrimSpace(statusRaw)
	if statusRaw != "" {
		status, ok := normalizeCheckResultStatus(statusRaw)
		if !ok {
			return CheckResultListFilter{}, NewValidationError(
				"invalid status filter",
				map[string]interface{}{"status": statusRaw, "allowed": []string{"OK", "WARN", "FAIL"}},
			)
		}
		filter.Status = &status
	}

	fromRaw = strings.TrimSpace(fromRaw)
	if fromRaw != "" {
		fromTime, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			return CheckResultListFilter{}, NewValidationError("invalid from filter", map[string]interface{}{"from": fromRaw})
		}
		fromTime = fromTime.UTC()
		filter.From = &fromTime
	}

	toRaw = strings.TrimSpace(toRaw)
	if toRaw != "" {
		toTime, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			return CheckResultListFilter{}, NewValidationError("invalid to filter", map[string]interface{}{"to": toRaw})
		}
		toTime = toTime.UTC()
		filter.To = &toTime
	}

	return filter, nil
}

func normalizeCheckResultStatus(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ok":
		return "ok", true
	case "warn":
		return "warn", true
	case "fail":
		return "fail", true
	default:
		return "", false
	}
}
