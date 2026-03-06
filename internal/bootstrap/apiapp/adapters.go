package apiapp

import (
	"database/sql"
	"net/http"

	"github.com/terrynullson/mntrng/internal/integration/registrationnotify"
	"github.com/terrynullson/mntrng/internal/repo/postgres"
	serviceapi "github.com/terrynullson/mntrng/internal/service/api"
	"github.com/terrynullson/mntrng/internal/telegram"
)

type adapters struct {
	companyStore          serviceapi.CompanyStore
	projectStore          serviceapi.ProjectStore
	streamStore           serviceapi.StreamStore
	checkJobStore         serviceapi.CheckJobStore
	checkResultStore      serviceapi.CheckResultStore
	aiIncidentStore       serviceapi.AIIncidentStore
	streamFavoriteStore   serviceapi.StreamFavoriteStore
	incidentStore         serviceapi.IncidentStore
	telegramSettingsStore serviceapi.TelegramSettingsStore
	embedWhitelistStore   serviceapi.EmbedWhitelistStore
	authStore             serviceapi.AuthStore
	registrationStore     serviceapi.RegistrationStore
	registrationNotifier  serviceapi.RegistrationNotifier
}

func buildAdapters(db *sql.DB, runtime RuntimeConfig) adapters {
	telegramClient := telegram.NewClient(&http.Client{Timeout: runtime.TelegramHTTPTimeout})
	return adapters{
		companyStore:          postgres.NewAPICompanyRepo(db),
		projectStore:          postgres.NewAPIProjectRepo(db),
		streamStore:           postgres.NewAPIStreamRepo(db),
		checkJobStore:         postgres.NewAPICheckJobRepo(db),
		checkResultStore:      postgres.NewAPICheckResultRepo(db),
		aiIncidentStore:       postgres.NewAPIAIIncidentRepo(db),
		streamFavoriteStore:   postgres.NewAPIStreamFavoriteRepo(db),
		incidentStore:         postgres.NewAPIIncidentRepo(db),
		telegramSettingsStore: postgres.NewAPITelegramSettingsRepo(db),
		embedWhitelistStore:   postgres.NewAPIEmbedWhitelistRepo(db),
		authStore:             postgres.NewAPIAuthRepo(db),
		registrationStore:     postgres.NewAPIRegistrationRepo(db),
		registrationNotifier: registrationnotify.NewTelegramNotifier(
			telegramClient,
			runtime.TelegramBotTokenDefault,
			runtime.SuperAdminTelegramChatID,
		),
	}
}
