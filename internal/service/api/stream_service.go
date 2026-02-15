package api

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

type StreamListFilter = domain.StreamListFilter
type StreamPatchInput = domain.StreamPatchInput

type StreamStore interface {
	CreateStream(ctx context.Context, companyID int64, projectID int64, name string, url string, isActive bool) (domain.Stream, error)
	ListStreams(ctx context.Context, companyID int64, filter StreamListFilter) ([]domain.Stream, error)
	GetStream(ctx context.Context, companyID int64, streamID int64) (domain.Stream, error)
	PatchStream(ctx context.Context, companyID int64, streamID int64, patch StreamPatchInput) (domain.Stream, error)
	DeleteStream(ctx context.Context, companyID int64, streamID int64) error
}

type CreateStreamInput struct {
	CompanyID int64
	ProjectID int64
	Name      string
	URL       string
	IsActive  *bool
}

type ListStreamsInput struct {
	CompanyID    int64
	ProjectIDRaw string
	IsActiveRaw  string
}

type PatchStreamRequest struct {
	CompanyID int64
	StreamID  int64
	Name      *string
	URL       *string
	IsActive  *bool
}

type StreamService struct {
	store StreamStore
}

func NewStreamService(store StreamStore) *StreamService {
	return &StreamService{store: store}
}

func (s *StreamService) CreateStream(ctx context.Context, input CreateStreamInput) (domain.Stream, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Stream{}, NewValidationError("name is required", map[string]interface{}{"field": "name"})
	}
	url := strings.TrimSpace(input.URL)
	if url == "" {
		return domain.Stream{}, NewValidationError("url is required", map[string]interface{}{"field": "url"})
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	item, err := s.store.CreateStream(ctx, input.CompanyID, input.ProjectID, name, url, isActive)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrStreamProjectMiss) {
		return domain.Stream{}, NewNotFoundError(
			"project not found for company",
			map[string]interface{}{"company_id": input.CompanyID, "project_id": input.ProjectID},
		)
	}
	if errors.Is(err, domain.ErrStreamAlreadyExists) {
		return domain.Stream{}, NewConflictError(
			"stream with the same name already exists in this project",
			map[string]interface{}{"company_id": input.CompanyID, "project_id": input.ProjectID, "field": "name"},
		)
	}
	return domain.Stream{}, NewInternalError()
}

func (s *StreamService) ListStreams(ctx context.Context, input ListStreamsInput) ([]domain.Stream, error) {
	filter, err := parseStreamListFilter(input.ProjectIDRaw, input.IsActiveRaw)
	if err != nil {
		return nil, err
	}
	items, storeErr := s.store.ListStreams(ctx, input.CompanyID, filter)
	if storeErr != nil {
		return nil, NewInternalError()
	}
	return items, nil
}

func (s *StreamService) GetStream(ctx context.Context, companyID int64, streamID int64) (domain.Stream, error) {
	item, err := s.store.GetStream(ctx, companyID, streamID)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrStreamNotFound) {
		return domain.Stream{}, NewNotFoundError(
			"stream not found",
			map[string]interface{}{"company_id": companyID, "stream_id": streamID},
		)
	}
	return domain.Stream{}, NewInternalError()
}

func (s *StreamService) PatchStream(ctx context.Context, request PatchStreamRequest) (domain.Stream, error) {
	patchInput, err := normalizeStreamPatchInput(request)
	if err != nil {
		return domain.Stream{}, err
	}

	item, patchErr := s.store.PatchStream(ctx, request.CompanyID, request.StreamID, patchInput)
	if patchErr == nil {
		return item, nil
	}
	if errors.Is(patchErr, domain.ErrStreamNotFound) {
		return domain.Stream{}, NewNotFoundError(
			"stream not found",
			map[string]interface{}{"company_id": request.CompanyID, "stream_id": request.StreamID},
		)
	}
	if errors.Is(patchErr, domain.ErrStreamAlreadyExists) {
		return domain.Stream{}, NewConflictError(
			"stream with the same name already exists in this project",
			map[string]interface{}{"company_id": request.CompanyID, "stream_id": request.StreamID, "field": "name"},
		)
	}
	return domain.Stream{}, NewInternalError()
}

func (s *StreamService) DeleteStream(ctx context.Context, companyID int64, streamID int64) error {
	err := s.store.DeleteStream(ctx, companyID, streamID)
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrStreamNotFound) {
		return NewNotFoundError(
			"stream not found",
			map[string]interface{}{"company_id": companyID, "stream_id": streamID},
		)
	}
	return NewInternalError()
}

func parseStreamListFilter(projectIDRaw string, isActiveRaw string) (StreamListFilter, error) {
	filter := StreamListFilter{}

	projectIDRaw = strings.TrimSpace(projectIDRaw)
	if projectIDRaw != "" {
		projectID, err := parsePositiveID(projectIDRaw)
		if err != nil {
			return StreamListFilter{}, NewValidationError(
				"invalid project_id filter",
				map[string]interface{}{"project_id": projectIDRaw},
			)
		}
		filter.ProjectID = &projectID
	}

	isActiveRaw = strings.TrimSpace(isActiveRaw)
	if isActiveRaw != "" {
		isActive, err := strconv.ParseBool(isActiveRaw)
		if err != nil {
			return StreamListFilter{}, NewValidationError(
				"invalid is_active filter",
				map[string]interface{}{"is_active": isActiveRaw},
			)
		}
		filter.IsActive = &isActive
	}

	return filter, nil
}

func normalizeStreamPatchInput(request PatchStreamRequest) (StreamPatchInput, error) {
	var patch StreamPatchInput
	hasChange := false

	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" {
			return StreamPatchInput{}, NewValidationError(
				"name must not be empty",
				map[string]interface{}{"field": "name"},
			)
		}
		patch.Name = &name
		hasChange = true
	}

	if request.URL != nil {
		url := strings.TrimSpace(*request.URL)
		if url == "" {
			return StreamPatchInput{}, NewValidationError(
				"url must not be empty",
				map[string]interface{}{"field": "url"},
			)
		}
		patch.URL = &url
		hasChange = true
	}

	if request.IsActive != nil {
		patch.IsActive = request.IsActive
		hasChange = true
	}

	if !hasChange {
		return StreamPatchInput{}, NewValidationError(
			"at least one field is required",
			map[string]interface{}{"fields": []string{"name", "url", "is_active"}},
		)
	}

	return patch, nil
}

func parsePositiveID(rawID string) (int64, error) {
	value, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid id")
	}
	return value, nil
}
