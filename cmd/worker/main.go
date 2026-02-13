package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/lib/pq"
)

type claimedJob struct {
	ID        int64
	CompanyID int64
	StreamID  int64
	PlannedAt time.Time
}

type worker struct {
	db                  *sql.DB
	pollInterval        time.Duration
	jobTimeout          time.Duration
	playlistTimeout     time.Duration
	segmentTimeout      time.Duration
	segmentsSampleCount int
	freshnessWarn       time.Duration
	freshnessFail       time.Duration
	freezeWarn          time.Duration
	freezeFail          time.Duration
	blackframeWarnRatio float64
	blackframeFailRatio float64
	effectiveWarnRatio  float64
	effectiveFailRatio  float64
	alertFailStreak     int
	alertCooldown       time.Duration
	alertSendRecovered  bool
	retryMax            int
	retryBackoff        time.Duration
}

type checkJobEvaluation struct {
	DBStatus  string
	Aggregate string
	Checks    map[string]interface{}
}

type playlistSegment struct {
	URL         string
	DurationSec float64
}

type segmentSample struct {
	URL         string
	Downloaded  bool
	DurationSec float64
	Bytes       int64
}

type declaredBitrateResult struct {
	Status      string
	DeclaredBPS int64
	Details     map[string]interface{}
}

var blackframeEventPattern = regexp.MustCompile(`frame:\s*\d+\s+pblack:\s*\d+`)

type alertDecision struct {
	ShouldSend     bool
	EventType      string
	Reason         string
	CurrentStatus  string
	PreviousStatus string
	FailStreak     int
	CooldownUntil  *time.Time
}

func main() {
	pollInterval := time.Duration(intAtLeast(config.GetInt("WORKER_HEARTBEAT_SEC", 15), 1)) * time.Second
	jobTimeout := time.Duration(intAtLeast(config.GetInt("WORKER_JOB_TIMEOUT_SEC", 30), 1)) * time.Second
	playlistTimeout := time.Duration(intAtLeast(config.GetInt("PLAYLIST_TIMEOUT_MS", 3000), 1)) * time.Millisecond
	segmentTimeout := time.Duration(intAtLeast(config.GetInt("SEGMENT_TIMEOUT_MS", 5000), 1)) * time.Millisecond
	segmentsSampleCount := intInRange(config.GetInt("SEGMENTS_SAMPLE_COUNT", 3), 3, 5)
	freshnessWarn := time.Duration(intAtLeast(config.GetInt("FRESHNESS_WARN_SEC", 10), 1)) * time.Second
	freshnessFail := time.Duration(intAtLeast(config.GetInt("FRESHNESS_FAIL_SEC", 30), 1)) * time.Second
	freezeWarn := time.Duration(intAtLeast(config.GetInt("FREEZE_WARN_SEC", 2), 1)) * time.Second
	freezeFail := time.Duration(intAtLeast(config.GetInt("FREEZE_FAIL_SEC", 5), 1)) * time.Second
	blackframeWarnRatio := floatInRange(envFloat("BLACKFRAME_WARN_RATIO", 0.9), 0, 1)
	blackframeFailRatio := floatInRange(envFloat("BLACKFRAME_FAIL_RATIO", 0.98), 0, 1)
	effectiveWarnRatio := floatAtLeast(envFloat("EFFECTIVE_BITRATE_WARN_RATIO", 0.7), 0)
	effectiveFailRatio := floatAtLeast(envFloat("EFFECTIVE_BITRATE_FAIL_RATIO", 0.4), 0)
	alertFailStreak := intAtLeast(config.GetInt("ALERT_FAIL_STREAK", 2), 1)
	alertCooldown := time.Duration(intAtLeast(config.GetInt("ALERT_COOLDOWN_MIN", 10), 1)) * time.Minute
	alertSendRecovered := envBool("ALERT_SEND_RECOVERED", false)
	if freshnessFail < freshnessWarn {
		freshnessFail = freshnessWarn
	}
	if freezeFail < freezeWarn {
		freezeFail = freezeWarn
	}
	if blackframeFailRatio < blackframeWarnRatio {
		blackframeFailRatio = blackframeWarnRatio
	}
	retryMax := intAtLeast(config.GetInt("WORKER_DB_RETRY_MAX", 2), 0)
	retryBackoff := time.Duration(intAtLeast(config.GetInt("WORKER_DB_RETRY_BACKOFF_MS", 500), 1)) * time.Millisecond
	databaseURL := config.GetString("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w := &worker{
		db:                  db,
		pollInterval:        pollInterval,
		jobTimeout:          jobTimeout,
		playlistTimeout:     playlistTimeout,
		segmentTimeout:      segmentTimeout,
		segmentsSampleCount: segmentsSampleCount,
		freshnessWarn:       freshnessWarn,
		freshnessFail:       freshnessFail,
		freezeWarn:          freezeWarn,
		freezeFail:          freezeFail,
		blackframeWarnRatio: blackframeWarnRatio,
		blackframeFailRatio: blackframeFailRatio,
		effectiveWarnRatio:  effectiveWarnRatio,
		effectiveFailRatio:  effectiveFailRatio,
		alertFailStreak:     alertFailStreak,
		alertCooldown:       alertCooldown,
		alertSendRecovered:  alertSendRecovered,
		retryMax:            retryMax,
		retryBackoff:        retryBackoff,
	}

	log.Printf(
		"worker skeleton started: poll_interval=%s, job_timeout=%s, playlist_timeout=%s, segment_timeout=%s, segments_sample_count=%d, freshness_warn=%s, freshness_fail=%s, freeze_warn=%s, freeze_fail=%s, blackframe_warn_ratio=%.2f, blackframe_fail_ratio=%.2f, effective_warn_ratio=%.2f, effective_fail_ratio=%.2f, alert_fail_streak=%d, alert_cooldown=%s, alert_send_recovered=%t, retry_max=%d, retry_backoff=%s",
		w.pollInterval,
		w.jobTimeout,
		w.playlistTimeout,
		w.segmentTimeout,
		w.segmentsSampleCount,
		w.freshnessWarn,
		w.freshnessFail,
		w.freezeWarn,
		w.freezeFail,
		w.blackframeWarnRatio,
		w.blackframeFailRatio,
		w.effectiveWarnRatio,
		w.effectiveFailRatio,
		w.alertFailStreak,
		w.alertCooldown,
		w.alertSendRecovered,
		w.retryMax,
		w.retryBackoff,
	)

	if err := w.processCycleWithRetry(ctx); err != nil {
		log.Printf("worker cycle failed: %v", err)
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("worker skeleton stopped")
			return
		case currentTime := <-ticker.C:
			log.Printf("worker skeleton heartbeat: %s", currentTime.UTC().Format(time.RFC3339))
			if err := w.processCycleWithRetry(ctx); err != nil {
				log.Printf("worker cycle failed: %v", err)
			}
		}
	}
}

