package api

import (
	"context"
	"log"
	"net/http"
	"time"

	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

func (s *Server) handleListPendingRegistrationRequests(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := s.registrationService.ListPendingRegistrationRequests(ctx)
	if err != nil {
		writeServiceError(w, r, "list pending registration requests", err)
		return
	}

	response := map[string]interface{}{
		"items":       items,
		"next_cursor": nil,
	}
	if err := WriteJSON(w, http.StatusOK, response); err != nil {
		log.Printf("list pending registration requests response encode error: %v", err)
	}
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := s.registrationService.ListUsers(ctx, serviceapi.ListAdminUsersInput{
		CompanyIDRaw: r.URL.Query().Get("company_id"),
		RoleRaw:      r.URL.Query().Get("role"),
		StatusRaw:    r.URL.Query().Get("status"),
		LimitRaw:     r.URL.Query().Get("limit"),
	})
	if err != nil {
		writeServiceError(w, r, "list admin users", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, adminUserListResponse{Items: items, NextCursor: nil}); err != nil {
		log.Printf("list admin users response encode error: %v", err)
	}
}

func (s *Server) handleApproveRegistrationRequest(w http.ResponseWriter, r *http.Request, requestID int64) {
	authContext, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "authentication context is missing", map[string]interface{}{})
		return
	}

	var request approveRegistrationRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.registrationService.ApproveRegistrationRequest(ctx, requestID, request, authContext.UserID)
	if err != nil {
		writeServiceError(w, r, "approve registration request", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("approve registration request response encode error: %v", err)
	}
}

func (s *Server) handleRejectRegistrationRequest(w http.ResponseWriter, r *http.Request, requestID int64) {
	authContext, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "authentication context is missing", map[string]interface{}{})
		return
	}

	var request rejectRegistrationRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.registrationService.RejectRegistrationRequest(ctx, requestID, request, authContext.UserID); err != nil {
		writeServiceError(w, r, "reject registration request", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleChangeUserRole(w http.ResponseWriter, r *http.Request, userID int64) {
	authContext, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "authentication context is missing", map[string]interface{}{})
		return
	}

	var request changeUserRoleRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.registrationService.ChangeUserRole(ctx, userID, request, authContext.UserID)
	if err != nil {
		writeServiceError(w, r, "change user role", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("change user role response encode error: %v", err)
	}
}

func (s *Server) handleChangeUserStatus(w http.ResponseWriter, r *http.Request, userID int64) {
	authContext, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "authentication context is missing", map[string]interface{}{})
		return
	}

	var request changeUserStatusRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.registrationService.ChangeUserStatus(ctx, userID, request, authContext.UserID)
	if err != nil {
		writeServiceError(w, r, "change user status", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("change user status response encode error: %v", err)
	}
}
