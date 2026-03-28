package interactiontransport

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Deps groups the dependencies for the interaction transport surface.
type Deps struct {
	Auth               authz.PolicyDeps
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	Character          storage.CharacterStore
	Event              storage.EventHistoryStore
	Session            storage.SessionStore
	SessionInteraction storage.SessionInteractionStore
	Scene              storage.SceneStore
	SceneCharacter     storage.SceneCharacterStore
	SceneInteraction   storage.SceneInteractionStore
	SceneGMInteraction storage.SceneGMInteractionStore
	Write              domainwrite.WritePath
	Applier            projection.Applier
}

// interactionApplication coordinates the scene-phase interaction service over
// projection-backed state plus explicit domain writes.
type interactionApplication struct {
	auth        authz.PolicyDeps
	stores      interactionApplicationStores
	write       domainwrite.WritePath
	applier     projection.Applier
	idGenerator func() (string, error)
}

type interactionApplicationStores struct {
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	Character          storage.CharacterStore
	Event              storage.EventHistoryStore
	Session            storage.SessionStore
	SessionInteraction storage.SessionInteractionStore
	Scene              storage.SceneStore
	SceneCharacter     storage.SceneCharacterStore
	SceneInteraction   storage.SceneInteractionStore
	SceneGMInteraction storage.SceneGMInteractionStore
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
			Event:              deps.Event,
			Session:            deps.Session,
			SessionInteraction: deps.SessionInteraction,
			Scene:              deps.Scene,
			SceneCharacter:     deps.SceneCharacter,
			SceneInteraction:   deps.SceneInteraction,
			SceneGMInteraction: deps.SceneGMInteraction,
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
		state.Control = deriveInteractionControlState(actor, storage.SessionInteraction{}, storage.SceneInteraction{})
		return state, nil
	}

	state.ActiveSession = &campaignv1.InteractionSession{
		SessionId: activeSession.ID,
		Name:      activeSession.Name,
	}
	state.Ooc = sessionInteractionToProto(sessionInteraction)
	state.CharacterControllers = sessionCharacterControllersToProto(sessionInteraction)
	state.GmAuthorityParticipantId = sessionInteraction.GMAuthorityParticipantID
	state.AiTurn = aiTurnToProto(sessionInteraction.AITurn)

	if strings.TrimSpace(sessionInteraction.ActiveSceneID) == "" {
		state.Control = deriveInteractionControlState(actor, sessionInteraction, storage.SceneInteraction{})
		return state, nil
	}

	sceneRecord, err := a.stores.Scene.GetScene(ctx, campaignID, sessionInteraction.ActiveSceneID)
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load active scene"); lookupErr != nil {
		return nil, lookupErr
	}
	if err != nil {
		return state, nil
	}
	activeScene, sceneInteraction, err := a.loadSceneState(ctx, campaignID, sceneRecord, sessionInteraction)
	if err != nil {
		return nil, err
	}
	state.ActiveScene = activeScene
	state.PlayerPhase = sceneInteractionToProto(sceneInteraction)
	state.Control = deriveInteractionControlState(actor, sessionInteraction, sceneInteraction)
	return state, nil
}

func (a interactionApplication) ActivateScene(ctx context.Context, campaignID string, in *campaignv1.ActivateSceneRequest) (*campaignv1.InteractionState, error) {
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
	activeSession, currentInteraction, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := requireSceneWritesUnblocked(currentInteraction); err != nil {
		return nil, err
	}
	if err := requireAuthoritativeGMActor(actor, currentInteraction); err != nil {
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

	payload := session.SceneActivatedPayload{
		SessionID:     ids.SessionID(activeSession.ID),
		ActiveSceneID: ids.SceneID(sceneID),
	}
	if err := a.executeSessionCommand(ctx, commandTypeSessionSceneActivate, campaignID, activeSession.ID, payload, "session.scene.activate"); err != nil {
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

func (a interactionApplication) SetSessionCharacterController(ctx context.Context, campaignID string, in *campaignv1.SetSessionCharacterControllerRequest) (*campaignv1.InteractionState, error) {
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
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
	activeSession, _, err := a.requireActiveSessionInteraction(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if _, err := a.stores.Character.GetCharacter(ctx, campaignID, characterID); err != nil {
		return nil, err
	}
	if _, err := a.stores.Participant.GetParticipant(ctx, campaignID, participantID); err != nil {
		return nil, err
	}
	payload := session.CharacterControllerSetPayload{
		SessionID:     ids.SessionID(activeSession.ID),
		CharacterID:   ids.CharacterID(characterID),
		ParticipantID: ids.ParticipantID(participantID),
	}
	if err := a.executeSessionCommand(
		ctx,
		commandTypeSessionCharacterControllerSet,
		campaignID,
		activeSession.ID,
		payload,
		"session.character_controller.set",
	); err != nil {
		return nil, err
	}
	return a.GetInteractionState(ctx, campaignID)
}
