package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/terrynullson/mntrng/internal/domain"
)

type rateLimitStub struct {
	calls []string
	allow bool
}

func (s *rateLimitStub) Allow(_ context.Context, key string) (bool, error) {
	s.calls = append(s.calls, key)
	return s.allow, nil
}

func TestClientIPStripsPortFromRemoteAddr(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	request.RemoteAddr = "198.51.100.44:54321"

	ip := clientIP(request)
	if ip != "198.51.100.44" {
		t.Fatalf("expected stripped ip, got %q", ip)
	}
}

func TestClientIPUsesFirstForwardedValue(t *testing.T) {
	t.Setenv("TRUST_PROXY_HEADERS", "true")

	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	request.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.4")

	ip := clientIP(request)
	if ip != "203.0.113.7" {
		t.Fatalf("expected first forwarded ip, got %q", ip)
	}
}

func TestClientIPIgnoresForwardedHeadersByDefault(t *testing.T) {
	t.Setenv("TRUST_PROXY_HEADERS", "false")

	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	request.RemoteAddr = "198.51.100.44:54321"
	request.Header.Set("X-Forwarded-For", "203.0.113.7")
	request.Header.Set("X-Real-IP", "203.0.113.8")

	ip := clientIP(request)
	if ip != "198.51.100.44" {
		t.Fatalf("expected remote addr ip, got %q", ip)
	}
}

func TestRateLimitMiddlewareProtectsRefreshEndpoint(t *testing.T) {
	limiter := &rateLimitStub{allow: false}
	middleware := rateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	request.RemoteAddr = "203.0.113.7:10001"
	response := httptest.NewRecorder()

	middleware.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", response.Code)
	}
	if len(limiter.calls) != 1 || limiter.calls[0] != "203.0.113.7" {
		t.Fatalf("unexpected limiter calls: %+v", limiter.calls)
	}

	var envelope domain.ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "rate_limit_exceeded" {
		t.Fatalf("expected rate_limit_exceeded, got %q", envelope.Code)
	}
}
