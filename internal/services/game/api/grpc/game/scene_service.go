package game

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SceneService implements the game.v1.SceneService gRPC API.
type SceneService struct {
	campaignv1.UnimplementedSceneServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewSceneService creates a SceneService with default dependencies.
func NewSceneService(stores Stores) *SceneService {
	return &SceneService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// CreateScene creates a new scene within an active session.
func (s *SceneService) CreateScene(ctx context.Context, in *campaignv1.CreateSceneRequest) (*campaignv1.CreateSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create scene request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	sceneID, err := newSceneApplication(s).CreateScene(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.CreateSceneResponse{SceneId: sceneID}, nil
}

// UpdateScene updates scene metadata.
func (s *SceneService) UpdateScene(ctx context.Context, in *campaignv1.UpdateSceneRequest) (*campaignv1.UpdateSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update scene request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).UpdateScene(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.UpdateSceneResponse{}, nil
}

// EndScene ends an active scene.
func (s *SceneService) EndScene(ctx context.Context, in *campaignv1.EndSceneRequest) (*campaignv1.EndSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "end scene request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).EndScene(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.EndSceneResponse{}, nil
}

// AddCharacterToScene adds a character to a scene.
func (s *SceneService) AddCharacterToScene(ctx context.Context, in *campaignv1.AddCharacterToSceneRequest) (*campaignv1.AddCharacterToSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "add character to scene request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).AddCharacterToScene(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.AddCharacterToSceneResponse{}, nil
}

// RemoveCharacterFromScene removes a character from a scene.
func (s *SceneService) RemoveCharacterFromScene(ctx context.Context, in *campaignv1.RemoveCharacterFromSceneRequest) (*campaignv1.RemoveCharacterFromSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "remove character from scene request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).RemoveCharacterFromScene(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.RemoveCharacterFromSceneResponse{}, nil
}

// TransferCharacter transfers a character from one scene to another.
func (s *SceneService) TransferCharacter(ctx context.Context, in *campaignv1.TransferCharacterRequest) (*campaignv1.TransferCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "transfer character request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).TransferCharacter(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.TransferCharacterResponse{}, nil
}

// TransitionScene transitions a scene to a new scene, atomically moving all characters.
func (s *SceneService) TransitionScene(ctx context.Context, in *campaignv1.TransitionSceneRequest) (*campaignv1.TransitionSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "transition scene request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	newSceneID, err := newSceneApplication(s).TransitionScene(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.TransitionSceneResponse{NewSceneId: newSceneID}, nil
}

// OpenSceneGate opens a gate that blocks scene actions until resolved.
func (s *SceneService) OpenSceneGate(ctx context.Context, in *campaignv1.OpenSceneGateRequest) (*campaignv1.OpenSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open scene gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).OpenSceneGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.OpenSceneGateResponse{}, nil
}

// ResolveSceneGate resolves an open scene gate.
func (s *SceneService) ResolveSceneGate(ctx context.Context, in *campaignv1.ResolveSceneGateRequest) (*campaignv1.ResolveSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve scene gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).ResolveSceneGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.ResolveSceneGateResponse{}, nil
}

// AbandonSceneGate abandons an open scene gate.
func (s *SceneService) AbandonSceneGate(ctx context.Context, in *campaignv1.AbandonSceneGateRequest) (*campaignv1.AbandonSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon scene gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).AbandonSceneGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.AbandonSceneGateResponse{}, nil
}

// SetSceneSpotlight sets the scene spotlight.
func (s *SceneService) SetSceneSpotlight(ctx context.Context, in *campaignv1.SetSceneSpotlightRequest) (*campaignv1.SetSceneSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set scene spotlight request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).SetSceneSpotlight(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.SetSceneSpotlightResponse{}, nil
}

// ClearSceneSpotlight clears the scene spotlight.
func (s *SceneService) ClearSceneSpotlight(ctx context.Context, in *campaignv1.ClearSceneSpotlightRequest) (*campaignv1.ClearSceneSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "clear scene spotlight request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).ClearSceneSpotlight(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.ClearSceneSpotlightResponse{}, nil
}

const (
	defaultListScenesPageSize = 20
	maxListScenesPageSize     = 50
)

// GetScene returns a scene by campaign ID and scene ID.
func (s *SceneService) GetScene(ctx context.Context, in *campaignv1.GetSceneRequest) (*campaignv1.GetSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get scene request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return nil, status.Error(codes.InvalidArgument, "scene id is required")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireReadPolicy(ctx, s.stores, campaignRecord); err != nil {
		return nil, err
	}

	rec, err := s.stores.Scene.GetScene(ctx, campaignID, sceneID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	characters, err := s.stores.SceneCharacter.ListSceneCharacters(ctx, campaignID, sceneID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list scene characters: %v", err)
	}

	return &campaignv1.GetSceneResponse{
		Scene: sceneToProto(rec, characters),
	}, nil
}

// ListScenes returns a page of scene records for a session.
func (s *SceneService) ListScenes(ctx context.Context, in *campaignv1.ListScenesRequest) (*campaignv1.ListScenesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list scenes request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireReadPolicy(ctx, s.stores, campaignRecord); err != nil {
		return nil, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListScenesPageSize,
		Max:     maxListScenesPageSize,
	})

	page, err := s.stores.Scene.ListScenes(ctx, campaignID, sessionID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list scenes: %v", err)
	}

	response := &campaignv1.ListScenesResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Scenes) == 0 {
		return response, nil
	}

	response.Scenes = make([]*campaignv1.Scene, 0, len(page.Scenes))
	for _, rec := range page.Scenes {
		// For list, we don't fetch characters per scene to avoid N+1 queries.
		response.Scenes = append(response.Scenes, sceneToProto(rec, nil))
	}

	return response, nil
}

// sceneToProto converts a scene record and optional character list to proto.
func sceneToProto(rec storage.SceneRecord, characters []storage.SceneCharacterRecord) *campaignv1.Scene {
	pb := &campaignv1.Scene{
		SceneId:     rec.SceneID,
		SessionId:   rec.SessionID,
		Name:        rec.Name,
		Description: rec.Description,
		Active:      rec.Active,
		CreatedAt:   timestamppb.New(rec.CreatedAt),
		UpdatedAt:   timestamppb.New(rec.UpdatedAt),
	}
	if rec.EndedAt != nil {
		pb.EndedAt = timestamppb.New(*rec.EndedAt)
	}
	if len(characters) > 0 {
		pb.CharacterIds = make([]string, 0, len(characters))
		for _, c := range characters {
			pb.CharacterIds = append(pb.CharacterIds, c.CharacterID)
		}
	}
	return pb
}
