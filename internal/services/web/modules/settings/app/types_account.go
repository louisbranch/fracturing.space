package app

import (
	"strings"
	"unicode/utf8"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"golang.org/x/text/language"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// UserProfileNameMaxLength is the maximum allowed rune length for profile names.
const UserProfileNameMaxLength = 64

// SettingsProfile stores editable user profile settings.
type SettingsProfile struct {
	Username      string
	Name          string
	AvatarSetID   string
	AvatarAssetID string
	Pronouns      string
	Bio           string
}

// settingsLocale defines an internal contract used at this web package boundary.
type settingsLocale string

const (
	settingsLocaleUnspecified settingsLocale = ""
	settingsLocaleEnUS        settingsLocale = "en-US"
	settingsLocalePtBR        settingsLocale = "pt-BR"
)

var settingsLocaleByTag = map[string]settingsLocale{
	"en":    settingsLocaleEnUS,
	"en-us": settingsLocaleEnUS,
	"pt":    settingsLocalePtBR,
	"pt-br": settingsLocalePtBR,
}

// RequireUserID validates and returns a trimmed user ID, or returns an
// unauthorized error if it is blank.
func RequireUserID(userID string) (string, error) {
	return userid.Require(userID)
}

// parseSettingsLocale parses inbound values into package-safe forms.
func parseSettingsLocale(value string) (settingsLocale, bool) {
	tag, err := language.Parse(strings.TrimSpace(value))
	if err != nil {
		return settingsLocaleUnspecified, false
	}
	normalized := strings.ToLower(tag.String())
	if locale, ok := settingsLocaleByTag[normalized]; ok {
		return locale, true
	}
	return settingsLocaleUnspecified, false
}

// normalizeSettingsLocale centralizes this web behavior in one helper seam.
func normalizeSettingsLocale(value settingsLocale) settingsLocale {
	locale, ok := parseSettingsLocale(string(value))
	if ok {
		return locale
	}
	return settingsLocaleEnUS
}

// ParseLocale validates a locale and returns the normalized value.
func ParseLocale(value string) (string, bool) {
	locale, ok := parseSettingsLocale(value)
	if !ok {
		return "", false
	}
	return string(locale), true
}

// NormalizeLocale returns a supported locale value, defaulting to en-US.
func NormalizeLocale(value string) string {
	return string(normalizeSettingsLocale(settingsLocale(value)))
}

// validateNameLength centralizes this web behavior in one helper seam.
func validateNameLength(name string) error {
	if utf8.RuneCountInString(name) > UserProfileNameMaxLength {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.user_profile.error_name_too_long", "name is too long")
	}
	return nil
}
