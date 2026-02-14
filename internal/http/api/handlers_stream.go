package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (s *Server) handleCreateStream(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	name, url, isActive, ok := decodeCreateStreamInput(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("create stream tx begin failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer tx.Rollback()

	item, err := insertStreamTx(ctx, tx, companyID, projectID, name, url, isActive)
	if err != nil {
		handleCreateStreamInsertError(w, r, companyID, projectID, err)
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("create stream tx commit failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create stream response encode error: %v", err)
	}
}

func (s *Server) handleListStreams(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	query, args, ok := buildListStreamsQuery(w, r, companyID)
	if !ok {
		return
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("list streams failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items, scanOK := scanStreamRows(w, r, rows)
	if !scanOK {
		return
	}

	if err := WriteJSON(w, http.StatusOK, streamListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list streams response encode error: %v", err)
	}
}

func (s *Server) handleGetStream(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := getStreamByID(ctx, s.db, companyID, streamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeStreamNotFound(w, r, companyID, streamID)
			return
		}
		log.Printf("get stream failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get stream response encode error: %v", err)
	}
}

func decodeCreateStreamInput(w http.ResponseWriter, r *http.Request) (string, string, bool, bool) {
	var request createStreamRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return "", "", false, false
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "name is required", map[string]interface{}{"field": "name"})
		return "", "", false, false
	}
	url := strings.TrimSpace(request.URL)
	if url == "" {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "url is required", map[string]interface{}{"field": "url"})
		return "", "", false, false
	}

	isActive := true
	if request.IsActive != nil {
		isActive = *request.IsActive
	}
	return name, url, isActive, true
}

func insertStreamTx(ctx context.Context, tx *sql.Tx, companyID int64, projectID int64, name string, url string, isActive bool) (stream, error) {
	var item stream
	err := tx.QueryRowContext(
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
	return item, err
}

func getStreamByID(ctx context.Context, db *sql.DB, companyID int64, streamID int64) (stream, error) {
	var item stream
	err := db.QueryRowContext(
		ctx,
		`SELECT id, company_id, project_id, name, url, is_active, created_at, updated_at
         FROM streams
         WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func buildListStreamsQuery(w http.ResponseWriter, r *http.Request, companyID int64) (string, []interface{}, bool) {
	args := []interface{}{companyID}
	conditions := []string{"company_id = $1"}
	nextPlaceholder := 2

	if projectIDRaw := strings.TrimSpace(r.URL.Query().Get("project_id")); projectIDRaw != "" {
		projectID, err := ParsePositiveID(projectIDRaw)
		if err != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid project_id filter", map[string]interface{}{"project_id": projectIDRaw})
			return "", nil, false
		}
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", nextPlaceholder))
		args = append(args, projectID)
		nextPlaceholder++
	}

	if isActiveRaw := strings.TrimSpace(r.URL.Query().Get("is_active")); isActiveRaw != "" {
		isActive, err := strconv.ParseBool(isActiveRaw)
		if err != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid is_active filter", map[string]interface{}{"is_active": isActiveRaw})
			return "", nil, false
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
	return query, args, true
}

func scanStreamRows(w http.ResponseWriter, r *http.Request, rows *sql.Rows) ([]stream, bool) {
	items := make([]stream, 0)
	for rows.Next() {
		var item stream
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			log.Printf("list streams scan failed: %v", err)
			WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return nil, false
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list streams rows failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return nil, false
	}
	return items, true
}

func handleCreateStreamInsertError(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64, err error) {
	if errors.Is(err, sql.ErrNoRows) || isForeignKeyViolation(err) {
		WriteJSONError(w, r, http.StatusNotFound, "not_found", "project not found for company", map[string]interface{}{"company_id": companyID, "project_id": projectID})
		return
	}
	if isUniqueViolation(err) {
		WriteJSONError(w, r, http.StatusConflict, "conflict", "stream with the same name already exists in this project", map[string]interface{}{"company_id": companyID, "project_id": projectID, "field": "name"})
		return
	}
	log.Printf("create stream failed: %v", err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
}

func writeStreamNotFound(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	WriteJSONError(w, r, http.StatusNotFound, "not_found", "stream not found", map[string]interface{}{"company_id": companyID, "stream_id": streamID})
}
