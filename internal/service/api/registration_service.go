package api

import (
	"context"
	"errors"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type RegistrationStore interface {
	CreateRegistrationRequest(ctx context.Context, companyID int64, email string, login string, passwordHash string, requestedRole string) (domain.RegistrationRequest, error)
	ListPendingRegistrationRequests(ctx context.Context) ([]domain.RegistrationRequest, error)
	ApproveRegistrationRequest(ctx context.Context, requestID int64, companyID int64, role string, actorUserID int64) (domain.AuthUser, error)
	RejectRegistrationRequest(ctx context.Context, requestID int64, actorUserID int64, reason *string) error
	ChangeUserRole(ctx context.Context, userID int64, role string, companyID *int64, actorUserID int64) (domain.AuthUser, error)
}

type RegistrationNotifier interface {
	NotifyNewRegistrationRequest(ctx context.Context, request domain.RegistrationRequest) error
}

type RegistrationService struct {
	store    RegistrationStore
	notifier RegistrationNotifier
}

func NewRegistrationService(store RegistrationStore, notifier RegistrationNotifier) *RegistrationService {
	return &RegistrationService{store: store, notifier: notifier}
}

func (s *RegistrationService) SubmitRegistrationRequest(ctx context.Context, request domain.RegistrationRequestCreate) (domain.RegistrationRequest, error) {
	email := strings.TrimSpace(request.Email)
	if email == "" {
		return domain.RegistrationRequest{}, NewValidationError("email is required", map[string]interface{}{"field": "email"})
	}
	login := strings.TrimSpace(request.Login)
	if login == "" {
		return domain.RegistrationRequest{}, NewValidationError("login is required", map[string]interface{}{"field": "login"})
	}
	if request.CompanyID <= 0 {
		return domain.RegistrationRequest{}, NewValidationError("company_id must be positive", map[string]interface{}{"field": "company_id"})
	}
	if request.Password == "" {
		return domain.RegistrationRequest{}, NewValidationError("password is required", map[string]interface{}{"field": "password"})
	}
	if len(request.Password) < 8 {
		return domain.RegistrationRequest{}, NewValidationError("password must be at least 8 characters", map[string]interface{}{"field": "password"})
	}
	if request.RequestedRole != domain.RoleCompanyAdmin && request.RequestedRole != domain.RoleViewer {
		return domain.RegistrationRequest{}, NewValidationError(
			"requested_role must be company_admin or viewer",
			map[string]interface{}{"field": "requested_role"},
		)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.RegistrationRequest{}, NewInternalError()
	}

	item, err := s.store.CreateRegistrationRequest(
		ctx,
		request.CompanyID,
		email,
		login,
		string(passwordHash),
		request.RequestedRole,
	)
	if err != nil {
		if errors.Is(err, domain.ErrRegistrationConflict) || errors.Is(err, domain.ErrUserAlreadyExists) {
			return domain.RegistrationRequest{}, NewConflictError("registration request already exists", map[string]interface{}{})
		}
		if errors.Is(err, domain.ErrCompanyNotFound) {
			return domain.RegistrationRequest{}, NewNotFoundError("company not found", map[string]interface{}{"company_id": request.CompanyID})
		}
		return domain.RegistrationRequest{}, NewInternalError()
	}

	if s.notifier != nil {
		_ = s.notifier.NotifyNewRegistrationRequest(ctx, item)
	}

	return item, nil
}

func (s *RegistrationService) ListPendingRegistrationRequests(ctx context.Context) ([]domain.RegistrationRequest, error) {
	items, err := s.store.ListPendingRegistrationRequests(ctx)
	if err != nil {
		return nil, NewInternalError()
	}
	return items, nil
}

func (s *RegistrationService) ApproveRegistrationRequest(ctx context.Context, requestID int64, request domain.ApproveRegistrationRequest, actorUserID int64) (domain.AuthUser, error) {
	if requestID <= 0 {
		return domain.AuthUser{}, NewValidationError("request_id must be positive", map[string]interface{}{"field": "request_id"})
	}
	if request.CompanyID <= 0 {
		return domain.AuthUser{}, NewValidationError("company_id must be positive", map[string]interface{}{"field": "company_id"})
	}
	if request.Role != domain.RoleCompanyAdmin && request.Role != domain.RoleViewer {
		return domain.AuthUser{}, NewValidationError("role must be company_admin or viewer", map[string]interface{}{"field": "role"})
	}

	item, err := s.store.ApproveRegistrationRequest(ctx, requestID, request.CompanyID, request.Role, actorUserID)
	if err != nil {
		if errors.Is(err, domain.ErrRegistrationNotFound) {
			return domain.AuthUser{}, NewNotFoundError("registration request not found", map[string]interface{}{"request_id": requestID})
		}
		if errors.Is(err, domain.ErrRegistrationNotPending) {
			return domain.AuthUser{}, NewConflictError("registration request is not pending", map[string]interface{}{"request_id": requestID})
		}
		if errors.Is(err, domain.ErrUserAlreadyExists) || errors.Is(err, domain.ErrRegistrationConflict) {
			return domain.AuthUser{}, NewConflictError("user with email or login already exists", map[string]interface{}{})
		}
		if errors.Is(err, domain.ErrCompanyNotFound) {
			return domain.AuthUser{}, NewNotFoundError("company not found", map[string]interface{}{"company_id": request.CompanyID})
		}
		return domain.AuthUser{}, NewInternalError()
	}

	return item, nil
}

func (s *RegistrationService) RejectRegistrationRequest(ctx context.Context, requestID int64, request domain.RejectRegistrationRequest, actorUserID int64) error {
	if requestID <= 0 {
		return NewValidationError("request_id must be positive", map[string]interface{}{"field": "request_id"})
	}

	var reason *string
	trimmedReason := strings.TrimSpace(request.Reason)
	if trimmedReason != "" {
		reason = &trimmedReason
	}

	err := s.store.RejectRegistrationRequest(ctx, requestID, actorUserID, reason)
	if err != nil {
		if errors.Is(err, domain.ErrRegistrationNotFound) {
			return NewNotFoundError("registration request not found", map[string]interface{}{"request_id": requestID})
		}
		if errors.Is(err, domain.ErrRegistrationNotPending) {
			return NewConflictError("registration request is not pending", map[string]interface{}{"request_id": requestID})
		}
		return NewInternalError()
	}
	return nil
}

func (s *RegistrationService) ChangeUserRole(ctx context.Context, userID int64, request domain.ChangeUserRoleRequest, actorUserID int64) (domain.AuthUser, error) {
	if userID <= 0 {
		return domain.AuthUser{}, NewValidationError("user_id must be positive", map[string]interface{}{"field": "user_id"})
	}
	if request.Role != domain.RoleCompanyAdmin && request.Role != domain.RoleViewer {
		return domain.AuthUser{}, NewValidationError(
			"role must be company_admin or viewer",
			map[string]interface{}{"field": "role"},
		)
	}
	if request.CompanyID == nil || *request.CompanyID <= 0 {
		return domain.AuthUser{}, NewValidationError(
			"company_id is required for company_admin/viewer",
			map[string]interface{}{"field": "company_id"},
		)
	}

	item, err := s.store.ChangeUserRole(ctx, userID, request.Role, request.CompanyID, actorUserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.AuthUser{}, NewNotFoundError("user not found", map[string]interface{}{"user_id": userID})
		}
		if errors.Is(err, domain.ErrCompanyNotFound) {
			return domain.AuthUser{}, NewNotFoundError("company not found", map[string]interface{}{})
		}
		return domain.AuthUser{}, NewInternalError()
	}

	return item, nil
}
