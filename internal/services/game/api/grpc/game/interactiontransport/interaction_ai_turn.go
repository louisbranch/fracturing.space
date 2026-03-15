package interactiontransport

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a interactionApplication) RetryAIGMTurn(ctx context.Context, campaignID string, _ *campaignv1.RetryAIGMTurnRequest) (*campaignv1.InteractionState, error) {
	campaignRecord, err := a.requireManageSessions(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, err
	}
	activeSession, sessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if sessionInteraction.AITurn.Status != session.AITurnStatusFailed || strings.TrimSpace(sessionInteraction.AITurn.TurnToken) == "" {
		return nil, status.Error(codes.FailedPrecondition, "session does not have a failed ai gm turn")
	}
	eligible, err := a.aiTurnEligibility(ctx, campaignRecord, activeSession, sessionInteraction, sessionInteraction.AITurn.SourceEventType)
	if err != nil {
		return nil, err
	}
	if !eligible.ok {
		return nil, status.Error(codes.FailedPrecondition, eligible.reason)
	}
	payload := session.AITurnQueuedPayload{
		SessionID:          ids.SessionID(activeSession.ID),
		TurnToken:          sessionInteraction.AITurn.TurnToken,
		OwnerParticipantID: ids.ParticipantID(sessionInteraction.AITurn.OwnerParticipantID),
		SourceEventType:    sessionInteraction.AITurn.SourceEventType,
		SourceSceneID:      ids.SceneID(sessionInteraction.AITurn.SourceSceneID),
		SourcePhaseID:      sessionInteraction.AITurn.SourcePhaseID,
	}
	if err := a.executeSessionCommand(ctx, commandTypeSessionAITurnQueue, campaignID, activeSession.ID, payload, "session.ai_turn.queue"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

type aiTurnEligibilityResult struct {
	ok               bool
	reason           string
	ownerParticipant storage.ParticipantRecord
}

func (a interactionApplication) clearAITurnIfPresent(ctx context.Context, campaignID, sessionID string, interaction storage.SessionInteraction, reason string) error {
	if interaction.AITurn.Status == session.AITurnStatusIdle && strings.TrimSpace(interaction.AITurn.TurnToken) == "" {
		return nil
	}
	payload := session.AITurnClearedPayload{
		SessionID: ids.SessionID(sessionID),
		TurnToken: interaction.AITurn.TurnToken,
		Reason:    strings.TrimSpace(reason),
	}
	return a.executeSessionCommand(ctx, commandTypeSessionAITurnClear, campaignID, sessionID, payload, "session.ai_turn.clear")
}

// shouldPreserveAITurnForSceneActivation keeps the owning AI GM turn alive while
// that same GM activates the scene it intends to narrate during bootstrap.
func shouldPreserveAITurnForSceneActivation(ctx context.Context, interaction storage.SessionInteraction) bool {
	actorID, actorType := handler.ResolveCommandActor(ctx)
	if actorType != command.ActorTypeParticipant {
		return false
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return false
	}
	if actorID != strings.TrimSpace(interaction.GMAuthorityParticipantID) {
		return false
	}
	if actorID != strings.TrimSpace(interaction.AITurn.OwnerParticipantID) {
		return false
	}
	if strings.TrimSpace(interaction.AITurn.TurnToken) == "" {
		return false
	}
	switch interaction.AITurn.Status {
	case session.AITurnStatusQueued, session.AITurnStatusRunning:
		return true
	default:
		return false
	}
}

func (a interactionApplication) aiTurnEligibility(
	ctx context.Context,
	campaignRecord storage.CampaignRecord,
	activeSession storage.SessionRecord,
	sessionInteraction storage.SessionInteraction,
	sourceEventType string,
) (aiTurnEligibilityResult, error) {
	bootstrap := strings.TrimSpace(sourceEventType) == string(session.EventTypeStarted)
	if campaignRecord.GmMode != campaign.GmModeAI && campaignRecord.GmMode != campaign.GmModeHybrid {
		return aiTurnEligibilityResult{reason: "campaign gm mode does not support ai orchestration"}, nil
	}
	if strings.TrimSpace(campaignRecord.AIAgentID) == "" {
		return aiTurnEligibilityResult{reason: "campaign ai binding is required"}, nil
	}
	if strings.TrimSpace(activeSession.ID) == "" {
		return aiTurnEligibilityResult{reason: "campaign has no active session"}, nil
	}
	if sessionInteraction.OOCPaused {
		return aiTurnEligibilityResult{reason: "session is paused for out-of-character discussion"}, nil
	}
	if strings.TrimSpace(sessionInteraction.ActiveSceneID) == "" && !bootstrap {
		return aiTurnEligibilityResult{reason: "session has no active scene"}, nil
	}
	if strings.TrimSpace(sessionInteraction.GMAuthorityParticipantID) == "" {
		return aiTurnEligibilityResult{reason: "session gm authority is not assigned"}, nil
	}
	owner, err := a.stores.Participant.GetParticipant(ctx, campaignRecord.ID, sessionInteraction.GMAuthorityParticipantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aiTurnEligibilityResult{reason: "gm authority participant was not found"}, nil
		}
		return aiTurnEligibilityResult{}, grpcerror.Internal("load gm authority participant", err)
	}
	if owner.Role != participant.RoleGM {
		return aiTurnEligibilityResult{reason: "gm authority participant is not a gm"}, nil
	}
	if owner.Controller != participant.ControllerAI {
		return aiTurnEligibilityResult{reason: "gm authority participant is not ai-controlled"}, nil
	}
	if strings.TrimSpace(sessionInteraction.ActiveSceneID) == "" {
		return aiTurnEligibilityResult{ok: true, ownerParticipant: owner}, nil
	}
	sceneInteraction, err := a.stores.SceneInteraction.GetSceneInteraction(ctx, campaignRecord.ID, sessionInteraction.ActiveSceneID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return aiTurnEligibilityResult{}, grpcerror.Internal("load active scene interaction", err)
	}
	if err == nil && sceneInteraction.PhaseOpen && strings.TrimSpace(sceneInteraction.PhaseID) != "" && sceneInteraction.PhaseStatus != scene.PlayerPhaseStatusGMReview {
		return aiTurnEligibilityResult{reason: "scene player phase is open"}, nil
	}
	return aiTurnEligibilityResult{ok: true, ownerParticipant: owner}, nil
}
