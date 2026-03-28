package daggerheart

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const readinessProfilePageSize = 200

type sessionStartReadinessStateLoader struct{}

func (sessionStartReadinessStateLoader) LoadSessionStartReadinessState(
	ctx context.Context,
	campaignID ids.CampaignID,
	storeSource any,
	state aggregate.State,
) (aggregate.State, error) {
	store := projectionStoreFromSource(storeSource)
	if store == nil {
		return aggregate.State{}, fmt.Errorf("daggerheart projection store is not configured")
	}
	if state.Systems == nil {
		state.Systems = make(map[module.Key]any)
	}

	snapshot := daggerheartstate.NewSnapshotState(campaignID)
	storedSnapshot, err := store.GetDaggerheartSnapshot(ctx, string(campaignID))
	switch {
	case err == nil:
		snapshot.GMFear = storedSnapshot.GMFear
	case errors.Is(err, storage.ErrNotFound):
	case err != nil:
		return aggregate.State{}, fmt.Errorf("get daggerheart snapshot: %w", err)
	}

	profiles, err := listAllCharacterProfiles(ctx, store, campaignID)
	if err != nil {
		return aggregate.State{}, err
	}
	for _, profile := range profiles {
		characterID := ids.CharacterID(strings.TrimSpace(profile.CharacterID))
		if characterID == "" {
			continue
		}
		if _, ok := state.Characters[characterID]; !ok {
			continue
		}
		snapshot.CharacterProfiles[characterID] = daggerheartstate.CharacterProfileFromStorage(profile)
	}

	state.Systems[module.Key{ID: SystemID, Version: SystemVersion}] = snapshot
	return state, nil
}

func projectionStoreFromSource(storeSource any) projectionstore.Store {
	if provider, ok := storeSource.(interface {
		DaggerheartProjectionStore() projectionstore.Store
	}); ok {
		return provider.DaggerheartProjectionStore()
	}
	store, _ := storeSource.(projectionstore.Store)
	return store
}

func listAllCharacterProfiles(
	ctx context.Context,
	store projectionstore.Store,
	campaignID ids.CampaignID,
) ([]projectionstore.DaggerheartCharacterProfile, error) {
	profiles := make([]projectionstore.DaggerheartCharacterProfile, 0, readinessProfilePageSize)
	pageToken := ""
	seenTokens := map[string]struct{}{"": {}}
	for {
		page, err := store.ListDaggerheartCharacterProfiles(ctx, string(campaignID), readinessProfilePageSize, pageToken)
		if err != nil {
			return nil, fmt.Errorf("list daggerheart character profiles: %w", err)
		}
		profiles = append(profiles, page.Profiles...)

		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" {
			return profiles, nil
		}
		if _, exists := seenTokens[nextPageToken]; exists {
			return nil, fmt.Errorf("list daggerheart character profiles returned a repeated page token")
		}
		seenTokens[nextPageToken] = struct{}{}
		pageToken = nextPageToken
	}
}

var _ bridge.SessionStartReadinessStateLoader = sessionStartReadinessStateLoader{}
