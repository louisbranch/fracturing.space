package folder

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/reducer"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

// LevelUpApplier applies level-up progression to a character profile.
type LevelUpApplier func(*snapstate.CharacterProfile, payload.LevelUpAppliedPayload)

// Folder folds Daggerheart system events into snapshot state.
type Folder struct {
	// Router is exported so root-package tests can verify registration
	// consistency via the type alias.
	Router       *module.FoldRouter[*snapstate.SnapshotState]
	applyLevelUp LevelUpApplier
}

// NewFolder creates a Folder with all fold handlers registered.
func NewFolder(applyLevelUp LevelUpApplier) *Folder {
	f := &Folder{applyLevelUp: applyLevelUp}
	router := module.NewFoldRouter(snapstate.AssertSnapshotState)
	f.registerFoldHandlers(router)
	f.Router = router
	return f
}

// FoldHandledTypes returns the event types this folder's Fold handles.
func (f *Folder) FoldHandledTypes() []event.Type {
	return f.Router.FoldHandledTypes()
}

// Fold folds a Daggerheart event into system state. It delegates to the
// FoldRouter after ensuring the snapshot CampaignID is populated from the
// event envelope — required because the first fold may receive nil state.
func (f *Folder) Fold(state any, evt event.Event) (any, error) {
	s, err := snapstate.AssertSnapshotState(state)
	if err != nil {
		return nil, err
	}
	if s.CampaignID == "" {
		s.CampaignID = ids.CampaignID(evt.CampaignID)
	}
	return f.Router.Fold(s, evt)
}

// registerFoldHandlers registers all Daggerheart fold handlers on the router.
func (f *Folder) registerFoldHandlers(r *module.FoldRouter[*snapstate.SnapshotState]) {
	module.HandleFold(r, payload.EventTypeGMFearChanged, f.foldGMFearChanged)
	module.HandleFold(r, payload.EventTypeCharacterProfileReplaced, f.foldCharacterProfileReplaced)
	module.HandleFold(r, payload.EventTypeCharacterProfileDeleted, f.foldCharacterProfileDeleted)
	module.HandleFold(r, payload.EventTypeCharacterStatePatched, f.foldCharacterStatePatched)
	module.HandleFold(r, payload.EventTypeBeastformTransformed, f.foldBeastformTransformed)
	module.HandleFold(r, payload.EventTypeBeastformDropped, f.foldBeastformDropped)
	module.HandleFold(r, payload.EventTypeCompanionExperienceBegun, f.foldCompanionExperienceBegun)
	module.HandleFold(r, payload.EventTypeCompanionReturned, f.foldCompanionReturned)
	module.HandleFold(r, payload.EventTypeConditionChanged, f.foldConditionChanged)
	module.HandleFold(r, payload.EventTypeLoadoutSwapped, f.foldLoadoutSwapped)
	module.HandleFold(r, payload.EventTypeCharacterTemporaryArmorApplied, f.foldCharacterTemporaryArmorApplied)
	module.HandleFold(r, payload.EventTypeRestTaken, f.foldRestTaken)
	module.HandleFold(r, payload.EventTypeCountdownCreated, f.foldCountdownCreated)
	module.HandleFold(r, payload.EventTypeCountdownUpdated, f.foldCountdownUpdated)
	module.HandleFold(r, payload.EventTypeCountdownDeleted, f.foldCountdownDeleted)
	module.HandleFold(r, payload.EventTypeDamageApplied, f.foldDamageApplied)
	module.HandleFold(r, payload.EventTypeAdversaryDamageApplied, f.foldAdversaryDamageApplied)
	module.HandleFold(r, payload.EventTypeDowntimeMoveApplied, f.foldDowntimeMoveApplied)
	module.HandleFold(r, payload.EventTypeAdversaryConditionChanged, f.foldAdversaryConditionChanged)
	module.HandleFold(r, payload.EventTypeAdversaryCreated, f.foldAdversaryCreated)
	module.HandleFold(r, payload.EventTypeAdversaryUpdated, f.foldAdversaryUpdated)
	module.HandleFold(r, payload.EventTypeAdversaryDeleted, f.foldAdversaryDeleted)
	module.HandleFold(r, payload.EventTypeEnvironmentEntityCreated, f.foldEnvironmentEntityCreated)
	module.HandleFold(r, payload.EventTypeEnvironmentEntityUpdated, f.foldEnvironmentEntityUpdated)
	module.HandleFold(r, payload.EventTypeEnvironmentEntityDeleted, f.foldEnvironmentEntityDeleted)
	module.HandleFold(r, payload.EventTypeLevelUpApplied, f.foldLevelUpApplied)
	module.HandleFold(r, payload.EventTypeGoldUpdated, f.foldGoldUpdated)
	module.HandleFold(r, payload.EventTypeDomainCardAcquired, f.foldDomainCardAcquired)
	module.HandleFold(r, payload.EventTypeEquipmentSwapped, f.foldEquipmentSwapped)
	module.HandleFold(r, payload.EventTypeConsumableUsed, f.foldConsumableUsed)
	module.HandleFold(r, payload.EventTypeConsumableAcquired, f.foldConsumableAcquired)
}

