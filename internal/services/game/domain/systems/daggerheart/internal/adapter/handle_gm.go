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
	tone := p.Tone
	if tone == "" {
		tone = p.Kind
	}
	startingValue := p.StartingValue
	if startingValue <= 0 {
		startingValue = p.Max
	}
	remainingValue := p.RemainingValue
	if remainingValue <= 0 && (p.Current > 0 || p.Max > 0) {
		remainingValue = p.Current
	}
	if remainingValue <= 0 {
		remainingValue = startingValue
	}
	loopBehavior := p.LoopBehavior
	if loopBehavior == "" {
		if p.Looping {
			loopBehavior = "reset"
		} else {
			loopBehavior = "none"
		}
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
		Tone:              tone,
		AdvancementPolicy: p.AdvancementPolicy,
		StartingValue:     startingValue,
		RemainingValue:    remainingValue,
		LoopBehavior:      loopBehavior,
		Status:            status,
		LinkedCountdownID: p.LinkedCountdownID.String(),
		StartingRollMin:   valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Min }),
		StartingRollMax:   valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Max }),
		StartingRollValue: valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Value }),
		Kind:              p.Kind,
		Current:           p.Current,
		Max:               p.Max,
		Direction:         p.Direction,
		Looping:           p.Looping,
		Variant:           p.Variant,
		TriggerEventType:  p.TriggerEventType,
	})
}

// HandleCountdownCreated is the legacy generic wrapper retained temporarily for
// older tests. It routes to the correct ownership path based on payload
// session/scene ownership.
func (a *Adapter) HandleCountdownCreated(ctx context.Context, evt event.Event, p payload.CountdownCreatedPayload) error {
	if p.SceneID != "" || p.SessionID != "" {
		return a.HandleSceneCountdownCreated(ctx, evt, p)
	}
	return a.HandleCampaignCountdownCreated(ctx, evt, p)
}

func (a *Adapter) HandleSceneCountdownAdvanced(ctx context.Context, evt event.Event, p payload.SceneCountdownAdvancedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	nextCountdown := countdown
	startingValue := countdown.StartingValue
	if startingValue <= 0 {
		startingValue = countdown.Max
	}
	afterRemaining := p.AfterRemaining
	if afterRemaining == 0 && (p.After != 0 || p.Value != 0) {
		if p.After != 0 {
			afterRemaining = p.After
		} else {
			afterRemaining = p.Value
		}
	}
	if afterRemaining < 0 || (startingValue > 0 && afterRemaining > startingValue) {
		return fmt.Errorf("countdown remaining value must be in range 0..%d", startingValue)
	}
	nextCountdown.RemainingValue = afterRemaining
	nextCountdown.Current = afterRemaining
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

// HandleCountdownUpdated is the legacy generic wrapper retained temporarily for
// older tests. It routes using the stored countdown ownership.
func (a *Adapter) HandleCountdownUpdated(ctx context.Context, evt event.Event, p payload.CampaignCountdownAdvancedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	if countdown.SceneID != "" || countdown.SessionID != "" {
		return a.HandleSceneCountdownAdvanced(ctx, evt, payload.SceneCountdownAdvancedPayload(p))
	}
	return a.HandleCampaignCountdownAdvanced(ctx, evt, p)
}

func (a *Adapter) HandleSceneCountdownDeleted(ctx context.Context, evt event.Event, p payload.SceneCountdownDeletedPayload) error {
	return a.store.DeleteDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
}

// HandleCountdownDeleted is the legacy generic wrapper retained temporarily for
// older tests. It routes using the stored countdown ownership.
func (a *Adapter) HandleCountdownDeleted(ctx context.Context, evt event.Event, p payload.CountdownDeletedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	if countdown.SceneID != "" || countdown.SessionID != "" {
		return a.HandleSceneCountdownDeleted(ctx, evt, p)
	}
	return a.HandleCampaignCountdownDeleted(ctx, evt, p)
}

func (a *Adapter) HandleCampaignCountdownCreated(ctx context.Context, evt event.Event, p payload.CampaignCountdownCreatedPayload) error {
	tone := p.Tone
	if tone == "" {
		tone = p.Kind
	}
	startingValue := p.StartingValue
	if startingValue <= 0 {
		startingValue = p.Max
	}
	remainingValue := p.RemainingValue
	if remainingValue <= 0 && (p.Current > 0 || p.Max > 0) {
		remainingValue = p.Current
	}
	if remainingValue <= 0 {
		remainingValue = startingValue
	}
	loopBehavior := p.LoopBehavior
	if loopBehavior == "" {
		if p.Looping {
			loopBehavior = "reset"
		} else {
			loopBehavior = "none"
		}
	}
	status := p.Status
	if status == "" {
		status = "active"
	}
	return a.store.PutDaggerheartCountdown(ctx, projectionstore.DaggerheartCountdown{
		CampaignID:        string(evt.CampaignID),
		CountdownID:       p.CountdownID.String(),
		Name:              p.Name,
		Tone:              tone,
		AdvancementPolicy: p.AdvancementPolicy,
		StartingValue:     startingValue,
		RemainingValue:    remainingValue,
		LoopBehavior:      loopBehavior,
		Status:            status,
		LinkedCountdownID: p.LinkedCountdownID.String(),
		StartingRollMin:   valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Min }),
		StartingRollMax:   valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Max }),
		StartingRollValue: valueOrZero(p.StartingRoll, func(v *payload.CountdownStartingRollPayload) int { return v.Value }),
		Kind:              p.Kind,
		Current:           p.Current,
		Max:               p.Max,
		Direction:         p.Direction,
		Looping:           p.Looping,
		Variant:           p.Variant,
		TriggerEventType:  p.TriggerEventType,
	})
}

func (a *Adapter) HandleCampaignCountdownAdvanced(ctx context.Context, evt event.Event, p payload.CampaignCountdownAdvancedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	nextCountdown := countdown
	startingValue := countdown.StartingValue
	if startingValue <= 0 {
		startingValue = countdown.Max
	}
	afterRemaining := p.AfterRemaining
	if afterRemaining == 0 && (p.After != 0 || p.Value != 0) {
		if p.After != 0 {
			afterRemaining = p.After
		} else {
			afterRemaining = p.Value
		}
	}
	if afterRemaining < 0 || (startingValue > 0 && afterRemaining > startingValue) {
		return fmt.Errorf("countdown remaining value must be in range 0..%d", startingValue)
	}
	nextCountdown.RemainingValue = afterRemaining
	nextCountdown.Current = afterRemaining
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
