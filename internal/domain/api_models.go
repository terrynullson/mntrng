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
	ID        int64     `json:"id"`
	CompanyID int64     `json:"company_id"`
	ProjectID int64     `json:"project_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StreamListResponse struct {
	Items      []Stream `json:"items"`
	NextCursor *string  `json:"next_cursor"`
}

type CreateStreamRequest struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	IsActive *bool  `json:"is_active"`
}

type PatchStreamRequest struct {
	Name     *string `json:"name"`
	URL      *string `json:"url"`
	IsActive *bool   `json:"is_active"`
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

const (
	AuditActorTypeAPI        = "api"
	AuditActorIDSystem       = "system"
	AuditEntityTypeCompany   = "company"
	AuditEntityTypeProject   = "project"
	AuditEntityTypeStream    = "stream"
	AuditActionCompanyCreate = "create"
	AuditActionCompanyUpdate = "update"
	AuditActionCompanyDelete = "delete"
	AuditActionProjectCreate = "create"
	AuditActionProjectUpdate = "update"
	AuditActionProjectDelete = "delete"
	AuditActionStreamCreate  = "create"
	AuditActionStreamUpdate  = "update"
	AuditActionStreamDelete  = "delete"
)
