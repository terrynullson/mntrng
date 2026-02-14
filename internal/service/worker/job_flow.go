package worker

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

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
		if finalizeErr := w.finalizeWithRetry(ctx, job, domain.WorkerJobStatusFailed, processErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, processErr.Error())
		return nil
	}

	if persistErr := w.persistCheckResultWithRetry(ctx, job, evaluation); persistErr != nil {
		if finalizeErr := w.finalizeWithRetry(ctx, job, domain.WorkerJobStatusFailed, persistErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, persistErr.Error())
		return nil
	}

	alertDecision, alertErr := w.applyAlertStateWithRetry(ctx, job, evaluation.DBStatus)
	if alertErr != nil {
		if finalizeErr := w.finalizeWithRetry(ctx, job, domain.WorkerJobStatusFailed, alertErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, alertErr.Error())
		return nil
	}
	w.logAlertDecision(job, alertDecision)
	w.processTelegramDelivery(ctx, job, evaluation, alertDecision)

	if finalizeErr := w.finalizeWithRetry(ctx, job, domain.WorkerJobStatusDone, ""); finalizeErr != nil {
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
             WHERE status = $1
             ORDER BY planned_at ASC, id ASC
             FOR UPDATE SKIP LOCKED
             LIMIT 1
         )
         UPDATE check_jobs AS j
         SET status = $2,
             started_at = NOW(),
             finished_at = NULL,
             error_message = NULL
         FROM candidate AS c
         WHERE j.id = c.id
           AND j.company_id = c.company_id
           AND j.status = $1
         RETURNING j.id, j.company_id, j.stream_id, j.planned_at`,
		domain.WorkerJobStatusQueued,
		domain.WorkerJobStatusRunning,
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
