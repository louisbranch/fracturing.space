package adapter

import (
	"context"
	"fmt"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (a *Adapter) HandleGMFearChanged(ctx context.Context, evt event.Event, p payload.GMFearChangedPayload) error {
	if p.Value < daggerheartstate.GMFearMin || p.Value > daggerheartstate.GMFearMax {
		return fmt.Errorf("gm_fear_changed value must be in range %d..%d", daggerheartstate.GMFearMin, daggerheartstate.GMFearMax)
	}
	shortRests := a.SnapshotShortRests(ctx, string(evt.CampaignID))
	return a.PutSnapshot(ctx, string(evt.CampaignID), p.Value, shortRests)
}

func (a *Adapter) HandleSceneCountdownCreated(ctx context.Context, evt event.Event, p payload.SceneCountdownCreatedPayload) error {
	sessionID := evt.SessionID.String()
	if sessionID == "" {
		sessionID = p.SessionID.String()
	}
	sceneID := evt.SceneID.String()
	if sceneID == "" {
		sceneID = p.SceneID.String()
	}
	remainingValue := p.RemainingValue
	if remainingValue <= 0 {
		remainingValue = p.StartingValue
	}
	loopBehavior := p.LoopBehavior
	if loopBehavior == "" {
		loopBehavior = "none"
	}
	status := p.Status
	if status == "" {
		status = "active"
	}
	return a.store.PutDaggerheartCountdown(ctx, projectionstore.DaggerheartCountdown{
		CampaignID:        string(evt.CampaignID),
		SessionID:         sessionID,
		SceneID:           sceneID,
		CountdownID:       p.CountdownID.String(),
		Name:              p.Name,
		Tone:              p.Tone,
		AdvancementPolicy: p.AdvancementPolicy,
		StartingValue:     p.StartingValue,
		RemainingValue:    remainingValue,
		LoopBehavior:      loopBehavior,
		Status:            status,
		LinkedCountdownID: p.LinkedCountdownID.String(),
		StartingRollMin:   valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Min }),
		StartingRollMax:   valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Max }),
		StartingRollValue: valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Value }),
	})
}

func (a *Adapter) HandleSceneCountdownAdvanced(ctx context.Context, evt event.Event, p payload.SceneCountdownAdvancedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	nextCountdown := countdown
	afterRemaining := p.AfterRemaining
	if afterRemaining < 0 || (countdown.StartingValue > 0 && afterRemaining > countdown.StartingValue) {
		return fmt.Errorf("countdown remaining value must be in range 0..%d", countdown.StartingValue)
	}
	nextCountdown.RemainingValue = afterRemaining
	if strings.TrimSpace(p.StatusAfter) != "" {
		nextCountdown.Status = p.StatusAfter
	}
	if evt.SessionID.String() != "" {
		nextCountdown.SessionID = evt.SessionID.String()
	}
	if evt.SceneID.String() != "" {
		nextCountdown.SceneID = evt.SceneID.String()
	}
	return a.store.PutDaggerheartCountdown(ctx, nextCountdown)
}

func (a *Adapter) HandleSceneCountdownDeleted(ctx context.Context, evt event.Event, p payload.SceneCountdownDeletedPayload) error {
	return a.store.DeleteDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
}

func (a *Adapter) HandleCampaignCountdownCreated(ctx context.Context, evt event.Event, p payload.CampaignCountdownCreatedPayload) error {
	remainingValue := p.RemainingValue
	if remainingValue <= 0 {
		remainingValue = p.StartingValue
	}
	loopBehavior := p.LoopBehavior
	if loopBehavior == "" {
		loopBehavior = "none"
	}
	status := p.Status
	if status == "" {
		status = "active"
	}
	return a.store.PutDaggerheartCountdown(ctx, projectionstore.DaggerheartCountdown{
		CampaignID:        string(evt.CampaignID),
		CountdownID:       p.CountdownID.String(),
		Name:              p.Name,
		Tone:              p.Tone,
		AdvancementPolicy: p.AdvancementPolicy,
		StartingValue:     p.StartingValue,
		RemainingValue:    remainingValue,
		LoopBehavior:      loopBehavior,
		Status:            status,
		LinkedCountdownID: p.LinkedCountdownID.String(),
		StartingRollMin:   valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Min }),
		StartingRollMax:   valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Max }),
		StartingRollValue: valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Value }),
	})
}

func (a *Adapter) HandleCampaignCountdownAdvanced(ctx context.Context, evt event.Event, p payload.CampaignCountdownAdvancedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	nextCountdown := countdown
	afterRemaining := p.AfterRemaining
	if afterRemaining < 0 || (countdown.StartingValue > 0 && afterRemaining > countdown.StartingValue) {
		return fmt.Errorf("countdown remaining value must be in range 0..%d", countdown.StartingValue)
	}
	nextCountdown.RemainingValue = afterRemaining
	if strings.TrimSpace(p.StatusAfter) != "" {
		nextCountdown.Status = p.StatusAfter
	}
	nextCountdown.SessionID = ""
	nextCountdown.SceneID = ""
	return a.store.PutDaggerheartCountdown(ctx, nextCountdown)
}

func (a *Adapter) HandleSceneCountdownTriggerResolved(ctx context.Context, evt event.Event, p payload.SceneCountdownTriggerResolvedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	countdown.StartingValue = p.StartingValueAfter
	countdown.RemainingValue = p.RemainingValueAfter
	countdown.Status = p.StatusAfter
	return a.store.PutDaggerheartCountdown(ctx, countdown)
}

func (a *Adapter) HandleCampaignCountdownTriggerResolved(ctx context.Context, evt event.Event, p payload.CampaignCountdownTriggerResolvedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	countdown.StartingValue = p.StartingValueAfter
	countdown.RemainingValue = p.RemainingValueAfter
	countdown.Status = p.StatusAfter
	return a.store.PutDaggerheartCountdown(ctx, countdown)
}

func (a *Adapter) HandleCampaignCountdownDeleted(ctx context.Context, evt event.Event, p payload.CampaignCountdownDeletedPayload) error {
	return a.store.DeleteDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
}

func valueOrZero[T any](value *T, get func(*T) int) int {
	if value == nil {
		return 0
	}
	return get(value)
}
