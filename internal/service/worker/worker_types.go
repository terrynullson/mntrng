package worker

import (
	"time"

	"github.com/example/hls-monitoring-platform/internal/ai"
	"github.com/example/hls-monitoring-platform/internal/domain"
)

type Config struct {
	PollInterval                  time.Duration
	RunningJobStaleTimeout        time.Duration
	RetentionTTL                  time.Duration
	RetentionCleanupInterval      time.Duration
	RetentionCleanupBatchSize     int
	JobTimeout                    time.Duration
	PlaylistTimeout               time.Duration
	SegmentTimeout                time.Duration
	SegmentsSampleCount           int
	FreshnessWarn                 time.Duration
	FreshnessFail                 time.Duration
	FreezeWarn                    time.Duration
	FreezeFail                    time.Duration
	BlackframeWarnRatio           float64
	BlackframeFailRatio           float64
	EffectiveWarnRatio            float64
	EffectiveFailRatio            float64
	AlertFailStreak               int
	AlertCooldown                 time.Duration
	AlertSendRecovered            bool
	TelegramHTTPTimeout           time.Duration
	TelegramRetryMax              int
	TelegramRetryBackoff          time.Duration
	TelegramBotTokenDefault       string
	RetryMax                      int
	RetryBackoff                  time.Duration
	IncidentAnalyzer              ai.Analyzer
	DataDir                       string
	IncidentScreenshotInterval    time.Duration
	DiagnosticCaptureTimeout      time.Duration
	DiagnosticFreezeInterval      time.Duration
	DiagnosticFreezeDiffThreshold float64
	AllowPrivateStreamURLs        bool
}

type claimedJob = domain.WorkerClaimedJob

type worker struct {
	jobRepo                       JobRepository
	streamRepo                    StreamRepository
	checkResultRepo               CheckResultRepository
	alertStateRepo                AlertStateRepository
	telegramSettingsRepo          TelegramSettingsRepository
	retentionRepo                 RetentionRepository
	aiIncidentRepo                AIIncidentRepository
	incidentRepo                  IncidentRepository
	incidentAnalyzer              ai.Analyzer
	retentionTTL                  time.Duration
	runningJobStaleTimeout        time.Duration
	retentionCleanupBatchSize     int
	jobTimeout                    time.Duration
	playlistTimeout               time.Duration
	segmentTimeout                time.Duration
	segmentsSampleCount           int
	freshnessWarn                 time.Duration
	freshnessFail                 time.Duration
	freezeWarn                    time.Duration
	freezeFail                    time.Duration
	blackframeWarnRatio           float64
	blackframeFailRatio           float64
	effectiveWarnRatio            float64
	effectiveFailRatio            float64
	alertFailStreak               int
	alertCooldown                 time.Duration
	alertSendRecovered            bool
	telegramHTTPTimeout           time.Duration
	telegramRetryMax              int
	telegramRetryBackoff          time.Duration
	telegramBotTokenDefault       string
	retryMax                      int
	retryBackoff                  time.Duration
	dataDir                       string
	incidentScreenshotInterval    time.Duration
	diagnosticCaptureTimeout      time.Duration
	diagnosticFreezeInterval      time.Duration
	diagnosticFreezeDiffThreshold float64
	allowPrivateStreamURLs        bool
}

type checkJobEvaluation = domain.WorkerCheckJobEvaluation
type playlistSegment = domain.WorkerPlaylistSegment
type segmentSample = domain.WorkerSegmentSample
type declaredBitrateResult = domain.WorkerDeclaredBitrateResult

type alertDecision = domain.WorkerAlertDecision
type telegramDeliverySettings = domain.WorkerTelegramDeliverySettings
type retentionCandidate = domain.WorkerRetentionCandidate

func NewWorker(cfg Config, repos Repositories) *worker {
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = "/data"
	}
	incidentScreenshotInterval := cfg.IncidentScreenshotInterval
	if incidentScreenshotInterval <= 0 {
		incidentScreenshotInterval = 10 * time.Minute
	}
	diagnosticCaptureTimeout := cfg.DiagnosticCaptureTimeout
	if diagnosticCaptureTimeout <= 0 {
		diagnosticCaptureTimeout = 6 * time.Second
	}
	diagnosticFreezeInterval := cfg.DiagnosticFreezeInterval
	if diagnosticFreezeInterval <= 0 {
		diagnosticFreezeInterval = 2 * time.Second
	}
	diagnosticFreezeDiffThreshold := cfg.DiagnosticFreezeDiffThreshold
	if diagnosticFreezeDiffThreshold <= 0 {
		diagnosticFreezeDiffThreshold = 0.01
	}
	runningJobStaleTimeout := cfg.RunningJobStaleTimeout
	if runningJobStaleTimeout <= 0 {
		runningJobStaleTimeout = 5 * time.Minute
	}
	return &worker{
		jobRepo:                       repos.JobRepo,
		streamRepo:                    repos.StreamRepo,
		checkResultRepo:               repos.CheckResultRepo,
		alertStateRepo:                repos.AlertStateRepo,
		telegramSettingsRepo:          repos.TelegramSettingsRepo,
		retentionRepo:                 repos.RetentionRepo,
		aiIncidentRepo:                repos.AIIncidentRepo,
		incidentRepo:                  repos.IncidentRepo,
		incidentAnalyzer:              cfg.IncidentAnalyzer,
		retentionTTL:                  cfg.RetentionTTL,
		runningJobStaleTimeout:        runningJobStaleTimeout,
		retentionCleanupBatchSize:     cfg.RetentionCleanupBatchSize,
		jobTimeout:                    cfg.JobTimeout,
		playlistTimeout:               cfg.PlaylistTimeout,
		segmentTimeout:                cfg.SegmentTimeout,
		segmentsSampleCount:           cfg.SegmentsSampleCount,
		freshnessWarn:                 cfg.FreshnessWarn,
		freshnessFail:                 cfg.FreshnessFail,
		freezeWarn:                    cfg.FreezeWarn,
		freezeFail:                    cfg.FreezeFail,
		blackframeWarnRatio:           cfg.BlackframeWarnRatio,
		blackframeFailRatio:           cfg.BlackframeFailRatio,
		effectiveWarnRatio:            cfg.EffectiveWarnRatio,
		effectiveFailRatio:            cfg.EffectiveFailRatio,
		alertFailStreak:               cfg.AlertFailStreak,
		alertCooldown:                 cfg.AlertCooldown,
		alertSendRecovered:            cfg.AlertSendRecovered,
		telegramHTTPTimeout:           cfg.TelegramHTTPTimeout,
		telegramRetryMax:              cfg.TelegramRetryMax,
		telegramRetryBackoff:          cfg.TelegramRetryBackoff,
		telegramBotTokenDefault:       cfg.TelegramBotTokenDefault,
		retryMax:                      cfg.RetryMax,
		retryBackoff:                  cfg.RetryBackoff,
		dataDir:                       dataDir,
		incidentScreenshotInterval:    incidentScreenshotInterval,
		diagnosticCaptureTimeout:      diagnosticCaptureTimeout,
		diagnosticFreezeInterval:      diagnosticFreezeInterval,
		diagnosticFreezeDiffThreshold: diagnosticFreezeDiffThreshold,
		allowPrivateStreamURLs:        cfg.AllowPrivateStreamURLs,
	}
}
