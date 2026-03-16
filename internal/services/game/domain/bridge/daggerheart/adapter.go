package daggerheart

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Adapter applies Daggerheart-specific events to system projections.
type Adapter struct {
	store  projectionstore.Store
	router *module.AdapterRouter
}

// NewAdapter creates a Daggerheart adapter with all handlers registered.
func NewAdapter(store projectionstore.Store) *Adapter {
	a := &Adapter{store: store}
	a.router = a.buildRouter()
	return a
}

// ID returns the Daggerheart system identifier.
func (a *Adapter) ID() string {
	return SystemID
}

// Version returns the Daggerheart system version.
func (a *Adapter) Version() string {
	return SystemVersion
}

// HandledTypes returns the event types this adapter's Apply handles.
// Delegates to the router so the list reflects actual HandleAdapter registrations
// rather than event definitions. If a developer adds an event definition but
// forgets HandleAdapter, startup validation catches it immediately.
func (a *Adapter) HandledTypes() []event.Type {
	return a.router.HandledTypes()
}

// Apply applies a system-specific event to Daggerheart projections.
func (a *Adapter) Apply(ctx context.Context, evt event.Event) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("daggerheart store is not configured")
	}
	return a.router.Apply(ctx, evt)
}

// Snapshot loads the Daggerheart snapshot projection.
func (a *Adapter) Snapshot(ctx context.Context, campaignID string) (any, error) {
	if a == nil || a.store == nil {
		return nil, fmt.Errorf("daggerheart store is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	return a.store.GetDaggerheartSnapshot(ctx, campaignID)
}

// buildRouter creates and populates the adapter router with all handlers.
func (a *Adapter) buildRouter() *module.AdapterRouter {
	r := module.NewAdapterRouter()
	module.HandleAdapter(r, EventTypeCharacterProfileReplaced, a.handleCharacterProfileReplaced)
	module.HandleAdapter(r, EventTypeCharacterProfileDeleted, a.handleCharacterProfileDeleted)
	module.HandleAdapter(r, EventTypeDamageApplied, a.handleDamageApplied)
	module.HandleAdapter(r, EventTypeRestTaken, a.handleRestTaken)
	module.HandleAdapter(r, EventTypeCharacterTemporaryArmorApplied, a.handleCharacterTemporaryArmorApplied)
	module.HandleAdapter(r, EventTypeDowntimeMoveApplied, a.handleDowntimeMoveApplied)
	module.HandleAdapter(r, EventTypeLoadoutSwapped, a.handleLoadoutSwapped)
	module.HandleAdapter(r, EventTypeCharacterStatePatched, a.handleCharacterStatePatched)
	module.HandleAdapter(r, EventTypeBeastformTransformed, a.handleBeastformTransformed)
	module.HandleAdapter(r, EventTypeBeastformDropped, a.handleBeastformDropped)
	module.HandleAdapter(r, EventTypeCompanionExperienceBegun, a.handleCompanionExperienceBegun)
	module.HandleAdapter(r, EventTypeCompanionReturned, a.handleCompanionReturned)
	module.HandleAdapter(r, EventTypeConditionChanged, a.handleConditionChanged)
	module.HandleAdapter(r, EventTypeAdversaryConditionChanged, a.handleAdversaryConditionChanged)
	module.HandleAdapter(r, EventTypeGMFearChanged, a.handleGMFearChanged)
	module.HandleAdapter(r, EventTypeCountdownCreated, a.handleCountdownCreated)
	module.HandleAdapter(r, EventTypeCountdownUpdated, a.handleCountdownUpdated)
	module.HandleAdapter(r, EventTypeCountdownDeleted, a.handleCountdownDeleted)
	module.HandleAdapter(r, EventTypeAdversaryCreated, a.handleAdversaryCreated)
	module.HandleAdapter(r, EventTypeAdversaryDamageApplied, a.handleAdversaryDamageApplied)
	module.HandleAdapter(r, EventTypeAdversaryUpdated, a.handleAdversaryUpdated)
	module.HandleAdapter(r, EventTypeAdversaryDeleted, a.handleAdversaryDeleted)
	module.HandleAdapter(r, EventTypeEnvironmentEntityCreated, a.handleEnvironmentEntityCreated)
	module.HandleAdapter(r, EventTypeEnvironmentEntityUpdated, a.handleEnvironmentEntityUpdated)
	module.HandleAdapter(r, EventTypeEnvironmentEntityDeleted, a.handleEnvironmentEntityDeleted)
	module.HandleAdapter(r, EventTypeLevelUpApplied, a.handleLevelUpApplied)
	module.HandleAdapter(r, EventTypeGoldUpdated, a.handleGoldUpdated)
	module.HandleAdapter(r, EventTypeDomainCardAcquired, a.handleDomainCardAcquired)
	module.HandleAdapter(r, EventTypeEquipmentSwapped, a.handleEquipmentSwapped)
	module.HandleAdapter(r, EventTypeConsumableUsed, a.handleConsumableUsed)
	module.HandleAdapter(r, EventTypeConsumableAcquired, a.handleConsumableAcquired)
	return r
}

// Convention: handlers that store entity IDs directly (adversary CRUD, profile
// updates) apply strings.TrimSpace to guard against whitespace from upstream
// serialization. Handlers that delegate to applyStatePatch pass IDs as-is
// because the underlying store layer normalizes them. If a new handler stores
// an ID directly, follow the trim convention.

func (a *Adapter) handleDamageApplied(ctx context.Context, evt event.Event, payload DamageAppliedPayload) error {
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), payload.Hp, nil, nil, payload.Stress, payload.Armor, nil, nil, nil, nil, nil)
}

