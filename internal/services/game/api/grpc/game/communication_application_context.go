package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"

	"context"
	"sort"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/sessiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	communicationStreamSystemLabel  = "System"
	communicationStreamTableLabel   = "Table"
	communicationStreamControlLabel = "Control"
)

func (a communicationApplication) GetCommunicationContext(ctx context.Context, campaignID string) (*campaignv1.CommunicationContext, error) {
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := authz.RequireReadPolicy(ctx, a.auth, campaignRecord); err != nil {
		return nil, err
	}

	actor, _, err := authz.ResolvePolicyActor(ctx, a.stores.Participant, campaignID)
	if err != nil {
		return nil, err
	}

	activeSessionState, err := a.sessions.GetActiveSessionContext(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	activeSession := activeSessionState.session
	ownedCharacters, err := a.listOwnedCharacters(ctx, campaignID, actor.ID)
	if err != nil {
		return nil, err
	}
	streams, err := a.buildStreams(ctx, campaignID, actor, activeSession, ownedCharacters)
	if err != nil {
		return nil, err
	}
	personas := buildCommunicationPersonas(actor, ownedCharacters)

	contextState := &campaignv1.CommunicationContext{
		CampaignId:       campaignRecord.ID,
		CampaignName:     campaignRecord.Name,
		Locale:           campaigntransport.LocaleStringToProto(campaignRecord.Locale),
		GmMode:           campaigntransport.GMModeToProto(campaignRecord.GmMode),
		AiAgentId:        campaignRecord.AIAgentID,
		Participant:      communicationParticipantToProto(actor),
		Streams:          streams,
		Personas:         personas,
		DefaultStreamId:  defaultCommunicationStreamID(campaignID),
		DefaultPersonaId: participantPersonaID(actor.ID),
	}
	if activeSession == nil {
		return contextState, nil
	}

	contextState.ActiveSession = &campaignv1.CommunicationSession{
		SessionId: activeSession.ID,
		Name:      activeSession.Name,
	}
	if activeSessionState.gate != nil {
		protoGate, err := sessiontransport.GateToProto(*activeSessionState.gate)
		if err != nil {
			return nil, grpcerror.Internal("map active session gate", err)
		}
		contextState.ActiveSessionGate = protoGate
	}
	if activeSessionState.spotlight != nil {
		contextState.ActiveSessionSpotlight = sessiontransport.SpotlightToProto(*activeSessionState.spotlight)
	}
	return contextState, nil
}

func (a communicationApplication) listOwnedCharacters(ctx context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	if a.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}

	owned, err := a.stores.Character.ListCharactersByOwnerParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list owned characters: %v", err)
	}

	sort.SliceStable(owned, func(i, j int) bool {
		if owned[i].Name == owned[j].Name {
			return owned[i].ID < owned[j].ID
		}
		return owned[i].Name < owned[j].Name
	})
	return owned, nil
}

func (a communicationApplication) buildStreams(
	ctx context.Context,
	campaignID string,
	actor storage.ParticipantRecord,
	activeSession *storage.SessionRecord,
	ownedCharacters []storage.CharacterRecord,
) ([]*campaignv1.CommunicationStream, error) {
	streams := []*campaignv1.CommunicationStream{
		{
			StreamId:   communicationSystemStreamID(campaignID),
			Kind:       campaignv1.CommunicationStreamKind_COMMUNICATION_STREAM_KIND_SYSTEM,
			Scope:      communicationBaseScope(activeSession),
			CampaignId: campaignID,
			Label:      communicationStreamSystemLabel,
		},
		{
			StreamId:   defaultCommunicationStreamID(campaignID),
			Kind:       campaignv1.CommunicationStreamKind_COMMUNICATION_STREAM_KIND_TABLE,
			Scope:      communicationBaseScope(activeSession),
			CampaignId: campaignID,
			Label:      communicationStreamTableLabel,
		},
		{
			StreamId:   communicationControlStreamID(campaignID),
			Kind:       campaignv1.CommunicationStreamKind_COMMUNICATION_STREAM_KIND_CONTROL,
			Scope:      communicationBaseScope(activeSession),
			CampaignId: campaignID,
			Label:      communicationStreamControlLabel,
		},
	}
	if activeSession == nil {
		return streams, nil
	}
	for _, stream := range streams {
		stream.SessionId = activeSession.ID
	}

	sceneStreams, err := a.buildSceneStreams(ctx, campaignID, actor, activeSession.ID, ownedCharacters)
	if err != nil {
		return nil, err
	}
	return append(streams, sceneStreams...), nil
}

