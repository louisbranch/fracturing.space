package daggerheart

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Projector applies Daggerheart system events to state.
type Projector struct{}

// FoldHandledTypes returns the event types this projector's Apply handles.
// Used by ValidateSystemFoldCoverage to verify every emittable event type
// has a fold handler.
func (Projector) FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeGMFearChanged,
		EventTypeCharacterStatePatched,
		EventTypeConditionChanged,
		EventTypeLoadoutSwapped,
		EventTypeCharacterTemporaryArmorApplied,
		EventTypeRestTaken,
		EventTypeCountdownCreated,
		EventTypeCountdownUpdated,
		EventTypeCountdownDeleted,
		EventTypeDamageApplied,
		EventTypeAdversaryDamageApplied,
		EventTypeDowntimeMoveApplied,
		EventTypeAdversaryConditionChanged,
		EventTypeAdversaryCreated,
		EventTypeAdversaryUpdated,
		EventTypeAdversaryDeleted,
	}
}

// Apply applies a Daggerheart event to state.
func (Projector) Apply(state any, evt event.Event) (any, error) {
	var fearPayload GMFearChangedPayload
	current, ok := snapshotFromState(state)
	if !ok && state != nil {
		return state, fmt.Errorf("unsupported state type %T", state)
	}
	if current.CampaignID == "" {
		current.CampaignID = evt.CampaignID
	}
	switch evt.Type {
	case EventTypeGMFearChanged:
		if err := json.Unmarshal(evt.PayloadJSON, &fearPayload); err != nil {
			return state, fmt.Errorf("decode gm_fear_changed payload: %w", err)
		}
		if fearPayload.After < GMFearMin || fearPayload.After > GMFearMax {
			return state, fmt.Errorf("gm fear after must be in range %d..%d", GMFearMin, GMFearMax)
		}
		current.GMFear = fearPayload.After
	case EventTypeCharacterStatePatched:
		var payload CharacterStatePatchedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode character_state_patched payload: %w", err)
		}
		applyCharacterStatePatched(&current, payload)
	case EventTypeConditionChanged:
		var payload ConditionChangedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode condition_changed payload: %w", err)
		}
		applyCharacterConditionsChanged(&current, payload)
	case EventTypeLoadoutSwapped:
		var payload LoadoutSwappedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode loadout_swapped payload: %w", err)
		}
		applyCharacterLoadoutSwapped(&current, payload)
	case EventTypeCharacterTemporaryArmorApplied:
		var payload CharacterTemporaryArmorAppliedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode character_temporary_armor_applied payload: %w", err)
		}
		applyCharacterTemporaryArmorApplied(&current, payload)
	case EventTypeRestTaken:
		var payload RestTakenPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode rest_taken payload: %w", err)
		}
		current.GMFear = payload.GMFearAfter
		if current.GMFear < GMFearMin || current.GMFear > GMFearMax {
			return state, fmt.Errorf("rest_taken gm_fear_after must be in range %d..%d", GMFearMin, GMFearMax)
		}
		for _, patch := range payload.CharacterStates {
			applyRestCharacterPatch(&current, patch)
			if payload.RefreshRest || payload.RefreshLongRest {
				clearRestTemporaryArmor(&current, patch.CharacterID, payload.RefreshRest, payload.RefreshLongRest)
			}
		}
	case EventTypeCountdownCreated:
		var payload CountdownCreatedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode countdown_created payload: %w", err)
		}
		applyCountdownUpsert(&current, payload.CountdownID, func(state *CountdownState) {
			state.Name = payload.Name
			state.Kind = payload.Kind
			state.Current = payload.Current
			state.Max = payload.Max
			state.Direction = payload.Direction
			state.Looping = payload.Looping
		})
	case EventTypeCountdownUpdated:
		var payload CountdownUpdatedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode countdown_updated payload: %w", err)
		}
		applyCountdownUpsert(&current, payload.CountdownID, func(state *CountdownState) {
			state.Current = payload.After
			if payload.Looped {
				state.Looping = true
			}
		})
	case EventTypeCountdownDeleted:
		var payload CountdownDeletedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode countdown_deleted payload: %w", err)
		}
		deleteCountdownState(&current, payload.CountdownID)
	case EventTypeDamageApplied:
		var payload DamageAppliedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode damage_applied payload: %w", err)
		}
		applyDamageApplied(&current, payload.CharacterID, payload.HpAfter, payload.ArmorAfter)
	case EventTypeAdversaryDamageApplied:
		var payload AdversaryDamageAppliedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode adversary_damage_applied payload: %w", err)
		}
		applyAdversaryDamage(&current, payload.AdversaryID, payload.HpAfter, payload.ArmorAfter)
	case EventTypeDowntimeMoveApplied:
		var payload DowntimeMoveAppliedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode downtime_move_applied payload: %w", err)
		}
		applyDowntimeMove(&current, payload.CharacterID, payload.Move, payload.HopeAfter, payload.StressAfter, payload.ArmorAfter)
	case EventTypeAdversaryConditionChanged:
		var payload AdversaryConditionChangedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode adversary_condition_changed payload: %w", err)
		}
		applyAdversaryConditionsChanged(&current, payload.AdversaryID, payload.ConditionsAfter)
	case EventTypeAdversaryCreated:
		var payload AdversaryCreatedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode adversary_created payload: %w", err)
		}
		applyAdversaryCreated(&current, payload)
	case EventTypeAdversaryUpdated:
		var payload AdversaryUpdatedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode adversary_updated payload: %w", err)
		}
		applyAdversaryUpdated(&current, payload)
	case EventTypeAdversaryDeleted:
		var payload AdversaryDeletedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("decode adversary_deleted payload: %w", err)
		}
		delete(current.AdversaryStates, strings.TrimSpace(payload.AdversaryID))
	default:
		return nil, fmt.Errorf("unhandled daggerheart projector event type: %s", evt.Type)
	}
	return current, nil
}

