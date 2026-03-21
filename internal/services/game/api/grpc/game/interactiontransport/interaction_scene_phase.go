package interactiontransport

import (
	"context"
	"slices"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a interactionApplication) StartScenePlayerPhase(ctx context.Context, campaignID string, in *campaignv1.StartScenePlayerPhaseRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
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
	if err := requireSceneWritesUnblocked(sessionInteraction); err != nil {
		return nil, err
	}
	if strings.TrimSpace(sessionInteraction.ActiveSceneID) != sceneID {
		return nil, status.Error(codes.FailedPrecondition, "scene is not the active scene")
	}
	sceneRecord, err := a.stores.Scene.GetScene(ctx, campaignID, sceneID)
	if err != nil {
		return nil, err
	}
	actingCharacterIDs, actingParticipantIDs, err := a.resolveActingSet(ctx, campaignID, sceneRecord, in.GetCharacterIds())
	if err != nil {
		return nil, err
	}
	phaseID, err := a.idGenerator()
	if err != nil {
		return nil, grpcerror.Internal("generate scene phase id", err)
	}
	if err := a.clearAITurnIfPresent(ctx, campaignID, activeSession.ID, sessionInteraction, "gm_frame_started"); err != nil {
		return nil, err
	}
	payload := scene.PlayerPhaseStartedPayload{
		SceneID:              ids.SceneID(sceneID),
		PhaseID:              phaseID,
		FrameText:            strings.TrimSpace(in.GetFrameText()),
		ActingCharacterIDs:   actingCharacterIDs,
		ActingParticipantIDs: actingParticipantIDs,
	}
	if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseStart, campaignID, activeSession.ID, sceneID, payload, "scene.player_phase.start"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) SubmitScenePlayerPost(ctx context.Context, campaignID string, in *campaignv1.SubmitScenePlayerPostRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	_, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	activeSession, sessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := requireSceneWritesUnblocked(sessionInteraction); err != nil {
		return nil, err
	}
	sceneRecord, sceneInteraction, err := a.requireActiveScenePhase(ctx, campaignID, activeSession.ID, sceneID, sessionInteraction)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(sceneInteraction.ActingParticipantIDs, actor.ID) {
		return nil, status.Error(codes.PermissionDenied, "participant is not acting in the current scene phase")
	}
	characterIDs, err := a.resolveParticipantPostCharacters(ctx, campaignID, sceneRecord, actor.ID, in.GetCharacterIds(), sceneInteraction.ActingCharacterIDs)
	if err != nil {
		return nil, err
	}
	payload := scene.PlayerPhasePostedPayload{
		SceneID:       ids.SceneID(sceneID),
		PhaseID:       sceneInteraction.PhaseID,
		ParticipantID: ids.ParticipantID(actor.ID),
		CharacterIDs:  characterIDs,
		SummaryText:   strings.TrimSpace(in.GetSummaryText()),
	}
	if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhasePost, campaignID, activeSession.ID, sceneID, payload, "scene.player_phase.post"); err != nil {
		return nil, err
	}
	if in.GetYieldAfterPost() {
		if err := a.yieldScenePhase(ctx, campaignID, activeSession.ID, sceneID, sceneInteraction.PhaseID, actor.ID); err != nil {
			return nil, err
		}
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) YieldScenePlayerPhase(ctx context.Context, campaignID string, in *campaignv1.YieldScenePlayerPhaseRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	_, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	activeSession, currentSessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := requireSceneWritesUnblocked(currentSessionInteraction); err != nil {
		return nil, err
	}
	_, currentSceneInteraction, err := a.requireActiveScenePhase(ctx, campaignID, activeSession.ID, sceneID, currentSessionInteraction)
	if err != nil {
		return nil, err
	}
	if err := a.yieldScenePhase(ctx, campaignID, activeSession.ID, sceneID, currentSceneInteraction.PhaseID, actor.ID); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) UnyieldScenePlayerPhase(ctx context.Context, campaignID string, in *campaignv1.UnyieldScenePlayerPhaseRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	_, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	activeSession, currentSessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := requireSceneWritesUnblocked(currentSessionInteraction); err != nil {
		return nil, err
	}
	_, currentSceneInteraction, err := a.requireActiveScenePhase(ctx, campaignID, activeSession.ID, sceneID, currentSessionInteraction)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(currentSceneInteraction.ActingParticipantIDs, actor.ID) {
		return nil, status.Error(codes.PermissionDenied, "participant is not acting in the current scene phase")
	}
	payload := scene.PlayerPhaseUnyieldedPayload{
		SceneID:       ids.SceneID(sceneID),
		PhaseID:       currentSceneInteraction.PhaseID,
		ParticipantID: ids.ParticipantID(actor.ID),
	}
	if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseUnyield, campaignID, activeSession.ID, sceneID, payload, "scene.player_phase.unyield"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) EndScenePlayerPhase(ctx context.Context, campaignID string, in *campaignv1.EndScenePlayerPhaseRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	campaignRecord, err := a.requireManageSessions(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, err
	}
	activeSession, currentSessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := requireSceneWritesUnblocked(currentSessionInteraction); err != nil {
		return nil, err
	}
	_, currentSceneInteraction, err := a.requireActiveScenePhase(ctx, campaignID, activeSession.ID, sceneID, currentSessionInteraction)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(in.GetReason())
	if reason == "" {
		reason = "gm_interrupted"
	}
	if err := a.endScenePhase(ctx, campaignID, activeSession.ID, sceneID, currentSceneInteraction.PhaseID, reason); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) CommitSceneGMOutput(ctx context.Context, campaignID string, in *campaignv1.CommitSceneGMOutputRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(in.GetText())
	if text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}
	campaignRecord, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, err
	}
	activeSession, currentSessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := requireSceneWritesUnblocked(currentSessionInteraction); err != nil {
		return nil, err
	}
	if err := requireAuthoritativeGMActor(actor, currentSessionInteraction); err != nil {
		return nil, err
	}
	_, currentSceneInteraction, err := a.requireActiveSceneForGM(ctx, campaignID, activeSession.ID, sceneID, currentSessionInteraction)
	if err != nil {
		return nil, err
	}
	if currentSceneInteraction.PhaseOpen && currentSceneInteraction.PhaseStatus == scene.PlayerPhaseStatusPlayers {
		return nil, status.Error(codes.FailedPrecondition, "scene player phase is open")
	}
	payload := scene.GMOutputCommittedPayload{
		SceneID:       ids.SceneID(sceneID),
		ParticipantID: ids.ParticipantID(actor.ID),
		Text:          text,
	}
	if err := a.executeSceneCommand(ctx, commandTypeSceneGMOutputCommit, campaignID, activeSession.ID, sceneID, payload, "scene.gm_output.commit"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) AcceptScenePlayerPhase(ctx context.Context, campaignID string, in *campaignv1.AcceptScenePlayerPhaseRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	campaignRecord, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, err
	}
	activeSession, currentSessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := requireSceneWritesUnblocked(currentSessionInteraction); err != nil {
		return nil, err
	}
	if err := requireAuthoritativeGMActor(actor, currentSessionInteraction); err != nil {
		return nil, err
	}
	_, currentSceneInteraction, err := a.requireActiveScenePhase(ctx, campaignID, activeSession.ID, sceneID, currentSessionInteraction)
	if err != nil {
		return nil, err
	}
	payload := scene.PlayerPhaseAcceptedPayload{
		SceneID: ids.SceneID(sceneID),
		PhaseID: currentSceneInteraction.PhaseID,
	}
	if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseAccept, campaignID, activeSession.ID, sceneID, payload, "scene.player_phase.accept"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) RequestScenePlayerRevisions(ctx context.Context, campaignID string, in *campaignv1.RequestScenePlayerRevisionsRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	campaignRecord, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, err
	}
	activeSession, currentSessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := requireSceneWritesUnblocked(currentSessionInteraction); err != nil {
		return nil, err
	}
	if err := requireAuthoritativeGMActor(actor, currentSessionInteraction); err != nil {
		return nil, err
	}
	sceneRecord, currentSceneInteraction, err := a.requireActiveScenePhase(ctx, campaignID, activeSession.ID, sceneID, currentSessionInteraction)
	if err != nil {
		return nil, err
	}
	revisions, err := a.resolveRevisionRequests(ctx, campaignID, sceneRecord, currentSceneInteraction, in.GetRevisions())
	if err != nil {
		return nil, err
	}
	payload := scene.PlayerPhaseRevisionsRequestedPayload{
		SceneID:   ids.SceneID(sceneID),
		PhaseID:   currentSceneInteraction.PhaseID,
		Revisions: revisions,
	}
	if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseRequestRevisions, campaignID, activeSession.ID, sceneID, payload, "scene.player_phase.request_revisions"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

