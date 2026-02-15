package api

import (
	"context"
	"testing"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

type registrationStoreStub struct {
	approveFn func(ctx context.Context, requestID int64, companyID int64, role string, actorUserID int64) (domain.AuthUser, error)
	rejectFn  func(ctx context.Context, requestID int64, actorUserID int64, reason *string) error
}

func (s *registrationStoreStub) CreateRegistrationRequest(ctx context.Context, companyID int64, email string, login string, passwordHash string, requestedRole string) (domain.RegistrationRequest, error) {
	return domain.RegistrationRequest{}, nil
}

func (s *registrationStoreStub) ListPendingRegistrationRequests(ctx context.Context) ([]domain.RegistrationRequest, error) {
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
