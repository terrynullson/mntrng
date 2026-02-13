package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/lib/pq"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
}

type company struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type companyListResponse struct {
	Items      []company `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}

type createCompanyRequest struct {
	Name string `json:"name"`
}

type patchCompanyRequest struct {
	Name string `json:"name"`
}

type project struct {
	ID        int64     `json:"id"`
	CompanyID int64     `json:"company_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type projectListResponse struct {
	Items      []project `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}

type createProjectRequest struct {
	Name string `json:"name"`
}

type patchProjectRequest struct {
	Name string `json:"name"`
}

type stream struct {
	ID        int64     `json:"id"`
	CompanyID int64     `json:"company_id"`
	ProjectID int64     `json:"project_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type streamListResponse struct {
	Items      []stream `json:"items"`
	NextCursor *string  `json:"next_cursor"`
}

type createStreamRequest struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	IsActive *bool  `json:"is_active"`
}

type patchStreamRequest struct {
	Name     *string `json:"name"`
	URL      *string `json:"url"`
	IsActive *bool   `json:"is_active"`
}

type checkJob struct {
	ID           int64      `json:"id"`
	CompanyID    int64      `json:"company_id"`
	StreamID     int64      `json:"stream_id"`
	PlannedAt    time.Time  `json:"planned_at"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	ErrorMessage *string    `json:"error_message"`
}

type checkJobListResponse struct {
	Items      []checkJob `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

type enqueueCheckJobRequest struct {
	PlannedAt string `json:"planned_at"`
}

type enqueueCheckJobResponse struct {
	Job checkJob `json:"job"`
}

type checkResult struct {
	ID             int64           `json:"id"`
	CompanyID      int64           `json:"company_id"`
	JobID          int64           `json:"job_id"`
	StreamID       int64           `json:"stream_id"`
	Status         string          `json:"status"`
	Checks         json.RawMessage `json:"checks"`
	ScreenshotPath *string         `json:"screenshot_path"`
	CreatedAt      time.Time       `json:"created_at"`
}

type checkResultListResponse struct {
	Items      []checkResult `json:"items"`
	NextCursor *string       `json:"next_cursor"`
}

type errorEnvelope struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details"`
	RequestID string      `json:"request_id"`
}

type apiServer struct {
	db *sql.DB
}

const (
	auditActorTypeAPI       = "api"
	auditActorIDSystem      = "system"
	auditEntityTypeStream   = "stream"
	auditActionStreamCreate = "create"
	auditActionStreamUpdate = "update"
	auditActionStreamDelete = "delete"
)

func main() {
	port := config.GetString("API_PORT", "8080")
	databaseURL := config.GetString("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	serverAPI := &apiServer{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", serverAPI.handleHealth)
	mux.HandleFunc("/api/v1/companies", serverAPI.handleCompanies)
	mux.HandleFunc("/api/v1/companies/", serverAPI.handleCompanyByID)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("api skeleton listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("api server failed: %v", err)
	}
}

func (s *apiServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	response := healthResponse{
		Status:  "ok",
		Service: "api",
		Time:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("health response encode error: %v", err)
	}
}

func (s *apiServer) handleCompanies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateCompany(w, r)
	case http.MethodGet:
		s.handleListCompanies(w, r)
	default:
		writeMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

