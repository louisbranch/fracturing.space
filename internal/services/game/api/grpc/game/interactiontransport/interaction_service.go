package interactiontransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InteractionService implements the game.v1.InteractionService gRPC API.
type InteractionService struct {
	campaignv1.UnimplementedInteractionServiceServer
	app interactionApplication
}

// NewInteractionService creates an InteractionService with default dependencies.
func NewInteractionService(deps Deps) *InteractionService {
	return &InteractionService{
		app: newInteractionApplicationWithDependencies(deps, id.NewID),
	}
}

func (s *InteractionService) GetInteractionState(ctx context.Context, in *campaignv1.GetInteractionStateRequest) (*campaignv1.GetInteractionStateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "interaction state request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.GetInteractionState(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return &campaignv1.GetInteractionStateResponse{State: state}, nil
}

func (s *InteractionService) ActivateScene(ctx context.Context, in *campaignv1.ActivateSceneRequest) (*campaignv1.ActivateSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "activate scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.ActivateScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.ActivateSceneResponse{State: state}, nil
}

func (s *InteractionService) OpenScenePlayerPhase(ctx context.Context, in *campaignv1.OpenScenePlayerPhaseRequest) (*campaignv1.OpenScenePlayerPhaseResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open scene player phase request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.OpenScenePlayerPhase(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.OpenScenePlayerPhaseResponse{State: state}, nil
}

func (s *InteractionService) SubmitScenePlayerAction(ctx context.Context, in *campaignv1.SubmitScenePlayerActionRequest) (*campaignv1.SubmitScenePlayerActionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "submit scene player action request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.SubmitScenePlayerAction(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.SubmitScenePlayerActionResponse{State: state}, nil
}

func (s *InteractionService) YieldScenePlayerPhase(ctx context.Context, in *campaignv1.YieldScenePlayerPhaseRequest) (*campaignv1.YieldScenePlayerPhaseResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "yield scene player phase request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.YieldScenePlayerPhase(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.YieldScenePlayerPhaseResponse{State: state}, nil
}

func (s *InteractionService) WithdrawScenePlayerYield(ctx context.Context, in *campaignv1.WithdrawScenePlayerYieldRequest) (*campaignv1.WithdrawScenePlayerYieldResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "withdraw scene player yield request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.WithdrawScenePlayerYield(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.WithdrawScenePlayerYieldResponse{State: state}, nil
}

func (s *InteractionService) InterruptScenePlayerPhase(ctx context.Context, in *campaignv1.InterruptScenePlayerPhaseRequest) (*campaignv1.InterruptScenePlayerPhaseResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "interrupt scene player phase request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.InterruptScenePlayerPhase(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.InterruptScenePlayerPhaseResponse{State: state}, nil
}

func (s *InteractionService) RecordSceneGMInteraction(ctx context.Context, in *campaignv1.RecordSceneGMInteractionRequest) (*campaignv1.RecordSceneGMInteractionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "record scene gm interaction request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.RecordSceneGMInteraction(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.RecordSceneGMInteractionResponse{State: state}, nil
}

func (s *InteractionService) ResolveScenePlayerReview(ctx context.Context, in *campaignv1.ResolveScenePlayerReviewRequest) (*campaignv1.ResolveScenePlayerReviewResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve scene player review request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.ResolveScenePlayerReview(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.ResolveScenePlayerReviewResponse{State: state}, nil
}

func (s *InteractionService) OpenSessionOOC(ctx context.Context, in *campaignv1.OpenSessionOOCRequest) (*campaignv1.OpenSessionOOCResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open session ooc request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.OpenSessionOOC(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.OpenSessionOOCResponse{State: state}, nil
}

func (s *InteractionService) PostSessionOOC(ctx context.Context, in *campaignv1.PostSessionOOCRequest) (*campaignv1.PostSessionOOCResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "post session ooc request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.PostSessionOOC(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.PostSessionOOCResponse{State: state}, nil
}

func (s *InteractionService) MarkOOCReadyToResume(ctx context.Context, in *campaignv1.MarkOOCReadyToResumeRequest) (*campaignv1.MarkOOCReadyToResumeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "mark ooc ready request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.MarkOOCReadyToResume(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.MarkOOCReadyToResumeResponse{State: state}, nil
}

func (s *InteractionService) ClearOOCReadyToResume(ctx context.Context, in *campaignv1.ClearOOCReadyToResumeRequest) (*campaignv1.ClearOOCReadyToResumeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "clear ooc ready request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.ClearOOCReadyToResume(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.ClearOOCReadyToResumeResponse{State: state}, nil
}

func (s *InteractionService) ResolveSessionOOC(ctx context.Context, in *campaignv1.ResolveSessionOOCRequest) (*campaignv1.ResolveSessionOOCResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve session ooc request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.ResolveSessionOOC(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.ResolveSessionOOCResponse{State: state}, nil
}

func (s *InteractionService) SetSessionGMAuthority(ctx context.Context, in *campaignv1.SetSessionGMAuthorityRequest) (*campaignv1.SetSessionGMAuthorityResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set session gm authority request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.SetSessionGMAuthority(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.SetSessionGMAuthorityResponse{State: state}, nil
}

func (s *InteractionService) SetSessionCharacterController(ctx context.Context, in *campaignv1.SetSessionCharacterControllerRequest) (*campaignv1.SetSessionCharacterControllerResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set session character controller request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.SetSessionCharacterController(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.SetSessionCharacterControllerResponse{State: state}, nil
}

func (s *InteractionService) RetryAIGMTurn(ctx context.Context, in *campaignv1.RetryAIGMTurnRequest) (*campaignv1.RetryAIGMTurnResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "retry ai gm turn request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.RetryAIGMTurn(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.RetryAIGMTurnResponse{State: state}, nil
}
