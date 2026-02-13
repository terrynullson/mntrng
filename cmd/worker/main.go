package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/lib/pq"
)

type claimedJob struct {
	ID        int64
	CompanyID int64
	StreamID  int64
	PlannedAt time.Time
}

type worker struct {
	db           *sql.DB
	pollInterval time.Duration
	jobTimeout   time.Duration
	stubDuration time.Duration
	retryMax     int
	retryBackoff time.Duration
}

func main() {
	pollInterval := time.Duration(intAtLeast(config.GetInt("WORKER_HEARTBEAT_SEC", 15), 1)) * time.Second
	jobTimeout := time.Duration(intAtLeast(config.GetInt("WORKER_JOB_TIMEOUT_SEC", 30), 1)) * time.Second
	stubDuration := time.Duration(intAtLeast(config.GetInt("WORKER_STUB_DURATION_MS", 200), 1)) * time.Millisecond
	retryMax := intAtLeast(config.GetInt("WORKER_DB_RETRY_MAX", 2), 0)
	retryBackoff := time.Duration(intAtLeast(config.GetInt("WORKER_DB_RETRY_BACKOFF_MS", 500), 1)) * time.Millisecond
	databaseURL := config.GetString("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w := &worker{
		db:           db,
		pollInterval: pollInterval,
		jobTimeout:   jobTimeout,
		stubDuration: stubDuration,
		retryMax:     retryMax,
		retryBackoff: retryBackoff,
	}

	log.Printf(
		"worker skeleton started: poll_interval=%s, job_timeout=%s, stub_duration=%s, retry_max=%d, retry_backoff=%s",
		w.pollInterval,
		w.jobTimeout,
		w.stubDuration,
		w.retryMax,
		w.retryBackoff,
	)

	if err := w.processCycleWithRetry(ctx); err != nil {
		log.Printf("worker cycle failed: %v", err)
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("worker skeleton stopped")
			return
		case currentTime := <-ticker.C:
			log.Printf("worker skeleton heartbeat: %s", currentTime.UTC().Format(time.RFC3339))
			if err := w.processCycleWithRetry(ctx); err != nil {
				log.Printf("worker cycle failed: %v", err)
			}
		}
	}
}

func (w *worker) processCycleWithRetry(ctx context.Context) error {
	for attempt := 0; ; attempt++ {
		err := w.processSingleJobCycle(ctx)
		if err == nil {
			return nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker retry attempt=%d backoff=%s err=%v", attempt+1, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func (w *worker) processSingleJobCycle(ctx context.Context) error {
	job, ok, err := w.claimNextQueuedJob(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	log.Printf(
		"worker claimed job: id=%d company_id=%d stream_id=%d planned_at=%s",
		job.ID,
		job.CompanyID,
		job.StreamID,
		job.PlannedAt.UTC().Format(time.RFC3339),
	)

	processErr := w.runStub(ctx, job)
	if processErr != nil {
		if finalizeErr := w.finalizeWithRetry(ctx, job, "failed", processErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, processErr.Error())
		return nil
	}

	if finalizeErr := w.finalizeWithRetry(ctx, job, "done", ""); finalizeErr != nil {
		return finalizeErr
	}
	log.Printf("worker finalized job as done: id=%d company_id=%d", job.ID, job.CompanyID)
	return nil
}

func (w *worker) claimNextQueuedJob(ctx context.Context) (claimedJob, bool, error) {
	row := w.db.QueryRowContext(
		ctx,
		`WITH candidate AS (
             SELECT id, company_id
             FROM check_jobs
             WHERE status = 'queued'
             ORDER BY planned_at ASC, id ASC
             FOR UPDATE SKIP LOCKED
             LIMIT 1
         )
         UPDATE check_jobs AS j
         SET status = 'running',
             started_at = NOW(),
             finished_at = NULL,
             error_message = NULL
         FROM candidate AS c
         WHERE j.id = c.id
           AND j.company_id = c.company_id
           AND j.status = 'queued'
         RETURNING j.id, j.company_id, j.stream_id, j.planned_at`,
	)

	var job claimedJob
	err := row.Scan(&job.ID, &job.CompanyID, &job.StreamID, &job.PlannedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return claimedJob{}, false, nil
		}
		return claimedJob{}, false, err
	}
	return job, true, nil
}

func (w *worker) runStub(ctx context.Context, job claimedJob) error {
	jobCtx, cancel := context.WithTimeout(ctx, w.jobTimeout)
	defer cancel()

	select {
	case <-time.After(w.stubDuration):
		return nil
	case <-jobCtx.Done():
		return jobCtx.Err()
	}
}

func (w *worker) finalizeWithRetry(ctx context.Context, job claimedJob, status string, errorMessage string) error {
	for attempt := 0; ; attempt++ {
		err := w.finalizeJob(ctx, job, status, errorMessage)
		if err == nil {
			return nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker finalize retry attempt=%d job_id=%d backoff=%s err=%v", attempt+1, job.ID, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func (w *worker) finalizeJob(ctx context.Context, job claimedJob, status string, errorMessage string) error {
	var nullableErrorMessage interface{}
	if errorMessage == "" {
		nullableErrorMessage = nil
	} else {
		nullableErrorMessage = errorMessage
	}

	result, err := w.db.ExecContext(
		ctx,
		`UPDATE check_jobs
         SET status = $1,
             finished_at = NOW(),
             error_message = $2
         WHERE id = $3
           AND company_id = $4
           AND status = 'running'`,
		status,
		nullableErrorMessage,
		job.ID,
		job.CompanyID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		log.Printf("worker finalize skipped (state already changed): id=%d company_id=%d target_status=%s", job.ID, job.CompanyID, status)
	}
	return nil
}

func isRetryableWorkerError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		errorClass := string(pqErr.Code.Class())
		return errorClass == "08" || errorClass == "53" || errorClass == "57"
	}

	return false
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func intAtLeast(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}
