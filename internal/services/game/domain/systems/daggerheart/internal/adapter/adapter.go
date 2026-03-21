package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// LevelUpApplier applies level-up progression to a character profile.
type LevelUpApplier func(*snapstate.CharacterProfile, payload.LevelUpAppliedPayload)

// Adapter applies Daggerheart-specific events to system projections.
type Adapter struct {
	store        projectionstore.Store
	Router       *module.AdapterRouter
	applyLevelUp LevelUpApplier
}

// NewAdapter creates a Daggerheart adapter with all handlers registered.
func NewAdapter(store projectionstore.Store, applyLevelUp LevelUpApplier) *Adapter {
	a := &Adapter{store: store, applyLevelUp: applyLevelUp}
	a.Router = a.buildRouter()
	return a
}

// ID returns the Daggerheart system identifier.
func (a *Adapter) ID() string {
	return snapstate.SystemID
}

// Version returns the Daggerheart system version.
func (a *Adapter) Version() string {
	return snapstate.SystemVersion
}

// HandledTypes returns the event types this adapter's Apply handles.
func (a *Adapter) HandledTypes() []event.Type {
	return a.Router.HandledTypes()
}

// Apply applies a system-specific event to Daggerheart projections.
func (a *Adapter) Apply(ctx context.Context, evt event.Event) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("daggerheart store is not configured")
	}
	return a.Router.Apply(ctx, evt)
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

func (a *Adapter) buildRouter() *module.AdapterRouter {
	r := module.NewAdapterRouter()
	module.HandleAdapter(r, payload.EventTypeCharacterProfileReplaced, a.HandleCharacterProfileReplaced)
	module.HandleAdapter(r, payload.EventTypeCharacterProfileDeleted, a.HandleCharacterProfileDeleted)
	module.HandleAdapter(r, payload.EventTypeDamageApplied, a.HandleDamageApplied)
	module.HandleAdapter(r, payload.EventTypeRestTaken, a.HandleRestTaken)
	module.HandleAdapter(r, payload.EventTypeCharacterTemporaryArmorApplied, a.HandleCharacterTemporaryArmorApplied)
	module.HandleAdapter(r, payload.EventTypeDowntimeMoveApplied, a.HandleDowntimeMoveApplied)
	module.HandleAdapter(r, payload.EventTypeLoadoutSwapped, a.HandleLoadoutSwapped)
	module.HandleAdapter(r, payload.EventTypeCharacterStatePatched, a.HandleCharacterStatePatched)
	module.HandleAdapter(r, payload.EventTypeBeastformTransformed, a.HandleBeastformTransformed)
	module.HandleAdapter(r, payload.EventTypeBeastformDropped, a.HandleBeastformDropped)
	module.HandleAdapter(r, payload.EventTypeCompanionExperienceBegun, a.HandleCompanionExperienceBegun)
	module.HandleAdapter(r, payload.EventTypeCompanionReturned, a.HandleCompanionReturned)
	module.HandleAdapter(r, payload.EventTypeConditionChanged, a.HandleConditionChanged)
	module.HandleAdapter(r, payload.EventTypeAdversaryConditionChanged, a.HandleAdversaryConditionChanged)
	module.HandleAdapter(r, payload.EventTypeGMFearChanged, a.HandleGMFearChanged)
	module.HandleAdapter(r, payload.EventTypeCountdownCreated, a.HandleCountdownCreated)
	module.HandleAdapter(r, payload.EventTypeCountdownUpdated, a.HandleCountdownUpdated)
	module.HandleAdapter(r, payload.EventTypeCountdownDeleted, a.HandleCountdownDeleted)
	module.HandleAdapter(r, payload.EventTypeAdversaryCreated, a.HandleAdversaryCreated)
	module.HandleAdapter(r, payload.EventTypeAdversaryDamageApplied, a.HandleAdversaryDamageApplied)
	module.HandleAdapter(r, payload.EventTypeAdversaryUpdated, a.HandleAdversaryUpdated)
	module.HandleAdapter(r, payload.EventTypeAdversaryDeleted, a.HandleAdversaryDeleted)
	module.HandleAdapter(r, payload.EventTypeEnvironmentEntityCreated, a.HandleEnvironmentEntityCreated)
	module.HandleAdapter(r, payload.EventTypeEnvironmentEntityUpdated, a.HandleEnvironmentEntityUpdated)
	module.HandleAdapter(r, payload.EventTypeEnvironmentEntityDeleted, a.HandleEnvironmentEntityDeleted)
	module.HandleAdapter(r, payload.EventTypeLevelUpApplied, a.HandleLevelUpApplied)
	module.HandleAdapter(r, payload.EventTypeGoldUpdated, a.HandleGoldUpdated)
	module.HandleAdapter(r, payload.EventTypeDomainCardAcquired, a.HandleDomainCardAcquired)
	module.HandleAdapter(r, payload.EventTypeEquipmentSwapped, a.HandleEquipmentSwapped)
	module.HandleAdapter(r, payload.EventTypeConsumableUsed, a.HandleConsumableUsed)
	module.HandleAdapter(r, payload.EventTypeConsumableAcquired, a.HandleConsumableAcquired)
	return r
}

