package interactiontransport

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Deps groups the dependencies for the interaction transport surface.
type Deps struct {
	Auth               authz.PolicyDeps
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	Character          storage.CharacterStore
	Session            storage.SessionStore
	SessionInteraction storage.SessionInteractionStore
	Scene              storage.SceneStore
	SceneCharacter     storage.SceneCharacterStore
	SceneInteraction   storage.SceneInteractionStore
	Write              domainwriteexec.WritePath
	Applier            projection.Applier
}

// interactionApplication coordinates the scene-phase interaction service over
// projection-backed state plus explicit domain writes.
type interactionApplication struct {
	auth        authz.PolicyDeps
	stores      interactionApplicationStores
	write       domainwriteexec.WritePath
	applier     projection.Applier
	idGenerator func() (string, error)
}

type interactionApplicationStores struct {
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	Character          storage.CharacterStore
	Session            storage.SessionStore
	SessionInteraction storage.SessionInteractionStore
	Scene              storage.SceneStore
	SceneCharacter     storage.SceneCharacterStore
	SceneInteraction   storage.SceneInteractionStore
}

func newInteractionApplicationWithDependencies(
	deps Deps,
	idGenerator func() (string, error),
) interactionApplication {
	return interactionApplication{
		auth: deps.Auth,
		stores: interactionApplicationStores{
			Campaign:           deps.Campaign,
			Participant:        deps.Participant,
			Character:          deps.Character,
			Session:            deps.Session,
			SessionInteraction: deps.SessionInteraction,
			Scene:              deps.Scene,
			SceneCharacter:     deps.SceneCharacter,
			SceneInteraction:   deps.SceneInteraction,
		},
		write:       deps.Write,
		applier:     deps.Applier,
		idGenerator: idGenerator,
	}
}

func (a interactionApplication) GetInteractionState(ctx context.Context, campaignID string) (*campaignv1.InteractionState, error) {
	campaignRecord, actor, err := a.loadViewerCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	state := &campaignv1.InteractionState{
		CampaignId:   campaignRecord.ID,
		CampaignName: campaignRecord.Name,
		Locale:       campaigntransport.LocaleStringToProto(campaignRecord.Locale),
		Viewer: &campaignv1.InteractionViewer{
			ParticipantId: actor.ID,
			Name:          actor.Name,
			Role:          participanttransport.RoleToProto(actor.Role),
		},
		Ooc: &campaignv1.OOCState{
			Posts:                       []*campaignv1.OOCPost{},
			ReadyToResumeParticipantIds: []string{},
		},
		AiTurn: &campaignv1.AITurnState{
			Status: campaignv1.AITurnStatus_AI_TURN_STATUS_IDLE,
		},
	}

	activeSession, sessionInteraction, err := a.loadActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if activeSession == nil {
		return state, nil
	}

	state.ActiveSession = &campaignv1.InteractionSession{
		SessionId: activeSession.ID,
		Name:      activeSession.Name,
	}
	state.Ooc = sessionInteractionToProto(sessionInteraction)
	state.GmAuthorityParticipantId = sessionInteraction.GMAuthorityParticipantID
	state.AiTurn = aiTurnToProto(sessionInteraction.AITurn)

	if strings.TrimSpace(sessionInteraction.ActiveSceneID) == "" {
		return state, nil
	}

	sceneRecord, err := a.stores.Scene.GetScene(ctx, campaignID, sessionInteraction.ActiveSceneID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return state, nil
		}
		return nil, grpcerror.Internal("load active scene", err)
	}
	activeScene, sceneInteraction, err := a.loadSceneState(ctx, campaignID, sceneRecord)
	if err != nil {
		return nil, err
	}
	state.ActiveScene = activeScene
	state.PlayerPhase = sceneInteractionToProto(sceneInteraction)
	return state, nil
}

