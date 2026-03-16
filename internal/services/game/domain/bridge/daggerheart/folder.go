package daggerheart

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/reducer"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
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
		s.CampaignID = ids.CampaignID(evt.CampaignID)
	}
	return f.router.Fold(s, evt)
}

// registerFoldHandlers registers all Daggerheart fold handlers on the router.
func registerFoldHandlers(r *module.FoldRouter[*SnapshotState]) {
	module.HandleFold(r, EventTypeGMFearChanged, foldGMFearChanged)
	module.HandleFold(r, EventTypeCharacterProfileReplaced, foldCharacterProfileReplaced)
	module.HandleFold(r, EventTypeCharacterProfileDeleted, foldCharacterProfileDeleted)
	module.HandleFold(r, EventTypeCharacterStatePatched, foldCharacterStatePatched)
	module.HandleFold(r, EventTypeBeastformTransformed, foldBeastformTransformed)
	module.HandleFold(r, EventTypeBeastformDropped, foldBeastformDropped)
	module.HandleFold(r, EventTypeCompanionExperienceBegun, foldCompanionExperienceBegun)
	module.HandleFold(r, EventTypeCompanionReturned, foldCompanionReturned)
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
	module.HandleFold(r, EventTypeEnvironmentEntityCreated, foldEnvironmentEntityCreated)
	module.HandleFold(r, EventTypeEnvironmentEntityUpdated, foldEnvironmentEntityUpdated)
	module.HandleFold(r, EventTypeEnvironmentEntityDeleted, foldEnvironmentEntityDeleted)
	module.HandleFold(r, EventTypeLevelUpApplied, foldLevelUpApplied)
	module.HandleFold(r, EventTypeGoldUpdated, foldGoldUpdated)
	module.HandleFold(r, EventTypeDomainCardAcquired, foldDomainCardAcquired)
	module.HandleFold(r, EventTypeEquipmentSwapped, foldEquipmentSwapped)
	module.HandleFold(r, EventTypeConsumableUsed, foldConsumableUsed)
	module.HandleFold(r, EventTypeConsumableAcquired, foldConsumableAcquired)
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
	if payload.Value < GMFearMin || payload.Value > GMFearMax {
		return fmt.Errorf("gm fear value must be in range %d..%d", GMFearMin, GMFearMax)
	}
	state.GMFear = payload.Value
	return nil
}

func foldCharacterProfileReplaced(state *SnapshotState, payload CharacterProfileReplacedPayload) error {
	characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
	if characterID == "" {
		return nil
	}
	profile := payload.Profile.Normalized()
	state.CharacterProfiles[characterID] = profile
	if _, exists := state.CharacterStates[characterID]; !exists {
		state.CharacterStates[characterID] = CharacterState{
			CampaignID:  strings.TrimSpace(string(state.CampaignID)),
			CharacterID: strings.TrimSpace(string(characterID)),
			HP:          profile.HpMax,
			Hope:        HopeDefault,
			HopeMax:     HopeMaxDefault,
			Stress:      StressDefault,
			Armor:       profile.ArmorMax,
			LifeState:   LifeStateAlive,
		}
	}
	if profile.CompanionSheet != nil {
		state.CharacterCompanions[characterID] = CharacterCompanionState{Status: CompanionStatusPresent}
	} else {
		delete(state.CharacterCompanions, characterID)
	}
	return nil
}

func foldCharacterProfileDeleted(state *SnapshotState, payload CharacterProfileDeletedPayload) error {
	characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
	if characterID == "" {
		return nil
	}
	delete(state.CharacterProfiles, characterID)
	delete(state.CharacterCompanions, characterID)
	return nil
}

func foldCharacterStatePatched(state *SnapshotState, payload CharacterStatePatchedPayload) error {
	applyCharacterStatePatched(state, payload)
	return nil
}

