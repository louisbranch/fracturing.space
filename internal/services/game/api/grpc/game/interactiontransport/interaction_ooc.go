package interactiontransport

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a interactionApplication) PauseSessionForOOC(ctx context.Context, campaignID string, in *campaignv1.PauseSessionForOOCRequest) (*campaignv1.InteractionState, error) {
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
	if sessionInteraction.ActiveSceneID != "" {
		if err := a.endScenePhaseIfOpen(ctx, campaignID, sessionInteraction.ActiveSceneID, "ooc_paused"); err != nil {
			return nil, err
		}
	}
	if err := a.clearAITurnIfPresent(ctx, campaignID, activeSession.ID, sessionInteraction, "ooc_paused"); err != nil {
		return nil, err
	}
	payload := session.OOCPausedPayload{
		SessionID: ids.SessionID(activeSession.ID),
		Reason:    strings.TrimSpace(in.GetReason()),
	}
	if err := a.executeSessionCommand(ctx, commandTypeSessionOOCPause, campaignID, activeSession.ID, payload, "session.ooc.pause"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) PostSessionOOC(ctx context.Context, campaignID string, in *campaignv1.PostSessionOOCRequest) (*campaignv1.InteractionState, error) {
	_, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	activeSession, sessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if !sessionInteraction.OOCPaused {
		return nil, status.Error(codes.FailedPrecondition, "session is not paused for out-of-character discussion")
	}
	postID, err := a.idGenerator()
	if err != nil {
		return nil, grpcerror.Internal("generate ooc post id", err)
	}
	payload := session.OOCPostedPayload{
		SessionID:     ids.SessionID(activeSession.ID),
		PostID:        postID,
		ParticipantID: ids.ParticipantID(actor.ID),
		Body:          strings.TrimSpace(in.GetBody()),
	}
	if err := a.executeSessionCommand(ctx, commandTypeSessionOOCPost, campaignID, activeSession.ID, payload, "session.ooc.post"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) MarkOOCReadyToResume(ctx context.Context, campaignID string, _ *campaignv1.MarkOOCReadyToResumeRequest) (*campaignv1.InteractionState, error) {
	return a.toggleOOCReady(ctx, campaignID, true)
}

func (a interactionApplication) ClearOOCReadyToResume(ctx context.Context, campaignID string, _ *campaignv1.ClearOOCReadyToResumeRequest) (*campaignv1.InteractionState, error) {
	return a.toggleOOCReady(ctx, campaignID, false)
}

func (a interactionApplication) ResumeFromOOC(ctx context.Context, campaignID string, _ *campaignv1.ResumeFromOOCRequest) (*campaignv1.InteractionState, error) {
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
	if !sessionInteraction.OOCPaused {
		return nil, status.Error(codes.FailedPrecondition, "session is not paused for out-of-character discussion")
	}
	payload := session.OOCResumedPayload{SessionID: ids.SessionID(activeSession.ID)}
	if err := a.executeSessionCommand(ctx, commandTypeSessionOOCResume, campaignID, activeSession.ID, payload, "session.ooc.resume"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) toggleOOCReady(ctx context.Context, campaignID string, ready bool) (*campaignv1.InteractionState, error) {
	_, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	activeSession, sessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if !sessionInteraction.OOCPaused {
		return nil, status.Error(codes.FailedPrecondition, "session is not paused for out-of-character discussion")
	}
	if ready {
		payload := session.OOCReadyMarkedPayload{
			SessionID:     ids.SessionID(activeSession.ID),
			ParticipantID: ids.ParticipantID(actor.ID),
		}
		if err := a.executeSessionCommand(ctx, commandTypeSessionOOCReadyMark, campaignID, activeSession.ID, payload, "session.ooc.ready_mark"); err != nil {
			return nil, err
		}
	} else {
		payload := session.OOCReadyClearedPayload{
			SessionID:     ids.SessionID(activeSession.ID),
			ParticipantID: ids.ParticipantID(actor.ID),
		}
		if err := a.executeSessionCommand(ctx, commandTypeSessionOOCReadyClear, campaignID, activeSession.ID, payload, "session.ooc.ready_clear"); err != nil {
			return nil, err
		}
	}
	return a.GetInteractionState(ctx, campaignID)
}
