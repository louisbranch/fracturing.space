package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/reducer"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// snapshotStatePatch carries optional field overrides for a character's
// snapshot state. Pointer fields that are nil are left unchanged.
type snapshotStatePatch struct {
	HP             *int
	Hope           *int
	HopeMax        *int
	Stress         *int
	Armor          *int
	LifeState      *string
	ClassState     *daggerheartstate.CharacterClassState
	SubclassState  *daggerheartstate.CharacterSubclassState
	CompanionState *daggerheartstate.CharacterCompanionState
}

func touchCharacter(state *daggerheartstate.SnapshotState, rawID ids.CharacterID) {
	characterID := normalize.ID(rawID)
	if characterID == "" {
		return
	}
	cs := state.CharacterStates[characterID]
	cs.CampaignID = state.CampaignID.String()
	cs.CharacterID = characterID.String()
	state.CharacterStates[characterID] = cs
}

func applyCharacterStatePatched(state *daggerheartstate.SnapshotState, p payload.CharacterStatePatchedPayload) {
	applyStatePatch(state, p.CharacterID, snapshotStatePatch{
		HP:            p.HP,
		Hope:          p.Hope,
		HopeMax:       p.HopeMax,
		Stress:        p.Stress,
		Armor:         p.Armor,
		LifeState:     p.LifeState,
		ClassState:    p.ClassState,
		SubclassState: p.SubclassState,
	})
}

