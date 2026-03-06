package workerapp

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/terrynullson/mntrng/internal/ai"
	"github.com/terrynullson/mntrng/internal/config"
	postgresrepo "github.com/terrynullson/mntrng/internal/repo/postgres"
	workerservice "github.com/terrynullson/mntrng/internal/service/worker"
)

type RuntimeConfig struct {
	Worker        workerservice.Config
	MetricsPort   int
	MetricsToken  string
	AppEnv        string
	DatabaseURL   string
	DBPingTimeout time.Duration
}

func LoadRuntimeConfig() RuntimeConfig {
	pollInterval := time.Duration(config.IntAtLeast(config.GetInt("WORKER_HEARTBEAT_SEC", 15), 1)) * time.Second
	retentionTTLDays := config.IntAtLeast(config.GetInt("RETENTION_TTL_DAYS", 30), 1)
	retentionTTL := time.Duration(retentionTTLDays) * 24 * time.Hour
	retentionCleanupInterval := time.Duration(config.IntAtLeast(config.GetInt("RETENTION_CLEANUP_INTERVAL_MIN", 60), 1)) * time.Minute
	retentionCleanupBatchSize := config.IntAtLeast(config.GetInt("RETENTION_CLEANUP_BATCH_SIZE", 100), 1)
	jobTimeout := time.Duration(config.IntAtLeast(config.GetInt("WORKER_JOB_TIMEOUT_SEC", 30), 1)) * time.Second
	playlistTimeout := time.Duration(config.IntAtLeast(config.GetInt("PLAYLIST_TIMEOUT_MS", 3000), 1)) * time.Millisecond
	segmentTimeout := time.Duration(config.IntAtLeast(config.GetInt("SEGMENT_TIMEOUT_MS", 5000), 1)) * time.Millisecond
	segmentsSampleCount := config.IntInRange(config.GetInt("SEGMENTS_SAMPLE_COUNT", 3), 3, 5)
	freshnessWarn := time.Duration(config.IntAtLeast(config.GetInt("FRESHNESS_WARN_SEC", 10), 1)) * time.Second
	freshnessFail := time.Duration(config.IntAtLeast(config.GetInt("FRESHNESS_FAIL_SEC", 30), 1)) * time.Second
	freezeWarn := time.Duration(config.IntAtLeast(config.GetInt("FREEZE_WARN_SEC", 2), 1)) * time.Second
	freezeFail := time.Duration(config.IntAtLeast(config.GetInt("FREEZE_FAIL_SEC", 5), 1)) * time.Second
	blackframeWarnRatio := config.FloatInRange(config.GetFloat("BLACKFRAME_WARN_RATIO", 0.9), 0, 1)
	blackframeFailRatio := config.FloatInRange(config.GetFloat("BLACKFRAME_FAIL_RATIO", 0.98), 0, 1)
	effectiveWarnRatio := config.FloatAtLeast(config.GetFloat("EFFECTIVE_BITRATE_WARN_RATIO", 0.7), 0)
	effectiveFailRatio := config.FloatAtLeast(config.GetFloat("EFFECTIVE_BITRATE_FAIL_RATIO", 0.4), 0)
	alertFailStreak := config.IntAtLeast(config.GetInt("ALERT_FAIL_STREAK", 2), 1)
	alertCooldown := time.Duration(config.IntAtLeast(config.GetInt("ALERT_COOLDOWN_MIN", 10), 1)) * time.Minute
	alertSendRecovered := config.GetBool("ALERT_SEND_RECOVERED", false)
	telegramHTTPTimeout := time.Duration(config.IntAtLeast(config.GetInt("TELEGRAM_HTTP_TIMEOUT_MS", 5000), 1)) * time.Millisecond
	telegramRetryMax := config.IntAtLeast(config.GetInt("TELEGRAM_SEND_RETRY_MAX", 2), 0)
	telegramRetryBackoff := time.Duration(config.IntAtLeast(config.GetInt("TELEGRAM_SEND_RETRY_BACKOFF_MS", 500), 1)) * time.Millisecond
	telegramBotTokenDefault := config.GetString("TELEGRAM_BOT_TOKEN_DEFAULT", "")
	dataDir := config.GetString("APP_DATA_DIR", "/data")
	incidentScreenshotInterval := time.Duration(config.IntAtLeast(config.GetInt("INCIDENT_SCREENSHOT_INTERVAL_MIN", 10), 1)) * time.Minute
	diagnosticCaptureTimeout := time.Duration(config.IntInRange(config.GetInt("DIAG_CAPTURE_TIMEOUT_SEC", 6), 3, 15)) * time.Second
	diagnosticFreezeInterval := time.Duration(config.IntInRange(config.GetInt("DIAG_FREEZE_INTERVAL_SEC", 2), 1, 10)) * time.Second
	diagnosticFreezeDiffThreshold := config.FloatInRange(config.GetFloat("DIAG_FREEZE_DIFF_THRESHOLD", 0.01), 0.0001, 1)
	runningJobStaleTimeout := time.Duration(config.IntAtLeast(config.GetInt("WORKER_RUNNING_JOB_STALE_SEC", 300), 30)) * time.Second
	allowPrivateStreamURLs := config.GetBool("WORKER_ALLOW_PRIVATE_STREAM_URLS", false)
	if freshnessFail < freshnessWarn {
		freshnessFail = freshnessWarn
	}
	if freezeFail < freezeWarn {
		freezeFail = freezeWarn
	}
	if blackframeFailRatio < blackframeWarnRatio {
		blackframeFailRatio = blackframeWarnRatio
	}
	retryMax := config.IntAtLeast(config.GetInt("WORKER_DB_RETRY_MAX", 2), 0)
	retryBackoff := time.Duration(config.IntAtLeast(config.GetInt("WORKER_DB_RETRY_BACKOFF_MS", 500), 1)) * time.Millisecond
	workerMetricsPort := config.IntAtLeast(config.GetInt("WORKER_METRICS_PORT", 9091), 1)
	workerMetricsToken := config.GetString("WORKER_METRICS_TOKEN", "")
	appEnv := strings.ToLower(strings.TrimSpace(config.GetString("APP_ENV", "development")))
	aiIncidentEnabled := config.GetBool("AI_INCIDENT_ENABLED", false)

	var incidentAnalyzer ai.Analyzer
	if aiIncidentEnabled {
		// Stub only; real provider not integrated. When enabled, stub runs and writes empty cause/summary.
		incidentAnalyzer = &ai.LogAnalyzer{Inner: ai.NewStubAnalyzer()}
	}
	// When AI_INCIDENT_ENABLED=false (default): no AI call, no stub. Honest non-production default.

	return RuntimeConfig{
		Worker: workerservice.Config{
			PollInterval:                  pollInterval,
			RetentionTTL:                  retentionTTL,
			RetentionCleanupInterval:      retentionCleanupInterval,
			RetentionCleanupBatchSize:     retentionCleanupBatchSize,
			JobTimeout:                    jobTimeout,
			PlaylistTimeout:               playlistTimeout,
			SegmentTimeout:                segmentTimeout,
			SegmentsSampleCount:           segmentsSampleCount,
			FreshnessWarn:                 freshnessWarn,
			FreshnessFail:                 freshnessFail,
			FreezeWarn:                    freezeWarn,
			FreezeFail:                    freezeFail,
			BlackframeWarnRatio:           blackframeWarnRatio,
			BlackframeFailRatio:           blackframeFailRatio,
			EffectiveWarnRatio:            effectiveWarnRatio,
			EffectiveFailRatio:            effectiveFailRatio,
			AlertFailStreak:               alertFailStreak,
			AlertCooldown:                 alertCooldown,
			AlertSendRecovered:            alertSendRecovered,
			TelegramHTTPTimeout:           telegramHTTPTimeout,
			TelegramRetryMax:              telegramRetryMax,
			TelegramRetryBackoff:          telegramRetryBackoff,
			TelegramBotTokenDefault:       telegramBotTokenDefault,
			RetryMax:                      retryMax,
			RetryBackoff:                  retryBackoff,
			IncidentAnalyzer:              incidentAnalyzer,
			DataDir:                       dataDir,
			IncidentScreenshotInterval:    incidentScreenshotInterval,
			DiagnosticCaptureTimeout:      diagnosticCaptureTimeout,
			DiagnosticFreezeInterval:      diagnosticFreezeInterval,
			DiagnosticFreezeDiffThreshold: diagnosticFreezeDiffThreshold,
			RunningJobStaleTimeout:        runningJobStaleTimeout,
			AllowPrivateStreamURLs:        allowPrivateStreamURLs,
		},
		MetricsPort:   workerMetricsPort,
		MetricsToken:  workerMetricsToken,
		AppEnv:        appEnv,
		DatabaseURL:   config.GetString("DATABASE_URL", ""),
		DBPingTimeout: 5 * time.Second,
	}
}