func (a *Adapter) HandleDamageApplied(ctx context.Context, evt event.Event, p payload.DamageAppliedPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), p.Hp, nil, nil, p.Stress, p.Armor, nil, nil, nil, nil, nil)
}

func (a *Adapter) HandleRestTaken(ctx context.Context, evt event.Event, p payload.RestTakenPayload) error {
	if err := a.PutSnapshot(ctx, string(evt.CampaignID), p.GMFear, p.ShortRests); err != nil {
		return err
	}
	for _, participantID := range p.Participants {
		characterID := strings.TrimSpace(participantID.String())
		if p.RefreshRest || p.RefreshLongRest {
			if err := a.ClearRestTemporaryArmor(ctx, string(evt.CampaignID), characterID, p.RefreshRest, p.RefreshLongRest); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Adapter) HandleDowntimeMoveApplied(ctx context.Context, evt event.Event, p payload.DowntimeMoveAppliedPayload) error {
	characterID := strings.TrimSpace(p.TargetCharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(p.ActorCharacterID.String())
	}
	if characterID == "" {
		return nil
	}
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), characterID, p.HP, p.Hope, nil, p.Stress, p.Armor, nil, nil, nil, nil, nil)
}

func (a *Adapter) HandleCharacterTemporaryArmorApplied(ctx context.Context, evt event.Event, p payload.CharacterTemporaryArmorAppliedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	state, err := a.GetCharacterStateOrDefault(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.CharacterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState, err := projection.ApplyTemporaryArmor(state, armorMax, p.Source, p.Duration, p.SourceID, p.Amount)
	if err != nil {
		return err
	}
	return a.PutCharacterState(ctx, nextState)
}

func (a *Adapter) HandleLoadoutSwapped(ctx context.Context, evt event.Event, p payload.LoadoutSwappedPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), nil, nil, nil, p.Stress, nil, nil, nil, nil, nil, nil)
}

func (a *Adapter) HandleCharacterStatePatched(ctx context.Context, evt event.Event, p payload.CharacterStatePatchedPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), p.HP, p.Hope, p.HopeMax, p.Stress, p.Armor, p.LifeState, p.ClassState, p.SubclassState, nil, p.ImpenetrableUsedThisShortRest)
}

func (a *Adapter) HandleBeastformTransformed(ctx context.Context, evt event.Event, p payload.BeastformTransformedPayload) error {
	state, err := a.GetCharacterStateOrDefault(ctx, string(evt.CampaignID), p.CharacterID.String())
	if err != nil {
		return err
	}
	nextClassState := snapstate.WithActiveBeastform(ClassStateFromProjection(state.ClassState), p.ActiveBeastform)
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), nil, p.Hope, nil, p.Stress, nil, nil, &nextClassState, nil, nil, nil)
}

