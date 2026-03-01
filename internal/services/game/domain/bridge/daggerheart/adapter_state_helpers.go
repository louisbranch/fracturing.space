package daggerheart

import (
	"context"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// putSnapshot centralizes snapshot write error wrapping so handler code stays
// focused on payload-to-state transformation rules.
func (a *Adapter) putSnapshot(ctx context.Context, campaignID string, gmFear, shortRests int) error {
	if err := a.store.PutDaggerheartSnapshot(ctx, storage.DaggerheartSnapshot{
		CampaignID:            campaignID,
		GMFear:                gmFear,
		ConsecutiveShortRests: shortRests,
	}); err != nil {
		return fmt.Errorf("put daggerheart snapshot: %w", err)
	}
	return nil
}

// snapshotShortRests returns the current short-rest streak or zero when no
// snapshot exists yet.
func (a *Adapter) snapshotShortRests(ctx context.Context, campaignID string) int {
	current, err := a.store.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return 0
	}
	return current.ConsecutiveShortRests
}

// getCharacterStateIfExists loads character state and reports existence. Missing
// rows are not considered errors.
func (a *Adapter) getCharacterStateIfExists(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, bool, error) {
	state, err := a.store.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.DaggerheartCharacterState{}, false, nil
		}
		return storage.DaggerheartCharacterState{}, false, fmt.Errorf("get daggerheart character state: %w", err)
	}
	return state, true, nil
}

// getCharacterStateOrDefault loads existing character state or builds a default
// state for first-write projection paths.
func (a *Adapter) getCharacterStateOrDefault(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	state, exists, err := a.getCharacterStateIfExists(ctx, campaignID, characterID)
	if err != nil {
		return storage.DaggerheartCharacterState{}, err
	}
	if exists {
		return state, nil
	}
	return storage.DaggerheartCharacterState{CampaignID: campaignID, CharacterID: characterID}, nil
}

// putCharacterState centralizes character state write error wrapping.
func (a *Adapter) putCharacterState(ctx context.Context, state storage.DaggerheartCharacterState) error {
	if err := a.store.PutDaggerheartCharacterState(ctx, state); err != nil {
		return fmt.Errorf("put daggerheart character state: %w", err)
	}
	return nil
}