func (a communicationApplication) buildSceneStreams(
	ctx context.Context,
	campaignID string,
	actor storage.ParticipantRecord,
	sessionID string,
	ownedCharacters []storage.CharacterRecord,
) ([]*campaignv1.CommunicationStream, error) {
	if a.stores.Scene == nil {
		return nil, status.Error(codes.Internal, "scene store is not configured")
	}

	activeScenes, err := a.listVisibleActiveScenes(ctx, campaignID, actor, sessionID, ownedCharacters)
	if err != nil {
		return nil, err
	}

	streams := make([]*campaignv1.CommunicationStream, 0, len(activeScenes))
	for _, sceneRecord := range activeScenes {
		label := strings.TrimSpace(sceneRecord.Name)
		if label == "" {
			label = sceneRecord.SceneID
		}
		streams = append(streams, &campaignv1.CommunicationStream{
			StreamId:   communicationSceneCharacterStreamID(sceneRecord.SceneID),
			Kind:       campaignv1.CommunicationStreamKind_COMMUNICATION_STREAM_KIND_CHARACTER,
			Scope:      campaignv1.CommunicationStreamScope_COMMUNICATION_STREAM_SCOPE_SCENE,
			CampaignId: campaignID,
			SessionId:  sceneRecord.SessionID,
			SceneId:    sceneRecord.SceneID,
			Label:      label,
		})
	}
	return streams, nil
}

func (a communicationApplication) listVisibleActiveScenes(
	ctx context.Context,
	campaignID string,
	actor storage.ParticipantRecord,
	sessionID string,
	ownedCharacters []storage.CharacterRecord,
) ([]storage.SceneRecord, error) {
	if actor.Role == participant.RoleGM {
		activeScenes, err := a.stores.Scene.ListActiveScenes(ctx, campaignID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list active scenes: %v", err)
		}
		filtered := make([]storage.SceneRecord, 0, len(activeScenes))
		for _, sceneRecord := range activeScenes {
			if sceneRecord.SessionID == sessionID {
				filtered = append(filtered, sceneRecord)
			}
		}
		sortSceneRecords(filtered)
		return filtered, nil
	}

	characterIDs := make([]string, 0, len(ownedCharacters))
	for _, characterRecord := range ownedCharacters {
		characterIDs = append(characterIDs, characterRecord.ID)
	}

	activeScenes, err := a.stores.Scene.ListVisibleActiveScenesForCharacters(ctx, campaignID, sessionID, characterIDs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list visible active scenes: %v", err)
	}
	sortSceneRecords(activeScenes)
	return activeScenes, nil
}

func sortSceneRecords(records []storage.SceneRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Name == records[j].Name {
			return records[i].SceneID < records[j].SceneID
		}
		return records[i].Name < records[j].Name
	})
}

func communicationParticipantToProto(record storage.ParticipantRecord) *campaignv1.CommunicationParticipant {
	return &campaignv1.CommunicationParticipant{
		ParticipantId: record.ID,
		Name:          record.Name,
		Role:          participanttransport.RoleToProto(record.Role),
	}
}

func buildCommunicationPersonas(actor storage.ParticipantRecord, ownedCharacters []storage.CharacterRecord) []*campaignv1.CommunicationPersona {
	personas := make([]*campaignv1.CommunicationPersona, 0, len(ownedCharacters)+1)
	personas = append(personas, &campaignv1.CommunicationPersona{
		PersonaId:     participantPersonaID(actor.ID),
		Kind:          campaignv1.CommunicationPersonaKind_COMMUNICATION_PERSONA_KIND_PARTICIPANT,
		ParticipantId: actor.ID,
		DisplayName:   actor.Name,
	})
	for _, characterRecord := range ownedCharacters {
		personas = append(personas, &campaignv1.CommunicationPersona{
			PersonaId:     characterPersonaID(characterRecord.ID),
			Kind:          campaignv1.CommunicationPersonaKind_COMMUNICATION_PERSONA_KIND_CHARACTER,
			ParticipantId: actor.ID,
			CharacterId:   characterRecord.ID,
			DisplayName:   characterRecord.Name,
		})
	}
	return personas
}

func communicationBaseScope(activeSession *storage.SessionRecord) campaignv1.CommunicationStreamScope {
	if activeSession == nil {
		return campaignv1.CommunicationStreamScope_COMMUNICATION_STREAM_SCOPE_CAMPAIGN
	}
	return campaignv1.CommunicationStreamScope_COMMUNICATION_STREAM_SCOPE_SESSION
}

func communicationSystemStreamID(campaignID string) string {
	return "campaign:" + campaignID + ":system"
}

func defaultCommunicationStreamID(campaignID string) string {
	return "campaign:" + campaignID + ":table"
}

func communicationControlStreamID(campaignID string) string {
	return "campaign:" + campaignID + ":control"
}

func communicationSceneCharacterStreamID(sceneID string) string {
	return "scene:" + sceneID + ":character"
}

func participantPersonaID(participantID string) string {
	return "participant:" + participantID
}

func characterPersonaID(characterID string) string {
	return "character:" + characterID
}
