package worker

import (
	"context"
	"log"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (w *worker) ProcessSingleJobCycle(ctx context.Context) error {
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

	w.runAIIncidentIfNeeded(ctx, job, evaluation)

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
	job, ok, err := w.jobRepo.ClaimNextQueuedJob(ctx)
	if err != nil {
		return claimedJob{}, false, err
	}
	return job, ok, nil
}
