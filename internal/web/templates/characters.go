package templates

import (
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
)

// CharacterRow represents a single row in the characters table.
type CharacterRow struct {
	ID         string
	CampaignID string
	Name       string
	Kind       string
	Controller string
}

// CharacterSheetView holds the data for rendering a character sheet.
type CharacterSheetView struct {
	CampaignID   string
	CampaignName string
	Character    *statev1.Character
	Controller   string
	RecentEvents []EventRow
	CreatedAt    string
	UpdatedAt    string
}

// formatCharacterKind returns a display label for a character kind.
func formatCharacterKind(kind statev1.CharacterKind, loc Localizer) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return T(loc, "label.pc")
	case statev1.CharacterKind_NPC:
		return T(loc, "label.npc")
	default:
		return T(loc, "label.unspecified")
	}
}