func (s *apiServer) handleCompanyByID(w http.ResponseWriter, r *http.Request) {
	companyID, pathRemainder, pathErr := parseCompanyPath(r.URL.Path)
	if pathErr == "not_found" {
		writeJSONError(
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
		writeJSONError(
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
			s.handleGetCompany(w, r, companyID)
		case http.MethodPatch:
			s.handlePatchCompany(w, r, companyID)
		case http.MethodDelete:
			s.handleDeleteCompany(w, r, companyID)
		default:
			writeMethodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)
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
			s.handleCreateProject(w, r, companyID)
		case http.MethodGet:
			s.handleListProjects(w, r, companyID)
		default:
			writeMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
		}
		return
	}
	if strings.HasPrefix(pathRemainder, projectItemPrefix) {
		projectPath := strings.TrimPrefix(pathRemainder, projectItemPrefix)
		if projectPath == "" {
			writeJSONError(
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
			writeJSONError(
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
				s.handleGetProject(w, r, companyID, projectID)
			case http.MethodPatch:
				s.handlePatchProject(w, r, companyID, projectID)
			case http.MethodDelete:
				s.handleDeleteProject(w, r, companyID, projectID)
			default:
				writeMethodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)
			}
			return
		}
		if len(projectParts) == 2 && projectParts[1] == streamCollectionPath {
			switch r.Method {
			case http.MethodPost:
				s.handleCreateStream(w, r, companyID, projectID)
			default:
				writeMethodNotAllowed(w, r, http.MethodPost)
			}
			return
		}

		writeJSONError(
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
			s.handleListStreams(w, r, companyID)
		default:
			writeMethodNotAllowed(w, r, http.MethodGet)
		}
		return
	}
	if strings.HasPrefix(pathRemainder, checkJobsItemPrefix) {
		jobPath := strings.TrimPrefix(pathRemainder, checkJobsItemPrefix)
		if jobPath == "" {
			writeJSONError(
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
			writeJSONError(
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
				s.handleGetCheckJob(w, r, companyID, jobID)
			default:
				writeMethodNotAllowed(w, r, http.MethodGet)
			}
			return
		}
		if len(jobParts) == 2 && jobParts[1] == "result" {
			switch r.Method {
			case http.MethodGet:
				s.handleGetCheckResultByJob(w, r, companyID, jobID)
			default:
				writeMethodNotAllowed(w, r, http.MethodGet)
			}
			return
		}

		writeJSONError(
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
			writeJSONError(
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
			writeJSONError(
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
			s.handleGetCheckResult(w, r, companyID, resultID)
		default:
			writeMethodNotAllowed(w, r, http.MethodGet)
		}
		return
	}
	if strings.HasPrefix(pathRemainder, streamItemPrefix) {
		streamPath := strings.TrimPrefix(pathRemainder, streamItemPrefix)
		if streamPath == "" {
			writeJSONError(
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
			writeJSONError(
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
				s.handleGetStream(w, r, companyID, streamID)
			case http.MethodPatch:
				s.handlePatchStream(w, r, companyID, streamID)
			case http.MethodDelete:
				s.handleDeleteStream(w, r, companyID, streamID)
			default:
				writeMethodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)
			}
			return
		}
		if len(streamParts) == 2 && streamParts[1] == checkJobsCollectionPath {
			switch r.Method {
			case http.MethodPost:
				s.handleEnqueueCheckJob(w, r, companyID, streamID)
			case http.MethodGet:
				s.handleListCheckJobs(w, r, companyID, streamID)
			default:
				writeMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
			}
			return
		}
		if len(streamParts) == 2 && streamParts[1] == checkResultsCollectionPath {
			switch r.Method {
			case http.MethodGet:
				s.handleListCheckResults(w, r, companyID, streamID)
			default:
				writeMethodNotAllowed(w, r, http.MethodGet)
			}
			return
		}

		writeJSONError(
			w,
			r,
			http.StatusNotFound,
			"not_found",
			"resource not found",
			map[string]interface{}{"path": r.URL.Path},
		)
		return
	}

	writeJSONError(
		w,
		r,
		http.StatusNotFound,
		"not_found",
		"resource not found",
		map[string]interface{}{"path": r.URL.Path},
	)
}

func (s *apiServer) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	var request createCompanyRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"invalid request body",
			map[string]interface{}{"error": err.Error()},
		)
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"name is required",
			map[string]interface{}{"field": "name"},
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item company
	err := s.db.QueryRowContext(
		ctx,
		`INSERT INTO companies (name) VALUES ($1) RETURNING id, name, created_at`,
		name,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusConflict,
				"conflict",
				"company with the same name already exists",
				map[string]interface{}{"field": "name"},
			)
			return
		}

		log.Printf("create company failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create company response encode error: %v", err)
	}
}

