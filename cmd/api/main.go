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
	"github.com/example/hls-monitoring-platform/internal/domain"
	httpapi "github.com/example/hls-monitoring-platform/internal/http/api"
	"github.com/lib/pq"
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
	mux := httpapi.NewRouter(httpapi.RouterHandlers{
		HandleHealth:              serverAPI.handleHealth,
		HandleCreateCompany:       serverAPI.handleCreateCompany,
		HandleListCompanies:       serverAPI.handleListCompanies,
		HandleGetCompany:          serverAPI.handleGetCompany,
		HandlePatchCompany:        serverAPI.handlePatchCompany,
		HandleDeleteCompany:       serverAPI.handleDeleteCompany,
		HandleCreateProject:       serverAPI.handleCreateProject,
		HandleListProjects:        serverAPI.handleListProjects,
		HandleGetProject:          serverAPI.handleGetProject,
		HandlePatchProject:        serverAPI.handlePatchProject,
		HandleDeleteProject:       serverAPI.handleDeleteProject,
		HandleCreateStream:        serverAPI.handleCreateStream,
		HandleListStreams:         serverAPI.handleListStreams,
		HandleGetStream:           serverAPI.handleGetStream,
		HandlePatchStream:         serverAPI.handlePatchStream,
		HandleDeleteStream:        serverAPI.handleDeleteStream,
		HandleEnqueueCheckJob:     serverAPI.handleEnqueueCheckJob,
		HandleGetCheckJob:         serverAPI.handleGetCheckJob,
		HandleListCheckJobs:       serverAPI.handleListCheckJobs,
		HandleGetCheckResult:      serverAPI.handleGetCheckResult,
		HandleListCheckResults:    serverAPI.handleListCheckResults,
		HandleGetCheckResultByJob: serverAPI.handleGetCheckResultByJob,
	})

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
		httpapi.WriteMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	response := healthResponse{
		Status:  "ok",
		Service: "api",
		Time:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := httpapi.WriteJSON(w, http.StatusOK, response); err != nil {
		log.Printf("health response encode error: %v", err)
	}
}

