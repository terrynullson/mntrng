package api

import (
	"context"
	"errors"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

var (
	ErrProjectAlreadyExists  = errors.New("project_already_exists")
	ErrProjectNotFound       = errors.New("project_not_found")
	ErrProjectCompanyMissing = errors.New("project_company_missing")
)

type ProjectStore interface {
	CreateProject(ctx context.Context, companyID int64, name string) (domain.Project, error)
	ListProjects(ctx context.Context, companyID int64) ([]domain.Project, error)
	GetProject(ctx context.Context, companyID int64, projectID int64) (domain.Project, error)
	UpdateProject(ctx context.Context, companyID int64, projectID int64, name string) (domain.Project, error)
	DeleteProject(ctx context.Context, companyID int64, projectID int64) error
}

type ProjectService struct {
	store ProjectStore
}

func NewProjectService(store ProjectStore) *ProjectService {
	return &ProjectService{store: store}
}

func (s *ProjectService) CreateProject(ctx context.Context, companyID int64, nameRaw string) (domain.Project, error) {
	name := strings.TrimSpace(nameRaw)
	if name == "" {
		return domain.Project{}, NewValidationError("name is required", map[string]interface{}{"field": "name"})
	}

	item, err := s.store.CreateProject(ctx, companyID, name)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, ErrProjectAlreadyExists) {
		return domain.Project{}, NewConflictError(
			"project with the same name already exists for this company",
			map[string]interface{}{"field": "name", "company_id": companyID},
		)
	}
	if errors.Is(err, ErrProjectCompanyMissing) {
		return domain.Project{}, NewNotFoundError("company not found", map[string]interface{}{"company_id": companyID})
	}
	return domain.Project{}, NewInternalError()
}

func (s *ProjectService) ListProjects(ctx context.Context, companyID int64) ([]domain.Project, error) {
	items, err := s.store.ListProjects(ctx, companyID)
	if err != nil {
		return nil, NewInternalError()
	}
	return items, nil
}

func (s *ProjectService) GetProject(ctx context.Context, companyID int64, projectID int64) (domain.Project, error) {
	item, err := s.store.GetProject(ctx, companyID, projectID)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, ErrProjectNotFound) {
		return domain.Project{}, NewNotFoundError(
			"project not found",
			map[string]interface{}{"company_id": companyID, "project_id": projectID},
		)
	}
	return domain.Project{}, NewInternalError()
}

func (s *ProjectService) PatchProject(ctx context.Context, companyID int64, projectID int64, nameRaw string) (domain.Project, error) {
	name := strings.TrimSpace(nameRaw)
	if name == "" {
		return domain.Project{}, NewValidationError("name is required", map[string]interface{}{"field": "name"})
	}

	item, err := s.store.UpdateProject(ctx, companyID, projectID, name)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, ErrProjectNotFound) {
		return domain.Project{}, NewNotFoundError(
			"project not found",
			map[string]interface{}{"company_id": companyID, "project_id": projectID},
		)
	}
	if errors.Is(err, ErrProjectAlreadyExists) {
		return domain.Project{}, NewConflictError(
			"project with the same name already exists for this company",
			map[string]interface{}{"field": "name", "company_id": companyID},
		)
	}
	return domain.Project{}, NewInternalError()
}

func (s *ProjectService) DeleteProject(ctx context.Context, companyID int64, projectID int64) error {
	err := s.store.DeleteProject(ctx, companyID, projectID)
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrProjectNotFound) {
		return NewNotFoundError(
			"project not found",
			map[string]interface{}{"company_id": companyID, "project_id": projectID},
		)
	}
	return NewInternalError()
}
