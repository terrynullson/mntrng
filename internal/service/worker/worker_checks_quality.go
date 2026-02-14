package worker

import (
	"strconv"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func checkDeclaredBitrate(playlistBody string) declaredBitrateResult {
	lines := strings.Split(playlistBody, "\n")
	streamInfoCount := 0
	invalidCount := 0
	missingAttributeCount := 0
	declaredValues := make([]int64, 0, 4)
	usedAverageBandwidth := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			continue
		}

		streamInfoCount++
		attributes := parseM3U8Attributes(strings.TrimPrefix(line, "#EXT-X-STREAM-INF:"))

		if valueRaw, ok := attributes["BANDWIDTH"]; ok {
			value, err := strconv.ParseInt(valueRaw, 10, 64)
			if err != nil || value <= 0 {
				invalidCount++
				continue
			}
			declaredValues = append(declaredValues, value)
			continue
		}

		if valueRaw, ok := attributes["AVERAGE-BANDWIDTH"]; ok {
			value, err := strconv.ParseInt(valueRaw, 10, 64)
			if err != nil || value <= 0 {
				invalidCount++
				continue
			}
			declaredValues = append(declaredValues, value)
			usedAverageBandwidth = true
			continue
		}

		missingAttributeCount++
	}

	if len(declaredValues) == 0 {
		if streamInfoCount == 0 {
			return declaredBitrateResult{
				Status:      domain.WorkerStatusWarn,
				DeclaredBPS: 0,
				Details: map[string]interface{}{
					"reason": "no_stream_inf_tags",
				},
			}
		}
		if invalidCount > 0 {
			return declaredBitrateResult{
				Status:      domain.WorkerStatusFail,
				DeclaredBPS: 0,
				Details: map[string]interface{}{
					"reason":          "invalid_declared_bitrate",
					"invalid_entries": invalidCount,
				},
			}
		}
		return declaredBitrateResult{
			Status:      domain.WorkerStatusWarn,
			DeclaredBPS: 0,
			Details: map[string]interface{}{
				"reason":                  "declared_bitrate_not_present",
				"stream_info_entries":     streamInfoCount,
				"missing_attribute_count": missingAttributeCount,
			},
		}
	}

	maxDeclared := declaredValues[0]
	for _, value := range declaredValues[1:] {
		if value > maxDeclared {
			maxDeclared = value
		}
	}

	bitrateSource := "bandwidth"
	if usedAverageBandwidth {
		bitrateSource = "average_bandwidth"
	}

	return declaredBitrateResult{
		Status:      domain.WorkerStatusOK,
		DeclaredBPS: maxDeclared,
		Details: map[string]interface{}{
			"parsed_bitrate_bps":  maxDeclared,
			"variants_considered": len(declaredValues),
			"source":              bitrateSource,
		},
	}
}

func parseM3U8Attributes(attributesRaw string) map[string]string {
	attributes := make(map[string]string)
	parts := strings.Split(attributesRaw, ",")
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		keyValue := strings.SplitN(item, "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		key := strings.TrimSpace(keyValue[0])
		value := strings.Trim(strings.TrimSpace(keyValue[1]), "\"")
		if key == "" || value == "" {
			continue
		}
		attributes[strings.ToUpper(key)] = value
	}
	return attributes
}

func (w *worker) checkEffectiveBitrate(samples []segmentSample, declared declaredBitrateResult) (string, map[string]interface{}) {
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

	if ratio < w.effectiveFailRatio {
		details["reason"] = "ratio_below_fail_threshold"
		return domain.WorkerStatusFail, details
	}
	if ratio < w.effectiveWarnRatio {
		details["reason"] = "ratio_below_warn_threshold"
		return domain.WorkerStatusWarn, details
	}

	details["reason"] = "within_threshold"
	return domain.WorkerStatusOK, details
}

func freezeStatusByThreshold(maxFreezeSec float64, warnSec float64, failSec float64) string {
	if maxFreezeSec >= failSec {
		return domain.WorkerStatusFail
	}
	if maxFreezeSec >= warnSec {
		return domain.WorkerStatusWarn
	}
	return domain.WorkerStatusOK
}

func blackframeStatusByThreshold(darkFrameRatio float64, warnRatio float64, failRatio float64) string {
	if darkFrameRatio >= failRatio {
		return domain.WorkerStatusFail
	}
	if darkFrameRatio >= warnRatio {
		return domain.WorkerStatusWarn
	}
	return domain.WorkerStatusOK
}

func aggregateStatuses(statuses ...string) string {
	hasWarn := false
	for _, status := range statuses {
		switch status {
		case domain.WorkerStatusFail:
			return domain.WorkerStatusFail
		case domain.WorkerStatusWarn:
			hasWarn = true
		}
	}
	if hasWarn {
		return domain.WorkerStatusWarn
	}
	return domain.WorkerStatusOK
}
