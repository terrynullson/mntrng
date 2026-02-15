package telegram

import (
	"strings"
	"testing"
)

func TestBuildDevLogMessage(t *testing.T) {
	payload := DevLogPayload{
		Module: "worker",
		Agent:  "BackendAgent",
		Commit: "abc123",
		Status: "success",
		Summary: []string{
			"Implemented isolated notifier.",
			"Added tests.",
			"Updated README.",
		},
		Mood: "OK",
	}

	message := BuildDevLogMessage(payload)

	expected := []string{
		"[WORKER COMPLETED]",
		"Agent: BackendAgent",
		"Commit: abc123",
		"Status: SUCCESS",
		"Summary:",
		"- Implemented isolated notifier.",
		"- Added tests.",
		"- Updated README.",
		"Mood: OK",
	}
	for _, line := range expected {
		if !strings.Contains(message, line) {
			t.Fatalf("expected message to contain %q, got %q", line, message)
		}
	}
}

func TestBuildDevLogMessageDefaults(t *testing.T) {
	message := BuildDevLogMessage(DevLogPayload{})
	if !strings.Contains(message, "[MODULE COMPLETED]") {
		t.Fatalf("expected default module marker, got %q", message)
	}
	if !strings.Contains(message, "Mood: OK") {
		t.Fatalf("expected default mood, got %q", message)
	}
	if !strings.Contains(message, "- Completed without additional summary.") {
		t.Fatalf("expected default summary line, got %q", message)
	}
}
