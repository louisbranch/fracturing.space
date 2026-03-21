package adapter

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func SubclassStateToProjection(value *daggerheartstate.CharacterSubclassState) *projectionstore.DaggerheartSubclassState {
	normalized := daggerheartstate.NormalizedSubclassStatePtr(value)
	if normalized == nil {
		return nil
	}
	return &projectionstore.DaggerheartSubclassState{
		BattleRitualUsedThisLongRest:           normalized.BattleRitualUsedThisLongRest,
		GiftedPerformerRelaxingSongUses:        normalized.GiftedPerformerRelaxingSongUses,
		GiftedPerformerEpicSongUses:            normalized.GiftedPerformerEpicSongUses,
		GiftedPerformerHeartbreakingSongUses:   normalized.GiftedPerformerHeartbreakingSongUses,
		ContactsEverywhereUsesThisSession:      normalized.ContactsEverywhereUsesThisSession,
		ContactsEverywhereActionDieBonus:       normalized.ContactsEverywhereActionDieBonus,
		ContactsEverywhereDamageDiceBonusCount: normalized.ContactsEverywhereDamageDiceBonusCount,
		SparingTouchUsesThisLongRest:           normalized.SparingTouchUsesThisLongRest,
		ElementalistActionBonus:                normalized.ElementalistActionBonus,
		ElementalistDamageBonus:                normalized.ElementalistDamageBonus,
		TranscendenceActive:                    normalized.TranscendenceActive,
		TranscendenceTraitBonusTarget:          normalized.TranscendenceTraitBonusTarget,
		TranscendenceTraitBonusValue:           normalized.TranscendenceTraitBonusValue,
		TranscendenceProficiencyBonus:          normalized.TranscendenceProficiencyBonus,
		TranscendenceEvasionBonus:              normalized.TranscendenceEvasionBonus,
		TranscendenceSevereThresholdBonus:      normalized.TranscendenceSevereThresholdBonus,
		ClarityOfNatureUsedThisLongRest:        normalized.ClarityOfNatureUsedThisLongRest,
		ElementalChannel:                       normalized.ElementalChannel,
		NemesisTargetID:                        normalized.NemesisTargetID,
		RousingSpeechUsedThisLongRest:          normalized.RousingSpeechUsedThisLongRest,
		WardensProtectionUsedThisLongRest:      normalized.WardensProtectionUsedThisLongRest,
	}
}

func ClassStateToProjection(value *daggerheartstate.CharacterClassState) *projectionstore.DaggerheartClassState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &projectionstore.DaggerheartClassState{
		AttackBonusUntilRest:       normalized.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest: normalized.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest: normalized.DifficultyPenaltyUntilRest,
		FocusTargetID:              normalized.FocusTargetID,
		ActiveBeastform:            ActiveBeastformToProjection(normalized.ActiveBeastform),
		StrangePatternsNumber:      normalized.StrangePatternsNumber,
		RallyDice:                  append([]int(nil), normalized.RallyDice...),
		PrayerDice:                 append([]int(nil), normalized.PrayerDice...),
		Unstoppable: projectionstore.DaggerheartUnstoppableState{
			Active:           normalized.Unstoppable.Active,
			CurrentValue:     normalized.Unstoppable.CurrentValue,
			DieSides:         normalized.Unstoppable.DieSides,
			UsedThisLongRest: normalized.Unstoppable.UsedThisLongRest,
		},
		ChannelRawPowerUsedThisLongRest: normalized.ChannelRawPowerUsedThisLongRest,
	}
}

func CompanionStateToProjection(value *daggerheartstate.CharacterCompanionState) *projectionstore.DaggerheartCompanionState {
	normalized := daggerheartstate.NormalizedCompanionStatePtr(value)
	if normalized == nil {
		return nil
	}
	return &projectionstore.DaggerheartCompanionState{
		Status:             normalized.Status,
		ActiveExperienceID: normalized.ActiveExperienceID,
	}
}

