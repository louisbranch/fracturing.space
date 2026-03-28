package settings

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// settingsSideMenu centralizes this web behavior in one helper seam.
func settingsSideMenu(currentPath string, loc webtemplates.Localizer, availability settingsSurfaceAvailability) *webtemplates.AppSideMenu {
	items := make([]webtemplates.AppSideMenuItem, 0, 3)
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
			IconID:     commonv1.IconId_ICON_ID_LOCK,
		})
	}
	aiItems := make([]webtemplates.AppSideMenuItem, 0, 2)
	if availability.aiKeys {
		aiItems = append(aiItems, webtemplates.AppSideMenuItem{
			Label:      webtemplates.T(loc, "layout.settings_ai_keys"),
			URL:        routepath.AppSettingsAIKeys,
			MatchExact: true,
			IconID:     commonv1.IconId_ICON_ID_KEY,
		})
	}
	if availability.aiAgents {
		aiItems = append(aiItems, webtemplates.AppSideMenuItem{
			Label:      webtemplates.T(loc, "layout.settings_ai_agents"),
			URL:        routepath.AppSettingsAIAgents,
			MatchExact: true,
			IconID:     commonv1.IconId_ICON_ID_BRAIN_COG,
		})
	}
	menu := &webtemplates.AppSideMenu{CurrentPath: currentPath, Items: items}
	if len(aiItems) > 0 {
		menu.Groups = []webtemplates.AppSideMenuGroup{{
			Title: webtemplates.T(loc, "layout.settings_ai"),
			Items: aiItems,
		}}
	}
	return menu
}

// mapPasskeyTemplateRows maps settings passkeys into template rows.
func mapPasskeyTemplateRows(passkeys []settingsapp.SettingsPasskey) []SettingsPasskeyRow {
	rows := make([]SettingsPasskeyRow, 0, len(passkeys))
	for _, passkey := range passkeys {
		rows = append(rows, SettingsPasskeyRow{
			Number:     passkey.Number,
			CreatedAt:  passkey.CreatedAt,
			LastUsedAt: passkey.LastUsedAt,
		})
	}
	return rows
}

// mapAIKeyTemplateRows maps settings AI key values into template rows.
func mapAIKeyTemplateRows(keys []settingsapp.SettingsAIKey) []SettingsAIKeyRow {
	rows := make([]SettingsAIKeyRow, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, SettingsAIKeyRow{
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
func mapAIAgentCredentialTemplateOptions(options []settingsapp.SettingsAICredentialOption) []SettingsAICredentialOption {
	rows := make([]SettingsAICredentialOption, 0, len(options))
	for _, option := range options {
		rows = append(rows, SettingsAICredentialOption{
			ID:       option.ID,
			Label:    option.Label,
			Provider: option.Provider,
		})
	}
	return rows
}

// mapAIModelTemplateOptions maps provider-backed models into template options.
func mapAIModelTemplateOptions(models []settingsapp.SettingsAIModelOption) []SettingsAIModelOption {
	rows := make([]SettingsAIModelOption, 0, len(models))
	for _, model := range models {
		rows = append(rows, SettingsAIModelOption{ID: model.ID})
	}
	return rows
}

// mapAIAgentTemplateRows maps settings AI agents into template rows.
func mapAIAgentTemplateRows(agents []settingsapp.SettingsAIAgent) []SettingsAIAgentRow {
	rows := make([]SettingsAIAgentRow, 0, len(agents))
	for _, agent := range agents {
		rows = append(rows, SettingsAIAgentRow{
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
