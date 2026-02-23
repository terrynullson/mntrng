package api

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/lib/pq"
)

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