func (s *apiServer) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	var request createCompanyRequest
	if err := httpapi.DecodeJSONBody(r, &request); err != nil {
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("create company tx begin failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var item company
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO companies (name) VALUES ($1) RETURNING id, name, created_at`,
		name,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"name": item.Name,
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		item.ID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeCompany,
		item.ID,
		domain.AuditActionCompanyCreate,
		auditPayload,
	); err != nil {
		log.Printf("create company audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("create company tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusCreated, item); err != nil {
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]company, 0)
	for rows.Next() {
		var item company
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			log.Printf("list companies scan failed: %v", err)
			httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list companies rows failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := companyListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := httpapi.WriteJSON(w, http.StatusOK, response); err != nil {
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get company response encode error: %v", err)
	}
}

func (s *apiServer) handlePatchCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request patchCompanyRequest
	if err := httpapi.DecodeJSONBody(r, &request); err != nil {
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("patch company tx begin failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var item company
	err = tx.QueryRowContext(
		ctx,
		`UPDATE companies SET name = $1 WHERE id = $2 RETURNING id, name, created_at`,
		name,
		companyID,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"changes": map[string]interface{}{
			"name": name,
		},
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		item.ID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeCompany,
		item.ID,
		domain.AuditActionCompanyUpdate,
		auditPayload,
	); err != nil {
		log.Printf("patch company audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("patch company tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch company response encode error: %v", err)
	}
}

func (s *apiServer) handleDeleteCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("delete company tx begin failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var existing company
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, name, created_at
         FROM companies
         WHERE id = $1
         FOR UPDATE`,
		companyID,
	).Scan(&existing.ID, &existing.Name, &existing.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpapi.WriteJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"company not found",
				map[string]interface{}{"company_id": companyID},
			)
			return
		}

		log.Printf("delete company lookup failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"name": existing.Name,
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		existing.ID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeCompany,
		existing.ID,
		domain.AuditActionCompanyDelete,
		auditPayload,
	); err != nil {
		log.Printf("delete company audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM companies WHERE id = $1`,
		companyID,
	)
	if err != nil {
		log.Printf("delete company failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("delete company rows affected failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	if rowsAffected == 0 {
		httpapi.WriteJSONError(
			w,
			r,
			http.StatusNotFound,
			"not_found",
			"company not found",
			map[string]interface{}{"company_id": companyID},
		)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("delete company tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *apiServer) handleCreateProject(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request createProjectRequest
	if err := httpapi.DecodeJSONBody(r, &request); err != nil {
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("create project tx begin failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var item project
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO projects (company_id, name) VALUES ($1, $2) RETURNING id, company_id, name, created_at, updated_at`,
		companyID,
		name,
	).Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"name": item.Name,
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeProject,
		item.ID,
		domain.AuditActionProjectCreate,
		auditPayload,
	); err != nil {
		log.Printf("create project audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("create project tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusCreated, item); err != nil {
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]project, 0)
	for rows.Next() {
		var item project
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			log.Printf("list projects scan failed: %v", err)
			httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list projects rows failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := projectListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := httpapi.WriteJSON(w, http.StatusOK, response); err != nil {
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get project response encode error: %v", err)
	}
}

func (s *apiServer) handlePatchProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	var request patchProjectRequest
	if err := httpapi.DecodeJSONBody(r, &request); err != nil {
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("patch project tx begin failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var item project
	err = tx.QueryRowContext(
		ctx,
		`UPDATE projects SET name = $1, updated_at = NOW() WHERE company_id = $2 AND id = $3 RETURNING id, company_id, name, created_at, updated_at`,
		name,
		companyID,
		projectID,
	).Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"changes": map[string]interface{}{
			"name": name,
		},
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeProject,
		item.ID,
		domain.AuditActionProjectUpdate,
		auditPayload,
	); err != nil {
		log.Printf("patch project audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("patch project tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch project response encode error: %v", err)
	}
}

func (s *apiServer) handleDeleteProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("delete project tx begin failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	var deleted project
	err = tx.QueryRowContext(
		ctx,
		`DELETE FROM projects
         WHERE company_id = $1 AND id = $2
         RETURNING id, company_id, name, created_at, updated_at`,
		companyID,
		projectID,
	).Scan(
		&deleted.ID,
		&deleted.CompanyID,
		&deleted.Name,
		&deleted.CreatedAt,
		&deleted.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpapi.WriteJSONError(
				w,
				r,
				http.StatusNotFound,
				"not_found",
				"project not found",
				map[string]interface{}{"company_id": companyID, "project_id": projectID},
			)
			return
		}

		log.Printf("delete project failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	auditPayload := map[string]interface{}{
		"name": deleted.Name,
	}
	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeProject,
		deleted.ID,
		domain.AuditActionProjectDelete,
		auditPayload,
	); err != nil {
		log.Printf("delete project audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("delete project tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *apiServer) handleCreateStream(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	var request createStreamRequest
	if err := httpapi.DecodeJSONBody(r, &request); err != nil {
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeStream,
		item.ID,
		domain.AuditActionStreamCreate,
		auditPayload,
	); err != nil {
		log.Printf("create stream audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("create stream tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusCreated, item); err != nil {
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
		projectID, err := httpapi.ParsePositiveID(projectIDRaw)
		if err != nil {
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]stream, 0)
	for rows.Next() {
		var item stream
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			log.Printf("list streams scan failed: %v", err)
			httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list streams rows failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := streamListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := httpapi.WriteJSON(w, http.StatusOK, response); err != nil {
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get stream response encode error: %v", err)
	}
}

func (s *apiServer) handlePatchStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	var request patchStreamRequest
	if err := httpapi.DecodeJSONBody(r, &request); err != nil {
		httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeStream,
		item.ID,
		domain.AuditActionStreamUpdate,
		auditPayload,
	); err != nil {
		log.Printf("patch stream audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("patch stream tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch stream response encode error: %v", err)
	}
}

func (s *apiServer) handleDeleteStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("delete stream tx begin failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeStream,
		deleted.ID,
		domain.AuditActionStreamDelete,
		auditPayload,
	); err != nil {
		log.Printf("delete stream audit insert failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("delete stream tx commit failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *apiServer) handleEnqueueCheckJob(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	var request enqueueCheckJobRequest
	if err := httpapi.DecodeJSONBody(r, &request); err != nil {
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusAccepted, enqueueCheckJobResponse{Job: item}); err != nil {
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if statusRaw := strings.TrimSpace(r.URL.Query().Get("status")); statusRaw != "" {
		status, ok := normalizeCheckJobStatus(statusRaw)
		if !ok {
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list check jobs rows failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := checkJobListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := httpapi.WriteJSON(w, http.StatusOK, response); err != nil {
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if statusRaw := strings.TrimSpace(r.URL.Query().Get("status")); statusRaw != "" {
		status, ok := normalizeCheckResultStatus(statusRaw)
		if !ok {
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]checkResult, 0)
	for rows.Next() {
		item, scanErr := scanCheckResult(rows)
		if scanErr != nil {
			log.Printf("list check results scan failed: %v", scanErr)
			httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list check results rows failed: %v", err)
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := checkResultListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := httpapi.WriteJSON(w, http.StatusOK, response); err != nil {
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
			httpapi.WriteJSONError(
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
		httpapi.WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, item); err != nil {
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
