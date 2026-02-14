package api

import (
	"context"
	"database/sql"
	"errors"
	"github.com/example/hls-monitoring-platform/internal/domain"
	"log"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request createProjectRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(
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
		WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			WriteJSONError(
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("create project tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create project response encode error: %v", err)
	}
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, company_id, name, created_at, updated_at FROM projects WHERE company_id = $1 ORDER BY id ASC`,
		companyID,
	)
	if err != nil {
		log.Printf("list projects failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]project, 0)
	for rows.Next() {
		var item project
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			log.Printf("list projects scan failed: %v", err)
			WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list projects rows failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := projectListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := WriteJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list projects response encode error: %v", err)
	}
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get project response encode error: %v", err)
	}
}

func (s *Server) handlePatchProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	var request patchProjectRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(
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
		WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			WriteJSONError(
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("patch project tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch project response encode error: %v", err)
	}
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("delete project tx begin failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("delete project tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
