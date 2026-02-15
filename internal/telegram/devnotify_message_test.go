package telegram

import (
	"strings"
	"testing"
)

func TestBuildDevLogMessageWithRussianContentAndThoughts(t *testing.T) {
	payload := DevLogPayload{
		Module: "worker",
		Agent:  "BackendAgent",
		Commit: "abc123",
		Status: "успех",
		Summary: []string{
			"Закрыт протокольный drift.",
			"Обновлены тесты.",
		},
		Mood: "Огонь",
		Thoughts: []string{
			"Наконец-то все синхронизировано.",
			"Можно спокойно двигаться дальше.",
		},
	}

	message := BuildDevLogMessage(payload)

	expected := []string{
		"[WORKER ЗАВЕРШЕНО]",
		"Агент: BackendAgent",
		"Коммит: abc123",
		"Статус: УСПЕХ",
		"Сводка:",
		"- Закрыт протокольный drift.",
		"- Обновлены тесты.",
		"Настроение: Огонь",
		"Мысли:",
		"- Наконец-то все синхронизировано.",
		"- Можно спокойно двигаться дальше.",
	}
	for _, line := range expected {
		if !strings.Contains(message, line) {
			t.Fatalf("expected message to contain %q, got %q", line, message)
		}
	}
}

func TestBuildDevLogMessageDefaults(t *testing.T) {
	message := BuildDevLogMessage(DevLogPayload{})
	if !strings.Contains(message, "[МОДУЛЬ ЗАВЕРШЕНО]") {
		t.Fatalf("expected default module marker, got %q", message)
	}
	if !strings.Contains(message, "Настроение: Спокойно") {
		t.Fatalf("expected default mood, got %q", message)
	}
	if !strings.Contains(message, "- Этап завершен без дополнительных комментариев.") {
		t.Fatalf("expected default summary line, got %q", message)
	}
}

func TestValidateDevLogPayloadGuardrails(t *testing.T) {
	testCases := []struct {
		name      string
		payload   DevLogPayload
		expectErr bool
	}{
		{
			name: "allows expressive non-addressed text",
			payload: DevLogPayload{
				Summary: []string{"Черт, это был сложный этап, но справились."},
				Mood:    "Пушка",
			},
			expectErr: false,
		},
		{
			name: "blocks personal insult",
			payload: DevLogPayload{
				Summary: []string{"Ты идиот и ничего не понимаешь"},
			},
			expectErr: true,
		},
		{
			name: "blocks hate or discrimination",
			payload: DevLogPayload{
				Mood: "ненавижу эту расу",
			},
			expectErr: true,
		},
		{
			name: "blocks secret token",
			payload: DevLogPayload{
				Summary: []string{"token=123456789:ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"},
			},
			expectErr: true,
		},
		{
			name: "blocks pii email",
			payload: DevLogPayload{
				Summary: []string{"Пишите мне на user@example.com"},
			},
			expectErr: true,
		},
		{
			name: "blocks architecture decisions",
			payload: DevLogPayload{
				Summary: []string{"Предлагаю ADR-0099 как новое архитектурное решение"},
			},
			expectErr: true,
		},
		{
			name: "blocks too many thoughts",
			payload: DevLogPayload{
				Summary:  []string{"Готово"},
				Thoughts: []string{"раз", "два", "три"},
			},
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateDevLogPayload(testCase.payload)
			if testCase.expectErr && err == nil {
				t.Fatalf("expected error")
			}
			if !testCase.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
