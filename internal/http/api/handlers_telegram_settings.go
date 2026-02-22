package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (s *Server) handleGetTelegramDeliverySettings(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	out, err := s.telegramSettingsService.GetTelegramDeliverySettings(ctx, companyID)
	if err != nil {
		writeServiceError(w, r, "get telegram delivery settings", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, out); err != nil {
		log.Printf("get telegram delivery settings response encode error: %v", err)
	}
}

func (s *Server) handlePatchTelegramDeliverySettings(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request domain.PatchTelegramDeliverySettingsRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	out, err := s.telegramSettingsService.PatchTelegramDeliverySettings(ctx, companyID, request)
	if err != nil {
		writeServiceError(w, r, "patch telegram delivery settings", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, out); err != nil {
		log.Printf("patch telegram delivery settings response encode error: %v", err)
	}
}
