package api

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/terrynullson/mntrng/internal/domain"
)

type StreamListFilter = domain.StreamListFilter
type StreamPatchInput = domain.StreamPatchInput

type StreamStore interface {
	CreateStream(ctx context.Context, companyID int64, projectID int64, name string, sourceType string, sourceURL string, isActive bool) (domain.Stream, error)
	ListStreams(ctx context.Context, companyID int64, filter StreamListFilter) ([]domain.Stream, error)
	ListLatestStatuses(ctx context.Context, companyID int64) ([]domain.StreamLatestStatus, error)
	GetStream(ctx context.Context, companyID int64, streamID int64) (domain.Stream, error)
	PatchStream(ctx context.Context, companyID int64, streamID int64, patch StreamPatchInput) (domain.Stream, error)
	DeleteStream(ctx context.Context, companyID int64, streamID int64) error
	IsEmbedDomainAllowed(ctx context.Context, companyID int64, host string) (bool, error)
}

type CreateStreamInput struct {
	CompanyID  int64
	ProjectID  int64
	Name       string
	SourceType string
	SourceURL  string
	URL        string
	IsActive   *bool
}

type ListStreamsInput struct {
	CompanyID    int64
	ProjectIDRaw string
	IsActiveRaw  string
}

type PatchStreamRequest struct {
	CompanyID  int64
	StreamID   int64
	Name       *string
	SourceType *string
	SourceURL  *string
	URL        *string
	IsActive   *bool
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
	sourceType, sourceURL, err := normalizeStreamSource(input.SourceType, input.SourceURL, input.URL)
	if err != nil {
		return domain.Stream{}, err
	}
	if sourceType == domain.StreamSourceTypeEmbed {
		ok, validateErr := s.validateEmbedDomain(ctx, input.CompanyID, sourceURL)
		if validateErr != nil {
			return domain.Stream{}, validateErr
		}
		if !ok {
			return domain.Stream{}, NewValidationError("Домен не разрешён в Embed whitelist", map[string]interface{}{"field": "source_url"})
		}
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	item, err := s.store.CreateStream(ctx, input.CompanyID, input.ProjectID, name, sourceType, sourceURL, isActive)
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

func (s *StreamService) ListLatestStatuses(ctx context.Context, companyID int64) ([]domain.StreamLatestStatus, error) {
	items, err := s.store.ListLatestStatuses(ctx, companyID)
	if err != nil {
		return nil, NewInternalError()
	}
	return items, nil
}

func (s *StreamService) PatchStream(ctx context.Context, request PatchStreamRequest) (domain.Stream, error) {
	patchInput, err := normalizeStreamPatchInput(request)
	if err != nil {
		return domain.Stream{}, err
	}
	existing, err := s.store.GetStream(ctx, request.CompanyID, request.StreamID)
	if err != nil {
		if errors.Is(err, domain.ErrStreamNotFound) {
			return domain.Stream{}, NewNotFoundError(
				"stream not found",
				map[string]interface{}{"company_id": request.CompanyID, "stream_id": request.StreamID},
			)
		}
		return domain.Stream{}, NewInternalError()
	}
	targetSourceType := existing.SourceType
	if patchInput.SourceType != nil {
		targetSourceType = *patchInput.SourceType
	}
	targetSourceURL := existing.SourceURL
	if patchInput.SourceURL != nil {
		targetSourceURL = *patchInput.SourceURL
	} else if patchInput.URL != nil {
		targetSourceURL = *patchInput.URL
	}
	if targetSourceType == domain.StreamSourceTypeEmbed {
		ok, validateErr := s.validateEmbedDomain(ctx, request.CompanyID, targetSourceURL)
		if validateErr != nil {
			return domain.Stream{}, validateErr
		}
		if !ok {
			return domain.Stream{}, NewValidationError("Домен не разрешён в Embed whitelist", map[string]interface{}{"field": "source_url"})
		}
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

	if request.SourceType != nil {
		sourceType := strings.ToUpper(strings.TrimSpace(*request.SourceType))
		if sourceType != domain.StreamSourceTypeHLS && sourceType != domain.StreamSourceTypeEmbed {
			return StreamPatchInput{}, NewValidationError(
				"source_type must be HLS or EMBED",
				map[string]interface{}{"field": "source_type"},
			)
		}
		patch.SourceType = &sourceType
		hasChange = true
	}

	if request.SourceURL != nil {
		sourceURL := strings.TrimSpace(*request.SourceURL)
		if sourceURL == "" {
			return StreamPatchInput{}, NewValidationError(
				"source_url must not be empty",
				map[string]interface{}{"field": "source_url"},
			)
		}
		patch.SourceURL = &sourceURL
		patch.URL = &sourceURL
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
		if patch.SourceURL == nil {
			patch.SourceURL = &url
		}
		hasChange = true
	}

	if request.IsActive != nil {
		patch.IsActive = request.IsActive
		hasChange = true
	}

	if !hasChange {
		return StreamPatchInput{}, NewValidationError(
			"at least one field is required",
			map[string]interface{}{"fields": []string{"name", "source_type", "source_url", "url", "is_active"}},
		)
	}

	return patch, nil
}

func normalizeStreamSource(rawSourceType string, rawSourceURL string, rawURL string) (string, string, error) {
	sourceType := strings.ToUpper(strings.TrimSpace(rawSourceType))
	if sourceType == "" {
		sourceType = domain.StreamSourceTypeHLS
	}
	if sourceType != domain.StreamSourceTypeHLS && sourceType != domain.StreamSourceTypeEmbed {
		return "", "", NewValidationError("source_type must be HLS or EMBED", map[string]interface{}{"field": "source_type"})
	}
	sourceURL := strings.TrimSpace(rawSourceURL)
	if sourceURL == "" {
		sourceURL = strings.TrimSpace(rawURL)
	}
	if sourceURL == "" {
		return "", "", NewValidationError("source_url is required", map[string]interface{}{"field": "source_url"})
	}
	return sourceType, sourceURL, nil
}

func (s *StreamService) validateEmbedDomain(ctx context.Context, companyID int64, sourceURL string) (bool, error) {
	host, err := extractHost(sourceURL)
	if err != nil {
		return false, NewValidationError("invalid source_url", map[string]interface{}{"field": "source_url"})
	}
	allowed, storeErr := s.store.IsEmbedDomainAllowed(ctx, companyID, host)
	if storeErr != nil {
		return false, NewInternalError()
	}
	return allowed, nil
}

func extractHost(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return "", errors.New("host is empty")
	}
	return host, nil
}

func parsePositiveID(rawID string) (int64, error) {
	value, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid id")
	}
	return value, nil
}
