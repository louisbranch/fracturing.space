package daggerheart

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// Folder folds Daggerheart system events into snapshot state.
type Folder struct {
	router *module.FoldRouter[*SnapshotState]
}

// NewFolder creates a Folder with all fold handlers registered.
func NewFolder() *Folder {
	router := module.NewFoldRouter(assertSnapshotState)
	registerFoldHandlers(router)
	return &Folder{router: router}
}

// FoldHandledTypes returns the event types this folder's Fold handles.
// Delegates to the router so the list reflects actual HandleFold registrations
// rather than event definitions. If a developer adds an event definition but
// forgets HandleFold, startup validation catches it immediately.
func (f *Folder) FoldHandledTypes() []event.Type {
	return f.router.FoldHandledTypes()
}

// Fold folds a Daggerheart event into system state. It delegates to the
// FoldRouter after ensuring the snapshot CampaignID is populated from the
// event envelope — required because the first fold may receive nil state.
func (f *Folder) Fold(state any, evt event.Event) (any, error) {
	// Pre-assert and set CampaignID before router dispatch so individual
	// fold handlers don't need to repeat this.
	s, err := assertSnapshotState(state)
	if err != nil {
		return nil, err
	}
	if s.CampaignID == "" {
		s.CampaignID = evt.CampaignID
	}
	return f.router.Fold(s, evt)
}

// registerFoldHandlers registers all Daggerheart fold handlers on the router.
func registerFoldHandlers(r *module.FoldRouter[*SnapshotState]) {
	module.HandleFold(r, EventTypeGMFearChanged, foldGMFearChanged)
	module.HandleFold(r, EventTypeCharacterStatePatched, foldCharacterStatePatched)
	module.HandleFold(r, EventTypeConditionChanged, foldConditionChanged)
	module.HandleFold(r, EventTypeLoadoutSwapped, foldLoadoutSwapped)
	module.HandleFold(r, EventTypeCharacterTemporaryArmorApplied, foldCharacterTemporaryArmorApplied)
	module.HandleFold(r, EventTypeRestTaken, foldRestTaken)
	module.HandleFold(r, EventTypeCountdownCreated, foldCountdownCreated)
	module.HandleFold(r, EventTypeCountdownUpdated, foldCountdownUpdated)
	module.HandleFold(r, EventTypeCountdownDeleted, foldCountdownDeleted)
	module.HandleFold(r, EventTypeDamageApplied, foldDamageApplied)
	module.HandleFold(r, EventTypeAdversaryDamageApplied, foldAdversaryDamageApplied)
	module.HandleFold(r, EventTypeDowntimeMoveApplied, foldDowntimeMoveApplied)
	module.HandleFold(r, EventTypeAdversaryConditionChanged, foldAdversaryConditionChanged)
	module.HandleFold(r, EventTypeAdversaryCreated, foldAdversaryCreated)
	module.HandleFold(r, EventTypeAdversaryUpdated, foldAdversaryUpdated)
	module.HandleFold(r, EventTypeAdversaryDeleted, foldAdversaryDeleted)
}

// assertSnapshotState converts untyped state to *SnapshotState for the fold
// router. It handles nil (first event), value types, and pointer types.
// EnsureMaps is called on the result so deserialized states with nil maps
// are safe to use immediately.
func assertSnapshotState(state any) (*SnapshotState, error) {
	var s *SnapshotState
	switch typed := state.(type) {
	case nil:
		v := SnapshotState{GMFear: GMFearDefault}
		s = &v
	case SnapshotState:
		s = &typed
	case *SnapshotState:
		if typed != nil {
			s = typed
		} else {
			v := SnapshotState{GMFear: GMFearDefault}
			s = &v
		}
	default:
		return nil, fmt.Errorf("unsupported state type %T", state)
	}
	s.EnsureMaps()
	return s, nil
}

