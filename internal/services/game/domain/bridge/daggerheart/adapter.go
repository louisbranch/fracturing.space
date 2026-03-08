package daggerheart

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/projection"
	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Adapter applies Daggerheart-specific events to system projections.
type Adapter struct {
	store  storage.DaggerheartStore
	router *module.AdapterRouter
}

// NewAdapter creates a Daggerheart adapter with all handlers registered.
func NewAdapter(store storage.DaggerheartStore) *Adapter {
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
	module.HandleAdapter(r, EventTypeDamageApplied, a.handleDamageApplied)
	module.HandleAdapter(r, EventTypeRestTaken, a.handleRestTaken)
	module.HandleAdapter(r, EventTypeCharacterTemporaryArmorApplied, a.handleCharacterTemporaryArmorApplied)
	module.HandleAdapter(r, EventTypeDowntimeMoveApplied, a.handleDowntimeMoveApplied)
	module.HandleAdapter(r, EventTypeLoadoutSwapped, a.handleLoadoutSwapped)
	module.HandleAdapter(r, EventTypeCharacterStatePatched, a.handleCharacterStatePatched)
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
	module.HandleAdapter(r, EventTypeLevelUpApplied, a.handleLevelUpApplied)
	module.HandleAdapter(r, EventTypeGoldUpdated, a.handleGoldUpdated)
	module.HandleAdapter(r, EventTypeDomainCardAcquired, a.handleDomainCardAcquired)
	module.HandleAdapter(r, EventTypeEquipmentSwapped, a.handleEquipmentSwapped)
	module.HandleAdapter(r, EventTypeConsumableUsed, a.handleConsumableUsed)
	module.HandleAdapter(r, EventTypeConsumableAcquired, a.handleConsumableAcquired)
	return r
}

func (a *Adapter) handleDamageApplied(ctx context.Context, evt event.Event, payload DamageAppliedPayload) error {
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), payload.Hp, nil, nil, nil, payload.Armor, nil)
}