func (s *apiServer) handleListCompanies(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, created_at FROM companies ORDER BY id ASC`,
	)
	if err != nil {
		log.Printf("list companies failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]company, 0)
	for rows.Next() {
		var item company
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			log.Printf("list companies scan failed: %v", err)
			writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list companies rows failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := companyListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list companies response encode error: %v", err)
	}
}

func (s *apiServer) handleGetCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item company
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, created_at FROM companies WHERE id = $1`,
		companyID,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"company not found",
				map[string]interface{}{"company_id": companyID},
			)
			return
		}

		log.Printf("get company failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get company response encode error: %v", err)
	}
}

func (s *apiServer) handlePatchCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request patchCompanyRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"invalid request body",
			map[string]interface{}{"error": err.Error()},
		)
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"name is required",
			map[string]interface{}{"field": "name"},
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item company
	err := s.db.QueryRowContext(
		ctx,
		`UPDATE companies SET name = $1 WHERE id = $2 RETURNING id, name, created_at`,
		name,
		companyID,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"company not found",
				map[string]interface{}{"company_id": companyID},
			)
			return
		}
		if isUniqueViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusConflict,
				"conflict",
				"company with the same name already exists",
				map[string]interface{}{"field": "name"},
			)
			return
		}

		log.Printf("patch company failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch company response encode error: %v", err)
	}
}