func foldGMFearChanged(state *SnapshotState, payload GMFearChangedPayload) error {
	if payload.After < GMFearMin || payload.After > GMFearMax {
		return fmt.Errorf("gm fear after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	state.GMFear = payload.After
	return nil
}

func foldCharacterStatePatched(state *SnapshotState, payload CharacterStatePatchedPayload) error {
	applyCharacterStatePatched(state, payload)
	return nil
}

func foldConditionChanged(state *SnapshotState, payload ConditionChangedPayload) error {
	applyCharacterConditionsChanged(state, payload)
	return nil
}

func foldLoadoutSwapped(state *SnapshotState, payload LoadoutSwappedPayload) error {
	applyCharacterLoadoutSwapped(state, payload)
	return nil
}

func foldCharacterTemporaryArmorApplied(state *SnapshotState, payload CharacterTemporaryArmorAppliedPayload) error {
	applyCharacterTemporaryArmorApplied(state, payload)
	return nil
}

func foldRestTaken(state *SnapshotState, payload RestTakenPayload) error {
	state.GMFear = payload.GMFearAfter
	if state.GMFear < GMFearMin || state.GMFear > GMFearMax {
		return fmt.Errorf("rest_taken gm_fear_after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	for _, patch := range payload.CharacterStates {
		applyRestCharacterPatch(state, patch)
		if payload.RefreshRest || payload.RefreshLongRest {
			clearRestTemporaryArmor(state, patch.CharacterID, payload.RefreshRest, payload.RefreshLongRest)
		}
	}
	return nil
}

func foldCountdownCreated(state *SnapshotState, payload CountdownCreatedPayload) error {
	applyCountdownUpsert(state, payload.CountdownID, func(cs *CountdownState) {
		cs.Name = payload.Name
		cs.Kind = payload.Kind
		cs.Current = payload.Current
		cs.Max = payload.Max
		cs.Direction = payload.Direction
		cs.Looping = payload.Looping
	})
	return nil
}

func foldCountdownUpdated(state *SnapshotState, payload CountdownUpdatedPayload) error {
	applyCountdownUpsert(state, payload.CountdownID, func(cs *CountdownState) {
		cs.Current = payload.After
		if payload.Looped {
			cs.Looping = true
		}
	})
	return nil
}

func foldCountdownDeleted(state *SnapshotState, payload CountdownDeletedPayload) error {
	deleteCountdownState(state, payload.CountdownID)
	return nil
}

func foldDamageApplied(state *SnapshotState, payload DamageAppliedPayload) error {
	applyDamageApplied(state, payload.CharacterID, payload.HpAfter, payload.ArmorAfter)
	return nil
}

func foldAdversaryDamageApplied(state *SnapshotState, payload AdversaryDamageAppliedPayload) error {
	applyAdversaryDamage(state, payload.AdversaryID, payload.HpAfter, payload.ArmorAfter)
	return nil
}

func foldDowntimeMoveApplied(state *SnapshotState, payload DowntimeMoveAppliedPayload) error {
	applyDowntimeMove(state, payload.CharacterID, payload.Move, payload.HopeAfter, payload.StressAfter, payload.ArmorAfter)
	return nil
}

func foldAdversaryConditionChanged(state *SnapshotState, payload AdversaryConditionChangedPayload) error {
	applyAdversaryConditionsChanged(state, payload.AdversaryID, payload.ConditionsAfter)
	return nil
}

func foldAdversaryCreated(state *SnapshotState, payload AdversaryCreatedPayload) error {
	applyAdversaryCreated(state, payload)
	return nil
}

func foldAdversaryUpdated(state *SnapshotState, payload AdversaryUpdatedPayload) error {
	applyAdversaryUpdated(state, payload)
	return nil
}

func foldAdversaryDeleted(state *SnapshotState, payload AdversaryDeletedPayload) error {
	delete(state.AdversaryStates, strings.TrimSpace(payload.AdversaryID))
	return nil
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
	state.AdversaryStates[adversaryID] = adversaryState
}

// snapshotOrDefault extracts a SnapshotState from the state value for the
// decider path. Returns (state, true) for known types, or a factory-aligned
// default with initialized maps on nil/unknown. Using the same defaults as
// NewSnapshotState ensures decider defaults never silently diverge from the
// factory.
func snapshotOrDefault(state any) (SnapshotState, bool) {
	switch typed := state.(type) {
	case SnapshotState:
		typed.EnsureMaps()
		return typed, true
	case *SnapshotState:
		if typed != nil {
			typed.EnsureMaps()
			return *typed, true
		}
	}
	s := SnapshotState{GMFear: GMFearDefault}
	s.EnsureMaps()
	return s, false
}
