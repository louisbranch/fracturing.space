package projection

import (
	"reflect"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/reducer"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// FallbackArmorMaxFromState derives a conservative armor cap from persisted
// state when profile data is unavailable.
func FallbackArmorMaxFromState(state projectionstore.DaggerheartCharacterState) int {
	temporaryArmor := 0
	for _, bucket := range state.TemporaryArmor {
		if bucket.Amount > 0 {
			temporaryArmor += bucket.Amount
		}
	}
	armorMax := state.Armor - temporaryArmor
	if armorMax < 0 {
		return 0
	}
	return armorMax
}

// CharacterStateFromStorage converts persisted read-model state into
// mechanics state for deterministic projection transforms.
func CharacterStateFromStorage(state projectionstore.DaggerheartCharacterState, armorMax int) mechanics.CharacterState {
	domainState := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  state.CampaignID,
		CharacterID: state.CharacterID,
		HP:          state.Hp,
		HPMax:       mechanics.HPMaxCap,
		Hope:        state.Hope,
		HopeMax:     state.HopeMax,
		Stress:      state.Stress,
		StressMax:   mechanics.StressMaxCap,
		Armor:       state.Armor,
		ArmorMax:    armorMax,
		LifeState:   state.LifeState,
	})
	domainState.Conditions = projectionConditionCodes(state.Conditions)
	domainState.ArmorBonus = make([]mechanics.TemporaryArmorBucket, 0, len(state.TemporaryArmor))
	for _, bucket := range state.TemporaryArmor {
		domainState.ArmorBonus = append(domainState.ArmorBonus, mechanics.TemporaryArmorBucket{
			Source:   strings.TrimSpace(bucket.Source),
			Duration: strings.TrimSpace(bucket.Duration),
			SourceID: strings.TrimSpace(bucket.SourceID),
			Amount:   bucket.Amount,
		})
	}
	if strings.TrimSpace(domainState.LifeState) == "" {
		domainState.LifeState = mechanics.LifeStateAlive
	}
	return *domainState
}

// StorageCharacterStateFromDomain converts mechanics state back to persisted
// read-model form.
func StorageCharacterStateFromDomain(state *mechanics.CharacterState) projectionstore.DaggerheartCharacterState {
	if state == nil {
		return projectionstore.DaggerheartCharacterState{}
	}
	temporaryArmor := make([]projectionstore.DaggerheartTemporaryArmor, 0, len(state.ArmorBonus))
	for _, bucket := range state.ArmorBonus {
		temporaryArmor = append(temporaryArmor, projectionstore.DaggerheartTemporaryArmor{
			Source:   strings.TrimSpace(bucket.Source),
			Duration: strings.TrimSpace(bucket.Duration),
			SourceID: strings.TrimSpace(bucket.SourceID),
			Amount:   bucket.Amount,
		})
	}
	return projectionstore.DaggerheartCharacterState{
		CampaignID:     strings.TrimSpace(state.CampaignID),
		CharacterID:    strings.TrimSpace(state.CharacterID),
		Hp:             state.HP,
		Hope:           state.Hope,
		HopeMax:        state.HopeMax,
		Stress:         state.Stress,
		Armor:          state.Armor,
		Conditions:     projectionStandardConditionsFromCodes(state.Conditions),
		TemporaryArmor: temporaryArmor,
		LifeState:      state.LifeState,
	}
}

