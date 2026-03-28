package interactiontransport

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	errSessionOOCPaused            = "session is paused for out-of-character discussion"
	errSessionOOCResolutionPending = "session is waiting for gm resolution after out-of-character discussion"
)

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
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load active session"); lookupErr != nil {
		return nil, storage.SessionInteraction{}, lookupErr
	}
	if err != nil {
		return nil, storage.SessionInteraction{}, nil
	}
	interaction, err := a.stores.SessionInteraction.GetSessionInteraction(ctx, campaignID, activeSession.ID)
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load session interaction"); lookupErr != nil {
		return nil, storage.SessionInteraction{}, lookupErr
	}
	if err != nil {
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

func requireSceneWritesUnblocked(sessionInteraction storage.SessionInteraction) error {
	if sessionInteraction.OOCPaused {
		return status.Error(codes.FailedPrecondition, errSessionOOCPaused)
	}
	if sessionInteraction.OOCResolutionPending {
		return status.Error(codes.FailedPrecondition, errSessionOOCResolutionPending)
	}
	return nil
}

func (a interactionApplication) clearOOCResolutionIfPending(ctx context.Context, campaignID, sessionID string, sessionInteraction storage.SessionInteraction, resolution string) error {
	if !sessionInteraction.OOCResolutionPending {
		return nil
	}
	payload := session.OOCResolvedPayload{
		SessionID:  ids.SessionID(sessionID),
		Resolution: strings.TrimSpace(resolution),
	}
	return a.executeSessionCommand(ctx, commandTypeSessionOOCResolve, campaignID, sessionID, payload, "session.ooc.resolve")
}

func (a interactionApplication) buildGMInteractionPayload(
	input *campaignv1.GMInteractionInput,
	sceneID ids.SceneID,
	phaseID string,
	participantID ids.ParticipantID,
) (scene.GMInteractionCommittedPayload, error) {
	if input == nil {
		return scene.GMInteractionCommittedPayload{}, status.Error(codes.InvalidArgument, "interaction is required")
	}
	interactionID, err := a.idGenerator()
	if err != nil {
		return scene.GMInteractionCommittedPayload{}, grpcerror.Internal("generate gm interaction id", err)
	}
	title := strings.TrimSpace(input.GetTitle())
	if title == "" {
		return scene.GMInteractionCommittedPayload{}, status.Error(codes.InvalidArgument, "interaction title is required")
	}
	characterIDs := make([]ids.CharacterID, 0, len(input.GetCharacterIds()))
	for _, rawCharacterID := range input.GetCharacterIds() {
		characterID := strings.TrimSpace(rawCharacterID)
		if characterID == "" {
			continue
		}
		characterIDs = append(characterIDs, ids.CharacterID(characterID))
	}
	beats := make([]scene.GMInteractionBeat, 0, len(input.GetBeats()))
	for _, beat := range input.GetBeats() {
		if beat == nil {
			return scene.GMInteractionCommittedPayload{}, status.Error(codes.InvalidArgument, "interaction beats must be present")
		}
		beatID := strings.TrimSpace(beat.GetBeatId())
		if beatID == "" {
			return scene.GMInteractionCommittedPayload{}, status.Error(codes.InvalidArgument, "interaction beat id is required")
		}
		text := strings.TrimSpace(beat.GetText())
		if text == "" {
			return scene.GMInteractionCommittedPayload{}, status.Error(codes.InvalidArgument, "interaction beat text is required")
		}
		beatType, err := gmInteractionBeatTypeFromProto(beat.GetType())
		if err != nil {
			return scene.GMInteractionCommittedPayload{}, err
		}
		beats = append(beats, scene.GMInteractionBeat{
			BeatID: beatID,
			Type:   beatType,
			Text:   text,
		})
	}
	if len(beats) == 0 {
		return scene.GMInteractionCommittedPayload{}, status.Error(codes.InvalidArgument, "at least one interaction beat is required")
	}

	var illustration *scene.GMInteractionIllustration
	if rawIllustration := input.GetIllustration(); rawIllustration != nil {
		imageURL := strings.TrimSpace(rawIllustration.GetImageUrl())
		alt := strings.TrimSpace(rawIllustration.GetAlt())
		caption := strings.TrimSpace(rawIllustration.GetCaption())
		if imageURL != "" || alt != "" || caption != "" {
			if imageURL == "" {
				return scene.GMInteractionCommittedPayload{}, status.Error(codes.InvalidArgument, "interaction illustration image_url is required")
			}
			if alt == "" {
				return scene.GMInteractionCommittedPayload{}, status.Error(codes.InvalidArgument, "interaction illustration alt is required")
			}
			illustration = &scene.GMInteractionIllustration{
				ImageURL: imageURL,
				Alt:      alt,
				Caption:  caption,
			}
		}
	}

	return scene.GMInteractionCommittedPayload{
		SceneID:       sceneID,
		InteractionID: interactionID,
		PhaseID:       strings.TrimSpace(phaseID),
		ParticipantID: participantID,
		Title:         title,
		CharacterIDs:  characterIDs,
		Illustration:  illustration,
		Beats:         beats,
	}, nil
}

func gmInteractionBeatTypeFromProto(value campaignv1.GMInteractionBeatType) (scene.GMInteractionBeatType, error) {
	switch value {
	case campaignv1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_FICTION:
		return scene.GMInteractionBeatTypeFiction, nil
	case campaignv1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT:
		return scene.GMInteractionBeatTypePrompt, nil
	case campaignv1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_RESOLUTION:
		return scene.GMInteractionBeatTypeResolution, nil
	case campaignv1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_CONSEQUENCE:
		return scene.GMInteractionBeatTypeConsequence, nil
	case campaignv1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_GUIDANCE:
		return scene.GMInteractionBeatTypeGuidance, nil
	default:
		return "", status.Error(codes.InvalidArgument, "interaction beat type is required")
	}
}

func interactionCharacterIDsToStrings(values []ids.CharacterID) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		characterID := strings.TrimSpace(value.String())
		if characterID == "" {
			continue
		}
		result = append(result, characterID)
	}
	return result
}

