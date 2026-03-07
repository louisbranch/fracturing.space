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
				Label:      webtemplates.T(loc, "layout.settings_ai_keys"),
				URL:        routepath.AppSettingsAIKeys,
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
