package api

import (
	"context"
	"testing"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
)

type registrationStoreStub struct {
	createFn       func(ctx context.Context, companyID int64, email string, login string, passwordHash string, requestedRole string) (domain.RegistrationRequest, error)
	listPendingFn  func(ctx context.Context) ([]domain.RegistrationRequest, error)
	approveFn      func(ctx context.Context, requestID int64, companyID int64, role string, actorUserID int64) (domain.AuthUser, error)
	rejectFn       func(ctx context.Context, requestID int64, actorUserID int64, reason *string) error
	listUsersFn    func(ctx context.Context, filter domain.AdminUserListFilter) ([]domain.AuthUser, error)
	changeUserRoleFn   func(ctx context.Context, userID int64, role string, companyID *int64, actorUserID int64) (domain.AuthUser, error)
	changeUserStatusFn func(ctx context.Context, userID int64, status string, actorUserID int64) (domain.AuthUser, error)
}

func (s *registrationStoreStub) CreateRegistrationRequest(ctx context.Context, companyID int64, email string, login string, passwordHash string, requestedRole string) (domain.RegistrationRequest, error) {
	if s.createFn != nil {
		return s.createFn(ctx, companyID, email, login, passwordHash, requestedRole)
	}
	return domain.RegistrationRequest{}, nil
}

func (s *registrationStoreStub) ListPendingRegistrationRequests(ctx context.Context) ([]domain.RegistrationRequest, error) {
	if s.listPendingFn != nil {
		return s.listPendingFn(ctx)
	}
	return nil, nil
}

func (s *registrationStoreStub) ApproveRegistrationRequest(ctx context.Context, requestID int64, companyID int64, role string, actorUserID int64) (domain.AuthUser, error) {
	if s.approveFn != nil {
		return s.approveFn(ctx, requestID, companyID, role, actorUserID)
	}
	return domain.AuthUser{}, nil
}

func (s *registrationStoreStub) RejectRegistrationRequest(ctx context.Context, requestID int64, actorUserID int64, reason *string) error {
	if s.rejectFn != nil {
		return s.rejectFn(ctx, requestID, actorUserID, reason)
	}
	return nil
}

func (s *registrationStoreStub) ChangeUserRole(ctx context.Context, userID int64, role string, companyID *int64, actorUserID int64) (domain.AuthUser, error) {
	if s.changeUserRoleFn != nil {
		return s.changeUserRoleFn(ctx, userID, role, companyID, actorUserID)
	}
	return domain.AuthUser{}, nil
}

func (s *registrationStoreStub) ListUsers(ctx context.Context, filter domain.AdminUserListFilter) ([]domain.AuthUser, error) {
	if s.listUsersFn != nil {
		return s.listUsersFn(ctx, filter)
	}
	return []domain.AuthUser{}, nil
}

func (s *registrationStoreStub) ChangeUserStatus(ctx context.Context, userID int64, status string, actorUserID int64) (domain.AuthUser, error) {
	if s.changeUserStatusFn != nil {
		return s.changeUserStatusFn(ctx, userID, status, actorUserID)
	}
	return domain.AuthUser{}, nil
}