func (a *Adapter) HandleBeastformDropped(ctx context.Context, evt event.Event, p payload.BeastformDroppedPayload) error {
	state, err := a.GetCharacterStateOrDefault(ctx, string(evt.CampaignID), p.CharacterID.String())
	if err != nil {
		return err
	}
	nextClassState := snapstate.WithActiveBeastform(ClassStateFromProjection(state.ClassState), nil)
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), nil, nil, nil, nil, nil, nil, &nextClassState, nil, nil, nil)
}

func (a *Adapter) HandleCompanionExperienceBegun(ctx context.Context, evt event.Event, p payload.CompanionExperienceBegunPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), nil, nil, nil, nil, nil, nil, nil, nil, p.CompanionState, nil)
}

func (a *Adapter) HandleCompanionReturned(ctx context.Context, evt event.Event, p payload.CompanionReturnedPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), nil, nil, nil, p.Stress, nil, nil, nil, nil, p.CompanionState, nil)
}

func (a *Adapter) HandleConditionChanged(ctx context.Context, evt event.Event, p payload.ConditionChangedPayload) error {
	if p.RollSeq != nil && *p.RollSeq == 0 {
		return fmt.Errorf("condition_changed roll_seq must be positive")
	}
	normalizedAfter, err := rules.NormalizeConditionStates(p.Conditions)
	if err != nil {
		return fmt.Errorf("condition_changed conditions_after: %w", err)
	}
	return a.ApplyConditionPatch(ctx, string(evt.CampaignID), p.CharacterID.String(), normalizedAfter)
}

func (a *Adapter) HandleAdversaryConditionChanged(ctx context.Context, evt event.Event, p payload.AdversaryConditionChangedPayload) error {
	if p.RollSeq != nil && *p.RollSeq == 0 {
		return fmt.Errorf("adversary_condition_changed roll_seq must be positive")
	}
	normalizedAfter, err := rules.NormalizeConditionStates(p.Conditions)
	if err != nil {
		return fmt.Errorf("adversary_condition_changed conditions_after: %w", err)
	}
	return a.ApplyAdversaryConditionPatch(ctx, string(evt.CampaignID), p.AdversaryID.String(), normalizedAfter)
}

func (a *Adapter) HandleGMFearChanged(ctx context.Context, evt event.Event, p payload.GMFearChangedPayload) error {
	if p.Value < snapstate.GMFearMin || p.Value > snapstate.GMFearMax {
		return fmt.Errorf("gm_fear_changed value must be in range %d..%d", snapstate.GMFearMin, snapstate.GMFearMax)
	}
	shortRests := a.SnapshotShortRests(ctx, string(evt.CampaignID))
	return a.PutSnapshot(ctx, string(evt.CampaignID), p.Value, shortRests)
}

func (a *Adapter) HandleCountdownCreated(ctx context.Context, evt event.Event, p payload.CountdownCreatedPayload) error {
	return a.store.PutDaggerheartCountdown(ctx, projectionstore.DaggerheartCountdown{
		CampaignID:        string(evt.CampaignID),
		CountdownID:       p.CountdownID.String(),
		Name:              p.Name,
		Kind:              p.Kind,
		Current:           p.Current,
		Max:               p.Max,
		Direction:         p.Direction,
		Looping:           p.Looping,
		Variant:           p.Variant,
		TriggerEventType:  p.TriggerEventType,
		LinkedCountdownID: p.LinkedCountdownID.String(),
	})
}

func (a *Adapter) HandleCountdownUpdated(ctx context.Context, evt event.Event, p payload.CountdownUpdatedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	nextCountdown, err := projection.ApplyCountdownUpdate(countdown, p.Value)
	if err != nil {
		return err
	}
	return a.store.PutDaggerheartCountdown(ctx, nextCountdown)
}

