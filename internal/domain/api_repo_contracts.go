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
