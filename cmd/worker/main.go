package main

import (
	"context"
	"database/sql"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	workerservice "github.com/example/hls-monitoring-platform/internal/service/worker"
	_ "github.com/lib/pq"
)

func main() {
	pollInterval := time.Duration(workerservice.IntAtLeast(config.GetInt("WORKER_HEARTBEAT_SEC", 15), 1)) * time.Second
	retentionTTLDays := workerservice.IntAtLeast(config.GetInt("RETENTION_TTL_DAYS", 30), 1)
	retentionTTL := time.Duration(retentionTTLDays) * 24 * time.Hour
	retentionCleanupInterval := time.Duration(workerservice.IntAtLeast(config.GetInt("RETENTION_CLEANUP_INTERVAL_MIN", 60), 1)) * time.Minute
	retentionCleanupBatchSize := workerservice.IntAtLeast(config.GetInt("RETENTION_CLEANUP_BATCH_SIZE", 100), 1)
	jobTimeout := time.Duration(workerservice.IntAtLeast(config.GetInt("WORKER_JOB_TIMEOUT_SEC", 30), 1)) * time.Second
	playlistTimeout := time.Duration(workerservice.IntAtLeast(config.GetInt("PLAYLIST_TIMEOUT_MS", 3000), 1)) * time.Millisecond
	segmentTimeout := time.Duration(workerservice.IntAtLeast(config.GetInt("SEGMENT_TIMEOUT_MS", 5000), 1)) * time.Millisecond
	segmentsSampleCount := workerservice.IntInRange(config.GetInt("SEGMENTS_SAMPLE_COUNT", 3), 3, 5)
	freshnessWarn := time.Duration(workerservice.IntAtLeast(config.GetInt("FRESHNESS_WARN_SEC", 10), 1)) * time.Second
	freshnessFail := time.Duration(workerservice.IntAtLeast(config.GetInt("FRESHNESS_FAIL_SEC", 30), 1)) * time.Second
	freezeWarn := time.Duration(workerservice.IntAtLeast(config.GetInt("FREEZE_WARN_SEC", 2), 1)) * time.Second
	freezeFail := time.Duration(workerservice.IntAtLeast(config.GetInt("FREEZE_FAIL_SEC", 5), 1)) * time.Second
	blackframeWarnRatio := workerservice.FloatInRange(workerservice.EnvFloat("BLACKFRAME_WARN_RATIO", 0.9), 0, 1)
	blackframeFailRatio := workerservice.FloatInRange(workerservice.EnvFloat("BLACKFRAME_FAIL_RATIO", 0.98), 0, 1)
	effectiveWarnRatio := workerservice.FloatAtLeast(workerservice.EnvFloat("EFFECTIVE_BITRATE_WARN_RATIO", 0.7), 0)
	effectiveFailRatio := workerservice.FloatAtLeast(workerservice.EnvFloat("EFFECTIVE_BITRATE_FAIL_RATIO", 0.4), 0)
	alertFailStreak := workerservice.IntAtLeast(config.GetInt("ALERT_FAIL_STREAK", 2), 1)
	alertCooldown := time.Duration(workerservice.IntAtLeast(config.GetInt("ALERT_COOLDOWN_MIN", 10), 1)) * time.Minute
	alertSendRecovered := workerservice.EnvBool("ALERT_SEND_RECOVERED", false)
	telegramHTTPTimeout := time.Duration(workerservice.IntAtLeast(config.GetInt("TELEGRAM_HTTP_TIMEOUT_MS", 5000), 1)) * time.Millisecond
	telegramRetryMax := workerservice.IntAtLeast(config.GetInt("TELEGRAM_SEND_RETRY_MAX", 2), 0)
	telegramRetryBackoff := time.Duration(workerservice.IntAtLeast(config.GetInt("TELEGRAM_SEND_RETRY_BACKOFF_MS", 500), 1)) * time.Millisecond
	telegramBotTokenDefault := config.GetString("TELEGRAM_BOT_TOKEN_DEFAULT", "")
	if freshnessFail < freshnessWarn {
		freshnessFail = freshnessWarn
	}
	if freezeFail < freezeWarn {
		freezeFail = freezeWarn
	}
	if blackframeFailRatio < blackframeWarnRatio {
		blackframeFailRatio = blackframeWarnRatio
	}
	retryMax := workerservice.IntAtLeast(config.GetInt("WORKER_DB_RETRY_MAX", 2), 0)
	retryBackoff := time.Duration(workerservice.IntAtLeast(config.GetInt("WORKER_DB_RETRY_BACKOFF_MS", 500), 1)) * time.Millisecond

	databaseURL := config.GetString("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	workerConfig := workerservice.Config{
		PollInterval:              pollInterval,
		RetentionTTL:              retentionTTL,
		RetentionCleanupInterval:  retentionCleanupInterval,
		RetentionCleanupBatchSize: retentionCleanupBatchSize,
		JobTimeout:                jobTimeout,
		PlaylistTimeout:           playlistTimeout,
		SegmentTimeout:            segmentTimeout,
		SegmentsSampleCount:       segmentsSampleCount,
		FreshnessWarn:             freshnessWarn,
		FreshnessFail:             freshnessFail,
		FreezeWarn:                freezeWarn,
		FreezeFail:                freezeFail,
		BlackframeWarnRatio:       blackframeWarnRatio,
		BlackframeFailRatio:       blackframeFailRatio,
		EffectiveWarnRatio:        effectiveWarnRatio,
		EffectiveFailRatio:        effectiveFailRatio,
		AlertFailStreak:           alertFailStreak,
		AlertCooldown:             alertCooldown,
		AlertSendRecovered:        alertSendRecovered,
		TelegramHTTPTimeout:       telegramHTTPTimeout,
		TelegramRetryMax:          telegramRetryMax,
		TelegramRetryBackoff:      telegramRetryBackoff,
		TelegramBotTokenDefault:   telegramBotTokenDefault,
		RetryMax:                  retryMax,
		RetryBackoff:              retryBackoff,
	}

	log.Printf(
		"worker skeleton started: poll_interval=%s, retention_ttl=%s, retention_cleanup_interval=%s, retention_cleanup_batch_size=%d, job_timeout=%s, playlist_timeout=%s, segment_timeout=%s, segments_sample_count=%d, freshness_warn=%s, freshness_fail=%s, freeze_warn=%s, freeze_fail=%s, blackframe_warn_ratio=%.2f, blackframe_fail_ratio=%.2f, effective_warn_ratio=%.2f, effective_fail_ratio=%.2f, alert_fail_streak=%d, alert_cooldown=%s, alert_send_recovered=%t, telegram_http_timeout=%s, telegram_retry_max=%d, telegram_retry_backoff=%s, telegram_default_token_set=%t, retry_max=%d, retry_backoff=%s",
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
	)

	w := workerservice.NewWorker(db, workerConfig)
	app := workerservice.NewApp(
		workerConfig.PollInterval,
		workerConfig.RetentionCleanupInterval,
		workerConfig.RetryMax,
		workerConfig.RetryBackoff,
		workerservice.IsRetryableWorkerError,
		w.ProcessSingleJobCycle,
		w.RunRetentionCleanup,
	)
	app.Run(ctx)
}