func (a *Adapter) HandleCountdownDeleted(ctx context.Context, evt event.Event, p payload.CountdownDeletedPayload) error {
	return a.store.DeleteDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
}

func (a *Adapter) HandleAdversaryCreated(ctx context.Context, evt event.Event, p payload.AdversaryCreatedPayload) error {
	if err := projection.ValidateAdversaryStats(p.HP, p.HPMax, p.Stress, p.StressMax, p.Evasion, p.Major, p.Severe, p.Armor); err != nil {
		return err
	}
	createdAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, projectionstore.DaggerheartAdversary{
		CampaignID:        string(evt.CampaignID),
		AdversaryID:       strings.TrimSpace(p.AdversaryID.String()),
		AdversaryEntryID:  strings.TrimSpace(p.AdversaryEntryID),
		Name:              strings.TrimSpace(p.Name),
		Kind:              strings.TrimSpace(p.Kind),
		SessionID:         strings.TrimSpace(p.SessionID.String()),
		SceneID:           strings.TrimSpace(p.SceneID.String()),
		Notes:             strings.TrimSpace(p.Notes),
		HP:                p.HP,
		HPMax:             p.HPMax,
		Stress:            p.Stress,
		StressMax:         p.StressMax,
		Evasion:           p.Evasion,
		Major:             p.Major,
		Severe:            p.Severe,
		Armor:             p.Armor,
		FeatureStates:     ToProjectionAdversaryFeatureStates(p.FeatureStates),
		PendingExperience: ToProjectionAdversaryPendingExperience(p.PendingExperience),
		SpotlightGateID:   strings.TrimSpace(p.SpotlightGateID.String()),
		SpotlightCount:    p.SpotlightCount,
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
	})
}

func (a *Adapter) HandleAdversaryUpdated(ctx context.Context, evt event.Event, p payload.AdversaryUpdatedPayload) error {
	adversaryID := strings.TrimSpace(p.AdversaryID.String())
	if err := projection.ValidateAdversaryStats(p.HP, p.HPMax, p.Stress, p.StressMax, p.Evasion, p.Major, p.Severe, p.Armor); err != nil {
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
		AdversaryEntryID:  strings.TrimSpace(p.AdversaryEntryID),
		Name:              strings.TrimSpace(p.Name),
		Kind:              strings.TrimSpace(p.Kind),
		SessionID:         strings.TrimSpace(p.SessionID.String()),
		SceneID:           strings.TrimSpace(p.SceneID.String()),
		Notes:             strings.TrimSpace(p.Notes),
		HP:                p.HP,
		HPMax:             p.HPMax,
		Stress:            p.Stress,
		StressMax:         p.StressMax,
		Evasion:           p.Evasion,
		Major:             p.Major,
		Severe:            p.Severe,
		Armor:             p.Armor,
		Conditions:        current.Conditions,
		FeatureStates:     ToProjectionAdversaryFeatureStates(p.FeatureStates),
		PendingExperience: ToProjectionAdversaryPendingExperience(p.PendingExperience),
		SpotlightGateID:   strings.TrimSpace(p.SpotlightGateID.String()),
		SpotlightCount:    p.SpotlightCount,
		CreatedAt:         current.CreatedAt,
		UpdatedAt:         updatedAt,
	})
}