func (a *Adapter) handleRestTaken(ctx context.Context, evt event.Event, payload RestTakenPayload) error {
	if err := a.putSnapshot(ctx, string(evt.CampaignID), payload.GMFear, payload.ShortRests); err != nil {
		return err
	}
	for _, participantID := range payload.Participants {
		characterID := strings.TrimSpace(participantID.String())
		if payload.RefreshRest || payload.RefreshLongRest {
			if err := a.clearRestTemporaryArmor(ctx, string(evt.CampaignID), characterID, payload.RefreshRest, payload.RefreshLongRest); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Adapter) clearRestTemporaryArmor(ctx context.Context, campaignID, characterID string, clearShortRest bool, clearLongRest bool) error {
	state, exists, err := a.getCharacterStateIfExists(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	armorMax, err := a.characterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState, changed := projection.ClearRestTemporaryArmor(state, armorMax, clearShortRest, clearLongRest)
	if !changed {
		return nil
	}
	if err := a.putCharacterState(ctx, nextState); err != nil {
		return err
	}
	return nil
}

func (a *Adapter) handleDowntimeMoveApplied(ctx context.Context, evt event.Event, payload DowntimeMoveAppliedPayload) error {
	characterID := strings.TrimSpace(payload.TargetCharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(payload.ActorCharacterID.String())
	}
	if characterID == "" {
		return nil
	}
	return a.applyStatePatch(ctx, string(evt.CampaignID), characterID, payload.HP, payload.Hope, nil, payload.Stress, payload.Armor, nil, nil, nil, nil, nil)
}

func (a *Adapter) handleCharacterTemporaryArmorApplied(ctx context.Context, evt event.Event, payload CharacterTemporaryArmorAppliedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID.String())

	state, err := a.getCharacterStateOrDefault(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.characterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState, err := projection.ApplyTemporaryArmor(
		state,
		armorMax,
		payload.Source,
		payload.Duration,
		payload.SourceID,
		payload.Amount,
	)
	if err != nil {
		return err
	}
	return a.putCharacterState(ctx, nextState)
}

func (a *Adapter) handleLoadoutSwapped(ctx context.Context, evt event.Event, payload LoadoutSwappedPayload) error {
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), nil, nil, nil, payload.Stress, nil, nil, nil, nil, nil, nil)
}

func (a *Adapter) handleCharacterStatePatched(ctx context.Context, evt event.Event, payload CharacterStatePatchedPayload) error {
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), payload.HP, payload.Hope, payload.HopeMax, payload.Stress, payload.Armor, payload.LifeState, payload.ClassState, payload.SubclassState, nil, payload.ImpenetrableUsedThisShortRest)
}

