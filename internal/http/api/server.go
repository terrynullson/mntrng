package api

import (
	"context"
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
type streamWithLatestStatusListResponse = domain.StreamWithLatestStatusListResponse
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

type CompanyPort interface {
	CreateCompany(ctx context.Context, nameRaw string) (domain.Company, error)
	ListCompanies(ctx context.Context) ([]domain.Company, error)
	GetCompany(ctx context.Context, companyID int64) (domain.Company, error)
	PatchCompany(ctx context.Context, companyID int64, nameRaw string) (domain.Company, error)
	DeleteCompany(ctx context.Context, companyID int64) error
}

type ProjectPort interface {
	CreateProject(ctx context.Context, companyID int64, nameRaw string) (domain.Project, error)
	ListProjects(ctx context.Context, companyID int64) ([]domain.Project, error)
	GetProject(ctx context.Context, companyID int64, projectID int64) (domain.Project, error)
	PatchProject(ctx context.Context, companyID int64, projectID int64, nameRaw string) (domain.Project, error)
	DeleteProject(ctx context.Context, companyID int64, projectID int64) error
}

type StreamPort interface {
	CreateStream(ctx context.Context, input serviceapi.CreateStreamInput) (domain.Stream, error)
	ListStreams(ctx context.Context, input serviceapi.ListStreamsInput) ([]domain.Stream, error)
	ListStreamsWithLatestStatus(ctx context.Context, input serviceapi.ListStreamsInput) ([]domain.StreamWithLatestStatus, error)
	ListLatestStatuses(ctx context.Context, companyID int64) ([]domain.StreamLatestStatus, error)
	GetStream(ctx context.Context, companyID int64, streamID int64) (domain.Stream, error)
	PatchStream(ctx context.Context, input serviceapi.PatchStreamRequest) (domain.Stream, error)
	DeleteStream(ctx context.Context, companyID int64, streamID int64) error
}

type CheckJobPort interface {
	EnqueueCheckJob(ctx context.Context, input serviceapi.EnqueueCheckJobInput) (domain.CheckJob, error)
	GetCheckJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckJob, error)
	ListCheckJobs(ctx context.Context, input serviceapi.ListCheckJobsInput) ([]domain.CheckJob, error)
}

type CheckResultPort interface {
	GetCheckResult(ctx context.Context, companyID int64, resultID int64) (domain.CheckResult, error)
	ListCheckResults(ctx context.Context, input serviceapi.ListCheckResultsInput) ([]domain.CheckResult, error)
	GetCheckResultByJob(ctx context.Context, companyID int64, jobID int64) (domain.CheckResult, error)
}

type AIIncidentPort interface {
	Get(ctx context.Context, companyID int64, streamID int64, jobID int64) (domain.AIIncidentResponse, error)
}

type StreamFavoritePort interface {
	ListFavorites(ctx context.Context, userID int64, companyID int64) ([]domain.StreamWithFavorite, error)
	AddFavorite(ctx context.Context, userID int64, companyID int64, streamID int64) error
	RemoveFavorite(ctx context.Context, userID int64, companyID int64, streamID int64) error
	AddPin(ctx context.Context, userID int64, companyID int64, streamID int64, sortOrder *int) error
	RemovePin(ctx context.Context, userID int64, companyID int64, streamID int64) error
}

type IncidentPort interface {
	List(ctx context.Context, input serviceapi.ListIncidentsInput) ([]domain.Incident, int64, *string, error)
	Get(ctx context.Context, companyID int64, incidentID int64) (domain.Incident, error)
}

type TelegramSettingsPort interface {
	GetTelegramDeliverySettings(ctx context.Context, companyID int64) (domain.TelegramDeliverySettings, error)
	PatchTelegramDeliverySettings(ctx context.Context, companyID int64, request domain.PatchTelegramDeliverySettingsRequest) (domain.TelegramDeliverySettings, error)
}