func (f *Folder) foldGMFearChanged(state *snapstate.SnapshotState, p payload.GMFearChangedPayload) error {
	if p.Value < snapstate.GMFearMin || p.Value > snapstate.GMFearMax {
		return fmt.Errorf("gm fear value must be in range %d..%d", snapstate.GMFearMin, snapstate.GMFearMax)
	}
	state.GMFear = p.Value
	return nil
}

func (f *Folder) foldCharacterProfileReplaced(state *snapstate.SnapshotState, p snapstate.CharacterProfileReplacedPayload) error {
	characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
	if characterID == "" {
		return nil
	}
	profile := p.Profile.Normalized()
	state.CharacterProfiles[characterID] = profile
	if _, exists := state.CharacterStates[characterID]; !exists {
		state.CharacterStates[characterID] = snapstate.CharacterState{
			CampaignID:  strings.TrimSpace(string(state.CampaignID)),
			CharacterID: strings.TrimSpace(string(characterID)),
			HP:          profile.HpMax,
			Hope:        snapstate.HopeDefault,
			HopeMax:     snapstate.HopeMaxDefault,
			Stress:      snapstate.StressDefault,
			Armor:       profile.ArmorMax,
			LifeState:   snapstate.LifeStateAlive,
		}
	}
	if profile.CompanionSheet != nil {
		state.CharacterCompanions[characterID] = snapstate.CharacterCompanionState{Status: snapstate.CompanionStatusPresent}
	} else {
		delete(state.CharacterCompanions, characterID)
	}
	return nil
}

func (f *Folder) foldCharacterProfileDeleted(state *snapstate.SnapshotState, p snapstate.CharacterProfileDeletedPayload) error {
	characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
	if characterID == "" {
		return nil
	}
	delete(state.CharacterProfiles, characterID)
	delete(state.CharacterCompanions, characterID)
	return nil
}

func (f *Folder) foldCharacterStatePatched(state *snapstate.SnapshotState, p payload.CharacterStatePatchedPayload) error {
	applyCharacterStatePatched(state, p)
	return nil
}

func (f *Folder) foldBeastformTransformed(state *snapstate.SnapshotState, p payload.BeastformTransformedPayload) error {
	characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
	if characterID == "" {
		return nil
	}
	nextClassState := snapstate.CharacterClassState{}
	if current, ok := state.CharacterClassStates[characterID]; ok {
		nextClassState = current
	}
	nextClassState = snapstate.WithActiveBeastform(nextClassState, p.ActiveBeastform)
	applyStatePatch(state, p.CharacterID, nil, p.Hope, nil, p.Stress, nil, nil, &nextClassState, nil, nil)
	return nil
}

