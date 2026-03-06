package checks

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

func CheckSegmentsAvailability(
	ctx context.Context,
	segments []domain.WorkerPlaylistSegment,
	segmentTimeout time.Duration,
) (string, []domain.WorkerSegmentSample) {
	if len(segments) == 0 {
		return domain.WorkerStatusFail, nil
	}

	availableCount := 0
	samples := make([]domain.WorkerSegmentSample, 0, len(segments))
	for _, segment := range segments {
		requestCtx, cancel := context.WithTimeout(ctx, segmentTimeout)
		bytesRead, err := downloadSegmentBytes(requestCtx, segment.URL)
		cancel()

		sample := domain.WorkerSegmentSample{
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

func ExtractLatestPlaylistSegments(
	playlistURL string,
	playlistBody string,
	sampleCount int,
) ([]domain.WorkerPlaylistSegment, error) {
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

	resolvedSegments := make([]domain.WorkerPlaylistSegment, 0, len(segments)-startIndex)
	for _, segment := range segments[startIndex:] {
		parsedReference, parseErr := url.Parse(segment.URL)
		if parseErr != nil {
			return nil, parseErr
		}
		resolvedURL := baseURL.ResolveReference(parsedReference)
		resolvedSegments = append(resolvedSegments, domain.WorkerPlaylistSegment{
			URL:         resolvedURL.String(),
			DurationSec: segment.DurationSec,
		})
	}

	return resolvedSegments, nil
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

func extractPlaylistSegments(playlistBody string) []domain.WorkerPlaylistSegment {
	lines := strings.Split(playlistBody, "\n")
	segments := make([]domain.WorkerPlaylistSegment, 0, len(lines))
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
			segments = append(segments, domain.WorkerPlaylistSegment{
				URL:         line,
				DurationSec: currentDuration,
			})
			expectSegmentURI = false
			currentDuration = 0
		}
	}

	return segments
}
