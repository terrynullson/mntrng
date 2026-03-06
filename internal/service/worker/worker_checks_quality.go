package worker

import "github.com/terrynullson/mntrng/internal/service/worker/checks"

func checkDeclaredBitrate(playlistBody string) declaredBitrateResult {
	return checks.DeclaredBitrate(playlistBody)
}

func (w *worker) checkEffectiveBitrate(samples []segmentSample, declared declaredBitrateResult) (string, map[string]interface{}) {
	return checks.EffectiveBitrateStatus(samples, declared, w.effectiveWarnRatio, w.effectiveFailRatio)
}

func freezeStatusByThreshold(maxFreezeSec float64, warnSec float64, failSec float64) string {
	return checks.FreezeStatusByThreshold(maxFreezeSec, warnSec, failSec)
}

func blackframeStatusByThreshold(darkFrameRatio float64, warnRatio float64, failRatio float64) string {
	return checks.BlackframeStatusByThreshold(darkFrameRatio, warnRatio, failRatio)
}

func aggregateStatuses(statuses ...string) string {
	return checks.AggregateStatuses(statuses...)
}
