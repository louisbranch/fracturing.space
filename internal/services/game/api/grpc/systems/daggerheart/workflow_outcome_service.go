package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

// outcomeWorkflowService owns outcome-application orchestration workflows.
type outcomeWorkflowService struct {
	service *DaggerheartService
}

func newOutcomeWorkflowService(service *DaggerheartService) outcomeWorkflowService {
	return outcomeWorkflowService{service: service}
}

func (w outcomeWorkflowService) ApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	return w.service.runApplyRollOutcome(ctx, in)
}

func (w outcomeWorkflowService) ApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	return w.service.runApplyAttackOutcome(ctx, in)
}

func (w outcomeWorkflowService) ApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	return w.service.runApplyAdversaryAttackOutcome(ctx, in)
}

func (w outcomeWorkflowService) ApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	return w.service.runApplyReactionOutcome(ctx, in)
}

func (s *DaggerheartService) ApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	return newOutcomeWorkflowService(s).ApplyRollOutcome(ctx, in)
}

func (s *DaggerheartService) ApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	return newOutcomeWorkflowService(s).ApplyAttackOutcome(ctx, in)
}

func (s *DaggerheartService) ApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	return newOutcomeWorkflowService(s).ApplyAdversaryAttackOutcome(ctx, in)
}

func (s *DaggerheartService) ApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	return newOutcomeWorkflowService(s).ApplyReactionOutcome(ctx, in)
}
