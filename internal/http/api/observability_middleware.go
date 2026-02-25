package api

import (
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpObservabilityOnce sync.Once
	httpRequestsTotal     = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_http_requests_total",
			Help: "Total number of HTTP requests served by API.",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds by method, path and status.",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"method", "path", "status"},
	)
	pathDigitsRegexp = regexp.MustCompile(`/\d+`)
)

func ensureHTTPObservabilityMetrics() {
	httpObservabilityOnce.Do(func() {
		prometheus.MustRegister(httpRequestsTotal, httpRequestDurationSeconds)
	})
}

func withHTTPObservability(next http.Handler) http.Handler {
	ensureHTTPObservabilityMetrics()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		responseRecorder := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		requestID := RequestIDFromRequest(r)
		responseRecorder.Header().Set("X-Request-ID", requestID)
		r = r.WithContext(withRequestID(r.Context(), requestID))

		next.ServeHTTP(responseRecorder, r)

		duration := time.Since(start)
		status := strconv.Itoa(responseRecorder.statusCode)
		normalizedPath := normalizePathForMetrics(r.URL.Path)
		httpRequestsTotal.WithLabelValues(r.Method, normalizedPath, status).Inc()
		httpRequestDurationSeconds.WithLabelValues(r.Method, normalizedPath, status).Observe(duration.Seconds())

		log.Printf(
			"api_request method=%s path=%s status=%d duration_ms=%d request_id=%s remote_ip=%s",
			r.Method,
			r.URL.Path,
			responseRecorder.statusCode,
			duration.Milliseconds(),
			requestID,
			clientIP(r),
		)
	})
}

func normalizePathForMetrics(path string) string {
	if path == "" {
		return "/"
	}
	return pathDigitsRegexp.ReplaceAllString(path, "/:id")
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
