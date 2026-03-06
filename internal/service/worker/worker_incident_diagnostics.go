package worker

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	diagCodeBlackframe  = "BLACKFRAME"
	diagCodeFreeze      = "FREEZE"
	diagCodeCaptureFail = "CAPTURE_FAIL"
	diagCodeUnknown     = "UNKNOWN"
)

var (
	diagMetricsOnce   sync.Once
	incidentDiagTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "incident_diag_total",
			Help: "Total incident diagnostics by code.",
		},
		[]string{"code"},
	)
	screenshotCaptureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "screenshot_capture_total",
			Help: "Total screenshot capture attempts by result.",
		},
		[]string{"result"},
	)
)

func ensureDiagnosticMetricsRegistered() {
	diagMetricsOnce.Do(func() {
		prometheus.MustRegister(incidentDiagTotal, screenshotCaptureTotal)
	})
}

func (w *worker) maybeCaptureIncidentDiagnostic(ctx context.Context, job claimedJob, incident domain.Incident, isNew bool, previousSeverity string, currentSeverity string) {
	ensureDiagnosticMetricsRegistered()
	shouldCapture := isNew
	if !shouldCapture && previousSeverity == domain.IncidentSeverityWarn && currentSeverity == domain.IncidentSeverityFail {
		shouldCapture = true
	}
	if !shouldCapture {
		if incident.ScreenshotTakenAt == nil {
			shouldCapture = true
		} else if time.Since(incident.ScreenshotTakenAt.UTC()) >= w.incidentScreenshotInterval {
			shouldCapture = true
		}
	}
	if !shouldCapture {
		return
	}

	screenshotPath, diagCode, details := w.captureAndDiagnose(ctx, job.CompanyID, job.StreamID, incident.ID)
	if err := w.incidentRepo.UpdateDiagnostic(ctx, incident.ID, job.CompanyID, job.StreamID, screenshotPath, time.Now().UTC(), diagCode, details); err != nil {
		return
	}
	incidentDiagTotal.WithLabelValues(diagCode).Inc()
}

func (w *worker) captureAndDiagnose(ctx context.Context, companyID int64, streamID int64, incidentID int64) (*string, string, map[string]interface{}) {
	streamURL, err := w.loadStreamURL(ctx, companyID, streamID)
	if err != nil {
		screenshotCaptureTotal.WithLabelValues("fail").Inc()
		return nil, diagCodeCaptureFail, map[string]interface{}{"error": err.Error()}
	}
	baseDir := filepath.Join(w.dataDir, "screenshots", "incidents", strconv.FormatInt(companyID, 10), strconv.FormatInt(incidentID, 10))
	if mkErr := os.MkdirAll(baseDir, 0o755); mkErr != nil {
		screenshotCaptureTotal.WithLabelValues("fail").Inc()
		return nil, diagCodeCaptureFail, map[string]interface{}{"error": mkErr.Error()}
	}
	timestamp := time.Now().UTC().Format("20060102T150405")
	frameAPath := filepath.Join(baseDir, fmt.Sprintf("%s-A.jpg", timestamp))
	frameBPath := filepath.Join(baseDir, fmt.Sprintf("%s-B.jpg", timestamp))

	if err := w.captureFrame(ctx, streamURL, frameAPath); err != nil {
		screenshotCaptureTotal.WithLabelValues("fail").Inc()
		return nil, diagCodeCaptureFail, map[string]interface{}{"error": err.Error()}
	}
	screenshotCaptureTotal.WithLabelValues("ok").Inc()

	sleepTimer := time.NewTimer(w.diagnosticFreezeInterval)
	select {
	case <-ctx.Done():
		sleepTimer.Stop()
	case <-sleepTimer.C:
	}

	if err := w.captureFrame(ctx, streamURL, frameBPath); err != nil {
		screenshotCaptureTotal.WithLabelValues("fail").Inc()
		path := frameAPath
		return &path, diagCodeCaptureFail, map[string]interface{}{"error": err.Error()}
	}
	screenshotCaptureTotal.WithLabelValues("ok").Inc()

	blackRatio, err := estimateBlackRatio(frameAPath)
	if err != nil {
		path := frameAPath
		return &path, diagCodeCaptureFail, map[string]interface{}{"error": err.Error()}
	}
	if blackRatio >= w.blackframeFailRatio {
		path := frameAPath
		return &path, diagCodeBlackframe, map[string]interface{}{
			"black_ratio": blackRatio,
			"threshold":   w.blackframeFailRatio,
		}
	}

	diffValue, err := compareFramesDiff(frameAPath, frameBPath)
	if err != nil {
		path := frameAPath
		return &path, diagCodeCaptureFail, map[string]interface{}{"error": err.Error()}
	}
	if diffValue < w.diagnosticFreezeDiffThreshold {
		path := frameAPath
		return &path, diagCodeFreeze, map[string]interface{}{
			"diff":         diffValue,
			"interval_sec": w.diagnosticFreezeInterval.Seconds(),
		}
	}

	path := frameAPath
	return &path, diagCodeUnknown, map[string]interface{}{
		"black_ratio": blackRatio,
		"diff":        diffValue,
	}
}