func (a interactionApplication) SetActiveScene(ctx context.Context, campaignID string, in *campaignv1.SetActiveSceneRequest) (*campaignv1.InteractionState, error) {
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
	activeSession, currentInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	targetScene, err := a.stores.Scene.GetScene(ctx, campaignID, sceneID)
	if err != nil {
		return nil, err
	}
	if targetScene.SessionID != activeSession.ID {
		return nil, status.Error(codes.FailedPrecondition, "scene is not in the active session")
	}
	if currentInteraction.ActiveSceneID != "" && currentInteraction.ActiveSceneID != sceneID {
		if err := a.endScenePhaseIfOpen(ctx, campaignID, currentInteraction.ActiveSceneID, "active_scene_switched"); err != nil {
			return nil, err
		}
	}
	if !shouldPreserveAITurnForSceneActivation(ctx, currentInteraction) {
		if err := a.clearAITurnIfPresent(ctx, campaignID, activeSession.ID, currentInteraction, "active_scene_switched"); err != nil {
			return nil, err
		}
	}

	payload := session.ActiveSceneSetPayload{
		SessionID:     ids.SessionID(activeSession.ID),
		ActiveSceneID: ids.SceneID(sceneID),
	}
	if err := a.executeSessionCommand(ctx, commandTypeSessionActiveSceneSet, campaignID, activeSession.ID, payload, "session.active_scene.set"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

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
	if sessionInteraction.OOCPaused {
		return nil, status.Error(codes.FailedPrecondition, "session is paused for out-of-character discussion")
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
	if sessionInteraction.OOCPaused {
		return nil, status.Error(codes.FailedPrecondition, "session is paused for out-of-character discussion")
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

func (a interactionApplication) SetSessionGMAuthority(ctx context.Context, campaignID string, in *campaignv1.SetSessionGMAuthorityRequest) (*campaignv1.InteractionState, error) {
	participantID, err := validate.RequiredID(in.GetParticipantId(), "participant id")
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
	participants, err := a.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("list campaign participants", err)
	}
	record, ok := findCampaignParticipant(participants, participantID)
	if !ok {
		return nil, status.Error(codes.NotFound, "participant not found")
	}
	if record.Role != participant.RoleGM {
		return nil, status.Error(codes.FailedPrecondition, "participant is not a gm")
	}
	if err := a.clearAITurnIfPresent(ctx, campaignID, activeSession.ID, sessionInteraction, "gm_authority_changed"); err != nil {
		return nil, err
	}
	payload := session.GMAuthoritySetPayload{
		SessionID:     ids.SessionID(activeSession.ID),
		ParticipantID: ids.ParticipantID(record.ID),
	}
	if err := a.executeSessionCommand(ctx, commandTypeSessionGMAuthoritySet, campaignID, activeSession.ID, payload, "session.gm_authority.set"); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}

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

func (a interactionApplication) loadViewerCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, storage.ParticipantRecord, error) {
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, err
	}
	actor, err := authz.RequirePolicyActor(ctx, a.auth, domainauthz.CapabilityReadCampaign(), campaignRecord)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, err
	}
	return campaignRecord, actor, nil
}

func (a interactionApplication) requireManageSessions(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	return campaignRecord, nil
}

func (a interactionApplication) loadActiveSessionInteraction(ctx context.Context, campaignID string) (*storage.SessionRecord, storage.SessionInteraction, error) {
	activeSession, err := a.stores.Session.GetActiveSession(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, storage.SessionInteraction{}, nil
		}
		return nil, storage.SessionInteraction{}, grpcerror.Internal("load active session", err)
	}
	interaction, err := a.stores.SessionInteraction.GetSessionInteraction(ctx, campaignID, activeSession.ID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, storage.SessionInteraction{}, grpcerror.Internal("load session interaction", err)
		}
		interaction = storage.SessionInteraction{
			CampaignID:                  campaignID,
			SessionID:                   activeSession.ID,
			AITurn:                      storage.SessionAITurn{Status: session.AITurnStatusIdle},
			OOCPosts:                    []storage.SessionOOCPost{},
			ReadyToResumeParticipantIDs: []string{},
		}
	}
	return &activeSession, interaction, nil
}

func (a interactionApplication) requireActiveSessionInteraction(ctx context.Context, campaignID string) (storage.SessionRecord, storage.SessionInteraction, error) {
	activeSession, interaction, err := a.loadActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return storage.SessionRecord{}, storage.SessionInteraction{}, err
	}
	if activeSession == nil {
		return storage.SessionRecord{}, storage.SessionInteraction{}, status.Error(codes.FailedPrecondition, "campaign has no active session")
	}
	return *activeSession, interaction, nil
}

