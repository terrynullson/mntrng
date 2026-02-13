package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
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
	db              *sql.DB
	pollInterval    time.Duration
	jobTimeout      time.Duration
	playlistTimeout time.Duration
	freshnessWarn   time.Duration
	freshnessFail   time.Duration
	retryMax        int
	retryBackoff    time.Duration
}

type checkJobEvaluation struct {
	DBStatus  string
	Aggregate string
	Checks    map[string]string
}

func main() {
	pollInterval := time.Duration(intAtLeast(config.GetInt("WORKER_HEARTBEAT_SEC", 15), 1)) * time.Second
	jobTimeout := time.Duration(intAtLeast(config.GetInt("WORKER_JOB_TIMEOUT_SEC", 30), 1)) * time.Second
	playlistTimeout := time.Duration(intAtLeast(config.GetInt("PLAYLIST_TIMEOUT_MS", 3000), 1)) * time.Millisecond
	freshnessWarn := time.Duration(intAtLeast(config.GetInt("FRESHNESS_WARN_SEC", 10), 1)) * time.Second
	freshnessFail := time.Duration(intAtLeast(config.GetInt("FRESHNESS_FAIL_SEC", 30), 1)) * time.Second
	if freshnessFail < freshnessWarn {
		freshnessFail = freshnessWarn
	}
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
		db:              db,
		pollInterval:    pollInterval,
		jobTimeout:      jobTimeout,
		playlistTimeout: playlistTimeout,
		freshnessWarn:   freshnessWarn,
		freshnessFail:   freshnessFail,
		retryMax:        retryMax,
		retryBackoff:    retryBackoff,
	}

	log.Printf(
		"worker skeleton started: poll_interval=%s, job_timeout=%s, playlist_timeout=%s, freshness_warn=%s, freshness_fail=%s, retry_max=%d, retry_backoff=%s",
		w.pollInterval,
		w.jobTimeout,
		w.playlistTimeout,
		w.freshnessWarn,
		w.freshnessFail,
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

	evaluation, processErr := w.processJob(ctx, job)
	if processErr != nil {
		if finalizeErr := w.finalizeWithRetry(ctx, job, "failed", processErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, processErr.Error())
		return nil
	}

	if persistErr := w.persistCheckResultWithRetry(ctx, job, evaluation); persistErr != nil {
		if finalizeErr := w.finalizeWithRetry(ctx, job, "failed", persistErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, persistErr.Error())
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

func (w *worker) processJob(ctx context.Context, job claimedJob) (checkJobEvaluation, error) {
	jobCtx, cancel := context.WithTimeout(ctx, w.jobTimeout)
	defer cancel()

	streamURL, err := w.loadStreamURL(jobCtx, job.CompanyID, job.StreamID)
	if err != nil {
		return checkJobEvaluation{}, err
	}

	playlistStatus := w.checkPlaylistAvailability(jobCtx, streamURL)
	freshnessStatus := w.checkFreshness(jobCtx, streamURL, playlistStatus)
	aggregate := aggregateStatuses(playlistStatus, freshnessStatus)

	return checkJobEvaluation{
		DBStatus:  strings.ToLower(aggregate),
		Aggregate: aggregate,
		Checks: map[string]string{
			"playlist":  playlistStatus,
			"freshness": freshnessStatus,
		},
	}, nil
}

func (w *worker) loadStreamURL(ctx context.Context, companyID int64, streamID int64) (string, error) {
	var streamURL string
	err := w.db.QueryRowContext(
		ctx,
		`SELECT url
         FROM streams
         WHERE company_id = $1
           AND id = $2`,
		companyID,
		streamID,
	).Scan(&streamURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("stream not found in tenant context")
		}
		return "", err
	}
	streamURL = strings.TrimSpace(streamURL)
	if streamURL == "" {
		return "", errors.New("stream url is empty")
	}
	return streamURL, nil
}

func (w *worker) checkPlaylistAvailability(ctx context.Context, streamURL string) string {
	requestCtx, cancel := context.WithTimeout(ctx, w.playlistTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "FAIL"
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "FAIL"
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "FAIL"
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return "FAIL"
	}
	if !strings.Contains(string(body), "#EXTM3U") {
		return "FAIL"
	}

	return "OK"
}

func (w *worker) checkFreshness(ctx context.Context, streamURL string, playlistStatus string) string {
	if playlistStatus == "FAIL" {
		return "FAIL"
	}

	requestCtx, cancel := context.WithTimeout(ctx, w.playlistTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "FAIL"
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "FAIL"
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "FAIL"
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return "FAIL"
	}

	lastProgramDateTime, ok := extractLatestProgramDateTime(string(body))
	if !ok {
		return "FAIL"
	}

	age := time.Since(lastProgramDateTime)
	if age < 0 {
		age = 0
	}

	if age >= w.freshnessFail {
		return "FAIL"
	}
	if age >= w.freshnessWarn {
		return "WARN"
	}
	return "OK"
}

func extractLatestProgramDateTime(playlist string) (time.Time, bool) {
	lines := strings.Split(playlist, "\n")
	var latest time.Time
	found := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:") {
			continue
		}

		value := strings.TrimSpace(strings.TrimPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:"))
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			continue
		}

		if !found || parsed.After(latest) {
			latest = parsed
			found = true
		}
	}

	return latest, found
}

func aggregateStatuses(statuses ...string) string {
	hasWarn := false
	for _, status := range statuses {
		switch status {
		case "FAIL":
			return "FAIL"
		case "WARN":
			hasWarn = true
		}
	}
	if hasWarn {
		return "WARN"
	}
	return "OK"
}

func (w *worker) persistCheckResultWithRetry(ctx context.Context, job claimedJob, evaluation checkJobEvaluation) error {
	for attempt := 0; ; attempt++ {
		err := w.persistCheckResult(ctx, job, evaluation)
		if err == nil {
			return nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker persist retry attempt=%d job_id=%d backoff=%s err=%v", attempt+1, job.ID, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func (w *worker) persistCheckResult(ctx context.Context, job claimedJob, evaluation checkJobEvaluation) error {
	checksJSON, err := json.Marshal(evaluation.Checks)
	if err != nil {
		return err
	}

	result, err := w.db.ExecContext(
		ctx,
		`INSERT INTO check_results (company_id, job_id, stream_id, status, checks)
         VALUES ($1, $2, $3, $4, $5::jsonb)
         ON CONFLICT (job_id) DO NOTHING`,
		job.CompanyID,
		job.ID,
		job.StreamID,
		evaluation.DBStatus,
		string(checksJSON),
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		var existingCount int
		countErr := w.db.QueryRowContext(
			ctx,
			`SELECT COUNT(1)
             FROM check_results
             WHERE company_id = $1
               AND job_id = $2
               AND stream_id = $3`,
			job.CompanyID,
			job.ID,
			job.StreamID,
		).Scan(&existingCount)
		if countErr != nil {
			return countErr
		}
		if existingCount == 0 {
			return errors.New("check_result conflict without matching tenant row")
		}
	}

	log.Printf(
		"worker stored check_result: job_id=%d company_id=%d stream_id=%d status=%s checks=%v",
		job.ID,
		job.CompanyID,
		job.StreamID,
		evaluation.Aggregate,
		evaluation.Checks,
	)
	return nil
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
