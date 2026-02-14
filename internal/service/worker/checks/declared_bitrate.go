package checks

import (
	"strconv"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func DeclaredBitrate(playlistBody string) domain.WorkerDeclaredBitrateResult {
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
		value, usedAverage, valueState := parseDeclaredBitrateValue(line)
		switch valueState {
		case declaredBitrateValueOK:
			declaredValues = append(declaredValues, value)
			if usedAverage {
				usedAverageBandwidth = true
			}
		case declaredBitrateValueInvalid:
			invalidCount++
		case declaredBitrateValueMissing:
			missingAttributeCount++
		}
	}

	return buildDeclaredBitrateResult(declaredValues, streamInfoCount, invalidCount, missingAttributeCount, usedAverageBandwidth)
}

type declaredBitrateValueState int

const (
	declaredBitrateValueMissing declaredBitrateValueState = iota
	declaredBitrateValueInvalid
	declaredBitrateValueOK
)

func parseDeclaredBitrateValue(streamInfoLine string) (int64, bool, declaredBitrateValueState) {
	attributes := parseM3U8Attributes(strings.TrimPrefix(streamInfoLine, "#EXT-X-STREAM-INF:"))

	if valueRaw, ok := attributes["BANDWIDTH"]; ok {
		value, err := strconv.ParseInt(valueRaw, 10, 64)
		if err != nil || value <= 0 {
			return 0, false, declaredBitrateValueInvalid
		}
		return value, false, declaredBitrateValueOK
	}

	if valueRaw, ok := attributes["AVERAGE-BANDWIDTH"]; ok {
		value, err := strconv.ParseInt(valueRaw, 10, 64)
		if err != nil || value <= 0 {
			return 0, false, declaredBitrateValueInvalid
		}
		return value, true, declaredBitrateValueOK
	}

	return 0, false, declaredBitrateValueMissing
}

func buildDeclaredBitrateResult(
	declaredValues []int64,
	streamInfoCount int,
	invalidCount int,
	missingAttributeCount int,
	usedAverageBandwidth bool,
) domain.WorkerDeclaredBitrateResult {
	if len(declaredValues) == 0 {
		return buildEmptyDeclaredBitrateResult(streamInfoCount, invalidCount, missingAttributeCount)
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

	return domain.WorkerDeclaredBitrateResult{
		Status:      domain.WorkerStatusOK,
		DeclaredBPS: maxDeclared,
		Details: map[string]interface{}{
			"parsed_bitrate_bps":  maxDeclared,
			"variants_considered": len(declaredValues),
			"source":              bitrateSource,
		},
	}
}

func buildEmptyDeclaredBitrateResult(streamInfoCount int, invalidCount int, missingAttributeCount int) domain.WorkerDeclaredBitrateResult {
	if streamInfoCount == 0 {
		return domain.WorkerDeclaredBitrateResult{
			Status:      domain.WorkerStatusWarn,
			DeclaredBPS: 0,
			Details: map[string]interface{}{
				"reason": "no_stream_inf_tags",
			},
		}
	}
	if invalidCount > 0 {
		return domain.WorkerDeclaredBitrateResult{
			Status:      domain.WorkerStatusFail,
			DeclaredBPS: 0,
			Details: map[string]interface{}{
				"reason":          "invalid_declared_bitrate",
				"invalid_entries": invalidCount,
			},
		}
	}

	return domain.WorkerDeclaredBitrateResult{
		Status:      domain.WorkerStatusWarn,
		DeclaredBPS: 0,
		Details: map[string]interface{}{
			"reason":                  "declared_bitrate_not_present",
			"stream_info_entries":     streamInfoCount,
			"missing_attribute_count": missingAttributeCount,
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
