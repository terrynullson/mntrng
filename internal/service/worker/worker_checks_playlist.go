package worker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (w *worker) fetchPlaylist(ctx context.Context, streamURL string) (string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, w.playlistTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", errors.New("playlist request returned non-2xx")
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	playlistBody := string(body)
	if !strings.Contains(playlistBody, "#EXTM3U") {
		return "", errors.New("playlist does not contain EXTM3U marker")
	}

	return playlistBody, nil
}

func (w *worker) checkFreshness(playlistBody string) string {
	lastProgramDateTime, ok := extractLatestProgramDateTime(playlistBody)
	if !ok {
		return domain.WorkerStatusFail
	}

	age := time.Since(lastProgramDateTime)
	if age < 0 {
		age = 0
	}

	if age >= w.freshnessFail {
		return domain.WorkerStatusFail
	}
	if age >= w.freshnessWarn {
		return domain.WorkerStatusWarn
	}
	return domain.WorkerStatusOK
}

func (w *worker) checkFreeze(playlistBody string) (string, map[string]interface{}) {
	lastProgramDateTime, ok := extractLatestProgramDateTime(playlistBody)
	if !ok {
		return freezeStatusByThreshold(0, w.freezeWarn.Seconds(), w.freezeFail.Seconds()), map[string]interface{}{
			"max_freeze_sec": 0.0,
			"reason":         "program_date_time_not_found",
			"source":         "playlist_ext_x_program_date_time",
		}
	}

	maxFreezeSec := time.Since(lastProgramDateTime).Seconds()
	if maxFreezeSec < 0 {
		maxFreezeSec = 0
	}

	status := freezeStatusByThreshold(maxFreezeSec, w.freezeWarn.Seconds(), w.freezeFail.Seconds())
	reason := "within_threshold"
	if status == domain.WorkerStatusWarn {
		reason = "freeze_warn_threshold_reached"
	}
	if status == domain.WorkerStatusFail {
		reason = "freeze_fail_threshold_reached"
	}

	return status, map[string]interface{}{
		"max_freeze_sec": maxFreezeSec,
		"reason":         reason,
		"source":         "playlist_ext_x_program_date_time",
	}
}

func (w *worker) checkBlackframe(ctx context.Context, samples []segmentSample) (string, map[string]interface{}) {
	downloadedCount := 0
	totalAnalyzedFrames := 0
	totalDarkFrames := 0

	for _, sample := range samples {
		if !sample.Downloaded || strings.TrimSpace(sample.URL) == "" {
			continue
		}
		downloadedCount++

		analyzeCtx, cancel := context.WithTimeout(ctx, w.segmentTimeout)
		darkFrames, analyzedFrames, err := analyzeBlackframeForSegment(analyzeCtx, sample.URL)
		cancel()
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return domain.WorkerStatusWarn, map[string]interface{}{
					"dark_frame_ratio": 0.0,
					"analyzed_frames":  0,
					"reason":           "blackframe_analyzer_not_available",
					"source":           "ffmpeg_blackframe",
				}
			}
			continue
		}

		totalDarkFrames += darkFrames
		totalAnalyzedFrames += analyzedFrames
	}

	if downloadedCount == 0 {
		return domain.WorkerStatusWarn, map[string]interface{}{
			"dark_frame_ratio": 0.0,
			"analyzed_frames":  0,
			"reason":           "no_downloaded_segments",
			"source":           "ffmpeg_blackframe",
		}
	}

	if totalAnalyzedFrames == 0 {
		return domain.WorkerStatusWarn, map[string]interface{}{
			"dark_frame_ratio": 0.0,
			"analyzed_frames":  0,
			"reason":           "blackframe_analysis_failed",
			"source":           "ffmpeg_blackframe",
		}
	}

	darkFrameRatio := float64(totalDarkFrames) / float64(totalAnalyzedFrames)
	status := blackframeStatusByThreshold(darkFrameRatio, w.blackframeWarnRatio, w.blackframeFailRatio)
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

func (w *worker) checkSegmentsAvailability(ctx context.Context, segments []playlistSegment) (string, []segmentSample) {
	if len(segments) == 0 {
		return domain.WorkerStatusFail, nil
	}

	availableCount := 0
	samples := make([]segmentSample, 0, len(segments))
	for _, segment := range segments {
		requestCtx, cancel := context.WithTimeout(ctx, w.segmentTimeout)
		bytesRead, err := downloadSegmentBytes(requestCtx, segment.URL)
		cancel()

		sample := segmentSample{
			URL:         segment.URL,
			Downloaded:  err == nil,
			DurationSec: segment.DurationSec,
			Bytes:       bytesRead,
		}
		samples = append(samples, sample)
		if sample.Downloaded {
			availableCount++
		}
	}

	if availableCount == len(segments) {
		return domain.WorkerStatusOK, samples
	}
	if availableCount == 0 {
		return domain.WorkerStatusFail, samples
	}
	return domain.WorkerStatusWarn, samples
}

func downloadSegmentBytes(ctx context.Context, resourceURL string) (int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err != nil {
		return 0, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, errors.New("resource request returned non-2xx")
	}

	return io.Copy(io.Discard, response.Body)
}

func extractLatestPlaylistSegments(playlistURL string, playlistBody string, sampleCount int) ([]playlistSegment, error) {
	if sampleCount <= 0 {
		return nil, errors.New("segments sample count must be positive")
	}

	baseURL, err := url.Parse(playlistURL)
	if err != nil {
		return nil, err
	}

	segments := extractPlaylistSegments(playlistBody)
	if len(segments) == 0 {
		return nil, errors.New("no segment references found in playlist")
	}

	startIndex := len(segments) - sampleCount
	if startIndex < 0 {
		startIndex = 0
	}

	resolvedSegments := make([]playlistSegment, 0, len(segments)-startIndex)
	for _, segment := range segments[startIndex:] {
		parsedReference, parseErr := url.Parse(segment.URL)
		if parseErr != nil {
			return nil, parseErr
		}
		resolvedURL := baseURL.ResolveReference(parsedReference)
		resolvedSegments = append(resolvedSegments, playlistSegment{
			URL:         resolvedURL.String(),
			DurationSec: segment.DurationSec,
		})
	}

	return resolvedSegments, nil
}

func extractPlaylistSegments(playlistBody string) []playlistSegment {
	lines := strings.Split(playlistBody, "\n")
	segments := make([]playlistSegment, 0, len(lines))
	expectSegmentURI := false
	currentDuration := 0.0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXTINF:") {
			durationValue := strings.TrimPrefix(line, "#EXTINF:")
			if commaIndex := strings.Index(durationValue, ","); commaIndex >= 0 {
				durationValue = durationValue[:commaIndex]
			}
			parsedDuration, err := strconv.ParseFloat(strings.TrimSpace(durationValue), 64)
			if err == nil && parsedDuration > 0 {
				currentDuration = parsedDuration
			} else {
				currentDuration = 0
			}
			expectSegmentURI = true
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if expectSegmentURI {
			segments = append(segments, playlistSegment{
				URL:         line,
				DurationSec: currentDuration,
			})
			expectSegmentURI = false
			currentDuration = 0
		}
	}

	return segments
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