func ActiveBeastformToProjection(value *daggerheartstate.CharacterActiveBeastformState) *projectionstore.DaggerheartActiveBeastformState {
	normalized := daggerheartstate.NormalizedActiveBeastformPtr(value)
	if normalized == nil {
		return nil
	}
	damageDice := make([]projectionstore.DaggerheartDamageDie, 0, len(normalized.DamageDice))
	for _, die := range normalized.DamageDice {
		damageDice = append(damageDice, projectionstore.DaggerheartDamageDie{Count: die.Count, Sides: die.Sides})
	}
	return &projectionstore.DaggerheartActiveBeastformState{
		BeastformID:            normalized.BeastformID,
		BaseTrait:              normalized.BaseTrait,
		AttackTrait:            normalized.AttackTrait,
		TraitBonus:             normalized.TraitBonus,
		EvasionBonus:           normalized.EvasionBonus,
		AttackRange:            normalized.AttackRange,
		DamageDice:             damageDice,
		DamageBonus:            normalized.DamageBonus,
		DamageType:             normalized.DamageType,
		EvolutionTraitOverride: normalized.EvolutionTraitOverride,
		DropOnAnyHPMark:        normalized.DropOnAnyHPMark,
	}
}

func ClassStateFromProjection(value projectionstore.DaggerheartClassState) daggerheartstate.CharacterClassState {
	damageDice := []daggerheartstate.CharacterDamageDie(nil)
	active := daggerheartstate.NormalizedActiveBeastformPtr(nil)
	if value.ActiveBeastform != nil {
		damageDice = make([]daggerheartstate.CharacterDamageDie, 0, len(value.ActiveBeastform.DamageDice))
		for _, die := range value.ActiveBeastform.DamageDice {
			damageDice = append(damageDice, daggerheartstate.CharacterDamageDie{Count: die.Count, Sides: die.Sides})
		}
		active = &daggerheartstate.CharacterActiveBeastformState{
			BeastformID:            value.ActiveBeastform.BeastformID,
			BaseTrait:              value.ActiveBeastform.BaseTrait,
			AttackTrait:            value.ActiveBeastform.AttackTrait,
			TraitBonus:             value.ActiveBeastform.TraitBonus,
			EvasionBonus:           value.ActiveBeastform.EvasionBonus,
			AttackRange:            value.ActiveBeastform.AttackRange,
			DamageDice:             damageDice,
			DamageBonus:            value.ActiveBeastform.DamageBonus,
			DamageType:             value.ActiveBeastform.DamageType,
			EvolutionTraitOverride: value.ActiveBeastform.EvolutionTraitOverride,
			DropOnAnyHPMark:        value.ActiveBeastform.DropOnAnyHPMark,
		}
	}
	return daggerheartstate.CharacterClassState{
		AttackBonusUntilRest:            value.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest:      value.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest:      value.DifficultyPenaltyUntilRest,
		FocusTargetID:                   value.FocusTargetID,
		ActiveBeastform:                 active,
		StrangePatternsNumber:           value.StrangePatternsNumber,
		RallyDice:                       append([]int(nil), value.RallyDice...),
		PrayerDice:                      append([]int(nil), value.PrayerDice...),
		ChannelRawPowerUsedThisLongRest: value.ChannelRawPowerUsedThisLongRest,
		Unstoppable: daggerheartstate.CharacterUnstoppableState{
			Active:           value.Unstoppable.Active,
			CurrentValue:     value.Unstoppable.CurrentValue,
			DieSides:         value.Unstoppable.DieSides,
			UsedThisLongRest: value.Unstoppable.UsedThisLongRest,
		},
	}.Normalized()
}

