package daggerheart

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	if err := a.store.PutDaggerheartSnapshot(ctx, storage.DaggerheartSnapshot{
		CampaignID:            evt.CampaignID,
		GMFear:                payload.GMFearAfter,
		ConsecutiveShortRests: payload.ShortRestsAfter,
	}); err != nil {
		return fmt.Errorf("put daggerheart snapshot: %w", err)
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
	state, err := a.store.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get daggerheart character state: %w", err)
	}

	domainState, err := a.characterStateFromStorage(ctx, state)
	if err != nil {
		return err
	}
	removed := 0
	if clearShortRest {
		removed += domainState.ClearTemporaryArmorByDuration("short_rest")
	}
	if clearLongRest {
		removed += domainState.ClearTemporaryArmorByDuration("long_rest")
	}
	if removed == 0 {
		return nil
	}

	domainState.SetArmor(domainState.ResourceCap(ResourceArmor))
	state = storageDaggerheartCharacterStateFromDomain(&domainState)
	if err := a.store.PutDaggerheartCharacterState(ctx, state); err != nil {
		return fmt.Errorf("put daggerheart character state: %w", err)
	}
	return nil
}

func (a *Adapter) handleDowntimeMoveApplied(ctx context.Context, evt event.Event, payload DowntimeMoveAppliedPayload) error {
	if strings.TrimSpace(payload.Move) == "repair_all_armor" {
		state, err := a.store.GetDaggerheartCharacterState(ctx, evt.CampaignID, payload.CharacterID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, nil, payload.HopeAfter, nil, payload.StressAfter, payload.ArmorAfter, nil)
			}
			return fmt.Errorf("get daggerheart character state: %w", err)
		}
		domainState, err := a.characterStateFromStorage(ctx, state)
		if err != nil {
			return err
		}
		removed := domainState.ClearTemporaryArmorByDuration("short_rest")
		if removed > 0 {
			storageState := storageDaggerheartCharacterStateFromDomain(&domainState)
			if payload.ArmorAfter == nil {
				payload.ArmorAfter = &storageState.Armor
			}
			if err := a.store.PutDaggerheartCharacterState(ctx, storageState); err != nil {
				return fmt.Errorf("put daggerheart character state: %w", err)
			}
		}
		if payload.ArmorAfter == nil {
			// Ensure repair_all_armor always re-hydrates armor from source state.
			payload.ArmorAfter = &state.Armor
		}
	}
	return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, nil, payload.HopeAfter, nil, payload.StressAfter, payload.ArmorAfter, nil)
}

func (a *Adapter) handleCharacterTemporaryArmorApplied(ctx context.Context, evt event.Event, payload CharacterTemporaryArmorAppliedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID)

	state, err := a.store.GetDaggerheartCharacterState(ctx, evt.CampaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			state = storage.DaggerheartCharacterState{CampaignID: evt.CampaignID, CharacterID: characterID}
		} else {
			return fmt.Errorf("get daggerheart character state: %w", err)
		}
	}

	domainState, err := a.characterStateFromStorage(ctx, state)
	if err != nil {
		return err
	}
	domainState.ApplyTemporaryArmor(TemporaryArmorBucket{
		Source:   strings.TrimSpace(payload.Source),
		Duration: strings.TrimSpace(payload.Duration),
		SourceID: strings.TrimSpace(payload.SourceID),
		Amount:   payload.Amount,
	})
	domainState.LifeState = strings.TrimSpace(domainState.LifeState)
	if domainState.LifeState == "" {
		domainState.LifeState = LifeStateAlive
	}
	state = storageDaggerheartCharacterStateFromDomain(&domainState)

	return a.store.PutDaggerheartCharacterState(ctx, state)
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
	shortRests := 0
	current, err := a.store.GetDaggerheartSnapshot(ctx, evt.CampaignID)
	if err == nil {
		shortRests = current.ConsecutiveShortRests
	}
	return a.store.PutDaggerheartSnapshot(ctx, storage.DaggerheartSnapshot{
		CampaignID:            evt.CampaignID,
		GMFear:                payload.After,
		ConsecutiveShortRests: shortRests,
	})
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
	if payload.Before != countdown.Current {
		return fmt.Errorf("countdown before mismatch")
	}
	if payload.After < 0 || payload.After > countdown.Max {
		return fmt.Errorf("countdown after must be in range 0..%d", countdown.Max)
	}
	countdown.Current = payload.After
	return a.store.PutDaggerheartCountdown(ctx, countdown)
}

