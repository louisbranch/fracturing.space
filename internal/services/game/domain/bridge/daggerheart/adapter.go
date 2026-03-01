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
	return r
}

func (a *Adapter) handleDamageApplied(ctx context.Context, evt event.Event, payload DamageAppliedPayload) error {
	return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, payload.HpAfter, nil, nil, nil, payload.ArmorAfter, nil)
}

func (a *Adapter) handleRestTaken(ctx context.Context, evt event.Event, payload RestTakenPayload) error {
	if err := a.putSnapshot(ctx, evt.CampaignID, payload.GMFearAfter, payload.ShortRestsAfter); err != nil {
		return err
	}
	for _, patch := range payload.CharacterStates {
		characterID := strings.TrimSpace(patch.CharacterID)
		if payload.RefreshRest || payload.RefreshLongRest {
			if err := a.clearRestTemporaryArmor(ctx, evt.CampaignID, characterID, payload.RefreshRest, payload.RefreshLongRest); err != nil {
				return err
			}
		}
		if err := a.applyStatePatch(ctx, evt.CampaignID, characterID, nil, patch.HopeAfter, nil, patch.StressAfter, patch.ArmorAfter, nil); err != nil {
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
	characterID := strings.TrimSpace(payload.CharacterID)
	state, err := a.getCharacterStateOrDefault(ctx, evt.CampaignID, characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.characterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState, err := projection.ApplyDowntimeMove(state, armorMax, payload.Move, payload.HopeAfter, payload.StressAfter, payload.ArmorAfter)
	if err != nil {
		return err
	}
	return a.putCharacterState(ctx, nextState)
}

func (a *Adapter) handleCharacterTemporaryArmorApplied(ctx context.Context, evt event.Event, payload CharacterTemporaryArmorAppliedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID)

	state, err := a.getCharacterStateOrDefault(ctx, evt.CampaignID, characterID)
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
	return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, nil, nil, nil, payload.StressAfter, nil, nil)
}

func (a *Adapter) handleCharacterStatePatched(ctx context.Context, evt event.Event, payload CharacterStatePatchedPayload) error {
	return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, payload.HPAfter, payload.HopeAfter, payload.HopeMaxAfter, payload.StressAfter, payload.ArmorAfter, payload.LifeStateAfter)
}

func (a *Adapter) handleConditionChanged(ctx context.Context, evt event.Event, payload ConditionChangedPayload) error {
	// RollSeq is event-only metadata not validated in ValidatePayload.
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return fmt.Errorf("condition_changed roll_seq must be positive")
	}
	normalizedAfter, err := NormalizeConditions(payload.ConditionsAfter)
	if err != nil {
		return fmt.Errorf("condition_changed conditions_after: %w", err)
	}
	return a.applyConditionPatch(ctx, evt.CampaignID, payload.CharacterID, normalizedAfter)
}

func (a *Adapter) handleAdversaryConditionChanged(ctx context.Context, evt event.Event, payload AdversaryConditionChangedPayload) error {
	// RollSeq is event-only metadata not validated in ValidatePayload.
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return fmt.Errorf("adversary_condition_changed roll_seq must be positive")
	}
	normalizedAfter, err := NormalizeConditions(payload.ConditionsAfter)
	if err != nil {
		return fmt.Errorf("adversary_condition_changed conditions_after: %w", err)
	}
	return a.applyAdversaryConditionPatch(ctx, evt.CampaignID, payload.AdversaryID, normalizedAfter)
}

func (a *Adapter) handleGMFearChanged(ctx context.Context, evt event.Event, payload GMFearChangedPayload) error {
	// Range validation before writing to storage.
	if payload.After < GMFearMin || payload.After > GMFearMax {
		return fmt.Errorf("gm_fear_changed after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	shortRests := a.snapshotShortRests(ctx, evt.CampaignID)
	return a.putSnapshot(ctx, evt.CampaignID, payload.After, shortRests)
}

func (a *Adapter) handleCountdownCreated(ctx context.Context, evt event.Event, payload CountdownCreatedPayload) error {
	return a.store.PutDaggerheartCountdown(ctx, storage.DaggerheartCountdown{
		CampaignID:  evt.CampaignID,
		CountdownID: payload.CountdownID,
		Name:        payload.Name,
		Kind:        payload.Kind,
		Current:     payload.Current,
		Max:         payload.Max,
		Direction:   payload.Direction,
		Looping:     payload.Looping,
	})
}

func (a *Adapter) handleCountdownUpdated(ctx context.Context, evt event.Event, payload CountdownUpdatedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, evt.CampaignID, payload.CountdownID)
	if err != nil {
		return err
	}
	nextCountdown, err := projection.ApplyCountdownUpdate(countdown, payload.Before, payload.After)
	if err != nil {
		return err
	}
	return a.store.PutDaggerheartCountdown(ctx, nextCountdown)
}

func (a *Adapter) handleCountdownDeleted(ctx context.Context, evt event.Event, payload CountdownDeletedPayload) error {
	return a.store.DeleteDaggerheartCountdown(ctx, evt.CampaignID, payload.CountdownID)
}

func (a *Adapter) handleAdversaryCreated(ctx context.Context, evt event.Event, payload AdversaryCreatedPayload) error {
	if err := projection.ValidateAdversaryStats(payload.HP, payload.HPMax, payload.Stress, payload.StressMax, payload.Evasion, payload.Major, payload.Severe, payload.Armor); err != nil {
		return err
	}
	createdAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{
		CampaignID:  evt.CampaignID,
		AdversaryID: strings.TrimSpace(payload.AdversaryID),
		Name:        strings.TrimSpace(payload.Name),
		Kind:        strings.TrimSpace(payload.Kind),
		SessionID:   strings.TrimSpace(payload.SessionID),
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
	adversaryID := strings.TrimSpace(payload.AdversaryID)
	if err := projection.ValidateAdversaryStats(payload.HP, payload.HPMax, payload.Stress, payload.StressMax, payload.Evasion, payload.Major, payload.Severe, payload.Armor); err != nil {
		return err
	}
	current, err := a.store.GetDaggerheartAdversary(ctx, evt.CampaignID, adversaryID)
	if err != nil {
		return err
	}
	updatedAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{
		CampaignID:  evt.CampaignID,
		AdversaryID: adversaryID,
		Name:        strings.TrimSpace(payload.Name),
		Kind:        strings.TrimSpace(payload.Kind),
		SessionID:   strings.TrimSpace(payload.SessionID),
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
	adversaryID := strings.TrimSpace(payload.AdversaryID)
	// State consistency: merge payload with current projection state.
	current, err := a.store.GetDaggerheartAdversary(ctx, evt.CampaignID, adversaryID)
	if err != nil {
		return err
	}
	next, err := projection.ApplyAdversaryDamagePatch(current, payload.HpAfter, payload.ArmorAfter)
	if err != nil {
		return err
	}
	next.CampaignID = evt.CampaignID
	next.AdversaryID = adversaryID
	updatedAt := evt.Timestamp.UTC()
	next.UpdatedAt = updatedAt
	return a.store.PutDaggerheartAdversary(ctx, next)
}

func (a *Adapter) handleAdversaryDeleted(ctx context.Context, evt event.Event, payload AdversaryDeletedPayload) error {
	return a.store.DeleteDaggerheartAdversary(ctx, evt.CampaignID, strings.TrimSpace(payload.AdversaryID))
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