func ToProjectionAdversaryFeatureStates(in []rules.AdversaryFeatureState) []projectionstore.DaggerheartAdversaryFeatureState {
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

func ToProjectionAdversaryPendingExperience(in *rules.AdversaryPendingExperience) *projectionstore.DaggerheartAdversaryPendingExperience {
	if in == nil {
		return nil
	}
	return &projectionstore.DaggerheartAdversaryPendingExperience{
		Name:     strings.TrimSpace(in.Name),
		Modifier: in.Modifier,
	}
}

func (a *Adapter) HandleAdversaryDamageApplied(ctx context.Context, evt event.Event, p payload.AdversaryDamageAppliedPayload) error {
	adversaryID := strings.TrimSpace(p.AdversaryID.String())
	current, err := a.store.GetDaggerheartAdversary(ctx, string(evt.CampaignID), adversaryID)
	if err != nil {
		return err
	}
	next, err := projection.ApplyAdversaryDamagePatch(current, p.Hp, p.Armor)
	if err != nil {
		return err
	}
	next.CampaignID = string(evt.CampaignID)
	next.AdversaryID = adversaryID
	next.UpdatedAt = evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, next)
}

func (a *Adapter) HandleAdversaryDeleted(ctx context.Context, evt event.Event, p payload.AdversaryDeletedPayload) error {
	return a.store.DeleteDaggerheartAdversary(ctx, string(evt.CampaignID), strings.TrimSpace(p.AdversaryID.String()))
}

func (a *Adapter) HandleEnvironmentEntityCreated(ctx context.Context, evt event.Event, p payload.EnvironmentEntityCreatedPayload) error {
	createdAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartEnvironmentEntity(ctx, projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          string(evt.CampaignID),
		EnvironmentEntityID: strings.TrimSpace(p.EnvironmentEntityID.String()),
		EnvironmentID:       strings.TrimSpace(p.EnvironmentID),
		Name:                strings.TrimSpace(p.Name),
		Type:                strings.TrimSpace(p.Type),
		Tier:                p.Tier,
		Difficulty:          p.Difficulty,
		SessionID:           strings.TrimSpace(p.SessionID.String()),
		SceneID:             strings.TrimSpace(p.SceneID.String()),
		Notes:               strings.TrimSpace(p.Notes),
		CreatedAt:           createdAt,
		UpdatedAt:           createdAt,
	})
}

func (a *Adapter) HandleEnvironmentEntityUpdated(ctx context.Context, evt event.Event, p payload.EnvironmentEntityUpdatedPayload) error {
	environmentEntityID := strings.TrimSpace(p.EnvironmentEntityID.String())
	current, err := a.store.GetDaggerheartEnvironmentEntity(ctx, string(evt.CampaignID), environmentEntityID)
	if err != nil {
		return err
	}
	return a.store.PutDaggerheartEnvironmentEntity(ctx, projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          string(evt.CampaignID),
		EnvironmentEntityID: environmentEntityID,
		EnvironmentID:       strings.TrimSpace(p.EnvironmentID),
		Name:                strings.TrimSpace(p.Name),
		Type:                strings.TrimSpace(p.Type),
		Tier:                p.Tier,
		Difficulty:          p.Difficulty,
		SessionID:           strings.TrimSpace(p.SessionID.String()),
		SceneID:             strings.TrimSpace(p.SceneID.String()),
		Notes:               strings.TrimSpace(p.Notes),
		CreatedAt:           current.CreatedAt,
		UpdatedAt:           evt.Timestamp.UTC(),
	})
}

func (a *Adapter) HandleEnvironmentEntityDeleted(ctx context.Context, evt event.Event, p payload.EnvironmentEntityDeletedPayload) error {
	return a.store.DeleteDaggerheartEnvironmentEntity(ctx, string(evt.CampaignID), strings.TrimSpace(p.EnvironmentEntityID.String()))
}

func (a *Adapter) HandleLevelUpApplied(ctx context.Context, evt event.Event, p payload.LevelUpAppliedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	storedProfile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for level-up: %w", err)
		}
		return nil
	}
	profile := snapstate.CharacterProfileFromStorage(storedProfile)
	a.applyLevelUp(&profile, p)
	return a.store.PutDaggerheartCharacterProfile(ctx, profile.ToStorage(string(evt.CampaignID), characterID))
}

