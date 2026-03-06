package apiapp

import (
	httpapi "github.com/terrynullson/mntrng/internal/http/api"
	serviceapi "github.com/terrynullson/mntrng/internal/service/api"
)

func buildPorts(adapters adapters, runtime RuntimeConfig) httpapi.Ports {
	return httpapi.Ports{
		Company:          serviceapi.NewCompanyService(adapters.companyStore),
		Project:          serviceapi.NewProjectService(adapters.projectStore),
		Stream:           serviceapi.NewStreamService(adapters.streamStore),
		CheckJob:         serviceapi.NewCheckJobService(adapters.checkJobStore),
		CheckResult:      serviceapi.NewCheckResultService(adapters.checkResultStore),
		AIIncident:       serviceapi.NewAIIncidentService(adapters.aiIncidentStore),
		StreamFavorite:   serviceapi.NewStreamFavoriteService(adapters.streamFavoriteStore),
		Incident:         serviceapi.NewIncidentService(adapters.incidentStore),
		TelegramSettings: serviceapi.NewTelegramSettingsService(adapters.telegramSettingsStore),
		EmbedWhitelist:   serviceapi.NewEmbedWhitelistService(adapters.embedWhitelistStore),
		Auth: serviceapi.NewAuthService(adapters.authStore, serviceapi.AuthConfig{
			AccessTTL:          runtime.AuthAccessTTL,
			RefreshTTL:         runtime.AuthRefreshTTL,
			TelegramBotToken:   runtime.TelegramBotTokenDefault,
			TelegramAuthMaxAge: runtime.TelegramAuthMaxAge,
		}),
		Registration: serviceapi.NewRegistrationService(adapters.registrationStore, adapters.registrationNotifier),
	}
}
