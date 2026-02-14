package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type companyHandler func(http.ResponseWriter, *http.Request, int64)
type companyResourceHandler func(http.ResponseWriter, *http.Request, int64, int64)

type RouterHandlers struct {
	HandleHealth http.HandlerFunc

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

	HandleCreateStream companyResourceHandler
	HandleListStreams  companyHandler
	HandleGetStream    companyResourceHandler
	HandlePatchStream  companyResourceHandler
	HandleDeleteStream companyResourceHandler

	HandleEnqueueCheckJob     companyResourceHandler
	HandleGetCheckJob         companyResourceHandler
	HandleListCheckJobs       companyResourceHandler
	HandleGetCheckResult      companyResourceHandler
	HandleListCheckResults    companyResourceHandler
	HandleGetCheckResultByJob companyResourceHandler
}

func NewRouter(handlers RouterHandlers) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", handlers.HandleHealth)
	mux.HandleFunc("/api/v1/companies", func(w http.ResponseWriter, r *http.Request) {
		routeCompanies(w, r, handlers)
	})
	mux.HandleFunc("/api/v1/companies/", func(w http.ResponseWriter, r *http.Request) {
		routeCompanyByID(w, r, handlers)
	})
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
		WriteJSONError(
			w,
			r,
			http.StatusNotFound,
			"not_found",
			"resource not found",
			map[string]interface{}{"path": r.URL.Path},
		)
		return
	}
	if pathErr == "validation_error" {
		WriteJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"invalid company_id",
			map[string]interface{}{"path": r.URL.Path},
		)
		return
	}

	if pathRemainder == "" {
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
		return
	}

	const projectCollectionPath = "projects"
	const projectItemPrefix = "projects/"
	const streamCollectionPath = "streams"
	const streamItemPrefix = "streams/"
	const checkJobsCollectionPath = "check-jobs"
	const checkJobsItemPrefix = "check-jobs/"
	const checkResultsCollectionPath = "check-results"
	const checkResultsItemPrefix = "check-results/"
	if pathRemainder == projectCollectionPath {
		switch r.Method {
		case http.MethodPost:
			handlers.HandleCreateProject(w, r, companyID)
		case http.MethodGet:
			handlers.HandleListProjects(w, r, companyID)
		default:
			WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
		}
		return
	}
	if strings.HasPrefix(pathRemainder, projectItemPrefix) {
		projectPath := strings.TrimPrefix(pathRemainder, projectItemPrefix)
		if projectPath == "" {
			WriteJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"resource not found",
				map[string]interface{}{"path": r.URL.Path},
			)
			return
		}

		projectParts := strings.Split(projectPath, "/")
		projectID, err := parsePositiveID(projectParts[0])
		if err != nil {
			WriteJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid project_id",
				map[string]interface{}{"path": r.URL.Path},
			)
			return
		}
		if len(projectParts) == 1 {
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
			return
		}
		if len(projectParts) == 2 && projectParts[1] == streamCollectionPath {
			switch r.Method {
			case http.MethodPost:
				handlers.HandleCreateStream(w, r, companyID, projectID)
			default:
				WriteMethodNotAllowed(w, r, http.MethodPost)
			}
			return
		}

		WriteJSONError(
			w,
			r,
			http.StatusNotFound,
			"not_found",
			"resource not found",
			map[string]interface{}{"path": r.URL.Path},
		)
		return
	}
	if pathRemainder == streamCollectionPath {
		switch r.Method {
		case http.MethodGet:
			handlers.HandleListStreams(w, r, companyID)
		default:
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return
	}
	if strings.HasPrefix(pathRemainder, checkJobsItemPrefix) {
		jobPath := strings.TrimPrefix(pathRemainder, checkJobsItemPrefix)
		if jobPath == "" {
			WriteJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"resource not found",
				map[string]interface{}{"path": r.URL.Path},
			)
			return
		}

		jobParts := strings.Split(jobPath, "/")
		jobID, err := parsePositiveID(jobParts[0])
		if err != nil {
			WriteJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid job_id",
				map[string]interface{}{"path": r.URL.Path},
			)
			return
		}
		if len(jobParts) == 1 {
			switch r.Method {
			case http.MethodGet:
				handlers.HandleGetCheckJob(w, r, companyID, jobID)
			default:
				WriteMethodNotAllowed(w, r, http.MethodGet)
			}
			return
		}
		if len(jobParts) == 2 && jobParts[1] == "result" {
			switch r.Method {
			case http.MethodGet:
				handlers.HandleGetCheckResultByJob(w, r, companyID, jobID)
			default:
				WriteMethodNotAllowed(w, r, http.MethodGet)
			}
			return
		}

		WriteJSONError(
			w,
			r,
			http.StatusNotFound,
			"not_found",
			"resource not found",
			map[string]interface{}{"path": r.URL.Path},
		)
		return
	}
	if strings.HasPrefix(pathRemainder, checkResultsItemPrefix) {
		resultIDRaw := strings.TrimPrefix(pathRemainder, checkResultsItemPrefix)
		if resultIDRaw == "" || strings.Contains(resultIDRaw, "/") {
			WriteJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"resource not found",
				map[string]interface{}{"path": r.URL.Path},
			)
			return
		}

		resultID, err := parsePositiveID(resultIDRaw)
		if err != nil {
			WriteJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid result_id",
				map[string]interface{}{"path": r.URL.Path},
			)
			return
		}

		switch r.Method {
		case http.MethodGet:
			handlers.HandleGetCheckResult(w, r, companyID, resultID)
		default:
			WriteMethodNotAllowed(w, r, http.MethodGet)
		}
		return
	}
	if strings.HasPrefix(pathRemainder, streamItemPrefix) {
		streamPath := strings.TrimPrefix(pathRemainder, streamItemPrefix)
		if streamPath == "" {
			WriteJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"resource not found",
				map[string]interface{}{"path": r.URL.Path},
			)
			return
		}

		streamParts := strings.Split(streamPath, "/")
		streamID, err := parsePositiveID(streamParts[0])
		if err != nil {
			WriteJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid stream_id",
				map[string]interface{}{"path": r.URL.Path},
			)
			return
		}
		if len(streamParts) == 1 {
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
			return
		}
		if len(streamParts) == 2 && streamParts[1] == checkJobsCollectionPath {
			switch r.Method {
			case http.MethodPost:
				handlers.HandleEnqueueCheckJob(w, r, companyID, streamID)
			case http.MethodGet:
				handlers.HandleListCheckJobs(w, r, companyID, streamID)
			default:
				WriteMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
			}
			return
		}
		if len(streamParts) == 2 && streamParts[1] == checkResultsCollectionPath {
			switch r.Method {
			case http.MethodGet:
				handlers.HandleListCheckResults(w, r, companyID, streamID)
			default:
				WriteMethodNotAllowed(w, r, http.MethodGet)
			}
			return
		}

		WriteJSONError(
			w,
			r,
			http.StatusNotFound,
			"not_found",
			"resource not found",
			map[string]interface{}{"path": r.URL.Path},
		)
		return
	}

	WriteJSONError(
		w,
		r,
		http.StatusNotFound,
		"not_found",
		"resource not found",
		map[string]interface{}{"path": r.URL.Path},
	)
}

func parseCompanyPath(path string) (int64, string, string) {
	const prefix = "/api/v1/companies/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", "not_found"
	}

	rawPath := strings.TrimPrefix(path, prefix)
	if rawPath == "" {
		return 0, "", "not_found"
	}

	parts := strings.SplitN(rawPath, "/", 2)
	companyID, err := parsePositiveID(parts[0])
	if err != nil {
		return 0, "", "validation_error"
	}

	if len(parts) == 1 {
		return companyID, "", ""
	}
	if parts[1] == "" {
		return 0, "", "not_found"
	}

	return companyID, parts[1], ""
}

func parsePositiveID(rawID string) (int64, error) {
	value, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid id")
	}
	return value, nil
}

func ParseCompanyPath(path string) (int64, string, string) {
	return parseCompanyPath(path)
}

func ParsePositiveID(rawID string) (int64, error) {
	return parsePositiveID(rawID)
}
