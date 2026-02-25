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
type streamLatestStatusListResponse = domain.StreamLatestStatusListResponse
type createStreamRequest = domain.CreateStreamRequest
type createCompanyStreamRequest = domain.CreateCompanyStreamRequest
type patchStreamRequest = domain.PatchStreamRequest
type checkJob = domain.CheckJob
type checkJobListResponse = domain.CheckJobListResponse
type enqueueCheckJobRequest = domain.EnqueueCheckJobRequest
type enqueueCheckJobResponse = domain.EnqueueCheckJobResponse
type checkResult = domain.CheckResult
type checkResultListResponse = domain.CheckResultListResponse
type aiIncidentResponse = domain.AIIncidentResponse
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
type changeUserStatusRequest = domain.ChangeUserStatusRequest
type adminUserListResponse = domain.AdminUserListResponse

type Server struct {
	db                      *sql.DB
	companyService          *serviceapi.CompanyService
	projectService          *serviceapi.ProjectService
	streamService           *serviceapi.StreamService
	checkJobService         *serviceapi.CheckJobService
	checkResultService      *serviceapi.CheckResultService
	aiIncidentService       *serviceapi.AIIncidentService
	streamFavoriteService   *serviceapi.StreamFavoriteService
	incidentService         *serviceapi.IncidentService
	telegramSettingsService *serviceapi.TelegramSettingsService
	embedWhitelistService   *serviceapi.EmbedWhitelistService
	authService             *serviceapi.AuthService
	authAccessTTL           time.Duration
	authRefreshTTL          time.Duration
	registrationService     *serviceapi.RegistrationService
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

	authAccessTTL := time.Duration(config.IntAtLeast(config.GetInt("AUTH_ACCESS_TTL_MIN", 15), 1)) * time.Minute
	authRefreshTTL := time.Duration(config.IntAtLeast(config.GetInt("AUTH_REFRESH_TTL_DAYS", 30), 1)) * 24 * time.Hour
	return &Server{
		db:                      db,
		companyService:          serviceapi.NewCompanyService(postgres.NewAPICompanyRepo(db)),
		projectService:          serviceapi.NewProjectService(postgres.NewAPIProjectRepo(db)),
		streamService:           serviceapi.NewStreamService(postgres.NewAPIStreamRepo(db)),
		checkJobService:         serviceapi.NewCheckJobService(postgres.NewAPICheckJobRepo(db)),
		checkResultService:      serviceapi.NewCheckResultService(postgres.NewAPICheckResultRepo(db)),
		aiIncidentService:       serviceapi.NewAIIncidentService(postgres.NewAPIAIIncidentRepo(db)),
		streamFavoriteService:   serviceapi.NewStreamFavoriteService(postgres.NewAPIStreamFavoriteRepo(db)),
		incidentService:         serviceapi.NewIncidentService(postgres.NewAPIIncidentRepo(db)),
		telegramSettingsService: serviceapi.NewTelegramSettingsService(postgres.NewAPITelegramSettingsRepo(db)),
		embedWhitelistService:   serviceapi.NewEmbedWhitelistService(postgres.NewAPIEmbedWhitelistRepo(db)),
		authService: serviceapi.NewAuthService(authRepo, serviceapi.AuthConfig{
			AccessTTL:          authAccessTTL,
			RefreshTTL:         authRefreshTTL,
			TelegramBotToken:   config.GetString("TELEGRAM_BOT_TOKEN_DEFAULT", ""),
			TelegramAuthMaxAge: time.Duration(config.GetInt("AUTH_TELEGRAM_MAX_AGE_SEC", 600)) * time.Second,
		}),
		authAccessTTL:       authAccessTTL,
		authRefreshTTL:      authRefreshTTL,
		registrationService: serviceapi.NewRegistrationService(registrationRepo, registrationNotifier),
	}
}

func (s *Server) RouterHandlers() RouterHandlers {
	return RouterHandlers{
		WrapWithAuth: s.authMiddleware,

		HandleHealth:                        s.handleHealth,
		HandleReady:                         s.handleReady,
		HandleCreateCompany:                 s.handleCreateCompany,
		HandleListCompanies:                 s.handleListCompanies,
		HandleGetCompany:                    s.handleGetCompany,
		HandlePatchCompany:                  s.handlePatchCompany,
		HandleDeleteCompany:                 s.handleDeleteCompany,
		HandleCreateProject:                 s.handleCreateProject,
		HandleListProjects:                  s.handleListProjects,
		HandleGetProject:                    s.handleGetProject,
		HandlePatchProject:                  s.handlePatchProject,
		HandleDeleteProject:                 s.handleDeleteProject,
		HandleCreateStream:                  s.handleCreateStream,
		HandleCreateStreamInCompany:         s.handleCreateStreamInCompany,
		HandleListStreams:                   s.handleListStreams,
		HandleListStreamLatestStatuses:      s.handleListStreamLatestStatuses,
		HandleGetStream:                     s.handleGetStream,
		HandlePatchStream:                   s.handlePatchStream,
		HandleDeleteStream:                  s.handleDeleteStream,
		HandleEnqueueCheckJob:               s.handleEnqueueCheckJob,
		HandleTriggerStreamCheck:            s.handleTriggerStreamCheck,
		HandleGetCheckJob:                   s.handleGetCheckJob,
		HandleListCheckJobs:                 s.handleListCheckJobs,
		HandleGetCheckResult:                s.handleGetCheckResult,
		HandleListCheckResults:              s.handleListCheckResults,
		HandleGetCheckResultByJob:           s.handleGetCheckResultByJob,
		HandleGetAIIncident:                 s.handleGetAIIncident,
		HandleListStreamFavorites:           s.handleListStreamFavorites,
		HandleAddStreamFavorite:             s.handleAddStreamFavorite,
		HandleRemoveStreamFavorite:          s.handleRemoveStreamFavorite,
		HandleAddStreamPin:                  s.handleAddStreamPin,
		HandleRemoveStreamPin:               s.handleRemoveStreamPin,
		HandleListIncidents:                 s.handleListIncidents,
		HandleGetIncident:                   s.handleGetIncident,
		HandleGetIncidentScreenshot:         s.handleGetIncidentScreenshot,
		HandleGetTelegramDeliverySettings:   s.handleGetTelegramDeliverySettings,
		HandlePatchTelegramDeliverySettings: s.handlePatchTelegramDeliverySettings,
		HandleListEmbedWhitelist:            s.handleListEmbedWhitelist,
		HandleCreateEmbedWhitelist:          s.handleCreateEmbedWhitelist,
		HandlePatchEmbedWhitelist:           s.handlePatchEmbedWhitelist,
		HandleDeleteEmbedWhitelist:          s.handleDeleteEmbedWhitelist,

		HandleRegisterRequest:            s.handleRegisterRequest,
		HandleLogin:                      s.handleLogin,
		HandleRefresh:                    s.handleRefresh,
		HandleLogout:                     s.handleLogout,
		HandleMe:                         s.handleMe,
		HandleTelegramLogin:              s.handleTelegramLogin,
		HandleTelegramLink:               s.handleTelegramLink,
		HandleListPendingRegistration:    s.handleListPendingRegistrationRequests,
		HandleListUsers:                  s.handleListUsers,
		HandleApproveRegistrationRequest: s.handleApproveRegistrationRequest,
		HandleRejectRegistrationRequest:  s.handleRejectRegistrationRequest,
		HandleChangeUserRole:             s.handleChangeUserRole,
		HandleChangeUserStatus:           s.handleChangeUserStatus,
	}
}
