package api

import (
	"database/sql"

	"github.com/example/hls-monitoring-platform/internal/domain"
	"github.com/example/hls-monitoring-platform/internal/repo/postgres"
	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
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
	companyService     *serviceapi.CompanyService
	projectService     *serviceapi.ProjectService
	streamService      *serviceapi.StreamService
	checkJobService    *serviceapi.CheckJobService
	checkResultService *serviceapi.CheckResultService
}

func NewServer(db *sql.DB) *Server {
	return &Server{
		companyService:     serviceapi.NewCompanyService(postgres.NewAPICompanyRepo(db)),
		projectService:     serviceapi.NewProjectService(postgres.NewAPIProjectRepo(db)),
		streamService:      serviceapi.NewStreamService(postgres.NewAPIStreamRepo(db)),
		checkJobService:    serviceapi.NewCheckJobService(postgres.NewAPICheckJobRepo(db)),
		checkResultService: serviceapi.NewCheckResultService(postgres.NewAPICheckResultRepo(db)),
	}
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
