package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/example/hls-monitoring-platform/internal/ai"
	"github.com/example/hls-monitoring-platform/internal/domain"
)

// runAIIncidentIfNeeded calls the AI analyzer on WARN/FAIL and persists the result.
// Does not fail the job on AI or save errors; logs and returns.
func (w *worker) runAIIncidentIfNeeded(ctx context.Context, job claimedJob, evaluation checkJobEvaluation) {
	if w.incidentAnalyzer == nil || w.aiIncidentRepo == nil {
		return
	}
	status := evaluation.DBStatus
	if status != domain.WorkerStatusDBWarn && status != domain.WorkerStatusDBFail {
		return
	}

	screenshotPath := fmt.Sprintf("storage/%d/%d/%d.jpg", job.CompanyID, job.StreamID, job.ID)
	input := ai.IncidentInput{
		Checks:         evaluation.Checks,
		ScreenshotPath: screenshotPath,
		CompanyID:      job.CompanyID,
		StreamID:       job.StreamID,
		JobID:          job.ID,
	}

	result, err := w.incidentAnalyzer.Analyze(ctx, input)
	if err != nil {
		log.Printf("worker ai incident: job_id=%d company_id=%d stream_id=%d skip_save=analysis_error", job.ID, job.CompanyID, job.StreamID)
		return
	}

	if err := w.aiIncidentRepo.SaveAIIncidentResult(ctx, job.ID, job.CompanyID, job.StreamID, result.Cause, result.Summary); err != nil {
		log.Printf("worker ai incident: job_id=%d company_id=%d stream_id=%d save_error=1", job.ID, job.CompanyID, job.StreamID)
		return
	}
}