func (a *Adapter) handleBeastformTransformed(ctx context.Context, evt event.Event, payload BeastformTransformedPayload) error {
	state, err := a.getCharacterStateOrDefault(ctx, string(evt.CampaignID), payload.CharacterID.String())
	if err != nil {
		return err
	}
	nextClassState := WithActiveBeastform(classStateFromProjection(state.ClassState), payload.ActiveBeastform)
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), nil, payload.Hope, nil, payload.Stress, nil, nil, &nextClassState, nil, nil, nil)
}

func (a *Adapter) handleBeastformDropped(ctx context.Context, evt event.Event, payload BeastformDroppedPayload) error {
	state, err := a.getCharacterStateOrDefault(ctx, string(evt.CampaignID), payload.CharacterID.String())
	if err != nil {
		return err
	}
	nextClassState := WithActiveBeastform(classStateFromProjection(state.ClassState), nil)
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), nil, nil, nil, nil, nil, nil, &nextClassState, nil, nil, nil)
}

func (a *Adapter) handleCompanionExperienceBegun(ctx context.Context, evt event.Event, payload CompanionExperienceBegunPayload) error {
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), nil, nil, nil, nil, nil, nil, nil, nil, payload.CompanionState, nil)
}

func (a *Adapter) handleCompanionReturned(ctx context.Context, evt event.Event, payload CompanionReturnedPayload) error {
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), nil, nil, nil, payload.Stress, nil, nil, nil, nil, payload.CompanionState, nil)
}

func (a *Adapter) handleConditionChanged(ctx context.Context, evt event.Event, payload ConditionChangedPayload) error {
	// RollSeq is event-only metadata not validated in ValidatePayload.
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return fmt.Errorf("condition_changed roll_seq must be positive")
	}
	normalizedAfter, err := NormalizeConditionStates(payload.Conditions)
	if err != nil {
		return fmt.Errorf("condition_changed conditions_after: %w", err)
	}
	return a.applyConditionPatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), normalizedAfter)
}

func (a *Adapter) handleAdversaryConditionChanged(ctx context.Context, evt event.Event, payload AdversaryConditionChangedPayload) error {
	// RollSeq is event-only metadata not validated in ValidatePayload.
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return fmt.Errorf("adversary_condition_changed roll_seq must be positive")
	}
	normalizedAfter, err := NormalizeConditionStates(payload.Conditions)
	if err != nil {
		return fmt.Errorf("adversary_condition_changed conditions_after: %w", err)
	}
	return a.applyAdversaryConditionPatch(ctx, string(evt.CampaignID), payload.AdversaryID.String(), normalizedAfter)
}

func (a *Adapter) handleGMFearChanged(ctx context.Context, evt event.Event, payload GMFearChangedPayload) error {
	// Range validation before writing to storage.
	if payload.Value < GMFearMin || payload.Value > GMFearMax {
		return fmt.Errorf("gm_fear_changed value must be in range %d..%d", GMFearMin, GMFearMax)
	}
	shortRests := a.snapshotShortRests(ctx, string(evt.CampaignID))
	return a.putSnapshot(ctx, string(evt.CampaignID), payload.Value, shortRests)
}

func (a *Adapter) handleCountdownCreated(ctx context.Context, evt event.Event, payload CountdownCreatedPayload) error {
	return a.store.PutDaggerheartCountdown(ctx, projectionstore.DaggerheartCountdown{
		CampaignID:        string(evt.CampaignID),
		CountdownID:       payload.CountdownID.String(),
		Name:              payload.Name,
		Kind:              payload.Kind,
		Current:           payload.Current,
		Max:               payload.Max,
		Direction:         payload.Direction,
		Looping:           payload.Looping,
		Variant:           payload.Variant,
		TriggerEventType:  payload.TriggerEventType,
		LinkedCountdownID: payload.LinkedCountdownID.String(),
	})
}

func (a *Adapter) handleCountdownUpdated(ctx context.Context, evt event.Event, payload CountdownUpdatedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), payload.CountdownID.String())
	if err != nil {
		return err
	}
	nextCountdown, err := projection.ApplyCountdownUpdate(countdown, payload.Value)
	if err != nil {
		return err
	}
	return a.store.PutDaggerheartCountdown(ctx, nextCountdown)
}