func (w *worker) processCycleWithRetry(ctx context.Context) error {
	for attempt := 0; ; attempt++ {
		err := w.processSingleJobCycle(ctx)
		if err == nil {
			return nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker retry attempt=%d backoff=%s err=%v", attempt+1, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func (w *worker) processSingleJobCycle(ctx context.Context) error {
	job, ok, err := w.claimNextQueuedJob(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	log.Printf(
		"worker claimed job: id=%d company_id=%d stream_id=%d planned_at=%s",
		job.ID,
		job.CompanyID,
		job.StreamID,
		job.PlannedAt.UTC().Format(time.RFC3339),
	)

	evaluation, processErr := w.processJob(ctx, job)
	if processErr != nil {
		if finalizeErr := w.finalizeWithRetry(ctx, job, "failed", processErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, processErr.Error())
		return nil
	}

	if persistErr := w.persistCheckResultWithRetry(ctx, job, evaluation); persistErr != nil {
		if finalizeErr := w.finalizeWithRetry(ctx, job, "failed", persistErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, persistErr.Error())
		return nil
	}

	alertDecision, alertErr := w.applyAlertStateWithRetry(ctx, job, evaluation.DBStatus)
	if alertErr != nil {
		if finalizeErr := w.finalizeWithRetry(ctx, job, "failed", alertErr.Error()); finalizeErr != nil {
			return finalizeErr
		}
		log.Printf("worker finalized job as failed: id=%d company_id=%d reason=%s", job.ID, job.CompanyID, alertErr.Error())
		return nil
	}
	w.logAlertDecision(job, alertDecision)

	if finalizeErr := w.finalizeWithRetry(ctx, job, "done", ""); finalizeErr != nil {
		return finalizeErr
	}
	log.Printf("worker finalized job as done: id=%d company_id=%d", job.ID, job.CompanyID)
	return nil
}

func (w *worker) claimNextQueuedJob(ctx context.Context) (claimedJob, bool, error) {
	row := w.db.QueryRowContext(
		ctx,
		`WITH candidate AS (
             SELECT id, company_id
             FROM check_jobs
             WHERE status = 'queued'
             ORDER BY planned_at ASC, id ASC
             FOR UPDATE SKIP LOCKED
             LIMIT 1
         )
         UPDATE check_jobs AS j
         SET status = 'running',
             started_at = NOW(),
             finished_at = NULL,
             error_message = NULL
         FROM candidate AS c
         WHERE j.id = c.id
           AND j.company_id = c.company_id
           AND j.status = 'queued'
         RETURNING j.id, j.company_id, j.stream_id, j.planned_at`,
	)

	var job claimedJob
	err := row.Scan(&job.ID, &job.CompanyID, &job.StreamID, &job.PlannedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return claimedJob{}, false, nil
		}
		return claimedJob{}, false, err
	}
	return job, true, nil
}

func (w *worker) processJob(ctx context.Context, job claimedJob) (checkJobEvaluation, error) {
	jobCtx, cancel := context.WithTimeout(ctx, w.jobTimeout)
	defer cancel()

	streamURL, err := w.loadStreamURL(jobCtx, job.CompanyID, job.StreamID)
	if err != nil {
		return checkJobEvaluation{}, err
	}

	playlistBody, playlistErr := w.fetchPlaylist(jobCtx, streamURL)
	playlistStatus := "OK"
	freshnessStatus := "FAIL"
	segmentsStatus := "FAIL"
	freezeStatus := "FAIL"
	blackframeStatus := "WARN"
	declaredBitrate := declaredBitrateResult{
		Status: "FAIL",
		Details: map[string]interface{}{
			"reason": "playlist_unavailable",
		},
	}
	freezeDetails := map[string]interface{}{
		"max_freeze_sec": w.freezeFail.Seconds(),
		"reason":         "playlist_unavailable",
		"source":         "playlist_http",
	}
	blackframeDetails := map[string]interface{}{
		"dark_frame_ratio": 0.0,
		"analyzed_frames":  0,
		"reason":           "playlist_unavailable",
		"source":           "ffmpeg_blackframe",
	}
	effectiveBitrateStatus := "FAIL"
	effectiveBitrateDetails := map[string]interface{}{
		"calculated_bps": 0.0,
		"declared_bps":   int64(0),
		"ratio":          nil,
		"reason":         "playlist_unavailable",
		"sample_count":   0,
	}

	if playlistErr != nil {
		playlistStatus = "FAIL"
	} else {
		freshnessStatus = w.checkFreshness(playlistBody)
		freezeStatus, freezeDetails = w.checkFreeze(playlistBody)

		segments, segmentParseErr := extractLatestPlaylistSegments(streamURL, playlistBody, w.segmentsSampleCount)
		segmentSamples := make([]segmentSample, 0)
		if segmentParseErr != nil {
			segmentsStatus = "FAIL"
			blackframeStatus = "WARN"
			blackframeDetails = map[string]interface{}{
				"dark_frame_ratio": 0.0,
				"analyzed_frames":  0,
				"reason":           "segments_not_available",
				"source":           "ffmpeg_blackframe",
			}
			effectiveBitrateStatus = "FAIL"
			effectiveBitrateDetails = map[string]interface{}{
				"calculated_bps": 0.0,
				"declared_bps":   int64(0),
				"ratio":          nil,
				"reason":         "segments_not_available",
				"sample_count":   0,
			}
		} else {
			segmentsStatus, segmentSamples = w.checkSegmentsAvailability(jobCtx, segments)
			blackframeStatus, blackframeDetails = w.checkBlackframe(jobCtx, segmentSamples)
		}

		declaredBitrate = checkDeclaredBitrate(playlistBody)
		if segmentParseErr == nil {
			effectiveBitrateStatus, effectiveBitrateDetails = w.checkEffectiveBitrate(segmentSamples, declaredBitrate)
		}
	}

	aggregate := aggregateStatuses(playlistStatus, freshnessStatus, segmentsStatus, freezeStatus, blackframeStatus, declaredBitrate.Status, effectiveBitrateStatus)

	return checkJobEvaluation{
		DBStatus:  strings.ToLower(aggregate),
		Aggregate: aggregate,
		Checks: map[string]interface{}{
			"playlist":                  playlistStatus,
			"freshness":                 freshnessStatus,
			"segments":                  segmentsStatus,
			"freeze":                    freezeStatus,
			"freeze_details":            freezeDetails,
			"blackframe":                blackframeStatus,
			"blackframe_details":        blackframeDetails,
			"declared_bitrate":          declaredBitrate.Status,
			"declared_bitrate_details":  declaredBitrate.Details,
			"effective_bitrate":         effectiveBitrateStatus,
			"effective_bitrate_details": effectiveBitrateDetails,
		},
	}, nil
}

func (w *worker) loadStreamURL(ctx context.Context, companyID int64, streamID int64) (string, error) {
	var streamURL string
	err := w.db.QueryRowContext(
		ctx,
		`SELECT url
         FROM streams
         WHERE company_id = $1
           AND id = $2`,
		companyID,
		streamID,
	).Scan(&streamURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("stream not found in tenant context")
		}
		return "", err
	}
	streamURL = strings.TrimSpace(streamURL)
	if streamURL == "" {
		return "", errors.New("stream url is empty")
	}
	return streamURL, nil
}

func (w *worker) fetchPlaylist(ctx context.Context, streamURL string) (string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, w.playlistTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", errors.New("playlist request returned non-2xx")
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	playlistBody := string(body)
	if !strings.Contains(playlistBody, "#EXTM3U") {
		return "", errors.New("playlist does not contain EXTM3U marker")
	}

	return playlistBody, nil
}

func (w *worker) checkFreshness(playlistBody string) string {
	lastProgramDateTime, ok := extractLatestProgramDateTime(playlistBody)
	if !ok {
		return "FAIL"
	}

	age := time.Since(lastProgramDateTime)
	if age < 0 {
		age = 0
	}

	if age >= w.freshnessFail {
		return "FAIL"
	}
	if age >= w.freshnessWarn {
		return "WARN"
	}
	return "OK"
}

func (w *worker) checkFreeze(playlistBody string) (string, map[string]interface{}) {
	lastProgramDateTime, ok := extractLatestProgramDateTime(playlistBody)
	if !ok {
		return freezeStatusByThreshold(0, w.freezeWarn.Seconds(), w.freezeFail.Seconds()), map[string]interface{}{
			"max_freeze_sec": 0.0,
			"reason":         "program_date_time_not_found",
			"source":         "playlist_ext_x_program_date_time",
		}
	}

	maxFreezeSec := time.Since(lastProgramDateTime).Seconds()
	if maxFreezeSec < 0 {
		maxFreezeSec = 0
	}

	status := freezeStatusByThreshold(maxFreezeSec, w.freezeWarn.Seconds(), w.freezeFail.Seconds())
	reason := "within_threshold"
	if status == "WARN" {
		reason = "freeze_warn_threshold_reached"
	}
	if status == "FAIL" {
		reason = "freeze_fail_threshold_reached"
	}

	return status, map[string]interface{}{
		"max_freeze_sec": maxFreezeSec,
		"reason":         reason,
		"source":         "playlist_ext_x_program_date_time",
	}
}

func (w *worker) checkBlackframe(ctx context.Context, samples []segmentSample) (string, map[string]interface{}) {
	downloadedCount := 0
	totalAnalyzedFrames := 0
	totalDarkFrames := 0

	for _, sample := range samples {
		if !sample.Downloaded || strings.TrimSpace(sample.URL) == "" {
			continue
		}
		downloadedCount++

		analyzeCtx, cancel := context.WithTimeout(ctx, w.segmentTimeout)
		darkFrames, analyzedFrames, err := analyzeBlackframeForSegment(analyzeCtx, sample.URL)
		cancel()
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return "WARN", map[string]interface{}{
					"dark_frame_ratio": 0.0,
					"analyzed_frames":  0,
					"reason":           "blackframe_analyzer_not_available",
					"source":           "ffmpeg_blackframe",
				}
			}
			continue
		}

		totalDarkFrames += darkFrames
		totalAnalyzedFrames += analyzedFrames
	}

	if downloadedCount == 0 {
		return "WARN", map[string]interface{}{
			"dark_frame_ratio": 0.0,
			"analyzed_frames":  0,
			"reason":           "no_downloaded_segments",
			"source":           "ffmpeg_blackframe",
		}
	}

	if totalAnalyzedFrames == 0 {
		return "WARN", map[string]interface{}{
			"dark_frame_ratio": 0.0,
			"analyzed_frames":  0,
			"reason":           "blackframe_analysis_failed",
			"source":           "ffmpeg_blackframe",
		}
	}

	darkFrameRatio := float64(totalDarkFrames) / float64(totalAnalyzedFrames)
	status := blackframeStatusByThreshold(darkFrameRatio, w.blackframeWarnRatio, w.blackframeFailRatio)
	reason := "within_threshold"
	if status == "WARN" {
		reason = "blackframe_warn_threshold_reached"
	}
	if status == "FAIL" {
		reason = "blackframe_fail_threshold_reached"
	}

	return status, map[string]interface{}{
		"dark_frame_ratio": darkFrameRatio,
		"analyzed_frames":  totalAnalyzedFrames,
		"reason":           reason,
		"source":           "ffmpeg_blackframe",
	}
}