func SubclassStateFromProjection(value *projectionstore.DaggerheartSubclassState) *daggerheartstate.CharacterSubclassState {
	if value == nil {
		return nil
	}
	return daggerheartstate.NormalizedSubclassStatePtr(&daggerheartstate.CharacterSubclassState{
		BattleRitualUsedThisLongRest:           value.BattleRitualUsedThisLongRest,
		GiftedPerformerRelaxingSongUses:        value.GiftedPerformerRelaxingSongUses,
		GiftedPerformerEpicSongUses:            value.GiftedPerformerEpicSongUses,
		GiftedPerformerHeartbreakingSongUses:   value.GiftedPerformerHeartbreakingSongUses,
		ContactsEverywhereUsesThisSession:      value.ContactsEverywhereUsesThisSession,
		ContactsEverywhereActionDieBonus:       value.ContactsEverywhereActionDieBonus,
		ContactsEverywhereDamageDiceBonusCount: value.ContactsEverywhereDamageDiceBonusCount,
		SparingTouchUsesThisLongRest:           value.SparingTouchUsesThisLongRest,
		ElementalistActionBonus:                value.ElementalistActionBonus,
		ElementalistDamageBonus:                value.ElementalistDamageBonus,
		TranscendenceActive:                    value.TranscendenceActive,
		TranscendenceTraitBonusTarget:          value.TranscendenceTraitBonusTarget,
		TranscendenceTraitBonusValue:           value.TranscendenceTraitBonusValue,
		TranscendenceProficiencyBonus:          value.TranscendenceProficiencyBonus,
		TranscendenceEvasionBonus:              value.TranscendenceEvasionBonus,
		TranscendenceSevereThresholdBonus:      value.TranscendenceSevereThresholdBonus,
		ClarityOfNatureUsedThisLongRest:        value.ClarityOfNatureUsedThisLongRest,
		ElementalChannel:                       value.ElementalChannel,
		NemesisTargetID:                        value.NemesisTargetID,
		RousingSpeechUsedThisLongRest:          value.RousingSpeechUsedThisLongRest,
		WardensProtectionUsedThisLongRest:      value.WardensProtectionUsedThisLongRest,
	})
}

func CompanionStateFromProjection(value *projectionstore.DaggerheartCompanionState) *daggerheartstate.CharacterCompanionState {
	if value == nil {
		return nil
	}
	return daggerheartstate.NormalizedCompanionStatePtr(&daggerheartstate.CharacterCompanionState{
		Status:             value.Status,
		ActiveExperienceID: value.ActiveExperienceID,
	})
}

// StatModifiersToProjection converts domain stat modifiers to projection form.
func StatModifiersToProjection(values []rules.StatModifierState) []projectionstore.DaggerheartStatModifier {
	if len(values) == 0 {
		return nil
	}
	result := make([]projectionstore.DaggerheartStatModifier, 0, len(values))
	for _, value := range values {
		entry := projectionstore.DaggerheartStatModifier{
			ID:       value.ID,
			Target:   string(value.Target),
			Delta:    value.Delta,
			Label:    value.Label,
			Source:   value.Source,
			SourceID: value.SourceID,
		}
		for _, trigger := range value.ClearTriggers {
			entry.ClearTriggers = append(entry.ClearTriggers, string(trigger))
		}
		result = append(result, entry)
	}
	return result
}

// StatModifiersFromProjection converts projection stat modifiers to domain form.
func StatModifiersFromProjection(values []projectionstore.DaggerheartStatModifier) []rules.StatModifierState {
	if len(values) == 0 {
		return nil
	}
	result := make([]rules.StatModifierState, 0, len(values))
	for _, value := range values {
		entry := rules.StatModifierState{
			ID:       value.ID,
			Target:   rules.StatModifierTarget(value.Target),
			Delta:    value.Delta,
			Label:    value.Label,
			Source:   value.Source,
			SourceID: value.SourceID,
		}
		for _, trigger := range value.ClearTriggers {
			entry.ClearTriggers = append(entry.ClearTriggers, rules.ConditionClearTrigger(trigger))
		}
		result = append(result, entry)
	}
	return result
}

func ConditionStatesToProjection(values []rules.ConditionState) []projectionstore.DaggerheartConditionState {
	if len(values) == 0 {
		return nil
	}
	result := make([]projectionstore.DaggerheartConditionState, 0, len(values))
	for _, value := range values {
		entry := projectionstore.DaggerheartConditionState{
			ID:       value.ID,
			Class:    string(value.Class),
			Standard: value.Standard,
			Code:     value.Code,
			Label:    value.Label,
			Source:   value.Source,
			SourceID: value.SourceID,
		}
		for _, trigger := range value.ClearTriggers {
			entry.ClearTriggers = append(entry.ClearTriggers, string(trigger))
		}
		result = append(result, entry)
	}
	return result
}
