package worker

import (
	"context"
	"database/sql"
	"regexp"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

type Config struct {
	PollInterval              time.Duration
	RetentionTTL              time.Duration
	RetentionCleanupInterval  time.Duration
	RetentionCleanupBatchSize int
	JobTimeout                time.Duration
	PlaylistTimeout           time.Duration
	SegmentTimeout            time.Duration
	SegmentsSampleCount       int
	FreshnessWarn             time.Duration
	FreshnessFail             time.Duration
	FreezeWarn                time.Duration
	FreezeFail                time.Duration
	BlackframeWarnRatio       float64
	BlackframeFailRatio       float64
	EffectiveWarnRatio        float64
	EffectiveFailRatio        float64
	AlertFailStreak           int
	AlertCooldown             time.Duration
	AlertSendRecovered        bool
	TelegramHTTPTimeout       time.Duration
	TelegramRetryMax          int
	TelegramRetryBackoff      time.Duration
	TelegramBotTokenDefault   string
	RetryMax                  int
	RetryBackoff              time.Duration
}

type claimedJob = domain.WorkerClaimedJob

type worker struct {
	db                        *sql.DB
	pollInterval              time.Duration
	retentionTTL              time.Duration
	retentionCleanupInterval  time.Duration
	retentionCleanupBatchSize int
	jobTimeout                time.Duration
	playlistTimeout           time.Duration
	segmentTimeout            time.Duration
	segmentsSampleCount       int
	freshnessWarn             time.Duration
	freshnessFail             time.Duration
	freezeWarn                time.Duration
	freezeFail                time.Duration
	blackframeWarnRatio       float64
	blackframeFailRatio       float64
	effectiveWarnRatio        float64
	effectiveFailRatio        float64
	alertFailStreak           int
	alertCooldown             time.Duration
	alertSendRecovered        bool
	telegramHTTPTimeout       time.Duration
	telegramRetryMax          int
	telegramRetryBackoff      time.Duration
	telegramBotTokenDefault   string
	retryMax                  int
	retryBackoff              time.Duration
}

type checkJobEvaluation = domain.WorkerCheckJobEvaluation
type playlistSegment = domain.WorkerPlaylistSegment
type segmentSample = domain.WorkerSegmentSample
type declaredBitrateResult = domain.WorkerDeclaredBitrateResult

var blackframeEventPattern = regexp.MustCompile(`frame:\s*\d+\s+pblack:\s*\d+`)

type alertDecision = domain.WorkerAlertDecision
type alertTransitionResult = domain.WorkerAlertTransitionResult
type telegramDeliverySettings = domain.WorkerTelegramDeliverySettings
type retentionCandidate = domain.WorkerRetentionCandidate

func NewWorker(db *sql.DB, cfg Config) *worker {
	return &worker{
		db:                        db,
		pollInterval:              cfg.PollInterval,
		retentionTTL:              cfg.RetentionTTL,
		retentionCleanupInterval:  cfg.RetentionCleanupInterval,
		retentionCleanupBatchSize: cfg.RetentionCleanupBatchSize,
		jobTimeout:                cfg.JobTimeout,
		playlistTimeout:           cfg.PlaylistTimeout,
		segmentTimeout:            cfg.SegmentTimeout,
		segmentsSampleCount:       cfg.SegmentsSampleCount,
		freshnessWarn:             cfg.FreshnessWarn,
		freshnessFail:             cfg.FreshnessFail,
		freezeWarn:                cfg.FreezeWarn,
		freezeFail:                cfg.FreezeFail,
		blackframeWarnRatio:       cfg.BlackframeWarnRatio,
		blackframeFailRatio:       cfg.BlackframeFailRatio,
		effectiveWarnRatio:        cfg.EffectiveWarnRatio,
		effectiveFailRatio:        cfg.EffectiveFailRatio,
		alertFailStreak:           cfg.AlertFailStreak,
		alertCooldown:             cfg.AlertCooldown,
		alertSendRecovered:        cfg.AlertSendRecovered,
		telegramHTTPTimeout:       cfg.TelegramHTTPTimeout,
		telegramRetryMax:          cfg.TelegramRetryMax,
		telegramRetryBackoff:      cfg.TelegramRetryBackoff,
		telegramBotTokenDefault:   cfg.TelegramBotTokenDefault,
		retryMax:                  cfg.RetryMax,
		retryBackoff:              cfg.RetryBackoff,
	}
}

func (w *worker) ProcessSingleJobCycle(ctx context.Context) error {
	return w.processSingleJobCycle(ctx)
}

func (w *worker) RunRetentionCleanup(ctx context.Context) error {
	return w.runRetentionCleanup(ctx)
}

func IsRetryableWorkerError(err error) bool {
	return isRetryableWorkerError(err)
}