func (s *apiServer) handleDeleteCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM companies WHERE id = $1`,
		companyID,
	)
	if err != nil {
		log.Printf("delete company failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("delete company rows affected failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	if rowsAffected == 0 {
		writeJSONError(
			w,
			r,
			http.StatusNotFound,
			"not_found",
			"company not found",
			map[string]interface{}{"company_id": companyID},
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *apiServer) handleCreateProject(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request createProjectRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"invalid request body",
			map[string]interface{}{"error": err.Error()},
		)
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"name is required",
			map[string]interface{}{"field": "name"},
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item project
	err := s.db.QueryRowContext(
		ctx,
		`INSERT INTO projects (company_id, name) VALUES ($1, $2) RETURNING id, company_id, name, created_at, updated_at`,
		companyID,
		name,
	).Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusConflict,
				"conflict",
				"project with the same name already exists for this company",
				map[string]interface{}{"field": "name", "company_id": companyID},
			)
			return
		}
		if isForeignKeyViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"company not found",
				map[string]interface{}{"company_id": companyID},
			)
			return
		}

		log.Printf("create project failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create project response encode error: %v", err)
	}
}

func (s *apiServer) handleListProjects(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, company_id, name, created_at, updated_at FROM projects WHERE company_id = $1 ORDER BY id ASC`,
		companyID,
	)
	if err != nil {
		log.Printf("list projects failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]project, 0)
	for rows.Next() {
		var item project
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			log.Printf("list projects scan failed: %v", err)
			writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list projects rows failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := projectListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list projects response encode error: %v", err)
	}
}

func (s *apiServer) handleGetProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item project
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, name, created_at, updated_at FROM projects WHERE company_id = $1 AND id = $2`,
		companyID,
		projectID,
	).Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"project not found",
				map[string]interface{}{"company_id": companyID, "project_id": projectID},
			)
			return
		}

		log.Printf("get project failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get project response encode error: %v", err)
	}
}

func (s *apiServer) handlePatchProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	var request patchProjectRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"invalid request body",
			map[string]interface{}{"error": err.Error()},
		)
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"name is required",
			map[string]interface{}{"field": "name"},
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item project
	err := s.db.QueryRowContext(
		ctx,
		`UPDATE projects SET name = $1, updated_at = NOW() WHERE company_id = $2 AND id = $3 RETURNING id, company_id, name, created_at, updated_at`,
		name,
		companyID,
		projectID,
	).Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"project not found",
				map[string]interface{}{"company_id": companyID, "project_id": projectID},
			)
			return
		}
		if isUniqueViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusConflict,
				"conflict",
				"project with the same name already exists for this company",
				map[string]interface{}{"field": "name", "company_id": companyID},
			)
			return
		}

		log.Printf("patch project failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch project response encode error: %v", err)
	}
}

func (s *apiServer) handleDeleteProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM projects WHERE company_id = $1 AND id = $2`,
		companyID,
		projectID,
	)
	if err != nil {
		log.Printf("delete project failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("delete project rows affected failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	if rowsAffected == 0 {
		writeJSONError(
			w,
			r,
			http.StatusNotFound,
			"not_found",
			"project not found",
			map[string]interface{}{"company_id": companyID, "project_id": projectID},
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *apiServer) handleCreateStream(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	var request createStreamRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"invalid request body",
			map[string]interface{}{"error": err.Error()},
		)
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"name is required",
			map[string]interface{}{"field": "name"},
		)
		return
	}
	url := strings.TrimSpace(request.URL)
	if url == "" {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"url is required",
			map[string]interface{}{"field": "url"},
		)
		return
	}

	isActive := true
	if request.IsActive != nil {
		isActive = *request.IsActive
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("create stream tx begin failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var item stream
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO streams (company_id, project_id, name, url, is_active)
         SELECT $1, $2, $3, $4, $5
         WHERE EXISTS (
             SELECT 1 FROM projects p
             WHERE p.company_id = $1 AND p.id = $2
         )
         RETURNING id, company_id, project_id, name, url, is_active, created_at, updated_at`,
		companyID,
		projectID,
		name,
		url,
		isActive,
	).Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || isForeignKeyViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"project not found for company",
				map[string]interface{}{"company_id": companyID, "project_id": projectID},
			)
			return
		}
		if isUniqueViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusConflict,
				"conflict",
				"stream with the same name already exists in this project",
				map[string]interface{}{"company_id": companyID, "project_id": projectID, "field": "name"},
			)
			return
		}

		log.Printf("create stream failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"project_id": item.ProjectID,
		"name":       item.Name,
		"url":        item.URL,
		"is_active":  item.IsActive,
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		auditActorTypeAPI,
		auditActorIDSystem,
		auditEntityTypeStream,
		item.ID,
		auditActionStreamCreate,
		auditPayload,
	); err != nil {
		log.Printf("create stream audit insert failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("create stream tx commit failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create stream response encode error: %v", err)
	}
}

func (s *apiServer) handleListStreams(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	args := []interface{}{companyID}
	conditions := []string{"company_id = $1"}
	nextPlaceholder := 2

	if projectIDRaw := strings.TrimSpace(r.URL.Query().Get("project_id")); projectIDRaw != "" {
		projectID, err := parsePositiveID(projectIDRaw)
		if err != nil {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid project_id filter",
				map[string]interface{}{"project_id": projectIDRaw},
			)
			return
		}

		conditions = append(conditions, fmt.Sprintf("project_id = $%d", nextPlaceholder))
		args = append(args, projectID)
		nextPlaceholder++
	}

	if isActiveRaw := strings.TrimSpace(r.URL.Query().Get("is_active")); isActiveRaw != "" {
		isActive, err := strconv.ParseBool(isActiveRaw)
		if err != nil {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid is_active filter",
				map[string]interface{}{"is_active": isActiveRaw},
			)
			return
		}

		conditions = append(conditions, fmt.Sprintf("is_active = $%d", nextPlaceholder))
		args = append(args, isActive)
		nextPlaceholder++
	}

	query := fmt.Sprintf(
		`SELECT id, company_id, project_id, name, url, is_active, created_at, updated_at
         FROM streams
         WHERE %s
         ORDER BY id ASC`,
		strings.Join(conditions, " AND "),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("list streams failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]stream, 0)
	for rows.Next() {
		var item stream
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			log.Printf("list streams scan failed: %v", err)
			writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list streams rows failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := streamListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list streams response encode error: %v", err)
	}
}

func (s *apiServer) handleGetStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item stream
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, project_id, name, url, is_active, created_at, updated_at
         FROM streams
         WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"stream not found",
				map[string]interface{}{"company_id": companyID, "stream_id": streamID},
			)
			return
		}

		log.Printf("get stream failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get stream response encode error: %v", err)
	}
}

func (s *apiServer) handlePatchStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	var request patchStreamRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"invalid request body",
			map[string]interface{}{"error": err.Error()},
		)
		return
	}

	setClauses := make([]string, 0, 3)
	args := make([]interface{}, 0, 5)
	changePayload := make(map[string]interface{})
	nextPlaceholder := 1

	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"name must not be empty",
				map[string]interface{}{"field": "name"},
			)
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", nextPlaceholder))
		args = append(args, name)
		changePayload["name"] = name
		nextPlaceholder++
	}
	if request.URL != nil {
		url := strings.TrimSpace(*request.URL)
		if url == "" {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"url must not be empty",
				map[string]interface{}{"field": "url"},
			)
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("url = $%d", nextPlaceholder))
		args = append(args, url)
		changePayload["url"] = url
		nextPlaceholder++
	}
	if request.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", nextPlaceholder))
		args = append(args, *request.IsActive)
		changePayload["is_active"] = *request.IsActive
		nextPlaceholder++
	}
	if len(setClauses) == 0 {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"at least one field is required",
			map[string]interface{}{"fields": []string{"name", "url", "is_active"}},
		)
		return
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	companyPlaceholder := nextPlaceholder
	streamPlaceholder := nextPlaceholder + 1
	args = append(args, companyID, streamID)

	query := fmt.Sprintf(
		`UPDATE streams
         SET %s
         WHERE company_id = $%d
           AND id = $%d
           AND EXISTS (
               SELECT 1 FROM projects p
               WHERE p.id = streams.project_id
                 AND p.company_id = streams.company_id
           )
         RETURNING id, company_id, project_id, name, url, is_active, created_at, updated_at`,
		strings.Join(setClauses, ", "),
		companyPlaceholder,
		streamPlaceholder,
	)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("patch stream tx begin failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var item stream
	err = tx.QueryRowContext(ctx, query, args...).Scan(
		&item.ID,
		&item.CompanyID,
		&item.ProjectID,
		&item.Name,
		&item.URL,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"stream not found",
				map[string]interface{}{"company_id": companyID, "stream_id": streamID},
			)
			return
		}
		if isUniqueViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusConflict,
				"conflict",
				"stream with the same name already exists in this project",
				map[string]interface{}{"company_id": companyID, "stream_id": streamID, "field": "name"},
			)
			return
		}

		log.Printf("patch stream failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"project_id": item.ProjectID,
		"changes":    changePayload,
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		auditActorTypeAPI,
		auditActorIDSystem,
		auditEntityTypeStream,
		item.ID,
		auditActionStreamUpdate,
		auditPayload,
	); err != nil {
		log.Printf("patch stream audit insert failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("patch stream tx commit failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch stream response encode error: %v", err)
	}
}

func (s *apiServer) handleDeleteStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("delete stream tx begin failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var deleted stream
	err = tx.QueryRowContext(
		ctx,
		`DELETE FROM streams
         WHERE company_id = $1 AND id = $2
         RETURNING id, company_id, project_id, name, url, is_active, created_at, updated_at`,
		companyID,
		streamID,
	).Scan(
		&deleted.ID,
		&deleted.CompanyID,
		&deleted.ProjectID,
		&deleted.Name,
		&deleted.URL,
		&deleted.IsActive,
		&deleted.CreatedAt,
		&deleted.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"stream not found",
				map[string]interface{}{"company_id": companyID, "stream_id": streamID},
			)
			return
		}

		log.Printf("delete stream failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"project_id": deleted.ProjectID,
		"name":       deleted.Name,
		"url":        deleted.URL,
		"is_active":  deleted.IsActive,
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		auditActorTypeAPI,
		auditActorIDSystem,
		auditEntityTypeStream,
		deleted.ID,
		auditActionStreamDelete,
		auditPayload,
	); err != nil {
		log.Printf("delete stream audit insert failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("delete stream tx commit failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *apiServer) handleEnqueueCheckJob(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	var request enqueueCheckJobRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"invalid request body",
			map[string]interface{}{"error": err.Error()},
		)
		return
	}

	plannedAtRaw := strings.TrimSpace(request.PlannedAt)
	if plannedAtRaw == "" {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"planned_at is required",
			map[string]interface{}{"field": "planned_at"},
		)
		return
	}

	plannedAt, err := time.Parse(time.RFC3339, plannedAtRaw)
	if err != nil {
		writeJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"planned_at must be RFC3339 timestamp",
			map[string]interface{}{"field": "planned_at", "value": plannedAtRaw},
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item checkJob
	err = s.db.QueryRowContext(
		ctx,
		`INSERT INTO check_jobs (company_id, stream_id, planned_at)
         SELECT $1, $2, $3
         WHERE EXISTS (
             SELECT 1 FROM streams s
             WHERE s.company_id = $1 AND s.id = $2
         )
         RETURNING id, company_id, stream_id, planned_at, status, created_at, started_at, finished_at, error_message`,
		companyID,
		streamID,
		plannedAt.UTC(),
	).Scan(
		&item.ID,
		&item.CompanyID,
		&item.StreamID,
		&item.PlannedAt,
		&item.Status,
		&item.CreatedAt,
		&item.StartedAt,
		&item.FinishedAt,
		&item.ErrorMessage,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || isForeignKeyViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"stream not found for company",
				map[string]interface{}{"company_id": companyID, "stream_id": streamID},
			)
			return
		}
		if isUniqueViolation(err) {
			writeJSONError(
				w,
				r,
				http.StatusConflict,
				"conflict",
				"check job already exists for stream and planned_at",
				map[string]interface{}{"company_id": companyID, "stream_id": streamID, "planned_at": plannedAtRaw},
			)
			return
		}

		log.Printf("enqueue check job failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusAccepted, enqueueCheckJobResponse{Job: item}); err != nil {
		log.Printf("enqueue check job response encode error: %v", err)
	}
}

func (s *apiServer) handleGetCheckJob(w http.ResponseWriter, r *http.Request, companyID int64, jobID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var item checkJob
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, stream_id, planned_at, status, created_at, started_at, finished_at, error_message
         FROM check_jobs
         WHERE company_id = $1 AND id = $2`,
		companyID,
		jobID,
	).Scan(
		&item.ID,
		&item.CompanyID,
		&item.StreamID,
		&item.PlannedAt,
		&item.Status,
		&item.CreatedAt,
		&item.StartedAt,
		&item.FinishedAt,
		&item.ErrorMessage,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"check job not found",
				map[string]interface{}{"company_id": companyID, "job_id": jobID},
			)
			return
		}

		log.Printf("get check job failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check job response encode error: %v", err)
	}
}