func analyzeBlackframeForSegment(ctx context.Context, segmentURL string) (int, int, error) {
	analyzedFrames, err := probeVideoFrameCount(ctx, segmentURL)
	if err != nil {
		return 0, 0, err
	}

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-v", "info",
		"-hide_banner",
		"-nostats",
		"-i", segmentURL,
		"-vf", "blackframe",
		"-an",
		"-f", "null",
		"-",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return 0, 0, err
	}

	darkFrames := len(blackframeEventPattern.FindAll(output, -1))
	return darkFrames, analyzedFrames, nil
}

func probeVideoFrameCount(ctx context.Context, segmentURL string) (int, error) {
	command := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "error",
		"-count_frames",
		"-select_streams", "v:0",
		"-show_entries", "stream=nb_read_frames",
		"-of", "default=noprint_wrappers=1:nokey=1",
		segmentURL,
	)
	output, err := command.Output()
	if err != nil {
		return 0, err
	}

	value := strings.TrimSpace(string(output))
	if value == "" || strings.EqualFold(value, "N/A") {
		return 0, errors.New("ffprobe returned empty frame count")
	}

	frameCount, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if frameCount <= 0 {
		return 0, errors.New("ffprobe returned non-positive frame count")
	}
	return frameCount, nil
}

func (w *worker) checkSegmentsAvailability(ctx context.Context, segments []playlistSegment) (string, []segmentSample) {
	if len(segments) == 0 {
		return "FAIL", nil
	}

	availableCount := 0
	samples := make([]segmentSample, 0, len(segments))
	for _, segment := range segments {
		requestCtx, cancel := context.WithTimeout(ctx, w.segmentTimeout)
		bytesRead, err := downloadSegmentBytes(requestCtx, segment.URL)
		cancel()

		sample := segmentSample{
			URL:         segment.URL,
			Downloaded:  err == nil,
			DurationSec: segment.DurationSec,
			Bytes:       bytesRead,
		}
		samples = append(samples, sample)
		if sample.Downloaded {
			availableCount++
		}
	}

	if availableCount == len(segments) {
		return "OK", samples
	}
	if availableCount == 0 {
		return "FAIL", samples
	}
	return "WARN", samples
}

