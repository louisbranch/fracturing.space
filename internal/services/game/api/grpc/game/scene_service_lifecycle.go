package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateScene creates a new scene within an active session.
func (s *SceneService) CreateScene(ctx context.Context, in *campaignv1.CreateSceneRequest) (*campaignv1.CreateSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	sceneID, err := newSceneApplication(s).CreateScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.CreateSceneResponse{SceneId: sceneID}, nil
}

// UpdateScene updates scene metadata.
func (s *SceneService) UpdateScene(ctx context.Context, in *campaignv1.UpdateSceneRequest) (*campaignv1.UpdateSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).UpdateScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.UpdateSceneResponse{}, nil
}

// EndScene ends an active scene.
func (s *SceneService) EndScene(ctx context.Context, in *campaignv1.EndSceneRequest) (*campaignv1.EndSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "end scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).EndScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.EndSceneResponse{}, nil
}

// TransitionScene transitions a scene to a new scene, atomically moving all characters.
func (s *SceneService) TransitionScene(ctx context.Context, in *campaignv1.TransitionSceneRequest) (*campaignv1.TransitionSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "transition scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	newSceneID, err := newSceneApplication(s).TransitionScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.TransitionSceneResponse{NewSceneId: newSceneID}, nil
}
