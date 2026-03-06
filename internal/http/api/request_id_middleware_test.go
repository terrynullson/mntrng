package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/terrynullson/mntrng/internal/domain"
)

func TestRequestIDConsistentBetweenHeaderAndErrorEnvelope(t *testing.T) {
	handler := withHTTPObservability(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "bad request", map[string]interface{}{})
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/streams", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}

	headerID := response.Header().Get("X-Request-ID")
	if headerID == "" {
		t.Fatal("expected X-Request-ID header")
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.RequestID != headerID {
		t.Fatalf("expected envelope request_id=%q, got %q", headerID, envelope.RequestID)
	}
}

func TestRequestIDUsesInboundHeaderWhenProvided(t *testing.T) {
	handler := withHTTPObservability(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "bad request", map[string]interface{}{})
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/companies/1/streams", nil)
	request.Header.Set("X-Request-ID", "req_from_client")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}

	if got := response.Header().Get("X-Request-ID"); got != "req_from_client" {
		t.Fatalf("expected response header request id req_from_client, got %q", got)
	}
}
