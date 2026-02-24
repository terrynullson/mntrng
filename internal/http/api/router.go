package api

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	projectCollectionPath        = "projects"
	projectItemPrefix            = "projects/"
	streamCollectionPath         = "streams"
	streamItemPrefix             = "streams/"
	checkJobsCollectionPath      = "check-jobs"
	checkJobsItemPrefix          = "check-jobs/"
	checkResultsCollectionPath   = "check-results"
	checkResultsItemPrefix       = "check-results/"
	telegramDeliverySettingsPath = "telegram-delivery-settings"
	aiIncidentPathSuffix         = "ai-incident"
)

type companyHandler func(http.ResponseWriter, *http.Request, int64)
type companyResourceHandler func(http.ResponseWriter, *http.Request, int64, int64)
type streamCompanyResourceHandler func(http.ResponseWriter, *http.Request, int64, int64, int64)
type adminResourceHandler func(http.ResponseWriter, *http.Request, int64)

type RouterHandlers struct {
	WrapWithAuth func(http.Handler) http.Handler

	HandleHealth http.HandlerFunc
	HandleReady  http.HandlerFunc

	HandleCreateCompany http.HandlerFunc
	HandleListCompanies http.HandlerFunc
	HandleGetCompany    companyHandler
	HandlePatchCompany  companyHandler
	HandleDeleteCompany companyHandler

	HandleCreateProject companyHandler
	HandleListProjects  companyHandler
	HandleGetProject    companyResourceHandler
	HandlePatchProject  companyResourceHandler
	HandleDeleteProject companyResourceHandler

	HandleCreateStream          companyResourceHandler
	HandleCreateStreamInCompany companyHandler
	HandleListStreams           companyHandler
	HandleGetStream             companyResourceHandler
	HandlePatchStream           companyResourceHandler
	HandleDeleteStream          companyResourceHandler

	HandleEnqueueCheckJob     companyResourceHandler
	HandleTriggerStreamCheck  companyResourceHandler
	HandleGetCheckJob         companyResourceHandler
	HandleListCheckJobs       companyResourceHandler
	HandleGetCheckResult      companyResourceHandler
	HandleListCheckResults    companyResourceHandler
	HandleGetCheckResultByJob companyResourceHandler
	HandleGetAIIncident       streamCompanyResourceHandler

	HandleListStreamFavorites  companyHandler
	HandleAddStreamFavorite    companyResourceHandler
	HandleRemoveStreamFavorite companyResourceHandler
	HandleAddStreamPin         companyResourceHandler
	HandleRemoveStreamPin      companyResourceHandler
	HandleListIncidents        companyHandler
	HandleGetIncident          companyResourceHandler

	HandleGetTelegramDeliverySettings   companyHandler
	HandlePatchTelegramDeliverySettings companyHandler

	HandleRegisterRequest            http.HandlerFunc
	HandleLogin                      http.HandlerFunc
	HandleRefresh                    http.HandlerFunc
	HandleLogout                     http.HandlerFunc
	HandleMe                         http.HandlerFunc
	HandleTelegramLogin              http.HandlerFunc
	HandleTelegramLink               http.HandlerFunc
	HandleListPendingRegistration    http.HandlerFunc
	HandleListUsers                  http.HandlerFunc
	HandleApproveRegistrationRequest adminResourceHandler
	HandleRejectRegistrationRequest  adminResourceHandler
	HandleChangeUserRole             adminResourceHandler
	HandleChangeUserStatus           adminResourceHandler
}