func (a interactionApplication) loadSceneState(ctx context.Context, campaignID string, sceneRecord storage.SceneRecord) (*campaignv1.InteractionScene, storage.SceneInteraction, error) {
	sceneCharacters, err := a.stores.SceneCharacter.ListSceneCharacters(ctx, campaignID, sceneRecord.SceneID)
	if err != nil {
		return nil, storage.SceneInteraction{}, grpcerror.Internal("list scene characters", err)
	}
	characters := make([]*campaignv1.InteractionCharacter, 0, len(sceneCharacters))
	for _, sceneCharacter := range sceneCharacters {
		characterRecord, err := a.stores.Character.GetCharacter(ctx, campaignID, sceneCharacter.CharacterID)
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load scene character"); lookupErr != nil {
			return nil, storage.SceneInteraction{}, lookupErr
		}
		if err != nil {
			continue
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
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load scene interaction"); lookupErr != nil {
		return nil, storage.SceneInteraction{}, lookupErr
	}
	if err != nil {
		sceneInteraction = storage.SceneInteraction{
			CampaignID:           campaignID,
			SceneID:              sceneRecord.SceneID,
			SessionID:            sceneRecord.SessionID,
			ActingCharacterIDs:   []string{},
			ActingParticipantIDs: []string{},
			Slots:                []storage.ScenePlayerSlot{},
		}
	}
	gmInteractions := []storage.SceneGMInteraction{}
	if a.stores.SceneGMInteraction != nil {
		var err error
		gmInteractions, err = a.stores.SceneGMInteraction.ListSceneGMInteractions(ctx, campaignID, sceneRecord.SceneID)
		if err != nil {
			return nil, storage.SceneInteraction{}, grpcerror.Internal("list scene gm interactions", err)
		}
	}
	if len(gmInteractions) == 0 && a.stores.Event != nil {
		var err error
		gmInteractions, err = a.loadSceneGMInteractionsFromEvents(ctx, campaignID, sceneRecord)
		if err != nil {
			return nil, storage.SceneInteraction{}, err
		}
	}

	var currentInteraction *campaignv1.GMInteraction
	history := make([]*campaignv1.GMInteraction, 0, max(len(gmInteractions)-1, 0))
	for i, interaction := range gmInteractions {
		protoInteraction := sceneGMInteractionToProto(interaction)
		if protoInteraction == nil {
			continue
		}
		if i == 0 {
			currentInteraction = protoInteraction
			continue
		}
		history = append(history, protoInteraction)
	}

	return &campaignv1.InteractionScene{
		SceneId:            sceneRecord.SceneID,
		SessionId:          sceneRecord.SessionID,
		Name:               sceneRecord.Name,
		Description:        sceneRecord.Description,
		Characters:         characters,
		CurrentInteraction: currentInteraction,
		InteractionHistory: history,
	}, sceneInteraction, nil
}

