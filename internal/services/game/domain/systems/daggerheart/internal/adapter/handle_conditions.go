package adapter

import (
	"context"
	"fmt"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

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