func (w *worker) captureFrame(ctx context.Context, streamURL string, outPath string) error {
	if err := w.validateExternalURL(streamURL); err != nil {
		return err
	}
	captureCtx, cancel := context.WithTimeout(ctx, w.diagnosticCaptureTimeout)
	defer cancel()
	cmd := exec.CommandContext(
		captureCtx,
		"ffmpeg",
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-i", strings.TrimSpace(streamURL),
		"-frames:v", "1",
		"-vf", "scale=640:-1",
		outPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg capture failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	info, err := os.Stat(outPath)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return errors.New("captured frame is empty")
	}
	return nil
}

func estimateBlackRatio(path string) (float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return 0, err
	}
	bounds := img.Bounds()
	if bounds.Empty() {
		return 0, errors.New("image has empty bounds")
	}
	total := 0
	black := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 4 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 4 {
			r, g, b, _ := img.At(x, y).RGBA()
			l := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 65535.0
			if l < 0.08 {
				black++
			}
			total++
		}
	}
	if total == 0 {
		return 0, errors.New("image has no sampled pixels")
	}
	return float64(black) / float64(total), nil
}

func compareFramesDiff(pathA string, pathB string) (float64, error) {
	imgA, err := decodeImage(pathA)
	if err != nil {
		return 0, err
	}
	imgB, err := decodeImage(pathB)
	if err != nil {
		return 0, err
	}
	bounds := imgA.Bounds()
	targetW := 160
	if bounds.Dx() < targetW {
		targetW = bounds.Dx()
	}
	if targetW <= 0 {
		return 0, errors.New("invalid frame width")
	}
	stepX := float64(bounds.Dx()) / float64(targetW)
	targetH := int(math.Max(1, float64(bounds.Dy())/stepX))
	sumDiff := 0.0
	count := 0
	for y := 0; y < targetH; y++ {
		sy := bounds.Min.Y + int(float64(y)*stepX)
		for x := 0; x < targetW; x++ {
			sx := bounds.Min.X + int(float64(x)*stepX)
			r1, g1, b1, _ := imgA.At(sx, sy).RGBA()
			r2, g2, b2, _ := imgB.At(sx, sy).RGBA()
			dr := math.Abs(float64(r1)-float64(r2)) / 65535.0
			dg := math.Abs(float64(g1)-float64(g2)) / 65535.0
			db := math.Abs(float64(b1)-float64(b2)) / 65535.0
			sumDiff += (dr + dg + db) / 3.0
			count++
		}
	}
	if count == 0 {
		return 0, errors.New("no pixels compared")
	}
	return sumDiff / float64(count), nil
}

func decodeImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}
