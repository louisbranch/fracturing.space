package adapter

import (
	"context"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func (a *Adapter) HandleEnvironmentEntityCreated(ctx context.Context, evt event.Event, p payload.EnvironmentEntityCreatedPayload) error {
	createdAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartEnvironmentEntity(ctx, projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          string(evt.CampaignID),
		EnvironmentEntityID: strings.TrimSpace(p.EnvironmentEntityID.String()),
		EnvironmentID:       strings.TrimSpace(p.EnvironmentID),
		Name:                strings.TrimSpace(p.Name),
		Type:                strings.TrimSpace(p.Type),
		Tier:                p.Tier,
		Difficulty:          p.Difficulty,
		SessionID:           strings.TrimSpace(p.SessionID.String()),
		SceneID:             strings.TrimSpace(p.SceneID.String()),
		Notes:               strings.TrimSpace(p.Notes),
		CreatedAt:           createdAt,
		UpdatedAt:           createdAt,
	})
}

func (a *Adapter) HandleEnvironmentEntityUpdated(ctx context.Context, evt event.Event, p payload.EnvironmentEntityUpdatedPayload) error {
	environmentEntityID := strings.TrimSpace(p.EnvironmentEntityID.String())
	current, err := a.store.GetDaggerheartEnvironmentEntity(ctx, string(evt.CampaignID), environmentEntityID)
	if err != nil {
		return err
	}
	return a.store.PutDaggerheartEnvironmentEntity(ctx, projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          string(evt.CampaignID),
		EnvironmentEntityID: environmentEntityID,
		EnvironmentID:       strings.TrimSpace(p.EnvironmentID),
		Name:                strings.TrimSpace(p.Name),
		Type:                strings.TrimSpace(p.Type),
		Tier:                p.Tier,
		Difficulty:          p.Difficulty,
		SessionID:           strings.TrimSpace(p.SessionID.String()),
		SceneID:             strings.TrimSpace(p.SceneID.String()),
		Notes:               strings.TrimSpace(p.Notes),
		CreatedAt:           current.CreatedAt,
		UpdatedAt:           evt.Timestamp.UTC(),
	})
}

func (a *Adapter) HandleEnvironmentEntityDeleted(ctx context.Context, evt event.Event, p payload.EnvironmentEntityDeletedPayload) error {
	return a.store.DeleteDaggerheartEnvironmentEntity(ctx, string(evt.CampaignID), strings.TrimSpace(p.EnvironmentEntityID.String()))
}
