package telegram

import (
	"strings"
)

const (
	defaultDevLogAgent  = "BackendAgent"
	defaultDevLogStatus = "SUCCESS"
	defaultDevLogMood   = "OK"
)

type DevLogPayload struct {
	Module  string
	Agent   string
	Commit  string
	Status  string
	Summary []string
	Mood    string
}

func BuildDevLogMessage(payload DevLogPayload) string {
	module := strings.ToUpper(strings.TrimSpace(payload.Module))
	if module == "" {
		module = "MODULE"
	}

	agent := strings.TrimSpace(payload.Agent)
	if agent == "" {
		agent = defaultDevLogAgent
	}

	commit := strings.TrimSpace(payload.Commit)
	if commit == "" {
		commit = "unknown"
	}

	status := strings.ToUpper(strings.TrimSpace(payload.Status))
	if status == "" {
		status = defaultDevLogStatus
	}

	mood := strings.TrimSpace(payload.Mood)
	if mood == "" {
		mood = defaultDevLogMood
	}

	summaryLines := normalizeSummaryLines(payload.Summary)
	lines := []string{
		"[" + module + " COMPLETED]",
		"Agent: " + agent,
		"Commit: " + commit,
		"Status: " + status,
		"Summary:",
	}
	for _, line := range summaryLines {
		lines = append(lines, "- "+line)
	}
	lines = append(lines, "Mood: "+mood)

	return strings.Join(lines, "\n")
}

func normalizeSummaryLines(lines []string) []string {
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return []string{"Completed without additional summary."}
	}
	return normalized
}