func (a *Adapter) handleCountdownDeleted(ctx context.Context, evt event.Event, payload CountdownDeletedPayload) error {
	return a.store.DeleteDaggerheartCountdown(ctx, string(evt.CampaignID), payload.CountdownID.String())
}

func (a *Adapter) handleAdversaryCreated(ctx context.Context, evt event.Event, payload AdversaryCreatedPayload) error {
	if err := projection.ValidateAdversaryStats(payload.HP, payload.HPMax, payload.Stress, payload.StressMax, payload.Evasion, payload.Major, payload.Severe, payload.Armor); err != nil {
		return err
	}
	createdAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, projectionstore.DaggerheartAdversary{
		CampaignID:        string(evt.CampaignID),
		AdversaryID:       strings.TrimSpace(payload.AdversaryID.String()),
		AdversaryEntryID:  strings.TrimSpace(payload.AdversaryEntryID),
		Name:              strings.TrimSpace(payload.Name),
		Kind:              strings.TrimSpace(payload.Kind),
		SessionID:         strings.TrimSpace(payload.SessionID.String()),
		SceneID:           strings.TrimSpace(payload.SceneID.String()),
		Notes:             strings.TrimSpace(payload.Notes),
		HP:                payload.HP,
		HPMax:             payload.HPMax,
		Stress:            payload.Stress,
		StressMax:         payload.StressMax,
		Evasion:           payload.Evasion,
		Major:             payload.Major,
		Severe:            payload.Severe,
		Armor:             payload.Armor,
		FeatureStates:     toProjectionAdversaryFeatureStates(payload.FeatureStates),
		PendingExperience: toProjectionAdversaryPendingExperience(payload.PendingExperience),
		SpotlightGateID:   strings.TrimSpace(payload.SpotlightGateID.String()),
		SpotlightCount:    payload.SpotlightCount,
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
	})
}

func (a *Adapter) handleAdversaryUpdated(ctx context.Context, evt event.Event, payload AdversaryUpdatedPayload) error {
	adversaryID := strings.TrimSpace(payload.AdversaryID.String())
	if err := projection.ValidateAdversaryStats(payload.HP, payload.HPMax, payload.Stress, payload.StressMax, payload.Evasion, payload.Major, payload.Severe, payload.Armor); err != nil {
		return err
	}
	current, err := a.store.GetDaggerheartAdversary(ctx, string(evt.CampaignID), adversaryID)
	if err != nil {
		return err
	}
	updatedAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, projectionstore.DaggerheartAdversary{
		CampaignID:        string(evt.CampaignID),
		AdversaryID:       adversaryID,
		AdversaryEntryID:  strings.TrimSpace(payload.AdversaryEntryID),
		Name:              strings.TrimSpace(payload.Name),
		Kind:              strings.TrimSpace(payload.Kind),
		SessionID:         strings.TrimSpace(payload.SessionID.String()),
		SceneID:           strings.TrimSpace(payload.SceneID.String()),
		Notes:             strings.TrimSpace(payload.Notes),
		HP:                payload.HP,
		HPMax:             payload.HPMax,
		Stress:            payload.Stress,
		StressMax:         payload.StressMax,
		Evasion:           payload.Evasion,
		Major:             payload.Major,
		Severe:            payload.Severe,
		Armor:             payload.Armor,
		Conditions:        current.Conditions,
		FeatureStates:     toProjectionAdversaryFeatureStates(payload.FeatureStates),
		PendingExperience: toProjectionAdversaryPendingExperience(payload.PendingExperience),
		SpotlightGateID:   strings.TrimSpace(payload.SpotlightGateID.String()),
		SpotlightCount:    payload.SpotlightCount,
		CreatedAt:         current.CreatedAt,
		UpdatedAt:         updatedAt,
	})
}

func toProjectionAdversaryFeatureStates(in []AdversaryFeatureState) []projectionstore.DaggerheartAdversaryFeatureState {
	out := make([]projectionstore.DaggerheartAdversaryFeatureState, 0, len(in))
	for _, featureState := range in {
		out = append(out, projectionstore.DaggerheartAdversaryFeatureState{
			FeatureID:       strings.TrimSpace(featureState.FeatureID),
			Status:          strings.TrimSpace(featureState.Status),
			FocusedTargetID: strings.TrimSpace(featureState.FocusedTargetID),
		})
	}
	return out
}

