package adapter

import (
	"context"
	"fmt"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func (a *Adapter) HandleCharacterProfileReplaced(ctx context.Context, evt event.Event, p snapstate.CharacterProfileReplacedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(evt.EntityID)
	}
	return a.PutCharacterProfile(ctx, string(evt.CampaignID), characterID, p.Profile)
}

func (a *Adapter) HandleCharacterProfileDeleted(ctx context.Context, evt event.Event, p snapstate.CharacterProfileDeletedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(evt.EntityID)
	}
	if err := a.store.DeleteDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID); err != nil {
		return fmt.Errorf("delete daggerheart profile: %w", err)
	}
	return nil
}

func (a *Adapter) PutCharacterProfile(ctx context.Context, campaignID, characterID string, profile snapstate.CharacterProfile) error {
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

	_, exists, err := a.GetCharacterStateIfExists(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return a.PutCharacterState(ctx, projectionstore.DaggerheartCharacterState{
		CampaignID:     campaignID,
		CharacterID:    characterID,
		Hp:             profile.HpMax,
		Hope:           snapstate.HopeDefault,
		HopeMax:        snapstate.HopeMaxDefault,
		Stress:         snapstate.StressDefault,
		Armor:          profile.ArmorMax,
		LifeState:      snapstate.LifeStateAlive,
		CompanionState: CompanionProjectionStateFromProfile(profile),
	})
}

// CompanionProjectionStateFromProfile derives companion projection state from
// a character profile's companion sheet presence.
func CompanionProjectionStateFromProfile(profile snapstate.CharacterProfile) *projectionstore.DaggerheartCompanionState {
	if profile.CompanionSheet == nil {
		return nil
	}
	return &projectionstore.DaggerheartCompanionState{Status: snapstate.CompanionStatusPresent}
}
