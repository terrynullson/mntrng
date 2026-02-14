package api

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	name, ok := decodeCreateCompanyName(w, r)
	if !ok {
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

	item, err := insertCompanyTx(ctx, tx, name)
	if err != nil {
		handleCreateCompanyInsertError(w, r, err)
		return
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
		map[string]interface{}{"name": item.Name},
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

	rows, err := s.db.QueryContext(ctx, `SELECT id, name, created_at FROM companies ORDER BY id ASC`)
	if err != nil {
		log.Printf("list companies failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items, ok := scanCompanyRows(w, r, rows)
	if !ok {
		return
	}

	if err := WriteJSON(w, http.StatusOK, companyListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list companies response encode error: %v", err)
	}
}

func (s *Server) handleGetCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := getCompanyByID(ctx, s.db, companyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeCompanyNotFound(w, r, companyID)
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
	name, ok := decodePatchCompanyName(w, r)
	if !ok {
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

	item, err := updateCompanyNameTx(ctx, tx, companyID, name)
	if err != nil {
		handlePatchCompanyUpdateError(w, r, companyID, err)
		return
	}

	auditPayload := map[string]interface{}{"changes": map[string]interface{}{"name": name}}
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

	existing, err := lockCompanyForDeleteTx(ctx, tx, companyID)
	if err != nil {
		handleDeleteCompanyLockError(w, r, companyID, err)
		return
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
		map[string]interface{}{"name": existing.Name},
	); err != nil {
		log.Printf("delete company audit insert failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	rowsAffected, err := deleteCompanyRowTx(ctx, tx, companyID)
	if err != nil {
		log.Printf("delete company failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	if rowsAffected == 0 {
		writeCompanyNotFound(w, r, companyID)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("delete company tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeCreateCompanyName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var request createCompanyRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return "", false
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "name is required", map[string]interface{}{"field": "name"})
		return "", false
	}
	return name, true
}

func decodePatchCompanyName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var request patchCompanyRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return "", false
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "name is required", map[string]interface{}{"field": "name"})
		return "", false
	}
	return name, true
}

func insertCompanyTx(ctx context.Context, tx *sql.Tx, name string) (company, error) {
	var item company
	err := tx.QueryRowContext(ctx, `INSERT INTO companies (name) VALUES ($1) RETURNING id, name, created_at`, name).Scan(&item.ID, &item.Name, &item.CreatedAt)
	return item, err
}

func getCompanyByID(ctx context.Context, db *sql.DB, companyID int64) (company, error) {
	var item company
	err := db.QueryRowContext(ctx, `SELECT id, name, created_at FROM companies WHERE id = $1`, companyID).Scan(&item.ID, &item.Name, &item.CreatedAt)
	return item, err
}

func updateCompanyNameTx(ctx context.Context, tx *sql.Tx, companyID int64, name string) (company, error) {
	var item company
	err := tx.QueryRowContext(ctx, `UPDATE companies SET name = $1 WHERE id = $2 RETURNING id, name, created_at`, name, companyID).Scan(&item.ID, &item.Name, &item.CreatedAt)
	return item, err
}

func lockCompanyForDeleteTx(ctx context.Context, tx *sql.Tx, companyID int64) (company, error) {
	var existing company
	err := tx.QueryRowContext(ctx, `SELECT id, name, created_at FROM companies WHERE id = $1 FOR UPDATE`, companyID).Scan(&existing.ID, &existing.Name, &existing.CreatedAt)
	return existing, err
}

func deleteCompanyRowTx(ctx context.Context, tx *sql.Tx, companyID int64) (int64, error) {
	result, err := tx.ExecContext(ctx, `DELETE FROM companies WHERE id = $1`, companyID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanCompanyRows(w http.ResponseWriter, r *http.Request, rows *sql.Rows) ([]company, bool) {
	items := make([]company, 0)
	for rows.Next() {
		var item company
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			log.Printf("list companies scan failed: %v", err)
			WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return nil, false
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list companies rows failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return nil, false
	}
	return items, true
}

func handleCreateCompanyInsertError(w http.ResponseWriter, r *http.Request, err error) {
	if isUniqueViolation(err) {
		WriteJSONError(w, r, http.StatusConflict, "conflict", "company with the same name already exists", map[string]interface{}{"field": "name"})
		return
	}
	log.Printf("create company failed: %v", err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
}

func handlePatchCompanyUpdateError(w http.ResponseWriter, r *http.Request, companyID int64, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeCompanyNotFound(w, r, companyID)
		return
	}
	if isUniqueViolation(err) {
		WriteJSONError(w, r, http.StatusConflict, "conflict", "company with the same name already exists", map[string]interface{}{"field": "name"})
		return
	}
	log.Printf("patch company failed: %v", err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
}

func handleDeleteCompanyLockError(w http.ResponseWriter, r *http.Request, companyID int64, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeCompanyNotFound(w, r, companyID)
		return
	}
	log.Printf("delete company lookup failed: %v", err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
}

func writeCompanyNotFound(w http.ResponseWriter, r *http.Request, companyID int64) {
	WriteJSONError(w, r, http.StatusNotFound, "not_found", "company not found", map[string]interface{}{"company_id": companyID})
}
