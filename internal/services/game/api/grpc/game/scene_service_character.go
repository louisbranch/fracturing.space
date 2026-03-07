package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
