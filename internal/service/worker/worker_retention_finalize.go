package worker

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/example/hls-monitoring-platform/internal/domain"
	"github.com/lib/pq"
)

func (w *worker) runRetentionCleanup(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-w.retentionTTL)
	companyIDs, err := w.loadCompanyIDsForRetention(ctx)
	if err != nil {
		return err
	}

	for _, companyID := range companyIDs {
		affectedRows, deletedFiles, errorsCount, cleanupErr := w.cleanupCompanyRetention(ctx, companyID, cutoff)
		if cleanupErr != nil {
			return cleanupErr
		}
		log.Printf(
			"worker retention cleanup: company_id=%d affected_rows=%d deleted_files=%d errors_count=%d",
			companyID,
			affectedRows,
			deletedFiles,
			errorsCount,
		)
	}
	return nil
}

func (w *worker) loadCompanyIDsForRetention(ctx context.Context) ([]int64, error) {
	return w.retentionRepo.ListCompanyIDsForRetention(ctx)
}

func (w *worker) cleanupCompanyRetention(ctx context.Context, companyID int64, cutoff time.Time) (int, int, int, error) {
	affectedRows := 0
	deletedFiles := 0
	errorsCount := 0

	for {
		if err := ctx.Err(); err != nil {
			return affectedRows, deletedFiles, errorsCount, err
		}

		candidates, err := w.loadRetentionCandidates(ctx, companyID, cutoff, w.retentionCleanupBatchSize)
		if err != nil {
			return affectedRows, deletedFiles, errorsCount, err
		}
		if len(candidates) == 0 {
			return affectedRows, deletedFiles, errorsCount, nil
		}

		for _, candidate := range candidates {
			wasDeleted, fileErr := removeScreenshotFile(candidate.ScreenshotPath)
			if fileErr != nil {
				errorsCount++
				log.Printf(
					"worker retention cleanup file-delete error: company_id=%d check_result_id=%d reason=%s err=%v",
					companyID,
					candidate.ID,
					"file_delete_failed",
					fileErr,
				)
			}
			if wasDeleted {
				deletedFiles++
			}

			rowsAffected, err := w.retentionRepo.DeleteStaleCheckResult(ctx, companyID, candidate.ID, cutoff)
			if err != nil {
				return affectedRows, deletedFiles, errorsCount, err
			}
			affectedRows += int(rowsAffected)
		}

		if len(candidates) < w.retentionCleanupBatchSize {
			return affectedRows, deletedFiles, errorsCount, nil
		}
	}
}

func (w *worker) loadRetentionCandidates(ctx context.Context, companyID int64, cutoff time.Time, batchSize int) ([]retentionCandidate, error) {
	return w.retentionRepo.ListRetentionCandidates(ctx, companyID, cutoff, batchSize)
}

func removeScreenshotFile(path string) (bool, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return false, nil
	}

	fileInfo, statErr := os.Stat(cleanPath)
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return false, nil
		}
		return false, statErr
	}
	if fileInfo.IsDir() {
		return false, errors.New("screenshot path is a directory")
	}

	err := os.Remove(cleanPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (w *worker) finalizeWithRetry(ctx context.Context, job claimedJob, status string, errorMessage string) error {
	for attempt := 0; ; attempt++ {
		err := w.finalizeJob(ctx, job, status, errorMessage)
		if err == nil {
			return nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker finalize retry attempt=%d job_id=%d backoff=%s err=%v", attempt+1, job.ID, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func (w *worker) finalizeJob(ctx context.Context, job claimedJob, status string, errorMessage string) error {
	rowsAffected, err := w.jobRepo.FinalizeJob(ctx, job, status, errorMessage)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		log.Printf("worker finalize skipped (state already changed): id=%d company_id=%d target_status=%s", job.ID, job.CompanyID, status)
	}
	return nil
}

func isRetryableWorkerError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		errorClass := string(pqErr.Code.Class())
		return errorClass == "08" || errorClass == "53" || errorClass == "57"
	}

	return false
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func intAtLeast(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

func IntAtLeast(value int, minimum int) int {
	return intAtLeast(value, minimum)
}

func intInRange(value int, minimum int, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func IntInRange(value int, minimum int, maximum int) int {
	return intInRange(value, minimum, maximum)
}

func envFloat(key string, fallback float64) float64 {
	valueRaw := config.GetString(key, "")
	if valueRaw == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(valueRaw, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func EnvFloat(key string, fallback float64) float64 {
	return envFloat(key, fallback)
}

func floatAtLeast(value float64, minimum float64) float64 {
	if value < minimum {
		return minimum
	}
	return value
}

func FloatAtLeast(value float64, minimum float64) float64 {
	return floatAtLeast(value, minimum)
}

func floatInRange(value float64, minimum float64, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func FloatInRange(value float64, minimum float64, maximum float64) float64 {
	return floatInRange(value, minimum, maximum)
}

func envBool(key string, fallback bool) bool {
	valueRaw := strings.TrimSpace(strings.ToLower(config.GetString(key, "")))
	if valueRaw == "" {
		return fallback
	}
	switch valueRaw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func EnvBool(key string, fallback bool) bool {
	return envBool(key, fallback)
}

func checkStatusToDBStatus(status string) string {
	switch strings.TrimSpace(strings.ToUpper(status)) {
	case domain.WorkerStatusOK:
		return domain.WorkerStatusDBOK
	case domain.WorkerStatusWarn:
		return domain.WorkerStatusDBWarn
	case domain.WorkerStatusFail:
		return domain.WorkerStatusDBFail
	default:
		return strings.TrimSpace(strings.ToLower(status))
	}
}

func normalizeAlertStatus(statusRaw string) (string, error) {
	normalized := checkStatusToDBStatus(statusRaw)
	switch normalized {
	case domain.WorkerStatusDBOK, domain.WorkerStatusDBWarn, domain.WorkerStatusDBFail:
		return normalized, nil
	default:
		return "", errors.New("unsupported alert status: " + statusRaw)
	}
}
