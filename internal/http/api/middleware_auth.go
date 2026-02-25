package api

import (
	"net/http"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/example/hls-monitoring-platform/internal/domain"
	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1") {
			next.ServeHTTP(w, r)
			return
		}
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		accessToken := bearerTokenFromHeader(r.Header.Get("Authorization"))
		if accessToken == "" {
			accessToken = readTokenFromCookie(r, loadAuthCookieConfig().accessName)
		}
		if accessToken == "" {
			WriteJSONError(w, r, http.StatusUnauthorized, "unauthorized", "access token is required", map[string]interface{}{})
			return
		}

		authContext, err := s.authService.AuthenticateAccessToken(r.Context(), accessToken)
		if err != nil {
			writeAuthMiddlewareError(w, r, err)
			return
		}

		if denyCode, denyMessage, denied := evaluateAccessPolicy(r, authContext); denied {
			statusCode := http.StatusForbidden
			WriteJSONError(w, r, statusCode, denyCode, denyMessage, map[string]interface{}{})
			return
		}

		r = r.WithContext(withAuthContext(r.Context(), authContext))
		next.ServeHTTP(w, r)
	})
}

func isPublicPath(path string) bool {
	switch path {
	case "/api/v1/health":
		return true
	case "/api/v1/metrics":
		return config.GetBool("API_METRICS_PUBLIC", false)
	case "/api/v1/auth/register", "/api/v1/auth/login", "/api/v1/auth/refresh", "/api/v1/auth/telegram/login":
		return true
	default:
		return false
	}
}

func evaluateAccessPolicy(r *http.Request, authContext domain.AuthContext) (string, string, bool) {
	path := r.URL.Path

	if strings.HasPrefix(path, "/api/v1/admin") {
		if authContext.Role != domain.RoleSuperAdmin {
			return "forbidden", "super_admin role is required", true
		}
		return "", "", false
	}
	if path == "/api/v1/metrics" && authContext.Role != domain.RoleSuperAdmin {
		return "forbidden", "super_admin role is required", true
	}

	if strings.HasPrefix(path, "/api/v1/auth/") {
		return "", "", false
	}

	if path == "/api/v1/companies" {
		if authContext.Role != domain.RoleSuperAdmin {
			return "forbidden", "super_admin role is required", true
		}
		return "", "", false
	}

	if !strings.HasPrefix(path, "/api/v1/companies/") {
		return "", "", false
	}

	companyID, remainder, parseErr := ParseCompanyPath(path)
	if parseErr != "" {
		return "", "", false
	}

	if authContext.Role != domain.RoleSuperAdmin {
		if authContext.CompanyID == nil || *authContext.CompanyID != companyID {
			return "tenant_scope_required", "company scope mismatch", true
		}
	}

	if remainder == "telegram-delivery-settings" && authContext.Role == domain.RoleViewer {
		return "forbidden", "company_admin or super_admin role is required", true
	}
	if (remainder == "embed-whitelist" || strings.HasPrefix(remainder, "embed-whitelist/")) &&
		authContext.Role == domain.RoleViewer {
		return "forbidden", "company_admin or super_admin role is required", true
	}

	if authContext.Role == domain.RoleViewer && !isReadOnlyMethod(r.Method) {
		return "forbidden", "viewer role is read-only", true
	}

	if authContext.Role == domain.RoleCompanyAdmin && remainder == "" && r.Method == http.MethodDelete {
		return "forbidden", "company deletion requires super_admin role", true
	}

	return "", "", false
}

func isReadOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func writeAuthMiddlewareError(w http.ResponseWriter, r *http.Request, err error) {
	if serviceErr, ok := serviceapi.AsServiceError(err); ok {
		WriteJSONError(w, r, serviceErr.StatusCode, serviceErr.Code, serviceErr.Message, serviceErr.Details)
		return
	}
	WriteJSONError(w, r, http.StatusInternalServerError, "internal_error", "internal server error", map[string]interface{}{})
}
