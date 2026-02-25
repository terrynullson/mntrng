package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/example/hls-monitoring-platform/internal/config"
)

const (
	defaultAccessCookieName  = "hm_access_token"
	defaultRefreshCookieName = "hm_refresh_token"
	defaultAuthCookiePath    = "/"
)

type authCookieConfig struct {
	accessName  string
	refreshName string
	path        string
	domain      string
	secure      bool
	sameSite    http.SameSite
}

func loadAuthCookieConfig() authCookieConfig {
	accessName := strings.TrimSpace(config.GetString("AUTH_ACCESS_COOKIE_NAME", defaultAccessCookieName))
	if accessName == "" {
		accessName = defaultAccessCookieName
	}

	refreshName := strings.TrimSpace(config.GetString("AUTH_REFRESH_COOKIE_NAME", defaultRefreshCookieName))
	if refreshName == "" {
		refreshName = defaultRefreshCookieName
	}

	path := strings.TrimSpace(config.GetString("AUTH_COOKIE_PATH", defaultAuthCookiePath))
	if path == "" {
		path = defaultAuthCookiePath
	}

	return authCookieConfig{
		accessName:  accessName,
		refreshName: refreshName,
		path:        path,
		domain:      strings.TrimSpace(config.GetString("AUTH_COOKIE_DOMAIN", "")),
		secure:      config.GetBool("AUTH_COOKIE_SECURE", true),
		sameSite:    parseCookieSameSite(config.GetString("AUTH_COOKIE_SAMESITE", "strict")),
	}
}

func parseCookieSameSite(raw string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

func (s *Server) setAuthCookies(w http.ResponseWriter, accessToken string, refreshToken string) {
	cfg := loadAuthCookieConfig()
	now := time.Now().UTC()
	s.setAuthCookie(w, cfg.accessName, accessToken, now.Add(s.authAccessTTL), cfg)
	s.setAuthCookie(w, cfg.refreshName, refreshToken, now.Add(s.authRefreshTTL), cfg)
}

func (s *Server) clearAuthCookies(w http.ResponseWriter) {
	cfg := loadAuthCookieConfig()
	s.clearAuthCookie(w, cfg.accessName, cfg)
	s.clearAuthCookie(w, cfg.refreshName, cfg)
}

func (s *Server) setAuthCookie(
	w http.ResponseWriter,
	name string,
	value string,
	expiresAt time.Time,
	cfg authCookieConfig,
) {
	maxAgeSeconds := int(time.Until(expiresAt).Seconds())
	if maxAgeSeconds < 0 {
		maxAgeSeconds = 0
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    strings.TrimSpace(value),
		Path:     cfg.path,
		Domain:   cfg.domain,
		HttpOnly: true,
		Secure:   cfg.secure,
		SameSite: cfg.sameSite,
		Expires:  expiresAt,
		MaxAge:   maxAgeSeconds,
	}
	http.SetCookie(w, cookie)
}

func (s *Server) clearAuthCookie(w http.ResponseWriter, name string, cfg authCookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     cfg.path,
		Domain:   cfg.domain,
		HttpOnly: true,
		Secure:   cfg.secure,
		SameSite: cfg.sameSite,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

func readTokenFromCookie(r *http.Request, cookieName string) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}