func requireAuthoritativeGMActor(actor storage.ParticipantRecord, sessionInteraction storage.SessionInteraction) error {
	gmAuthorityParticipantID := strings.TrimSpace(sessionInteraction.GMAuthorityParticipantID)
	if gmAuthorityParticipantID == "" {
		return status.Error(codes.FailedPrecondition, "session gm authority is not assigned")
	}
	if strings.TrimSpace(actor.ID) != gmAuthorityParticipantID {
		return status.Error(codes.PermissionDenied, "participant does not own gm authority for the active session")
	}
	return nil
}

func (a interactionApplication) loadSceneState(ctx context.Context, campaignID string, sceneRecord storage.SceneRecord) (*campaignv1.InteractionScene, storage.SceneInteraction, error) {
	sceneCharacters, err := a.stores.SceneCharacter.ListSceneCharacters(ctx, campaignID, sceneRecord.SceneID)
	if err != nil {
		return nil, storage.SceneInteraction{}, grpcerror.Internal("list scene characters", err)
	}
	characters := make([]*campaignv1.InteractionCharacter, 0, len(sceneCharacters))
	for _, sceneCharacter := range sceneCharacters {
		characterRecord, err := a.stores.Character.GetCharacter(ctx, campaignID, sceneCharacter.CharacterID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, storage.SceneInteraction{}, grpcerror.Internal("load scene character", err)
		}
		characters = append(characters, &campaignv1.InteractionCharacter{
			CharacterId:        characterRecord.ID,
			Name:               characterRecord.Name,
			OwnerParticipantId: characterRecord.OwnerParticipantID,
		})
	}
	sort.SliceStable(characters, func(i, j int) bool {
		if characters[i].Name == characters[j].Name {
			return characters[i].CharacterId < characters[j].CharacterId
		}
		return characters[i].Name < characters[j].Name
	})

	sceneInteraction, err := a.stores.SceneInteraction.GetSceneInteraction(ctx, campaignID, sceneRecord.SceneID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, storage.SceneInteraction{}, grpcerror.Internal("load scene interaction", err)
		}
		sceneInteraction = storage.SceneInteraction{
			CampaignID:           campaignID,
			SceneID:              sceneRecord.SceneID,
			SessionID:            sceneRecord.SessionID,
			ActingCharacterIDs:   []string{},
			ActingParticipantIDs: []string{},
			Slots:                []storage.ScenePlayerSlot{},
		}
	}

	return &campaignv1.InteractionScene{
		SceneId:     sceneRecord.SceneID,
		SessionId:   sceneRecord.SessionID,
		Name:        sceneRecord.Name,
		Description: sceneRecord.Description,
		Characters:  characters,
		GmOutput:    sceneGMOutputToProto(sceneInteraction),
	}, sceneInteraction, nil
}

func (a interactionApplication) requireActiveSceneForGM(
	ctx context.Context,
	campaignID string,
	activeSessionID string,
	sceneID string,
	sessionInteraction storage.SessionInteraction,
) (storage.SceneRecord, storage.SceneInteraction, error) {
	if sessionInteraction.OOCPaused {
		return storage.SceneRecord{}, storage.SceneInteraction{}, status.Error(codes.FailedPrecondition, "session is paused for out-of-character discussion")
	}
	if strings.TrimSpace(sessionInteraction.ActiveSceneID) != sceneID {
		return storage.SceneRecord{}, storage.SceneInteraction{}, status.Error(codes.FailedPrecondition, "scene is not the active scene")
	}
	sceneRecord, err := a.stores.Scene.GetScene(ctx, campaignID, sceneID)
	if err != nil {
		return storage.SceneRecord{}, storage.SceneInteraction{}, err
	}
	if sceneRecord.SessionID != activeSessionID {
		return storage.SceneRecord{}, storage.SceneInteraction{}, status.Error(codes.FailedPrecondition, "scene is not in the active session")
	}
	sceneInteraction, err := a.stores.SceneInteraction.GetSceneInteraction(ctx, campaignID, sceneID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return storage.SceneRecord{}, storage.SceneInteraction{}, grpcerror.Internal("load scene interaction", err)
		}
		sceneInteraction = storage.SceneInteraction{
			CampaignID:           campaignID,
			SceneID:              sceneID,
			SessionID:            activeSessionID,
			ActingCharacterIDs:   []string{},
			ActingParticipantIDs: []string{},
			Slots:                []storage.ScenePlayerSlot{},
		}
	}
	return sceneRecord, sceneInteraction, nil
}

