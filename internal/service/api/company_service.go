package api

import (
	"context"
	"errors"
	"strings"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

type CompanyStore interface {
	CreateCompany(ctx context.Context, name string) (domain.Company, error)
	ListCompanies(ctx context.Context) ([]domain.Company, error)
	GetCompany(ctx context.Context, companyID int64) (domain.Company, error)
	UpdateCompany(ctx context.Context, companyID int64, name string) (domain.Company, error)
	DeleteCompany(ctx context.Context, companyID int64) error
}

type CompanyService struct {
	store CompanyStore
}

func NewCompanyService(store CompanyStore) *CompanyService {
	return &CompanyService{store: store}
}

func (s *CompanyService) CreateCompany(ctx context.Context, nameRaw string) (domain.Company, error) {
	name := strings.TrimSpace(nameRaw)
	if name == "" {
		return domain.Company{}, NewValidationError("name is required", map[string]interface{}{"field": "name"})
	}

	item, err := s.store.CreateCompany(ctx, name)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrCompanyAlreadyExists) {
		return domain.Company{}, NewConflictError("company with the same name already exists", map[string]interface{}{"field": "name"})
	}
	return domain.Company{}, NewInternalError()
}

func (s *CompanyService) ListCompanies(ctx context.Context) ([]domain.Company, error) {
	items, err := s.store.ListCompanies(ctx)
	if err != nil {
		return nil, NewInternalError()
	}
	return items, nil
}

func (s *CompanyService) GetCompany(ctx context.Context, companyID int64) (domain.Company, error) {
	item, err := s.store.GetCompany(ctx, companyID)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrCompanyNotFound) {
		return domain.Company{}, NewNotFoundError("company not found", map[string]interface{}{"company_id": companyID})
	}
	return domain.Company{}, NewInternalError()
}

func (s *CompanyService) PatchCompany(ctx context.Context, companyID int64, nameRaw string) (domain.Company, error) {
	name := strings.TrimSpace(nameRaw)
	if name == "" {
		return domain.Company{}, NewValidationError("name is required", map[string]interface{}{"field": "name"})
	}

	item, err := s.store.UpdateCompany(ctx, companyID, name)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, domain.ErrCompanyNotFound) {
		return domain.Company{}, NewNotFoundError("company not found", map[string]interface{}{"company_id": companyID})
	}
	if errors.Is(err, domain.ErrCompanyAlreadyExists) {
		return domain.Company{}, NewConflictError("company with the same name already exists", map[string]interface{}{"field": "name"})
	}
	return domain.Company{}, NewInternalError()
}

func (s *CompanyService) DeleteCompany(ctx context.Context, companyID int64) error {
	err := s.store.DeleteCompany(ctx, companyID)
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrCompanyNotFound) {
		return NewNotFoundError("company not found", map[string]interface{}{"company_id": companyID})
	}
	return NewInternalError()
}
