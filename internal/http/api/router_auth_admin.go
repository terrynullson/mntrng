package api

import (
	"net/http"
	"strings"
)

func routeAuthRegister(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodPost {
		handlers.HandleRegisterRequest(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodPost)
}

func routeAuthLogin(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodPost {
		handlers.HandleLogin(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodPost)
}

func routeAuthRefresh(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodPost {
		handlers.HandleRefresh(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodPost)
}

func routeAuthLogout(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodPost {
		handlers.HandleLogout(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodPost)
}

func routeAuthMe(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodGet {
		handlers.HandleMe(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodGet)
}

func routeAuthTelegramLogin(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodPost {
		handlers.HandleTelegramLogin(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodPost)
}

func routeAuthTelegramLink(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodPost {
		handlers.HandleTelegramLink(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodPost)
}

func routeAdminRegistrationRequestsCollection(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodGet {
		handlers.HandleListPendingRegistration(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodGet)
}

func routeAdminRegistrationRequests(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	const prefix = "/api/v1/admin/registration-requests/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		writeRouterPathNotFound(w, r)
		return
	}

	rawPath := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(rawPath, "/")
	if len(parts) != 2 {
		writeRouterPathNotFound(w, r)
		return
	}

	requestID, err := parsePositiveID(parts[0])
	if err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid request_id", map[string]interface{}{"path": r.URL.Path})
		return
	}

	switch parts[1] {
	case "approve":
		if r.Method == http.MethodPost {
			handlers.HandleApproveRegistrationRequest(w, r, requestID)
			return
		}
		WriteMethodNotAllowed(w, r, http.MethodPost)
	case "reject":
		if r.Method == http.MethodPost {
			handlers.HandleRejectRegistrationRequest(w, r, requestID)
			return
		}
		WriteMethodNotAllowed(w, r, http.MethodPost)
	default:
		writeRouterPathNotFound(w, r)
	}
}

func routeAdminUsers(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	const prefix = "/api/v1/admin/users/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		writeRouterPathNotFound(w, r)
		return
	}

	rawPath := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(rawPath, "/")
	if len(parts) != 2 {
		writeRouterPathNotFound(w, r)
		return
	}

	userID, err := parsePositiveID(parts[0])
	if err != nil {
		WriteJSONError(w, r, http.StatusBadRequest, "validation_error", "invalid user_id", map[string]interface{}{"path": r.URL.Path})
		return
	}

	switch parts[1] {
	case "role":
		if r.Method == http.MethodPatch {
			handlers.HandleChangeUserRole(w, r, userID)
			return
		}
		WriteMethodNotAllowed(w, r, http.MethodPatch)
	case "status":
		if r.Method == http.MethodPatch {
			handlers.HandleChangeUserStatus(w, r, userID)
			return
		}
		WriteMethodNotAllowed(w, r, http.MethodPatch)
	default:
		writeRouterPathNotFound(w, r)
	}
}

func routeAdminUsersCollection(w http.ResponseWriter, r *http.Request, handlers RouterHandlers) {
	if r.Method == http.MethodGet {
		handlers.HandleListUsers(w, r)
		return
	}
	WriteMethodNotAllowed(w, r, http.MethodGet)
}