func toProjectionAdversaryPendingExperience(in *AdversaryPendingExperience) *projectionstore.DaggerheartAdversaryPendingExperience {
	if in == nil {
		return nil
	}
	return &projectionstore.DaggerheartAdversaryPendingExperience{
		Name:     strings.TrimSpace(in.Name),
		Modifier: in.Modifier,
	}
}

func (a *Adapter) handleAdversaryDamageApplied(ctx context.Context, evt event.Event, payload AdversaryDamageAppliedPayload) error {
	adversaryID := strings.TrimSpace(payload.AdversaryID.String())
	// State consistency: merge payload with current projection state.
	current, err := a.store.GetDaggerheartAdversary(ctx, string(evt.CampaignID), adversaryID)
	if err != nil {
		return err
	}
	next, err := projection.ApplyAdversaryDamagePatch(current, payload.Hp, payload.Armor)
	if err != nil {
		return err
	}
	next.CampaignID = string(evt.CampaignID)
	next.AdversaryID = adversaryID
	updatedAt := evt.Timestamp.UTC()
	next.UpdatedAt = updatedAt
	return a.store.PutDaggerheartAdversary(ctx, next)
}

func (a *Adapter) handleAdversaryDeleted(ctx context.Context, evt event.Event, payload AdversaryDeletedPayload) error {
	return a.store.DeleteDaggerheartAdversary(ctx, string(evt.CampaignID), strings.TrimSpace(payload.AdversaryID.String()))
}

func (a *Adapter) handleEnvironmentEntityCreated(ctx context.Context, evt event.Event, payload EnvironmentEntityCreatedPayload) error {
	createdAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartEnvironmentEntity(ctx, projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          string(evt.CampaignID),
		EnvironmentEntityID: strings.TrimSpace(payload.EnvironmentEntityID.String()),
		EnvironmentID:       strings.TrimSpace(payload.EnvironmentID),
		Name:                strings.TrimSpace(payload.Name),
		Type:                strings.TrimSpace(payload.Type),
		Tier:                payload.Tier,
		Difficulty:          payload.Difficulty,
		SessionID:           strings.TrimSpace(payload.SessionID.String()),
		SceneID:             strings.TrimSpace(payload.SceneID.String()),
		Notes:               strings.TrimSpace(payload.Notes),
		CreatedAt:           createdAt,
		UpdatedAt:           createdAt,
	})
}

func (a *Adapter) handleEnvironmentEntityUpdated(ctx context.Context, evt event.Event, payload EnvironmentEntityUpdatedPayload) error {
	environmentEntityID := strings.TrimSpace(payload.EnvironmentEntityID.String())
	current, err := a.store.GetDaggerheartEnvironmentEntity(ctx, string(evt.CampaignID), environmentEntityID)
	if err != nil {
		return err
	}
	return a.store.PutDaggerheartEnvironmentEntity(ctx, projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          string(evt.CampaignID),
		EnvironmentEntityID: environmentEntityID,
		EnvironmentID:       strings.TrimSpace(payload.EnvironmentID),
		Name:                strings.TrimSpace(payload.Name),
		Type:                strings.TrimSpace(payload.Type),
		Tier:                payload.Tier,
		Difficulty:          payload.Difficulty,
		SessionID:           strings.TrimSpace(payload.SessionID.String()),
		SceneID:             strings.TrimSpace(payload.SceneID.String()),
		Notes:               strings.TrimSpace(payload.Notes),
		CreatedAt:           current.CreatedAt,
		UpdatedAt:           evt.Timestamp.UTC(),
	})
}

func (a *Adapter) handleEnvironmentEntityDeleted(ctx context.Context, evt event.Event, payload EnvironmentEntityDeletedPayload) error {
	return a.store.DeleteDaggerheartEnvironmentEntity(ctx, string(evt.CampaignID), strings.TrimSpace(payload.EnvironmentEntityID.String()))
}