func (a *Adapter) handleCountdownDeleted(ctx context.Context, evt event.Event, payload CountdownDeletedPayload) error {
	return a.store.DeleteDaggerheartCountdown(ctx, evt.CampaignID, payload.CountdownID)
}

func (a *Adapter) handleAdversaryCreated(ctx context.Context, evt event.Event, payload AdversaryCreatedPayload) error {
	if err := validateAdversaryStats(payload.HP, payload.HPMax, payload.Stress, payload.StressMax, payload.Evasion, payload.Major, payload.Severe, payload.Armor); err != nil {
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
	if err := validateAdversaryStats(payload.HP, payload.HPMax, payload.Stress, payload.StressMax, payload.Evasion, payload.Major, payload.Severe, payload.Armor); err != nil {
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
	hp := current.HP
	armor := current.Armor
	if payload.HpAfter != nil {
		hp = *payload.HpAfter
	}
	if payload.ArmorAfter != nil {
		armor = *payload.ArmorAfter
	}
	if err := validateAdversaryStats(hp, current.HPMax, current.Stress, current.StressMax, current.Evasion, current.Major, current.Severe, armor); err != nil {
		return err
	}
	updatedAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{
		CampaignID:  evt.CampaignID,
		AdversaryID: adversaryID,
		Name:        current.Name,
		Kind:        current.Kind,
		SessionID:   current.SessionID,
		Notes:       current.Notes,
		HP:          hp,
		HPMax:       current.HPMax,
		Stress:      current.Stress,
		StressMax:   current.StressMax,
		Evasion:     current.Evasion,
		Major:       current.Major,
		Severe:      current.Severe,
		Armor:       armor,
		Conditions:  current.Conditions,
		CreatedAt:   current.CreatedAt,
		UpdatedAt:   updatedAt,
	})
}

func (a *Adapter) handleAdversaryDeleted(ctx context.Context, evt event.Event, payload AdversaryDeletedPayload) error {
	return a.store.DeleteDaggerheartAdversary(ctx, evt.CampaignID, strings.TrimSpace(payload.AdversaryID))
}

func validateAdversaryStats(hp, hpMax, stress, stressMax, evasion, major, severe, armor int) error {
	if hpMax <= 0 {
		return fmt.Errorf("hp_max must be positive")
	}
	if hp < 0 || hp > hpMax {
		return fmt.Errorf("hp must be in range 0..%d", hpMax)
	}
	if stressMax < 0 {
		return fmt.Errorf("stress_max must be non-negative")
	}
	if stress < 0 || stress > stressMax {
		return fmt.Errorf("stress must be in range 0..%d", stressMax)
	}
	if evasion < 0 {
		return fmt.Errorf("evasion must be non-negative")
	}
	if major < 0 || severe < 0 {
		return fmt.Errorf("thresholds must be non-negative")
	}
	if severe < major {
		return fmt.Errorf("severe_threshold must be >= major_threshold")
	}
	if armor < 0 {
		return fmt.Errorf("armor must be non-negative")
	}
	return nil
}

func daggerheartCharacterStateFromStorage(state storage.DaggerheartCharacterState, armorMax int) CharacterState {
	domainState := NewCharacterState(CharacterStateConfig{
		CampaignID:  state.CampaignID,
		CharacterID: state.CharacterID,
		HP:          state.Hp,
		HPMax:       HPMaxCap,
		Hope:        state.Hope,
		HopeMax:     state.HopeMax,
		Stress:      state.Stress,
		StressMax:   StressMaxCap,
		Armor:       state.Armor,
		ArmorMax:    armorMax,
		LifeState:   state.LifeState,
	})
	domainState.Conditions = append([]string(nil), state.Conditions...)
	domainState.ArmorBonus = make([]TemporaryArmorBucket, 0, len(state.TemporaryArmor))
	for _, bucket := range state.TemporaryArmor {
		domainState.ArmorBonus = append(domainState.ArmorBonus, TemporaryArmorBucket{
			Source:   strings.TrimSpace(bucket.Source),
			Duration: strings.TrimSpace(bucket.Duration),
			SourceID: strings.TrimSpace(bucket.SourceID),
			Amount:   bucket.Amount,
		})
	}
	if strings.TrimSpace(domainState.LifeState) == "" {
		domainState.LifeState = LifeStateAlive
	}
	return *domainState
}

func (a *Adapter) characterStateFromStorage(ctx context.Context, state storage.DaggerheartCharacterState) (CharacterState, error) {
	armorMax := state.Armor
	if strings.TrimSpace(state.CampaignID) == "" || strings.TrimSpace(state.CharacterID) == "" {
		return daggerheartCharacterStateFromStorage(state, armorMax), nil
	}

	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, state.CampaignID, state.CharacterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return daggerheartCharacterStateFromStorage(state, armorMax), nil
		}
		return CharacterState{}, fmt.Errorf("get daggerheart character profile: %w", err)
	}
	armorMax = profile.ArmorMax
	return daggerheartCharacterStateFromStorage(state, armorMax), nil
}