func (a interactionApplication) loadSceneGMInteractionsFromEvents(
	ctx context.Context,
	campaignID string,
	sceneRecord storage.SceneRecord,
) ([]storage.SceneGMInteraction, error) {
	const pageSize = 200

	afterSeq := uint64(0)
	interactions := make([]storage.SceneGMInteraction, 0, 8)
	for {
		events, err := a.stores.Event.ListEventsBySession(ctx, campaignID, sceneRecord.SessionID, afterSeq, pageSize)
		if err != nil {
			return nil, grpcerror.Internal("list scene gm interaction events", err)
		}
		if len(events) == 0 {
			break
		}
		for _, evt := range events {
			afterSeq = evt.Seq
			if evt.Type != scene.EventTypeGMInteractionCommitted || evt.SceneID.String() != sceneRecord.SceneID {
				continue
			}
			var payload scene.GMInteractionCommittedPayload
			if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
				return nil, grpcerror.Internal("decode scene gm interaction event", err)
			}
			interaction := storage.SceneGMInteraction{
				CampaignID:    campaignID,
				SceneID:       sceneRecord.SceneID,
				SessionID:     sceneRecord.SessionID,
				InteractionID: strings.TrimSpace(payload.InteractionID),
				PhaseID:       strings.TrimSpace(payload.PhaseID),
				ParticipantID: strings.TrimSpace(payload.ParticipantID.String()),
				Title:         strings.TrimSpace(payload.Title),
				CharacterIDs:  interactionCharacterIDsToStrings(payload.CharacterIDs),
				Beats:         make([]storage.SceneGMInteractionBeat, 0, len(payload.Beats)),
				CreatedAt:     evt.Timestamp.UTC(),
			}
			if payload.Illustration != nil {
				interaction.Illustration = &storage.SceneGMInteractionIllustration{
					ImageURL: strings.TrimSpace(payload.Illustration.ImageURL),
					Alt:      strings.TrimSpace(payload.Illustration.Alt),
					Caption:  strings.TrimSpace(payload.Illustration.Caption),
				}
			}
			for _, beat := range payload.Beats {
				interaction.Beats = append(interaction.Beats, storage.SceneGMInteractionBeat{
					BeatID: strings.TrimSpace(beat.BeatID),
					Type:   beat.Type,
					Text:   strings.TrimSpace(beat.Text),
				})
			}
			interactions = append(interactions, interaction)
		}
		if len(events) < pageSize {
			break
		}
	}
	sort.SliceStable(interactions, func(i, j int) bool {
		if interactions[i].CreatedAt.Equal(interactions[j].CreatedAt) {
			return interactions[i].InteractionID > interactions[j].InteractionID
		}
		return interactions[i].CreatedAt.After(interactions[j].CreatedAt)
	})
	return interactions, nil
}

func (a interactionApplication) requireActiveSceneForGM(
	ctx context.Context,
	campaignID string,
	activeSessionID string,
	sceneID string,
	sessionInteraction storage.SessionInteraction,
) (storage.SceneRecord, storage.SceneInteraction, error) {
	if sessionInteraction.OOCPaused {
		return storage.SceneRecord{}, storage.SceneInteraction{}, status.Error(codes.FailedPrecondition, errSessionOOCPaused)
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
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load scene interaction"); lookupErr != nil {
		return storage.SceneRecord{}, storage.SceneInteraction{}, lookupErr
	}
	if err != nil {
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
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load scene"); lookupErr != nil {
		return lookupErr
	}
	if err != nil {
		return nil
	}
	sceneInteraction, err := a.stores.SceneInteraction.GetSceneInteraction(ctx, campaignID, sceneID)
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load scene interaction"); lookupErr != nil {
		return lookupErr
	}
	if err != nil {
		return nil
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
