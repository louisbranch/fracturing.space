package state

import "strings"

const (
	ElementalChannelAir   = "air"
	ElementalChannelEarth = "earth"
	ElementalChannelFire  = "fire"
	ElementalChannelWater = "water"
)

// CharacterSubclassState stores mutable subclass-owned runtime state outside
// the static Daggerheart profile.
type CharacterSubclassState struct {
	BattleRitualUsedThisLongRest           bool   `json:"battle_ritual_used_this_long_rest,omitempty"`
	GiftedPerformerRelaxingSongUses        int    `json:"gifted_performer_relaxing_song_uses,omitempty"`
	GiftedPerformerEpicSongUses            int    `json:"gifted_performer_epic_song_uses,omitempty"`
	GiftedPerformerHeartbreakingSongUses   int    `json:"gifted_performer_heartbreaking_song_uses,omitempty"`
	ContactsEverywhereUsesThisSession      int    `json:"contacts_everywhere_uses_this_session,omitempty"`
	ContactsEverywhereActionDieBonus       int    `json:"contacts_everywhere_action_die_bonus,omitempty"`
	ContactsEverywhereDamageDiceBonusCount int    `json:"contacts_everywhere_damage_dice_bonus_count,omitempty"`
	SparingTouchUsesThisLongRest           int    `json:"sparing_touch_uses_this_long_rest,omitempty"`
	ElementalistActionBonus                int    `json:"elementalist_action_bonus,omitempty"`
	ElementalistDamageBonus                int    `json:"elementalist_damage_bonus,omitempty"`
	TranscendenceActive                    bool   `json:"transcendence_active,omitempty"`
	TranscendenceTraitBonusTarget          string `json:"transcendence_trait_bonus_target,omitempty"`
	TranscendenceTraitBonusValue           int    `json:"transcendence_trait_bonus_value,omitempty"`
	TranscendenceProficiencyBonus          int    `json:"transcendence_proficiency_bonus,omitempty"`
	TranscendenceEvasionBonus              int    `json:"transcendence_evasion_bonus,omitempty"`
	TranscendenceSevereThresholdBonus      int    `json:"transcendence_severe_threshold_bonus,omitempty"`
	ClarityOfNatureUsedThisLongRest        bool   `json:"clarity_of_nature_used_this_long_rest,omitempty"`
	ElementalChannel                       string `json:"elemental_channel,omitempty"`
	NemesisTargetID                        string `json:"nemesis_target_id,omitempty"`
	RousingSpeechUsedThisLongRest          bool   `json:"rousing_speech_used_this_long_rest,omitempty"`
	WardensProtectionUsedThisLongRest      bool   `json:"wardens_protection_used_this_long_rest,omitempty"`
}

// Normalized clamps invalid values so all write paths persist the same
// subclass-runtime shape.
func (s CharacterSubclassState) Normalized() CharacterSubclassState {
	normalized := s
	if normalized.GiftedPerformerRelaxingSongUses < 0 {
		normalized.GiftedPerformerRelaxingSongUses = 0
	}
	if normalized.GiftedPerformerEpicSongUses < 0 {
		normalized.GiftedPerformerEpicSongUses = 0
	}
	if normalized.GiftedPerformerHeartbreakingSongUses < 0 {
		normalized.GiftedPerformerHeartbreakingSongUses = 0
	}
	if normalized.ContactsEverywhereUsesThisSession < 0 {
		normalized.ContactsEverywhereUsesThisSession = 0
	}
	if normalized.ContactsEverywhereActionDieBonus < 0 {
		normalized.ContactsEverywhereActionDieBonus = 0
	}
	if normalized.ContactsEverywhereDamageDiceBonusCount < 0 {
		normalized.ContactsEverywhereDamageDiceBonusCount = 0
	}
	if normalized.SparingTouchUsesThisLongRest < 0 {
		normalized.SparingTouchUsesThisLongRest = 0
	}
	if normalized.ElementalistActionBonus < 0 {
		normalized.ElementalistActionBonus = 0
	}
	if normalized.ElementalistDamageBonus < 0 {
		normalized.ElementalistDamageBonus = 0
	}
	normalized.TranscendenceTraitBonusTarget = strings.TrimSpace(normalized.TranscendenceTraitBonusTarget)
	if normalized.TranscendenceTraitBonusValue < 0 {
		normalized.TranscendenceTraitBonusValue = 0
	}
	if normalized.TranscendenceProficiencyBonus < 0 {
		normalized.TranscendenceProficiencyBonus = 0
	}
	if normalized.TranscendenceEvasionBonus < 0 {
		normalized.TranscendenceEvasionBonus = 0
	}
	if normalized.TranscendenceSevereThresholdBonus < 0 {
		normalized.TranscendenceSevereThresholdBonus = 0
	}
	if !normalized.TranscendenceActive {
		normalized.TranscendenceTraitBonusTarget = ""
		normalized.TranscendenceTraitBonusValue = 0
		normalized.TranscendenceProficiencyBonus = 0
		normalized.TranscendenceEvasionBonus = 0
		normalized.TranscendenceSevereThresholdBonus = 0
	}
	switch strings.ToLower(strings.TrimSpace(normalized.ElementalChannel)) {
	case "", ElementalChannelAir, ElementalChannelEarth, ElementalChannelFire, ElementalChannelWater:
		normalized.ElementalChannel = strings.ToLower(strings.TrimSpace(normalized.ElementalChannel))
	default:
		normalized.ElementalChannel = ""
	}
	normalized.NemesisTargetID = strings.TrimSpace(normalized.NemesisTargetID)
	return normalized
}

// IsZero reports whether the subclass state carries no mutable runtime data.
func (s CharacterSubclassState) IsZero() bool {
	normalized := s.Normalized()
	return !normalized.BattleRitualUsedThisLongRest &&
		normalized.GiftedPerformerRelaxingSongUses == 0 &&
		normalized.GiftedPerformerEpicSongUses == 0 &&
		normalized.GiftedPerformerHeartbreakingSongUses == 0 &&
		normalized.ContactsEverywhereUsesThisSession == 0 &&
		normalized.ContactsEverywhereActionDieBonus == 0 &&
		normalized.ContactsEverywhereDamageDiceBonusCount == 0 &&
		normalized.SparingTouchUsesThisLongRest == 0 &&
		normalized.ElementalistActionBonus == 0 &&
		normalized.ElementalistDamageBonus == 0 &&
		!normalized.TranscendenceActive &&
		normalized.TranscendenceTraitBonusTarget == "" &&
		normalized.TranscendenceTraitBonusValue == 0 &&
		normalized.TranscendenceProficiencyBonus == 0 &&
		normalized.TranscendenceEvasionBonus == 0 &&
		normalized.TranscendenceSevereThresholdBonus == 0 &&
		!normalized.ClarityOfNatureUsedThisLongRest &&
		normalized.ElementalChannel == "" &&
		normalized.NemesisTargetID == "" &&
		!normalized.RousingSpeechUsedThisLongRest &&
		!normalized.WardensProtectionUsedThisLongRest
}

// NormalizedSubclassStatePtr normalizes a subclass state pointer, returning
// nil when the input is nil.
func NormalizedSubclassStatePtr(value *CharacterSubclassState) *CharacterSubclassState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &normalized
}
