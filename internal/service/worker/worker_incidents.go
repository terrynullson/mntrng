package worker

import (
	"context"
	"log"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

// applyIncidentState creates, updates or resolves an incident based on check result.
// Does not fail the job; logs and returns.
func (w *worker) applyIncidentState(ctx context.Context, job claimedJob, evaluation checkJobEvaluation) {
	if w.incidentRepo == nil {
		return
	}
	status := evaluation.DBStatus
	severity := status
	failReason := evaluation.Aggregate
	if failReason == "" {
		failReason = status
	}

	switch status {
	case domain.WorkerStatusDBOK:
		incidentID, ok, err := w.incidentRepo.GetOpenByStream(ctx, job.CompanyID, job.StreamID)
		if err != nil {
			log.Printf("worker incident: get_open err company_id=%d stream_id=%d err=%v", job.CompanyID, job.StreamID, err)
			return
		}
		if !ok {
			return
		}
		if err := w.incidentRepo.Resolve(ctx, incidentID, job.CompanyID, job.StreamID); err != nil {
			log.Printf("worker incident: resolve err incident_id=%d company_id=%d stream_id=%d err=%v", incidentID, job.CompanyID, job.StreamID, err)
			return
		}
		log.Printf("worker incident: resolved incident_id=%d company_id=%d stream_id=%d", incidentID, job.CompanyID, job.StreamID)
		return

	case domain.WorkerStatusDBWarn, domain.WorkerStatusDBFail:
		incidentID, ok, err := w.incidentRepo.GetOpenByStream(ctx, job.CompanyID, job.StreamID)
		if err != nil {
			log.Printf("worker incident: get_open err company_id=%d stream_id=%d err=%v", job.CompanyID, job.StreamID, err)
			return
		}
		if ok {
			err = w.incidentRepo.UpdateOpen(ctx, incidentID, job.CompanyID, severity, failReason, nil, nil)
			if err != nil {
				log.Printf("worker incident: update err incident_id=%d company_id=%d stream_id=%d err=%v", incidentID, job.CompanyID, job.StreamID, err)
				return
			}
			log.Printf("worker incident: updated incident_id=%d company_id=%d stream_id=%d severity=%s", incidentID, job.CompanyID, job.StreamID, severity)
			return
		}
		incidentID, err = w.incidentRepo.Create(ctx, job.CompanyID, job.StreamID, severity, failReason, nil, nil)
		if err != nil {
			log.Printf("worker incident: create err company_id=%d stream_id=%d err=%v", job.CompanyID, job.StreamID, err)
			return
		}
		log.Printf("worker incident: created incident_id=%d company_id=%d stream_id=%d severity=%s", incidentID, job.CompanyID, job.StreamID, severity)
	default:
		return
	}
}
