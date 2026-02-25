package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func TestRecoveryMiddlewareHandlesPanic(t *testing.T) {
	handler := recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/test-panic", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", response.Code)
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "internal_error" {
		t.Fatalf("expected internal_error, got %s", envelope.Code)
	}
}
