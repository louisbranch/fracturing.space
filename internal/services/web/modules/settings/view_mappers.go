package settings

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

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
				IconID:     commonv1.IconId_ICON_ID_SETTINGS,
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
