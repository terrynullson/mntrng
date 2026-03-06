package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	serviceapi "github.com/terrynullson/hls_mntrng/internal/service/api"
)

func (s *Server) handleRegisterRequest(w http.ResponseWriter, r *http.Request) {
	var request registrationCreateRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.registrationService.SubmitRegistrationRequest(ctx, request)
	if err != nil {
		writeServiceError(w, r, "create registration request", err)
		return
	}

	if err := WriteJSON(w, http.StatusAccepted, item); err != nil {
		log.Printf("register request response encode error: %v", err)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := DecodeJSONBody(r, &request); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.authService.Login(ctx, request)
	if err != nil {
		writeServiceError(w, r, "login", err)
		return
	}
	s.setAuthCookies(w, item.AccessToken, item.RefreshToken)

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("login response encode error: %v", err)
	}
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if r.ContentLength > 0 {
		if err := DecodeJSONBody(r, &request); err != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
			return
		}
	}
	if strings.TrimSpace(request.RefreshToken) == "" {
		request.RefreshToken = readTokenFromCookie(r, loadAuthCookieConfig().refreshName)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.authService.Refresh(ctx, request)
	if err != nil {
		if serviceErr, ok := serviceapi.AsServiceError(err); ok && serviceErr.StatusCode == http.StatusUnauthorized {
			s.clearAuthCookies(w)
		}
		writeServiceError(w, r, "refresh", err)
		return
	}
	s.setAuthCookies(w, item.AccessToken, item.RefreshToken)

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("refresh response encode error: %v", err)
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	authContext, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "authentication context is missing", map[string]interface{}{})
		return
	}

	var request logoutRequest
	if r.ContentLength > 0 {
		if err := DecodeJSONBody(r, &request); err != nil {
			WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.authService.Logout(ctx, authContext, request); err != nil {
		writeServiceError(w, r, "logout", err)
		return
	}
	s.clearAuthCookies(w)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	authContext, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "authentication context is missing", map[string]interface{}{})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.authService.Me(ctx, authContext.UserID)
	if err != nil {
		writeServiceError(w, r, "me", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("me response encode error: %v", err)
	}
}

func (s *Server) handleTelegramLink(w http.ResponseWriter, r *http.Request) {
	authContext, ok := authContextFromRequest(r)
	if !ok {
		WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "authentication context is missing", map[string]interface{}{})
		return
	}

	var payload map[string]string
	if err := DecodeJSONBody(r, &payload); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.authService.LinkTelegram(ctx, authContext.UserID, payload); err != nil {
		writeServiceError(w, r, "telegram link", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTelegramLogin(w http.ResponseWriter, r *http.Request) {
	var payload map[string]string
	if err := DecodeJSONBody(r, &payload); err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := s.authService.TelegramLogin(ctx, payload)
	if err != nil {
		writeServiceError(w, r, "telegram login", err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, item); err != nil {
		log.Printf("telegram login response encode error: %v", err)
	}
}
