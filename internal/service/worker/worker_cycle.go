package worker

import (
	"context"
	"errors"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

type jobProcessingState struct {
	playlistStatus          string
	freshnessStatus         string
	segmentsStatus          string
	freezeStatus            string
	blackframeStatus        string
	declaredBitrate         declaredBitrateResult
	freezeDetails           map[string]interface{}
	blackframeDetails       map[string]interface{}
	effectiveBitrateStatus  string
	effectiveBitrateDetails map[string]interface{}
	segmentSamples          []segmentSample
}

func newJobProcessingState(freezeFailSec float64) jobProcessingState {
	return jobProcessingState{
		playlistStatus:   domain.WorkerStatusOK,
		freshnessStatus:  domain.WorkerStatusFail,
		segmentsStatus:   domain.WorkerStatusFail,
		freezeStatus:     domain.WorkerStatusFail,
		blackframeStatus: domain.WorkerStatusWarn,
		declaredBitrate: declaredBitrateResult{
			Status: domain.WorkerStatusFail,
			Details: map[string]interface{}{
				"reason": "playlist_unavailable",
			},
		},
		freezeDetails: map[string]interface{}{
			"max_freeze_sec": freezeFailSec,
			"reason":         "playlist_unavailable",
			"source":         "playlist_http",
		},
		blackframeDetails: map[string]interface{}{
			"dark_frame_ratio": 0.0,
			"analyzed_frames":  0,
			"reason":           "playlist_unavailable",
			"source":           "ffmpeg_blackframe",
		},
		effectiveBitrateStatus: domain.WorkerStatusFail,
		effectiveBitrateDetails: map[string]interface{}{
			"calculated_bps": 0.0,
			"declared_bps":   int64(0),
			"ratio":          nil,
			"reason":         "playlist_unavailable",
			"sample_count":   0,
		},
		segmentSamples: make([]segmentSample, 0),
	}
}

func (w *worker) processJob(ctx context.Context, job claimedJob) (checkJobEvaluation, error) {
	jobCtx, cancel := context.WithTimeout(ctx, w.jobTimeout)
	defer cancel()

	streamURL, err := w.loadStreamURL(jobCtx, job.CompanyID, job.StreamID)
	if err != nil {
		return checkJobEvaluation{}, err
	}

	state := newJobProcessingState(w.freezeFail.Seconds())
	playlistBody, playlistErr := w.fetchPlaylist(jobCtx, streamURL)
	if playlistErr != nil {
		state.playlistStatus = domain.WorkerStatusFail
		return w.buildJobEvaluation(state), nil
	}

	w.evaluatePlaylistChecks(jobCtx, streamURL, playlistBody, &state)
	return w.buildJobEvaluation(state), nil
}

func (w *worker) evaluatePlaylistChecks(ctx context.Context, streamURL string, playlistBody string, state *jobProcessingState) {
	state.freshnessStatus = w.checkFreshness(playlistBody)
	state.freezeStatus, state.freezeDetails = w.checkFreeze(playlistBody)

	segments, segmentParseErr := extractLatestPlaylistSegments(streamURL, playlistBody, w.segmentsSampleCount)
	if segmentParseErr != nil {
		w.applySegmentParseFallbacks(state)
	} else {
		state.segmentsStatus, state.segmentSamples = w.checkSegmentsAvailability(ctx, segments)
		state.blackframeStatus, state.blackframeDetails = w.checkBlackframe(ctx, state.segmentSamples)
	}

	state.declaredBitrate = checkDeclaredBitrate(playlistBody)
	if segmentParseErr == nil {
		state.effectiveBitrateStatus, state.effectiveBitrateDetails = w.checkEffectiveBitrate(state.segmentSamples, state.declaredBitrate)
	}
}

func (w *worker) applySegmentParseFallbacks(state *jobProcessingState) {
	state.segmentsStatus = domain.WorkerStatusFail
	state.blackframeStatus = domain.WorkerStatusWarn
	state.blackframeDetails = map[string]interface{}{
		"dark_frame_ratio": 0.0,
		"analyzed_frames":  0,
		"reason":           "segments_not_available",
		"source":           "ffmpeg_blackframe",
	}
	state.effectiveBitrateStatus = domain.WorkerStatusFail
	state.effectiveBitrateDetails = map[string]interface{}{
		"calculated_bps": 0.0,
		"declared_bps":   int64(0),
		"ratio":          nil,
		"reason":         "segments_not_available",
		"sample_count":   0,
	}
}

func (w *worker) buildJobEvaluation(state jobProcessingState) checkJobEvaluation {
	aggregate := aggregateStatuses(
		state.playlistStatus,
		state.freshnessStatus,
		state.segmentsStatus,
		state.freezeStatus,
		state.blackframeStatus,
		state.declaredBitrate.Status,
		state.effectiveBitrateStatus,
	)

	return checkJobEvaluation{
		DBStatus:  checkStatusToDBStatus(aggregate),
		Aggregate: aggregate,
		Checks: map[string]interface{}{
			"playlist":                  state.playlistStatus,
			"freshness":                 state.freshnessStatus,
			"segments":                  state.segmentsStatus,
			"freeze":                    state.freezeStatus,
			"freeze_details":            state.freezeDetails,
			"blackframe":                state.blackframeStatus,
			"blackframe_details":        state.blackframeDetails,
			"declared_bitrate":          state.declaredBitrate.Status,
			"declared_bitrate_details":  state.declaredBitrate.Details,
			"effective_bitrate":         state.effectiveBitrateStatus,
			"effective_bitrate_details": state.effectiveBitrateDetails,
		},
	}
}

func (w *worker) loadStreamURL(ctx context.Context, companyID int64, streamID int64) (string, error) {
	streamURL, err := w.streamRepo.LoadStreamURL(ctx, companyID, streamID)
	if err != nil {
		return "", err
	}
	streamURL = strings.TrimSpace(streamURL)
	if streamURL == "" {
		return "", errors.New("stream url is empty")
	}
	return streamURL, nil
}
