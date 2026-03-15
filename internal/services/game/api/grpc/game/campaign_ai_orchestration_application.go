package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type campaignAIOrchestrationApplication struct {
	interaction interactionApplication
}

func newCampaignAIOrchestrationApplicationWithDependencies(
	stores Stores,
	idGenerator func() (string, error),
) campaignAIOrchestrationApplication {
	return campaignAIOrchestrationApplication{
		interaction: newInteractionApplicationWithDependencies(stores, idGenerator),
	}
}

func (a campaignAIOrchestrationApplication) QueueAIGMTurn(
	ctx context.Context,
	campaignID, sessionID, sourceEventType, sourceSceneID, sourcePhaseID string,
) (*campaignv1.AITurnState, error) {
	campaignRecord, err := a.interaction.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	activeSession, sessionInteraction, err := a.interaction.loadActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if activeSession == nil || strings.TrimSpace(activeSession.ID) != strings.TrimSpace(sessionID) {
		return aiTurnToProto(storage.SessionAITurn{Status: session.AITurnStatusIdle}), nil
	}
	eligible, err := a.interaction.aiTurnEligibility(ctx, campaignRecord, *activeSession, sessionInteraction, sourceEventType)
	if err != nil {
		return nil, err
	}
	if !eligible.ok {
		if err := a.interaction.clearAITurnIfPresent(ctx, campaignID, activeSession.ID, sessionInteraction, "ineligible_state"); err != nil {
			return nil, err
		}
		return aiTurnToProto(storage.SessionAITurn{Status: session.AITurnStatusIdle}), nil
	}
	turnToken := aiTurnToken(activeSession.ID, eligible.ownerParticipant.ID, sourceEventType, sourceSceneID, sourcePhaseID)
	current := sessionInteraction.AITurn
	if (current.Status == session.AITurnStatusQueued || current.Status == session.AITurnStatusRunning || current.Status == session.AITurnStatusFailed) &&
		strings.TrimSpace(current.TurnToken) == turnToken {
		return aiTurnToProto(current), nil
	}
	if err := a.interaction.clearAITurnIfPresent(ctx, campaignID, activeSession.ID, sessionInteraction, "superseded"); err != nil {
		return nil, err
	}
	payload := session.AITurnQueuedPayload{
		SessionID:          ids.SessionID(activeSession.ID),
		TurnToken:          turnToken,
		OwnerParticipantID: ids.ParticipantID(eligible.ownerParticipant.ID),
		SourceEventType:    strings.TrimSpace(sourceEventType),
		SourceSceneID:      ids.SceneID(strings.TrimSpace(sourceSceneID)),
		SourcePhaseID:      strings.TrimSpace(sourcePhaseID),
	}
	if err := a.interaction.executeSessionCommand(ctx, commandTypeSessionAITurnQueue, campaignID, activeSession.ID, payload, "session.ai_turn.queue"); err != nil {
		return nil, err
	}
	updated, err := a.interaction.stores.SessionInteraction.GetSessionInteraction(ctx, campaignID, activeSession.ID)
	if err != nil {
		return nil, grpcerror.Internal("load queued ai turn state", err)
	}
	return aiTurnToProto(updated.AITurn), nil
}

func (a campaignAIOrchestrationApplication) StartAIGMTurn(ctx context.Context, campaignID, sessionID, turnToken string) (*campaignv1.AITurnState, error) {
	payload := session.AITurnRunningPayload{
		SessionID: ids.SessionID(sessionID),
		TurnToken: strings.TrimSpace(turnToken),
	}
	if err := a.interaction.executeSessionCommand(ctx, commandTypeSessionAITurnStart, campaignID, sessionID, payload, "session.ai_turn.start"); err != nil {
		return nil, err
	}
	updated, err := a.interaction.stores.SessionInteraction.GetSessionInteraction(ctx, campaignID, sessionID)
	if err != nil {
		return nil, grpcerror.Internal("load running ai turn state", err)
	}
	return aiTurnToProto(updated.AITurn), nil
}

func (a campaignAIOrchestrationApplication) FailAIGMTurn(ctx context.Context, campaignID, sessionID, turnToken, lastError string) (*campaignv1.AITurnState, error) {
	payload := session.AITurnFailedPayload{
		SessionID: ids.SessionID(sessionID),
		TurnToken: strings.TrimSpace(turnToken),
		LastError: strings.TrimSpace(lastError),
	}
	if err := a.interaction.executeSessionCommand(ctx, commandTypeSessionAITurnFail, campaignID, sessionID, payload, "session.ai_turn.fail"); err != nil {
		return nil, err
	}
	updated, err := a.interaction.stores.SessionInteraction.GetSessionInteraction(ctx, campaignID, sessionID)
	if err != nil {
		return nil, grpcerror.Internal("load failed ai turn state", err)
	}
	return aiTurnToProto(updated.AITurn), nil
}

func (a campaignAIOrchestrationApplication) CompleteAIGMTurn(ctx context.Context, campaignID, sessionID, turnToken string) (*campaignv1.AITurnState, error) {
	payload := session.AITurnClearedPayload{
		SessionID: ids.SessionID(sessionID),
		TurnToken: strings.TrimSpace(turnToken),
		Reason:    "completed",
	}
	if err := a.interaction.executeSessionCommand(ctx, commandTypeSessionAITurnClear, campaignID, sessionID, payload, "session.ai_turn.clear"); err != nil {
		return nil, err
	}
	return aiTurnToProto(storage.SessionAITurn{Status: session.AITurnStatusIdle}), nil
}

func (a campaignAIOrchestrationApplication) campaignSupportsAI(campaignRecord storage.CampaignRecord) bool {
	return campaignRecord.GmMode == campaign.GmModeAI || campaignRecord.GmMode == campaign.GmModeHybrid
}