func (f *Folder) foldBeastformDropped(state *snapstate.SnapshotState, p payload.BeastformDroppedPayload) error {
	characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
	if characterID == "" {
		return nil
	}
	nextClassState := snapstate.CharacterClassState{}
	if current, ok := state.CharacterClassStates[characterID]; ok {
		nextClassState = current
	}
	nextClassState = snapstate.WithActiveBeastform(nextClassState, nil)
	applyStatePatch(state, p.CharacterID, nil, nil, nil, nil, nil, nil, &nextClassState, nil, nil)
	return nil
}

func (f *Folder) foldCompanionExperienceBegun(state *snapstate.SnapshotState, p payload.CompanionExperienceBegunPayload) error {
	applyStatePatch(state, p.CharacterID, nil, nil, nil, nil, nil, nil, nil, nil, p.CompanionState)
	return nil
}

func (f *Folder) foldCompanionReturned(state *snapstate.SnapshotState, p payload.CompanionReturnedPayload) error {
	applyStatePatch(state, p.CharacterID, nil, nil, nil, p.Stress, nil, nil, nil, nil, p.CompanionState)
	return nil
}

func (f *Folder) foldConditionChanged(state *snapstate.SnapshotState, p payload.ConditionChangedPayload) error {
	applyCharacterConditionsChanged(state, p)
	return nil
}

func (f *Folder) foldLoadoutSwapped(state *snapstate.SnapshotState, p payload.LoadoutSwappedPayload) error {
	applyCharacterLoadoutSwapped(state, p)
	return nil
}

func (f *Folder) foldCharacterTemporaryArmorApplied(state *snapstate.SnapshotState, p payload.CharacterTemporaryArmorAppliedPayload) error {
	applyCharacterTemporaryArmorApplied(state, p)
	return nil
}

func (f *Folder) foldRestTaken(state *snapstate.SnapshotState, p payload.RestTakenPayload) error {
	state.GMFear = p.GMFear
	if state.GMFear < snapstate.GMFearMin || state.GMFear > snapstate.GMFearMax {
		return fmt.Errorf("rest_taken gm_fear_after must be in range %d..%d", snapstate.GMFearMin, snapstate.GMFearMax)
	}
	for _, participantID := range p.Participants {
		if p.RefreshRest || p.RefreshLongRest {
			clearRestTemporaryArmor(state, participantID.String(), p.RefreshRest, p.RefreshLongRest)
		}
	}
	return nil
}

func (f *Folder) foldCountdownCreated(state *snapstate.SnapshotState, p payload.CountdownCreatedPayload) error {
	applyCountdownUpsert(state, p.CountdownID, func(cs *snapstate.CountdownState) {
		cs.Name = p.Name
		cs.Kind = p.Kind
		cs.Current = p.Current
		cs.Max = p.Max
		cs.Direction = p.Direction
		cs.Looping = p.Looping
		cs.Variant = p.Variant
		cs.TriggerEventType = p.TriggerEventType
		cs.LinkedCountdownID = p.LinkedCountdownID
	})
	return nil
}

func (f *Folder) foldCountdownUpdated(state *snapstate.SnapshotState, p payload.CountdownUpdatedPayload) error {
	applyCountdownUpsert(state, p.CountdownID, func(cs *snapstate.CountdownState) {
		cs.Current = p.Value
		if p.Looped {
			cs.Looping = true
		}
	})
	return nil
}

func (f *Folder) foldCountdownDeleted(state *snapstate.SnapshotState, p payload.CountdownDeletedPayload) error {
	deleteCountdownState(state, p.CountdownID)
	return nil
}

func (f *Folder) foldDamageApplied(state *snapstate.SnapshotState, p payload.DamageAppliedPayload) error {
	applyDamageApplied(state, p.CharacterID, p.Hp, p.Stress, p.Armor)
	return nil
}

