package worker

import (
	"context"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
)

type JobRepository interface {
	ClaimNextQueuedJob(ctx context.Context) (domain.WorkerClaimedJob, bool, error)
	RequeueStaleRunningJobs(ctx context.Context, staleAfter time.Duration) (int64, error)
	FinalizeJob(ctx context.Context, job domain.WorkerClaimedJob, status string, errorMessage string) (int64, error)
}

type StreamRepository interface {
	LoadStreamURL(ctx context.Context, companyID int64, streamID int64) (string, error)
}

type CheckResultRepository interface {
	PersistCheckResult(ctx context.Context, job domain.WorkerClaimedJob, dbStatus string, checksJSON string) error
}

type AlertStateRepository interface {
	ApplyAlertState(
		ctx context.Context,
		companyID int64,
		streamID int64,
		currentStatus string,
		failStreakThreshold int,
		alertCooldown time.Duration,
		alertSendRecovered bool,
	) (domain.WorkerAlertDecision, error)
}

type TelegramSettingsRepository interface {
	LoadTelegramDeliverySettings(ctx context.Context, companyID int64) (domain.WorkerTelegramDeliverySettings, bool, error)
}

type RetentionRepository interface {
	ListCompanyIDsForRetention(ctx context.Context) ([]int64, error)
	ListRetentionCandidates(ctx context.Context, companyID int64, cutoff time.Time, batchSize int) ([]domain.WorkerRetentionCandidate, error)
	DeleteStaleCheckResult(ctx context.Context, companyID int64, resultID int64, cutoff time.Time) (int64, error)
}

type AIIncidentRepository interface {
	SaveAIIncidentResult(ctx context.Context, jobID int64, companyID int64, streamID int64, cause string, summary string) error
}

type IncidentRepository interface {
	GetOpenByStream(ctx context.Context, companyID int64, streamID int64) (incident domain.Incident, ok bool, err error)
	Create(ctx context.Context, companyID int64, streamID int64, severity string, failReason string, sampleScreenshotPath *string, lastCheckID *int64) (incidentID int64, err error)
	UpdateOpen(ctx context.Context, incidentID int64, companyID int64, severity string, failReason string, sampleScreenshotPath *string, lastCheckID *int64) error
	UpdateDiagnostic(ctx context.Context, incidentID int64, companyID int64, streamID int64, sampleScreenshotPath *string, screenshotTakenAt time.Time, diagCode string, diagDetails map[string]interface{}) error
	Resolve(ctx context.Context, incidentID int64, companyID int64, streamID int64) error
}

type Repositories struct {
	JobRepo              JobRepository
	StreamRepo           StreamRepository
	CheckResultRepo      CheckResultRepository
	AlertStateRepo       AlertStateRepository
	TelegramSettingsRepo TelegramSettingsRepository
	RetentionRepo        RetentionRepository
	AIIncidentRepo       AIIncidentRepository
	IncidentRepo         IncidentRepository
}
