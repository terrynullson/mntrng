package api

import (
	"context"
	"log"
	"net/http"
	"time"
)

type streamWithFavoriteResponse = struct {
	Stream    stream `json:"stream"`
	IsPinned  bool   `json:"is_pinned"`
	SortOrder int    `json:"sort_order"`
}

type streamFavoritesListResponse struct {
	Items []streamWithFavoriteResponse `json:"items"`
}

func (s *Server) handleListStreamFavorites(w http.ResponseWriter, r *http.Request, companyID int64) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowed(w, r, http.MethodGet)
		return
	}
	auth, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "auth context missing", nil)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	items, err := s.streamFavoriteService.ListFavorites(ctx, auth.UserID, companyID)
	if err != nil {
		writeServiceError(w, r, "list favorites", err)
		return
	}
	out := make([]streamWithFavoriteResponse, len(items))
	for i := range items {
		out[i] = streamWithFavoriteResponse{
			Stream:    items[i].Stream,
			IsPinned:  items[i].IsPinned,
			SortOrder: items[i].SortOrder,
		}
	}
	if err := WriteJSON(w, http.StatusOK, streamFavoritesListResponse{Items: out}); err != nil {
		log.Printf("list favorites response encode error: %v", err)
	}
}

func (s *Server) handleAddStreamFavorite(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	if r.Method != http.MethodPost {
		WriteMethodNotAllowed(w, r, http.MethodPost)
		return
	}
	auth, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "auth context missing", nil)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.streamFavoriteService.AddFavorite(ctx, auth.UserID, companyID, streamID); err != nil {
		writeServiceError(w, r, "add favorite", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoveStreamFavorite(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	if r.Method != http.MethodDelete {
		WriteMethodNotAllowed(w, r, http.MethodDelete)
		return
	}
	auth, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "auth context missing", nil)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_ = s.streamFavoriteService.RemoveFavorite(ctx, auth.UserID, companyID, streamID)
	w.WriteHeader(http.StatusNoContent)
}

type pinRequest struct {
	SortOrder *int `json:"sort_order"`
}

func (s *Server) handleAddStreamPin(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	if r.Method != http.MethodPost {
		WriteMethodNotAllowed(w, r, http.MethodPost)
		return
	}
	auth, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "auth context missing", nil)
		return
	}
	var req pinRequest
	_ = DecodeJSONBody(r, &req)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.streamFavoriteService.AddPin(ctx, auth.UserID, companyID, streamID, req.SortOrder); err != nil {
		writeServiceError(w, r, "add pin", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoveStreamPin(w http.ResponseWriter, r *http.Request, companyID int64, streamID int64) {
	if r.Method != http.MethodDelete {
		WriteMethodNotAllowed(w, r, http.MethodDelete)
		return
	}
	auth, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "auth context missing", nil)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_ = s.streamFavoriteService.RemovePin(ctx, auth.UserID, companyID, streamID)
	w.WriteHeader(http.StatusNoContent)
}
