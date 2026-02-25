package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func WriteJSON(w http.ResponseWriter, statusCode int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(payload)
}

func WriteJSONError(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	code string,
	message string,
	details interface{},
) {
	err := WriteJSON(w, statusCode, domain.ErrorEnvelope{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: RequestIDFromRequest(r),
	})
	if err != nil {
		log.Printf("error response encode failed: %v", err)
	}
}

func WriteMethodNotAllowed(w http.ResponseWriter, r *http.Request, allowedMethods ...string) {
	w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	WriteJSONError(
		w,
		r,
		http.StatusMethodNotAllowed,
		"method_not_allowed",
		"method is not allowed for this endpoint",
		map[string]interface{}{
			"method":          r.Method,
			"allowed_methods": allowedMethods,
		},
	)
}

func DecodeJSONBody(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func RequestIDFromRequest(r *http.Request) string {
	if r != nil {
		if requestID, ok := requestIDFromContext(r.Context()); ok {
			return requestID
		}
		if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
			return requestID
		}
	}
	return "req_" + time.Now().UTC().Format("20060102150405.000000000")
}