func foldBeastformTransformed(state *SnapshotState, payload BeastformTransformedPayload) error {
	characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
	if characterID == "" {
		return nil
	}
	nextClassState := CharacterClassState{}
	if current, ok := state.CharacterClassStates[characterID]; ok {
		nextClassState = current
	}
	nextClassState = WithActiveBeastform(nextClassState, payload.ActiveBeastform)
	applyStatePatch(state, payload.CharacterID, nil, payload.Hope, nil, payload.Stress, nil, nil, &nextClassState, nil, nil)
	return nil
}

func foldBeastformDropped(state *SnapshotState, payload BeastformDroppedPayload) error {
	characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
	if characterID == "" {
		return nil
	}
	nextClassState := CharacterClassState{}
	if current, ok := state.CharacterClassStates[characterID]; ok {
		nextClassState = current
	}
	nextClassState = WithActiveBeastform(nextClassState, nil)
	applyStatePatch(state, payload.CharacterID, nil, nil, nil, nil, nil, nil, &nextClassState, nil, nil)
	return nil
}

func foldCompanionExperienceBegun(state *SnapshotState, payload CompanionExperienceBegunPayload) error {
	applyStatePatch(state, payload.CharacterID, nil, nil, nil, nil, nil, nil, nil, nil, payload.CompanionState)
	return nil
}

func foldCompanionReturned(state *SnapshotState, payload CompanionReturnedPayload) error {
	applyStatePatch(state, payload.CharacterID, nil, nil, nil, payload.Stress, nil, nil, nil, nil, payload.CompanionState)
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
	state.GMFear = payload.GMFear
	if state.GMFear < GMFearMin || state.GMFear > GMFearMax {
		return fmt.Errorf("rest_taken gm_fear_after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	for _, participantID := range payload.Participants {
		if payload.RefreshRest || payload.RefreshLongRest {
			clearRestTemporaryArmor(state, participantID.String(), payload.RefreshRest, payload.RefreshLongRest)
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
		cs.Variant = payload.Variant
		cs.TriggerEventType = payload.TriggerEventType
		cs.LinkedCountdownID = payload.LinkedCountdownID
	})
	return nil
}

func foldCountdownUpdated(state *SnapshotState, payload CountdownUpdatedPayload) error {
	applyCountdownUpsert(state, payload.CountdownID, func(cs *CountdownState) {
		cs.Current = payload.Value
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
	applyDamageApplied(state, payload.CharacterID, payload.Hp, payload.Stress, payload.Armor)
	return nil
}

func foldAdversaryDamageApplied(state *SnapshotState, payload AdversaryDamageAppliedPayload) error {
	applyAdversaryDamage(state, payload.AdversaryID, payload.Hp, payload.Armor)
	return nil
}

func foldDowntimeMoveApplied(state *SnapshotState, payload DowntimeMoveAppliedPayload) error {
	targetID := payload.TargetCharacterID
	if strings.TrimSpace(targetID.String()) == "" {
		targetID = payload.ActorCharacterID
	}
	if strings.TrimSpace(targetID.String()) == "" {
		return nil
	}
	applyStatePatch(state, targetID, payload.HP, payload.Hope, nil, payload.Stress, payload.Armor, nil, nil, nil, nil)
	return nil
}

func foldAdversaryConditionChanged(state *SnapshotState, payload AdversaryConditionChangedPayload) error {
	applyAdversaryConditionsChanged(state, payload.AdversaryID, payload.Conditions)
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
	delete(state.AdversaryStates, ids.AdversaryID(strings.TrimSpace(payload.AdversaryID.String())))
	return nil
}

func foldEnvironmentEntityCreated(state *SnapshotState, payload EnvironmentEntityCreatedPayload) error {
	environmentEntityID := ids.EnvironmentEntityID(strings.TrimSpace(payload.EnvironmentEntityID.String()))
	if environmentEntityID == "" {
		return nil
	}
	state.EnvironmentStates[environmentEntityID] = EnvironmentEntityState{
		CampaignID:          state.CampaignID,
		EnvironmentEntityID: environmentEntityID,
		EnvironmentID:       strings.TrimSpace(payload.EnvironmentID),
		Name:                strings.TrimSpace(payload.Name),
		Type:                strings.TrimSpace(payload.Type),
		Tier:                payload.Tier,
		Difficulty:          payload.Difficulty,
		SessionID:           ids.SessionID(strings.TrimSpace(payload.SessionID.String())),
		SceneID:             ids.SceneID(strings.TrimSpace(payload.SceneID.String())),
		Notes:               strings.TrimSpace(payload.Notes),
	}
	return nil
}

func foldEnvironmentEntityUpdated(state *SnapshotState, payload EnvironmentEntityUpdatedPayload) error {
	return foldEnvironmentEntityCreated(state, EnvironmentEntityCreatedPayload(payload))
}

func foldEnvironmentEntityDeleted(state *SnapshotState, payload EnvironmentEntityDeletedPayload) error {
	delete(state.EnvironmentStates, ids.EnvironmentEntityID(strings.TrimSpace(payload.EnvironmentEntityID.String())))
	return nil
}

func foldLevelUpApplied(state *SnapshotState, payload LevelUpAppliedPayload) error {
	touchCharacter(state, payload.CharacterID)
	if profile, ok := state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))]; ok {
		applyLevelUpToCharacterProfile(&profile, payload)
		state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))] = profile
	}
	return nil
}

