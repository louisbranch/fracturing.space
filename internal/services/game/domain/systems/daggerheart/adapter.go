package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Adapter applies Daggerheart-specific events to system projections.
type Adapter struct {
	store storage.DaggerheartStore
}

// NewAdapter creates a Daggerheart adapter.
func NewAdapter(store storage.DaggerheartStore) *Adapter {
	return &Adapter{store: store}
}

// ID returns the Daggerheart system identifier.
func (a *Adapter) ID() commonv1.GameSystem {
	return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
}

// Version returns the Daggerheart system version.
func (a *Adapter) Version() string {
	return SystemVersion
}

// ApplyEvent applies a system-specific event to Daggerheart projections.
func (a *Adapter) ApplyEvent(ctx context.Context, evt event.Event) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("daggerheart store is not configured")
	}
	switch evt.Type {
	case EventTypeDamageApplied:
		return a.applyDamageApplied(ctx, evt)
	case EventTypeRestTaken:
		return a.applyRestTaken(ctx, evt)
	case EventTypeDowntimeMoveApplied:
		return a.applyDowntimeMoveApplied(ctx, evt)
	case EventTypeLoadoutSwapped:
		return a.applyLoadoutSwapped(ctx, evt)
	case EventTypeCharacterStatePatched:
		return a.applyCharacterStatePatched(ctx, evt)
	case EventTypeConditionChanged:
		return a.applyConditionChanged(ctx, evt)
	case EventTypeGMFearChanged:
		return a.applyGMFearChanged(ctx, evt)
	case EventTypeGMMoveApplied:
		return a.applyGMMoveApplied(ctx, evt)
	case EventTypeDeathMoveResolved:
		return a.applyDeathMoveResolved(ctx, evt)
	case EventTypeBlazeOfGloryResolved:
		return a.applyBlazeOfGloryResolved(ctx, evt)
	case EventTypeAttackResolved:
		return a.applyAttackResolved(ctx, evt)
	case EventTypeReactionResolved:
		return a.applyReactionResolved(ctx, evt)
	case EventTypeDamageRollResolved:
		return a.applyDamageRollResolved(ctx, evt)
	case EventTypeGroupActionResolved:
		return a.applyGroupActionResolved(ctx, evt)
	case EventTypeTagTeamResolved:
		return a.applyTagTeamResolved(ctx, evt)
	case EventTypeCountdownCreated:
		return a.applyCountdownCreated(ctx, evt)
	case EventTypeCountdownUpdated:
		return a.applyCountdownUpdated(ctx, evt)
	case EventTypeCountdownDeleted:
		return a.applyCountdownDeleted(ctx, evt)
	case EventTypeAdversaryRollResolved:
		return a.applyAdversaryRollResolved(ctx, evt)
	case EventTypeAdversaryAttackResolved:
		return a.applyAdversaryAttackResolved(ctx, evt)
	case EventTypeAdversaryCreated:
		return a.applyAdversaryCreated(ctx, evt)
	case EventTypeAdversaryUpdated:
		return a.applyAdversaryUpdated(ctx, evt)
	case EventTypeAdversaryDeleted:
		return a.applyAdversaryDeleted(ctx, evt)
	default:
		return nil
	}
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

func (a *Adapter) applyDamageApplied(ctx context.Context, evt event.Event) error {
	var payload DamageAppliedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.damage_applied payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	if payload.ArmorSpent < 0 || payload.ArmorSpent > ArmorMaxCap {
		return fmt.Errorf("damage_applied armor_spent must be in range 0..%d", ArmorMaxCap)
	}
	if payload.Marks < 0 || payload.Marks > 4 {
		return fmt.Errorf("damage_applied marks must be in range 0..4")
	}
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return fmt.Errorf("damage_applied roll_seq must be positive")
	}
	if severity := strings.TrimSpace(payload.Severity); severity != "" {
		switch severity {
		case "none", "minor", "major", "severe", "massive":
			// allowed
		default:
			return fmt.Errorf("damage_applied severity must be one of none, minor, major, severe, massive")
		}
	}
	for _, id := range payload.SourceCharacterIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("damage_applied source_character_ids must not contain empty values")
		}
	}
	return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, payload.HpAfter, nil, nil, nil, payload.ArmorAfter, nil)
}

