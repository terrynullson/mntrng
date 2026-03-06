package checks

import "github.com/terrynullson/hls_mntrng/internal/domain"

func AggregateStatuses(statuses ...string) string {
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