func downloadSegmentBytes(ctx context.Context, resourceURL string) (int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err != nil {
		return 0, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, errors.New("resource request returned non-2xx")
	}

	return io.Copy(io.Discard, response.Body)
}

func extractLatestPlaylistSegments(playlistURL string, playlistBody string, sampleCount int) ([]playlistSegment, error) {
	if sampleCount <= 0 {
		return nil, errors.New("segments sample count must be positive")
	}

	baseURL, err := url.Parse(playlistURL)
	if err != nil {
		return nil, err
	}

	segments := extractPlaylistSegments(playlistBody)
	if len(segments) == 0 {
		return nil, errors.New("no segment references found in playlist")
	}

	startIndex := len(segments) - sampleCount
	if startIndex < 0 {
		startIndex = 0
	}

	resolvedSegments := make([]playlistSegment, 0, len(segments)-startIndex)
	for _, segment := range segments[startIndex:] {
		parsedReference, parseErr := url.Parse(segment.URL)
		if parseErr != nil {
			return nil, parseErr
		}
		resolvedURL := baseURL.ResolveReference(parsedReference)
		resolvedSegments = append(resolvedSegments, playlistSegment{
			URL:         resolvedURL.String(),
			DurationSec: segment.DurationSec,
		})
	}

	return resolvedSegments, nil
}

