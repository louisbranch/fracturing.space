package render

import (
	"strconv"
	"strings"
)

// campaignParticipantCardClass keeps participant highlight styling localized to the render seam.
func campaignParticipantCardClass(participant ParticipantView) string {
	if participant.IsViewer {
		return "card bg-base-100 border border-primary shadow-sm md:card-side"
	}
	return "card bg-base-100 border border-base-300 shadow-sm md:card-side"
}

// campaignParticipantViewerAttr exposes viewer state as a stable data-attribute value.
func campaignParticipantViewerAttr(participant ParticipantView) string {
	return strconv.FormatBool(participant.IsViewer)
}

// campaignCharacterCardClass keeps owned-character highlight styling localized to the render seam.
func campaignCharacterCardClass(character CharacterView) string {
	if character.OwnedByViewer {
		return "card bg-base-100 border border-primary shadow-sm md:card-side"
	}
	return "card bg-base-100 border border-base-300 shadow-sm md:card-side"
}

// campaignCharacterOwnedByViewerAttr exposes viewer-ownership state as a stable data-attribute value.
func campaignCharacterOwnedByViewerAttr(character CharacterView) string {
	return strconv.FormatBool(character.OwnedByViewer)
}

// campaignCharacterAliases renders aliases in the same display shape as the old shared fragment.
func campaignCharacterAliases(value []string) string {
	if len(value) == 0 {
		return ""
	}
	return strings.Join(value, ", ")
}

// campaignSystemLabel maps persisted system identifiers to contributor-facing copy.
func campaignSystemLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	case "daggerheart":
		return T(loc, "game.campaigns.system_daggerheart")
	default:
		return raw
	}
}

// campaignGMModeLabel maps GM mode values to localized overview labels.
func campaignGMModeLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "", "unspecified":
		return T(loc, "game.campaign.gm_mode_unspecified")
	case "human":
		return T(loc, "game.create.field_gm_mode_human")
	case "ai":
		return T(loc, "game.create.field_gm_mode_ai")
	case "hybrid":
		return T(loc, "game.create.field_gm_mode_hybrid")
	default:
		return raw
	}
}

// campaignActionsLocked exposes the detail-page mutation lock state to templates.
func campaignActionsLocked(locked bool) bool {
	return locked
}

// campaignParticipantRoleLabel maps participant roles to localized card and form copy.
func campaignParticipantRoleLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "gm":
		return T(loc, "game.participants.value.gm")
	case "player":
		return T(loc, "game.participants.value.player")
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	default:
		return raw
	}
}

// campaignParticipantAccessLabel maps participant access values to localized labels.
func campaignParticipantAccessLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "member":
		return T(loc, "game.participants.value.member")
	case "manager":
		return T(loc, "game.participants.value.manager")
	case "owner":
		return T(loc, "game.participants.value.owner")
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	default:
		return raw
	}
}

// campaignParticipantControllerLabel maps controller values to localized participant copy.
func campaignParticipantControllerLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "human":
		return T(loc, "game.participants.value.human")
	case "ai":
		return T(loc, "game.participants.value_ai")
	case "unassigned":
		return T(loc, "game.participants.value_unassigned")
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	default:
		return raw
	}
}

// participantPronounsLabel preserves the display mapping used across participant and character cards.
func participantPronounsLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return raw
	case "she/her":
		return T(loc, "game.participants.value_she_her")
	case "he/him":
		return T(loc, "game.participants.value_he_him")
	case "they/them":
		return T(loc, "game.participants.value_they_them")
	case "it/its":
		return T(loc, "game.participants.value_it_its")
	default:
		return raw
	}
}

// campaignCharacterKindLabel maps character kind values to localized detail copy.
func campaignCharacterKindLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "pc":
		return T(loc, "game.characters.value_pc")
	case "npc":
		return T(loc, "game.characters.value_npc")
	case "", "unspecified":
		return T(loc, "game.character_detail.kind_unspecified")
	default:
		return raw
	}
}