func storageDaggerheartCharacterStateFromDomain(state *CharacterState) storage.DaggerheartCharacterState {
	if state == nil {
		return storage.DaggerheartCharacterState{}
	}
	temporaryArmor := make([]storage.DaggerheartTemporaryArmor, 0, len(state.ArmorBonus))
	for _, bucket := range state.ArmorBonus {
		temporaryArmor = append(temporaryArmor, storage.DaggerheartTemporaryArmor{
			Source:   strings.TrimSpace(bucket.Source),
			Duration: strings.TrimSpace(bucket.Duration),
			SourceID: strings.TrimSpace(bucket.SourceID),
			Amount:   bucket.Amount,
		})
	}
	return storage.DaggerheartCharacterState{
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

func (a *Adapter) applyStatePatch(ctx context.Context, campaignID, characterID string, hpAfter, hopeAfter, hopeMaxAfter, stressAfter, armorAfter *int, lifeStateAfter *string) error {
	state, err := a.store.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			state = storage.DaggerheartCharacterState{CampaignID: campaignID, CharacterID: characterID}
		} else {
			return fmt.Errorf("get daggerheart character state: %w", err)
		}
	}
	if hpAfter != nil {
		state.Hp = *hpAfter
	}
	if hopeAfter != nil {
		state.Hope = *hopeAfter
	}
	if hopeMaxAfter != nil {
		state.HopeMax = *hopeMaxAfter
	}
	if stressAfter != nil {
		state.Stress = *stressAfter
	}
	if armorAfter != nil {
		state.Armor = *armorAfter
	}
	if lifeStateAfter != nil {
		state.LifeState = *lifeStateAfter
	}
	if state.Hp < HPMin || state.Hp > HPMaxCap {
		return fmt.Errorf("character_state hp must be in range %d..%d", HPMin, HPMaxCap)
	}
	if state.HopeMax == 0 {
		state.HopeMax = HopeMax
	}
	if state.HopeMax < HopeMin || state.HopeMax > HopeMax {
		return fmt.Errorf("character_state hope_max must be in range %d..%d", HopeMin, HopeMax)
	}
	if state.Hope < HopeMin || state.Hope > state.HopeMax {
		return fmt.Errorf("character_state hope must be in range %d..%d", HopeMin, state.HopeMax)
	}
	if state.Stress < StressMin || state.Stress > StressMaxCap {
		return fmt.Errorf("character_state stress must be in range %d..%d", StressMin, StressMaxCap)
	}
	if state.Armor < ArmorMin || state.Armor > ArmorMaxCap {
		return fmt.Errorf("character_state armor must be in range %d..%d", ArmorMin, ArmorMaxCap)
	}
	if strings.TrimSpace(state.LifeState) == "" {
		state.LifeState = LifeStateAlive
	} else if _, err := NormalizeLifeState(state.LifeState); err != nil {
		return fmt.Errorf("character_state life_state: %w", err)
	}
	if state.Hope > state.HopeMax {
		state.Hope = state.HopeMax
	}
	return a.store.PutDaggerheartCharacterState(ctx, state)
}

func (a *Adapter) applyConditionPatch(ctx context.Context, campaignID, characterID string, conditions []string) error {
	state, err := a.store.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			state = storage.DaggerheartCharacterState{CampaignID: campaignID, CharacterID: characterID}
		} else {
			return fmt.Errorf("get daggerheart character state: %w", err)
		}
	}
	state.Conditions = conditions
	if err := a.store.PutDaggerheartCharacterState(ctx, state); err != nil {
		return fmt.Errorf("put daggerheart character state: %w", err)
	}
	return nil
}

func (a *Adapter) applyAdversaryConditionPatch(ctx context.Context, campaignID, adversaryID string, conditions []string) error {
	adversary, err := a.store.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return fmt.Errorf("get daggerheart adversary: %w", err)
	}
	adversary.Conditions = conditions
	if err := a.store.PutDaggerheartAdversary(ctx, adversary); err != nil {
		return fmt.Errorf("put daggerheart adversary: %w", err)
	}
	return nil
}
