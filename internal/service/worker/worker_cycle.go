package worker

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

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