// ApplyStatePatch applies a character state patch and normalizes bounds.
func ApplyStatePatch(
	state projectionstore.DaggerheartCharacterState,
	armorMax int,
	hpAfter, hopeAfter, hopeMaxAfter, stressAfter, armorAfter *int,
	lifeStateAfter *string,
	classStateAfter *projectionstore.DaggerheartClassState,
	subclassStateAfter *projectionstore.DaggerheartSubclassState,
	companionStateAfter *projectionstore.DaggerheartCompanionState,
	impenetrableUsedThisShortRestAfter *bool,
) (projectionstore.DaggerheartCharacterState, error) {
	domainState := CharacterStateFromStorage(state, armorMax)
	reducer.ApplyCharacterStatePatch(&domainState, reducer.CharacterStatePatch{
		HPAfter:        hpAfter,
		HopeAfter:      hopeAfter,
		HopeMaxAfter:   hopeMaxAfter,
		StressAfter:    stressAfter,
		ArmorAfter:     armorAfter,
		LifeStateAfter: lifeStateAfter,
	})
	if err := reducer.NormalizeAndValidateCharacterState(&domainState); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}
	next := StorageCharacterStateFromDomain(&domainState)
	next.ClassState = state.ClassState
	if classStateAfter != nil {
		next.ClassState = *classStateAfter
	}
	next.SubclassState = state.SubclassState
	if subclassStateAfter != nil {
		next.SubclassState = subclassStateAfter
		if next.SubclassState.BattleRitualUsedThisLongRest == false &&
			next.SubclassState.GiftedPerformerRelaxingSongUses == 0 &&
			next.SubclassState.GiftedPerformerEpicSongUses == 0 &&
			next.SubclassState.GiftedPerformerHeartbreakingSongUses == 0 &&
			next.SubclassState.ContactsEverywhereUsesThisSession == 0 &&
			next.SubclassState.ContactsEverywhereActionDieBonus == 0 &&
			next.SubclassState.ContactsEverywhereDamageDiceBonusCount == 0 &&
			next.SubclassState.SparingTouchUsesThisLongRest == 0 &&
			next.SubclassState.ElementalistActionBonus == 0 &&
			next.SubclassState.ElementalistDamageBonus == 0 &&
			next.SubclassState.TranscendenceActive == false &&
			next.SubclassState.TranscendenceTraitBonusTarget == "" &&
			next.SubclassState.TranscendenceTraitBonusValue == 0 &&
			next.SubclassState.TranscendenceProficiencyBonus == 0 &&
			next.SubclassState.TranscendenceEvasionBonus == 0 &&
			next.SubclassState.TranscendenceSevereThresholdBonus == 0 &&
			next.SubclassState.ClarityOfNatureUsedThisLongRest == false &&
			next.SubclassState.ElementalChannel == "" &&
			next.SubclassState.NemesisTargetID == "" &&
			next.SubclassState.RousingSpeechUsedThisLongRest == false &&
			next.SubclassState.WardensProtectionUsedThisLongRest == false {
			next.SubclassState = nil
		}
	}
	next.CompanionState = state.CompanionState
	if companionStateAfter != nil {
		next.CompanionState = companionStateAfter
	}
	next.ImpenetrableUsedThisShortRest = state.ImpenetrableUsedThisShortRest
	if impenetrableUsedThisShortRestAfter != nil {
		next.ImpenetrableUsedThisShortRest = *impenetrableUsedThisShortRestAfter
	}
	return next, nil
}

// ApplyConditionPatch applies a normalized condition list to state.
func ApplyConditionPatch(state projectionstore.DaggerheartCharacterState, armorMax int, conditions []projectionstore.DaggerheartConditionState) projectionstore.DaggerheartCharacterState {
	domainState := CharacterStateFromStorage(state, armorMax)
	reducer.ApplyConditionPatch(&domainState, projectionConditionCodes(conditions))
	next := StorageCharacterStateFromDomain(&domainState)
	next.Conditions = append([]projectionstore.DaggerheartConditionState(nil), conditions...)
	next.ClassState = state.ClassState
	next.SubclassState = state.SubclassState
	next.CompanionState = state.CompanionState
	next.ImpenetrableUsedThisShortRest = state.ImpenetrableUsedThisShortRest
	return next
}

// ApplyTemporaryArmor applies a temporary armor patch and normalizes bounds.
func ApplyTemporaryArmor(
	state projectionstore.DaggerheartCharacterState,
	armorMax int,
	source, duration, sourceID string,
	amount int,
) (projectionstore.DaggerheartCharacterState, error) {
	domainState := CharacterStateFromStorage(state, armorMax)
	reducer.ApplyTemporaryArmor(&domainState, reducer.TemporaryArmorPatch{
		Source:   strings.TrimSpace(source),
		Duration: strings.TrimSpace(duration),
		SourceID: strings.TrimSpace(sourceID),
		Amount:   amount,
	})
	if err := reducer.NormalizeAndValidateCharacterState(&domainState); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}
	next := StorageCharacterStateFromDomain(&domainState)
	next.ClassState = state.ClassState
	next.SubclassState = state.SubclassState
	next.CompanionState = state.CompanionState
	next.ImpenetrableUsedThisShortRest = state.ImpenetrableUsedThisShortRest
	return next, nil
}

// ApplyDowntimeMove applies downtime move effects and normalizes bounds.
func ApplyDowntimeMove(
	state projectionstore.DaggerheartCharacterState,
	armorMax int,
	move string,
	hopeAfter, stressAfter, armorAfter *int,
) (projectionstore.DaggerheartCharacterState, error) {
	domainState := CharacterStateFromStorage(state, armorMax)
	reducer.ApplyDowntimeMove(&domainState, move, hopeAfter, stressAfter, armorAfter)
	if err := reducer.NormalizeAndValidateCharacterState(&domainState); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}
	next := StorageCharacterStateFromDomain(&domainState)
	next.ClassState = state.ClassState
	next.SubclassState = state.SubclassState
	next.CompanionState = state.CompanionState
	next.ImpenetrableUsedThisShortRest = state.ImpenetrableUsedThisShortRest
	return next, nil
}

