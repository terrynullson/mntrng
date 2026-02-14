package api

import (
	"database/sql"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

type healthResponse = domain.HealthResponse
type company = domain.Company
type companyListResponse = domain.CompanyListResponse
type createCompanyRequest = domain.CreateCompanyRequest
type patchCompanyRequest = domain.PatchCompanyRequest
type project = domain.Project
type projectListResponse = domain.ProjectListResponse
type createProjectRequest = domain.CreateProjectRequest
type patchProjectRequest = domain.PatchProjectRequest
type stream = domain.Stream
type streamListResponse = domain.StreamListResponse
type createStreamRequest = domain.CreateStreamRequest
type patchStreamRequest = domain.PatchStreamRequest
type checkJob = domain.CheckJob
type checkJobListResponse = domain.CheckJobListResponse
type enqueueCheckJobRequest = domain.EnqueueCheckJobRequest
type enqueueCheckJobResponse = domain.EnqueueCheckJobResponse
type checkResult = domain.CheckResult
type checkResultListResponse = domain.CheckResultListResponse

type Server struct {
	db *sql.DB
}

func NewServer(db *sql.DB) *Server {
	return &Server{db: db}
}

func (s *Server) RouterHandlers() RouterHandlers {
	return RouterHandlers{
		HandleHealth:              s.handleHealth,
		HandleCreateCompany:       s.handleCreateCompany,
		HandleListCompanies:       s.handleListCompanies,
		HandleGetCompany:          s.handleGetCompany,
		HandlePatchCompany:        s.handlePatchCompany,
		HandleDeleteCompany:       s.handleDeleteCompany,
		HandleCreateProject:       s.handleCreateProject,
		HandleListProjects:        s.handleListProjects,
		HandleGetProject:          s.handleGetProject,
		HandlePatchProject:        s.handlePatchProject,
		HandleDeleteProject:       s.handleDeleteProject,
		HandleCreateStream:        s.handleCreateStream,
		HandleListStreams:         s.handleListStreams,
		HandleGetStream:           s.handleGetStream,
		HandlePatchStream:         s.handlePatchStream,
		HandleDeleteStream:        s.handleDeleteStream,
		HandleEnqueueCheckJob:     s.handleEnqueueCheckJob,
		HandleGetCheckJob:         s.handleGetCheckJob,
		HandleListCheckJobs:       s.handleListCheckJobs,
		HandleGetCheckResult:      s.handleGetCheckResult,
		HandleListCheckResults:    s.handleListCheckResults,
		HandleGetCheckResultByJob: s.handleGetCheckResultByJob,
	}
}