func (cfg RuntimeConfig) Validate() error {
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if strings.TrimSpace(cfg.MetricsToken) == "" && cfg.AppEnv == "production" {
		return fmt.Errorf("WORKER_METRICS_TOKEN is required in production")
	}
	return nil
}

func LogRuntimeConfig(cfg RuntimeConfig) {
	workerConfig := cfg.Worker
	if strings.TrimSpace(cfg.MetricsToken) == "" {
		log.Printf("worker metrics: WORKER_METRICS_TOKEN is empty; /metrics endpoint is public")
	}

	log.Printf(
		"worker skeleton started: poll_interval=%s, retention_ttl=%s, retention_cleanup_interval=%s, retention_cleanup_batch_size=%d, job_timeout=%s, playlist_timeout=%s, segment_timeout=%s, segments_sample_count=%d, freshness_warn=%s, freshness_fail=%s, freeze_warn=%s, freeze_fail=%s, blackframe_warn_ratio=%.2f, blackframe_fail_ratio=%.2f, effective_warn_ratio=%.2f, effective_fail_ratio=%.2f, alert_fail_streak=%d, alert_cooldown=%s, alert_send_recovered=%t, telegram_http_timeout=%s, telegram_retry_max=%d, telegram_retry_backoff=%s, telegram_default_token_set=%t, retry_max=%d, retry_backoff=%s, ai_incident_enabled=%t, data_dir=%s, incident_screenshot_interval=%s, diag_capture_timeout=%s, diag_freeze_interval=%s, diag_freeze_diff_threshold=%.4f, running_job_stale_timeout=%s, allow_private_stream_urls=%t",
		workerConfig.PollInterval,
		workerConfig.RetentionTTL,
		workerConfig.RetentionCleanupInterval,
		workerConfig.RetentionCleanupBatchSize,
		workerConfig.JobTimeout,
		workerConfig.PlaylistTimeout,
		workerConfig.SegmentTimeout,
		workerConfig.SegmentsSampleCount,
		workerConfig.FreshnessWarn,
		workerConfig.FreshnessFail,
		workerConfig.FreezeWarn,
		workerConfig.FreezeFail,
		workerConfig.BlackframeWarnRatio,
		workerConfig.BlackframeFailRatio,
		workerConfig.EffectiveWarnRatio,
		workerConfig.EffectiveFailRatio,
		workerConfig.AlertFailStreak,
		workerConfig.AlertCooldown,
		workerConfig.AlertSendRecovered,
		workerConfig.TelegramHTTPTimeout,
		workerConfig.TelegramRetryMax,
		workerConfig.TelegramRetryBackoff,
		workerConfig.TelegramBotTokenDefault != "",
		workerConfig.RetryMax,
		workerConfig.RetryBackoff,
		workerConfig.IncidentAnalyzer != nil,
		workerConfig.DataDir,
		workerConfig.IncidentScreenshotInterval,
		workerConfig.DiagnosticCaptureTimeout,
		workerConfig.DiagnosticFreezeInterval,
		workerConfig.DiagnosticFreezeDiffThreshold,
		workerConfig.RunningJobStaleTimeout,
		workerConfig.AllowPrivateStreamURLs,
	)
}

