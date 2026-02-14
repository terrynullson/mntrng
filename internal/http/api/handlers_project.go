package api

import (
	"context"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request, companyID int64) {
	var request createProjectRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.projectService.CreateProject(ctx, companyID, request.Name)
	if err != nil {
		writeServiceError(w, r, "create project", err)
		return
	}

	if err := WriteJSON(w, http.StatusCreated, item); err != nil {
		log.Printf("create project response encode error: %v", err)
	}
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request, companyID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := s.projectService.ListProjects(ctx, companyID)
	if err != nil {
		writeServiceError(w, r, "list projects", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, projectListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list projects response encode error: %v", err)
	}
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.projectService.GetProject(ctx, companyID, projectID)
	if err != nil {
		writeServiceError(w, r, "get project", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("get project response encode error: %v", err)
	}
}

func (s *Server) handlePatchProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	var request patchProjectRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.projectService.PatchProject(ctx, companyID, projectID, request.Name)
	if err != nil {
		writeServiceError(w, r, "patch project", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("patch project response encode error: %v", err)
	}
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request, companyID int64, projectID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.projectService.DeleteProject(ctx, companyID, projectID); err != nil {
		writeServiceError(w, r, "delete project", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
