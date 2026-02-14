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
			WriteJSONError(
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

	var streamExists int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM streams WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&streamExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	args := []interface{}{companyID, streamID}
	conditions := []string{"company_id = $1", "stream_id = $2"}
	nextPlaceholder := 3

	if statusRaw := strings.TrimSpace(r.URL.Query().Get("status")); statusRaw != "" {
		status, ok := normalizeCheckResultStatus(statusRaw)
		if !ok {
			WriteJSONError(
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
			WriteJSONError(
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}
	defer rows.Close()

	items := make([]checkResult, 0)
	for rows.Next() {
		item, scanErr := scanCheckResult(rows)
		if scanErr != nil {
			log.Printf("list check results scan failed: %v", scanErr)
			WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list check results rows failed: %v", err)
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	response := checkResultListResponse{
		Items:      items,
		NextCursor: nil,
	}
	if err := WriteJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list check results response encode error: %v", err)
	}
}

func (s *Server) handleGetCheckResultByJob(w http.ResponseWriter, r *http.Request, companyID int64, jobID int64) {
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
			WriteJSONError(
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
		WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get check result by job response encode error: %v", err)
	}
}