func (a interactionApplication) requireActiveScenePhase(
	ctx context.Context,
	campaignID string,
	activeSessionID string,
	sceneID string,
	sessionInteraction storage.SessionInteraction,
) (storage.SceneRecord, storage.SceneInteraction, error) {
	sceneRecord, sceneInteraction, err := a.requireActiveSceneForGM(ctx, campaignID, activeSessionID, sceneID, sessionInteraction)
	if err != nil {
		return storage.SceneRecord{}, storage.SceneInteraction{}, err
	}
	if !sceneInteraction.PhaseOpen || strings.TrimSpace(sceneInteraction.PhaseID) == "" {
		return storage.SceneRecord{}, storage.SceneInteraction{}, status.Error(codes.FailedPrecondition, "scene player phase is not open")
	}
	return sceneRecord, sceneInteraction, nil
}

func (a interactionApplication) resolveActingSet(
	ctx context.Context,
	campaignID string,
	sceneRecord storage.SceneRecord,
	requestedCharacterIDs []string,
) ([]ids.CharacterID, []ids.ParticipantID, error) {
	if len(requestedCharacterIDs) == 0 {
		return nil, nil, status.Error(codes.InvalidArgument, "character ids are required")
	}
	sceneCharacters, err := a.stores.SceneCharacter.ListSceneCharacters(ctx, campaignID, sceneRecord.SceneID)
	if err != nil {
		return nil, nil, grpcerror.Internal("list scene characters", err)
	}
	sceneCharacterSet := make(map[string]struct{}, len(sceneCharacters))
	for _, sceneCharacter := range sceneCharacters {
		sceneCharacterSet[sceneCharacter.CharacterID] = struct{}{}
	}

	actingCharacterIDs := make([]ids.CharacterID, 0, len(requestedCharacterIDs))
	actingParticipants := make([]ids.ParticipantID, 0, len(requestedCharacterIDs))
	seenParticipants := make(map[string]struct{})
	for _, rawCharacterID := range requestedCharacterIDs {
		characterID := strings.TrimSpace(rawCharacterID)
		if characterID == "" {
			continue
		}
		if _, ok := sceneCharacterSet[characterID]; !ok {
			return nil, nil, status.Error(codes.FailedPrecondition, "acting character is not in the scene")
		}
		characterRecord, err := a.stores.Character.GetCharacter(ctx, campaignID, characterID)
		if err != nil {
			return nil, nil, err
		}
		ownerParticipantID := strings.TrimSpace(characterRecord.OwnerParticipantID)
		if ownerParticipantID == "" {
			return nil, nil, status.Error(codes.FailedPrecondition, "acting character has no owner participant")
		}
		actingCharacterIDs = append(actingCharacterIDs, ids.CharacterID(characterID))
		if _, ok := seenParticipants[ownerParticipantID]; !ok {
			seenParticipants[ownerParticipantID] = struct{}{}
			actingParticipants = append(actingParticipants, ids.ParticipantID(ownerParticipantID))
		}
	}
	if len(actingCharacterIDs) == 0 || len(actingParticipants) == 0 {
		return nil, nil, status.Error(codes.InvalidArgument, "character ids are required")
	}
	return actingCharacterIDs, actingParticipants, nil
}

