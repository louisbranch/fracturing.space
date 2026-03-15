package projection

import (
	"reflect"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/reducer"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
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
	domainState.Conditions = append([]string(nil), state.Conditions...)
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
		Conditions:     append([]string(nil), state.Conditions...),
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
	return StorageCharacterStateFromDomain(&domainState), nil
}

// ApplyConditionPatch applies a normalized condition list to state.
func ApplyConditionPatch(state projectionstore.DaggerheartCharacterState, armorMax int, conditions []string) projectionstore.DaggerheartCharacterState {
	domainState := CharacterStateFromStorage(state, armorMax)
	reducer.ApplyConditionPatch(&domainState, conditions)
	return StorageCharacterStateFromDomain(&domainState)
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
	return StorageCharacterStateFromDomain(&domainState), nil
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
	return StorageCharacterStateFromDomain(&domainState), nil
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
	if !changed {
		return state, false
	}
	return StorageCharacterStateFromDomain(&domainState), true
}