func NewRouter(handlers RouterHandlers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", handlers.HandleHealth)
	mux.HandleFunc("/api/v1/ready", handlers.HandleReady)
	mux.Handle("/api/v1/metrics", promhttp.Handler())
	mux.HandleFunc("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		routeAuthRegister(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		routeAuthLogin(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		routeAuthRefresh(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		routeAuthLogout(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		routeAuthMe(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/auth/telegram/login", func(w http.ResponseWriter, r *http.Request) {
		routeAuthTelegramLogin(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/auth/telegram/link", func(w http.ResponseWriter, r *http.Request) {
		routeAuthTelegramLink(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/admin/registration-requests", func(w http.ResponseWriter, r *http.Request) {
		routeAdminRegistrationRequestsCollection(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/admin/registration-requests/", func(w http.ResponseWriter, r *http.Request) {
		routeAdminRegistrationRequests(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/admin/users", func(w http.ResponseWriter, r *http.Request) {
		routeAdminUsersCollection(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/admin/users/", func(w http.ResponseWriter, r *http.Request) {
		routeAdminUsers(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/companies", func(w http.ResponseWriter, r *http.Request) {
		routeCompanies(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/companies/", func(w http.ResponseWriter, r *http.Request) {
		routeCompanyByID(w, r, handlers)
	})
	if handlers.WrapWithAuth != nil {
		return handlers.WrapWithAuth(mux)
	}
	return mux
}

func routeCompanies(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	switch r.Method {
	case http.MethodPost:
		handlers.HandleCreateCompany(w, r)
	case http.MethodGet:
		handlers.HandleListCompanies(w, r)
	default:
		WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

func routeCompanyByID(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	companyID, pathRemainder, pathErr := parseCompanyPath(r.URL.Path)
	if pathErr == "not_found" {
		writeRouterPathNotFound(w, r)
		return
	}
	if pathErr == "validation_error" {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid company_id", map[string]interface{}{"path": r.URL.Path})
		return
	}

	if pathRemainder == "" {
		routeCompanyRoot(w, r, handlers, companyID)
		return
	}
	if pathRemainder == telegramDeliverySettingsPath {
		routeTelegramDeliverySettings(w, r, handlers, companyID)
		return
	}
	if pathRemainder == "incidents" {
		if r.Method == http.MethodGet {
			handlers.HandleListIncidents(w, r, companyID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return
	}
	if strings.HasPrefix(pathRemainder, "incidents/") {
		incidentPath := strings.TrimPrefix(pathRemainder, "incidents/")
		if incidentPath == "" || strings.Contains(incidentPath, "/") {
			writeRouterPathNotFound(w, r)
			return
		}
		incidentID, err := parsePositiveID(incidentPath)
		if err != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid incident_id", map[string]interface{}{"path": r.URL.Path})
			return
		}
		if r.Method == http.MethodGet {
			handlers.HandleGetIncident(w, r, companyID, incidentID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return
	}
	if pathRemainder == "streams/favorites" {
		if r.Method == http.MethodGet {
			handlers.HandleListStreamFavorites(w, r, companyID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return
	}
	if routeCompanyProjectPath(w, r, handlers, companyID, pathRemainder) {
		return
	}
	if routeCompanyStreamCollection(w, r, handlers, companyID, pathRemainder) {
		return
	}
	if routeCompanyCheckJobsPath(w, r, handlers, companyID, pathRemainder) {
		return
	}
	if routeCompanyCheckResultsPath(w, r, handlers, companyID, pathRemainder) {
		return
	}
	if routeCompanyStreamPath(w, r, handlers, companyID, pathRemainder) {
		return
	}

	writeRouterPathNotFound(w, r)
}

func routeTelegramDeliverySettings(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64) {
	switch r.Method {
	case http.MethodGet:
		handlers.HandleGetTelegramDeliverySettings(w, r, companyID)
	case http.MethodPatch:
		handlers.HandlePatchTelegramDeliverySettings(w, r, companyID)
	default:
		WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPatch)
	}
}

func routeCompanyRoot(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64) {
	switch r.Method {
	case http.MethodGet:
		handlers.HandleGetCompany(w, r, companyID)
	case http.MethodPatch:
		handlers.HandlePatchCompany(w, r, companyID)
	case http.MethodDelete:
		handlers.HandleDeleteCompany(w, r, companyID)
	default:
		WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)
	}
}

func routeCompanyProjectPath(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64, pathRemainder string) bool {
	if pathRemainder == projectCollectionPath {
		switch r.Method {
		case http.MethodPost:
			handlers.HandleCreateProject(w, r, companyID)
		case http.MethodGet:
			handlers.HandleListProjects(w, r, companyID)
		default:
			WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
		}
		return true
	}
	if !strings.HasPrefix(pathRemainder, projectItemPrefix) {
		return false
	}

	projectPath := strings.TrimPrefix(pathRemainder, projectItemPrefix)
	if projectPath == "" {
		writeRouterPathNotFound(w, r)
		return true
	}
	projectParts := strings.Split(projectPath, "/")
	projectID, err := parsePositiveID(projectParts[0])
	if err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid project_id", map[string]interface{}{"path": r.URL.Path})
		return true
	}
	if len(projectParts) == 1 {
		routeProjectItem(w, r, handlers, companyID, projectID)
		return true
	}
	if len(projectParts) == 2 && projectParts[1] == streamCollectionPath {
		if r.Method == http.MethodPost {
			handlers.HandleCreateStream(w, r, companyID, projectID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodPost)
		}
		return true
	}

	writeRouterPathNotFound(w, r)
	return true
}

func routeProjectItem(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64, projectID int64) {
	switch r.Method {
	case http.MethodGet:
		handlers.HandleGetProject(w, r, companyID, projectID)
	case http.MethodPatch:
		handlers.HandlePatchProject(w, r, companyID, projectID)
	case http.MethodDelete:
		handlers.HandleDeleteProject(w, r, companyID, projectID)
	default:
		WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)
	}
}

func routeCompanyStreamCollection(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64, pathRemainder string) bool {
	if pathRemainder != streamCollectionPath {
		return false
	}
	if r.Method == http.MethodGet {
		handlers.HandleListStreams(w, r, companyID)
	} else if r.Method == http.MethodPost {
		handlers.HandleCreateStreamInCompany(w, r, companyID)
	} else {
		WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
	return true
}

func routeCompanyCheckJobsPath(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64, pathRemainder string) bool {
	if !strings.HasPrefix(pathRemainder, checkJobsItemPrefix) {
		return false
	}
	jobPath := strings.TrimPrefix(pathRemainder, checkJobsItemPrefix)
	if jobPath == "" {
		writeRouterPathNotFound(w, r)
		return true
	}

	jobParts := strings.Split(jobPath, "/")
	jobID, err := parsePositiveID(jobParts[0])
	if err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid job_id", map[string]interface{}{"path": r.URL.Path})
		return true
	}
	if len(jobParts) == 1 {
		if r.Method == http.MethodGet {
			handlers.HandleGetCheckJob(w, r, companyID, jobID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return true
	}
	if len(jobParts) == 2 && jobParts[1] == "result" {
		if r.Method == http.MethodGet {
			handlers.HandleGetCheckResultByJob(w, r, companyID, jobID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return true
	}

	writeRouterPathNotFound(w, r)
	return true
}

func routeCompanyCheckResultsPath(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64, pathRemainder string) bool {
	if !strings.HasPrefix(pathRemainder, checkResultsItemPrefix) {
		return false
	}
	resultIDRaw := strings.TrimPrefix(pathRemainder, checkResultsItemPrefix)
	if resultIDRaw == "" || strings.Contains(resultIDRaw, "/") {
		writeRouterPathNotFound(w, r)
		return true
	}

	resultID, err := parsePositiveID(resultIDRaw)
	if err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid result_id", map[string]interface{}{"path": r.URL.Path})
		return true
	}
	if r.Method == http.MethodGet {
		handlers.HandleGetCheckResult(w, r, companyID, resultID)
	} else {
		WriteMethodNotAllowed(w, r, http.MethodGet)
	}
	return true
}

func routeCompanyStreamPath(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64, pathRemainder string) bool {
	if !strings.HasPrefix(pathRemainder, streamItemPrefix) {
		return false
	}
	streamPath := strings.TrimPrefix(pathRemainder, streamItemPrefix)
	if streamPath == "" {
		writeRouterPathNotFound(w, r)
		return true
	}

	streamParts := strings.Split(streamPath, "/")
	streamID, err := parsePositiveID(streamParts[0])
	if err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid stream_id", map[string]interface{}{"path": r.URL.Path})
		return true
	}
	if len(streamParts) == 1 {
		routeStreamItem(w, r, handlers, companyID, streamID)
		return true
	}
	if len(streamParts) == 2 && streamParts[1] == "favorite" {
		switch r.Method {
		case http.MethodPost:
			handlers.HandleAddStreamFavorite(w, r, companyID, streamID)
		case http.MethodDelete:
			handlers.HandleRemoveStreamFavorite(w, r, companyID, streamID)
		default:
			WriteMethodNotAllowed(w, r, http.MethodPost, http.MethodDelete)
		}
		return true
	}
	if len(streamParts) == 2 && streamParts[1] == "pin" {
		switch r.Method {
		case http.MethodPost:
			handlers.HandleAddStreamPin(w, r, companyID, streamID)
		case http.MethodDelete:
			handlers.HandleRemoveStreamPin(w, r, companyID, streamID)
		default:
			WriteMethodNotAllowed(w, r, http.MethodPost, http.MethodDelete)
		}
		return true
	}
	if len(streamParts) == 2 && streamParts[1] == checkJobsCollectionPath {
		routeStreamCheckJobsCollection(w, r, handlers, companyID, streamID)
		return true
	}
	if len(streamParts) == 2 && streamParts[1] == "check" {
		if r.Method == http.MethodPost {
			handlers.HandleTriggerStreamCheck(w, r, companyID, streamID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodPost)
		}
		return true
	}
	if len(streamParts) == 4 && streamParts[1] == checkJobsCollectionPath && streamParts[3] == aiIncidentPathSuffix {
		jobID, err := parsePositiveID(streamParts[2])
		if err != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid job_id", map[string]interface{}{"path": r.URL.Path})
			return true
		}
		if r.Method == http.MethodGet {
			handlers.HandleGetAIIncident(w, r, companyID, streamID, jobID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return true
	}
	if len(streamParts) == 2 && streamParts[1] == checkResultsCollectionPath {
		if r.Method == http.MethodGet {
			handlers.HandleListCheckResults(w, r, companyID, streamID)
		} else {
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return true
	}

	writeRouterPathNotFound(w, r)
	return true
}

func routeStreamItem(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64, streamID int64) {
	switch r.Method {
	case http.MethodGet:
		handlers.HandleGetStream(w, r, companyID, streamID)
	case http.MethodPatch:
		handlers.HandlePatchStream(w, r, companyID, streamID)
	case http.MethodDelete:
		handlers.HandleDeleteStream(w, r, companyID, streamID)
	default:
		WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)
	}
}

func routeStreamCheckJobsCollection(w http.ResponseWriter, r *http.Request, handlers RouterHandlers, companyID int64, streamID int64) {
	switch r.Method {
	case http.MethodPost:
		handlers.HandleEnqueueCheckJob(w, r, companyID, streamID)
	case http.MethodGet:
		handlers.HandleListCheckJobs(w, r, companyID, streamID)
	default:
		WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

func writeRouterPathNotFound(w http.ResponseWriter, r *http.Request) {
	WriteJSONError(w, r, http.StatusNotFound, "not_found", "resource not found", map[string]interface{}{"path": r.URL.Path})
}