func (a *Adapter) handleRestTaken(ctx context.Context, evt event.Event, payload RestTakenPayload) error {
	if err := a.putSnapshot(ctx, string(evt.CampaignID), payload.GMFear, payload.ShortRests); err != nil {
		return err
	}
	for _, patch := range payload.CharacterStates {
		characterID := strings.TrimSpace(patch.CharacterID.String())
		if payload.RefreshRest || payload.RefreshLongRest {
			if err := a.clearRestTemporaryArmor(ctx, string(evt.CampaignID), characterID, payload.RefreshRest, payload.RefreshLongRest); err != nil {
				return err
			}
		}
		if err := a.applyStatePatch(ctx, string(evt.CampaignID), characterID, nil, patch.Hope, nil, patch.Stress, patch.Armor, nil); err != nil {
			return err
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
	characterID := strings.TrimSpace(payload.CharacterID.String())
	state, err := a.getCharacterStateOrDefault(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.characterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState, err := projection.ApplyDowntimeMove(state, armorMax, payload.Move, payload.Hope, payload.Stress, payload.Armor)
	if err != nil {
		return err
	}
	return a.putCharacterState(ctx, nextState)
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
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), nil, nil, nil, payload.Stress, nil, nil)
}

func (a *Adapter) handleCharacterStatePatched(ctx context.Context, evt event.Event, payload CharacterStatePatchedPayload) error {
	return a.applyStatePatch(ctx, string(evt.CampaignID), payload.CharacterID.String(), payload.HP, payload.Hope, payload.HopeMax, payload.Stress, payload.Armor, payload.LifeState)
}

func (a *Adapter) handleConditionChanged(ctx context.Context, evt event.Event, payload ConditionChangedPayload) error {
	// RollSeq is event-only metadata not validated in ValidatePayload.
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return fmt.Errorf("condition_changed roll_seq must be positive")
	}
	normalizedAfter, err := NormalizeConditions(payload.Conditions)
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
	normalizedAfter, err := NormalizeConditions(payload.Conditions)
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
	return a.store.PutDaggerheartCountdown(ctx, storage.DaggerheartCountdown{
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
	return a.store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{
		CampaignID:  string(evt.CampaignID),
		AdversaryID: strings.TrimSpace(payload.AdversaryID.String()),
		Name:        strings.TrimSpace(payload.Name),
		Kind:        strings.TrimSpace(payload.Kind),
		SessionID:   strings.TrimSpace(payload.SessionID.String()),
		Notes:       strings.TrimSpace(payload.Notes),
		HP:          payload.HP,
		HPMax:       payload.HPMax,
		Stress:      payload.Stress,
		StressMax:   payload.StressMax,
		Evasion:     payload.Evasion,
		Major:       payload.Major,
		Severe:      payload.Severe,
		Armor:       payload.Armor,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
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
	return a.store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{
		CampaignID:  string(evt.CampaignID),
		AdversaryID: adversaryID,
		Name:        strings.TrimSpace(payload.Name),
		Kind:        strings.TrimSpace(payload.Kind),
		SessionID:   strings.TrimSpace(payload.SessionID.String()),
		Notes:       strings.TrimSpace(payload.Notes),
		HP:          payload.HP,
		HPMax:       payload.HPMax,
		Stress:      payload.Stress,
		StressMax:   payload.StressMax,
		Evasion:     payload.Evasion,
		Major:       payload.Major,
		Severe:      payload.Severe,
		Armor:       payload.Armor,
		Conditions:  current.Conditions,
		CreatedAt:   current.CreatedAt,
		UpdatedAt:   updatedAt,
	})
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

func (a *Adapter) handleLevelUpApplied(ctx context.Context, evt event.Event, payload LevelUpAppliedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID.String())
	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for level-up: %w", err)
		}
		// No profile yet — nothing to project onto.
		return nil
	}

	profile.Level = payload.Level
	profile.MajorThreshold += payload.ThresholdDelta
	profile.SevereThreshold += payload.ThresholdDelta * 2

	for _, adv := range payload.Advancements {
		switch adv.Type {
		case "trait_increase":
			applyProfileTraitIncrease(&profile, adv.Trait)
		case "add_hp_slots":
			profile.HpMax++
		case "add_stress_slots":
			profile.StressMax++
		case "increase_evasion":
			profile.Evasion++
		case "increase_proficiency":
			profile.Proficiency++
		case "increase_experience":
			// Experience additions are content-level; no profile field change needed.
		case "domain_card":
			if adv.DomainCardID != "" {
				profile.DomainCardIDs = appendUnique(profile.DomainCardIDs, adv.DomainCardID)
			}
		case "upgraded_subclass":
			if adv.SubclassCardID != "" {
				profile.DomainCardIDs = appendUnique(profile.DomainCardIDs, adv.SubclassCardID)
			}
		}
	}

	// Step 4 domain card acquisition.
	if payload.NewDomainCardID != "" {
		profile.DomainCardIDs = appendUnique(profile.DomainCardIDs, payload.NewDomainCardID)
	}

	return a.store.PutDaggerheartCharacterProfile(ctx, profile)
}

func applyProfileTraitIncrease(profile *storage.DaggerheartCharacterProfile, trait string) {
	switch trait {
	case "agility":
		profile.Agility++
	case "strength":
		profile.Strength++
	case "finesse":
		profile.Finesse++
	case "instinct":
		profile.Instinct++
	case "presence":
		profile.Presence++
	case "knowledge":
		profile.Knowledge++
	}
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

func (a *Adapter) handleEquipmentSwapped(_ context.Context, _ event.Event, _ EquipmentSwappedPayload) error {
	// Equipment state is event-sourced; no projection update needed.
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

func (a *Adapter) characterArmorMax(ctx context.Context, state storage.DaggerheartCharacterState) (int, error) {
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

func (a *Adapter) applyStatePatch(ctx context.Context, campaignID, characterID string, hpAfter, hopeAfter, hopeMaxAfter, stressAfter, armorAfter *int, lifeStateAfter *string) error {
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
	)
	if err != nil {
		return err
	}
	return a.putCharacterState(ctx, nextState)
}

func (a *Adapter) applyConditionPatch(ctx context.Context, campaignID, characterID string, conditions []string) error {
	state, err := a.getCharacterStateOrDefault(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.characterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState := projection.ApplyConditionPatch(state, armorMax, conditions)
	return a.putCharacterState(ctx, nextState)
}

func (a *Adapter) applyAdversaryConditionPatch(ctx context.Context, campaignID, adversaryID string, conditions []string) error {
	adversary, err := a.store.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return fmt.Errorf("get daggerheart adversary: %w", err)
	}
	next := projection.ApplyAdversaryConditionPatch(adversary, conditions)
	if err := a.store.PutDaggerheartAdversary(ctx, next); err != nil {
		return fmt.Errorf("put daggerheart adversary: %w", err)
	}
	return nil
}
