package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

var testTime = time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

func assertErrorCode(t *testing.T, body []byte, code string) {
	t.Helper()
	var e domain.ErrorEnvelope
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if e.Code != code {
		t.Fatalf("expected code %q, got %q", code, e.Code)
	}
}

type mockStreamStore struct {
	listResp   []domain.Stream
	listErr    error
	latestResp []domain.StreamLatestStatus
	latestErr  error
	getResp    domain.Stream
	getErr     error
	createResp domain.Stream
	createErr  error
	patchResp  domain.Stream
	patchErr   error
	deleteErr  error
}

func (m *mockStreamStore) ListStreams(ctx context.Context, companyID int64, filter domain.StreamListFilter) ([]domain.Stream, error) {
	return m.listResp, m.listErr
}
func (m *mockStreamStore) ListLatestStatuses(ctx context.Context, companyID int64) ([]domain.StreamLatestStatus, error) {
	return m.latestResp, m.latestErr
}
func (m *mockStreamStore) GetStream(ctx context.Context, companyID int64, streamID int64) (domain.Stream, error) {
	return m.getResp, m.getErr
}
func (m *mockStreamStore) CreateStream(ctx context.Context, companyID int64, projectID int64, name string, sourceType string, sourceURL string, isActive bool) (domain.Stream, error) {
	return m.createResp, m.createErr
}
func (m *mockStreamStore) PatchStream(ctx context.Context, companyID int64, streamID int64, patch domain.StreamPatchInput) (domain.Stream, error) {
	return m.patchResp, m.patchErr
}
func (m *mockStreamStore) DeleteStream(ctx context.Context, companyID int64, streamID int64) error {
	return m.deleteErr
}

func (m *mockStreamStore) IsEmbedDomainAllowed(ctx context.Context, companyID int64, host string) (bool, error) {
	return true, nil
}

type mockCheckJobStore struct {
	enqueueResp     domain.CheckJob
	enqueueErr      error
	getResp         domain.CheckJob
	getErr          error
	streamExists    bool
	streamExistsErr error
	listResp        []domain.CheckJob
	listErr         error
}

func (m *mockCheckJobStore) EnqueueCheckJob(ctx context.Context, companyID int64, streamID int64, plannedAt time.Time) (domain.CheckJob, error) {
	return m.enqueueResp, m.enqueueErr
}
func (m *mockCheckJobStore) GetCheckJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckJob, error) {
	return m.getResp, m.getErr
}
func (m *mockCheckJobStore) StreamExistsForCheckJobs(ctx context.Context, companyID int64, streamID int64) (bool, error) {
	return m.streamExists, m.streamExistsErr
}
func (m *mockCheckJobStore) ListCheckJobs(ctx context.Context, companyID int64, streamID int64, filter domain.CheckJobListFilter) ([]domain.CheckJob, error) {
	return m.listResp, m.listErr
}

type mockCheckResultStore struct {
	getByIDResp     domain.CheckResult
	getByIDErr      error
	getByJobResp    domain.CheckResult
	getByJobErr     error
	streamExists    bool
	streamExistsErr error
	listResp        []domain.CheckResult
	listErr         error
}

func (m *mockCheckResultStore) GetCheckResultByID(ctx context.Context, companyID int64, resultID int64) (domain.CheckResult, error) {
	return m.getByIDResp, m.getByIDErr
}
func (m *mockCheckResultStore) GetCheckResultByJobID(ctx context.Context, companyID int64, jobID int64) (domain.CheckResult, error) {
	return m.getByJobResp, m.getByJobErr
}
func (m *mockCheckResultStore) StreamExistsForCheckResults(ctx context.Context, companyID int64, streamID int64) (bool, error) {
	return m.streamExists, m.streamExistsErr
}
func (m *mockCheckResultStore) ListCheckResults(ctx context.Context, companyID int64, streamID int64, filter domain.CheckResultListFilter) ([]domain.CheckResult, error) {
	return m.listResp, m.listErr
}
