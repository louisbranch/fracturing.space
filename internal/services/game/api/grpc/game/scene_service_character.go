package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AddCharacterToScene adds a character to a scene.
func (s *SceneService) AddCharacterToScene(ctx context.Context, in *campaignv1.AddCharacterToSceneRequest) (*campaignv1.AddCharacterToSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "add character to scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).AddCharacterToScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.AddCharacterToSceneResponse{}, nil
}

// RemoveCharacterFromScene removes a character from a scene.
func (s *SceneService) RemoveCharacterFromScene(ctx context.Context, in *campaignv1.RemoveCharacterFromSceneRequest) (*campaignv1.RemoveCharacterFromSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "remove character from scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).RemoveCharacterFromScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.RemoveCharacterFromSceneResponse{}, nil
}

// TransferCharacter transfers a character from one scene to another.
func (s *SceneService) TransferCharacter(ctx context.Context, in *campaignv1.TransferCharacterRequest) (*campaignv1.TransferCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "transfer character request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).TransferCharacter(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.TransferCharacterResponse{}, nil
}
