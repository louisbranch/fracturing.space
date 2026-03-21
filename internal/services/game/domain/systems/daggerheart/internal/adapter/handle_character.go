package adapter

import (
	"context"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (a *Adapter) HandleDamageApplied(ctx context.Context, evt event.Event, p payload.DamageAppliedPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), StatePatch{
		HP: p.Hp, Stress: p.Stress, Armor: p.Armor,
	})
}

func (a *Adapter) HandleDowntimeMoveApplied(ctx context.Context, evt event.Event, p payload.DowntimeMoveAppliedPayload) error {
	characterID := strings.TrimSpace(p.TargetCharacterID.String())
	if characterID == "" {
		characterID = strings.TrimSpace(p.ActorCharacterID.String())
	}
	if characterID == "" {
		return nil
	}
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), characterID, StatePatch{
		HP: p.HP, Hope: p.Hope, Stress: p.Stress, Armor: p.Armor,
	})
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
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), StatePatch{Stress: p.Stress})
}

func (a *Adapter) HandleCharacterStatePatched(ctx context.Context, evt event.Event, p payload.CharacterStatePatchedPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), StatePatch{
		HP:                            p.HP,
		Hope:                          p.Hope,
		HopeMax:                       p.HopeMax,
		Stress:                        p.Stress,
		Armor:                         p.Armor,
		LifeState:                     p.LifeState,
		ClassState:                    p.ClassState,
		SubclassState:                 p.SubclassState,
		ImpenetrableUsedThisShortRest: p.ImpenetrableUsedThisShortRest,
	})
}

func (a *Adapter) HandleBeastformTransformed(ctx context.Context, evt event.Event, p payload.BeastformTransformedPayload) error {
	state, err := a.GetCharacterStateOrDefault(ctx, string(evt.CampaignID), p.CharacterID.String())
	if err != nil {
		return err
	}
	nextClassState := daggerheartstate.WithActiveBeastform(ClassStateFromProjection(state.ClassState), p.ActiveBeastform)
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), StatePatch{
		Hope: p.Hope, Stress: p.Stress, ClassState: &nextClassState,
	})
}

func (a *Adapter) HandleBeastformDropped(ctx context.Context, evt event.Event, p payload.BeastformDroppedPayload) error {
	state, err := a.GetCharacterStateOrDefault(ctx, string(evt.CampaignID), p.CharacterID.String())
	if err != nil {
		return err
	}
	nextClassState := daggerheartstate.WithActiveBeastform(ClassStateFromProjection(state.ClassState), nil)
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), StatePatch{ClassState: &nextClassState})
}

func (a *Adapter) HandleCompanionExperienceBegun(ctx context.Context, evt event.Event, p payload.CompanionExperienceBegunPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), StatePatch{CompanionState: p.CompanionState})
}

func (a *Adapter) HandleCompanionReturned(ctx context.Context, evt event.Event, p payload.CompanionReturnedPayload) error {
	return a.ApplyStatePatch(ctx, string(evt.CampaignID), p.CharacterID.String(), StatePatch{
		Stress: p.Stress, CompanionState: p.CompanionState,
	})
}
