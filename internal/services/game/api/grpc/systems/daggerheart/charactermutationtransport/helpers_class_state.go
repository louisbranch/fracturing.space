package charactermutationtransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// classStateFromProjection converts the projection-layer class state into the
// domain-layer representation used by transport payloads.
func classStateFromProjection(state projectionstore.DaggerheartClassState) daggerheartstate.CharacterClassState {
	return daggerheartstate.CharacterClassState{
		AttackBonusUntilRest:            state.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest:      state.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest:      state.DifficultyPenaltyUntilRest,
		FocusTargetID:                   state.FocusTargetID,
		ActiveBeastform:                 activeBeastformFromProjection(state.ActiveBeastform),
		StrangePatternsNumber:           state.StrangePatternsNumber,
		RallyDice:                       append([]int(nil), state.RallyDice...),
		PrayerDice:                      append([]int(nil), state.PrayerDice...),
		ChannelRawPowerUsedThisLongRest: state.ChannelRawPowerUsedThisLongRest,
		Unstoppable: daggerheartstate.CharacterUnstoppableState{
			Active:           state.Unstoppable.Active,
			CurrentValue:     state.Unstoppable.CurrentValue,
			DieSides:         state.Unstoppable.DieSides,
			UsedThisLongRest: state.Unstoppable.UsedThisLongRest,
		},
	}.Normalized()
}

// classStatePtr returns a pointer to a normalized copy of the class state.
func classStatePtr(state daggerheartstate.CharacterClassState) *daggerheartstate.CharacterClassState {
	normalized := state.Normalized()
	return &normalized
}

// activeBeastformFromProjection converts the projection-layer beastform state
// into the domain-layer representation.
func activeBeastformFromProjection(state *projectionstore.DaggerheartActiveBeastformState) *daggerheartstate.CharacterActiveBeastformState {
	if state == nil {
		return nil
	}
	damageDice := make([]daggerheartstate.CharacterDamageDie, 0, len(state.DamageDice))
	for _, die := range state.DamageDice {
		damageDice = append(damageDice, daggerheartstate.CharacterDamageDie{Count: die.Count, Sides: die.Sides})
	}
	return &daggerheartstate.CharacterActiveBeastformState{
		BeastformID:            state.BeastformID,
		BaseTrait:              state.BaseTrait,
		AttackTrait:            state.AttackTrait,
		TraitBonus:             state.TraitBonus,
		EvasionBonus:           state.EvasionBonus,
		AttackRange:            state.AttackRange,
		DamageDice:             damageDice,
		DamageBonus:            state.DamageBonus,
		DamageType:             state.DamageType,
		EvolutionTraitOverride: state.EvolutionTraitOverride,
		DropOnAnyHPMark:        state.DropOnAnyHPMark,
	}
}
