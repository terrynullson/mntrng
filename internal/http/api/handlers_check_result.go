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

func (s *Server) handleGetCheckResult(w http.ResponseWriter, r *http.Request, companyID int64, resultID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := getCheckResultByID(ctx, s.db, companyID, resultID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeCheckResultNotFound(w, r, companyID, resultID)
			return
		}
		log.Printf("get check result failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check result response encode error: %v", err)
	}
}

func (s *Server) handleListCheckResults(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !ensureStreamExistsForResults(ctx, w, r, s.db, companyID, streamID) {
		return
	}

	query, args, ok := buildListCheckResultsQuery(w, r, companyID, streamID)
	if !ok {
		return
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("list check results failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items, scanOK := scanCheckResultRows(w, r, rows)
	if !scanOK {
		return
	}

	if err := WriteJSON(w, http.StatusOK, checkResultListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list check results response encode error: %v", err)
	}
}

func (s *Server) handleGetCheckResultByJob(w http.ResponseWriter, r *http.Request, companyID int64, jobID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := getCheckResultByJobID(ctx, s.db, companyID, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			WriteJSONError(w, r, http.StatusNotFound, "not_found", "check result not found for job", map[string]interface{}{"company_id": companyID, "job_id": jobID})
			return
		}
		log.Printf("get check result by job failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check result by job response encode error: %v", err)
	}
}

func getCheckResultByID(ctx context.Context, db *sql.DB, companyID int64, resultID int64) (checkResult, error) {
	row := db.QueryRowContext(
		ctx,
		`SELECT id, company_id, job_id, stream_id, status, checks, screenshot_path, created_at
         FROM check_results
         WHERE company_id = $1 AND id = $2`,
		companyID,
		resultID,
	)
	return scanCheckResult(row)
}

func getCheckResultByJobID(ctx context.Context, db *sql.DB, companyID int64, jobID int64) (checkResult, error) {
	row := db.QueryRowContext(
		ctx,
		`SELECT id, company_id, job_id, stream_id, status, checks, screenshot_path, created_at
         FROM check_results
         WHERE company_id = $1 AND job_id = $2`,
		companyID,
		jobID,
	)
	return scanCheckResult(row)
}

func ensureStreamExistsForResults(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB, companyID int64, streamID int64) bool {
	var streamExists int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM streams WHERE company_id = $1 AND id = $2`, companyID, streamID).Scan(&streamExists)
	if err == nil {
		return true
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeStreamMissingForCompany(w, r, companyID, streamID)
		return false
	}
	log.Printf("check stream existence for check results failed: %v", err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
	return false
}

func buildListCheckResultsQuery(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) (string, []interface{}, bool) {
	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if statusRaw := strings.TrimSpace(r.URL.Query().Get("status")); statusRaw != "" {
		status, ok := normalizeCheckResultStatus(statusRaw)
		if !ok {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid status filter", map[string]interface{}{"status": statusRaw, "allowed": []string{"OK", "WARN", "FAIL"}})
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
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", nextPlaceholder))
		args = append(args, fromTime.UTC())
		nextPlaceholder++
	}

	if toRaw := strings.TrimSpace(r.URL.Query().Get("to")); toRaw != "" {
		toTime, parseErr := time.Parse(time.RFC3339, toRaw)
		if parseErr != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid to filter", map[string]interface{}{"to": toRaw})
			return "", nil, false
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
	return query, args, true
}

func scanCheckResultRows(w http.ResponseWriter, r *http.Request, rows *sql.Rows) ([]checkResult, bool) {
	items := make([]checkResult, 0)
	for rows.Next() {
		item, scanErr := scanCheckResult(rows)
		if scanErr != nil {
			log.Printf("list check results scan failed: %v", scanErr)
			WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return nil, false
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list check results rows failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return nil, false
	}
	return items, true
}

func writeCheckResultNotFound(w http.ResponseWriter, r *http.Request, companyID int64, resultID int64) {
	WriteJSONError(w, r, http.StatusNotFound, "not_found", "check result not found", map[string]interface{}{"company_id": companyID, "result_id": resultID})
}