type EmbedWhitelistPort interface {
	List(ctx context.Context, companyID int64) ([]domain.EmbedWhitelistItem, error)
	Create(ctx context.Context, companyID int64, domainValue string, actorUserID int64) (domain.EmbedWhitelistItem, error)
	Patch(ctx context.Context, companyID int64, itemID int64, enabled *bool, actorUserID int64) (domain.EmbedWhitelistItem, error)
	Delete(ctx context.Context, companyID int64, itemID int64, actorUserID int64) error
}

type AuthPort interface {
	Login(ctx context.Context, request domain.LoginRequest) (domain.AuthTokensResponse, error)
	Refresh(ctx context.Context, request domain.RefreshRequest) (domain.AuthTokensResponse, error)
	Logout(ctx context.Context, authContext domain.AuthContext, request domain.LogoutRequest) error
	Me(ctx context.Context, userID int64) (domain.AuthUser, error)
	AuthenticateAccessToken(ctx context.Context, accessToken string) (domain.AuthContext, error)
	LinkTelegram(ctx context.Context, userID int64, payload map[string]string) error
	TelegramLogin(ctx context.Context, payload map[string]string) (domain.AuthTokensResponse, error)
}

type RegistrationPort interface {
	SubmitRegistrationRequest(ctx context.Context, request domain.RegistrationRequestCreate) (domain.RegistrationRequest, error)
	ListPendingRegistrationRequests(ctx context.Context) ([]domain.RegistrationRequest, error)
	ListUsers(ctx context.Context, input serviceapi.ListAdminUsersInput) ([]domain.AuthUser, error)
	ApproveRegistrationRequest(ctx context.Context, requestID int64, request domain.ApproveRegistrationRequest, actorUserID int64) (domain.AuthUser, error)
	RejectRegistrationRequest(ctx context.Context, requestID int64, request domain.RejectRegistrationRequest, actorUserID int64) error
	ChangeUserRole(ctx context.Context, userID int64, request domain.ChangeUserRoleRequest, actorUserID int64) (domain.AuthUser, error)
	ChangeUserStatus(ctx context.Context, userID int64, request domain.ChangeUserStatusRequest, actorUserID int64) (domain.AuthUser, error)
}

type Ports struct {
	Company          CompanyPort
	Project          ProjectPort
	Stream           StreamPort
	CheckJob         CheckJobPort
	CheckResult      CheckResultPort
	AIIncident       AIIncidentPort
	StreamFavorite   StreamFavoritePort
	Incident         IncidentPort
	TelegramSettings TelegramSettingsPort
	EmbedWhitelist   EmbedWhitelistPort
	Auth             AuthPort
	Registration     RegistrationPort
}

type AuthTTLConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type Server struct {
	db                      *sql.DB
	companyService          CompanyPort
	projectService          ProjectPort
	streamService           StreamPort
	checkJobService         CheckJobPort
	checkResultService      CheckResultPort
	aiIncidentService       AIIncidentPort
	streamFavoriteService   StreamFavoritePort
	incidentService         IncidentPort
	telegramSettingsService TelegramSettingsPort
	embedWhitelistService   EmbedWhitelistPort
	authService             AuthPort
	authAccessTTL           time.Duration
	authRefreshTTL          time.Duration
	registrationService     RegistrationPort
}

func NewServer(db *sql.DB, ports Ports, authTTL AuthTTLConfig) *Server {
	return &Server{
		db:                      db,
		companyService:          ports.Company,
		projectService:          ports.Project,
		streamService:           ports.Stream,
		checkJobService:         ports.CheckJob,
		checkResultService:      ports.CheckResult,
		aiIncidentService:       ports.AIIncident,
		streamFavoriteService:   ports.StreamFavorite,
		incidentService:         ports.Incident,
		telegramSettingsService: ports.TelegramSettings,
		embedWhitelistService:   ports.EmbedWhitelist,
		authService:             ports.Auth,
		authAccessTTL:           authTTL.AccessTTL,
		authRefreshTTL:          authTTL.RefreshTTL,
		registrationService:     ports.Registration,
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
		HandleListStreamsWithLatestStatus:   s.handleListStreamsWithLatestStatus,
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