func (a interactionApplication) ResolveScenePlayerPhaseReview(ctx context.Context, campaignID string, in *campaignv1.ResolveScenePlayerPhaseReviewRequest) (*campaignv1.InteractionState, error) {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	campaignRecord, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, err
	}
	activeSession, currentSessionInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if currentSessionInteraction.OOCPaused {
		return nil, status.Error(codes.FailedPrecondition, errSessionOOCPaused)
	}
	if err := requireAuthoritativeGMActor(actor, currentSessionInteraction); err != nil {
		return nil, err
	}
	sceneRecord, currentSceneInteraction, err := a.requireActiveScenePhase(ctx, campaignID, activeSession.ID, sceneID, currentSessionInteraction)
	if err != nil {
		return nil, err
	}
	if currentSceneInteraction.PhaseStatus != scene.PlayerPhaseStatusGMReview {
		return nil, status.Error(codes.FailedPrecondition, "scene player phase is not waiting for gm review")
	}

	switch resolution := in.GetResolution().(type) {
	case *campaignv1.ResolveScenePlayerPhaseReviewRequest_AdvanceToPlayers:
		advance := resolution.AdvanceToPlayers
		if advance == nil {
			return nil, status.Error(codes.InvalidArgument, "advance_to_players is required")
		}
		gmOutputText := strings.TrimSpace(advance.GetGmOutputText())
		if gmOutputText == "" {
			return nil, status.Error(codes.InvalidArgument, "gm_output_text is required")
		}
		actingCharacterIDs, actingParticipantIDs, err := a.resolveActingSet(ctx, campaignID, sceneRecord, advance.GetNextCharacterIds())
		if err != nil {
			return nil, err
		}
		commitPayload := scene.GMOutputCommittedPayload{
			SceneID:       ids.SceneID(sceneID),
			ParticipantID: ids.ParticipantID(actor.ID),
			Text:          gmOutputText,
		}
		if err := a.executeSceneCommand(ctx, commandTypeSceneGMOutputCommit, campaignID, activeSession.ID, sceneID, commitPayload, "scene.gm_output.commit"); err != nil {
			return nil, err
		}
		acceptPayload := scene.PlayerPhaseAcceptedPayload{
			SceneID: ids.SceneID(sceneID),
			PhaseID: currentSceneInteraction.PhaseID,
		}
		if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseAccept, campaignID, activeSession.ID, sceneID, acceptPayload, "scene.player_phase.accept"); err != nil {
			return nil, err
		}
		phaseID, err := a.idGenerator()
		if err != nil {
			return nil, grpcerror.Internal("generate scene phase id", err)
		}
		startPayload := scene.PlayerPhaseStartedPayload{
			SceneID:              ids.SceneID(sceneID),
			PhaseID:              phaseID,
			FrameText:            strings.TrimSpace(advance.GetNextFrameText()),
			ActingCharacterIDs:   actingCharacterIDs,
			ActingParticipantIDs: actingParticipantIDs,
		}
		if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseStart, campaignID, activeSession.ID, sceneID, startPayload, "scene.player_phase.start"); err != nil {
			return nil, err
		}
		if err := a.clearOOCResolutionIfPending(ctx, campaignID, activeSession.ID, currentSessionInteraction, "review_advanced_to_players"); err != nil {
			return nil, err
		}
	case *campaignv1.ResolveScenePlayerPhaseReviewRequest_RequestRevisions:
		request := resolution.RequestRevisions
		if request == nil {
			return nil, status.Error(codes.InvalidArgument, "request_revisions is required")
		}
		revisions, err := a.resolveRevisionRequests(ctx, campaignID, sceneRecord, currentSceneInteraction, request.GetRevisions())
		if err != nil {
			return nil, err
		}
		payload := scene.PlayerPhaseRevisionsRequestedPayload{
			SceneID:   ids.SceneID(sceneID),
			PhaseID:   currentSceneInteraction.PhaseID,
			Revisions: revisions,
		}
		if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseRequestRevisions, campaignID, activeSession.ID, sceneID, payload, "scene.player_phase.request_revisions"); err != nil {
			return nil, err
		}
		if err := a.clearOOCResolutionIfPending(ctx, campaignID, activeSession.ID, currentSessionInteraction, "review_requested_revisions"); err != nil {
			return nil, err
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "review resolution is required")
	}

	return a.GetInteractionState(ctx, campaignID)
}