func (f *Folder) foldAdversaryDamageApplied(state *snapstate.SnapshotState, p payload.AdversaryDamageAppliedPayload) error {
	applyAdversaryDamage(state, p.AdversaryID, p.Hp, p.Armor)
	return nil
}

func (f *Folder) foldDowntimeMoveApplied(state *snapstate.SnapshotState, p payload.DowntimeMoveAppliedPayload) error {
	targetID := p.TargetCharacterID
	if strings.TrimSpace(targetID.String()) == "" {
		targetID = p.ActorCharacterID
	}
	if strings.TrimSpace(targetID.String()) == "" {
		return nil
	}
	applyStatePatch(state, targetID, p.HP, p.Hope, nil, p.Stress, p.Armor, nil, nil, nil, nil)
	return nil
}

func (f *Folder) foldAdversaryConditionChanged(state *snapstate.SnapshotState, p payload.AdversaryConditionChangedPayload) error {
	applyAdversaryConditionsChanged(state, p.AdversaryID, p.Conditions)
	return nil
}

func (f *Folder) foldAdversaryCreated(state *snapstate.SnapshotState, p payload.AdversaryCreatePayload) error {
	applyAdversaryCreated(state, p)
	return nil
}

func (f *Folder) foldAdversaryUpdated(state *snapstate.SnapshotState, p payload.AdversaryUpdatePayload) error {
	applyAdversaryUpdated(state, p)
	return nil
}

func (f *Folder) foldAdversaryDeleted(state *snapstate.SnapshotState, p payload.AdversaryDeletedPayload) error {
	delete(state.AdversaryStates, ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String())))
	return nil
}

func (f *Folder) foldEnvironmentEntityCreated(state *snapstate.SnapshotState, p payload.EnvironmentEntityCreatedPayload) error {
	environmentEntityID := ids.EnvironmentEntityID(strings.TrimSpace(p.EnvironmentEntityID.String()))
	if environmentEntityID == "" {
		return nil
	}
	state.EnvironmentStates[environmentEntityID] = snapstate.EnvironmentEntityState{
		CampaignID:          state.CampaignID,
		EnvironmentEntityID: environmentEntityID,
		EnvironmentID:       strings.TrimSpace(p.EnvironmentID),
		Name:                strings.TrimSpace(p.Name),
		Type:                strings.TrimSpace(p.Type),
		Tier:                p.Tier,
		Difficulty:          p.Difficulty,
		SessionID:           ids.SessionID(strings.TrimSpace(p.SessionID.String())),
		SceneID:             ids.SceneID(strings.TrimSpace(p.SceneID.String())),
		Notes:               strings.TrimSpace(p.Notes),
	}
	return nil
}

func (f *Folder) foldEnvironmentEntityUpdated(state *snapstate.SnapshotState, p payload.EnvironmentEntityUpdatedPayload) error {
	return f.foldEnvironmentEntityCreated(state, payload.EnvironmentEntityCreatedPayload(p))
}

func (f *Folder) foldEnvironmentEntityDeleted(state *snapstate.SnapshotState, p payload.EnvironmentEntityDeletedPayload) error {
	delete(state.EnvironmentStates, ids.EnvironmentEntityID(strings.TrimSpace(p.EnvironmentEntityID.String())))
	return nil
}

func (f *Folder) foldLevelUpApplied(state *snapstate.SnapshotState, p payload.LevelUpAppliedPayload) error {
	touchCharacter(state, p.CharacterID)
	if profile, ok := state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))]; ok {
		f.applyLevelUp(&profile, p)
		state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))] = profile
	}
	return nil
}

func (f *Folder) foldGoldUpdated(state *snapstate.SnapshotState, p payload.GoldUpdatedPayload) error {
	touchCharacter(state, p.CharacterID)
	if profile, ok := state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))]; ok {
		profile.GoldHandfuls = p.Handfuls
		profile.GoldBags = p.Bags
		profile.GoldChests = p.Chests
		state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))] = profile
	}
	return nil
}