func foldGoldUpdated(state *SnapshotState, payload GoldUpdatedPayload) error {
	touchCharacter(state, payload.CharacterID)
	if profile, ok := state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))]; ok {
		profile.GoldHandfuls = payload.Handfuls
		profile.GoldBags = payload.Bags
		profile.GoldChests = payload.Chests
		state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))] = profile
	}
	return nil
}

func foldDomainCardAcquired(state *SnapshotState, payload DomainCardAcquiredPayload) error {
	touchCharacter(state, payload.CharacterID)
	if profile, ok := state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))]; ok {
		profile.DomainCardIDs = appendUnique(profile.DomainCardIDs, payload.CardID)
		state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))] = profile
	}
	return nil
}

func foldEquipmentSwapped(state *SnapshotState, payload EquipmentSwappedPayload) error {
	touchCharacter(state, payload.CharacterID)
	if strings.TrimSpace(payload.ItemType) == "armor" {
		characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
		if profile, ok := state.CharacterProfiles[characterID]; ok {
			profile.EquippedArmorID = strings.TrimSpace(payload.EquippedArmorID)
			if payload.EvasionAfter != nil {
				profile.Evasion = *payload.EvasionAfter
			}
			if payload.MajorThresholdAfter != nil {
				profile.MajorThreshold = *payload.MajorThresholdAfter
			}
			if payload.SevereThresholdAfter != nil {
				profile.SevereThreshold = *payload.SevereThresholdAfter
			}
			if payload.ArmorScoreAfter != nil {
				profile.ArmorScore = *payload.ArmorScoreAfter
			}
			if payload.ArmorMaxAfter != nil {
				profile.ArmorMax = *payload.ArmorMaxAfter
			}
			if payload.SpellcastRollBonusAfter != nil {
				profile.SpellcastRollBonus = *payload.SpellcastRollBonusAfter
			}
			if payload.AgilityAfter != nil {
				profile.Agility = *payload.AgilityAfter
			}
			if payload.StrengthAfter != nil {
				profile.Strength = *payload.StrengthAfter
			}
			if payload.FinesseAfter != nil {
				profile.Finesse = *payload.FinesseAfter
			}
			if payload.InstinctAfter != nil {
				profile.Instinct = *payload.InstinctAfter
			}
			if payload.PresenceAfter != nil {
				profile.Presence = *payload.PresenceAfter
			}
			if payload.KnowledgeAfter != nil {
				profile.Knowledge = *payload.KnowledgeAfter
			}
			state.CharacterProfiles[characterID] = profile
		}
		if payload.ArmorAfter != nil {
			applyStatePatch(state, payload.CharacterID, nil, nil, nil, nil, payload.ArmorAfter, nil, nil, nil, nil)
		}
	}
	if payload.StressCost > 0 {
		characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
		characterState := state.CharacterStates[characterID]
		characterState.CampaignID = state.CampaignID.String()
		characterState.CharacterID = characterID.String()
		characterState.Stress += payload.StressCost
		state.CharacterStates[characterID] = characterState
	}
	return nil
}