func applyCharacterStatePatched(state *SnapshotState, payload CharacterStatePatchedPayload) {
	characterID := strings.TrimSpace(payload.CharacterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID
	characterState.CharacterID = characterID
	if payload.HPAfter != nil {
		characterState.HP = *payload.HPAfter
	}
	if payload.HopeAfter != nil {
		characterState.Hope = *payload.HopeAfter
	}
	if payload.HopeMaxAfter != nil {
		characterState.HopeMax = *payload.HopeMaxAfter
	}
	if payload.StressAfter != nil {
		characterState.Stress = *payload.StressAfter
	}
	if payload.ArmorAfter != nil {
		characterState.Armor = *payload.ArmorAfter
	}
	if payload.LifeStateAfter != nil {
		characterState.LifeState = *payload.LifeStateAfter
	}
	if state.CharacterStates == nil {
		state.CharacterStates = make(map[string]CharacterState)
	}
	state.CharacterStates[characterID] = characterState
}

func applyCharacterConditionsChanged(state *SnapshotState, payload ConditionChangedPayload) {
	characterID := strings.TrimSpace(payload.CharacterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID
	characterState.CharacterID = characterID
	characterState.Conditions = append([]string(nil), payload.ConditionsAfter...)
	if state.CharacterStates == nil {
		state.CharacterStates = make(map[string]CharacterState)
	}
	state.CharacterStates[characterID] = characterState
}

func applyCharacterLoadoutSwapped(state *SnapshotState, payload LoadoutSwappedPayload) {
	characterID := strings.TrimSpace(payload.CharacterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID
	characterState.CharacterID = characterID
	if payload.StressAfter != nil {
		characterState.Stress = *payload.StressAfter
	}
	if state.CharacterStates == nil {
		state.CharacterStates = make(map[string]CharacterState)
	}
	state.CharacterStates[characterID] = characterState
}

func applyCharacterTemporaryArmorApplied(state *SnapshotState, payload CharacterTemporaryArmorAppliedPayload) {
	characterID := strings.TrimSpace(payload.CharacterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID
	characterState.CharacterID = characterID
	characterState.ApplyTemporaryArmor(TemporaryArmorBucket{
		Source:   strings.TrimSpace(payload.Source),
		Duration: strings.TrimSpace(payload.Duration),
		SourceID: strings.TrimSpace(payload.SourceID),
		Amount:   payload.Amount,
	})
	if state.CharacterStates == nil {
		state.CharacterStates = make(map[string]CharacterState)
	}
	state.CharacterStates[characterID] = characterState
}

func applyRestCharacterPatch(state *SnapshotState, payload RestCharacterStatePatch) {
	characterID := strings.TrimSpace(payload.CharacterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID
	characterState.CharacterID = characterID
	if payload.HopeAfter != nil {
		characterState.Hope = *payload.HopeAfter
	}
	if payload.StressAfter != nil {
		characterState.Stress = *payload.StressAfter
	}
	if payload.ArmorAfter != nil {
		characterState.Armor = *payload.ArmorAfter
	}
	if state.CharacterStates == nil {
		state.CharacterStates = make(map[string]CharacterState)
	}
	state.CharacterStates[characterID] = characterState
}

func clearRestTemporaryArmor(state *SnapshotState, characterID string, clearShortRest bool, clearLongRest bool) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID
	characterState.CharacterID = characterID
	if clearShortRest {
		characterState.ClearTemporaryArmorByDuration("short_rest")
	}
	if clearLongRest {
		characterState.ClearTemporaryArmorByDuration("long_rest")
	}
	characterState.SetArmor(characterState.ResourceCap(ResourceArmor))
	if state.CharacterStates == nil {
		state.CharacterStates = make(map[string]CharacterState)
	}
	state.CharacterStates[characterID] = characterState
}

func applyCountdownUpsert(state *SnapshotState, countdownID string, mutate func(*CountdownState)) {
	countdownID = strings.TrimSpace(countdownID)
	if countdownID == "" {
		return
	}
	countdownState := state.CountdownStates[countdownID]
	countdownState.CampaignID = state.CampaignID
	countdownState.CountdownID = countdownID
	if mutate != nil {
		mutate(&countdownState)
	}
	if state.CountdownStates == nil {
		state.CountdownStates = make(map[string]CountdownState)
	}
	state.CountdownStates[countdownID] = countdownState
}

func deleteCountdownState(state *SnapshotState, countdownID string) {
	countdownID = strings.TrimSpace(countdownID)
	if countdownID == "" {
		return
	}
	delete(state.CountdownStates, countdownID)
}

func applyDamageApplied(state *SnapshotState, characterID string, hpAfter, armorAfter *int) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID
	characterState.CharacterID = characterID
	if hpAfter != nil {
		characterState.HP = *hpAfter
	}
	if armorAfter != nil {
		characterState.Armor = *armorAfter
	}
	if state.CharacterStates == nil {
		state.CharacterStates = make(map[string]CharacterState)
	}
	state.CharacterStates[characterID] = characterState
}

func applyDowntimeMove(state *SnapshotState, characterID string, move string, hopeAfter, stressAfter, armorAfter *int) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID
	characterState.CharacterID = characterID
	if hopeAfter != nil {
		characterState.Hope = *hopeAfter
	}
	if stressAfter != nil {
		characterState.Stress = *stressAfter
	}
	if move == "repair_all_armor" {
		characterState.ClearTemporaryArmorByDuration("short_rest")
		characterState.SetArmor(characterState.Armor)
	}
	if armorAfter != nil {
		characterState.Armor = *armorAfter
		if move == "repair_all_armor" {
			characterState.SetArmor(*armorAfter)
		}
	}
	if state.CharacterStates == nil {
		state.CharacterStates = make(map[string]CharacterState)
	}
	state.CharacterStates[characterID] = characterState
}

func applyAdversaryDamage(state *SnapshotState, adversaryID string, hpAfter, armorAfter *int) {
	adversaryID = strings.TrimSpace(adversaryID)
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
	if state.AdversaryStates == nil {
		state.AdversaryStates = make(map[string]AdversaryState)
	}
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryCreated(state *SnapshotState, payload AdversaryCreatePayload) {
	adversaryID := strings.TrimSpace(payload.AdversaryID)
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.Name = payload.Name
	adversaryState.Kind = strings.TrimSpace(payload.Kind)
	adversaryState.SessionID = strings.TrimSpace(payload.SessionID)
	adversaryState.Notes = payload.Notes
	adversaryState.HP = payload.HP
	adversaryState.HPMax = payload.HPMax
	adversaryState.Stress = payload.Stress
	adversaryState.StressMax = payload.StressMax
	adversaryState.Evasion = payload.Evasion
	adversaryState.Major = payload.Major
	adversaryState.Severe = payload.Severe
	adversaryState.Armor = payload.Armor
	if state.AdversaryStates == nil {
		state.AdversaryStates = make(map[string]AdversaryState)
	}
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryUpdated(state *SnapshotState, payload AdversaryUpdatePayload) {
	adversaryID := strings.TrimSpace(payload.AdversaryID)
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.Name = payload.Name
	adversaryState.Kind = payload.Kind
	adversaryState.SessionID = payload.SessionID
	adversaryState.Notes = payload.Notes
	adversaryState.HP = payload.HP
	adversaryState.HPMax = payload.HPMax
	adversaryState.Stress = payload.Stress
	adversaryState.StressMax = payload.StressMax
	adversaryState.Evasion = payload.Evasion
	adversaryState.Major = payload.Major
	adversaryState.Severe = payload.Severe
	adversaryState.Armor = payload.Armor
	if state.AdversaryStates == nil {
		state.AdversaryStates = make(map[string]AdversaryState)
	}
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryConditionsChanged(state *SnapshotState, adversaryID string, after []string) {
	adversaryID = strings.TrimSpace(adversaryID)
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.Conditions = append([]string(nil), after...)
	if state.AdversaryStates == nil {
		state.AdversaryStates = make(map[string]AdversaryState)
	}
	state.AdversaryStates[adversaryID] = adversaryState
}

// snapshotFromState extracts a SnapshotState from the state value.
//
// The aggregate applier already extracts the system-specific state before
// calling RouteEvent/RouteCommand, so only SnapshotState and *SnapshotState
// are expected in production. Any other type returns (zero, false).
func snapshotFromState(state any) (SnapshotState, bool) {
	switch typed := state.(type) {
	case SnapshotState:
		return typed, true
	case *SnapshotState:
		if typed != nil {
			return *typed, true
		}
	}
	return SnapshotState{GMFear: GMFearDefault}, false
}
