package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// PutSnapshot centralizes snapshot write error wrapping so handler code stays
// focused on payload-to-state transformation rules.
func (a *Adapter) PutSnapshot(ctx context.Context, campaignID string, gmFear, shortRests int) error {
	if err := a.store.PutDaggerheartSnapshot(ctx, projectionstore.DaggerheartSnapshot{
		CampaignID:            campaignID,
		GMFear:                gmFear,
		ConsecutiveShortRests: shortRests,
	}); err != nil {
		return fmt.Errorf("put daggerheart snapshot: %w", err)
	}
	return nil
}

// SnapshotShortRests returns the current short-rest streak or zero when no
// snapshot exists yet.
func (a *Adapter) SnapshotShortRests(ctx context.Context, campaignID string) int {
	current, err := a.store.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return 0
	}
	return current.ConsecutiveShortRests
}

// GetCharacterStateIfExists loads character state and reports existence.
// Missing rows are not considered errors.
func (a *Adapter) GetCharacterStateIfExists(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, bool, error) {
	state, err := a.store.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return projectionstore.DaggerheartCharacterState{}, false, nil
		}
		return projectionstore.DaggerheartCharacterState{}, false, fmt.Errorf("get daggerheart character state: %w", err)
	}
	return state, true, nil
}

// GetCharacterStateOrDefault loads existing character state or builds a default
// state for first-write projection paths.
func (a *Adapter) GetCharacterStateOrDefault(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	state, exists, err := a.GetCharacterStateIfExists(ctx, campaignID, characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}
	if exists {
		return state, nil
	}
	return projectionstore.DaggerheartCharacterState{CampaignID: campaignID, CharacterID: characterID}, nil
}

// PutCharacterState centralizes character state write error wrapping.
func (a *Adapter) PutCharacterState(ctx context.Context, state projectionstore.DaggerheartCharacterState) error {
	if err := a.store.PutDaggerheartCharacterState(ctx, state); err != nil {
		return fmt.Errorf("put daggerheart character state: %w", err)
	}
	return nil
}

// CharacterArmorMax resolves the armor max for a character, falling back to a
// state-derived value when no profile exists.
func (a *Adapter) CharacterArmorMax(ctx context.Context, state projectionstore.DaggerheartCharacterState) (int, error) {
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

// ClearRestTemporaryArmor removes temporary armor entries that expire on short
// or long rest, then persists the updated state.
func (a *Adapter) ClearRestTemporaryArmor(ctx context.Context, campaignID, characterID string, clearShortRest bool, clearLongRest bool) error {
	state, exists, err := a.GetCharacterStateIfExists(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	armorMax, err := a.CharacterArmorMax(ctx, state)
	if err != nil {
		return err
	}
	nextState, changed := projection.ClearRestTemporaryArmor(state, armorMax, clearShortRest, clearLongRest)
	if !changed {
		return nil
	}
	if err := a.PutCharacterState(ctx, nextState); err != nil {
		return err
	}
	return nil
}

// ClearRestStatModifiers removes stat modifiers with matching rest triggers.
func (a *Adapter) ClearRestStatModifiers(ctx context.Context, campaignID, characterID string, clearShortRest, clearLongRest bool) error {
	state, exists, err := a.GetCharacterStateIfExists(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	if !exists || len(state.StatModifiers) == 0 {
		return nil
	}
	modifiers := StatModifiersFromProjection(state.StatModifiers)
	changed := false
	if clearShortRest {
		remaining, _ := rules.ClearStatModifiersByTrigger(modifiers, rules.ConditionClearTriggerShortRest)
		if len(remaining) != len(modifiers) {
			modifiers = remaining
			changed = true
		}
	}
	if clearLongRest {
		remaining, _ := rules.ClearStatModifiersByTrigger(modifiers, rules.ConditionClearTriggerLongRest)
		if len(remaining) != len(modifiers) {
			modifiers = remaining
			changed = true
		}
	}
	if !changed {
		return nil
	}
	state.StatModifiers = StatModifiersToProjection(modifiers)
	return a.PutCharacterState(ctx, state)
}
