package domain

import (
	"database/sql"
	"time"
)

type WorkerClaimedJob struct {
	ID        int64
	CompanyID int64
	StreamID  int64
	PlannedAt time.Time
}

type WorkerCheckJobEvaluation struct {
	DBStatus  string
	Aggregate string
	Checks    map[string]interface{}
}

type WorkerPlaylistSegment struct {
	URL         string
	DurationSec float64
}

type WorkerSegmentSample struct {
	URL         string
	Downloaded  bool
	DurationSec float64
	Bytes       int64
}

type WorkerDeclaredBitrateResult struct {
	Status      string
	DeclaredBPS int64
	Details     map[string]interface{}
}

type WorkerAlertDecision struct {
	ShouldSend     bool
	EventType      string
	Reason         string
	CurrentStatus  string
	PreviousStatus string
	FailStreak     int
	CooldownUntil  *time.Time
}

type WorkerAlertTransitionResult struct {
	Decision          WorkerAlertDecision
	NextFailStreak    int
	NextCooldownUntil sql.NullTime
	NextLastAlertAt   sql.NullTime
}

type WorkerTelegramDeliverySettings struct {
	IsEnabled     bool
	ChatID        string
	SendRecovered bool
	BotTokenRef   string
}

type WorkerRetentionCandidate struct {
	ID             int64
	ScreenshotPath string
}

const (
	WorkerJobStatusQueued  = "queued"
	WorkerJobStatusRunning = "running"
	WorkerJobStatusDone    = "done"
	WorkerJobStatusFailed  = "failed"

	WorkerStatusOK   = "OK"
	WorkerStatusWarn = "WARN"
	WorkerStatusFail = "FAIL"

	WorkerStatusDBOK   = "ok"
	WorkerStatusDBWarn = "warn"
	WorkerStatusDBFail = "fail"

	WorkerAlertEventFail      = "fail"
	WorkerAlertEventRecovered = "recovered"
)
