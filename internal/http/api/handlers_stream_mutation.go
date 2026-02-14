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

type streamPatchMutation struct {
	setClauses         []string
	args               []interface{}
	changePayload      map[string]interface{}
	nextPlaceholder    int
	companyPlaceholder int
	streamPlaceholder  int
	query              string
}

func (s *Server) handlePatchStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	mutation, ok := decodeStreamPatchMutation(w, r)
	if !ok {
		return
	}
	prepareStreamPatchMutation(&mutation, companyID, streamID)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("patch stream tx begin failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	item, err := executePatchStreamTx(ctx, tx, mutation)
	if err != nil {
		handlePatchStreamError(w, r, companyID, streamID, err)
		return
	}

	auditPayload := map[string]interface{}{
		"project_id": item.ProjectID,
		"changes":    mutation.changePayload,
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

	deleted, err := deleteStreamRowTx(ctx, tx, companyID, streamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeStreamNotFound(w, r, companyID, streamID)
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

func decodeStreamPatchMutation(w http.ResponseWriter, r *http.Request) (streamPatchMutation, bool) {
	var request patchStreamRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return streamPatchMutation{}, false
	}

	mutation := streamPatchMutation{
		setClauses:      make([]string, 0, 3),
		args:            make([]interface{}, 0, 5),
		changePayload:   make(map[string]interface{}),
		nextPlaceholder: 1,
	}

	if !appendStreamPatchName(w, r, request, &mutation) {
		return streamPatchMutation{}, false
	}
	if !appendStreamPatchURL(w, r, request, &mutation) {
		return streamPatchMutation{}, false
	}
	appendStreamPatchIsActive(request, &mutation)

	if len(mutation.setClauses) == 0 {
		WriteJSONError(
			w,
			r,
			http.StatusBadRequest,
			"validation_error",
			"at least one field is required",
			map[string]interface{}{"fields": []string{"name", "url", "is_active"}},
		)
		return streamPatchMutation{}, false
	}

	return mutation, true
}

func appendStreamPatchName(w http.ResponseWriter, r *http.Request, request patchStreamRequest, mutation *streamPatchMutation) bool {
	if request.Name == nil {
		return true
	}
	name := strings.TrimSpace(*request.Name)
	if name == "" {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "name must not be empty", map[string]interface{}{"field": "name"})
		return false
	}
	mutation.setClauses = append(mutation.setClauses, fmt.Sprintf("name = $%d", mutation.nextPlaceholder))
	mutation.args = append(mutation.args, name)
	mutation.changePayload["name"] = name
	mutation.nextPlaceholder++
	return true
}

func appendStreamPatchURL(w http.ResponseWriter, r *http.Request, request patchStreamRequest, mutation *streamPatchMutation) bool {
	if request.URL == nil {
		return true
	}
	url := strings.TrimSpace(*request.URL)
	if url == "" {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "url must not be empty", map[string]interface{}{"field": "url"})
		return false
	}
	mutation.setClauses = append(mutation.setClauses, fmt.Sprintf("url = $%d", mutation.nextPlaceholder))
	mutation.args = append(mutation.args, url)
	mutation.changePayload["url"] = url
	mutation.nextPlaceholder++
	return true
}

func appendStreamPatchIsActive(request patchStreamRequest, mutation *streamPatchMutation) {
	if request.IsActive == nil {
		return
	}
	mutation.setClauses = append(mutation.setClauses, fmt.Sprintf("is_active = $%d", mutation.nextPlaceholder))
	mutation.args = append(mutation.args, *request.IsActive)
	mutation.changePayload["is_active"] = *request.IsActive
	mutation.nextPlaceholder++
}

func prepareStreamPatchMutation(mutation *streamPatchMutation, companyID int64, streamID int64) {
	mutation.setClauses = append(mutation.setClauses, "updated_at = NOW()")
	mutation.companyPlaceholder = mutation.nextPlaceholder
	mutation.streamPlaceholder = mutation.nextPlaceholder + 1
	mutation.args = append(mutation.args, companyID, streamID)
	mutation.query = fmt.Sprintf(
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
		strings.Join(mutation.setClauses, ", "),
		mutation.companyPlaceholder,
		mutation.streamPlaceholder,
	)
}

func executePatchStreamTx(ctx context.Context, tx *sql.Tx, mutation streamPatchMutation) (stream, error) {
	var item stream
	err := tx.QueryRowContext(ctx, mutation.query, mutation.args...).Scan(
		&item.ID,
		&item.CompanyID,
		&item.ProjectID,
		&item.Name,
		&item.URL,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func deleteStreamRowTx(ctx context.Context, tx *sql.Tx, companyID int64, streamID int64) (stream, error) {
	var deleted stream
	err := tx.QueryRowContext(
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
	return deleted, err
}

func handlePatchStreamError(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeStreamNotFound(w, r, companyID, streamID)
		return
	}
	if isUniqueViolation(err) {
		WriteJSONError(w, r, http.StatusConflict, "conflict", "stream with the same name already exists in this project", map[string]interface{}{"company_id": companyID, "stream_id": streamID, "field": "name"})
		return
	}
	log.Printf("patch stream failed: %v", err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
}