func extractPlaylistSegments(playlistBody string) []playlistSegment {
	lines := strings.Split(playlistBody, "\n")
	segments := make([]playlistSegment, 0, len(lines))
	expectSegmentURI := false
	currentDuration := 0.0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXTINF:") {
			durationValue := strings.TrimPrefix(line, "#EXTINF:")
			if commaIndex := strings.Index(durationValue, ","); commaIndex >= 0 {
				durationValue = durationValue[:commaIndex]
			}
			parsedDuration, err := strconv.ParseFloat(strings.TrimSpace(durationValue), 64)
			if err == nil && parsedDuration > 0 {
				currentDuration = parsedDuration
			} else {
				currentDuration = 0
			}
			expectSegmentURI = true
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if expectSegmentURI {
			segments = append(segments, playlistSegment{
				URL:         line,
				DurationSec: currentDuration,
			})
			expectSegmentURI = false
			currentDuration = 0
		}
	}

	return segments
}

func extractLatestProgramDateTime(playlist string) (time.Time, bool) {
	lines := strings.Split(playlist, "\n")
	var latest time.Time
	found := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:") {
			continue
		}

		value := strings.TrimSpace(strings.TrimPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:"))
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			continue
		}

		if !found || parsed.After(latest) {
			latest = parsed
			found = true
		}
	}

	return latest, found
}