func foldConsumableUsed(state *SnapshotState, payload ConsumableUsedPayload) error {
	touchCharacter(state, payload.CharacterID)
	return nil
}

func foldConsumableAcquired(state *SnapshotState, payload ConsumableAcquiredPayload) error {
	touchCharacter(state, payload.CharacterID)
	return nil
}

// touchCharacter ensures a CharacterState entry exists for the given character.
func touchCharacter(state *SnapshotState, rawID ids.CharacterID) {
	characterID := ids.CharacterID(strings.TrimSpace(rawID.String()))
	if characterID == "" {
		return
	}
	cs := state.CharacterStates[characterID]
	cs.CampaignID = state.CampaignID.String()
	cs.CharacterID = characterID.String()
	state.CharacterStates[characterID] = cs
}

func applyCharacterStatePatched(state *SnapshotState, payload CharacterStatePatchedPayload) {
	applyStatePatch(state, payload.CharacterID, payload.HP, payload.Hope, payload.HopeMax, payload.Stress, payload.Armor, payload.LifeState, payload.ClassState, payload.SubclassState, nil)
}

func applyStatePatch(state *SnapshotState, characterID ids.CharacterID, hpAfter, hopeAfter, hopeMaxAfter, stressAfter, armorAfter *int, lifeStateAfter *string, classStateAfter *CharacterClassState, subclassStateAfter *CharacterSubclassState, companionStateAfter *CharacterCompanionState) {
	characterID = ids.CharacterID(strings.TrimSpace(characterID.String()))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyCharacterStatePatch(&characterState, reducer.CharacterStatePatch{
		HPAfter:        hpAfter,
		HopeAfter:      hopeAfter,
		HopeMaxAfter:   hopeMaxAfter,
		StressAfter:    stressAfter,
		ArmorAfter:     armorAfter,
		LifeStateAfter: lifeStateAfter,
	})
	state.CharacterStates[characterID] = characterState
	if classStateAfter != nil {
		state.CharacterClassStates[characterID] = classStateAfter.Normalized()
	}
	if subclassStateAfter != nil {
		normalized := subclassStateAfter.Normalized()
		if normalized.IsZero() {
			delete(state.CharacterSubclassStates, characterID)
		} else {
			state.CharacterSubclassStates[characterID] = normalized
		}
	}
	if companionStateAfter != nil {
		normalized := companionStateAfter.Normalized()
		if normalized.IsZero() {
			delete(state.CharacterCompanions, characterID)
		} else {
			state.CharacterCompanions[characterID] = normalized
		}
	}
}

func applyCharacterConditionsChanged(state *SnapshotState, payload ConditionChangedPayload) {
	characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyConditionPatch(&characterState, ConditionCodes(payload.Conditions))
	state.CharacterStates[characterID] = characterState
}

func applyCharacterLoadoutSwapped(state *SnapshotState, payload LoadoutSwappedPayload) {
	characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyLoadoutSwap(&characterState, payload.Stress)
	state.CharacterStates[characterID] = characterState
}

func applyCharacterTemporaryArmorApplied(state *SnapshotState, payload CharacterTemporaryArmorAppliedPayload) {
	characterID := ids.CharacterID(strings.TrimSpace(payload.CharacterID.String()))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyTemporaryArmor(&characterState, reducer.TemporaryArmorPatch{
		Source:   strings.TrimSpace(payload.Source),
		Duration: strings.TrimSpace(payload.Duration),
		SourceID: strings.TrimSpace(payload.SourceID),
		Amount:   payload.Amount,
	})
	state.CharacterStates[characterID] = characterState
}

