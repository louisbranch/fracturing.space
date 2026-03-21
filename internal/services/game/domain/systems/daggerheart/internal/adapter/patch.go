package adapter

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func (a *Adapter) ApplyStatePatch(ctx context.Context, campaignID, characterID string, patch StatePatch) error {
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
		projection.StatePatch{
			HP:                            patch.HP,
			Hope:                          patch.Hope,
			HopeMax:                       patch.HopeMax,
			Stress:                        patch.Stress,
			Armor:                         patch.Armor,
			LifeState:                     patch.LifeState,
			ClassState:                    ClassStateToProjection(patch.ClassState),
			SubclassState:                 SubclassStateToProjection(patch.SubclassState),
			CompanionState:                CompanionStateToProjection(patch.CompanionState),
			ImpenetrableUsedThisShortRest: patch.ImpenetrableUsedThisShortRest,
		},
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