func checkDeclaredBitrate(playlistBody string) declaredBitrateResult {
	lines := strings.Split(playlistBody, "\n")
	streamInfoCount := 0
	invalidCount := 0
	missingAttributeCount := 0
	declaredValues := make([]int64, 0, 4)
	usedAverageBandwidth := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			continue
		}

		streamInfoCount++
		attributes := parseM3U8Attributes(strings.TrimPrefix(line, "#EXT-X-STREAM-INF:"))

		if valueRaw, ok := attributes["BANDWIDTH"]; ok {
			value, err := strconv.ParseInt(valueRaw, 10, 64)
			if err != nil || value <= 0 {
				invalidCount++
				continue
			}
			declaredValues = append(declaredValues, value)
			continue
		}

		if valueRaw, ok := attributes["AVERAGE-BANDWIDTH"]; ok {
			value, err := strconv.ParseInt(valueRaw, 10, 64)
			if err != nil || value <= 0 {
				invalidCount++
				continue
			}
			declaredValues = append(declaredValues, value)
			usedAverageBandwidth = true
			continue
		}

		missingAttributeCount++
	}

	if len(declaredValues) == 0 {
		if streamInfoCount == 0 {
			return declaredBitrateResult{
				Status:      "WARN",
				DeclaredBPS: 0,
				Details: map[string]interface{}{
					"reason": "no_stream_inf_tags",
				},
			}
		}
		if invalidCount > 0 {
			return declaredBitrateResult{
				Status:      "FAIL",
				DeclaredBPS: 0,
				Details: map[string]interface{}{
					"reason":          "invalid_declared_bitrate",
					"invalid_entries": invalidCount,
				},
			}
		}
		return declaredBitrateResult{
			Status:      "WARN",
			DeclaredBPS: 0,
			Details: map[string]interface{}{
				"reason":                  "declared_bitrate_not_present",
				"stream_info_entries":     streamInfoCount,
				"missing_attribute_count": missingAttributeCount,
			},
		}
	}

	maxDeclared := declaredValues[0]
	for _, value := range declaredValues[1:] {
		if value > maxDeclared {
			maxDeclared = value
		}
	}

	bitrateSource := "bandwidth"
	if usedAverageBandwidth {
		bitrateSource = "average_bandwidth"
	}

	return declaredBitrateResult{
		Status:      "OK",
		DeclaredBPS: maxDeclared,
		Details: map[string]interface{}{
			"parsed_bitrate_bps":  maxDeclared,
			"variants_considered": len(declaredValues),
			"source":              bitrateSource,
		},
	}
}

func parseM3U8Attributes(attributesRaw string) map[string]string {
	attributes := make(map[string]string)
	parts := strings.Split(attributesRaw, ",")
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		keyValue := strings.SplitN(item, "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		key := strings.TrimSpace(keyValue[0])
		value := strings.Trim(strings.TrimSpace(keyValue[1]), "\"")
		if key == "" || value == "" {
			continue
		}
		attributes[strings.ToUpper(key)] = value
	}
	return attributes
}

func (w *worker) checkEffectiveBitrate(samples []segmentSample, declared declaredBitrateResult) (string, map[string]interface{}) {
	availableSamples := 0
	totalBytes := int64(0)
	totalDurationSec := 0.0

	for _, sample := range samples {
		if !sample.Downloaded {
			continue
		}
		availableSamples++
		totalBytes += sample.Bytes
		totalDurationSec += sample.DurationSec
	}

	details := map[string]interface{}{
		"calculated_bps": 0.0,
		"declared_bps":   declared.DeclaredBPS,
		"ratio":          nil,
		"reason":         "",
		"sample_count":   availableSamples,
	}

	if availableSamples == 0 {
		details["reason"] = "no_downloaded_segments"
		return "FAIL", details
	}

	if totalDurationSec <= 0 {
		details["reason"] = "invalid_segment_duration"
		return "FAIL", details
	}

	calculatedBPS := (float64(totalBytes) * 8.0) / totalDurationSec
	details["calculated_bps"] = calculatedBPS

	if declared.DeclaredBPS <= 0 {
		details["reason"] = "declared_bitrate_unavailable"
		return "WARN", details
	}

	ratio := calculatedBPS / float64(declared.DeclaredBPS)
	details["ratio"] = ratio

	if ratio < w.effectiveFailRatio {
		details["reason"] = "ratio_below_fail_threshold"
		return "FAIL", details
	}
	if ratio < w.effectiveWarnRatio {
		details["reason"] = "ratio_below_warn_threshold"
		return "WARN", details
	}

	details["reason"] = "within_threshold"
	return "OK", details
}

