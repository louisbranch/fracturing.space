package interactiontransport

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a interactionApplication) PauseSessionForOOC(ctx context.Context, campaignID string, in *campaignv1.PauseSessionForOOCRequest) (*campaignv1.InteractionState, error) {
	campaignRecord, actor, err := a.loadViewerCampaign(ctx, campaignID)
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
	payload := session.OOCPausedPayload{
		SessionID:                ids.SessionID(activeSession.ID),
		RequestedByParticipantID: ids.ParticipantID(actor.ID),
		Reason:                   strings.TrimSpace(in.GetReason()),
	}
	if err := a.clearAITurnIfPresent(ctx, campaignID, activeSession.ID, sessionInteraction, "ooc_paused"); err != nil {
		return nil, err
	}
	if sessionInteraction.ActiveSceneID != "" && a.stores.SceneInteraction != nil {
		sceneInteraction, err := a.stores.SceneInteraction.GetSceneInteraction(ctx, campaignID, sessionInteraction.ActiveSceneID)
		if err == nil && sceneInteraction.PhaseOpen && strings.TrimSpace(sceneInteraction.PhaseID) != "" {
			payload.InterruptedSceneID = ids.SceneID(sessionInteraction.ActiveSceneID)
			payload.InterruptedPhaseID = sceneInteraction.PhaseID
			payload.InterruptedPhaseStatus = string(sceneInteraction.PhaseStatus)
		}
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

func (a interactionApplication) ResolveInterruptedScenePhase(ctx context.Context, campaignID string, in *campaignv1.ResolveInterruptedScenePhaseRequest) (*campaignv1.InteractionState, error) {
	campaignRecord, actor, err := a.loadViewerCampaign(ctx, campaignID)
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
	if sessionInteraction.OOCPaused {
		return nil, status.Error(codes.FailedPrecondition, errSessionOOCPaused)
	}
	if !sessionInteraction.OOCResolutionPending {
		return nil, status.Error(codes.FailedPrecondition, "session is not waiting for post-ooc resolution")
	}
	if err := requireAuthoritativeGMActor(actor, sessionInteraction); err != nil {
		return nil, err
	}
	interruptedSceneID := strings.TrimSpace(sessionInteraction.OOCInterruptedSceneID)
	if interruptedSceneID == "" {
		return nil, status.Error(codes.FailedPrecondition, "ooc interruption context is missing the interrupted scene")
	}

	switch resolution := in.GetResolution().(type) {
	case *campaignv1.ResolveInterruptedScenePhaseRequest_ResumeOriginalPhase:
		if _, _, err := a.requireActiveScenePhase(ctx, campaignID, activeSession.ID, interruptedSceneID, sessionInteraction); err != nil {
			return nil, err
		}
		if err := a.clearOOCResolutionIfPending(ctx, campaignID, activeSession.ID, sessionInteraction, "resume_original_phase"); err != nil {
			return nil, err
		}
	case *campaignv1.ResolveInterruptedScenePhaseRequest_ReplaceWithPlayerPhase:
		replace := resolution.ReplaceWithPlayerPhase
		if replace == nil {
			return nil, status.Error(codes.InvalidArgument, "replace_with_player_phase is required")
		}
		targetSceneID := strings.TrimSpace(replace.GetSceneId())
		if targetSceneID == "" {
			targetSceneID = interruptedSceneID
		}
		if strings.TrimSpace(sessionInteraction.ActiveSceneID) != targetSceneID {
			targetScene, err := a.stores.Scene.GetScene(ctx, campaignID, targetSceneID)
			if err != nil {
				return nil, err
			}
			if targetScene.SessionID != activeSession.ID {
				return nil, status.Error(codes.FailedPrecondition, "scene is not in the active session")
			}
			if strings.TrimSpace(sessionInteraction.ActiveSceneID) != "" {
				if err := a.endScenePhaseIfOpen(ctx, campaignID, sessionInteraction.ActiveSceneID, "ooc_replaced"); err != nil {
					return nil, err
				}
			}
			payload := session.ActiveSceneSetPayload{
				SessionID:     ids.SessionID(activeSession.ID),
				ActiveSceneID: ids.SceneID(targetSceneID),
			}
			if err := a.executeSessionCommand(ctx, commandTypeSessionActiveSceneSet, campaignID, activeSession.ID, payload, "session.active_scene.set"); err != nil {
				return nil, err
			}
			sessionInteraction.ActiveSceneID = targetSceneID
		} else if strings.TrimSpace(sessionInteraction.OOCInterruptedPhaseID) != "" {
			if err := a.endScenePhase(ctx, campaignID, activeSession.ID, targetSceneID, sessionInteraction.OOCInterruptedPhaseID, "ooc_replaced"); err != nil {
				return nil, err
			}
		}

		sceneRecord, err := a.stores.Scene.GetScene(ctx, campaignID, targetSceneID)
		if err != nil {
			return nil, err
		}
		actingCharacterIDs, actingParticipantIDs, err := a.resolveActingSet(ctx, campaignID, sceneRecord, replace.GetNextCharacterIds())
		if err != nil {
			return nil, err
		}
		phaseID, err := a.idGenerator()
		if err != nil {
			return nil, grpcerror.Internal("generate scene phase id", err)
		}
		interactionPayload, err := a.buildGMInteractionPayload(replace.GetInteraction(), ids.SceneID(targetSceneID), phaseID, ids.ParticipantID(actor.ID))
		if err != nil {
			return nil, err
		}
		if err := a.executeSceneCommand(ctx, commandTypeSceneGMInteractionCommit, campaignID, activeSession.ID, targetSceneID, interactionPayload, "scene.gm_interaction.commit"); err != nil {
			return nil, err
		}
		payload := scene.PlayerPhaseStartedPayload{
			SceneID:              ids.SceneID(targetSceneID),
			PhaseID:              phaseID,
			ActingCharacterIDs:   actingCharacterIDs,
			ActingParticipantIDs: actingParticipantIDs,
		}
		if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseStart, campaignID, activeSession.ID, targetSceneID, payload, "scene.player_phase.start"); err != nil {
			return nil, err
		}
		if err := a.clearOOCResolutionIfPending(ctx, campaignID, activeSession.ID, sessionInteraction, "replace_with_player_phase"); err != nil {
			return nil, err
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "interrupted scene resolution is required")
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