func clearRestTemporaryArmor(state *SnapshotState, rawID string, clearShortRest bool, clearLongRest bool) {
	characterID := ids.CharacterID(strings.TrimSpace(rawID))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ClearRestTemporaryArmor(&characterState, clearShortRest, clearLongRest)
	state.CharacterStates[characterID] = characterState
}

func applyCountdownUpsert(state *SnapshotState, countdownID ids.CountdownID, mutate func(*CountdownState)) {
	trimmed := ids.CountdownID(strings.TrimSpace(countdownID.String()))
	if trimmed == "" {
		return
	}
	countdownState := state.CountdownStates[trimmed]
	countdownState.CampaignID = state.CampaignID
	countdownState.CountdownID = trimmed
	if mutate != nil {
		mutate(&countdownState)
	}
	state.CountdownStates[trimmed] = countdownState
}

func deleteCountdownState(state *SnapshotState, countdownID ids.CountdownID) {
	trimmed := ids.CountdownID(strings.TrimSpace(countdownID.String()))
	if trimmed == "" {
		return
	}
	delete(state.CountdownStates, trimmed)
}

func applyDamageApplied(state *SnapshotState, rawID ids.CharacterID, hpAfter, stressAfter, armorAfter *int) {
	characterID := ids.CharacterID(strings.TrimSpace(rawID.String()))
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

func applyAdversaryDamage(state *SnapshotState, rawID ids.AdversaryID, hpAfter, armorAfter *int) {
	adversaryID := ids.AdversaryID(strings.TrimSpace(rawID.String()))
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
	adversaryID := ids.AdversaryID(strings.TrimSpace(payload.AdversaryID.String()))
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.AdversaryEntryID = strings.TrimSpace(payload.AdversaryEntryID)
	adversaryState.Name = payload.Name
	adversaryState.Kind = strings.TrimSpace(payload.Kind)
	adversaryState.SessionID = ids.SessionID(strings.TrimSpace(payload.SessionID.String()))
	adversaryState.SceneID = ids.SceneID(strings.TrimSpace(payload.SceneID.String()))
	adversaryState.Notes = payload.Notes
	adversaryState.HP = payload.HP
	adversaryState.HPMax = payload.HPMax
	adversaryState.Stress = payload.Stress
	adversaryState.StressMax = payload.StressMax
	adversaryState.Evasion = payload.Evasion
	adversaryState.Major = payload.Major
	adversaryState.Severe = payload.Severe
	adversaryState.Armor = payload.Armor
	adversaryState.FeatureStates = payload.FeatureStates
	adversaryState.PendingExperience = payload.PendingExperience
	adversaryState.SpotlightGateID = ids.GateID(strings.TrimSpace(payload.SpotlightGateID.String()))
	adversaryState.SpotlightCount = payload.SpotlightCount
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryUpdated(state *SnapshotState, payload AdversaryUpdatePayload) {
	adversaryID := ids.AdversaryID(strings.TrimSpace(payload.AdversaryID.String()))
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.AdversaryEntryID = strings.TrimSpace(payload.AdversaryEntryID)
	adversaryState.Name = payload.Name
	adversaryState.Kind = payload.Kind
	adversaryState.SessionID = payload.SessionID
	adversaryState.SceneID = payload.SceneID
	adversaryState.Notes = payload.Notes
	adversaryState.HP = payload.HP
	adversaryState.HPMax = payload.HPMax
	adversaryState.Stress = payload.Stress
	adversaryState.StressMax = payload.StressMax
	adversaryState.Evasion = payload.Evasion
	adversaryState.Major = payload.Major
	adversaryState.Severe = payload.Severe
	adversaryState.Armor = payload.Armor
	adversaryState.FeatureStates = payload.FeatureStates
	adversaryState.PendingExperience = payload.PendingExperience
	adversaryState.SpotlightGateID = payload.SpotlightGateID
	adversaryState.SpotlightCount = payload.SpotlightCount
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryConditionsChanged(state *SnapshotState, rawID ids.AdversaryID, after []ConditionState) {
	adversaryID := ids.AdversaryID(strings.TrimSpace(rawID.String()))
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.Conditions = ConditionCodes(after)
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