func freezeStatusByThreshold(maxFreezeSec float64, warnSec float64, failSec float64) string {
	if maxFreezeSec >= failSec {
		return "FAIL"
	}
	if maxFreezeSec >= warnSec {
		return "WARN"
	}
	return "OK"
}

func blackframeStatusByThreshold(darkFrameRatio float64, warnRatio float64, failRatio float64) string {
	if darkFrameRatio >= failRatio {
		return "FAIL"
	}
	if darkFrameRatio >= warnRatio {
		return "WARN"
	}
	return "OK"
}

func aggregateStatuses(statuses ...string) string {
	hasWarn := false
	for _, status := range statuses {
		switch status {
		case "FAIL":
			return "FAIL"
		case "WARN":
			hasWarn = true
		}
	}
	if hasWarn {
		return "WARN"
	}
	return "OK"
}

func (w *worker) persistCheckResultWithRetry(ctx context.Context, job claimedJob, evaluation checkJobEvaluation) error {
	for attempt := 0; ; attempt++ {
		err := w.persistCheckResult(ctx, job, evaluation)
		if err == nil {
			return nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker persist retry attempt=%d job_id=%d backoff=%s err=%v", attempt+1, job.ID, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func (w *worker) persistCheckResult(ctx context.Context, job claimedJob, evaluation checkJobEvaluation) error {
	checksJSON, err := json.Marshal(evaluation.Checks)
	if err != nil {
		return err
	}

	result, err := w.db.ExecContext(
		ctx,
		`INSERT INTO check_results (company_id, job_id, stream_id, status, checks)
         VALUES ($1, $2, $3, $4, $5::jsonb)
         ON CONFLICT (job_id) DO NOTHING`,
		job.CompanyID,
		job.ID,
		job.StreamID,
		evaluation.DBStatus,
		string(checksJSON),
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		var existingCount int
		countErr := w.db.QueryRowContext(
			ctx,
			`SELECT COUNT(1)
             FROM check_results
             WHERE company_id = $1
               AND job_id = $2
               AND stream_id = $3`,
			job.CompanyID,
			job.ID,
			job.StreamID,
		).Scan(&existingCount)
		if countErr != nil {
			return countErr
		}
		if existingCount == 0 {
			return errors.New("check_result conflict without matching tenant row")
		}
	}

	log.Printf(
		"worker stored check_result: job_id=%d company_id=%d stream_id=%d status=%s checks=%v",
		job.ID,
		job.CompanyID,
		job.StreamID,
		evaluation.Aggregate,
		evaluation.Checks,
	)
	return nil
}

func (w *worker) applyAlertStateWithRetry(ctx context.Context, job claimedJob, resultStatus string) (alertDecision, error) {
	for attempt := 0; ; attempt++ {
		decision, err := w.applyAlertState(ctx, job, resultStatus)
		if err == nil {
			return decision, nil
		}
		if !isRetryableWorkerError(err) || attempt >= w.retryMax {
			return alertDecision{}, err
		}

		backoff := w.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker alert_state retry attempt=%d job_id=%d backoff=%s err=%v", attempt+1, job.ID, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return alertDecision{}, err
		}
	}
}

func (w *worker) applyAlertState(ctx context.Context, job claimedJob, resultStatus string) (alertDecision, error) {
	currentStatus, err := normalizeAlertStatus(resultStatus)
	if err != nil {
		return alertDecision{}, err
	}

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return alertDecision{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO alert_state (company_id, stream_id, fail_streak, cooldown_until, last_alert_at, last_status, created_at, updated_at)
         VALUES ($1, $2, 0, NULL, NULL, NULL, NOW(), NOW())
         ON CONFLICT (stream_id) DO NOTHING`,
		job.CompanyID,
		job.StreamID,
	)
	if err != nil {
		return alertDecision{}, err
	}

	var previousFailStreak int
	var previousCooldownUntil sql.NullTime
	var previousLastAlertAt sql.NullTime
	var previousStatusRaw sql.NullString
	scanErr := tx.QueryRowContext(
		ctx,
		`SELECT fail_streak, cooldown_until, last_alert_at, last_status
         FROM alert_state
         WHERE company_id = $1
           AND stream_id = $2
         FOR UPDATE`,
		job.CompanyID,
		job.StreamID,
	).Scan(&previousFailStreak, &previousCooldownUntil, &previousLastAlertAt, &previousStatusRaw)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return alertDecision{}, errors.New("alert_state row not found in tenant context")
		}
		return alertDecision{}, scanErr
	}

	previousStatus := ""
	if previousStatusRaw.Valid {
		normalizedPrevious, prevErr := normalizeAlertStatus(previousStatusRaw.String)
		if prevErr == nil {
			previousStatus = normalizedPrevious
		}
	}

	now := time.Now().UTC()
	cooldownActive := previousCooldownUntil.Valid && previousCooldownUntil.Time.After(now)

	newFailStreak := 0
	if currentStatus == "fail" {
		if previousStatus == "fail" {
			newFailStreak = previousFailStreak + 1
		} else {
			newFailStreak = 1
		}
	}

	decision := alertDecision{
		ShouldSend:     false,
		EventType:      "",
		Reason:         "no_alert_condition",
		CurrentStatus:  currentStatus,
		PreviousStatus: previousStatus,
		FailStreak:     newFailStreak,
		CooldownUntil:  nil,
	}

	nextCooldownUntil := previousCooldownUntil
	nextLastAlertAt := previousLastAlertAt

	if currentStatus == "fail" {
		if newFailStreak < w.alertFailStreak {
			decision.Reason = "fail_streak_below_threshold"
		} else if cooldownActive {
			decision.Reason = "cooldown_active"
		} else {
			decision.ShouldSend = true
			decision.EventType = "fail"
			decision.Reason = "fail_streak_threshold_met"
			nextLastAlertAt = sql.NullTime{Time: now, Valid: true}
			nextCooldownUntil = sql.NullTime{Time: now.Add(w.alertCooldown), Valid: true}
		}
	} else if currentStatus == "ok" && previousStatus == "fail" {
		if !w.alertSendRecovered {
			decision.Reason = "recovered_suppressed_by_config"
		} else if cooldownActive {
			decision.Reason = "cooldown_active"
		} else {
			decision.ShouldSend = true
			decision.EventType = "recovered"
			decision.Reason = "recovered_transition"
			nextLastAlertAt = sql.NullTime{Time: now, Valid: true}
			nextCooldownUntil = sql.NullTime{Time: now.Add(w.alertCooldown), Valid: true}
		}
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE alert_state
         SET fail_streak = $1,
             cooldown_until = $2,
             last_alert_at = $3,
             last_status = $4,
             updated_at = NOW()
         WHERE company_id = $5
           AND stream_id = $6`,
		newFailStreak,
		nullTimeToValue(nextCooldownUntil),
		nullTimeToValue(nextLastAlertAt),
		currentStatus,
		job.CompanyID,
		job.StreamID,
	)
	if err != nil {
		return alertDecision{}, err
	}

	if err := tx.Commit(); err != nil {
		return alertDecision{}, err
	}

	if nextCooldownUntil.Valid {
		cooldownCopy := nextCooldownUntil.Time.UTC()
		decision.CooldownUntil = &cooldownCopy
	}

	return decision, nil
}

func (w *worker) logAlertDecision(job claimedJob, decision alertDecision) {
	cooldownUntil := "null"
	if decision.CooldownUntil != nil {
		cooldownUntil = decision.CooldownUntil.Format(time.RFC3339)
	}
	log.Printf(
		"worker alert decision: company_id=%d stream_id=%d current_status=%s previous_status=%s fail_streak=%d fail_threshold=%d cooldown_until=%s should_send=%t event_type=%s reason=%s",
		job.CompanyID,
		job.StreamID,
		decision.CurrentStatus,
		decision.PreviousStatus,
		decision.FailStreak,
		w.alertFailStreak,
		cooldownUntil,
		decision.ShouldSend,
		decision.EventType,
		decision.Reason,
	)
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
	var nullableErrorMessage interface{}
	if errorMessage == "" {
		nullableErrorMessage = nil
	} else {
		nullableErrorMessage = errorMessage
	}

	result, err := w.db.ExecContext(
		ctx,
		`UPDATE check_jobs
         SET status = $1,
             finished_at = NOW(),
             error_message = $2
         WHERE id = $3
           AND company_id = $4
           AND status = 'running'`,
		status,
		nullableErrorMessage,
		job.ID,
		job.CompanyID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
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

func intInRange(value int, minimum int, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
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

func floatAtLeast(value float64, minimum float64) float64 {
	if value < minimum {
		return minimum
	}
	return value
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

func normalizeAlertStatus(statusRaw string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(statusRaw))
	switch normalized {
	case "ok", "warn", "fail":
		return normalized, nil
	default:
		return "", errors.New("unsupported alert status: " + statusRaw)
	}
}

func nullTimeToValue(value sql.NullTime) interface{} {
	if !value.Valid {
		return nil
	}
	return value.Time
}
