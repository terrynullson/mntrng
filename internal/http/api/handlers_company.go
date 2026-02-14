package api

import (
	"context"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	var request createCompanyRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.companyService.CreateCompany(ctx, request.Name)
	if err != nil {
		writeServiceError(w, r, "create company", err)
		return
	}

	if err := WriteJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create company response encode error: %v", err)
	}
}

func (s *Server) handleListCompanies(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := s.companyService.ListCompanies(ctx)
	if err != nil {
		writeServiceError(w, r, "list companies", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, companyListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list companies response encode error: %v", err)
	}
}

func (s *Server) handleGetCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.companyService.GetCompany(ctx, companyID)
	if err != nil {
		writeServiceError(w, r, "get company", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get company response encode error: %v", err)
	}
}

func (s *Server) handlePatchCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request patchCompanyRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.companyService.PatchCompany(ctx, companyID, request.Name)
	if err != nil {
		writeServiceError(w, r, "patch company", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch company response encode error: %v", err)
	}
}

func (s *Server) handleDeleteCompany(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.companyService.DeleteCompany(ctx, companyID); err != nil {
		writeServiceError(w, r, "delete company", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
