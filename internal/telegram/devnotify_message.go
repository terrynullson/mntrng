package telegram

import (
	"errors"
	"regexp"
	"strings"
)

const (
	defaultDevLogAgent  = "BackendAgent"
	defaultDevLogStatus = "УСПЕХ"
	defaultDevLogMood   = "Спокойно"
)

var (
	reEmailLike        = regexp.MustCompile(`(?i)\b[\w._%+-]+@[\w.-]+\.[a-z]{2,}\b`)
	rePhoneLike        = regexp.MustCompile(`\+?\d[\d\s\-()]{8,}\d`)
	reTelegramToken    = regexp.MustCompile(`\b\d{7,12}:[A-Za-z0-9_-]{20,}\b`)
	reSecretAssignment = regexp.MustCompile(`(?i)\b(token|secret|password|passwd|api[_-]?key|bearer)\b\s*[:=]\s*\S+`)
)

type DevLogPayload struct {
	Module   string
	Agent    string
	Commit   string
	Status   string
	Summary  []string
	Mood     string
	Thoughts []string
}

func BuildDevLogMessage(payload DevLogPayload) string {
	module := strings.ToUpper(strings.TrimSpace(payload.Module))
	if module == "" {
		module = "МОДУЛЬ"
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
	thoughtLines := normalizeThoughtLines(payload.Thoughts)
	if len(thoughtLines) > 2 {
		thoughtLines = thoughtLines[:2]
	}

	lines := []string{
		"[" + module + " ЗАВЕРШЕНО]",
		"Агент: " + agent,
		"Коммит: " + commit,
		"Статус: " + status,
		"Сводка:",
	}
	for _, line := range summaryLines {
		lines = append(lines, "- "+line)
	}
	lines = append(lines, "Настроение: "+mood)

	if len(thoughtLines) > 0 {
		lines = append(lines, "Мысли:")
		for _, line := range thoughtLines {
			lines = append(lines, "- "+line)
		}
	}

	return strings.Join(lines, "\n")
}

func ValidateDevLogPayload(payload DevLogPayload) error {
	thoughtLines := normalizeThoughtLines(payload.Thoughts)
	if len(thoughtLines) > 2 {
		return errors.New("thoughts must contain at most 2 lines")
	}

	lines := make([]string, 0, len(payload.Summary)+len(thoughtLines)+1)
	lines = append(lines, normalizeSummaryLines(payload.Summary)...)
	lines = append(lines, thoughtLines...)

	trimmedMood := strings.TrimSpace(payload.Mood)
	if trimmedMood != "" {
		lines = append(lines, trimmedMood)
	}

	for _, line := range lines {
		if err := validateSafetyLine(line); err != nil {
			return err
		}
	}

	return nil
}

func validateSafetyLine(line string) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	if reTelegramToken.MatchString(trimmed) || reSecretAssignment.MatchString(trimmed) {
		return errors.New("content contains secret or token value")
	}
	if reEmailLike.MatchString(trimmed) || rePhoneLike.MatchString(trimmed) {
		return errors.New("content contains pii")
	}

	lower := strings.ToLower(trimmed)
	if containsArchitectureDecisionText(lower) {
		return errors.New("content contains architecture decision")
	}
	if containsHateOrDiscrimination(lower) {
		return errors.New("content contains hate or discrimination")
	}
	if containsPersonalInsult(lower) {
		return errors.New("content contains personal insult")
	}

	return nil
}

func containsArchitectureDecisionText(lower string) bool {
	for _, marker := range []string{"adr-", "архитектурн", "architecture decision", "архрешение"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func containsHateOrDiscrimination(lower string) bool {
	for _, marker := range []string{"hate", "ненавиж", "расист", "дискримина", "нацист", "supremac"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func containsPersonalInsult(lower string) bool {
	tokens := tokenize(lower)
	if len(tokens) == 0 {
		return false
	}

	insultWords := map[string]struct{}{
		"идиот":   {},
		"дебил":   {},
		"кретин":  {},
		"дурак":   {},
		"мразь":   {},
		"ублюдок": {},
		"moron":   {},
		"idiot":   {},
		"stupid":  {},
	}
	targetWords := map[string]struct{}{
		"ты":      {},
		"вы":      {},
		"тебя":    {},
		"тебе":    {},
		"твой":    {},
		"вас":     {},
		"вам":     {},
		"you":     {},
		"your":    {},
		"агент":   {},
		"коллега": {},
	}

	hasInsult := false
	hasTarget := strings.Contains(lower, "@")

	for _, token := range tokens {
		if _, ok := insultWords[token]; ok {
			hasInsult = true
		}
		if _, ok := targetWords[token]; ok {
			hasTarget = true
		}
	}

	return hasInsult && hasTarget
}

func tokenize(line string) []string {
	replacer := strings.NewReplacer(
		",", " ",
		".", " ",
		"!", " ",
		"?", " ",
		":", " ",
		";", " ",
		"\"", " ",
		"'", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
	)
	prepared := replacer.Replace(line)
	return strings.Fields(prepared)
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
		return []string{"Этап завершен без дополнительных комментариев."}
	}
	return normalized
}

func normalizeThoughtLines(lines []string) []string {
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
