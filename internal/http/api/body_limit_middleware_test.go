package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func TestRequestBodyLimitMiddlewareRejectsLargePayload(t *testing.T) {
	t.Setenv("API_MAX_BODY_BYTES", "1024")
	handlerCalled := false
	middleware := requestBodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	body := `{"payload":"` + strings.Repeat("a", 2000) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	request.ContentLength = int64(len(body))
	response := httptest.NewRecorder()

	middleware.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", response.Code)
	}
	if handlerCalled {
		t.Fatal("handler must not be called for oversized payload")
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "payload_too_large" {
		t.Fatalf("expected payload_too_large code, got %q", envelope.Code)
	}
}

func TestRequestBodyLimitMiddlewareAllowsSmallPayload(t *testing.T) {
	t.Setenv("API_MAX_BODY_BYTES", "64")
	handlerCalled := false
	middleware := requestBodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"ok":true}`))
	response := httptest.NewRecorder()

	middleware.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
	if !handlerCalled {
		t.Fatal("handler must be called for small payload")
	}
}

func TestRequestBodyLimitMiddlewareBlocksOversizeOnRead(t *testing.T) {
	t.Setenv("API_MAX_BODY_BYTES", "1024")
	middleware := requestBodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		WriteJSONError(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "request body is too large", map[string]interface{}{})
	}))

	body := `{"payload":"` + strings.Repeat("b", 2000) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	request.ContentLength = -1
	response := httptest.NewRecorder()

	middleware.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", response.Code)
	}
}
