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

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	var request createCompanyRequest
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
		log.Printf("create company tx begin failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("create company tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create company response encode error: %v", err)
	}
}

func (s *Server) handleListCompanies(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, created_at FROM companies ORDER BY id ASC`,
	)
	if err != nil {
		log.Printf("list companies failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]company, 0)
	for rows.Next() {
		var item company
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			log.Printf("list companies scan failed: %v", err)
			WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list companies rows failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := companyListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := WriteJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list companies response encode error: %v", err)
	}
}

func (s *Server) handleGetCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
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

		log.Printf("get company failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get company response encode error: %v", err)
	}
}

func (s *Server) handlePatchCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request patchCompanyRequest
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
		log.Printf("patch company tx begin failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		if isUniqueViolation(err) {
			WriteJSONError(
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
		item.ID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeCompany,
		item.ID,
		domain.AuditActionCompanyUpdate,
		auditPayload,
	); err != nil {
		log.Printf("patch company audit insert failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("patch company tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch company response encode error: %v", err)
	}
}

func (s *Server) handleDeleteCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("delete company tx begin failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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

		log.Printf("delete company lookup failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM companies WHERE id = $1`,
		companyID,
	)
	if err != nil {
		log.Printf("delete company failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("delete company rows affected failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	if rowsAffected == 0 {
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

	if err := tx.Commit(); err != nil {
		log.Printf("delete company tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