func TestApproveRejectWorkflow(t *testing.T) {
	var approved bool
	var rejected bool

	store := &registrationStoreStub{
		approveFn: func(ctx context.Context, requestID int64, companyID int64, role string, actorUserID int64) (domain.AuthUser, error) {
			approved = requestID == 10 && companyID == 3 && role == domain.RoleCompanyAdmin && actorUserID == 900
			return domain.AuthUser{ID: 55, CompanyID: &companyID, Role: role, Status: domain.UserStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
		rejectFn: func(ctx context.Context, requestID int64, actorUserID int64, reason *string) error {
			rejected = requestID == 11 && actorUserID == 900 && reason != nil && *reason == "incomplete data"
			return nil
		},
	}

	service := NewRegistrationService(store, nil)

	_, approveErr := service.ApproveRegistrationRequest(context.Background(), 10, domain.ApproveRegistrationRequest{CompanyID: 3, Role: domain.RoleCompanyAdmin}, 900)
	if approveErr != nil {
		t.Fatalf("approve failed: %v", approveErr)
	}
	if !approved {
		t.Fatal("approve path did not call store with expected arguments")
	}

	rejectErr := service.RejectRegistrationRequest(context.Background(), 11, domain.RejectRegistrationRequest{Reason: "incomplete data"}, 900)
	if rejectErr != nil {
		t.Fatalf("reject failed: %v", rejectErr)
	}
	if !rejected {
		t.Fatal("reject path did not call store with expected arguments")
	}
}

func TestListUsersAndStatusChangeValidation(t *testing.T) {
	var observedFilter domain.AdminUserListFilter
	var observedStatus string
	var observedUserID int64
	var observedActorID int64

	store := &registrationStoreStub{
		listUsersFn: func(ctx context.Context, filter domain.AdminUserListFilter) ([]domain.AuthUser, error) {
			observedFilter = filter
			return []domain.AuthUser{}, nil
		},
		changeUserStatusFn: func(ctx context.Context, userID int64, status string, actorUserID int64) (domain.AuthUser, error) {
			observedUserID = userID
			observedStatus = status
			observedActorID = actorUserID
			return domain.AuthUser{ID: userID, Status: status, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}
	service := NewRegistrationService(store, nil)

	_, err := service.ListUsers(context.Background(), ListAdminUsersInput{
		CompanyIDRaw: "42",
		RoleRaw:      domain.RoleViewer,
		StatusRaw:    domain.UserStatusActive,
		LimitRaw:     "500",
	})
	if err != nil {
		t.Fatalf("list users failed: %v", err)
	}
	if observedFilter.CompanyID == nil || *observedFilter.CompanyID != 42 {
		t.Fatalf("unexpected company filter: %#v", observedFilter.CompanyID)
	}
	if observedFilter.Role == nil || *observedFilter.Role != domain.RoleViewer {
		t.Fatalf("unexpected role filter: %#v", observedFilter.Role)
	}
	if observedFilter.Status == nil || *observedFilter.Status != domain.UserStatusActive {
		t.Fatalf("unexpected status filter: %#v", observedFilter.Status)
	}
	if observedFilter.Limit != maxAdminUsersLimit {
		t.Fatalf("expected capped limit=%d got=%d", maxAdminUsersLimit, observedFilter.Limit)
	}

	_, err = service.ChangeUserStatus(context.Background(), 99, domain.ChangeUserStatusRequest{Status: "DISABLED"}, 777)
	if err != nil {
		t.Fatalf("change user status failed: %v", err)
	}
	if observedUserID != 99 || observedActorID != 777 || observedStatus != domain.UserStatusDisabled {
		t.Fatalf("unexpected status change arguments user=%d actor=%d status=%s", observedUserID, observedActorID, observedStatus)
	}

	_, err = service.ChangeUserStatus(context.Background(), 99, domain.ChangeUserStatusRequest{Status: "blocked"}, 777)
	if err == nil {
		t.Fatal("expected validation error for invalid status")
	}
}

func TestSubmitRegistrationRequestValidationAndConflict(t *testing.T) {
	t.Run("invalid payload returns validation error", func(t *testing.T) {
		service := NewRegistrationService(&registrationStoreStub{}, nil)

		_, err := service.SubmitRegistrationRequest(context.Background(), domain.RegistrationRequestCreate{
			CompanyID:     0,
			Email:         " ",
			Login:         "",
			Password:      "short",
			RequestedRole: "invalid_role",
		})
		if err == nil {
			t.Fatalf("expected validation error")
		}
		serviceErr, ok := AsServiceError(err)
		if !ok {
			t.Fatalf("expected ServiceError, got %T", err)
		}
		if serviceErr.Code != "validation_error" {
			t.Fatalf("expected validation_error, got %s", serviceErr.Code)
		}
	})

	t.Run("store conflict mapped to conflict error", func(t *testing.T) {
		store := &registrationStoreStub{
			createFn: func(ctx context.Context, companyID int64, email string, login string, passwordHash string, requestedRole string) (domain.RegistrationRequest, error) {
				return domain.RegistrationRequest{}, domain.ErrRegistrationConflict
			},
		}
		service := NewRegistrationService(store, nil)

		_, err := service.SubmitRegistrationRequest(context.Background(), domain.RegistrationRequestCreate{
			CompanyID:     10,
			Email:         "user@example.com",
			Login:         "user1",
			Password:      "StrongPass123",
			RequestedRole: domain.RoleViewer,
		})
		if err == nil {
			t.Fatalf("expected error")
		}
		serviceErr, ok := AsServiceError(err)
		if !ok {
			t.Fatalf("expected ServiceError, got %T", err)
		}
		if serviceErr.Code != "conflict" {
			t.Fatalf("expected conflict code, got %s", serviceErr.Code)
		}
	})

	t.Run("happy path hashes password and passes through", func(t *testing.T) {
		var capturedPasswordHash string
		store := &registrationStoreStub{
			createFn: func(ctx context.Context, companyID int64, email string, login string, passwordHash string, requestedRole string) (domain.RegistrationRequest, error) {
				capturedPasswordHash = passwordHash
				return domain.RegistrationRequest{
					ID:        123,
					CompanyID: companyID,
					Email:     email,
					Login:     login,
					Status:    domain.RegistrationStatusPending,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		}
		service := NewRegistrationService(store, nil)

		out, err := service.SubmitRegistrationRequest(context.Background(), domain.RegistrationRequestCreate{
			CompanyID:     7,
			Email:         "user@example.com",
			Login:         "user7",
			Password:      "StrongPass123",
			RequestedRole: domain.RoleCompanyAdmin,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.ID != 123 || out.CompanyID != 7 {
			t.Fatalf("unexpected registration: %+v", out)
		}
		if capturedPasswordHash == "" || capturedPasswordHash == "StrongPass123" {
			t.Fatalf("expected hashed password, got %q", capturedPasswordHash)
		}
	})
}