func (a *Adapter) applyRestTaken(ctx context.Context, evt event.Event) error {
	var payload RestTakenPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.rest_taken payload: %w", err)
	}
	if payload.GMFearAfter < GMFearMin || payload.GMFearAfter > GMFearMax {
		return fmt.Errorf("rest_taken gm_fear_after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	if payload.ShortRestsAfter < 0 {
		return fmt.Errorf("rest_taken short_rests_after must be non-negative")
	}
	if err := a.store.PutDaggerheartSnapshot(ctx, storage.DaggerheartSnapshot{
		CampaignID:            evt.CampaignID,
		GMFear:                payload.GMFearAfter,
		ConsecutiveShortRests: payload.ShortRestsAfter,
	}); err != nil {
		return fmt.Errorf("put daggerheart snapshot: %w", err)
	}
	for _, patch := range payload.CharacterStates {
		if strings.TrimSpace(patch.CharacterID) == "" {
			return fmt.Errorf("character_id is required")
		}
		if err := a.applyStatePatch(ctx, evt.CampaignID, patch.CharacterID, nil, patch.HopeAfter, nil, patch.StressAfter, patch.ArmorAfter, nil); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) applyDowntimeMoveApplied(ctx context.Context, evt event.Event) error {
	var payload DowntimeMoveAppliedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.downtime_move_applied payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, nil, payload.HopeAfter, nil, payload.StressAfter, payload.ArmorAfter, nil)
}

func (a *Adapter) applyLoadoutSwapped(ctx context.Context, evt event.Event) error {
	var payload LoadoutSwappedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.loadout_swapped payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, nil, nil, nil, payload.StressAfter, nil, nil)
}

func (a *Adapter) applyCharacterStatePatched(ctx context.Context, evt event.Event) error {
	var payload CharacterStatePatchedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.character_state_patched payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	return a.applyStatePatch(ctx, evt.CampaignID, payload.CharacterID, payload.HpAfter, payload.HopeAfter, payload.HopeMaxAfter, payload.StressAfter, payload.ArmorAfter, payload.LifeStateAfter)
}

func (a *Adapter) applyConditionChanged(ctx context.Context, evt event.Event) error {
	var payload ConditionChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.condition_changed payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return fmt.Errorf("condition_changed roll_seq must be positive")
	}
	if payload.ConditionsAfter == nil {
		return fmt.Errorf("condition_changed conditions_after is required")
	}
	normalizedAfter, err := NormalizeConditions(payload.ConditionsAfter)
	if err != nil {
		return fmt.Errorf("condition_changed conditions_after: %w", err)
	}
	if len(payload.Added) > 0 {
		if _, err := NormalizeConditions(payload.Added); err != nil {
			return fmt.Errorf("condition_changed added: %w", err)
		}
	}
	if len(payload.Removed) > 0 {
		if _, err := NormalizeConditions(payload.Removed); err != nil {
			return fmt.Errorf("condition_changed removed: %w", err)
		}
	}
	return a.applyConditionPatch(ctx, evt.CampaignID, payload.CharacterID, normalizedAfter)
}

func (a *Adapter) applyGMFearChanged(ctx context.Context, evt event.Event) error {
	var payload GMFearChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.gm_fear_changed payload: %w", err)
	}
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

func (a *Adapter) applyGMMoveApplied(ctx context.Context, evt event.Event) error {
	var payload GMMoveAppliedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.gm_move_applied payload: %w", err)
	}
	if strings.TrimSpace(payload.Move) == "" {
		return fmt.Errorf("gm move is required")
	}
	if payload.FearSpent < 0 {
		return fmt.Errorf("gm move fear_spent must be non-negative")
	}
	return nil
}

func (a *Adapter) applyDeathMoveResolved(ctx context.Context, evt event.Event) error {
	var payload DeathMoveResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.death_move_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	if strings.TrimSpace(payload.Move) == "" {
		return fmt.Errorf("move is required")
	}
	if _, err := NormalizeDeathMove(payload.Move); err != nil {
		return fmt.Errorf("death_move move: %w", err)
	}
	if payload.LifeStateAfter == "" {
		return fmt.Errorf("life_state_after is required")
	}
	if _, err := NormalizeLifeState(payload.LifeStateAfter); err != nil {
		return fmt.Errorf("death_move life_state_after: %w", err)
	}
	if payload.HopeDie != nil && (*payload.HopeDie < 1 || *payload.HopeDie > 12) {
		return fmt.Errorf("death_move hope_die must be in range 1..12")
	}
	if payload.FearDie != nil && (*payload.FearDie < 1 || *payload.FearDie > 12) {
		return fmt.Errorf("death_move fear_die must be in range 1..12")
	}

	return a.applyStatePatch(
		ctx,
		evt.CampaignID,
		payload.CharacterID,
		payload.HpAfter,
		payload.HopeAfter,
		payload.HopeMaxAfter,
		payload.StressAfter,
		nil,
		&payload.LifeStateAfter,
	)
}

