package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (s *Server) handlePatchStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	var request patchStreamRequest
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

	setClauses := make([]string, 0, 3)
	args := make([]interface{}, 0, 5)
	changePayload := make(map[string]interface{})
	nextPlaceholder := 1

	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" {
			WriteJSONError(
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
			WriteJSONError(
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
		WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			WriteJSONError(
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("patch stream tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch stream response encode error: %v", err)
	}
}

func (s *Server) handleDeleteStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("delete stream tx begin failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("delete stream tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