func (f *Folder) foldDomainCardAcquired(state *snapstate.SnapshotState, p payload.DomainCardAcquiredPayload) error {
	touchCharacter(state, p.CharacterID)
	if profile, ok := state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))]; ok {
		profile.DomainCardIDs = snapstate.AppendUnique(profile.DomainCardIDs, p.CardID)
		state.CharacterProfiles[ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))] = profile
	}
	return nil
}

func (f *Folder) foldEquipmentSwapped(state *snapstate.SnapshotState, p payload.EquipmentSwappedPayload) error {
	touchCharacter(state, p.CharacterID)
	if strings.TrimSpace(p.ItemType) == "armor" {
		characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
		if profile, ok := state.CharacterProfiles[characterID]; ok {
			profile.EquippedArmorID = strings.TrimSpace(p.EquippedArmorID)
			if p.EvasionAfter != nil {
				profile.Evasion = *p.EvasionAfter
			}
			if p.MajorThresholdAfter != nil {
				profile.MajorThreshold = *p.MajorThresholdAfter
			}
			if p.SevereThresholdAfter != nil {
				profile.SevereThreshold = *p.SevereThresholdAfter
			}
			if p.ArmorScoreAfter != nil {
				profile.ArmorScore = *p.ArmorScoreAfter
			}
			if p.ArmorMaxAfter != nil {
				profile.ArmorMax = *p.ArmorMaxAfter
			}
			if p.SpellcastRollBonusAfter != nil {
				profile.SpellcastRollBonus = *p.SpellcastRollBonusAfter
			}
			if p.AgilityAfter != nil {
				profile.Agility = *p.AgilityAfter
			}
			if p.StrengthAfter != nil {
				profile.Strength = *p.StrengthAfter
			}
			if p.FinesseAfter != nil {
				profile.Finesse = *p.FinesseAfter
			}
			if p.InstinctAfter != nil {
				profile.Instinct = *p.InstinctAfter
			}
			if p.PresenceAfter != nil {
				profile.Presence = *p.PresenceAfter
			}
			if p.KnowledgeAfter != nil {
				profile.Knowledge = *p.KnowledgeAfter
			}
			state.CharacterProfiles[characterID] = profile
		}
		if p.ArmorAfter != nil {
			applyStatePatch(state, p.CharacterID, nil, nil, nil, nil, p.ArmorAfter, nil, nil, nil, nil)
		}
	}
	if p.StressCost > 0 {
		characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
		characterState := state.CharacterStates[characterID]
		characterState.CampaignID = state.CampaignID.String()
		characterState.CharacterID = characterID.String()
		characterState.Stress += p.StressCost
		state.CharacterStates[characterID] = characterState
	}
	return nil
}

func (f *Folder) foldConsumableUsed(state *snapstate.SnapshotState, p payload.ConsumableUsedPayload) error {
	touchCharacter(state, p.CharacterID)
	return nil
}

func (f *Folder) foldConsumableAcquired(state *snapstate.SnapshotState, p payload.ConsumableAcquiredPayload) error {
	touchCharacter(state, p.CharacterID)
	return nil
}

// --- helpers ---

func touchCharacter(state *snapstate.SnapshotState, rawID ids.CharacterID) {
	characterID := ids.CharacterID(strings.TrimSpace(rawID.String()))
	if characterID == "" {
		return
	}
	cs := state.CharacterStates[characterID]
	cs.CampaignID = state.CampaignID.String()
	cs.CharacterID = characterID.String()
	state.CharacterStates[characterID] = cs
}

func applyCharacterStatePatched(state *snapstate.SnapshotState, p payload.CharacterStatePatchedPayload) {
	applyStatePatch(state, p.CharacterID, p.HP, p.Hope, p.HopeMax, p.Stress, p.Armor, p.LifeState, p.ClassState, p.SubclassState, nil)
}

