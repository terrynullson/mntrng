package ai

import (
	"context"
	"log"

	"github.com/terrynullson/mntrng/internal/config"
)

// IncidentInput is the input for AI incident analysis (B6 contract).
type IncidentInput struct {
	Checks         map[string]interface{}
	ScreenshotPath string
	CompanyID      int64
	StreamID       int64
	JobID          int64
}

// IncidentResult is the output: cause and summary.
type IncidentResult struct {
	Cause   string
	Summary string
}

// Analyzer performs on-demand incident analysis. Worker is the only caller.
type Analyzer interface {
	Analyze(ctx context.Context, input IncidentInput) (IncidentResult, error)
}

// StubAnalyzer is a no-op implementation when no API key is configured.
// When AI_INCIDENT_API_KEY is set, it still returns empty result (real provider can be wired later).
// Never logs secrets or API keys.
type StubAnalyzer struct{}

func NewStubAnalyzer() *StubAnalyzer {
	return &StubAnalyzer{}
}

func (s *StubAnalyzer) Analyze(ctx context.Context, input IncidentInput) (IncidentResult, error) {
	_ = input
	key := config.GetString("AI_INCIDENT_API_KEY", "")
	if key == "" {
		return IncidentResult{}, nil
	}
	// Key is set but no real implementation yet; return empty so worker does not fail.
	return IncidentResult{}, nil
}

// LogAnalyzer wraps an Analyzer and logs non-nil errors without secrets.
type LogAnalyzer struct {
	Inner Analyzer
}

func (l *LogAnalyzer) Analyze(ctx context.Context, input IncidentInput) (IncidentResult, error) {
	res, err := l.Inner.Analyze(ctx, input)
	if err != nil {
		log.Printf("worker ai incident: job_id=%d company_id=%d stream_id=%d analysis_error=1", input.JobID, input.CompanyID, input.StreamID)
		return IncidentResult{}, err
	}
	return res, nil
}
