package campaign

import (
	"strings"
)

// CreateInput describes the metadata needed to create a campaign.
type CreateInput struct {
	Name         string
	Locale       string
	System       GameSystem
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
	normalizedSystem, ok := NormalizeGameSystem(input.System.String())
	if !ok {
		return CreateInput{}, ErrInvalidGameSystem
	}
	input.System = normalizedSystem
	if input.GmMode == GmModeUnspecified {
		input.GmMode = GmModeAI
	}
	input.Locale = normalizeCampaignLocale(input.Locale)
	if input.Intent == IntentUnspecified {
		input.Intent = IntentStandard
	}
	if input.AccessPolicy == AccessPolicyUnspecified {
		input.AccessPolicy = AccessPolicyPrivate
	}
	return input, nil
}
