package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

// sessionWorkflowService owns session-centric gameplay workflow orchestration.
type sessionWorkflowService struct {
	service *DaggerheartService
}

func newSessionWorkflowService(service *DaggerheartService) sessionWorkflowService {
	return sessionWorkflowService{service: service}
}

func (w sessionWorkflowService) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	return w.service.runSessionActionRoll(ctx, in)
}

func (w sessionWorkflowService) SessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	return w.service.runSessionDamageRoll(ctx, in)
}

func (w sessionWorkflowService) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	return w.service.runSessionAttackFlow(ctx, in)
}

func (w sessionWorkflowService) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	return w.service.runSessionReactionFlow(ctx, in)
}

func (w sessionWorkflowService) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	return w.service.runSessionAdversaryAttackRoll(ctx, in)
}

func (w sessionWorkflowService) SessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	return w.service.runSessionAdversaryActionCheck(ctx, in)
}

func (w sessionWorkflowService) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	return w.service.runSessionAdversaryAttackFlow(ctx, in)
}

func (w sessionWorkflowService) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	return w.service.runSessionGroupActionFlow(ctx, in)
}

func (w sessionWorkflowService) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	return w.service.runSessionTagTeamFlow(ctx, in)
}

func (s *DaggerheartService) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	return newSessionWorkflowService(s).SessionActionRoll(ctx, in)
}

func (s *DaggerheartService) SessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	return newSessionWorkflowService(s).SessionDamageRoll(ctx, in)
}

func (s *DaggerheartService) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	return newSessionWorkflowService(s).SessionAttackFlow(ctx, in)
}

func (s *DaggerheartService) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	return newSessionWorkflowService(s).SessionReactionFlow(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	return newSessionWorkflowService(s).SessionAdversaryAttackRoll(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	return newSessionWorkflowService(s).SessionAdversaryActionCheck(ctx, in)
}

func (s *DaggerheartService) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	return newSessionWorkflowService(s).SessionAdversaryAttackFlow(ctx, in)
}

func (s *DaggerheartService) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	return newSessionWorkflowService(s).SessionGroupActionFlow(ctx, in)
}

func (s *DaggerheartService) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	return newSessionWorkflowService(s).SessionTagTeamFlow(ctx, in)
}
