package worker

import (
	"context"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
	"github.com/example/hls-monitoring-platform/internal/service/worker/checks"
)

func (w *worker) fetchPlaylist(ctx context.Context, streamURL string) (string, error) {
	return checks.FetchPlaylist(ctx, streamURL, w.playlistTimeout)
}

func (w *worker) checkFreshness(playlistBody string) string {
	return checks.FreshnessStatus(playlistBody, time.Now(), w.freshnessWarn, w.freshnessFail)
}

func (w *worker) checkFreeze(playlistBody string) (string, map[string]interface{}) {
	return checks.FreezeCheck(playlistBody, time.Now(), w.freezeWarn, w.freezeFail)
}

func (w *worker) checkBlackframe(ctx context.Context, samples []segmentSample) (string, map[string]interface{}) {
	return checks.BlackframeCheck(ctx, samples, w.segmentTimeout, w.blackframeWarnRatio, w.blackframeFailRatio)
}

func (w *worker) checkSegmentsAvailability(ctx context.Context, segments []playlistSegment) (string, []segmentSample) {
	for _, segment := range segments {
		if err := w.validateExternalURL(segment.URL); err != nil {
			return domain.WorkerStatusFail, nil
		}
	}
	return checks.CheckSegmentsAvailability(ctx, segments, w.segmentTimeout)
}

func extractLatestPlaylistSegments(playlistURL string, playlistBody string, sampleCount int) ([]playlistSegment, error) {
	return checks.ExtractLatestPlaylistSegments(playlistURL, playlistBody, sampleCount)
}
