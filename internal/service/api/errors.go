package api

import (
	"errors"
	"net/http"
)

type ServiceError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]interface{}
}

func (e *ServiceError) Error() string {
	return e.Message
}

func NewValidationError(message string, details map[string]interface{}) *ServiceError {
	return &ServiceError{
		StatusCode: http.StatusBadRequest,
		Code:       "validation_error",
		Message:    message,
		Details:    details,
	}
}

func NewNotFoundError(message string, details map[string]interface{}) *ServiceError {
	return &ServiceError{
		StatusCode: http.StatusNotFound,
		Code:       "not_found",
		Message:    message,
		Details:    details,
	}
}

func NewConflictError(message string, details map[string]interface{}) *ServiceError {
	return &ServiceError{
		StatusCode: http.StatusConflict,
		Code:       "conflict",
		Message:    message,
		Details:    details,
	}
}

func NewInternalError() *ServiceError {
	return &ServiceError{
		StatusCode: http.StatusInternalServerError,
		Code:       "internal_error",
		Message:    "internal server error",
		Details:    map[string]interface{}{},
	}
}

func AsServiceError(err error) (*ServiceError, bool) {
	if err == nil {
		return nil, false
	}
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr, true
	}
	return nil, false
}