func (s *apiServer) handleListCheckJobs(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var streamExists int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM streams WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&streamExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"stream not found for company",
				map[string]interface{}{"company_id": companyID, "stream_id": streamID},
			)
			return
		}

		log.Printf("check stream existence for check jobs failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if statusRaw := strings.TrimSpace(r.URL.Query().Get("status")); statusRaw != "" {
		status, ok := normalizeCheckJobStatus(statusRaw)
		if !ok {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid status filter",
				map[string]interface{}{"status": statusRaw, "allowed": []string{"queued", "running", "done", "failed"}},
			)
			return
		}
		conditions = append(conditions, fmt.Sprintf("status = $%d", nextPlaceholder))
		args = append(args, status)
		nextPlaceholder++
	}

	if fromRaw := strings.TrimSpace(r.URL.Query().Get("from")); fromRaw != "" {
		fromTime, parseErr := time.Parse(time.RFC3339, fromRaw)
		if parseErr != nil {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid from filter",
				map[string]interface{}{"from": fromRaw},
			)
			return
		}
		conditions = append(conditions, fmt.Sprintf("planned_at >= $%d", nextPlaceholder))
		args = append(args, fromTime.UTC())
		nextPlaceholder++
	}

	if toRaw := strings.TrimSpace(r.URL.Query().Get("to")); toRaw != "" {
		toTime, parseErr := time.Parse(time.RFC3339, toRaw)
		if parseErr != nil {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid to filter",
				map[string]interface{}{"to": toRaw},
			)
			return
		}
		conditions = append(conditions, fmt.Sprintf("planned_at <= $%d", nextPlaceholder))
		args = append(args, toTime.UTC())
		nextPlaceholder++
	}

	query := fmt.Sprintf(
		`SELECT id, company_id, stream_id, planned_at, status, created_at, started_at, finished_at, error_message
         FROM check_jobs
         WHERE %s
         ORDER BY planned_at DESC, id DESC`,
		strings.Join(conditions, " AND "),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("list check jobs failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]checkJob, 0)
	for rows.Next() {
		var item checkJob
		if err := rows.Scan(
			&item.ID,
			&item.CompanyID,
			&item.StreamID,
			&item.PlannedAt,
			&item.Status,
			&item.CreatedAt,
			&item.StartedAt,
			&item.FinishedAt,
			&item.ErrorMessage,
		); err != nil {
			log.Printf("list check jobs scan failed: %v", err)
			writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list check jobs rows failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := checkJobListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list check jobs response encode error: %v", err)
	}
}

func (s *apiServer) handleGetCheckResult(w http.ResponseWriter, r *http.Request, companyID int64, resultID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, job_id, stream_id, status, checks, screenshot_path, created_at
         FROM check_results
         WHERE company_id = $1 AND id = $2`,
		companyID,
		resultID,
	)

	item, err := scanCheckResult(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"check result not found",
				map[string]interface{}{"company_id": companyID, "result_id": resultID},
			)
			return
		}

		log.Printf("get check result failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check result response encode error: %v", err)
	}
}

func (s *apiServer) handleListCheckResults(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var streamExists int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM streams WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&streamExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"stream not found for company",
				map[string]interface{}{"company_id": companyID, "stream_id": streamID},
			)
			return
		}

		log.Printf("check stream existence for check results failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if statusRaw := strings.TrimSpace(r.URL.Query().Get("status")); statusRaw != "" {
		status, ok := normalizeCheckResultStatus(statusRaw)
		if !ok {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid status filter",
				map[string]interface{}{"status": statusRaw, "allowed": []string{"OK", "WARN", "FAIL"}},
			)
			return
		}

		conditions = append(conditions, fmt.Sprintf("status = $%d", nextPlaceholder))
		args = append(args, status)
		nextPlaceholder++
	}

	if fromRaw := strings.TrimSpace(r.URL.Query().Get("from")); fromRaw != "" {
		fromTime, parseErr := time.Parse(time.RFC3339, fromRaw)
		if parseErr != nil {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid from filter",
				map[string]interface{}{"from": fromRaw},
			)
			return
		}
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", nextPlaceholder))
		args = append(args, fromTime.UTC())
		nextPlaceholder++
	}

	if toRaw := strings.TrimSpace(r.URL.Query().Get("to")); toRaw != "" {
		toTime, parseErr := time.Parse(time.RFC3339, toRaw)
		if parseErr != nil {
			writeJSONError(
				w,
				r,
				http.StatusBadRequest,
				"validation_error",
				"invalid to filter",
				map[string]interface{}{"to": toRaw},
			)
			return
		}
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", nextPlaceholder))
		args = append(args, toTime.UTC())
		nextPlaceholder++
	}

	query := fmt.Sprintf(
		`SELECT id, company_id, job_id, stream_id, status, checks, screenshot_path, created_at
         FROM check_results
         WHERE %s
         ORDER BY created_at DESC, id DESC`,
		strings.Join(conditions, " AND "),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("list check results failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]checkResult, 0)
	for rows.Next() {
		item, scanErr := scanCheckResult(rows)
		if scanErr != nil {
			log.Printf("list check results scan failed: %v", scanErr)
			writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list check results rows failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := checkResultListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list check results response encode error: %v", err)
	}
}

func (s *apiServer) handleGetCheckResultByJob(w http.ResponseWriter, r *http.Request, companyID int64, jobID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, job_id, stream_id, status, checks, screenshot_path, created_at
         FROM check_results
         WHERE company_id = $1 AND job_id = $2`,
		companyID,
		jobID,
	)

	item, err := scanCheckResult(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"check result not found for job",
				map[string]interface{}{"company_id": companyID, "job_id": jobID},
			)
			return
		}

		log.Printf("get check result by job failed: %v", err)
		writeJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := writeJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check result by job response encode error: %v", err)
	}
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanCheckResult(scanner rowScanner) (checkResult, error) {
	var item checkResult
	var dbStatus string
	var checksRaw []byte

	err := scanner.Scan(
		&item.ID,
		&item.CompanyID,
		&item.JobID,
		&item.StreamID,
		&dbStatus,
		&checksRaw,
		&item.ScreenshotPath,
		&item.CreatedAt,
	)
	if err != nil {
		return checkResult{}, err
	}

	if len(checksRaw) == 0 {
		checksRaw = []byte("{}")
	}
	item.Checks = json.RawMessage(checksRaw)
	item.Status = formatCheckResultStatus(dbStatus)
	return item, nil
}

