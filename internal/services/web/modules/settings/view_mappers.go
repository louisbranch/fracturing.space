package settings

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// settingsSideMenu centralizes this web behavior in one helper seam.
func settingsSideMenu(currentPath string, loc webtemplates.Localizer) *webtemplates.AppSideMenu {
	return &webtemplates.AppSideMenu{
		CurrentPath: currentPath,
		Items: []webtemplates.AppSideMenuItem{
			{
				Label:      webtemplates.T(loc, "layout.settings_user_profile"),
				URL:        routepath.AppSettingsProfile,
				MatchExact: true,
				IconID:     commonv1.IconId_ICON_ID_PROFILE,
			},
			{
				Label:      webtemplates.T(loc, "layout.locale"),
				URL:        routepath.AppSettingsLocale,
				MatchExact: true,
				IconID:     commonv1.IconId_ICON_ID_LOCALE,
			},
			{
				Label:      webtemplates.T(loc, "layout.settings_security"),
				URL:        routepath.AppSettingsSecurity,
				MatchExact: true,
				IconID:     commonv1.IconId_ICON_ID_KEY,
			},
			{
				Label:      webtemplates.T(loc, "layout.settings_ai_keys"),
				URL:        routepath.AppSettingsAIKeys,
				MatchExact: true,
				IconID:     commonv1.IconId_ICON_ID_AI,
			},
			{
				Label:      webtemplates.T(loc, "layout.settings_ai_agents"),
				URL:        routepath.AppSettingsAIAgents,
				MatchExact: true,
				IconID:     commonv1.IconId_ICON_ID_AI,
			},
		},
	}
}

// mapAIKeyTemplateRows maps settings AI key values into template rows.
func mapAIKeyTemplateRows(keys []SettingsAIKey) []webtemplates.SettingsAIKeyRow {
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

// mapPasskeyTemplateRows maps settings passkeys into template rows.
func mapPasskeyTemplateRows(passkeys []SettingsPasskey) []webtemplates.SettingsPasskeyRow {
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

// mapAIAgentCredentialTemplateOptions maps credential options into template options.
func mapAIAgentCredentialTemplateOptions(options []SettingsAICredentialOption) []webtemplates.SettingsAICredentialOption {
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
func mapAIModelTemplateOptions(models []SettingsAIModelOption) []webtemplates.SettingsAIModelOption {
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
func mapAIAgentTemplateRows(agents []SettingsAIAgent) []webtemplates.SettingsAIAgentRow {
	rows := make([]webtemplates.SettingsAIAgentRow, 0, len(agents))
	for _, agent := range agents {
		rows = append(rows, webtemplates.SettingsAIAgentRow{
			ID:           agent.ID,
			Name:         agent.Name,
			Provider:     agent.Provider,
			Model:        agent.Model,
			Status:       agent.Status,
			CreatedAt:    agent.CreatedAt,
			Instructions: agent.Instructions,
		})
	}
	return rows
}
