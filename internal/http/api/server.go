package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/example/hls-monitoring-platform/internal/domain"
	"github.com/example/hls-monitoring-platform/internal/repo/postgres"
	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
	"github.com/example/hls-monitoring-platform/internal/telegram"
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
type authTokensResponse = domain.AuthTokensResponse
type authUser = domain.AuthUser
type loginRequest = domain.LoginRequest
type refreshRequest = domain.RefreshRequest
type logoutRequest = domain.LogoutRequest
type registrationCreateRequest = domain.RegistrationRequestCreate
type registrationRequest = domain.RegistrationRequest
type approveRegistrationRequest = domain.ApproveRegistrationRequest
type rejectRegistrationRequest = domain.RejectRegistrationRequest
type changeUserRoleRequest = domain.ChangeUserRoleRequest

type Server struct {
	companyService      *serviceapi.CompanyService
	projectService      *serviceapi.ProjectService
	streamService       *serviceapi.StreamService
	checkJobService     *serviceapi.CheckJobService
	checkResultService  *serviceapi.CheckResultService
	authService         *serviceapi.AuthService
	registrationService *serviceapi.RegistrationService
}

func NewServer(db *sql.DB) *Server {
	authRepo := postgres.NewAPIAuthRepo(db)
	registrationRepo := postgres.NewAPIRegistrationRepo(db)

	telegramHTTPTimeoutMS := config.GetInt("TELEGRAM_HTTP_TIMEOUT_MS", 5000)
	if telegramHTTPTimeoutMS <= 0 {
		telegramHTTPTimeoutMS = 5000
	}
	telegramClient := telegram.NewClient(&http.Client{Timeout: time.Duration(telegramHTTPTimeoutMS) * time.Millisecond})
	registrationNotifier := newRegistrationNotifier(
		telegramClient,
		config.GetString("TELEGRAM_BOT_TOKEN_DEFAULT", ""),
		config.GetString("SUPER_ADMIN_TELEGRAM_CHAT_ID", ""),
	)

	return &Server{
		companyService:     serviceapi.NewCompanyService(postgres.NewAPICompanyRepo(db)),
		projectService:     serviceapi.NewProjectService(postgres.NewAPIProjectRepo(db)),
		streamService:      serviceapi.NewStreamService(postgres.NewAPIStreamRepo(db)),
		checkJobService:    serviceapi.NewCheckJobService(postgres.NewAPICheckJobRepo(db)),
		checkResultService: serviceapi.NewCheckResultService(postgres.NewAPICheckResultRepo(db)),
		authService: serviceapi.NewAuthService(authRepo, serviceapi.AuthConfig{
			AccessTTL:          time.Duration(config.GetInt("AUTH_ACCESS_TTL_MIN", 15)) * time.Minute,
			RefreshTTL:         time.Duration(config.GetInt("AUTH_REFRESH_TTL_DAYS", 30)) * 24 * time.Hour,
			TelegramBotToken:   config.GetString("TELEGRAM_BOT_TOKEN_DEFAULT", ""),
			TelegramAuthMaxAge: time.Duration(config.GetInt("AUTH_TELEGRAM_MAX_AGE_SEC", 600)) * time.Second,
		}),
		registrationService: serviceapi.NewRegistrationService(registrationRepo, registrationNotifier),
	}
}

func (s *Server) RouterHandlers() RouterHandlers {
	return RouterHandlers{
		WrapWithAuth: s.authMiddleware,

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

		HandleRegisterRequest:            s.handleRegisterRequest,
		HandleLogin:                      s.handleLogin,
		HandleRefresh:                    s.handleRefresh,
		HandleLogout:                     s.handleLogout,
		HandleMe:                         s.handleMe,
		HandleTelegramLogin:              s.handleTelegramLogin,
		HandleTelegramLink:               s.handleTelegramLink,
		HandleListPendingRegistration:    s.handleListPendingRegistrationRequests,
		HandleApproveRegistrationRequest: s.handleApproveRegistrationRequest,
		HandleRejectRegistrationRequest:  s.handleRejectRegistrationRequest,
		HandleChangeUserRole:             s.handleChangeUserRole,
	}
}
