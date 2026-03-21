package adapter

import (
	"context"
	"fmt"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (a *Adapter) HandleCharacterProfileReplaced(ctx context.Context, evt event.Event, p daggerheartstate.CharacterProfileReplacedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(evt.EntityID)
	}
	return a.PutCharacterProfile(ctx, string(evt.CampaignID), characterID, p.Profile)
}

func (a *Adapter) HandleCharacterProfileDeleted(ctx context.Context, evt event.Event, p daggerheartstate.CharacterProfileDeletedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(evt.EntityID)
	}
	if err := a.store.DeleteDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID); err != nil {
		return fmt.Errorf("delete daggerheart profile: %w", err)
	}
	return nil
}

func (a *Adapter) PutCharacterProfile(ctx context.Context, campaignID, characterID string, profile daggerheartstate.CharacterProfile) error {
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
		Hope:           daggerheartstate.HopeDefault,
		HopeMax:        daggerheartstate.HopeMaxDefault,
		Stress:         daggerheartstate.StressDefault,
		Armor:          profile.ArmorMax,
		LifeState:      daggerheartstate.LifeStateAlive,
		CompanionState: CompanionProjectionStateFromProfile(profile),
	})
}

// CompanionProjectionStateFromProfile derives companion projection state from
// a character profile's companion sheet presence.
func CompanionProjectionStateFromProfile(profile daggerheartstate.CharacterProfile) *projectionstore.DaggerheartCompanionState {
	if profile.CompanionSheet == nil {
		return nil
	}
	return &projectionstore.DaggerheartCompanionState{Status: daggerheartstate.CompanionStatusPresent}
}
