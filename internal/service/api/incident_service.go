package api

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

// IncidentStore for incidents (API, tenant-scoped).
type IncidentStore interface {
	List(ctx context.Context, companyID int64, filter domain.IncidentListFilter) ([]domain.Incident, int64, error)
	GetByID(ctx context.Context, companyID int64, incidentID int64) (domain.Incident, error)
}

// IncidentService handles incident list/get.
type IncidentService struct {
	store IncidentStore
}

// NewIncidentService returns a new IncidentService.
func NewIncidentService(store IncidentStore) *IncidentService {
	return &IncidentService{store: store}
}

// ListInput for incident list.
type ListIncidentsInput struct {
	CompanyID int64
	Status    string
	Severity  string
	StreamID  string
	Q         string
	Page      string
	PageSize  string
}

// List returns paginated incidents.
func (s *IncidentService) List(ctx context.Context, input ListIncidentsInput) ([]domain.Incident, int64, *string, error) {
	filter := domain.IncidentListFilter{
		Page:     0,
		PageSize: 20,
	}
	if input.Status != "" {
		filter.Status = &input.Status
	}
	if input.Severity != "" {
		filter.Severity = &input.Severity
	}
	if input.StreamID != "" {
		id, err := strconv.ParseInt(strings.TrimSpace(input.StreamID), 10, 64)
		if err == nil && id > 0 {
			filter.StreamID = &id
		}
	}
	filter.Q = strings.TrimSpace(input.Q)
	if input.Page != "" {
		if p, err := strconv.Atoi(input.Page); err == nil && p >= 0 {
			filter.Page = p
		}
	}
	if input.PageSize != "" {
		if ps, err := strconv.Atoi(input.PageSize); err == nil && ps > 0 {
			filter.PageSize = ps
		}
	}

	items, total, err := s.store.List(ctx, input.CompanyID, filter)
	if err != nil {
		return nil, 0, nil, NewInternalError()
	}
	var nextCursor *string
	if int64(len(items)) < total && filter.PageSize > 0 {
		next := strconv.Itoa(filter.Page + 1)
		nextCursor = &next
	}
	return items, total, nextCursor, nil
}

// Get returns one incident by id in tenant scope.
func (s *IncidentService) Get(ctx context.Context, companyID int64, incidentID int64) (domain.Incident, error) {
	item, err := s.store.GetByID(ctx, companyID, incidentID)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrIncidentNotFound) {
		return domain.Incident{}, NewNotFoundError(
			"incident not found",
			map[string]interface{}{"incident_id": incidentID},
		)
	}
	return domain.Incident{}, NewInternalError()
}
