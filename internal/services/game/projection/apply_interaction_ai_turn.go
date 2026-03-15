package projection

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applySessionAITurnQueued(ctx context.Context, evt event.Event, payload session.AITurnQueuedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.AITurn = storage.SessionAITurn{
		Status:             session.AITurnStatusQueued,
		TurnToken:          strings.TrimSpace(payload.TurnToken),
		OwnerParticipantID: strings.TrimSpace(payload.OwnerParticipantID.String()),
		SourceEventType:    strings.TrimSpace(payload.SourceEventType),
		SourceSceneID:      strings.TrimSpace(payload.SourceSceneID.String()),
		SourcePhaseID:      strings.TrimSpace(payload.SourcePhaseID),
	}
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionAITurnRunning(ctx context.Context, evt event.Event, payload session.AITurnRunningPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.AITurn.Status = session.AITurnStatusRunning
	current.AITurn.TurnToken = strings.TrimSpace(payload.TurnToken)
	current.AITurn.LastError = ""
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionAITurnFailed(ctx context.Context, evt event.Event, payload session.AITurnFailedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.AITurn.Status = session.AITurnStatusFailed
	current.AITurn.TurnToken = strings.TrimSpace(payload.TurnToken)
	current.AITurn.LastError = strings.TrimSpace(payload.LastError)
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionAITurnCleared(ctx context.Context, evt event.Event, _ session.AITurnClearedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.AITurn = storage.SessionAITurn{Status: session.AITurnStatusIdle}
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}
