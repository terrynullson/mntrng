package domain

import (
	"encoding/json"
	"time"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
}

type Company struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CompanyListResponse struct {
	Items      []Company `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}

type CreateCompanyRequest struct {
	Name string `json:"name"`
}

type PatchCompanyRequest struct {
	Name string `json:"name"`
}

type Project struct {
	ID        int64     `json:"id"`
	CompanyID int64     `json:"company_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProjectListResponse struct {
	Items      []Project `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

type PatchProjectRequest struct {
	Name string `json:"name"`
}

type Stream struct {
	ID         int64     `json:"id"`
	CompanyID  int64     `json:"company_id"`
	ProjectID  int64     `json:"project_id"`
	Name       string    `json:"name"`
	SourceType string    `json:"source_type"`
	SourceURL  string    `json:"source_url"`
	URL        string    `json:"url"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type StreamListResponse struct {
	Items      []Stream `json:"items"`
	NextCursor *string  `json:"next_cursor"`
}

type CreateStreamRequest struct {
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	SourceURL  string `json:"source_url"`
	URL        string `json:"url"`
	IsActive   *bool  `json:"is_active"`
}

type CreateCompanyStreamRequest struct {
	ProjectID  int64  `json:"project_id"`
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	SourceURL  string `json:"source_url"`
	URL        string `json:"url"`
	IsActive   *bool  `json:"is_active"`
}

type PatchStreamRequest struct {
	Name       *string `json:"name"`
	SourceType *string `json:"source_type"`
	SourceURL  *string `json:"source_url"`
	URL        *string `json:"url"`
	IsActive   *bool   `json:"is_active"`
}

type EmbedWhitelistItem struct {
	ID              int64     `json:"id"`
	CompanyID       int64     `json:"company_id"`
	Domain          string    `json:"domain"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedByUserID *int64    `json:"created_by_user_id,omitempty"`
}

type EmbedWhitelistListResponse struct {
	Items []EmbedWhitelistItem `json:"items"`
}

type CreateEmbedWhitelistRequest struct {
	Domain string `json:"domain"`
}

type PatchEmbedWhitelistRequest struct {
	Enabled *bool `json:"enabled"`
}

type CheckJob struct {
	ID           int64      `json:"id"`
	CompanyID    int64      `json:"company_id"`
	StreamID     int64      `json:"stream_id"`
	PlannedAt    time.Time  `json:"planned_at"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	ErrorMessage *string    `json:"error_message"`
}

type CheckJobListResponse struct {
	Items      []CheckJob `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

type EnqueueCheckJobRequest struct {
	PlannedAt string `json:"planned_at"`
}

type EnqueueCheckJobResponse struct {
	Job CheckJob `json:"job"`
}

type CheckResult struct {
	ID             int64           `json:"id"`
	CompanyID      int64           `json:"company_id"`
	JobID          int64           `json:"job_id"`
	StreamID       int64           `json:"stream_id"`
	Status         string          `json:"status"`
	Checks         json.RawMessage `json:"checks"`
	ScreenshotPath *string         `json:"screenshot_path"`
	CreatedAt      time.Time       `json:"created_at"`
}

type CheckResultListResponse struct {
	Items      []CheckResult `json:"items"`
	NextCursor *string       `json:"next_cursor"`
}

// AIIncidentResponse is the read-only API response for AI incident analysis (cause/summary) per check job.
type AIIncidentResponse struct {
	Cause   string `json:"cause"`
	Summary string `json:"summary"`
}

type TelegramDeliverySettings struct {
	IsEnabled     bool      `json:"is_enabled"`
	ChatID        string    `json:"chat_id"`
	SendRecovered bool      `json:"send_recovered"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PatchTelegramDeliverySettingsRequest struct {
	IsEnabled     *bool   `json:"is_enabled"`
	ChatID        *string `json:"chat_id"`
	SendRecovered *bool   `json:"send_recovered"`
}

type ErrorEnvelope struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details"`
	RequestID string      `json:"request_id"`
}

// StreamFavorite is a user's favorite or pinned stream (API/domain).
type StreamFavorite struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	StreamID  int64     `json:"stream_id"`
	IsPinned  bool      `json:"is_pinned"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// StreamWithFavorite is a stream with favorite/pin metadata for list.
type StreamWithFavorite struct {
	Stream    Stream `json:"stream"`
	IsPinned  bool   `json:"is_pinned"`
	SortOrder int    `json:"sort_order"`
}

// Incident is a monitoring incident (open or resolved) for a stream.
type Incident struct {
	ID                   int64           `json:"id"`
	CompanyID            int64           `json:"company_id"`
	StreamID             int64           `json:"stream_id"`
	StreamName           string          `json:"stream_name,omitempty"`
	Status               string          `json:"status"`
	Severity             string          `json:"severity"`
	StartedAt            time.Time       `json:"started_at"`
	LastEventAt          time.Time       `json:"last_event_at"`
	ResolvedAt           *time.Time      `json:"resolved_at,omitempty"`
	FailReason           *string         `json:"fail_reason,omitempty"`
	SampleScreenshotPath *string         `json:"sample_screenshot_path,omitempty"`
	HasScreenshot        bool            `json:"has_screenshot"`
	ScreenshotTakenAt    *time.Time      `json:"screenshot_taken_at,omitempty"`
	DiagCode             *string         `json:"diag_code,omitempty"`
	DiagDetails          json.RawMessage `json:"diag_details,omitempty"`
	LastCheckID          *int64          `json:"last_check_id,omitempty"`
}

// IncidentListResponse for paginated incidents.
type IncidentListResponse struct {
	Items      []Incident `json:"items"`
	NextCursor *string    `json:"next_cursor"`
	Total      int64      `json:"total,omitempty"`
}

const (
	AuditActorTypeAPI                    = "api"
	AuditActorIDSystem                   = "system"
	AuditActorTypeWorker                 = "worker"
	AuditEntityTypeCompany               = "company"
	AuditEntityTypeProject               = "project"
	AuditEntityTypeStream                = "stream"
	AuditEntityTypeEmbedWhitelist        = "embed_whitelist"
	AuditEntityTypeIncident              = "incident"
	AuditActionCompanyCreate             = "create"
	AuditActionCompanyUpdate             = "update"
	AuditActionCompanyDelete             = "delete"
	AuditActionProjectCreate             = "create"
	AuditActionProjectUpdate             = "update"
	AuditActionProjectDelete             = "delete"
	AuditActionStreamCreate              = "create"
	AuditActionStreamUpdate              = "update"
	AuditActionStreamDelete              = "delete"
	AuditActionEmbedWhitelistAdd         = "add"
	AuditActionEmbedWhitelistRemove      = "remove"
	AuditActionEmbedWhitelistToggle      = "toggle"
	AuditActionIncidentOpen              = "open"
	AuditActionIncidentResolve           = "resolve"
	AuditActionIncidentDiagnosticUpdated = "incident_diagnostic_updated"
)