func (a *Adapter) HandleGoldUpdated(ctx context.Context, evt event.Event, p payload.GoldUpdatedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for gold update: %w", err)
		}
		return nil
	}
	profile.GoldHandfuls = p.Handfuls
	profile.GoldBags = p.Bags
	profile.GoldChests = p.Chests
	return a.store.PutDaggerheartCharacterProfile(ctx, profile)
}

func (a *Adapter) HandleDomainCardAcquired(ctx context.Context, evt event.Event, p payload.DomainCardAcquiredPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for domain card acquire: %w", err)
		}
		return nil
	}
	profile.DomainCardIDs = snapstate.AppendUnique(profile.DomainCardIDs, strings.TrimSpace(p.CardID))
	return a.store.PutDaggerheartCharacterProfile(ctx, profile)
}

func (a *Adapter) HandleEquipmentSwapped(ctx context.Context, evt event.Event, p payload.EquipmentSwappedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	if characterID == "" {
		return nil
	}
	if strings.TrimSpace(p.ItemType) == "armor" {
		profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return fmt.Errorf("get daggerheart character profile for equipment swap: %w", err)
			}
		} else {
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
			if err := a.store.PutDaggerheartCharacterProfile(ctx, profile); err != nil {
				return fmt.Errorf("put daggerheart character profile for equipment swap: %w", err)
			}
		}
		if p.ArmorAfter != nil {
			if err := a.ApplyStatePatch(ctx, string(evt.CampaignID), characterID, nil, nil, nil, nil, p.ArmorAfter, nil, nil, nil, nil, nil); err != nil {
				return err
			}
		}
	}
	if p.StressCost > 0 {
		state, err := a.GetCharacterStateOrDefault(ctx, string(evt.CampaignID), characterID)
		if err != nil {
			return err
		}
		stressAfter := state.Stress + p.StressCost
		if err := a.ApplyStatePatch(ctx, string(evt.CampaignID), characterID, nil, nil, nil, &stressAfter, nil, nil, nil, nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) HandleConsumableUsed(_ context.Context, _ event.Event, _ payload.ConsumableUsedPayload) error {
	return nil
}

func (a *Adapter) HandleConsumableAcquired(_ context.Context, _ event.Event, _ payload.ConsumableAcquiredPayload) error {
	return nil
}

func (a *Adapter) ApplyStatePatch(ctx context.Context, campaignID, characterID string, hpAfter, hopeAfter, hopeMaxAfter, stressAfter, armorAfter *int, lifeStateAfter *string, classStateAfter *snapstate.CharacterClassState, subclassStateAfter *snapstate.CharacterSubclassState, companionStateAfter *snapstate.CharacterCompanionState, impenetrableUsedThisShortRestAfter *bool) error {
	state, err := a.GetCharacterStateOrDefault(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.CharacterArmorMax(ctx, state)
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
		ClassStateToProjection(classStateAfter),
		SubclassStateToProjection(subclassStateAfter),
		CompanionStateToProjection(companionStateAfter),
		impenetrableUsedThisShortRestAfter,
	)
	if err != nil {
		return err
	}
	return a.PutCharacterState(ctx, nextState)
}

func (a *Adapter) ApplyConditionPatch(ctx context.Context, campaignID, characterID string, conditions []rules.ConditionState) error {
	state, err := a.GetCharacterStateOrDefault(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	armorMax, err := a.CharacterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState := projection.ApplyConditionPatch(state, armorMax, ConditionStatesToProjection(conditions))
	return a.PutCharacterState(ctx, nextState)
}

func (a *Adapter) ApplyAdversaryConditionPatch(ctx context.Context, campaignID, adversaryID string, conditions []rules.ConditionState) error {
	adversary, err := a.store.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return fmt.Errorf("get daggerheart adversary: %w", err)
	}
	next := projection.ApplyAdversaryConditionPatch(adversary, ConditionStatesToProjection(conditions))
	if err := a.store.PutDaggerheartAdversary(ctx, next); err != nil {
		return fmt.Errorf("put daggerheart adversary: %w", err)
	}
	return nil
}

func SubclassStateToProjection(value *snapstate.CharacterSubclassState) *projectionstore.DaggerheartSubclassState {
	normalized := snapstate.NormalizedSubclassStatePtr(value)
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

func ClassStateToProjection(value *snapstate.CharacterClassState) *projectionstore.DaggerheartClassState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &projectionstore.DaggerheartClassState{
		AttackBonusUntilRest:       normalized.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest: normalized.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest: normalized.DifficultyPenaltyUntilRest,
		FocusTargetID:              normalized.FocusTargetID,
		ActiveBeastform:            ActiveBeastformToProjection(normalized.ActiveBeastform),
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

func CompanionStateToProjection(value *snapstate.CharacterCompanionState) *projectionstore.DaggerheartCompanionState {
	normalized := snapstate.NormalizedCompanionStatePtr(value)
	if normalized == nil {
		return nil
	}
	return &projectionstore.DaggerheartCompanionState{
		Status:             normalized.Status,
		ActiveExperienceID: normalized.ActiveExperienceID,
	}
}

func ActiveBeastformToProjection(value *snapstate.CharacterActiveBeastformState) *projectionstore.DaggerheartActiveBeastformState {
	normalized := snapstate.NormalizedActiveBeastformPtr(value)
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

func ClassStateFromProjection(value projectionstore.DaggerheartClassState) snapstate.CharacterClassState {
	damageDice := []snapstate.CharacterDamageDie(nil)
	active := snapstate.NormalizedActiveBeastformPtr(nil)
	if value.ActiveBeastform != nil {
		damageDice = make([]snapstate.CharacterDamageDie, 0, len(value.ActiveBeastform.DamageDice))
		for _, die := range value.ActiveBeastform.DamageDice {
			damageDice = append(damageDice, snapstate.CharacterDamageDie{Count: die.Count, Sides: die.Sides})
		}
		active = &snapstate.CharacterActiveBeastformState{
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
	return snapstate.CharacterClassState{
		AttackBonusUntilRest:            value.AttackBonusUntilRest,
		EvasionBonusUntilHitOrRest:      value.EvasionBonusUntilHitOrRest,
		DifficultyPenaltyUntilRest:      value.DifficultyPenaltyUntilRest,
		FocusTargetID:                   value.FocusTargetID,
		ActiveBeastform:                 active,
		StrangePatternsNumber:           value.StrangePatternsNumber,
		RallyDice:                       append([]int(nil), value.RallyDice...),
		PrayerDice:                      append([]int(nil), value.PrayerDice...),
		ChannelRawPowerUsedThisLongRest: value.ChannelRawPowerUsedThisLongRest,
		Unstoppable: snapstate.CharacterUnstoppableState{
			Active:           value.Unstoppable.Active,
			CurrentValue:     value.Unstoppable.CurrentValue,
			DieSides:         value.Unstoppable.DieSides,
			UsedThisLongRest: value.Unstoppable.UsedThisLongRest,
		},
	}.Normalized()
}

func SubclassStateFromProjection(value *projectionstore.DaggerheartSubclassState) *snapstate.CharacterSubclassState {
	if value == nil {
		return nil
	}
	return snapstate.NormalizedSubclassStatePtr(&snapstate.CharacterSubclassState{
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

func CompanionStateFromProjection(value *projectionstore.DaggerheartCompanionState) *snapstate.CharacterCompanionState {
	if value == nil {
		return nil
	}
	return snapstate.NormalizedCompanionStatePtr(&snapstate.CharacterCompanionState{
		Status:             value.Status,
		ActiveExperienceID: value.ActiveExperienceID,
	})
}

func ConditionStatesToProjection(values []rules.ConditionState) []projectionstore.DaggerheartConditionState {
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
