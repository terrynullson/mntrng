package checks

import (
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

func FreezeStatusByThreshold(maxFreezeSec float64, warnSec float64, failSec float64) string {
	if maxFreezeSec >= failSec {
		return domain.WorkerStatusFail
	}
	if maxFreezeSec >= warnSec {
		return domain.WorkerStatusWarn
	}
	return domain.WorkerStatusOK
}

func FreezeCheck(playlistBody string, now time.Time, warnThreshold time.Duration, failThreshold time.Duration) (string, map[string]interface{}) {
	lastProgramDateTime, ok := extractLatestProgramDateTime(playlistBody)
	if !ok {
		return FreezeStatusByThreshold(0, warnThreshold.Seconds(), failThreshold.Seconds()), map[string]interface{}{
			"max_freeze_sec": 0.0,
			"reason":         "program_date_time_not_found",
			"source":         "playlist_ext_x_program_date_time",
		}
	}

	maxFreezeSec := now.UTC().Sub(lastProgramDateTime).Seconds()
	if maxFreezeSec < 0 {
		maxFreezeSec = 0
	}

	status := FreezeStatusByThreshold(maxFreezeSec, warnThreshold.Seconds(), failThreshold.Seconds())
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
