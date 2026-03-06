package api

import (
	"database/sql"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
	serviceapi "github.com/terrynullson/mntrng/internal/service/api"
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

type ServiceSet struct {
	CompanyService          *serviceapi.CompanyService
	ProjectService          *serviceapi.ProjectService
	StreamService           *serviceapi.StreamService
	CheckJobService         *serviceapi.CheckJobService
	CheckResultService      *serviceapi.CheckResultService
	AIIncidentService       *serviceapi.AIIncidentService
	StreamFavoriteService   *serviceapi.StreamFavoriteService
	IncidentService         *serviceapi.IncidentService
	TelegramSettingsService *serviceapi.TelegramSettingsService
	EmbedWhitelistService   *serviceapi.EmbedWhitelistService
	AuthService             *serviceapi.AuthService
	RegistrationService     *serviceapi.RegistrationService
}

type AuthTTLConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

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

func NewServer(db *sql.DB, services ServiceSet, authTTL AuthTTLConfig) *Server {
	return &Server{
		db:                      db,
		companyService:          services.CompanyService,
		projectService:          services.ProjectService,
		streamService:           services.StreamService,
		checkJobService:         services.CheckJobService,
		checkResultService:      services.CheckResultService,
		aiIncidentService:       services.AIIncidentService,
		streamFavoriteService:   services.StreamFavoriteService,
		incidentService:         services.IncidentService,
		telegramSettingsService: services.TelegramSettingsService,
		embedWhitelistService:   services.EmbedWhitelistService,
		authService:             services.AuthService,
		authAccessTTL:           authTTL.AccessTTL,
		authRefreshTTL:          authTTL.RefreshTTL,
		registrationService:     services.RegistrationService,
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