func (a *Adapter) handleLevelUpApplied(ctx context.Context, evt event.Event, payload LevelUpAppliedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID.String())
	storedProfile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for level-up: %w", err)
		}
		// No profile yet — nothing to project onto.
		return nil
	}

	profile := CharacterProfileFromStorage(storedProfile)
	applyLevelUpToCharacterProfile(&profile, payload)
	return a.store.PutDaggerheartCharacterProfile(ctx, profile.ToStorage(string(evt.CampaignID), characterID))
}

func (a *Adapter) handleGoldUpdated(ctx context.Context, evt event.Event, payload GoldUpdatedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID.String())
	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for gold update: %w", err)
		}
		return nil
	}
	profile.GoldHandfuls = payload.Handfuls
	profile.GoldBags = payload.Bags
	profile.GoldChests = payload.Chests
	return a.store.PutDaggerheartCharacterProfile(ctx, profile)
}

func (a *Adapter) handleDomainCardAcquired(ctx context.Context, evt event.Event, payload DomainCardAcquiredPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID.String())
	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for domain card acquire: %w", err)
		}
		return nil
	}
	profile.DomainCardIDs = appendUnique(profile.DomainCardIDs, strings.TrimSpace(payload.CardID))
	return a.store.PutDaggerheartCharacterProfile(ctx, profile)
}

func (a *Adapter) handleEquipmentSwapped(ctx context.Context, evt event.Event, payload EquipmentSwappedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		return nil
	}
	if strings.TrimSpace(payload.ItemType) == "armor" {
		profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return fmt.Errorf("get daggerheart character profile for equipment swap: %w", err)
			}
		} else {
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
			if err := a.store.PutDaggerheartCharacterProfile(ctx, profile); err != nil {
				return fmt.Errorf("put daggerheart character profile for equipment swap: %w", err)
			}
		}
		if payload.ArmorAfter != nil {
			if err := a.applyStatePatch(ctx, string(evt.CampaignID), characterID, nil, nil, nil, nil, payload.ArmorAfter, nil, nil, nil, nil, nil); err != nil {
				return err
			}
		}
	}
	if payload.StressCost > 0 {
		state, err := a.getCharacterStateOrDefault(ctx, string(evt.CampaignID), characterID)
		if err != nil {
			return err
		}
		stressAfter := state.Stress + payload.StressCost
		if err := a.applyStatePatch(ctx, string(evt.CampaignID), characterID, nil, nil, nil, &stressAfter, nil, nil, nil, nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) handleConsumableUsed(_ context.Context, _ event.Event, _ ConsumableUsedPayload) error {
	// Consumable state is event-sourced; no projection update needed.
	return nil
}

func (a *Adapter) handleConsumableAcquired(_ context.Context, _ event.Event, _ ConsumableAcquiredPayload) error {
	// Consumable state is event-sourced; no projection update needed.
	return nil
}

func appendUnique(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

func (a *Adapter) characterArmorMax(ctx context.Context, state projectionstore.DaggerheartCharacterState) (int, error) {
	armorMax := projection.FallbackArmorMaxFromState(state)
	if strings.TrimSpace(state.CampaignID) == "" || strings.TrimSpace(state.CharacterID) == "" {
		return armorMax, nil
	}

	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, state.CampaignID, state.CharacterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return armorMax, nil
		}
		return 0, fmt.Errorf("get daggerheart character profile: %w", err)
	}
	return profile.ArmorMax, nil
}

