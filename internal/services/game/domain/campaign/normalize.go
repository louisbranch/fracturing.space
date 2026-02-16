package campaign

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
)

// CreateInput describes the metadata needed to create a campaign.
type CreateInput struct {
	Name         string
	Locale       commonv1.Locale
	System       commonv1.GameSystem
	GmMode       GmMode
	Intent       Intent
	AccessPolicy AccessPolicy
	ThemePrompt  string
}

// NormalizeCreateInput trims and validates campaign input metadata.
func NormalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CreateInput{}, ErrEmptyName
	}
	if input.System == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return CreateInput{}, ErrInvalidGameSystem
	}
	if input.GmMode == GmModeUnspecified {
		return CreateInput{}, ErrInvalidGmMode
	}
	input.Locale = platformi18n.NormalizeLocale(input.Locale)
	if input.Intent == IntentUnspecified {
		input.Intent = IntentStandard
	}
	if input.AccessPolicy == AccessPolicyUnspecified {
		input.AccessPolicy = AccessPolicyPrivate
	}
	return input, nil
}
