package characters

import (
	"net/url"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// parseCreateCharacterInput normalizes the character-create form into app input.
func parseCreateCharacterInput(form url.Values) (campaignapp.CreateCharacterInput, error) {
	kindValue := strings.TrimSpace(form.Get("kind"))
	if kindValue == "" {
		kindValue = "pc"
	}
	kind, ok := parseAppCharacterKind(kindValue)
	if !ok {
		return campaignapp.CreateCharacterInput{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid")
	}
	return campaignapp.CreateCharacterInput{
		Name:     strings.TrimSpace(form.Get("name")),
		Pronouns: strings.TrimSpace(form.Get("pronouns")),
		Kind:     kind,
	}, nil
}

// parseUpdateCharacterInput normalizes the character-edit form into app input.
func parseUpdateCharacterInput(form url.Values) campaignapp.UpdateCharacterInput {
	return campaignapp.UpdateCharacterInput{
		Name:     strings.TrimSpace(form.Get("name")),
		Pronouns: strings.TrimSpace(form.Get("pronouns")),
	}
}

// parseSetCharacterOwnerInput reads the selected participant owner.
func parseSetCharacterOwnerInput(form url.Values) string {
	return strings.TrimSpace(form.Get("participant_id"))
}

// parseAppCharacterKind maps transport values onto app-level character kinds.
func parseAppCharacterKind(value string) (campaignapp.CharacterKind, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pc", "character_kind_pc":
		return campaignapp.CharacterKindPC, true
	case "npc", "character_kind_npc":
		return campaignapp.CharacterKindNPC, true
	default:
		return campaignapp.CharacterKindUnspecified, false
	}
}
