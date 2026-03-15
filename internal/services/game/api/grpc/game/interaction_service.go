package game

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
func NewInteractionService(stores Stores) *InteractionService {
	return &InteractionService{
		app: newInteractionApplicationWithDependencies(stores, id.NewID),
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

func (s *InteractionService) SetActiveScene(ctx context.Context, in *campaignv1.SetActiveSceneRequest) (*campaignv1.SetActiveSceneResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set active scene request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.SetActiveScene(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.SetActiveSceneResponse{State: state}, nil
}

func (s *InteractionService) StartScenePlayerPhase(ctx context.Context, in *campaignv1.StartScenePlayerPhaseRequest) (*campaignv1.StartScenePlayerPhaseResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start scene player phase request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.StartScenePlayerPhase(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.StartScenePlayerPhaseResponse{State: state}, nil
}

func (s *InteractionService) SubmitScenePlayerPost(ctx context.Context, in *campaignv1.SubmitScenePlayerPostRequest) (*campaignv1.SubmitScenePlayerPostResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "submit scene player post request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.SubmitScenePlayerPost(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.SubmitScenePlayerPostResponse{State: state}, nil
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

func (s *InteractionService) UnyieldScenePlayerPhase(ctx context.Context, in *campaignv1.UnyieldScenePlayerPhaseRequest) (*campaignv1.UnyieldScenePlayerPhaseResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "unyield scene player phase request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.UnyieldScenePlayerPhase(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.UnyieldScenePlayerPhaseResponse{State: state}, nil
}

func (s *InteractionService) EndScenePlayerPhase(ctx context.Context, in *campaignv1.EndScenePlayerPhaseRequest) (*campaignv1.EndScenePlayerPhaseResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "end scene player phase request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.EndScenePlayerPhase(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.EndScenePlayerPhaseResponse{State: state}, nil
}

func (s *InteractionService) CommitSceneGMOutput(ctx context.Context, in *campaignv1.CommitSceneGMOutputRequest) (*campaignv1.CommitSceneGMOutputResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "commit scene gm output request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.CommitSceneGMOutput(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.CommitSceneGMOutputResponse{State: state}, nil
}

func (s *InteractionService) AcceptScenePlayerPhase(ctx context.Context, in *campaignv1.AcceptScenePlayerPhaseRequest) (*campaignv1.AcceptScenePlayerPhaseResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "accept scene player phase request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.AcceptScenePlayerPhase(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.AcceptScenePlayerPhaseResponse{State: state}, nil
}

func (s *InteractionService) RequestScenePlayerRevisions(ctx context.Context, in *campaignv1.RequestScenePlayerRevisionsRequest) (*campaignv1.RequestScenePlayerRevisionsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request scene player revisions request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.RequestScenePlayerRevisions(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.RequestScenePlayerRevisionsResponse{State: state}, nil
}

func (s *InteractionService) PauseSessionForOOC(ctx context.Context, in *campaignv1.PauseSessionForOOCRequest) (*campaignv1.PauseSessionForOOCResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "pause session for ooc request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.PauseSessionForOOC(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.PauseSessionForOOCResponse{State: state}, nil
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

func (s *InteractionService) ResumeFromOOC(ctx context.Context, in *campaignv1.ResumeFromOOCRequest) (*campaignv1.ResumeFromOOCResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resume from ooc request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	state, err := s.app.ResumeFromOOC(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.ResumeFromOOCResponse{State: state}, nil
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
