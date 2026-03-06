package api

import (
	"errors"
	"log"
	"net/http"

	serviceapi "github.com/terrynullson/hls_mntrng/internal/service/api"
)

func writeServiceError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	var serviceErr *serviceapi.ServiceError
	if errors.As(err, &serviceErr) {
		WriteJSONError(w, r, serviceErr.StatusCode, serviceErr.Code, serviceErr.Message, serviceErr.Details)
		return
	}

	log.Printf("%s failed: %v", operation, err)
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
}
