package domain

import (
	"errors"
	"time"
)

var (
	ErrCompanyAlreadyExists = errors.New("company_already_exists")
	ErrCompanyNotFound      = errors.New("company_not_found")

	ErrProjectAlreadyExists  = errors.New("project_already_exists")
	ErrProjectNotFound       = errors.New("project_not_found")
	ErrProjectCompanyMissing = errors.New("project_company_missing")

	ErrStreamAlreadyExists = errors.New("stream_already_exists")
	ErrStreamNotFound      = errors.New("stream_not_found")
	ErrStreamProjectMiss   = errors.New("stream_project_missing")

	ErrCheckJobNotFound      = errors.New("check_job_not_found")
	ErrCheckJobStreamMissing = errors.New("check_job_stream_missing")
	ErrCheckJobConflict      = errors.New("check_job_conflict")

	ErrCheckResultNotFound      = errors.New("check_result_not_found")
	ErrCheckResultByJobNotFound = errors.New("check_result_by_job_not_found")
	ErrCheckResultStreamMissing = errors.New("check_result_stream_missing")

	ErrUserNotFound            = errors.New("user_not_found")
	ErrUserAlreadyExists       = errors.New("user_already_exists")
	ErrUserDisabled            = errors.New("user_disabled")
	ErrUserScopeNotSupported   = errors.New("user_scope_not_supported")
	ErrInvalidCredentials      = errors.New("invalid_credentials")
	ErrSessionNotFound         = errors.New("session_not_found")
	ErrSessionExpired          = errors.New("session_expired")
	ErrSessionRevoked          = errors.New("session_revoked")
	ErrRegistrationConflict    = errors.New("registration_conflict")
	ErrRegistrationNotFound    = errors.New("registration_not_found")
	ErrRegistrationNotPending  = errors.New("registration_not_pending")
	ErrPendingRegistrationOnly = errors.New("pending_registration_only")
	ErrTelegramLinkNotFound    = errors.New("telegram_link_not_found")
	ErrTelegramLinkConflict    = errors.New("telegram_link_conflict")

	ErrTelegramDeliverySettingsNotFound    = errors.New("telegram_delivery_settings_not_found")
	ErrTelegramDeliverySettingsInvalidInput = errors.New("telegram_delivery_settings_invalid_input")

	ErrAIIncidentNotFound = errors.New("ai_incident_not_found")
)

type StreamListFilter struct {
	ProjectID *int64
	IsActive  *bool
}

type StreamPatchInput struct {
	Name     *string
	URL      *string
	IsActive *bool
}

type CheckJobListFilter struct {
	Status *string
	From   *time.Time
	To     *time.Time
}

type CheckResultListFilter struct {
	Status *string
	From   *time.Time
	To     *time.Time
}
