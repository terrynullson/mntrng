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
)

func (s *Server) handleEnqueueCheckJob(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	plannedAtRaw, plannedAt, ok := decodeEnqueuePlannedAt(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := insertCheckJob(ctx, s.db, companyID, streamID, plannedAt)
	if err != nil {
		handleEnqueueCheckJobError(w, r, companyID, streamID, plannedAtRaw, err)
		return
	}

	if err := WriteJSON(w, http.StatusAccepted, enqueueCheckJobResponse{Job: item}); err != nil {
		log.Printf("enqueue check job response encode error: %v", err)
	}
}

func (s *Server) handleGetCheckJob(w http.ResponseWriter, r *http.Request, companyID int64, jobID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := getCheckJobByID(ctx, s.db, companyID, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeCheckJobNotFound(w, r, companyID, jobID)
			return
		}
		log.Printf("get check job failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check job response encode error: %v", err)
	}
}

func (s *Server) handleListCheckJobs(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !ensureStreamExistsForJobs(ctx, w, r, s.db, companyID, streamID) {
		return
	}

	query, args, ok := buildListCheckJobsQuery(w, r, companyID, streamID)
	if !ok {
		return
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("list check jobs failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items, scanOK := scanCheckJobRows(w, r, rows)
	if !scanOK {
		return
	}

	if err := WriteJSON(w, http.StatusOK, checkJobListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list check jobs response encode error: %v", err)
	}
}

func decodeEnqueuePlannedAt(w http.ResponseWriter, r *http.Request) (string, time.Time, bool) {
	var request enqueueCheckJobRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return "", time.Time{}, false
	}

	plannedAtRaw := strings.TrimSpace(request.PlannedAt)
	if plannedAtRaw == "" {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "planned_at is required", map[string]interface{}{"field": "planned_at"})
		return "", time.Time{}, false
	}

	plannedAt, err := time.Parse(time.RFC3339, plannedAtRaw)
	if err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "planned_at must be RFC3339 timestamp", map[string]interface{}{"field": "planned_at", "value": plannedAtRaw})
		return "", time.Time{}, false
	}

	return plannedAtRaw, plannedAt, true
}

func insertCheckJob(ctx context.Context, db *sql.DB, companyID int64, streamID int64, plannedAt time.Time) (checkJob, error) {
	var item checkJob
	err := db.QueryRowContext(
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
	return item, err
}

func getCheckJobByID(ctx context.Context, db *sql.DB, companyID int64, jobID int64) (checkJob, error) {
	var item checkJob
	err := db.QueryRowContext(
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
	return item, err
}

func ensureStreamExistsForJobs(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB, companyID int64, streamID int64) bool {
	var streamExists int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM streams WHERE company_id = $1 AND id = $2`, companyID, streamID).Scan(&streamExists)
	if err == nil {
		return true
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeStreamMissingForCompany(w, r, companyID, streamID)
		return false
	}
	log.Printf("check stream existence for check jobs failed: %v", err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
	return false
}

func buildListCheckJobsQuery(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) (string, []interface{}, bool) {
	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if statusRaw := strings.TrimSpace(r.URL.Query().Get("status")); statusRaw != "" {
		status, ok := normalizeCheckJobStatus(statusRaw)
		if !ok {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid status filter", map[string]interface{}{"status": statusRaw, "allowed": []string{"queued", "running", "done", "failed"}})
			return "", nil, false
		}
		conditions = append(conditions, fmt.Sprintf("status = $%d", nextPlaceholder))
		args = append(args, status)
		nextPlaceholder++
	}

	if fromRaw := strings.TrimSpace(r.URL.Query().Get("from")); fromRaw != "" {
		fromTime, parseErr := time.Parse(time.RFC3339, fromRaw)
		if parseErr != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid from filter", map[string]interface{}{"from": fromRaw})
			return "", nil, false
		}
		conditions = append(conditions, fmt.Sprintf("planned_at >= $%d", nextPlaceholder))
		args = append(args, fromTime.UTC())
		nextPlaceholder++
	}

	if toRaw := strings.TrimSpace(r.URL.Query().Get("to")); toRaw != "" {
		toTime, parseErr := time.Parse(time.RFC3339, toRaw)
		if parseErr != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid to filter", map[string]interface{}{"to": toRaw})
			return "", nil, false
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
	return query, args, true
}

func scanCheckJobRows(w http.ResponseWriter, r *http.Request, rows *sql.Rows) ([]checkJob, bool) {
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
			WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return nil, false
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list check jobs rows failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return nil, false
	}
	return items, true
}

func handleEnqueueCheckJobError(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64, plannedAtRaw string, err error) {
	if errors.Is(err, sql.ErrNoRows) || isForeignKeyViolation(err) {
		writeStreamMissingForCompany(w, r, companyID, streamID)
		return
	}
	if isUniqueViolation(err) {
		WriteJSONError(w, r, http.StatusConflict, "conflict", "check job already exists for stream and planned_at", map[string]interface{}{"company_id": companyID, "stream_id": streamID, "planned_at": plannedAtRaw})
		return
	}
	log.Printf("enqueue check job failed: %v", err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
}

func writeCheckJobNotFound(w http.ResponseWriter, r *http.Request, companyID int64, jobID int64) {
	WriteJSONError(w, r, http.StatusNotFound, "not_found", "check job not found", map[string]interface{}{"company_id": companyID, "job_id": jobID})
}

func writeStreamMissingForCompany(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	WriteJSONError(w, r, http.StatusNotFound, "not_found", "stream not found for company", map[string]interface{}{"company_id": companyID, "stream_id": streamID})
}
