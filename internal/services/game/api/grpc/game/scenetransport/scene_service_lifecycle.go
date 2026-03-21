package scenetransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateScene creates a new scene within an active session.
func (s *Service) CreateScene(ctx context.Context, in *campaignv1.CreateSceneRequest) (*campaignv1.CreateSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	app := newSceneApplication(s)
	sceneID, err := app.CreateScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	var interactionState *campaignv1.InteractionState
	if shouldActivateSceneFromCreate(in) {
		interactionState, err = app.activateScene(ctx, campaignID, sceneID)
	} else {
		interactionState, err = app.interactionState(ctx, campaignID)
	}
	if err != nil {
		return nil, err
	}

	return &campaignv1.CreateSceneResponse{SceneId: sceneID, InteractionState: interactionState}, nil
}

// UpdateScene updates scene metadata.
func (s *Service) UpdateScene(ctx context.Context, in *campaignv1.UpdateSceneRequest) (*campaignv1.UpdateSceneResponse, error) {
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
func (s *Service) EndScene(ctx context.Context, in *campaignv1.EndSceneRequest) (*campaignv1.EndSceneResponse, error) {
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
func (s *Service) TransitionScene(ctx context.Context, in *campaignv1.TransitionSceneRequest) (*campaignv1.TransitionSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "transition scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	app := newSceneApplication(s)
	newSceneID, err := app.TransitionScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	var interactionState *campaignv1.InteractionState
	if shouldActivateSceneFromTransition(in) {
		interactionState, err = app.activateScene(ctx, campaignID, newSceneID)
	} else {
		interactionState, err = app.interactionState(ctx, campaignID)
	}
	if err != nil {
		return nil, err
	}

	return &campaignv1.TransitionSceneResponse{NewSceneId: newSceneID, InteractionState: interactionState}, nil
}
