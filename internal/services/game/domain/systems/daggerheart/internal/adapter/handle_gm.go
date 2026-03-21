package adapter

import (
	"context"
	"fmt"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/projection"
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

func (a *Adapter) HandleCountdownCreated(ctx context.Context, evt event.Event, p payload.CountdownCreatedPayload) error {
	return a.store.PutDaggerheartCountdown(ctx, projectionstore.DaggerheartCountdown{
		CampaignID:        string(evt.CampaignID),
		CountdownID:       p.CountdownID.String(),
		Name:              p.Name,
		Kind:              p.Kind,
		Current:           p.Current,
		Max:               p.Max,
		Direction:         p.Direction,
		Looping:           p.Looping,
		Variant:           p.Variant,
		TriggerEventType:  p.TriggerEventType,
		LinkedCountdownID: p.LinkedCountdownID.String(),
	})
}

func (a *Adapter) HandleCountdownUpdated(ctx context.Context, evt event.Event, p payload.CountdownUpdatedPayload) error {
	countdown, err := a.store.GetDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
	if err != nil {
		return err
	}
	nextCountdown, err := projection.ApplyCountdownUpdate(countdown, p.Value)
	if err != nil {
		return err
	}
	return a.store.PutDaggerheartCountdown(ctx, nextCountdown)
}

func (a *Adapter) HandleCountdownDeleted(ctx context.Context, evt event.Event, p payload.CountdownDeletedPayload) error {
	return a.store.DeleteDaggerheartCountdown(ctx, string(evt.CampaignID), p.CountdownID.String())
}
