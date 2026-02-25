package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONBodyRejectsMultipleJSONObjects(t *testing.T) {
	request := httptest.NewRequest("POST", "/api/v1/test", strings.NewReader(`{"a":1}{"b":2}`))

	var payload map[string]interface{}
	err := DecodeJSONBody(request, &payload)
	if err == nil {
		t.Fatal("expected decode error for multiple JSON objects")
	}
}
