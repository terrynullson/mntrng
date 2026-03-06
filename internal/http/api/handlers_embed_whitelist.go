package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
)

type embedWhitelistItem = domain.EmbedWhitelistItem
type embedWhitelistListResponse = domain.EmbedWhitelistListResponse
type createEmbedWhitelistRequest = domain.CreateEmbedWhitelistRequest
type patchEmbedWhitelistRequest = domain.PatchEmbedWhitelistRequest

func (s *Server) handleListEmbedWhitelist(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	items, err := s.embedWhitelistService.List(ctx, companyID)
	if err != nil {
		writeServiceError(w, r, "list embed whitelist", err)
		return
	}
	if err := WriteJSON(w, http.StatusOK, embedWhitelistListResponse{Items: items}); err != nil {
		log.Printf("list embed whitelist response encode error: %v", err)
	}
}

func (s *Server) handleCreateEmbedWhitelist(w http.ResponseWriter, r *http.Request, companyID int64) {
	auth, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "auth context missing", nil)
		return
	}
	var request createEmbedWhitelistRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := s.embedWhitelistService.Create(ctx, companyID, request.Domain, auth.UserID)
	if err != nil {
		writeServiceError(w, r, "create embed whitelist", err)
		return
	}
	if err := WriteJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create embed whitelist response encode error: %v", err)
	}
}

func (s *Server) handlePatchEmbedWhitelist(w http.ResponseWriter, r *http.Request, companyID int64, id int64) {
	auth, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "auth context missing", nil)
		return
	}
	var request patchEmbedWhitelistRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := s.embedWhitelistService.Patch(ctx, companyID, id, request.Enabled, auth.UserID)
	if err != nil {
		writeServiceError(w, r, "patch embed whitelist", err)
		return
	}
	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch embed whitelist response encode error: %v", err)
	}
}

func (s *Server) handleDeleteEmbedWhitelist(w http.ResponseWriter, r *http.Request, companyID int64, id int64) {
	auth, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "auth context missing", nil)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.embedWhitelistService.Delete(ctx, companyID, id, auth.UserID); err != nil {
		writeServiceError(w, r, "delete embed whitelist", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
