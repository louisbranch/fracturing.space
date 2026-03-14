package settings

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// settingsSideMenu centralizes this web behavior in one helper seam.
func settingsSideMenu(currentPath string, loc webtemplates.Localizer, availability settingsSurfaceAvailability) *webtemplates.AppSideMenu {
	items := make([]webtemplates.AppSideMenuItem, 0, 5)
	if availability.profile {
		items = append(items, webtemplates.AppSideMenuItem{
			Label:      webtemplates.T(loc, "layout.settings_user_profile"),
			URL:        routepath.AppSettingsProfile,
			MatchExact: true,
			IconID:     commonv1.IconId_ICON_ID_PROFILE,
		})
	}
	if availability.locale {
		items = append(items, webtemplates.AppSideMenuItem{
			Label:      webtemplates.T(loc, "layout.locale"),
			URL:        routepath.AppSettingsLocale,
			MatchExact: true,
			IconID:     commonv1.IconId_ICON_ID_LOCALE,
		})
	}
	if availability.security {
		items = append(items, webtemplates.AppSideMenuItem{
			Label:      webtemplates.T(loc, "layout.settings_security"),
			URL:        routepath.AppSettingsSecurity,
			MatchExact: true,
			IconID:     commonv1.IconId_ICON_ID_KEY,
		})
	}
	if availability.aiKeys {
		items = append(items, webtemplates.AppSideMenuItem{
			Label:      webtemplates.T(loc, "layout.settings_ai_keys"),
			URL:        routepath.AppSettingsAIKeys,
			MatchExact: true,
			IconID:     commonv1.IconId_ICON_ID_AI,
		})
	}
	if availability.aiAgents {
		items = append(items, webtemplates.AppSideMenuItem{
			Label:      webtemplates.T(loc, "layout.settings_ai_agents"),
			URL:        routepath.AppSettingsAIAgents,
			MatchExact: true,
			IconID:     commonv1.IconId_ICON_ID_AI,
		})
	}
	return &webtemplates.AppSideMenu{CurrentPath: currentPath, Items: items}
}

// mapPasskeyTemplateRows maps settings passkeys into template rows.
func mapPasskeyTemplateRows(passkeys []settingsapp.SettingsPasskey) []webtemplates.SettingsPasskeyRow {
	rows := make([]webtemplates.SettingsPasskeyRow, 0, len(passkeys))
	for _, passkey := range passkeys {
		rows = append(rows, webtemplates.SettingsPasskeyRow{
			Number:     passkey.Number,
			CreatedAt:  passkey.CreatedAt,
			LastUsedAt: passkey.LastUsedAt,
		})
	}
	return rows
}

// mapAIKeyTemplateRows maps settings AI key values into template rows.
func mapAIKeyTemplateRows(keys []settingsapp.SettingsAIKey) []webtemplates.SettingsAIKeyRow {
	rows := make([]webtemplates.SettingsAIKeyRow, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, webtemplates.SettingsAIKeyRow{
			ID:        key.ID,
			Label:     key.Label,
			Provider:  key.Provider,
			Status:    key.Status,
			CreatedAt: key.CreatedAt,
			RevokedAt: key.RevokedAt,
			CanRevoke: key.CanRevoke,
		})
	}
	return rows
}

// mapAIAgentCredentialTemplateOptions maps credential options into template options.
func mapAIAgentCredentialTemplateOptions(options []settingsapp.SettingsAICredentialOption) []webtemplates.SettingsAICredentialOption {
	rows := make([]webtemplates.SettingsAICredentialOption, 0, len(options))
	for _, option := range options {
		rows = append(rows, webtemplates.SettingsAICredentialOption{
			ID:       option.ID,
			Label:    option.Label,
			Provider: option.Provider,
		})
	}
	return rows
}

// mapAIModelTemplateOptions maps provider-backed models into template options.
func mapAIModelTemplateOptions(models []settingsapp.SettingsAIModelOption) []webtemplates.SettingsAIModelOption {
	rows := make([]webtemplates.SettingsAIModelOption, 0, len(models))
	for _, model := range models {
		rows = append(rows, webtemplates.SettingsAIModelOption{
			ID:      model.ID,
			OwnedBy: model.OwnedBy,
		})
	}
	return rows
}

// mapAIAgentTemplateRows maps settings AI agents into template rows.
func mapAIAgentTemplateRows(agents []settingsapp.SettingsAIAgent) []webtemplates.SettingsAIAgentRow {
	rows := make([]webtemplates.SettingsAIAgentRow, 0, len(agents))
	for _, agent := range agents {
		rows = append(rows, webtemplates.SettingsAIAgentRow{
			ID:                  agent.ID,
			Label:               agent.Label,
			Provider:            agent.Provider,
			Model:               agent.Model,
			AuthState:           agent.AuthState,
			CanDelete:           agent.CanDelete,
			ActiveCampaignCount: agent.ActiveCampaignCount,
			CreatedAt:           agent.CreatedAt,
			Instructions:        agent.Instructions,
		})
	}
	return rows
}
