package checks

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

var blackframeEventPattern = regexp.MustCompile(`frame:\s*\d+\s+pblack:\s*\d+`)

func BlackframeStatusByThreshold(darkFrameRatio float64, warnRatio float64, failRatio float64) string {
	if darkFrameRatio >= failRatio {
		return domain.WorkerStatusFail
	}
	if darkFrameRatio >= warnRatio {
		return domain.WorkerStatusWarn
	}
	return domain.WorkerStatusOK
}

func BlackframeCheck(
	ctx context.Context,
	samples []domain.WorkerSegmentSample,
	segmentTimeout time.Duration,
	warnRatio float64,
	failRatio float64,
) (string, map[string]interface{}) {
	downloadedCount := 0
	totalAnalyzedFrames := 0
	totalDarkFrames := 0

	for _, sample := range samples {
		if !sample.Downloaded || strings.TrimSpace(sample.URL) == "" {
			continue
		}
		downloadedCount++

		analyzeCtx, cancel := context.WithTimeout(ctx, segmentTimeout)
		darkFrames, analyzedFrames, err := analyzeBlackframeForSegment(analyzeCtx, sample.URL)
		cancel()
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return blackframeWarn("blackframe_analyzer_not_available")
			}
			continue
		}

		totalDarkFrames += darkFrames
		totalAnalyzedFrames += analyzedFrames
	}

	if downloadedCount == 0 {
		return blackframeWarn("no_downloaded_segments")
	}
	if totalAnalyzedFrames == 0 {
		return blackframeWarn("blackframe_analysis_failed")
	}

	darkFrameRatio := float64(totalDarkFrames) / float64(totalAnalyzedFrames)
	status := BlackframeStatusByThreshold(darkFrameRatio, warnRatio, failRatio)
	reason := "within_threshold"
	if status == domain.WorkerStatusWarn {
		reason = "blackframe_warn_threshold_reached"
	}
	if status == domain.WorkerStatusFail {
		reason = "blackframe_fail_threshold_reached"
	}

	return status, map[string]interface{}{
		"dark_frame_ratio": darkFrameRatio,
		"analyzed_frames":  totalAnalyzedFrames,
		"reason":           reason,
		"source":           "ffmpeg_blackframe",
	}
}

func blackframeWarn(reason string) (string, map[string]interface{}) {
	return domain.WorkerStatusWarn, map[string]interface{}{
		"dark_frame_ratio": 0.0,
		"analyzed_frames":  0,
		"reason":           reason,
		"source":           "ffmpeg_blackframe",
	}
}

func analyzeBlackframeForSegment(ctx context.Context, segmentURL string) (int, int, error) {
	analyzedFrames, err := probeVideoFrameCount(ctx, segmentURL)
	if err != nil {
		return 0, 0, err
	}

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-v", "info",
		"-hide_banner",
		"-nostats",
		"-i", segmentURL,
		"-vf", "blackframe",
		"-an",
		"-f", "null",
		"-",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return 0, 0, err
	}

	darkFrames := len(blackframeEventPattern.FindAll(output, -1))
	return darkFrames, analyzedFrames, nil
}

func probeVideoFrameCount(ctx context.Context, segmentURL string) (int, error) {
	command := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "error",
		"-count_frames",
		"-select_streams", "v:0",
		"-show_entries", "stream=nb_read_frames",
		"-of", "default=noprint_wrappers=1:nokey=1",
		segmentURL,
	)
	output, err := command.Output()
	if err != nil {
		return 0, err
	}

	value := strings.TrimSpace(string(output))
	if value == "" || strings.EqualFold(value, "N/A") {
		return 0, errors.New("ffprobe returned empty frame count")
	}

	frameCount, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if frameCount <= 0 {
		return 0, errors.New("ffprobe returned non-positive frame count")
	}
	return frameCount, nil
}
