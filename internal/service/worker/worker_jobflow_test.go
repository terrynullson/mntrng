package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
)

// --- Job flow (high level) ---

type mockJobRepoNoJob struct{}

func (m *mockJobRepoNoJob) ClaimNextQueuedJob(ctx context.Context) (domain.WorkerClaimedJob, bool, error) {
	return domain.WorkerClaimedJob{}, false, nil
}

func (m *mockJobRepoNoJob) RequeueStaleRunningJobs(ctx context.Context, staleAfter time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockJobRepoNoJob) FinalizeJob(ctx context.Context, job domain.WorkerClaimedJob, status string, errorMessage string) (int64, error) {
	return 0, nil
}

func TestProcessSingleJobCycle_NoJob_NoError(t *testing.T) {
	t.Parallel()

	w := NewWorker(
		Config{},
		Repositories{
			JobRepo: &mockJobRepoNoJob{},
		},
	)

	if err := w.ProcessSingleJobCycle(context.Background()); err != nil {
		t.Fatalf("expected no error when no job is claimed, got %v", err)
	}
}

// --- persistCheckResult ---

type mockCheckResultRepo struct {
	called     bool
	job        domain.WorkerClaimedJob
	dbStatus   string
	checksJSON string
	err        error
}

func (m *mockCheckResultRepo) PersistCheckResult(ctx context.Context, job domain.WorkerClaimedJob, dbStatus string, checksJSON string) error {
	m.called = true
	m.job = job
	m.dbStatus = dbStatus
	m.checksJSON = checksJSON
	return m.err
}

func TestPersistCheckResult_PersistsTenantAndChecks(t *testing.T) {
	t.Parallel()

	repo := &mockCheckResultRepo{}

	w := NewWorker(
		Config{},
		Repositories{
			CheckResultRepo: repo,
		},
	)

	job := domain.WorkerClaimedJob{
		ID:        123,
		CompanyID: 10,
		StreamID:  20,
		PlannedAt: time.Unix(0, 0).UTC(),
	}
	evaluation := checkJobEvaluation{
		DBStatus:  domain.WorkerStatusDBFail,
		Aggregate: domain.WorkerStatusFail,
		Checks: map[string]interface{}{
			"playlist": "FAIL",
			"freeze":   "OK",
		},
	}

	if err := w.persistCheckResult(context.Background(), job, evaluation); err != nil {
		t.Fatalf("persistCheckResult returned error: %v", err)
	}
	if !repo.called {
		t.Fatalf("expected PersistCheckResult to be called")
	}
	if repo.job.CompanyID != job.CompanyID || repo.job.StreamID != job.StreamID || repo.job.ID != job.ID {
		t.Fatalf("unexpected job passed to repo: %+v", repo.job)
	}
	if repo.dbStatus != evaluation.DBStatus {
		t.Fatalf("expected dbStatus=%s, got %s", evaluation.DBStatus, repo.dbStatus)
	}

	var checks map[string]interface{}
	if err := json.Unmarshal([]byte(repo.checksJSON), &checks); err != nil {
		t.Fatalf("failed to unmarshal checksJSON: %v", err)
	}
	if checks["playlist"] != "FAIL" || checks["freeze"] != "OK" {
		t.Fatalf("unexpected checks payload: %#v", checks)
	}
}

// --- applyAlertState ---

type mockAlertStateRepo struct {
	called           bool
	companyID        int64
	streamID         int64
	currentStatus    string
	failThreshold    int
	alertCooldown    time.Duration
	sendRecovered    bool
	decisionToReturn domain.WorkerAlertDecision
	err              error
}

func (m *mockAlertStateRepo) ApplyAlertState(
	ctx context.Context,
	companyID int64,
	streamID int64,
	currentStatus string,
	failStreakThreshold int,
	alertCooldown time.Duration,
	alertSendRecovered bool,
) (domain.WorkerAlertDecision, error) {
	m.called = true
	m.companyID = companyID
	m.streamID = streamID
	m.currentStatus = currentStatus
	m.failThreshold = failStreakThreshold
	m.alertCooldown = alertCooldown
	m.sendRecovered = alertSendRecovered
	return m.decisionToReturn, m.err
}

