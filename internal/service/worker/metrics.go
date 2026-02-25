package worker

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	workerMetricsOnce = sync.Once{}

	workerCycleTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_cycle_total",
			Help: "Total number of worker cycles by result.",
		},
		[]string{"result"},
	)
	workerCycleDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "worker_cycle_duration_seconds",
			Help:    "Worker cycle duration in seconds by result.",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
		[]string{"result"},
	)
	workerRetentionCleanupTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_retention_cleanup_total",
			Help: "Total number of retention cleanup runs by result.",
		},
		[]string{"result"},
	)
	workerJobFinalizedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_job_finalized_total",
			Help: "Total number of finalized jobs by final status.",
		},
		[]string{"status"},
	)
	workerJobDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "worker_job_duration_seconds",
			Help:    "Duration of worker job processing by final status.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"status"},
	)
)

func ensureWorkerMetricsRegistered() {
	workerMetricsOnce.Do(func() {
		prometheus.MustRegister(
			workerCycleTotal,
			workerCycleDurationSeconds,
			workerRetentionCleanupTotal,
			workerJobFinalizedTotal,
			workerJobDurationSeconds,
		)
	})
}

func observeWorkerCycle(result string, startedAt time.Time) {
	ensureWorkerMetricsRegistered()
	workerCycleTotal.WithLabelValues(result).Inc()
	workerCycleDurationSeconds.WithLabelValues(result).Observe(time.Since(startedAt).Seconds())
}

func observeRetentionCleanup(result string) {
	ensureWorkerMetricsRegistered()
	workerRetentionCleanupTotal.WithLabelValues(result).Inc()
}

func observeFinalizedJob(status string, startedAt time.Time) {
	ensureWorkerMetricsRegistered()
	workerJobFinalizedTotal.WithLabelValues(status).Inc()
	workerJobDurationSeconds.WithLabelValues(status).Observe(time.Since(startedAt).Seconds())
}