func (a *Adapter) applyBlazeOfGloryResolved(ctx context.Context, evt event.Event) error {
	var payload BlazeOfGloryResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.blaze_of_glory_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	if payload.LifeStateAfter == "" {
		return fmt.Errorf("life_state_after is required")
	}
	if _, err := NormalizeLifeState(payload.LifeStateAfter); err != nil {
		return fmt.Errorf("blaze_of_glory life_state_after: %w", err)
	}

	return a.applyStatePatch(
		ctx,
		evt.CampaignID,
		payload.CharacterID,
		nil,
		nil,
		nil,
		nil,
		nil,
		&payload.LifeStateAfter,
	)
}

func (a *Adapter) applyAttackResolved(ctx context.Context, evt event.Event) error {
	var payload AttackResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.attack_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	if payload.RollSeq == 0 {
		return fmt.Errorf("roll_seq is required")
	}
	if len(payload.Targets) == 0 {
		return fmt.Errorf("targets are required")
	}
	for _, target := range payload.Targets {
		if strings.TrimSpace(target) == "" {
			return fmt.Errorf("targets must not contain empty values")
		}
	}
	if strings.TrimSpace(payload.Outcome) == "" {
		return fmt.Errorf("outcome is required")
	}
	return nil
}

func (a *Adapter) applyReactionResolved(ctx context.Context, evt event.Event) error {
	var payload ReactionResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.reaction_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	if payload.RollSeq == 0 {
		return fmt.Errorf("roll_seq is required")
	}
	if strings.TrimSpace(payload.Outcome) == "" {
		return fmt.Errorf("outcome is required")
	}
	return nil
}

func (a *Adapter) applyDamageRollResolved(ctx context.Context, evt event.Event) error {
	var payload DamageRollResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.damage_roll_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return fmt.Errorf("character_id is required")
	}
	if payload.RollSeq == 0 {
		return fmt.Errorf("roll_seq is required")
	}
	if len(payload.Rolls) == 0 {
		return fmt.Errorf("rolls are required")
	}
	return nil
}

func (a *Adapter) applyGroupActionResolved(ctx context.Context, evt event.Event) error {
	var payload GroupActionResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.group_action_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.LeaderCharacterID) == "" {
		return fmt.Errorf("leader_character_id is required")
	}
	if payload.LeaderRollSeq == 0 {
		return fmt.Errorf("leader_roll_seq is required")
	}
	if payload.SupportSuccesses < 0 || payload.SupportFailures < 0 {
		return fmt.Errorf("support successes/failures must be non-negative")
	}
	for _, supporter := range payload.Supporters {
		if strings.TrimSpace(supporter.CharacterID) == "" {
			return fmt.Errorf("supporter character_id is required")
		}
		if supporter.RollSeq == 0 {
			return fmt.Errorf("supporter roll_seq is required")
		}
	}
	return nil
}

func (a *Adapter) applyTagTeamResolved(ctx context.Context, evt event.Event) error {
	var payload TagTeamResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.tag_team_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.FirstCharacterID) == "" {
		return fmt.Errorf("first_character_id is required")
	}
	if strings.TrimSpace(payload.SecondCharacterID) == "" {
		return fmt.Errorf("second_character_id is required")
	}
	if payload.FirstRollSeq == 0 || payload.SecondRollSeq == 0 {
		return fmt.Errorf("tag_team roll seqs are required")
	}
	if strings.TrimSpace(payload.SelectedCharacterID) == "" {
		return fmt.Errorf("selected_character_id is required")
	}
	if payload.SelectedRollSeq == 0 {
		return fmt.Errorf("selected_roll_seq is required")
	}
	if payload.SelectedCharacterID != payload.FirstCharacterID && payload.SelectedCharacterID != payload.SecondCharacterID {
		return fmt.Errorf("selected_character_id must match a participant")
	}
	if payload.SelectedCharacterID == payload.FirstCharacterID && payload.SelectedRollSeq != payload.FirstRollSeq {
		return fmt.Errorf("selected_roll_seq must match first_roll_seq")
	}
	if payload.SelectedCharacterID == payload.SecondCharacterID && payload.SelectedRollSeq != payload.SecondRollSeq {
		return fmt.Errorf("selected_roll_seq must match second_roll_seq")
	}
	return nil
}

