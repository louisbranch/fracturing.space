package charactermutationtransport

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// subclassStateFromProjection converts the projection-layer subclass state into
// the domain-layer representation used by transport payloads.
func subclassStateFromProjection(state *projectionstore.DaggerheartSubclassState) daggerheartstate.CharacterSubclassState {
	if state == nil {
		return daggerheartstate.CharacterSubclassState{}
	}
	return daggerheartstate.CharacterSubclassState{
		BattleRitualUsedThisLongRest:           state.BattleRitualUsedThisLongRest,
		GiftedPerformerRelaxingSongUses:        state.GiftedPerformerRelaxingSongUses,
		GiftedPerformerEpicSongUses:            state.GiftedPerformerEpicSongUses,
		GiftedPerformerHeartbreakingSongUses:   state.GiftedPerformerHeartbreakingSongUses,
		ContactsEverywhereUsesThisSession:      state.ContactsEverywhereUsesThisSession,
		ContactsEverywhereActionDieBonus:       state.ContactsEverywhereActionDieBonus,
		ContactsEverywhereDamageDiceBonusCount: state.ContactsEverywhereDamageDiceBonusCount,
		SparingTouchUsesThisLongRest:           state.SparingTouchUsesThisLongRest,
		ElementalistActionBonus:                state.ElementalistActionBonus,
		ElementalistDamageBonus:                state.ElementalistDamageBonus,
		TranscendenceActive:                    state.TranscendenceActive,
		TranscendenceTraitBonusTarget:          state.TranscendenceTraitBonusTarget,
		TranscendenceTraitBonusValue:           state.TranscendenceTraitBonusValue,
		TranscendenceProficiencyBonus:          state.TranscendenceProficiencyBonus,
		TranscendenceEvasionBonus:              state.TranscendenceEvasionBonus,
		TranscendenceSevereThresholdBonus:      state.TranscendenceSevereThresholdBonus,
		ClarityOfNatureUsedThisLongRest:        state.ClarityOfNatureUsedThisLongRest,
		ElementalChannel:                       state.ElementalChannel,
		NemesisTargetID:                        state.NemesisTargetID,
		RousingSpeechUsedThisLongRest:          state.RousingSpeechUsedThisLongRest,
		WardensProtectionUsedThisLongRest:      state.WardensProtectionUsedThisLongRest,
	}.Normalized()
}

// subclassStatePtr returns a pointer to a normalized copy of the subclass state.
func subclassStatePtr(state daggerheartstate.CharacterSubclassState) *daggerheartstate.CharacterSubclassState {
	normalized := state.Normalized()
	return &normalized
}

// hasUnlockedSubclassRank reports whether the character profile has unlocked
// the given subclass at or above the specified minimum rank (foundation,
// specialization, or mastery).
func hasUnlockedSubclassRank(profile projectionstore.DaggerheartCharacterProfile, subclassID, minimum string) bool {
	order := map[string]int{"foundation": 1, "specialization": 2, "mastery": 3}
	want := order[strings.TrimSpace(minimum)]
	if want == 0 {
		return false
	}
	for _, track := range profile.SubclassTracks {
		if strings.TrimSpace(track.SubclassID) != strings.TrimSpace(subclassID) {
			continue
		}
		if order[strings.TrimSpace(string(track.Rank))] >= want {
			return true
		}
	}
	if strings.TrimSpace(profile.SubclassID) == strings.TrimSpace(subclassID) && want <= 1 {
		return true
	}
	return false
}

// uniqueTrimmedIDs deduplicates and trims the given string slice, preserving
// insertion order. Empty strings are dropped.
func uniqueTrimmedIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// projectionConditionStatesToDomain converts projection-layer condition states
// into the domain rules representation.
func projectionConditionStatesToDomain(current []projectionstore.DaggerheartConditionState) []rules.ConditionState {
	if len(current) == 0 {
		return nil
	}
	next := make([]rules.ConditionState, 0, len(current))
	for _, condition := range current {
		entry := rules.ConditionState{
			ID:       condition.ID,
			Class:    rules.ConditionClass(condition.Class),
			Standard: condition.Standard,
			Code:     condition.Code,
			Label:    condition.Label,
			Source:   condition.Source,
			SourceID: condition.SourceID,
		}
		for _, trigger := range condition.ClearTriggers {
			entry.ClearTriggers = append(entry.ClearTriggers, rules.ConditionClearTrigger(trigger))
		}
		next = append(next, entry)
	}
	return next
}

// addStandardConditionState appends a standard condition (by code) to the given
// slice, normalizes, and returns the updated list plus the added entries.
func addStandardConditionState(current []rules.ConditionState, condition string) ([]rules.ConditionState, []rules.ConditionState, error) {
	return addStandardConditionStateWithOptions(current, condition)
}

// addStandardConditionStateWithOptions is the option-accepting variant of
// addStandardConditionState, forwarding functional options to the rules layer.
func addStandardConditionStateWithOptions(
	current []rules.ConditionState,
	condition string,
	options ...func(*rules.ConditionState),
) ([]rules.ConditionState, []rules.ConditionState, error) {
	next := append([]rules.ConditionState(nil), current...)
	entry, err := rules.StandardConditionState(condition, options...)
	if err != nil {
		return nil, nil, err
	}
	next = append(next, entry)
	normalized, err := rules.NormalizeConditionStates(next)
	if err != nil {
		return nil, nil, err
	}
	added, _ := rules.DiffConditionStates(current, normalized)
	return normalized, added, nil
}
