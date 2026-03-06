package apiapp

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/terrynullson/mntrng/internal/config"
	httpapi "github.com/terrynullson/mntrng/internal/http/api"
	"github.com/terrynullson/mntrng/internal/ratelimit"
	"github.com/terrynullson/mntrng/internal/repo/postgres"
	serviceapi "github.com/terrynullson/mntrng/internal/service/api"
	"github.com/terrynullson/mntrng/internal/telegram"
)

func NewHTTPServer(addr string, db *sql.DB, limiter ratelimit.Limiter) *http.Server {
	authAccessTTL := time.Duration(config.IntAtLeast(config.GetInt("AUTH_ACCESS_TTL_MIN", 15), 1)) * time.Minute
	authRefreshTTL := time.Duration(config.IntAtLeast(config.GetInt("AUTH_REFRESH_TTL_DAYS", 30), 1)) * 24 * time.Hour

	authRepo := postgres.NewAPIAuthRepo(db)
	registrationRepo := postgres.NewAPIRegistrationRepo(db)

	telegramHTTPTimeoutMS := config.GetInt("TELEGRAM_HTTP_TIMEOUT_MS", 5000)
	if telegramHTTPTimeoutMS <= 0 {
		telegramHTTPTimeoutMS = 5000
	}

	telegramClient := telegram.NewClient(&http.Client{
		Timeout: time.Duration(telegramHTTPTimeoutMS) * time.Millisecond,
	})
	registrationNotifier := httpapi.NewRegistrationNotifier(
		telegramClient,
		config.GetString("TELEGRAM_BOT_TOKEN_DEFAULT", ""),
		config.GetString("SUPER_ADMIN_TELEGRAM_CHAT_ID", ""),
	)

	services := httpapi.ServiceSet{
		CompanyService:          serviceapi.NewCompanyService(postgres.NewAPICompanyRepo(db)),
		ProjectService:          serviceapi.NewProjectService(postgres.NewAPIProjectRepo(db)),
		StreamService:           serviceapi.NewStreamService(postgres.NewAPIStreamRepo(db)),
		CheckJobService:         serviceapi.NewCheckJobService(postgres.NewAPICheckJobRepo(db)),
		CheckResultService:      serviceapi.NewCheckResultService(postgres.NewAPICheckResultRepo(db)),
		AIIncidentService:       serviceapi.NewAIIncidentService(postgres.NewAPIAIIncidentRepo(db)),
		StreamFavoriteService:   serviceapi.NewStreamFavoriteService(postgres.NewAPIStreamFavoriteRepo(db)),
		IncidentService:         serviceapi.NewIncidentService(postgres.NewAPIIncidentRepo(db)),
		TelegramSettingsService: serviceapi.NewTelegramSettingsService(postgres.NewAPITelegramSettingsRepo(db)),
		EmbedWhitelistService:   serviceapi.NewEmbedWhitelistService(postgres.NewAPIEmbedWhitelistRepo(db)),
		AuthService: serviceapi.NewAuthService(authRepo, serviceapi.AuthConfig{
			AccessTTL:          authAccessTTL,
			RefreshTTL:         authRefreshTTL,
			TelegramBotToken:   config.GetString("TELEGRAM_BOT_TOKEN_DEFAULT", ""),
			TelegramAuthMaxAge: time.Duration(config.GetInt("AUTH_TELEGRAM_MAX_AGE_SEC", 600)) * time.Second,
		}),
		RegistrationService: serviceapi.NewRegistrationService(registrationRepo, registrationNotifier),
	}

	server := httpapi.NewServer(db, services, httpapi.AuthTTLConfig{
		AccessTTL:  authAccessTTL,
		RefreshTTL: authRefreshTTL,
	})
	return httpapi.NewHTTPServer(addr, server, limiter)
}
