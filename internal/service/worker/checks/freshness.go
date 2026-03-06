package checks

import (
	"strings"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
)

func FreshnessStatus(playlistBody string, now time.Time, warnThreshold time.Duration, failThreshold time.Duration) string {
	lastProgramDateTime, ok := extractLatestProgramDateTime(playlistBody)
	if !ok {
		return domain.WorkerStatusFail
	}

	age := now.UTC().Sub(lastProgramDateTime)
	if age < 0 {
		age = 0
	}

	if age >= failThreshold {
		return domain.WorkerStatusFail
	}
	if age >= warnThreshold {
		return domain.WorkerStatusWarn
	}
	return domain.WorkerStatusOK
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