func TestApplyAlertState_PassesTenantAndConfig(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertStateRepo{
		decisionToReturn: domain.WorkerAlertDecision{
			ShouldSend:     true,
			EventType:      domain.WorkerAlertEventFail,
			Reason:         "test",
			CurrentStatus:  domain.WorkerStatusDBFail,
			PreviousStatus: domain.WorkerStatusDBWarn,
			FailStreak:     2,
		},
	}

	cfg := Config{
		AlertFailStreak:    2,
		AlertCooldown:      5 * time.Minute,
		AlertSendRecovered: true,
	}

	w := NewWorker(
		cfg,
		Repositories{
			AlertStateRepo: alertRepo,
		},
	)

	job := claimedJob{
		ID:        200,
		CompanyID: 42,
		StreamID:  7,
		PlannedAt: time.Unix(0, 0).UTC(),
	}

	decision, err := w.applyAlertState(context.Background(), job, domain.WorkerStatusFail)
	if err != nil {
		t.Fatalf("applyAlertState returned error: %v", err)
	}
	if !alertRepo.called {
		t.Fatalf("expected ApplyAlertState on repo to be called")
	}
	if alertRepo.companyID != job.CompanyID || alertRepo.streamID != job.StreamID {
		t.Fatalf("unexpected tenant in ApplyAlertState: company_id=%d stream_id=%d", alertRepo.companyID, alertRepo.streamID)
	}
	if alertRepo.currentStatus != domain.WorkerStatusDBFail {
		t.Fatalf("expected currentStatus=%s, got %s", domain.WorkerStatusDBFail, alertRepo.currentStatus)
	}
	if alertRepo.failThreshold != cfg.AlertFailStreak {
		t.Fatalf("expected failThreshold=%d, got %d", cfg.AlertFailStreak, alertRepo.failThreshold)
	}
	if alertRepo.alertCooldown != cfg.AlertCooldown {
		t.Fatalf("expected alertCooldown=%s, got %s", cfg.AlertCooldown, alertRepo.alertCooldown)
	}
	if !alertRepo.sendRecovered {
		t.Fatalf("expected sendRecovered=true")
	}
	if !decision.ShouldSend || decision.EventType != domain.WorkerAlertEventFail {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

// --- finalizeWithRetry: stops after retryMax ---

func retryableNetError() error {
	return &net.OpError{Op: "write", Err: errors.New("connection reset")}
}

type mockJobRepoFinalizeFails struct {
	claimResp     domain.WorkerClaimedJob
	claimOk       bool
	finalizeCalls int
	finalizeErr   error
}

func (m *mockJobRepoFinalizeFails) ClaimNextQueuedJob(ctx context.Context) (domain.WorkerClaimedJob, bool, error) {
	return m.claimResp, m.claimOk, nil
}

func (m *mockJobRepoFinalizeFails) RequeueStaleRunningJobs(ctx context.Context, staleAfter time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockJobRepoFinalizeFails) FinalizeJob(ctx context.Context, job domain.WorkerClaimedJob, status string, errorMessage string) (int64, error) {
	m.finalizeCalls++
	return 0, m.finalizeErr
}

type mockStreamRepoLoadFails struct{ err error }

func (m *mockStreamRepoLoadFails) LoadStreamURL(ctx context.Context, companyID int64, streamID int64) (string, error) {
	return "", m.err
}

func TestFinalizeWithRetry_StopsAfterRetryMax(t *testing.T) {
	t.Parallel()

	job := domain.WorkerClaimedJob{ID: 1, CompanyID: 10, StreamID: 20, PlannedAt: time.Now().UTC()}
	mockJob := &mockJobRepoFinalizeFails{
		claimResp:   job,
		claimOk:    true,
		finalizeErr: retryableNetError(),
	}

	// processJob fails (stream load fails), so we hit finalizeWithRetry(failed, ...); FinalizeJob keeps returning retryable error
	streamRepo := &mockStreamRepoLoadFails{err: errors.New("stream url load failed")}

	cfg := Config{
		JobTimeout:    5 * time.Second,
		RetryMax:      2,
		RetryBackoff:  1 * time.Millisecond,
	}

	w := NewWorker(cfg, Repositories{
		JobRepo:    mockJob,
		StreamRepo: streamRepo,
	})

	ctx := context.Background()
	err := w.ProcessSingleJobCycle(ctx)
	if err == nil {
		t.Fatal("expected error when FinalizeJob always fails")
	}
	// retryMax=2 => attempts 0, 1, 2 => 3 calls then give up
	if mockJob.finalizeCalls != 3 {
		t.Fatalf("expected 3 FinalizeJob calls (retryMax=2), got %d", mockJob.finalizeCalls)
	}
}
