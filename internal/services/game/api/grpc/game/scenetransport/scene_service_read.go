package scenetransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Scenes use a slightly larger default than handler.PageSmall because scene lists
	// are typically displayed in a timeline view where more items improve UX.
	defaultListScenesPageSize = 20
	maxListScenesPageSize     = handler.PageMedium
)

type sceneReadDependencies struct {
	auth   authz.PolicyDeps
	stores sceneReadStores
}

type sceneReadStores struct {
	Campaign       storage.CampaignStore
	Scene          storage.SceneStore
	SceneCharacter storage.SceneCharacterStore
}

func newSceneReadDependencies(deps Deps) sceneReadDependencies {
	return sceneReadDependencies{
		auth: deps.Auth,
		stores: sceneReadStores{
			Campaign:       deps.Campaign,
			Scene:          deps.Scene,
			SceneCharacter: deps.SceneCharacter,
		},
	}
}

// GetScene returns a scene by campaign ID and scene ID.
func (s *Service) GetScene(ctx context.Context, in *campaignv1.GetSceneRequest) (*campaignv1.GetSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}

	campaignRecord, err := s.reads.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := authz.RequireReadPolicy(ctx, s.reads.auth, campaignRecord); err != nil {
		return nil, err
	}

	rec, err := s.reads.stores.Scene.GetScene(ctx, campaignID, sceneID)
	if err != nil {
		return nil, err
	}

	characters, err := s.reads.stores.SceneCharacter.ListSceneCharacters(ctx, campaignID, sceneID)
	if err != nil {
		return nil, grpcerror.Internal("list scene characters", err)
	}

	return &campaignv1.GetSceneResponse{
		Scene: SceneToProto(rec, characters),
	}, nil
}

// ListScenes returns a page of scene records for a session.
func (s *Service) ListScenes(ctx context.Context, in *campaignv1.ListScenesRequest) (*campaignv1.ListScenesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list scenes request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}

	campaignRecord, err := s.reads.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := authz.RequireReadPolicy(ctx, s.reads.auth, campaignRecord); err != nil {
		return nil, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListScenesPageSize,
		Max:     maxListScenesPageSize,
	})

	page, err := s.reads.stores.Scene.ListScenes(ctx, campaignID, sessionID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, grpcerror.Internal("list scenes", err)
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
		response.Scenes = append(response.Scenes, SceneToProto(rec, nil))
	}

	return response, nil
}

// SceneToProto converts a scene record and optional character list to proto.
func SceneToProto(rec storage.SceneRecord, characters []storage.SceneCharacterRecord) *campaignv1.Scene {
	pb := &campaignv1.Scene{
		SceneId:     rec.SceneID,
		SessionId:   rec.SessionID,
		Name:        rec.Name,
		Description: rec.Description,
		Open:        rec.Open,
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
