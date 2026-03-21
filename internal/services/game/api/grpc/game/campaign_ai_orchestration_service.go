package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/interactiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CampaignAIOrchestrationService implements internal automation lifecycle
// writes for AI-owned GM turns.
type CampaignAIOrchestrationService struct {
	campaignv1.UnimplementedCampaignAIOrchestrationServiceServer
	app interactiontransport.AIOrchestrationApplication
}

// NewCampaignAIOrchestrationService creates the internal AI GM turn
// orchestration service from explicit capability-owned dependencies.
func NewCampaignAIOrchestrationService(deps CampaignAIOrchestrationDeps) *CampaignAIOrchestrationService {
	return newCampaignAIOrchestrationServiceWithDependencies(deps, id.NewID)
}

func newCampaignAIOrchestrationServiceWithDependencies(
	deps CampaignAIOrchestrationDeps,
	idGenerator func() (string, error),
) *CampaignAIOrchestrationService {
	return &CampaignAIOrchestrationService{
		app: newCampaignAIOrchestrationApplicationWithDependencies(deps, idGenerator),
	}
}

func (s *CampaignAIOrchestrationService) QueueAIGMTurn(ctx context.Context, in *campaignv1.QueueAIGMTurnRequest) (*campaignv1.QueueAIGMTurnResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "queue ai gm turn request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	turn, err := s.app.QueueAIGMTurn(ctx, campaignID, sessionID, in.GetSourceEventType(), in.GetSourceSceneId(), in.GetSourcePhaseId())
	if err != nil {
		return nil, err
	}
	return &campaignv1.QueueAIGMTurnResponse{AiTurn: turn}, nil
}

func (s *CampaignAIOrchestrationService) StartAIGMTurn(ctx context.Context, in *campaignv1.StartAIGMTurnRequest) (*campaignv1.StartAIGMTurnResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start ai gm turn request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	turnToken, err := validate.RequiredID(in.GetTurnToken(), "turn token")
	if err != nil {
		return nil, err
	}
	turn, err := s.app.StartAIGMTurn(ctx, campaignID, sessionID, turnToken)
	if err != nil {
		return nil, err
	}
	return &campaignv1.StartAIGMTurnResponse{AiTurn: turn}, nil
}

func (s *CampaignAIOrchestrationService) FailAIGMTurn(ctx context.Context, in *campaignv1.FailAIGMTurnRequest) (*campaignv1.FailAIGMTurnResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "fail ai gm turn request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	turnToken, err := validate.RequiredID(in.GetTurnToken(), "turn token")
	if err != nil {
		return nil, err
	}
	turn, err := s.app.FailAIGMTurn(ctx, campaignID, sessionID, turnToken, in.GetLastError())
	if err != nil {
		return nil, err
	}
	return &campaignv1.FailAIGMTurnResponse{AiTurn: turn}, nil
}

func (s *CampaignAIOrchestrationService) CompleteAIGMTurn(ctx context.Context, in *campaignv1.CompleteAIGMTurnRequest) (*campaignv1.CompleteAIGMTurnResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "complete ai gm turn request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	turnToken, err := validate.RequiredID(in.GetTurnToken(), "turn token")
	if err != nil {
		return nil, err
	}
	turn, err := s.app.CompleteAIGMTurn(ctx, campaignID, sessionID, turnToken)
	if err != nil {
		return nil, err
	}
	return &campaignv1.CompleteAIGMTurnResponse{AiTurn: turn}, nil
}
