package render

import (
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// campaignCharacterDetailURL centralizes character detail links for render-owned cards.
func campaignCharacterDetailURL(campaignID string, character CharacterView) string {
	campaignID = strings.TrimSpace(campaignID)
	characterID := strings.TrimSpace(character.ID)
	if campaignID == "" || characterID == "" {
		return ""
	}
	return routepath.AppCampaignCharacter(campaignID, characterID)
}

// campaignCharacterEditURL centralizes character edit links for detail pages.
func campaignCharacterEditURL(campaignID string, characterID string) string {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return ""
	}
	return routepath.AppCampaignCharacterEdit(campaignID, characterID)
}

// campaignCharacterSheetTitle derives the character-creation panel title from the campaign system.
func campaignCharacterSheetTitle(loc Localizer, system string) string {
	system = strings.TrimSpace(campaignOverviewSystem(loc, system))
	if system == "" {
		system = T(loc, "game.campaign.system_unspecified")
	}
	return system + " " + T(loc, "game.character_detail.character_sheet_suffix")
}

// campaignCharacterHasDaggerheartSummary guards Daggerheart-only metadata sections.
func campaignCharacterHasDaggerheartSummary(character CharacterView) bool {
	if character.Daggerheart == nil {
		return false
	}
	return strings.TrimSpace(character.Daggerheart.ClassName) != "" &&
		strings.TrimSpace(character.Daggerheart.SubclassName) != "" &&
		strings.TrimSpace(character.Daggerheart.HeritageName) != "" &&
		strings.TrimSpace(character.Daggerheart.CommunityName) != "" &&
		character.Daggerheart.Level > 0
}

// campaignCharacterDaggerheartLevelAttr exposes level as a stable data attribute.
func campaignCharacterDaggerheartLevelAttr(character CharacterView) string {
	if !campaignCharacterHasDaggerheartSummary(character) {
		return ""
	}
	return strconv.FormatInt(int64(character.Daggerheart.Level), 10)
}

// campaignCharacterControlOptionLabel keeps controller reassignment labels stable.
func campaignCharacterControlOptionLabel(loc Localizer, option CharacterControlOptionView) string {
	if strings.TrimSpace(option.ParticipantID) == "" {
		return T(loc, "game.participants.value_unassigned")
	}
	label := strings.TrimSpace(option.Label)
	if label == "" {
		return strings.TrimSpace(option.ParticipantID)
	}
	return label
}

// campaignCharacterPronounPresets keeps character-edit suggestions in the render seam.
func campaignCharacterPronounPresets(loc Localizer) []string {
	return []string{
		T(loc, "game.participants.value_they_them"),
		T(loc, "game.participants.value_he_him"),
		T(loc, "game.participants.value_she_her"),
		T(loc, "game.participants.value_it_its"),
	}
}

// campaignCharacterDisplayName preserves the detail-page fallback title for unnamed characters.
func campaignCharacterDisplayName(loc Localizer, character CharacterView) string {
	name := strings.TrimSpace(character.Name)
	if name != "" {
		return name
	}
	return T(loc, "game.character_detail.title")
}