func (a *Adapter) applyStatePatch(ctx context.Context, campaignID, characterID string, hpAfter, hopeAfter, hopeMaxAfter, stressAfter, armorAfter *int, lifeStateAfter *string, classStateAfter *CharacterClassState, subclassStateAfter *CharacterSubclassState, companionStateAfter *CharacterCompanionState, impenetrableUsedThisShortRestAfter *bool) error {
	state, err := a.getCharacterStateOrDefault(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.characterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState, err := projection.ApplyStatePatch(
		state,
		armorMax,
		hpAfter,
		hopeAfter,
		hopeMaxAfter,
		stressAfter,
		armorAfter,
		lifeStateAfter,
		classStateToProjection(classStateAfter),
		subclassStateToProjection(subclassStateAfter),
		companionStateToProjection(companionStateAfter),
		impenetrableUsedThisShortRestAfter,
	)
	if err != nil {
		return err
	}
	return a.putCharacterState(ctx, nextState)
}

func subclassStateToProjection(value *CharacterSubclassState) *projectionstore.DaggerheartSubclassState {
	normalized := normalizedSubclassStatePtr(value)
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

func classStateToProjection(value *CharacterClassState) *projectionstore.DaggerheartClassState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &projectionstore.DaggerheartClassState{
		AttackBonusUntilRest:       normalized.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest: normalized.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest: normalized.DifficultyPenaltyUntilRest,
		FocusTargetID:              normalized.FocusTargetID,
		ActiveBeastform:            activeBeastformToProjection(normalized.ActiveBeastform),
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

func companionStateToProjection(value *CharacterCompanionState) *projectionstore.DaggerheartCompanionState {
	normalized := normalizedCompanionStatePtr(value)
	if normalized == nil {
		return nil
	}
	return &projectionstore.DaggerheartCompanionState{
		Status:             normalized.Status,
		ActiveExperienceID: normalized.ActiveExperienceID,
	}
}

func activeBeastformToProjection(value *CharacterActiveBeastformState) *projectionstore.DaggerheartActiveBeastformState {
	normalized := normalizedActiveBeastformPtr(value)
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

func classStateFromProjection(value projectionstore.DaggerheartClassState) CharacterClassState {
	damageDice := []CharacterDamageDie(nil)
	active := normalizedActiveBeastformPtr(nil)
	if value.ActiveBeastform != nil {
		damageDice = make([]CharacterDamageDie, 0, len(value.ActiveBeastform.DamageDice))
		for _, die := range value.ActiveBeastform.DamageDice {
			damageDice = append(damageDice, CharacterDamageDie{Count: die.Count, Sides: die.Sides})
		}
		active = &CharacterActiveBeastformState{
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
	return CharacterClassState{
		AttackBonusUntilRest:            value.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest:      value.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest:      value.DifficultyPenaltyUntilRest,
		FocusTargetID:                   value.FocusTargetID,
		ActiveBeastform:                 active,
		StrangePatternsNumber:           value.StrangePatternsNumber,
		RallyDice:                       append([]int(nil), value.RallyDice...),
		PrayerDice:                      append([]int(nil), value.PrayerDice...),
		ChannelRawPowerUsedThisLongRest: value.ChannelRawPowerUsedThisLongRest,
		Unstoppable: CharacterUnstoppableState{
			Active:           value.Unstoppable.Active,
			CurrentValue:     value.Unstoppable.CurrentValue,
			DieSides:         value.Unstoppable.DieSides,
			UsedThisLongRest: value.Unstoppable.UsedThisLongRest,
		},
	}.Normalized()
}

func subclassStateFromProjection(value *projectionstore.DaggerheartSubclassState) *CharacterSubclassState {
	if value == nil {
		return nil
	}
	return normalizedSubclassStatePtr(&CharacterSubclassState{
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

func companionStateFromProjection(value *projectionstore.DaggerheartCompanionState) *CharacterCompanionState {
	if value == nil {
		return nil
	}
	return normalizedCompanionStatePtr(&CharacterCompanionState{
		Status:             value.Status,
		ActiveExperienceID: value.ActiveExperienceID,
	})
}

func (a *Adapter) applyConditionPatch(ctx context.Context, campaignID, characterID string, conditions []ConditionState) error {
	state, err := a.getCharacterStateOrDefault(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.characterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState := projection.ApplyConditionPatch(state, armorMax, conditionStatesToProjection(conditions))
	return a.putCharacterState(ctx, nextState)
}

func (a *Adapter) applyAdversaryConditionPatch(ctx context.Context, campaignID, adversaryID string, conditions []ConditionState) error {
	adversary, err := a.store.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return fmt.Errorf("get daggerheart adversary: %w", err)
	}
	next := projection.ApplyAdversaryConditionPatch(adversary, conditionStatesToProjection(conditions))
	if err := a.store.PutDaggerheartAdversary(ctx, next); err != nil {
		return fmt.Errorf("put daggerheart adversary: %w", err)
	}
	return nil
}

func conditionStatesToProjection(values []ConditionState) []projectionstore.DaggerheartConditionState {
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
