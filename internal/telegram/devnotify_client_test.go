package telegram

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestDevLogNotifierDoesNotLeakTokenInError(t *testing.T) {
	secretToken := "dev-log-secret-token"
	client := &http.Client{
		Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			return nil, errors.New("request failed for bot token " + secretToken)
		}),
	}

	notifier := NewDevLogNotifier(true, secretToken, "12345", client)
	err := notifier.NotifyCompletion(context.Background(), DevLogPayload{Module: "api", Commit: "abc123"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), secretToken) {
		t.Fatalf("error leaked token: %q", err.Error())
	}
}

func TestDevLogNotifierDisabledIsNoop(t *testing.T) {
	notifier := NewDevLogNotifier(false, "", "", nil)
	err := notifier.NotifyCompletion(context.Background(), DevLogPayload{})
	if err != nil {
		t.Fatalf("expected nil error for disabled notifier, got %v", err)
	}
}
