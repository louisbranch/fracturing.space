package daggerheart

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func (a *Adapter) handleCharacterProfileReplaced(ctx context.Context, evt event.Event, payload CharacterProfileReplacedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(evt.EntityID)
	}
	return a.putCharacterProfile(ctx, string(evt.CampaignID), characterID, payload.Profile)
}

func (a *Adapter) handleCharacterProfileDeleted(ctx context.Context, evt event.Event, payload CharacterProfileDeletedPayload) error {
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(evt.EntityID)
	}
	if err := a.store.DeleteDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID); err != nil {
		return fmt.Errorf("delete daggerheart profile: %w", err)
	}
	return nil
}

func (a *Adapter) putCharacterProfile(ctx context.Context, campaignID, characterID string, profile CharacterProfile) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("daggerheart store is not configured")
	}
	profile = profile.Normalized()
	if err := profile.Validate(); err != nil {
		return err
	}
	if err := a.store.PutDaggerheartCharacterProfile(ctx, profile.ToStorage(campaignID, characterID)); err != nil {
		return err
	}

	_, exists, err := a.getCharacterStateIfExists(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return a.putCharacterState(ctx, projectionstore.DaggerheartCharacterState{
		CampaignID:     campaignID,
		CharacterID:    characterID,
		Hp:             profile.HpMax,
		Hope:           HopeDefault,
		HopeMax:        HopeMaxDefault,
		Stress:         StressDefault,
		Armor:          profile.ArmorMax,
		LifeState:      LifeStateAlive,
		CompanionState: companionProjectionStateFromProfile(profile),
	})
}

func companionProjectionStateFromProfile(profile CharacterProfile) *projectionstore.DaggerheartCompanionState {
	if profile.CompanionSheet == nil {
		return nil
	}
	return &projectionstore.DaggerheartCompanionState{Status: CompanionStatusPresent}
}