func applyStatePatch(state *daggerheartstate.SnapshotState, characterID ids.CharacterID, patch snapshotStatePatch) {
	characterID = normalize.ID(characterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyCharacterStatePatch(&characterState, reducer.CharacterStatePatch{
		HPAfter:        patch.HP,
		HopeAfter:      patch.Hope,
		HopeMaxAfter:   patch.HopeMax,
		StressAfter:    patch.Stress,
		ArmorAfter:     patch.Armor,
		LifeStateAfter: patch.LifeState,
	})
	state.CharacterStates[characterID] = characterState
	if patch.ClassState != nil {
		state.CharacterClassStates[characterID] = patch.ClassState.Normalized()
	}
	if patch.SubclassState != nil {
		normalized := patch.SubclassState.Normalized()
		if normalized.IsZero() {
			delete(state.CharacterSubclassStates, characterID)
		} else {
			state.CharacterSubclassStates[characterID] = normalized
		}
	}
	if patch.CompanionState != nil {
		normalized := patch.CompanionState.Normalized()
		if normalized.IsZero() {
			delete(state.CharacterCompanions, characterID)
		} else {
			state.CharacterCompanions[characterID] = normalized
		}
	}
}

func applyCharacterConditionsChanged(state *daggerheartstate.SnapshotState, p payload.ConditionChangedPayload) {
	characterID := normalize.ID(p.CharacterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyConditionPatch(&characterState, rules.ConditionCodes(p.Conditions))
	state.CharacterStates[characterID] = characterState
}

func applyCharacterLoadoutSwapped(state *daggerheartstate.SnapshotState, p payload.LoadoutSwappedPayload) {
	characterID := normalize.ID(p.CharacterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyLoadoutSwap(&characterState, p.Stress)
	state.CharacterStates[characterID] = characterState
}

func applyCharacterTemporaryArmorApplied(state *daggerheartstate.SnapshotState, p payload.CharacterTemporaryArmorAppliedPayload) {
	characterID := normalize.ID(p.CharacterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyTemporaryArmor(&characterState, reducer.TemporaryArmorPatch{
		Source:   normalize.String(p.Source),
		Duration: normalize.String(p.Duration),
		SourceID: normalize.String(p.SourceID),
		Amount:   p.Amount,
	})
	state.CharacterStates[characterID] = characterState
}

func clearRestTemporaryArmor(state *daggerheartstate.SnapshotState, rawID string, clearShortRest bool, clearLongRest bool) {
	characterID := ids.CharacterID(normalize.String(rawID))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ClearRestTemporaryArmor(&characterState, clearShortRest, clearLongRest)
	state.CharacterStates[characterID] = characterState
}

func clearRestStatModifiers(state *daggerheartstate.SnapshotState, rawID ids.CharacterID, clearShortRest, clearLongRest bool) {
	characterID := normalize.ID(rawID)
	if characterID == "" {
		return
	}
	modifiers := state.CharacterStatModifiers[characterID]
	if len(modifiers) == 0 {
		return
	}
	if clearShortRest {
		modifiers, _ = rules.ClearStatModifiersByTrigger(modifiers, rules.ConditionClearTriggerShortRest)
	}
	if clearLongRest {
		modifiers, _ = rules.ClearStatModifiersByTrigger(modifiers, rules.ConditionClearTriggerLongRest)
	}
	state.CharacterStatModifiers[characterID] = modifiers
}

func applySceneCountdownUpsert(state *daggerheartstate.SnapshotState, countdownID ids.CountdownID, mutate func(*daggerheartstate.SceneCountdownState)) {
	state.EnsureMaps()
	trimmed := normalize.ID(countdownID)
	if trimmed == "" {
		return
	}
	countdownState := state.SceneCountdownStates[trimmed]
	countdownState.CampaignID = state.CampaignID
	countdownState.CountdownID = trimmed
	if mutate != nil {
		mutate(&countdownState)
	}
	state.SceneCountdownStates[trimmed] = countdownState
}

func deleteSceneCountdownState(state *daggerheartstate.SnapshotState, countdownID ids.CountdownID) {
	state.EnsureMaps()
	trimmed := normalize.ID(countdownID)
	if trimmed == "" {
		return
	}
	delete(state.SceneCountdownStates, trimmed)
}

func applyCampaignCountdownUpsert(state *daggerheartstate.SnapshotState, countdownID ids.CountdownID, mutate func(*daggerheartstate.CampaignCountdownState)) {
	state.EnsureMaps()
	trimmed := normalize.ID(countdownID)
	if trimmed == "" {
		return
	}
	countdownState := state.CampaignCountdownStates[trimmed]
	countdownState.CampaignID = state.CampaignID
	countdownState.CountdownID = trimmed
	if mutate != nil {
		mutate(&countdownState)
	}
	state.CampaignCountdownStates[trimmed] = countdownState
}

func deleteCampaignCountdownState(state *daggerheartstate.SnapshotState, countdownID ids.CountdownID) {
	state.EnsureMaps()
	trimmed := normalize.ID(countdownID)
	if trimmed == "" {
		return
	}
	delete(state.CampaignCountdownStates, trimmed)
}

func applyDamageApplied(state *daggerheartstate.SnapshotState, rawID ids.CharacterID, hpAfter, stressAfter, armorAfter *int) {
	characterID := normalize.ID(rawID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyDamage(&characterState, hpAfter, armorAfter)
	if stressAfter != nil {
		characterState.Stress = *stressAfter
	}
	state.CharacterStates[characterID] = characterState
}

func applyAdversaryDamage(state *daggerheartstate.SnapshotState, rawID ids.AdversaryID, hpAfter, armorAfter *int) {
	adversaryID := normalize.ID(rawID)
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	if hpAfter != nil {
		adversaryState.HP = *hpAfter
	}
	if armorAfter != nil {
		adversaryState.Armor = *armorAfter
	}
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryCreated(state *daggerheartstate.SnapshotState, p payload.AdversaryCreatePayload) {
	adversaryID := normalize.ID(p.AdversaryID)
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.AdversaryEntryID = normalize.String(p.AdversaryEntryID)
	adversaryState.Name = p.Name
	adversaryState.Kind = normalize.String(p.Kind)
	adversaryState.SessionID = normalize.ID(p.SessionID)
	adversaryState.SceneID = normalize.ID(p.SceneID)
	adversaryState.Notes = p.Notes
	adversaryState.HP = p.HP
	adversaryState.HPMax = p.HPMax
	adversaryState.Stress = p.Stress
	adversaryState.StressMax = p.StressMax
	adversaryState.Evasion = p.Evasion
	adversaryState.Major = p.Major
	adversaryState.Severe = p.Severe
	adversaryState.Armor = p.Armor
	adversaryState.FeatureStates = p.FeatureStates
	adversaryState.PendingExperience = p.PendingExperience
	adversaryState.SpotlightGateID = normalize.ID(p.SpotlightGateID)
	adversaryState.SpotlightCount = p.SpotlightCount
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryUpdated(state *daggerheartstate.SnapshotState, p payload.AdversaryUpdatePayload) {
	adversaryID := normalize.ID(p.AdversaryID)
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.AdversaryEntryID = normalize.String(p.AdversaryEntryID)
	adversaryState.Name = p.Name
	adversaryState.Kind = p.Kind
	adversaryState.SessionID = p.SessionID
	adversaryState.SceneID = p.SceneID
	adversaryState.Notes = p.Notes
	adversaryState.HP = p.HP
	adversaryState.HPMax = p.HPMax
	adversaryState.Stress = p.Stress
	adversaryState.StressMax = p.StressMax
	adversaryState.Evasion = p.Evasion
	adversaryState.Major = p.Major
	adversaryState.Severe = p.Severe
	adversaryState.Armor = p.Armor
	adversaryState.FeatureStates = p.FeatureStates
	adversaryState.PendingExperience = p.PendingExperience
	adversaryState.SpotlightGateID = p.SpotlightGateID
	adversaryState.SpotlightCount = p.SpotlightCount
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryConditionsChanged(state *daggerheartstate.SnapshotState, rawID ids.AdversaryID, after []rules.ConditionState) {
	adversaryID := normalize.ID(rawID)
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.Conditions = rules.ConditionCodes(after)
	state.AdversaryStates[adversaryID] = adversaryState
}
