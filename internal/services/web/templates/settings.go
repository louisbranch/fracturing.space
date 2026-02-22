package templates

import (
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// AIKeyRow is one AI key row rendered in settings.
type AIKeyRow struct {
	ID        string
	Label     string
	Provider  string
	Status    string
	CreatedAt string
	RevokedAt string
	CanRevoke bool
}

// AIKeysPageState captures form and listing state for the AI keys page.
type AIKeysPageState struct {
	FormLabel    string
	FormProvider string
	ErrorMessage string
	Keys         []AIKeyRow
}

// UserProfileSettingsPageState captures form state for user profile settings.
type UserProfileSettingsPageState struct {
	Username      string
	Name          string
	AvatarSetID   string
	AvatarAssetID string
	Bio           string
	ErrorMessage  string
}

// SettingsLayoutOptions returns layout options for the root settings page.
func SettingsLayoutOptions(page PageContext) LayoutOptions {
	options := LayoutOptionsForPage(page, "layout.settings", true)
	options.CustomBreadcrumbs = []sharedtemplates.BreadcrumbItem{}
	options.ChromeMenu = SettingsMenu(page)
	return options
}

// SettingsAIKeysLayoutOptions returns layout options for the AI keys settings page.
func SettingsAIKeysLayoutOptions(page PageContext) LayoutOptions {
	options := LayoutOptionsForPage(page, "layout.settings_ai_keys", true)
	options.CustomBreadcrumbs = []sharedtemplates.BreadcrumbItem{
		{Label: T(page.Loc, "layout.settings"), URL: routepath.AppSettings},
		{Label: T(page.Loc, "layout.settings_ai_keys"), URL: ""},
	}
	options.ChromeMenu = SettingsMenu(page)
	return options
}

// SettingsUserProfileLayoutOptions returns layout options for the user profile settings page.
func SettingsUserProfileLayoutOptions(page PageContext) LayoutOptions {
	options := LayoutOptionsForPage(page, "layout.settings_user_profile", true)
	options.CustomBreadcrumbs = []sharedtemplates.BreadcrumbItem{
		{Label: T(page.Loc, "layout.settings"), URL: routepath.AppSettings},
		{Label: T(page.Loc, "layout.settings_user_profile"), URL: ""},
	}
	options.ChromeMenu = SettingsMenu(page)
	return options
}
