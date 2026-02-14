package worker

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
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

func (w *worker) processJob(ctx context.Context, job claimedJob) (checkJobEvaluation, error) {
	jobCtx, cancel := context.WithTimeout(ctx, w.jobTimeout)
	defer cancel()

	streamURL, err := w.loadStreamURL(jobCtx, job.CompanyID, job.StreamID)
	if err != nil {
		return checkJobEvaluation{}, err
	}

	playlistBody, playlistErr := w.fetchPlaylist(jobCtx, streamURL)
	playlistStatus := domain.WorkerStatusOK
	freshnessStatus := domain.WorkerStatusFail
	segmentsStatus := domain.WorkerStatusFail
	freezeStatus := domain.WorkerStatusFail
	blackframeStatus := domain.WorkerStatusWarn
	declaredBitrate := declaredBitrateResult{
		Status: domain.WorkerStatusFail,
		Details: map[string]interface{}{
			"reason": "playlist_unavailable",
		},
	}
	freezeDetails := map[string]interface{}{
		"max_freeze_sec": w.freezeFail.Seconds(),
		"reason":         "playlist_unavailable",
		"source":         "playlist_http",
	}
	blackframeDetails := map[string]interface{}{
		"dark_frame_ratio": 0.0,
		"analyzed_frames":  0,
		"reason":           "playlist_unavailable",
		"source":           "ffmpeg_blackframe",
	}
	effectiveBitrateStatus := domain.WorkerStatusFail
	effectiveBitrateDetails := map[string]interface{}{
		"calculated_bps": 0.0,
		"declared_bps":   int64(0),
		"ratio":          nil,
		"reason":         "playlist_unavailable",
		"sample_count":   0,
	}

	if playlistErr != nil {
		playlistStatus = domain.WorkerStatusFail
	} else {
		freshnessStatus = w.checkFreshness(playlistBody)
		freezeStatus, freezeDetails = w.checkFreeze(playlistBody)

		segments, segmentParseErr := extractLatestPlaylistSegments(streamURL, playlistBody, w.segmentsSampleCount)
		segmentSamples := make([]segmentSample, 0)
		if segmentParseErr != nil {
			segmentsStatus = domain.WorkerStatusFail
			blackframeStatus = domain.WorkerStatusWarn
			blackframeDetails = map[string]interface{}{
				"dark_frame_ratio": 0.0,
				"analyzed_frames":  0,
				"reason":           "segments_not_available",
				"source":           "ffmpeg_blackframe",
			}
			effectiveBitrateStatus = domain.WorkerStatusFail
			effectiveBitrateDetails = map[string]interface{}{
				"calculated_bps": 0.0,
				"declared_bps":   int64(0),
				"ratio":          nil,
				"reason":         "segments_not_available",
				"sample_count":   0,
			}
		} else {
			segmentsStatus, segmentSamples = w.checkSegmentsAvailability(jobCtx, segments)
			blackframeStatus, blackframeDetails = w.checkBlackframe(jobCtx, segmentSamples)
		}

		declaredBitrate = checkDeclaredBitrate(playlistBody)
		if segmentParseErr == nil {
			effectiveBitrateStatus, effectiveBitrateDetails = w.checkEffectiveBitrate(segmentSamples, declaredBitrate)
		}
	}

	aggregate := aggregateStatuses(playlistStatus, freshnessStatus, segmentsStatus, freezeStatus, blackframeStatus, declaredBitrate.Status, effectiveBitrateStatus)

	return checkJobEvaluation{
		DBStatus:  checkStatusToDBStatus(aggregate),
		Aggregate: aggregate,
		Checks: map[string]interface{}{
			"playlist":                  playlistStatus,
			"freshness":                 freshnessStatus,
			"segments":                  segmentsStatus,
			"freeze":                    freezeStatus,
			"freeze_details":            freezeDetails,
			"blackframe":                blackframeStatus,
			"blackframe_details":        blackframeDetails,
			"declared_bitrate":          declaredBitrate.Status,
			"declared_bitrate_details":  declaredBitrate.Details,
			"effective_bitrate":         effectiveBitrateStatus,
			"effective_bitrate_details": effectiveBitrateDetails,
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