func applyStatePatch(state *snapstate.SnapshotState, characterID ids.CharacterID, hpAfter, hopeAfter, hopeMaxAfter, stressAfter, armorAfter *int, lifeStateAfter *string, classStateAfter *snapstate.CharacterClassState, subclassStateAfter *snapstate.CharacterSubclassState, companionStateAfter *snapstate.CharacterCompanionState) {
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

func applyCharacterConditionsChanged(state *snapstate.SnapshotState, p payload.ConditionChangedPayload) {
	characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyConditionPatch(&characterState, rules.ConditionCodes(p.Conditions))
	state.CharacterStates[characterID] = characterState
}

func applyCharacterLoadoutSwapped(state *snapstate.SnapshotState, p payload.LoadoutSwappedPayload) {
	characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyLoadoutSwap(&characterState, p.Stress)
	state.CharacterStates[characterID] = characterState
}

func applyCharacterTemporaryArmorApplied(state *snapstate.SnapshotState, p payload.CharacterTemporaryArmorAppliedPayload) {
	characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
	if characterID == "" {
		return
	}
	characterState := state.CharacterStates[characterID]
	characterState.CampaignID = state.CampaignID.String()
	characterState.CharacterID = characterID.String()
	reducer.ApplyTemporaryArmor(&characterState, reducer.TemporaryArmorPatch{
		Source:   strings.TrimSpace(p.Source),
		Duration: strings.TrimSpace(p.Duration),
		SourceID: strings.TrimSpace(p.SourceID),
		Amount:   p.Amount,
	})
	state.CharacterStates[characterID] = characterState
}

func clearRestTemporaryArmor(state *snapstate.SnapshotState, rawID string, clearShortRest bool, clearLongRest bool) {
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

func applyCountdownUpsert(state *snapstate.SnapshotState, countdownID ids.CountdownID, mutate func(*snapstate.CountdownState)) {
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

func deleteCountdownState(state *snapstate.SnapshotState, countdownID ids.CountdownID) {
	trimmed := ids.CountdownID(strings.TrimSpace(countdownID.String()))
	if trimmed == "" {
		return
	}
	delete(state.CountdownStates, trimmed)
}

func applyDamageApplied(state *snapstate.SnapshotState, rawID ids.CharacterID, hpAfter, stressAfter, armorAfter *int) {
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

func applyAdversaryDamage(state *snapstate.SnapshotState, rawID ids.AdversaryID, hpAfter, armorAfter *int) {
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

func applyAdversaryCreated(state *snapstate.SnapshotState, p payload.AdversaryCreatePayload) {
	adversaryID := ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.AdversaryEntryID = strings.TrimSpace(p.AdversaryEntryID)
	adversaryState.Name = p.Name
	adversaryState.Kind = strings.TrimSpace(p.Kind)
	adversaryState.SessionID = ids.SessionID(strings.TrimSpace(p.SessionID.String()))
	adversaryState.SceneID = ids.SceneID(strings.TrimSpace(p.SceneID.String()))
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
	adversaryState.SpotlightGateID = ids.GateID(strings.TrimSpace(p.SpotlightGateID.String()))
	adversaryState.SpotlightCount = p.SpotlightCount
	state.AdversaryStates[adversaryID] = adversaryState
}

func applyAdversaryUpdated(state *snapstate.SnapshotState, p payload.AdversaryUpdatePayload) {
	adversaryID := ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.AdversaryEntryID = strings.TrimSpace(p.AdversaryEntryID)
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

func applyAdversaryConditionsChanged(state *snapstate.SnapshotState, rawID ids.AdversaryID, after []rules.ConditionState) {
	adversaryID := ids.AdversaryID(strings.TrimSpace(rawID.String()))
	if adversaryID == "" {
		return
	}
	adversaryState := state.AdversaryStates[adversaryID]
	adversaryState.CampaignID = state.CampaignID
	adversaryState.AdversaryID = adversaryID
	adversaryState.Conditions = rules.ConditionCodes(after)
	state.AdversaryStates[adversaryID] = adversaryState
}