func (a interactionApplication) resolveParticipantPostCharacters(
	ctx context.Context,
	campaignID string,
	sceneRecord storage.SceneRecord,
	participantID string,
	requestedCharacterIDs []string,
	actingCharacterIDs []string,
) ([]ids.CharacterID, error) {
	allowed := make(map[string]struct{}, len(actingCharacterIDs))
	for _, characterID := range actingCharacterIDs {
		allowed[strings.TrimSpace(characterID)] = struct{}{}
	}
	sceneCharacters, err := a.stores.SceneCharacter.ListSceneCharacters(ctx, campaignID, sceneRecord.SceneID)
	if err != nil {
		return nil, grpcerror.Internal("list scene characters", err)
	}
	inScene := make(map[string]struct{}, len(sceneCharacters))
	for _, sceneCharacter := range sceneCharacters {
		inScene[sceneCharacter.CharacterID] = struct{}{}
	}

	characterIDs := make([]ids.CharacterID, 0, len(requestedCharacterIDs))
	for _, rawCharacterID := range requestedCharacterIDs {
		characterID := strings.TrimSpace(rawCharacterID)
		if characterID == "" {
			continue
		}
		if _, ok := inScene[characterID]; !ok {
			return nil, status.Error(codes.FailedPrecondition, "character is not in the scene")
		}
		if _, ok := allowed[characterID]; !ok {
			return nil, status.Error(codes.PermissionDenied, "character is not acting in the current scene phase")
		}
		characterRecord, err := a.stores.Character.GetCharacter(ctx, campaignID, characterID)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(characterRecord.OwnerParticipantID) != participantID {
			return nil, status.Error(codes.PermissionDenied, "participant does not own the requested character")
		}
		characterIDs = append(characterIDs, ids.CharacterID(characterID))
	}
	if len(characterIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "character ids are required")
	}
	return characterIDs, nil
}

func (a interactionApplication) resolveRevisionRequests(
	ctx context.Context,
	campaignID string,
	sceneRecord storage.SceneRecord,
	sceneInteraction storage.SceneInteraction,
	requests []*campaignv1.ScenePlayerRevisionRequest,
) ([]scene.PlayerPhaseRevisionRequest, error) {
	if len(requests) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one revision request is required")
	}
	sceneCharacters, err := a.stores.SceneCharacter.ListSceneCharacters(ctx, campaignID, sceneRecord.SceneID)
	if err != nil {
		return nil, grpcerror.Internal("list scene characters", err)
	}
	inScene := make(map[string]struct{}, len(sceneCharacters))
	for _, sceneCharacter := range sceneCharacters {
		inScene[sceneCharacter.CharacterID] = struct{}{}
	}
	actingCharacters := make(map[string]struct{}, len(sceneInteraction.ActingCharacterIDs))
	for _, characterID := range sceneInteraction.ActingCharacterIDs {
		actingCharacters[strings.TrimSpace(characterID)] = struct{}{}
	}
	actingParticipants := make(map[string]struct{}, len(sceneInteraction.ActingParticipantIDs))
	for _, participantID := range sceneInteraction.ActingParticipantIDs {
		actingParticipants[strings.TrimSpace(participantID)] = struct{}{}
	}
	revisions := make([]scene.PlayerPhaseRevisionRequest, 0, len(requests))
	seenParticipants := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		participantID, err := validate.RequiredID(request.GetParticipantId(), "participant id")
		if err != nil {
			return nil, err
		}
		reason := strings.TrimSpace(request.GetReason())
		if reason == "" {
			return nil, status.Error(codes.InvalidArgument, "revision reason is required")
		}
		if _, ok := actingParticipants[participantID]; !ok {
			return nil, status.Error(codes.PermissionDenied, "revision participant is not acting in the current scene phase")
		}
		if _, exists := seenParticipants[participantID]; exists {
			return nil, status.Error(codes.InvalidArgument, "revision participants must be unique")
		}
		seenParticipants[participantID] = struct{}{}
		characterIDs := make([]ids.CharacterID, 0, len(request.GetCharacterIds()))
		for _, rawCharacterID := range request.GetCharacterIds() {
			characterID := strings.TrimSpace(rawCharacterID)
			if characterID == "" {
				continue
			}
			if _, ok := inScene[characterID]; !ok {
				return nil, status.Error(codes.FailedPrecondition, "revision character is not in the scene")
			}
			if _, ok := actingCharacters[characterID]; !ok {
				return nil, status.Error(codes.FailedPrecondition, "revision character is not acting in the current scene phase")
			}
			characterRecord, err := a.stores.Character.GetCharacter(ctx, campaignID, characterID)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(characterRecord.OwnerParticipantID) != participantID {
				return nil, status.Error(codes.PermissionDenied, "revision character does not belong to the targeted participant")
			}
			characterIDs = append(characterIDs, ids.CharacterID(characterID))
		}
		revisions = append(revisions, scene.PlayerPhaseRevisionRequest{
			ParticipantID: ids.ParticipantID(participantID),
			Reason:        reason,
			CharacterIDs:  characterIDs,
		})
	}
	return revisions, nil
}

