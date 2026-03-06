package checks

import (
	"github.com/terrynullson/hls_mntrng/internal/domain"
)

func EffectiveBitrateStatus(
	samples []domain.WorkerSegmentSample,
	declared domain.WorkerDeclaredBitrateResult,
	warnRatio float64,
	failRatio float64,
) (string, map[string]interface{}) {
	availableSamples := 0
	totalBytes := int64(0)
	totalDurationSec := 0.0

	for _, sample := range samples {
		if !sample.Downloaded {
			continue
		}
		availableSamples++
		totalBytes += sample.Bytes
		totalDurationSec += sample.DurationSec
	}

	details := map[string]interface{}{
		"calculated_bps": 0.0,
		"declared_bps":   declared.DeclaredBPS,
		"ratio":          nil,
		"reason":         "",
		"sample_count":   availableSamples,
	}

	if availableSamples == 0 {
		details["reason"] = "no_downloaded_segments"
		return domain.WorkerStatusFail, details
	}
	if totalDurationSec <= 0 {
		details["reason"] = "invalid_segment_duration"
		return domain.WorkerStatusFail, details
	}

	calculatedBPS := (float64(totalBytes) * 8.0) / totalDurationSec
	details["calculated_bps"] = calculatedBPS

	if declared.DeclaredBPS <= 0 {
		details["reason"] = "declared_bitrate_unavailable"
		return domain.WorkerStatusWarn, details
	}

	ratio := calculatedBPS / float64(declared.DeclaredBPS)
	details["ratio"] = ratio

	if ratio < failRatio {
		details["reason"] = "ratio_below_fail_threshold"
		return domain.WorkerStatusFail, details
	}
	if ratio < warnRatio {
		details["reason"] = "ratio_below_warn_threshold"
		return domain.WorkerStatusWarn, details
	}

	details["reason"] = "within_threshold"
	return domain.WorkerStatusOK, details
}