func (a *Adapter) applyCountdownCreated(ctx context.Context, evt event.Event) error {
	var payload CountdownCreatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.countdown_created payload: %w", err)
	}
	if strings.TrimSpace(payload.CountdownID) == "" {
		return fmt.Errorf("countdown_id is required")
	}
	if strings.TrimSpace(payload.Name) == "" {
		return fmt.Errorf("countdown name is required")
	}
	if payload.Max <= 0 {
		return fmt.Errorf("countdown max must be positive")
	}
	if payload.Current < 0 || payload.Current > payload.Max {
		return fmt.Errorf("countdown current must be in range 0..%d", payload.Max)
	}
	if _, err := NormalizeCountdownKind(payload.Kind); err != nil {
		return err
	}
	if _, err := NormalizeCountdownDirection(payload.Direction); err != nil {
		return err
	}

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

func (a *Adapter) applyCountdownUpdated(ctx context.Context, evt event.Event) error {
	var payload CountdownUpdatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.countdown_updated payload: %w", err)
	}
	if strings.TrimSpace(payload.CountdownID) == "" {
		return fmt.Errorf("countdown_id is required")
	}
	if payload.Before < 0 || payload.After < 0 {
		return fmt.Errorf("countdown values must be non-negative")
	}

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

func (a *Adapter) applyCountdownDeleted(ctx context.Context, evt event.Event) error {
	var payload CountdownDeletedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.countdown_deleted payload: %w", err)
	}
	if strings.TrimSpace(payload.CountdownID) == "" {
		return fmt.Errorf("countdown_id is required")
	}
	return a.store.DeleteDaggerheartCountdown(ctx, evt.CampaignID, payload.CountdownID)
}

func (a *Adapter) applyAdversaryRollResolved(ctx context.Context, evt event.Event) error {
	var payload AdversaryRollResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.adversary_roll_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return fmt.Errorf("adversary_id is required")
	}
	if payload.RollSeq == 0 {
		return fmt.Errorf("roll_seq is required")
	}
	if payload.Roll < 1 || payload.Roll > 20 {
		return fmt.Errorf("roll must be in range 1..20")
	}
	if len(payload.Rolls) == 0 {
		return fmt.Errorf("rolls are required")
	}
	return nil
}

func (a *Adapter) applyAdversaryAttackResolved(ctx context.Context, evt event.Event) error {
	var payload AdversaryAttackResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.adversary_attack_resolved payload: %w", err)
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return fmt.Errorf("adversary_id is required")
	}
	if payload.RollSeq == 0 {
		return fmt.Errorf("roll_seq is required")
	}
	if len(payload.Targets) == 0 {
		return fmt.Errorf("targets are required")
	}
	for _, target := range payload.Targets {
		if strings.TrimSpace(target) == "" {
			return fmt.Errorf("targets must not contain empty values")
		}
	}
	if payload.Roll < 1 || payload.Roll > 20 {
		return fmt.Errorf("roll must be in range 1..20")
	}
	if payload.Difficulty < 0 {
		return fmt.Errorf("difficulty must be non-negative")
	}
	return nil
}

func (a *Adapter) applyAdversaryCreated(ctx context.Context, evt event.Event) error {
	var payload AdversaryCreatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.adversary_created payload: %w", err)
	}
	adversaryID := strings.TrimSpace(payload.AdversaryID)
	if adversaryID == "" {
		return fmt.Errorf("adversary_id is required")
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateAdversaryStats(payload.HP, payload.HPMax, payload.Stress, payload.StressMax, payload.Evasion, payload.Major, payload.Severe, payload.Armor); err != nil {
		return err
	}
	createdAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{
		CampaignID:  evt.CampaignID,
		AdversaryID: adversaryID,
		Name:        name,
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

func (a *Adapter) applyAdversaryUpdated(ctx context.Context, evt event.Event) error {
	var payload AdversaryUpdatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.adversary_updated payload: %w", err)
	}
	adversaryID := strings.TrimSpace(payload.AdversaryID)
	if adversaryID == "" {
		return fmt.Errorf("adversary_id is required")
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
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
		Name:        name,
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
		CreatedAt:   current.CreatedAt,
		UpdatedAt:   updatedAt,
	})
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

func (a *Adapter) applyAdversaryDeleted(ctx context.Context, evt event.Event) error {
	var payload AdversaryDeletedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode action.adversary_deleted payload: %w", err)
	}
	adversaryID := strings.TrimSpace(payload.AdversaryID)
	if adversaryID == "" {
		return fmt.Errorf("adversary_id is required")
	}
	return a.store.DeleteDaggerheartAdversary(ctx, evt.CampaignID, adversaryID)
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

var _ systems.Adapter = (*Adapter)(nil)