func insertAuditLogTx(
	ctx context.Context,
	tx *sql.Tx,
	companyID int64,
	actorType string,
	actorID string,
	entityType string,
	entityID int64,
	action string,
	payload map[string]interface{},
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO audit_log (company_id, actor_type, actor_id, entity_type, entity_id, action, payload)
         VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)`,
		companyID,
		actorType,
		actorID,
		entityType,
		entityID,
		action,
		string(payloadJSON),
	)
	return err
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(payload)
}

func writeJSONError(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	code string,
	message string,
	details interface{},
) {
	err := writeJSON(w, statusCode, errorEnvelope{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: requestIDFromRequest(r),
	})
	if err != nil {
		log.Printf("error response encode failed: %v", err)
	}
}

func writeMethodNotAllowed(w http.ResponseWriter, r *http.Request, allowedMethods ...string) {
	w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	writeJSONError(
		w,
		r,
		http.StatusMethodNotAllowed,
		"method_not_allowed",
		"method is not allowed for this endpoint",
		map[string]interface{}{
			"method":          r.Method,
			"allowed_methods": allowedMethods,
		},
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

func decodeJSONBody(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pq.Error
	return errors.As(err, &pgErr) && string(pgErr.Code) == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pq.Error
	return errors.As(err, &pgErr) && string(pgErr.Code) == "23503"
}

func normalizeCheckJobStatus(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "queued":
		return "queued", true
	case "running":
		return "running", true
	case "done":
		return "done", true
	case "failed":
		return "failed", true
	default:
		return "", false
	}
}

func normalizeCheckResultStatus(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ok":
		return "ok", true
	case "warn":
		return "warn", true
	case "fail":
		return "fail", true
	default:
		return "", false
	}
}

func formatCheckResultStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ok":
		return "OK"
	case "warn":
		return "WARN"
	case "fail":
		return "FAIL"
	default:
		return strings.ToUpper(strings.TrimSpace(raw))
	}
}

func requestIDFromRequest(r *http.Request) string {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	return "req_" + time.Now().UTC().Format("20060102150405.000000000")
}
