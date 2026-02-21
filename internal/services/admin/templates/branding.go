package templates

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
)

// AppName returns the canonical product name.
func AppName() string {
	return branding.AppName
}

// ComposeAdminPageTitle normalizes existing app-brand suffixes and appends the admin suffix.
func ComposeAdminPageTitle(title string) string {
	normalizedTitle := strings.TrimSpace(title)
	baseAppName := AppName()

	normalizedTitle = strings.TrimSpace(strings.TrimSuffix(normalizedTitle, " - "+baseAppName))
	normalizedTitle = strings.TrimSpace(strings.TrimSuffix(normalizedTitle, " | "+baseAppName))
	if normalizedTitle == "" {
		return "Admin | " + baseAppName
	}

	return sharedtemplates.ComposePageTitle(normalizedTitle + " - Admin")
}