// ClearRestTemporaryArmor removes temporary armor by rest duration and returns
// whether persisted state should be updated.
func ClearRestTemporaryArmor(
	state projectionstore.DaggerheartCharacterState,
	armorMax int,
	clearShortRest bool,
	clearLongRest bool,
) (projectionstore.DaggerheartCharacterState, bool) {
	domainState := CharacterStateFromStorage(state, armorMax)
	beforeArmor := domainState.Armor
	beforeArmorBonus := append([]mechanics.TemporaryArmorBucket(nil), domainState.ArmorBonus...)
	reducer.ClearRestTemporaryArmor(&domainState, clearShortRest, clearLongRest)
	changed := beforeArmor != domainState.Armor || !reflect.DeepEqual(beforeArmorBonus, domainState.ArmorBonus)
	if (clearShortRest || clearLongRest) && state.ImpenetrableUsedThisShortRest {
		changed = true
	}
	if !changed {
		return state, false
	}
	next := StorageCharacterStateFromDomain(&domainState)
	next.Conditions = append([]projectionstore.DaggerheartConditionState(nil), state.Conditions...)
	if clearShortRest {
		next.Conditions = clearProjectionConditionsByTrigger(next.Conditions, "short_rest")
	}
	if clearLongRest {
		next.Conditions = clearProjectionConditionsByTrigger(next.Conditions, "long_rest")
	}
	next.ClassState = state.ClassState
	next.SubclassState = state.SubclassState
	next.CompanionState = state.CompanionState
	next.ClassState.AttackBonusUntilRest = 0
	next.ClassState.EvasionBonusUntilHitOrRest = 0
	next.ClassState.DifficultyPenaltyUntilRest = 0
	if next.SubclassState != nil {
		next.SubclassState.ElementalistActionBonus = 0
		next.SubclassState.ElementalistDamageBonus = 0
		next.SubclassState.TranscendenceActive = false
		next.SubclassState.TranscendenceTraitBonusTarget = ""
		next.SubclassState.TranscendenceTraitBonusValue = 0
		next.SubclassState.TranscendenceProficiencyBonus = 0
		next.SubclassState.TranscendenceEvasionBonus = 0
		next.SubclassState.TranscendenceSevereThresholdBonus = 0
		next.SubclassState.ClarityOfNatureUsedThisLongRest = false
		next.SubclassState.ElementalChannel = ""
		next.SubclassState.NemesisTargetID = ""
	}
	if clearLongRest {
		next.ClassState.Unstoppable.UsedThisLongRest = false
		next.ClassState.ChannelRawPowerUsedThisLongRest = false
		if next.SubclassState != nil {
			next.SubclassState.BattleRitualUsedThisLongRest = false
			next.SubclassState.GiftedPerformerRelaxingSongUses = 0
			next.SubclassState.GiftedPerformerEpicSongUses = 0
			next.SubclassState.GiftedPerformerHeartbreakingSongUses = 0
			next.SubclassState.SparingTouchUsesThisLongRest = 0
			next.SubclassState.ClarityOfNatureUsedThisLongRest = false
			next.SubclassState.RousingSpeechUsedThisLongRest = false
			next.SubclassState.WardensProtectionUsedThisLongRest = false
		}
	}
	next.ImpenetrableUsedThisShortRest = false
	return next, true
}

func projectionConditionCodes(values []projectionstore.DaggerheartConditionState) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		switch {
		case strings.TrimSpace(value.Code) != "":
			result = append(result, strings.TrimSpace(value.Code))
		case strings.TrimSpace(value.Standard) != "":
			result = append(result, strings.TrimSpace(value.Standard))
		case strings.TrimSpace(value.ID) != "":
			result = append(result, strings.TrimSpace(value.ID))
		}
	}
	return result
}

func projectionStandardConditionsFromCodes(values []string) []projectionstore.DaggerheartConditionState {
	if len(values) == 0 {
		return []projectionstore.DaggerheartConditionState{}
	}
	result := make([]projectionstore.DaggerheartConditionState, 0, len(values))
	for _, value := range values {
		code := strings.ToLower(strings.TrimSpace(value))
		if code == "" {
			continue
		}
		result = append(result, projectionstore.DaggerheartConditionState{
			ID:       code,
			Class:    "standard",
			Standard: code,
			Code:     code,
			Label:    code,
		})
	}
	return result
}

func clearProjectionConditionsByTrigger(values []projectionstore.DaggerheartConditionState, trigger string) []projectionstore.DaggerheartConditionState {
	if len(values) == 0 || strings.TrimSpace(trigger) == "" {
		return append([]projectionstore.DaggerheartConditionState(nil), values...)
	}
	result := make([]projectionstore.DaggerheartConditionState, 0, len(values))
	for _, value := range values {
		if !projectionConditionHasTrigger(value, trigger) {
			result = append(result, value)
		}
	}
	return result
}

func projectionConditionHasTrigger(value projectionstore.DaggerheartConditionState, trigger string) bool {
	for _, current := range value.ClearTriggers {
		if strings.EqualFold(strings.TrimSpace(current), trigger) {
			return true
		}
	}
	return false
}
