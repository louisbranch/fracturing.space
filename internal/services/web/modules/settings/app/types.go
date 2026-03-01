package app

import (
	"context"
	"strings"
	"unicode/utf8"

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

// SettingsAIKey stores a credential row displayed in the AI keys page.
type SettingsAIKey struct {
	ID        string
	Label     string
	Provider  string
	Status    string
	CreatedAt string
	RevokedAt string
	CanRevoke bool
}

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

// Gateway loads and updates settings data for web handlers.
type Gateway interface {
	LoadProfile(context.Context, string) (SettingsProfile, error)
	SaveProfile(context.Context, string, SettingsProfile) error
	LoadLocale(context.Context, string) (string, error)
	SaveLocale(context.Context, string, string) error
	ListAIKeys(context.Context, string) ([]SettingsAIKey, error)
	CreateAIKey(context.Context, string, string, string) error
	RevokeAIKey(context.Context, string, string) error
}

// Service exposes settings orchestration methods used by transport handlers.
type Service interface {
	LoadProfile(context.Context, string) (SettingsProfile, error)
	SaveProfile(context.Context, string, SettingsProfile) error
	LoadLocale(context.Context, string) (string, error)
	SaveLocale(context.Context, string, string) error
	ListAIKeys(context.Context, string) ([]SettingsAIKey, error)
	CreateAIKey(context.Context, string, string, string) error
	RevokeAIKey(context.Context, string, string) error
}

// RequireUserID validates and returns a trimmed user ID, or returns an
// unauthorized error if it is blank.
func RequireUserID(userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", apperrors.EK(apperrors.KindUnauthorized, "error.web.message.user_id_is_required", "user id is required")
	}
	return userID, nil
}

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

func isSafeCredentialPathID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}

func validateNameLength(name string) error {
	if utf8.RuneCountInString(name) > UserProfileNameMaxLength {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.user_profile.error_name_too_long", "name is too long")
	}
	return nil
}
