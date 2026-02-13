package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

type errorEnvelope struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details"`
	RequestID string      `json:"request_id"`
}

type apiServer struct {
	db *sql.DB
}

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
		projectIDRaw := strings.TrimPrefix(pathRemainder, projectItemPrefix)
		if projectIDRaw == "" || strings.Contains(projectIDRaw, "/") {
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

		projectID, err := parsePositiveID(projectIDRaw)
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

func requestIDFromRequest(r *http.Request) string {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	return "req_" + time.Now().UTC().Format("20060102150405.000000000")
}
