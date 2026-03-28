package projection

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applySessionSceneActivate(ctx context.Context, evt event.Event, payload session.SceneActivatedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.ActiveSceneID = strings.TrimSpace(payload.ActiveSceneID.String())
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionGMAuthoritySet(ctx context.Context, evt event.Event, payload session.GMAuthoritySetPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.GMAuthorityParticipantID = strings.TrimSpace(payload.ParticipantID.String())
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionCharacterControllerSet(ctx context.Context, evt event.Event, payload session.CharacterControllerSetPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.CharacterControllers = upsertSessionCharacterController(
		current.CharacterControllers,
		storage.SessionCharacterController{
			CharacterID:   strings.TrimSpace(payload.CharacterID.String()),
			ParticipantID: strings.TrimSpace(payload.ParticipantID.String()),
		},
	)
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}