func (a interactionApplication) executeSessionCommand(ctx context.Context, commandType command.Type, campaignID, sessionID string, payload any, label string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}
	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandType,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents(label+" did not emit an event"),
	)
	return err
}

func (a interactionApplication) executeSceneCommand(ctx context.Context, commandType command.Type, campaignID, sessionID, sceneID string, payload any, label string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}
	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandType,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents(label+" did not emit an event"),
	)
	return err
}

func (a interactionApplication) endScenePhase(ctx context.Context, campaignID, sessionID, sceneID, phaseID, reason string) error {
	payload := scene.PlayerPhaseEndedPayload{
		SceneID: ids.SceneID(sceneID),
		PhaseID: phaseID,
		Reason:  strings.TrimSpace(reason),
	}
	return a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseEnd, campaignID, sessionID, sceneID, payload, "scene.player_phase.end")
}

func (a interactionApplication) endScenePhaseIfOpen(ctx context.Context, campaignID, sceneID, reason string) error {
	sceneRecord, err := a.stores.Scene.GetScene(ctx, campaignID, sceneID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		return err
	}
	sceneInteraction, err := a.stores.SceneInteraction.GetSceneInteraction(ctx, campaignID, sceneID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		return grpcerror.Internal("load scene interaction", err)
	}
	if !sceneInteraction.PhaseOpen || strings.TrimSpace(sceneInteraction.PhaseID) == "" {
		return nil
	}
	return a.endScenePhase(ctx, campaignID, sceneRecord.SessionID, sceneID, sceneInteraction.PhaseID, reason)
}

func (a interactionApplication) yieldScenePhase(ctx context.Context, campaignID, activeSessionID, sceneID, phaseID, participantID string) error {
	payload := scene.PlayerPhaseYieldedPayload{
		SceneID:       ids.SceneID(sceneID),
		PhaseID:       phaseID,
		ParticipantID: ids.ParticipantID(participantID),
	}
	if err := a.executeSceneCommand(ctx, commandTypeScenePlayerPhaseYield, campaignID, activeSessionID, sceneID, payload, "scene.player_phase.yield"); err != nil {
		return err
	}
	return nil
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

func sessionInteractionToProto(interaction storage.SessionInteraction) *campaignv1.OOCState {
	posts := make([]*campaignv1.OOCPost, 0, len(interaction.OOCPosts))
	for _, post := range interaction.OOCPosts {
		posts = append(posts, &campaignv1.OOCPost{
			PostId:        post.PostID,
			ParticipantId: post.ParticipantID,
			Body:          post.Body,
			CreatedAt:     timestamppb.New(post.CreatedAt),
		})
	}
	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].CreatedAt.AsTime().Before(posts[j].CreatedAt.AsTime())
	})
	ready := append([]string(nil), interaction.ReadyToResumeParticipantIDs...)
	sort.Strings(ready)
	return &campaignv1.OOCState{
		Open:                        interaction.OOCPaused,
		Posts:                       posts,
		ReadyToResumeParticipantIds: ready,
	}
}

func aiTurnToProto(turn storage.SessionAITurn) *campaignv1.AITurnState {
	return &campaignv1.AITurnState{
		Status:             aiTurnStatusToProto(turn.Status),
		TurnToken:          turn.TurnToken,
		OwnerParticipantId: turn.OwnerParticipantID,
		SourceEventType:    turn.SourceEventType,
		SourceSceneId:      turn.SourceSceneID,
		SourcePhaseId:      turn.SourcePhaseID,
		LastError:          turn.LastError,
	}
}