func NewApp(db *sql.DB, cfg RuntimeConfig) *workerservice.App {
	workerRepo := postgresrepo.NewWorkerRepo(db)
	incidentRepo := postgresrepo.NewWorkerIncidentRepo(db)
	w := workerservice.NewWorker(
		cfg.Worker,
		workerservice.Repositories{
			JobRepo:              workerRepo,
			StreamRepo:           workerRepo,
			CheckResultRepo:      workerRepo,
			AlertStateRepo:       workerRepo,
			TelegramSettingsRepo: workerRepo,
			RetentionRepo:        workerRepo,
			AIIncidentRepo:       workerRepo,
			IncidentRepo:         incidentRepo,
		},
	)

	return workerservice.NewApp(
		cfg.Worker.PollInterval,
		cfg.Worker.RetentionCleanupInterval,
		cfg.Worker.RetryMax,
		cfg.Worker.RetryBackoff,
		workerservice.IsRetryableWorkerError,
		w.ProcessSingleJobCycle,
		w.RunRetentionCleanup,
	)
}

func StartMetricsServer(ctx context.Context, port int, metricsToken string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsAuthMiddleware(metricsToken, promhttp.Handler()))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"worker"}`))
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	go func() {
		log.Printf("worker metrics server listening on :%d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("worker metrics server error: %v", err)
		}
	}()
}

func metricsAuthMiddleware(expectedToken string, next http.Handler) http.Handler {
	trimmedToken := strings.TrimSpace(expectedToken)
	if trimmedToken == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("metrics token is required"))
			return
		}

		token := strings.TrimSpace(authHeader[len("Bearer "):])
		if token != trimmedToken {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("invalid metrics token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
