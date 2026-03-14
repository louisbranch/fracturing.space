package game

import (
	"context"
	"errors"
	"sort"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
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

// CommunicationService exposes game-owned communication context to transport
// layers so chat/web do not infer gameplay routing rules on their own.
type CommunicationService struct {
	campaignv1.UnimplementedCommunicationServiceServer
	stores      Stores
	idGenerator func() (string, error)
}

// NewCommunicationService creates a CommunicationService with projection-backed
// read dependencies.
func NewCommunicationService(stores Stores) *CommunicationService {
	return &CommunicationService{
		stores:      stores,
		idGenerator: id.NewID,
	}
}

// GetCommunicationContext returns caller-specific communication metadata for a campaign.
func (s *CommunicationService) GetCommunicationContext(ctx context.Context, in *campaignv1.GetCommunicationContextRequest) (*campaignv1.GetCommunicationContextResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "communication context request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicy(ctx, s.stores, campaignRecord); err != nil {
		return nil, err
	}

	actor, _, err := resolvePolicyActor(ctx, s.stores.Participant, campaignID)
	if err != nil {
		return nil, err
	}

	activeSession, err := s.loadActiveSession(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	ownedCharacters, err := s.listOwnedCharacters(ctx, campaignID, actor.ID)
	if err != nil {
		return nil, err
	}
	streams, err := s.buildStreams(ctx, campaignID, actor, activeSession, ownedCharacters)
	if err != nil {
		return nil, err
	}
	personas := buildCommunicationPersonas(actor, ownedCharacters)

	context := &campaignv1.CommunicationContext{
		CampaignId:       campaignRecord.ID,
		CampaignName:     campaignRecord.Name,
		Locale:           campaignRecord.Locale,
		GmMode:           gmModeToProto(campaignRecord.GmMode),
		AiAgentId:        campaignRecord.AIAgentID,
		Participant:      communicationParticipantToProto(actor),
		Streams:          streams,
		Personas:         personas,
		DefaultStreamId:  defaultCommunicationStreamID(campaignID),
		DefaultPersonaId: participantPersonaID(actor.ID),
	}
	if activeSession != nil {
		context.ActiveSession = &campaignv1.CommunicationSession{
			SessionId: activeSession.ID,
			Name:      activeSession.Name,
		}
		activeGate, err := s.loadActiveSessionGate(ctx, campaignID, activeSession.ID)
		if err != nil {
			return nil, err
		}
		context.ActiveSessionGate = activeGate

		activeSpotlight, err := s.loadActiveSessionSpotlight(ctx, campaignID, activeSession.ID)
		if err != nil {
			return nil, err
		}
		context.ActiveSessionSpotlight = activeSpotlight
	}

	return &campaignv1.GetCommunicationContextResponse{Context: context}, nil
}

func (s *CommunicationService) loadActiveSession(ctx context.Context, campaignID string) (*storage.SessionRecord, error) {
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	activeSession, err := s.stores.Session.GetActiveSession(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, nil
		}
		return nil, status.Errorf(codes.Internal, "get active session: %v", err)
	}
	return &activeSession, nil
}

func (s *CommunicationService) loadActiveSessionGate(ctx context.Context, campaignID, sessionID string) (*campaignv1.SessionGate, error) {
	if s.stores.SessionGate == nil {
		return nil, nil
	}

	activeGate, err := s.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, nil
		}
		return nil, status.Errorf(codes.Internal, "get open session gate: %v", err)
	}
	protoGate, err := sessionGateToProto(activeGate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "map session gate: %v", err)
	}
	return protoGate, nil
}

func (s *CommunicationService) loadActiveSessionSpotlight(ctx context.Context, campaignID, sessionID string) (*campaignv1.SessionSpotlight, error) {
	if s.stores.SessionSpotlight == nil {
		return nil, nil
	}

	spotlight, err := s.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, nil
		}
		return nil, status.Errorf(codes.Internal, "get session spotlight: %v", err)
	}
	return sessionSpotlightToProto(spotlight), nil
}

func (s *CommunicationService) listOwnedCharacters(ctx context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}

	const pageSize = 200
	pageToken := ""
	owned := make([]storage.CharacterRecord, 0)
	for {
		page, err := s.stores.Character.ListCharacters(ctx, campaignID, pageSize, pageToken)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list characters: %v", err)
		}
		for _, characterRecord := range page.Characters {
			if strings.TrimSpace(characterRecord.OwnerParticipantID) == participantID {
				owned = append(owned, characterRecord)
			}
		}
		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" {
			break
		}
		pageToken = nextPageToken
	}

	sort.SliceStable(owned, func(i, j int) bool {
		if owned[i].Name == owned[j].Name {
			return owned[i].ID < owned[j].ID
		}
		return owned[i].Name < owned[j].Name
	})
	return owned, nil
}

func (s *CommunicationService) buildStreams(
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

	sceneStreams, err := s.buildSceneStreams(ctx, campaignID, actor, activeSession.ID, ownedCharacters)
	if err != nil {
		return nil, err
	}
	streams = append(streams, sceneStreams...)
	return streams, nil
}

func (s *CommunicationService) buildSceneStreams(
	ctx context.Context,
	campaignID string,
	actor storage.ParticipantRecord,
	sessionID string,
	ownedCharacters []storage.CharacterRecord,
) ([]*campaignv1.CommunicationStream, error) {
	if s.stores.Scene == nil {
		return nil, status.Error(codes.Internal, "scene store is not configured")
	}
	if s.stores.SceneCharacter == nil {
		return nil, status.Error(codes.Internal, "scene character store is not configured")
	}

	activeScenes, err := s.stores.Scene.ListActiveScenes(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list active scenes: %v", err)
	}
	sort.SliceStable(activeScenes, func(i, j int) bool {
		if activeScenes[i].Name == activeScenes[j].Name {
			return activeScenes[i].SceneID < activeScenes[j].SceneID
		}
		return activeScenes[i].Name < activeScenes[j].Name
	})

	ownedCharacterIDs := make(map[string]struct{}, len(ownedCharacters))
	for _, characterRecord := range ownedCharacters {
		ownedCharacterIDs[characterRecord.ID] = struct{}{}
	}

	streams := make([]*campaignv1.CommunicationStream, 0, len(activeScenes))
	for _, sceneRecord := range activeScenes {
		if sceneRecord.SessionID != sessionID {
			continue
		}
		if actor.Role != participant.RoleGM {
			sceneCharacters, err := s.stores.SceneCharacter.ListSceneCharacters(ctx, campaignID, sceneRecord.SceneID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "list scene characters: %v", err)
			}
			visible := false
			for _, sceneCharacter := range sceneCharacters {
				if _, ok := ownedCharacterIDs[sceneCharacter.CharacterID]; ok {
					visible = true
					break
				}
			}
			if !visible {
				continue
			}
		}
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

func communicationParticipantToProto(record storage.ParticipantRecord) *campaignv1.CommunicationParticipant {
	return &campaignv1.CommunicationParticipant{
		ParticipantId: record.ID,
		Name:          record.Name,
		Role:          participantRoleToProto(record.Role),
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