func aiTurnStatusToProto(status session.AITurnStatus) campaignv1.AITurnStatus {
	switch status {
	case session.AITurnStatusQueued:
		return campaignv1.AITurnStatus_AI_TURN_STATUS_QUEUED
	case session.AITurnStatusRunning:
		return campaignv1.AITurnStatus_AI_TURN_STATUS_RUNNING
	case session.AITurnStatusFailed:
		return campaignv1.AITurnStatus_AI_TURN_STATUS_FAILED
	default:
		return campaignv1.AITurnStatus_AI_TURN_STATUS_IDLE
	}
}

func sceneInteractionToProto(interaction storage.SceneInteraction) *campaignv1.ScenePlayerPhase {
	if !interaction.PhaseOpen || strings.TrimSpace(interaction.PhaseID) == "" {
		return &campaignv1.ScenePlayerPhase{
			Status:               campaignv1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM,
			ActingCharacterIds:   []string{},
			ActingParticipantIds: []string{},
			Slots:                []*campaignv1.ScenePlayerSlot{},
		}
	}
	slots := make([]*campaignv1.ScenePlayerSlot, 0, len(interaction.Slots))
	for _, slot := range interaction.Slots {
		slots = append(slots, &campaignv1.ScenePlayerSlot{
			ParticipantId:      slot.ParticipantID,
			SummaryText:        slot.SummaryText,
			CharacterIds:       append([]string(nil), slot.CharacterIDs...),
			UpdatedAt:          timestamppb.New(slot.UpdatedAt),
			Yielded:            slot.Yielded,
			ReviewStatus:       scenePlayerSlotReviewStatusToProto(slot.ReviewStatus),
			ReviewReason:       slot.ReviewReason,
			ReviewCharacterIds: append([]string(nil), slot.ReviewCharacterIDs...),
		})
	}
	sort.SliceStable(slots, func(i, j int) bool {
		if slots[i].ParticipantId == slots[j].ParticipantId {
			return slots[i].UpdatedAt.AsTime().Before(slots[j].UpdatedAt.AsTime())
		}
		return slots[i].ParticipantId < slots[j].ParticipantId
	})
	actingCharacters := append([]string(nil), interaction.ActingCharacterIDs...)
	actingParticipants := append([]string(nil), interaction.ActingParticipantIDs...)
	sort.Strings(actingCharacters)
	sort.Strings(actingParticipants)
	return &campaignv1.ScenePlayerPhase{
		PhaseId:              interaction.PhaseID,
		Status:               scenePhaseStatusToProto(interaction.PhaseStatus),
		FrameText:            interaction.FrameText,
		ActingCharacterIds:   actingCharacters,
		ActingParticipantIds: actingParticipants,
		Slots:                slots,
	}
}

func sceneGMOutputToProto(interaction storage.SceneInteraction) *campaignv1.InteractionGMOutput {
	if strings.TrimSpace(interaction.GMOutputText) == "" {
		return nil
	}
	output := &campaignv1.InteractionGMOutput{
		Text:          interaction.GMOutputText,
		ParticipantId: interaction.GMOutputParticipantID,
	}
	if interaction.GMOutputUpdatedAt != nil {
		output.UpdatedAt = timestamppb.New(*interaction.GMOutputUpdatedAt)
	}
	return output
}

func scenePhaseStatusToProto(status scene.PlayerPhaseStatus) campaignv1.ScenePhaseStatus {
	switch status {
	case scene.PlayerPhaseStatusGMReview:
		return campaignv1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW
	case scene.PlayerPhaseStatusPlayers:
		return campaignv1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS
	default:
		return campaignv1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS
	}
}

func scenePlayerSlotReviewStatusToProto(status scene.PlayerPhaseSlotReviewStatus) campaignv1.ScenePlayerSlotReviewStatus {
	switch status {
	case scene.PlayerPhaseSlotReviewStatusUnderReview:
		return campaignv1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW
	case scene.PlayerPhaseSlotReviewStatusAccepted:
		return campaignv1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED
	case scene.PlayerPhaseSlotReviewStatusChangesRequested:
		return campaignv1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED
	default:
		return campaignv1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_OPEN
	}
}
