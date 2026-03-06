package api

import (
	"context"
	"testing"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
)

type checkJobStoreStub struct {
	streamExists bool
	listErr      error
}

func (s *checkJobStoreStub) EnqueueCheckJob(ctx context.Context, companyID int64, streamID int64, plannedAt time.Time) (domain.CheckJob, error) {
	return domain.CheckJob{}, nil
}

func (s *checkJobStoreStub) GetCheckJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckJob, error) {
	return domain.CheckJob{}, nil
}

func (s *checkJobStoreStub) StreamExistsForCheckJobs(ctx context.Context, companyID int64, streamID int64) (bool, error) {
	return s.streamExists, nil
}

func (s *checkJobStoreStub) ListCheckJobs(ctx context.Context, companyID int64, streamID int64, filter CheckJobListFilter) ([]domain.CheckJob, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []domain.CheckJob{}, nil
}

func TestParseCheckJobListFilter_ValidAndInvalid(t *testing.T) {
	t.Run("normalizes status and parses range", func(t *testing.T) {
		from := "2026-03-01T10:00:00Z"
		to := "2026-03-01T11:00:00Z"

		filter, err := parseCheckJobListFilter("DONE", from, to)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.Status == nil || *filter.Status != "done" {
			t.Fatalf("expected status=done, got %#v", filter.Status)
		}
		if filter.From == nil || filter.To == nil {
			t.Fatalf("expected from/to to be set, got from=%#v to=%#v", filter.From, filter.To)
		}
		if !filter.From.Before(*filter.To) {
			t.Fatalf("expected from < to, got from=%s to=%s", filter.From, filter.To)
		}
	})

	t.Run("invalid status mapped to validation error", func(t *testing.T) {
		_, err := parseCheckJobListFilter("UNKNOWN", "", "")
		if err == nil {
			t.Fatalf("expected error")
		}
		serviceErr, ok := AsServiceError(err)
		if !ok {
			t.Fatalf("expected ServiceError, got %T", err)
		}
		if serviceErr.Code != "validation_error" {
			t.Fatalf("expected validation_error, got %s", serviceErr.Code)
		}
	})

	t.Run("invalid from/to timestamps mapped to validation error", func(t *testing.T) {
		_, err := parseCheckJobListFilter("queued", "not-a-time", "")
		if err == nil {
			t.Fatalf("expected error for invalid from")
		}
		if se, ok := AsServiceError(err); !ok || se.Code != "validation_error" {
			t.Fatalf("expected validation_error for from, got %#v", se)
		}

		_, err = parseCheckJobListFilter("queued", "", "also-not-a-time")
		if err == nil {
			t.Fatalf("expected error for invalid to")
		}
		if se, ok := AsServiceError(err); !ok || se.Code != "validation_error" {
			t.Fatalf("expected validation_error for to, got %#v", se)
		}
	})
}

func TestListCheckJobs_StreamNotFoundAndHappyPath(t *testing.T) {
	t.Run("stream missing mapped to not_found", func(t *testing.T) {
		store := &checkJobStoreStub{streamExists: false}
		service := NewCheckJobService(store)

		_, err := service.ListCheckJobs(context.Background(), ListCheckJobsInput{
			CompanyID: 10,
			StreamID:  99,
			StatusRaw: "",
		})
		if err == nil {
			t.Fatalf("expected error")
		}
		serviceErr, ok := AsServiceError(err)
		if !ok {
			t.Fatalf("expected ServiceError, got %T", err)
		}
		if serviceErr.Code != "not_found" {
			t.Fatalf("expected not_found, got %s", serviceErr.Code)
		}
	})

	t.Run("happy path passes through to store", func(t *testing.T) {
		store := &checkJobStoreStub{streamExists: true}
		service := NewCheckJobService(store)

		items, err := service.ListCheckJobs(context.Background(), ListCheckJobsInput{
			CompanyID: 10,
			StreamID:  1,
			StatusRaw: "running",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if items == nil {
			t.Fatalf("expected items slice, got nil")
		}
	})
}

